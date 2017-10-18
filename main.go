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
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
	"gopkg.in/reportportal/commons-go.v1/commons"
	"gopkg.in/reportportal/commons-go.v1/conf"
	"gopkg.in/reportportal/commons-go.v1/server"
	"github.com/pkg/errors"
)

var log = logrus.New()

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.Formatter = &logrus.TextFormatter{}

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.Out = os.Stdout
}

func main() {

	defCfg := conf.EmptyConfig()
	defCfg.Consul.Address = "registry:8500"
	defCfg.Consul.Tags = []string{"urlprefix-/analyzer opts strip=/analyzer"}
	cfg := struct {
		*conf.RpConfig
		ESHosts []string `env:"ES_HOSTS" envDefault:"http://elasticsearch:9200"`
	}{
		RpConfig: defCfg,
	}

	err := conf.LoadConfig(&cfg)
	if nil != err {
		log.Fatalf("Cannot load configuration")
	}

	cfg.AppName = "analyzer"
	info := commons.GetBuildInfo()
	info.Name = "Analysis Service"

	srv := server.New(cfg.RpConfig, info)

	c := NewClient(cfg.ESHosts)

	srv.WithRouter(func(router *chi.Mux) {
		router.Use(middleware.Logger)
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

	srv.StartServer()
}

type requestHandler func([]Launch) (interface{}, error)

func handleRequest(w http.ResponseWriter, rq *http.Request, handler requestHandler) error {
	var launches []Launch
	err := server.ReadJSON(*rq, &launches)
	if err != nil {
		return server.ToStatusError(http.StatusBadRequest, errors.WithStack(err))
	}

	rs, err := handler(launches)
	if err != nil {
		return server.ToStatusError(http.StatusInternalServerError, errors.WithStack(err))
	} else {
		server.WriteJSON(http.StatusOK, rs, w)
	}

	return nil
}
