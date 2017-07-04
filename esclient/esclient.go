package esclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
)

// ESClient interface
type ESClient interface {
	ListIndices() (*[]Index, error)
	IndexExists(name string) (bool, error)
	CreateIndex(name string) (*ESResponse, error)
	DeleteIndex(name string) (*ESResponse, error)
	RecreateIndex(name string, force bool)
	IndexLogs(name string, launch *Launch) (*ESResponse, error)
	AnalyzeLogs(name string, launch *Launch) (*Launch, error)
	SanitizeText(text string) string
}

// ESResponse struct
type ESResponse struct {
	Acknowledged bool `json:"acknowledged"`
	Error        struct {
		RootCause []struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"root_cause"`
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
	Status int `json:"status"`
}

// Launch struct
type Launch struct {
	LaunchID   string `json:"launchId"`
	LaunchName string `json:"launchName"`
	TestItems  []struct {
		TestItemID        string `json:"testItemId"`
		IssueType         string `json:"issueType"`
		OriginalIssueType string `json:"originalIssueType"`
		Logs              []struct {
			LogID    string `json:"logId"`
			LogLevel int    `json:"logLevel"`
			Message  string `json:"message"`
		} `json:"logs"`
	} `json:"testItems"`
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

// SearchQueryResponse struct
type SearchQueryResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     struct {
		Total    int     `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string  `json:"_index"`
			Type   string  `json:"_type"`
			ID     string  `json:"_id"`
			Score  float64 `json:"_score"`
			Source struct {
				TestItem   string `json:"test_item"`
				IssueType  string `json:"issue_type"`
				Message    string `json:"message"`
				LogLevel   int    `json:"log_level"`
				LaunchName string `json:"launch_name"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type client struct {
	url      string
	indicies []Index
	re       *regexp.Regexp
}

// NewClient creates new ESClient
func NewClient(url string) ESClient {
	c := &client{}
	c.url = url
	c.indicies = []Index{}
	c.re = regexp.MustCompile("\\d+")
	return c
}

func (rs *ESResponse) String() string {
	s, err := json.Marshal(rs)
	if err != nil {
		s = []byte{}
	}
	return fmt.Sprintf("%v", string(s))
}

func (c *client) ListIndices() (*[]Index, error) {
	rs, err := sendRequest("GET", c.url+"_cat/indices?format=json", nil)
	if err != nil {
		return nil, err
	}

	indices := &[]Index{}
	err = json.Unmarshal(rs, indices)
	if err != nil {
		return nil, err
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
	if exists {
		if force {
			dRs, err := c.DeleteIndex(name)
			if err != nil {
				fmt.Printf("Delete index error: %v\n", err)
				return
			}
			fmt.Printf("Delete index response: %v\n", dRs)
		} else {
			return
		}
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
	return sendOpRequest("DELETE", c.url+name)
}

func (c *client) CreateIndex(name string) (*ESResponse, error) {
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
				},
			},
		},
	}

	return sendOpRequest("PUT", c.url+name, body)
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

			message := c.SanitizeText(l.Message)

			body := map[string]interface{}{
				"launch_name": launch.LaunchName,
				"test_item":   ti.TestItemID,
				"issue_type":  ti.IssueType,
				"log_level":   l.LogLevel,
				"message":     message,
			}

			bodies = append(bodies, body)
		}
	}

	if len(bodies) == 0 {
		return &ESResponse{}, nil
	}

	return sendOpRequest("PUT", c.url+"_bulk", bodies...)
}

func (c *client) AnalyzeLogs(name string, launch *Launch) (*Launch, error) {
	for i, ti := range launch.TestItems {

		issueTypes := make(map[string]float64)

		for _, l := range ti.Logs {
			message := c.SanitizeText(l.Message)

			query := buildQuery(launch.LaunchName, message)

			rs, err := sendRequest("GET", c.url+name+"/log/_search", query)

			if err != nil {
				return nil, err
			}

			esRs := &SearchQueryResponse{}
			err = json.Unmarshal(rs, esRs)
			if err != nil {
				return nil, err
			}

			// Two iterations over hits needed
			// to achieve stable prediction
			if esRs.Hits.Total > 0 {
				k := 10
				n := len(esRs.Hits.Hits)
				if n < k {
					k = n
				}
				totalScore := 0.0
				hits := esRs.Hits.Hits[:k]
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

			// if esRs.Hits.Total > 0 {
			// 	k := 10
			// 	n := len(esRs.Hits.Hits)
			// 	if n < k {
			// 		k = n
			// 	}
			// 	hits := esRs.Hits.Hits[:k]
			// 	for _, h := range hits {
			// 		score, ok := issueTypes[h.Source.IssueType]
			// 		if ok {
			// 			score += 1.0
			// 		} else {
			// 			score = 1.0
			// 		}
			// 		issueTypes[h.Source.IssueType] = score
			// 	}
			// }
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

		launch.TestItems[i].IssueType = predictedIssueType
	}

	return launch, nil
}

func (c *client) SanitizeText(text string) string {
	return c.re.ReplaceAllString(text, "")
}

func buildQuery(launchName, logMessage string) interface{} {
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
							"log_level": 40000,
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
				"should": map[string]interface{}{
					"term": map[string]interface{}{
						"launch_name": map[string]interface{}{
							"value": launchName,
							"boost": 2.0,
						},
					},
				},
			},
		},
	}
}

func sendOpRequest(method, url string, bodies ...interface{}) (*ESResponse, error) {
	rs, err := sendRequest(method, url, bodies...)
	if err != nil {
		return nil, err
	}

	esRs := &ESResponse{}
	err = json.Unmarshal(rs, esRs)
	if err != nil {
		return nil, err
	}

	return esRs, nil
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

	return rsBody, nil
}
