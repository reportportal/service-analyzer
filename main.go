package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/reportportal/commons-go/commons"
	"github.com/reportportal/commons-go/conf"
	"github.com/reportportal/commons-go/server"
	"goji.io"
	"goji.io/pat"
)

func main() {
	defaults := map[string]interface{}{
		"AppName":             "analyzer",
		"Registry":            nil,
		"Server.Port":         9000,
		"Elasticsearch.Hosts": "http://localhost:9200",
	}

	rpConf := conf.LoadConfig("", defaults)

	info := commons.GetBuildInfo()
	info.Name = "Service Analyzer"

	srv := server.New(rpConf, info)

	c := NewClient(rpConf.Get("Elasticsearch.Hosts").(string))

	srv.AddRoute(func(router *goji.Mux) {
		router.Use(func(next http.Handler) http.Handler {
			return handlers.LoggingHandler(os.Stdout, next)
		})

		router.HandleFunc(pat.Get("/"), func(w http.ResponseWriter, rq *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "It Works!")
		})

		router.HandleFunc(pat.Delete("/:project"), func(w http.ResponseWriter, rq *http.Request) {
			project := pat.Param(rq, "project")
			_, err := c.DeleteIndex(project)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Unable to delete index '%s'\n", project)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "Index '%s' successfully deleted\n", project)
			}
		})

		router.HandleFunc(pat.Post("/:project/_index"), func(w http.ResponseWriter, rq *http.Request) {
			project := pat.Param(rq, "project")

			c.RecreateIndex(project, false)

			handleLauchRequest(w, rq,
				func(launch *Launch) (interface{}, error) {
					return c.IndexLogs(project, launch)
				})
		})

		router.HandleFunc(pat.Post("/:project/_analyze"), func(w http.ResponseWriter, rq *http.Request) {
			project := pat.Param(rq, "project")

			handleLauchRequest(w, rq,
				func(launch *Launch) (interface{}, error) {
					return c.AnalyzeLogs(project, launch)
				})
		})
	})

	srv.StartServer()
}

type launchHandler func(*Launch) (interface{}, error)

func handleLauchRequest(w http.ResponseWriter, rq *http.Request, handler launchHandler) {
	launch, err := readRequestBody(rq)
	if err != nil {
		commons.WriteJSON(http.StatusBadRequest, err, w)
	} else {
		rs, err := handler(launch)
		if err != nil {
			commons.WriteJSON(http.StatusInternalServerError, err, w)
		} else {
			commons.WriteJSON(http.StatusOK, rs, w)
		}
	}
}

func readRequestBody(rq *http.Request) (*Launch, error) {
	defer rq.Body.Close()

	rqBody, err := ioutil.ReadAll(rq.Body)
	if err != nil {
		return nil, err
	}

	launch := &Launch{}
	err = json.Unmarshal(rqBody, launch)
	if err != nil {
		return nil, err
	}

	return launch, err
}
