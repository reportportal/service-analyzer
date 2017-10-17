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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/gorilla/handlers"
	"gopkg.in/reportportal/commons-go.v1/commons"
	"gopkg.in/reportportal/commons-go.v1/conf"
	"gopkg.in/reportportal/commons-go.v1/server"
	"github.com/go-chi/chi/middleware"
	"github.com/reportportal/commons-go/commons"
	"github.com/reportportal/commons-go/conf"
	"github.com/reportportal/commons-go/server"
	"github.com/sirupsen/logrus"
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

		router.MethodFunc(http.MethodPost, "/_index", func(w http.ResponseWriter, rq *http.Request) {
			handleRequest(w, rq,
				func(launches []Launch) (interface{}, error) {
					return c.IndexLogs(launches)
				})
		})

		router.MethodFunc(http.MethodPost, "/_analyze", func(w http.ResponseWriter, rq *http.Request) {
			handleRequest(w, rq,
				func(launches []Launch) (interface{}, error) {
					return c.AnalyzeLogs(launches)
				})
		})
	})

	srv.StartServer()
}

type requestHandler func([]Launch) (interface{}, error)

func handleRequest(w http.ResponseWriter, rq *http.Request, handler requestHandler) {
	launches, err := readRequestBody(rq)
	if err != nil {
		server.WriteJSON(http.StatusBadRequest, err, w)
	} else {
		rs, err := handler(launches)
		if err != nil {
			server.WriteJSON(http.StatusInternalServerError, err, w)
		} else {
			server.WriteJSON(http.StatusOK, rs, w)
		}
	}
}

func readRequestBody(rq *http.Request) ([]Launch, error) {
	defer rq.Body.Close()

	rqBody, err := ioutil.ReadAll(rq.Body)
	if err != nil {
		return nil, err
	}

	launches := []Launch{}
	err = json.Unmarshal(rqBody, &launches)
	if err != nil {
		return nil, err
	}

	return launches, err
}
