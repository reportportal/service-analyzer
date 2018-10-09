/*
Copyright 2017 EPAM Systems


This file is part of EPAM Report Portal.
https://github.com/reportportal/service-analyzer

Report Portal is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Report Portal is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Report Portal.  If not, see <http://www.gnu.org/licenses/>.
*/
package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
	"gopkg.in/reportportal/commons-go.v1/commons"
	"gopkg.in/reportportal/commons-go.v1/conf"
	"gopkg.in/reportportal/commons-go.v1/server"
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

//SearchConfig specified details of queries to elastic search
type SearchConfig struct {
	BoostLaunch    float64 `env:"ES_BOOST_LAUNCH" envDefault:"2.0"`
	BoostUniqueID  float64 `env:"ES_BOOST_UNIQUE_ID" envDefault:"2.0"`
	BoostAA        float64 `env:"ES_BOOST_AA" envDefault:"2.0"`
	MinDocFreq     float64 `env:"ES_MIN_DOC_FREQ" envDefault:"7"`
	MinTermFreq    float64 `env:"ES_MIN_TERM_FREQ" envDefault:"1"`
	MinShouldMatch string  `env:"ES_MIN_SHOULD_MATCH" envDefault:"80%"`
}

func main() {

	defCfg := conf.EmptyConfig()
	defCfg.Consul.Address = "http://localhost:8500"
	defCfg.Consul.Tags = []string{
		"urlprefix-/analyzer opts strip=/analyzer",
		"traefik.frontend.rule=PathPrefixStrip:/analyzer",
		"analyzer=ML",
		"analyzer_index=true",
		"analyzer_priority=10",
	}
	cfg := struct {
		*conf.RpConfig
		*SearchConfig
		ESHosts  []string `env:"ES_HOSTS" envDefault:"http://dev.epm-rpp.projects.epam.com:9200"`
		LogLevel string   `env:"LOGGING_LEVEL" envDefault:"DEBUG"`
	}{
		RpConfig:     defCfg,
		SearchConfig: &SearchConfig{},
	}

	err := conf.LoadConfig(&cfg)
	if nil != err {
		log.Fatalf("Cannot load configuration")
	}
	//defCfg.Consul.Address = "http://localhost:8500"

	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if nil != err {
		log.Warnf("Unknown logging level %s", cfg.LogLevel)
		logLevel = logrus.DebugLevel
	}
	log.SetLevel(logLevel)

	cfg.AppName = "analyzer"
	info := commons.GetBuildInfo()
	info.Name = "Analysis Service"

	srv := server.New(cfg.RpConfig, info)

	c := NewClient(cfg.ESHosts, cfg.SearchConfig)

	srv.AddHealthCheckFunc(func() error {
		if !c.Healthy() {
			return errors.New("ES Cluster is down")
		}
		return nil
	})

	srv.AddHandler(http.MethodPost, "/_index", func(w http.ResponseWriter, rq *http.Request) error {
		return handleRequest(w, rq,
			func(launches []Launch) (interface{}, error) {
				return c.IndexLogs(launches)
			})
	})
	srv.AddHandler(http.MethodPost, "/_analyze", func(w http.ResponseWriter, rq *http.Request) error {
		return handleRequest(w, rq,
			func(launches []Launch) (interface{}, error) {
				return c.AnalyzeLogs(launches)
			})
	})

	srv.AddHandler(http.MethodDelete, "/_index/{index_id}", deleteIndexHandler(c))
	srv.AddHandler(http.MethodPut, "/_index/delete", cleanIndexHandler(c))

	srv.StartServer()
}

func deleteIndexHandler(c ESClient) func(w http.ResponseWriter, rq *http.Request) error {
	return func(w http.ResponseWriter, rq *http.Request) error {
		if id := chi.URLParam(rq, "index_id"); "" != id {
			_, err := c.DeleteIndex(id)
			return err
		}
		return server.ToStatusError(http.StatusBadRequest, errors.New("Index ID is incorrect"))
	}
}

func cleanIndexHandler(c ESClient) func(w http.ResponseWriter, rq *http.Request) error {
	return func(w http.ResponseWriter, rq *http.Request) error {
		var ci CleanIndex
		err := server.ReadJSON(rq, &ci)
		if nil != err {
			return server.ToStatusError(http.StatusBadRequest, errors.Wrap(err, "Cannot read request body"))
		}
		err = server.Validate(ci)
		if nil != err {
			return server.ToStatusError(http.StatusBadRequest, errors.WithStack(err))
		}

		rs, err := c.DeleteLogs(&ci)
		if nil != err {
			return server.ToStatusError(http.StatusBadRequest, errors.WithStack(err))
		}
		return server.WriteJSON(http.StatusOK, rs, w)
	}
}

type requestHandler func([]Launch) (interface{}, error)

func handleRequest(w http.ResponseWriter, rq *http.Request, handler requestHandler) error {
	var launches []Launch
	err := server.ReadJSON(rq, &launches)
	if err != nil {
		return server.ToStatusError(http.StatusBadRequest, errors.WithStack(err))
	}

	for i, l := range launches {
		if valErr := server.Validate(l); nil != valErr {
			return server.ToStatusError(http.StatusBadRequest, errors.Wrapf(valErr, "Validation failed on Launch[%d]", i))
		}
	}

	rs, err := handler(launches)
	if err != nil {
		return server.ToStatusError(http.StatusInternalServerError, errors.WithStack(err))
	}
	return server.WriteJSON(http.StatusOK, rs, w)
}
