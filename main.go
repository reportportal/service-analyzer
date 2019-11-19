/*
* Copyright 2019 EPAM Systems
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */
package main

import (
	"context"
	"github.com/caarlos0/env"
	"github.com/pkg/errors"
	"github.com/reportportal/commons-go/commons"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/x-cray/logrus-prefixed-formatter"
	"go.uber.org/fx"
	"os"
	"time"
)

var log = logrus.New()

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.Formatter = &prefixed.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		ForceFormatting: true,
	}

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.Out = os.Stdout
}

type (
	//AppConfig is the application configuration
	AppConfig struct {
		*SearchConfig
		//ESHosts  []string `env:"ES_HOSTS" envDefault:"http://localhost:9200"`
		ESHosts  []string `env:"ES_HOSTS" envDefault:"http://elasticsearch:9200"`
		LogLevel string   `env:"LOGGING_LEVEL" envDefault:"DEBUG"`
		//AmqpURL  string   `env:"AMQP_URL" envDefault:"amqp://rabbitmq:rabbitmq@localhost:5672/"`
		AmqpURL           string `env:"AMQP_URL" envDefault:"amqp://rabbitmq:rabbitmq@rabbitmq:5672"`
		AmqpExchangeName  string `env:"AMQP_EXCHANGE_NAME" envDefault:"analyzer"`
		AnalyzerPriority  int    `env:"ANALYZER_PRIORITY" envDefault:"1"`
		AnalyzerIndex     bool   `env:"ANALYZER_INDEX" envDefault:"true"`
		AnalyzerLogSearch bool   `env:"ANALYZER_LOG_SEARCH" envDefault:"true"`
	}

	//SearchConfig specified details of queries to elastic search
	SearchConfig struct {
		BoostLaunch              float64 `env:"ES_BOOST_LAUNCH" envDefault:"2.0"`
		BoostUniqueID            float64 `env:"ES_BOOST_UNIQUE_ID" envDefault:"2.0"`
		BoostAA                  float64 `env:"ES_BOOST_AA" envDefault:"2.0"`
		MinDocFreq               float64 `env:"ES_MIN_DOC_FREQ" envDefault:"7"`
		MinTermFreq              float64 `env:"ES_MIN_TERM_FREQ" envDefault:"1"`
		MinShouldMatch           string  `env:"ES_MIN_SHOULD_MATCH" envDefault:"80%"`
		SearchLogsMinShouldMatch string  `env:"ES_LOGS_MIN_SHOULD_MATCH" envDefault:"98%"`
	}
)

func main() {
	app := fx.New(
		fx.Logger(log),

		// Provide all the constructors we need, which teaches Fx how we'd like to
		// construct the *log.Logger, http.Handler, and *http.ServeMux types.
		// Remember that constructors are called lazily, so this block doesn't do
		// much on its own.
		fx.Provide(
			newConfig,
			newESClient,
			NewAmqpClient,
			NewRequestHandler,

			newAmqpConnection,
		),
		// Since constructors are called lazily, we need some invocations to
		// kick-start our application. In this case, we'll use Register. Since it
		// depends on an http.Handler and *http.ServeMux, calling it requires Fx
		// to build those types using the constructors above. Since we call
		// NewMux, we also register Lifecycle hooks to start and stop an HTTP
		// server.
		fx.Invoke(initLogger, initAmqp),
	)

	app.Run()
	if nil != app.Err() {
		log.Errorf("Terminated with error: %v", app.Err())
	}
	log.Error(app.Err())
}

func initLogger(cfg *AppConfig) {
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if nil != err {
		log.Warnf("Unknown logging level %s", cfg.LogLevel)
		logLevel = logrus.DebugLevel
	}
	log.SetLevel(logLevel)
}

func newConfig() (*AppConfig, error) {
	cfg := &AppConfig{
		SearchConfig: &SearchConfig{},
	}

	return cfg, env.Parse(cfg)
}

func newAmqpConnection(lc fx.Lifecycle, cfg *AppConfig) (*amqp.Connection, error) {
	connection, err := amqp.DialConfig(cfg.AmqpURL, amqp.Config{
		Vhost:     "analyzer",
		Heartbeat: 10 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Warn("Closing AMQP connection")
			return connection.Close()
		},
	})
	log.Info("Connection to AMQP server has been established")
	return connection, err

}
func newESClient(cfg *AppConfig) ESClient {
	return NewClient(cfg.ESHosts, cfg.SearchConfig)
}

func bindQueue(ch *amqp.Channel, name string, exchangeName string) error {
	q, err := ch.QueueDeclare(
		name,  // name
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return errors.Wrapf(err, "Failed to declare a queue: %s", q.Name)
	}
	log.Infof("Queue '%s' has been declared", q.Name)

	err = ch.QueueBind(
		q.Name,       // queue name
		name,         // routing key
		exchangeName, // exchange
		false,
		nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to bind a queue: %s", q.Name)
	}

	return nil
}

func initAmqp(lc fx.Lifecycle, client *AmqpClient, h *RequestHandler, cfg *AppConfig) error {

	var indexQueue = "index"
	var analyzeQueue = "analyze"
	var deleteQueue = "delete"
	var clearQueue = "clean"
	var searchQueue = "search"

	var queues = [5]string{indexQueue, analyzeQueue, deleteQueue, clearQueue, searchQueue}

	err := client.DoOnChannel(func(ch *amqp.Channel) error {
		log.Infof("ExchangeName: %s", cfg.AmqpExchangeName)

		err := ch.ExchangeDeclare(
			cfg.AmqpExchangeName, // name
			amqp.ExchangeDirect,  // kind
			false,                // durable
			true,                 // delete when unused
			false,                // internal
			false,                // noWait
			amqp.Table(map[string]interface{}{
				"analyzer":            cfg.AmqpExchangeName,
				"analyzer_index":      cfg.AnalyzerIndex,
				"analyzer_priority":   cfg.AnalyzerPriority,
				"analyzer_log_search": cfg.AnalyzerLogSearch,
				"version":             commons.GetBuildInfo().Version,
			}), // arguments
		)

		if err != nil {
			return errors.Wrap(err, "Failed to declare a exchange")
		}
		log.Infof("Exchange '%s' has been declared", cfg.AmqpExchangeName)

		for _, queue := range queues {
			err := bindQueue(ch, queue, cfg.AmqpExchangeName)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to init AMQP objects: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			cancel()
			return nil
		},
	})

	go func() {
		if err := client.Receive(ctx, analyzeQueue, false, true, false, false,
			func(d amqp.Delivery) error {
				return client.DoOnChannel(func(channel *amqp.Channel) error {
					return handleAmqpRequest(channel, d, h.AnalyzeLogs)
				})
			}); err != nil {
			log.Error(err)
		}
	}()

	go func() {
		if err := client.Receive(ctx, indexQueue, false, true, false, false,
			func(d amqp.Delivery) error {
				return client.DoOnChannel(func(channel *amqp.Channel) error {
					return handleAmqpRequest(channel, d, h.IndexLaunches)
				})
			}); err != nil {
			log.Error(err)
		}
	}()

	go func() {
		if err := client.Receive(ctx, deleteQueue, false, true, false, false,
			func(d amqp.Delivery) error {
				return client.DoOnChannel(func(channel *amqp.Channel) error {
					return handleDeleteRequest(d, h)
				})
			}); err != nil {
			log.Error(err)
		}
	}()

	go func() {
		if err := client.Receive(ctx, clearQueue, false, true, false, false,
			func(d amqp.Delivery) error {
				return client.DoOnChannel(func(channel *amqp.Channel) error {
					return handleCleanRequest(d, h)
				})
			}); err != nil {
			log.Error(err)
		}
	}()

	go func() {
		if err := client.Receive(ctx, searchQueue, false, true, false, false,
			func(d amqp.Delivery) error {
				return client.DoOnChannel(func(channel *amqp.Channel) error {
					return handleSearchRequest(channel, d, h.SearchLogs)
				})
			}); err != nil {
			log.Error(err)
		}
	}()

	return nil
}
