package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// ESClient interface
type ESClient interface {
	IndexExists(name string) (bool, error)
	CreateIndex(name string) (bool, error)
	DeleteIndex(name string) (bool, error)
	RecreateIndex(name string) (bool, error)
	IndexLogs(name string, launch *Launch) (*ESResponse, error)
	ListIndices() (*[]Index, error)
}

// ESErrorCause struct
type ESErrorCause struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// ESError struct
type ESError struct {
	RootCause []ESErrorCause `json:"root_cause"`
	Type      string         `json:"type"`
	Reason    string         `json:"reason"`
}

// ESResponse struct
type ESResponse struct {
	Acknowledged bool    `json:"acknowledged"`
	Error        ESError `json:"error"`
	Status       int     `json:"status"`
}

// Log struct
type Log struct {
	LogID    string `json:"logId"`
	LogLevel int    `json:"logLevel"`
	Message  string `json:"message"`
}

// TestItem struct
type TestItem struct {
	TestItemID string `json:"testItemId"`
	IssueType  string `json:"issueType"`
	Logs       []Log  `json:"logs"`
}

// Launch struct
type Launch struct {
	LaunchID   string     `json:"launchId"`
	LaunchName string     `json:"launchName"`
	TestItems  []TestItem `json:"testItems"`
}

// Index struct
type Index struct {
	Health       string `json:"health"`
	Status       string `json:"status"`
	Index        string `json:"index"`
	UUID         string `json:"uuid"`
	Pri          string `json:"pri"`
	Rep          string `json:"rep"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

func main() {
	defaults := map[string]interface{}{
		"AppName":     "analyzer",
		"Registry":    nil,
		"Server.Port": 9000,
	}

	rpConf := conf.LoadConfig("", defaults)

	info := commons.GetBuildInfo()
	info.Name = "Service Analyzer"

	srv := server.New(rpConf, info)

	c := &client{"http://localhost:9200/", []Index{}}

	srv.AddRoute(func(router *goji.Mux) {
		router.Use(func(next http.Handler) http.Handler {
			return handlers.LoggingHandler(os.Stdout, next)
		})

		router.HandleFunc(pat.Get("/"), func(w http.ResponseWriter, rq *http.Request) {
			indicies, err := c.ListIndices()
			if err != nil {
				commons.WriteJSON(http.StatusInternalServerError, "Unable to list indices", w)
			} else {
				commons.WriteJSON(http.StatusOK, indicies, w)
			}
		})

		router.HandleFunc(pat.Post("/index/:project"), func(w http.ResponseWriter, rq *http.Request) {
			project := pat.Param(rq, "project")

			c.RecreateIndex(project, false)

			defer rq.Body.Close()

			rqBody, err := ioutil.ReadAll(rq.Body)
			if err != nil {
				commons.WriteJSON(http.StatusBadRequest, err, w)
			} else {
				launch := &Launch{}
				err = json.Unmarshal(rqBody, launch)
				if err != nil {
					commons.WriteJSON(http.StatusBadRequest, err, w)
				} else {
					esRs, err := c.IndexLogs(project, launch)
					if err != nil {
						commons.WriteJSON(http.StatusInternalServerError, err, w)
					} else {
						commons.WriteJSON(http.StatusOK, esRs, w)
					}

				}
			}
		})
	})

	srv.StartServer()
}

type client struct {
	url      string
	indicies []Index
}

func (rs *ESResponse) String() string {
	s, err := json.Marshal(rs)
	if err != nil {
		s = []byte{}
	}
	return fmt.Sprintf("%v", string(s))
}

func (c *client) ListIndices() (*[]Index, error) {
	httpClient := &http.Client{}
	rs, err := httpClient.Get(c.url + "_cat/indices?format=json")
	if err != nil {
		return &[]Index{}, err
	}

	defer rs.Body.Close()

	rsBody, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		return &[]Index{}, err
	}

	indices := &[]Index{}
	err = json.Unmarshal(rsBody, indices)
	if err != nil {
		return &[]Index{}, err
	}

	c.indicies = *indices

	return indices, nil
}

func (c *client) RecreateIndex(name string, force bool) {
	exists, err := c.IndexExists(name)
	if err != nil {
		fmt.Println(err)
		return
	}
	if exists && force {
		dRs, err := c.DeleteIndex(name)
		if err != nil {
			fmt.Printf("Delete index error: %v\n", err)
			return
		}
		fmt.Printf("Delete index response: %v\n", dRs)
	} else {
		return
	}
	cRs, err := c.CreateIndex(name)
	if err != nil {
		fmt.Printf("Create index error: %v\n", err)
		return
	}
	fmt.Printf("Create index response: %v\n", cRs)
}

func (c *client) IndexExists(name string) (bool, error) {
	url := c.url + name

	httpClient := &http.Client{}
	rs, err := httpClient.Head(url)
	if err != nil {
		return false, err
	}

	return rs.StatusCode == http.StatusOK, nil
}

func (c *client) DeleteIndex(name string) (*ESResponse, error) {
	return sendRequest("DELETE", c.url+name)
}

func (c *client) CreateIndex(name string) (*ESResponse, error) {
	body := map[string]interface{}{
		"mappings": map[string]interface{}{
			"log": map[string]interface{}{
				"properties": map[string]interface{}{
					"test_item": map[string]interface{}{
						"type": "keyword",
					},
					"issue_type": map[string]interface{}{
						"type": "keyword",
					},
					"message": map[string]interface{}{
						"type":     "text",
						"analyzer": "standard",
					},
					"log_level": map[string]interface{}{
						"type": "integer",
					},
					"launch_name": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
		},
	}

	return sendRequest("PUT", c.url+name, body)
}

func (c *client) IndexLogs(name string, launch *Launch) (*ESResponse, error) {
	indexType := "log"

	var bodies []interface{}

	for _, ti := range launch.TestItems {
		for _, l := range ti.Logs {

			op := map[string]interface{}{
				"index": map[string]interface{}{
					"_index": name,
					"_type":  indexType,
					"_id":    l.LogID,
				},
			}

			bodies = append(bodies, op)

			body := map[string]interface{}{
				"launch_name": launch.LaunchName,
				"test_item":   ti.TestItemID,
				"issue_type":  ti.IssueType,
				"log_level":   l.LogLevel,
				"message":     l.Message,
			}

			bodies = append(bodies, body)
		}
	}

	if len(bodies) == 0 {
		return &ESResponse{}, nil
	}

	return sendRequest("PUT", c.url+"_bulk", bodies...)
}

func sendRequest(method, url string, bodies ...interface{}) (*ESResponse, error) {
	var rdr io.Reader

	nl := []byte("\n")
	if len(bodies) > 0 {
		buff := bytes.NewBuffer([]byte{})
		for _, body := range bodies {
			rqBody, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			buff.Write(rqBody)
			buff.Write(nl)
		}
		rdr = buff
	}

	rq, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, err
	}
	rq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	rs, err := client.Do(rq)
	if err != nil {
		return nil, err
	}
	defer rs.Body.Close()

	rsBody, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		return nil, err
	}

	umRs := &ESResponse{}
	err = json.Unmarshal(rsBody, umRs)
	if err != nil {
		return nil, err
	}

	return umRs, nil
}
