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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

//ErrorLoggingLevel is integer representation of ERROR logging level
const ErrorLoggingLevel int = 40000

// ESClient interface
type ESClient interface {
	ListIndices() ([]Index, error)
	CreateIndex(name string) (*Response, error)
	IndexExists(name string) (bool, error)
	DeleteIndex(name string) (*Response, error)

	IndexLogs(launches []Launch) (*BulkResponse, error)
	AnalyzeLogs(launches []Launch) ([]Launch, error)

	createIndexIfNotExists(indexName string) error
	buildURL(pathElements ...string) string
	sanitizeText(text string) string
}

// Response struct
type Response struct {
	Acknowledged bool `json:"acknowledged,omitempty"`
	Error        struct {
		RootCause []struct {
			Type   string `json:"type,omitempty"`
			Reason string `json:"reason,omitempty"`
		} `json:"root_cause,omitempty"`
		Type   string `json:"type,omitempty"`
		Reason string `json:"reason,omitempty"`
	} `json:"error,omitempty"`
	Status int `json:"status,omitempty"`
}

// BulkResponse struct
type BulkResponse struct {
	Took   int  `json:"took,omitempty"`
	Errors bool `json:"errors,omitempty"`
	Items  []struct {
		Index struct {
			Index   string `json:"_index,omitempty"`
			Type    string `json:"_type,omitempty"`
			ID      string `json:"_id,omitempty"`
			Version int    `json:"_version,omitempty"`
			Result  string `json:"result,omitempty"`
			Created bool   `json:"created,omitempty"`
			Status  int    `json:"status,omitempty"`
		} `json:"index,omitempty"`
	} `json:"items,omitempty"`
	Status int `json:"status,omitempty"`
}

// Launch struct
type Launch struct {
	LaunchID   string `json:"launchId,required"`
	Project    string `json:"project,required"`
	LaunchName string `json:"launchName,omitempty"`
	TestItems  []struct {
		TestItemID        string `json:"testItemId,required"`
		UniqueID          string `json:"uniqueId,required"`
		IssueType         string `json:"issueType,omitempty"`
		OriginalIssueType string `json:"originalIssueType,omitempty"`
		Logs              []struct {
			LogLevel int    `json:"logLevel,omitempty"`
			Message  string `json:"message,required"`
		} `json:"logs,omitempty"`
	} `json:"testItems,omitempty"`
}

// Index struct
type Index struct {
	Health       string `json:"health,omitempty"`
	Status       string `json:"status,omitempty"`
	Index        string `json:"index,omitempty"`
	UUID         string `json:"uuid,omitempty"`
	Pri          string `json:"pri,omitempty"`
	Rep          string `json:"rep,omitempty"`
	DocsCount    string `json:"docs.count,omitempty"`
	DocsDeleted  string `json:"docs.deleted,omitempty"`
	StoreSize    string `json:"store.size,omitempty"`
	PriStoreSize string `json:"pri.store.size,omitempty"`
}

// SearchResult struct
type SearchResult struct {
	Took     int  `json:"took,omitempty"`
	TimedOut bool `json:"timed_out,omitempty"`
	Hits     struct {
		Total    int     `json:"total,omitempty"`
		MaxScore float64 `json:"max_score,omitempty"`
		Hits     []struct {
			Index  string  `json:"_index,omitempty"`
			Type   string  `json:"_type,omitempty"`
			ID     string  `json:"_id,omitempty"`
			Score  float64 `json:"_score,omitempty"`
			Source struct {
				TestItem   string `json:"test_item,omitempty"`
				IssueType  string `json:"issue_type,omitempty"`
				Message    string `json:"message,omitempty"`
				LogLevel   int    `json:"log_level,omitempty"`
				LaunchName string `json:"launch_name,omitempty"`
			} `json:"_source,omitempty"`
		} `json:"hits,omitempty"`
	} `json:"hits,omitempty"`
}

type client struct {
	hosts []string
	re    *regexp.Regexp
}

// NewClient creates new ESClient
func NewClient(hosts []string) ESClient {
	c := &client{}
	c.hosts = hosts
	c.re = regexp.MustCompile("\\d+")
	return c
}

func (rs *Response) String() string {
	s, err := json.Marshal(rs)
	if err != nil {
		s = []byte{}
	}
	return fmt.Sprintf("%v", string(s))
}

func (c *client) ListIndices() ([]Index, error) {
	url := c.buildURL("_cat", "indices?format=json")

	indices := []Index{}

	err := sendOpRequest("GET", url, &indices)
	if err != nil {
		return nil, err
	}

	return indices, nil
}

func (c *client) CreateIndex(name string) (*Response, error) {
	body := map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards": 1,
		},
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
					"unique_id": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
		},
	}

	url := c.buildURL(name)

	rs := &Response{}

	return rs, sendOpRequest(http.MethodPut, url, rs, body)
}

func (c *client) IndexExists(name string) (bool, error) {
	url := c.buildURL(name)

	httpClient := &http.Client{}
	rs, err := httpClient.Head(url)
	if err != nil {
		return false, err
	}

	return rs.StatusCode == http.StatusOK, nil
}

func (c *client) DeleteIndex(name string) (*Response, error) {
	url := c.buildURL(name)
	rs := &Response{}
	return rs, sendOpRequest(http.MethodDelete, url, rs)
}

func (c *client) IndexLogs(launches []Launch) (*BulkResponse, error) {
	indexType := "log"

	var bodies []interface{}

	for _, lc := range launches {
		c.createIndexIfNotExists(lc.Project)
		for _, ti := range lc.TestItems {
			for _, l := range ti.Logs {

				op := map[string]interface{}{
					"index": map[string]interface{}{
						"_index": lc.Project,
						"_type":  indexType,
					},
				}

				bodies = append(bodies, op)

				message := c.sanitizeText(l.Message)

				body := map[string]interface{}{
					"launch_name": lc.LaunchName,
					"test_item":   ti.TestItemID,
					"unique_id":   ti.UniqueID,
					"issue_type":  ti.IssueType,
					"log_level":   l.LogLevel,
					"message":     message,
				}

				bodies = append(bodies, body)
			}
		}
	}

	rs := &BulkResponse{}

	if len(bodies) == 0 {
		return rs, nil
	}

	url := c.buildURL("_bulk")

	return rs, sendOpRequest(http.MethodPut, url, rs, bodies...)
}

func (c *client) AnalyzeLogs(launches []Launch) ([]Launch, error) {
	for _, lc := range launches {
		url := c.buildURL(lc.Project, "log", "_search")

		for j, ti := range lc.TestItems {
			issueTypes := make(map[string]float64)

			for _, l := range ti.Logs {
				message := c.sanitizeText(l.Message)

				query := buildQuery(lc.LaunchName, ti.UniqueID, message)

				rs := &SearchResult{}
				err := sendOpRequest(http.MethodGet, url, rs, query)
				if err != nil {
					return nil, err
				}

				calculateScores(rs, 10, issueTypes)
			}

			var predictedIssueType string

			if len(issueTypes) > 0 {
				max := 0.0
				for k, v := range issueTypes {
					if v > max {
						max = v
						predictedIssueType = k
					}
				}
			}

			lc.TestItems[j].IssueType = predictedIssueType
		}
	}

	return launches, nil
}

func (c *client) createIndexIfNotExists(indexName string) error {
	exists, err := c.IndexExists(indexName)
	if err != nil {
		return errors.Wrap(err, "Cannot check ES index exists")
	}
	if !exists {
		_, err = c.CreateIndex(indexName)
	}
	return errors.Wrap(err, "Cannot create ES index")
}

func (c *client) sanitizeText(text string) string {
	return c.re.ReplaceAllString(text, "")
}

func (c *client) buildURL(pathElements ...string) string {
	return c.hosts[0] + "/" + strings.Join(pathElements, "/")
}

func buildQuery(launchName, uniqueID, logMessage string) interface{} {
	return map[string]interface{}{
		"size": 10,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{
					"wildcard": map[string]interface{}{
						"issue_type": "TI*",
					},
				},
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"log_level": ErrorLoggingLevel,
						},
					},
					map[string]interface{}{
						"exists": map[string]interface{}{
							"field": "issue_type",
						},
					},
					map[string]interface{}{
						"more_like_this": map[string]interface{}{
							"fields":               []string{"message"},
							"like":                 logMessage,
							"minimum_should_match": "90%",
						},
					},
				},
				"should": []map[string]interface{}{
					{"term": map[string]interface{}{
						"launch_name": map[string]interface{}{
							"value": launchName,
							"boost": 2.0,
						},
					}},
					{"term": map[string]interface{}{
						"unique_id": map[string]interface{}{
							"value": uniqueID,
							"boost": 2.0,
						},
					}},
				},
			},
		},
	}
}

func calculateScores(rs *SearchResult, k int, issueTypes map[string]float64) {
	if rs.Hits.Total > 0 {
		n := len(rs.Hits.Hits)
		if n < k {
			k = n
		}
		totalScore := 0.0
		hits := rs.Hits.Hits[:k]
		// Two iterations over hits needed
		// to achieve stable prediction
		for _, h := range hits {
			totalScore += h.Score
		}
		for _, h := range hits {
			typeScore, ok := issueTypes[h.Source.IssueType]
			score := h.Score / totalScore
			if ok {
				typeScore += score
			} else {
				typeScore = score
			}
			issueTypes[h.Source.IssueType] = typeScore
		}
	}
}

func sendOpRequest(method, url string, response interface{}, bodies ...interface{}) error {
	rs, err := sendRequest(method, url, bodies...)
	if err != nil {

		return err
	}

	err = json.Unmarshal(rs, &response)
	if err != nil {
		return errors.Wrap(err, "Cannot unmarshal ES OP response")
	}

	return nil
}

func sendRequest(method, url string, bodies ...interface{}) ([]byte, error) {
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
		return nil, errors.Wrap(err, "Cannot build request to ES")
	}
	rq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	rs, err := client.Do(rq)
	if err != nil {
		log.Errorf("Cannot send request to ES: %s", err.Error())

		return nil, errors.Wrap(err, "Cannot send request to ES")
	}
	defer rs.Body.Close()

	rsBody, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		log.Errorf("Cannot ES response: %s", err.Error())
		return nil, errors.Wrap(err, "Cannot read ES response")
	}

	log.Infof("ES responded with %d status code", rs.StatusCode)
	if rs.StatusCode > http.StatusCreated && rs.StatusCode < http.StatusNotFound {
		body := string(rsBody)
		log.Errorf("ES communication error. Status code %d, Body %s", rs.StatusCode, body)
		return nil, errors.New(body)
	}

	return rsBody, nil
}
