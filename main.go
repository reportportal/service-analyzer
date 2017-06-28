package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		name := os.Args[1]
		recreateIndex(name)
	}
}

// {
//   "error" : {
//     "root_cause" : [
//       {
//         "type" : "not_x_content_exception",
//         "reason" : "Compressor detection can only be called on some xcontent bytes or compressed xcontent bytes"
//       }
//     ],
//     "type" : "not_x_content_exception",
//     "reason" : "Compressor detection can only be called on some xcontent bytes or compressed xcontent bytes"
//   },
//   "status" : 500
// }

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
	IssueType  int    `json:"issueType"`
	Logs       []Log  `json:"logs"`
}

// Launch struct
type Launch struct {
	LaunchID   string     `json:"launchId"`
	LaunchName string     `json:"launchName"`
	TestItems  []TestItem `json:"testItems"`
}

func (rs *ESResponse) String() string {
	s, err := json.Marshal(rs)
	if err != nil {
		s = []byte{}
	}
	return fmt.Sprintf("%v", string(s))
}

func recreateIndex(name string) {
	exists, err := indexExists(name)
	if err != nil {
		fmt.Println(err)
		return
	}
	if exists {
		dRs, err := deleteIndex(name)
		if err != nil {
			fmt.Printf("Delete index error: %v\n", err)
			return
		}
		fmt.Printf("Delete index response: %v\n", dRs)
	}
	cRs, err := createIndex(name)
	if err != nil {
		fmt.Printf("Create index error: %v\n", err)
		return
	}
	fmt.Printf("Create index response: %v\n", cRs)
}

func indexExists(name string) (bool, error) {
	url := "http://localhost:9200/" + name

	c := &http.Client{}
	rs, err := c.Head(url)
	if err != nil {
		return false, err
	}

	return rs.StatusCode == http.StatusOK, nil
}

func deleteIndex(name string) (*ESResponse, error) {
	url := "http://localhost:9200/" + name

	return sendRequest("DELETE", url)
}

func createIndex(name string) (*ESResponse, error) {
	url := "http://localhost:9200/" + name

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

	return sendRequest("PUT", url, body)
}

func indexLogs(launch Launch) {
	indexName := ""
	indexType := "log"

	bodies := make([]interface{}, 100)

	i := 0

	for _, ti := range launch.TestItems {
		for _, l := range ti.Logs {

			op := map[string]interface{}{
				"index": map[string]interface{}{
					"_index": indexName,
					"_type":  indexType,
					"_id":    l.LogID,
				},
			}

			bodies[i] = op

			i++

			body := map[string]interface{}{
				"launch_name": launch.LaunchName,
				"test_item":   ti.TestItemID,
				"issue_type":  ti.IssueType,
				"log_level":   l.LogLevel,
			}

			bodies[i] = body

			i++
		}
	}
}

func sendRequest(method, url string, bodies ...interface{}) (*ESResponse, error) {
	var rdr io.Reader

	if len(bodies) > 0 {
		buff := bytes.NewBuffer([]byte{})
		for _, body := range bodies {
			rqBody, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			buff.Write(rqBody)
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

// {
//   "size": 1,
//   "query": {
//     "bool": {
//       "must_not": {
//         "wildcard":  { "issue_type": "TI*" }
//       },
//       "must": [
//         {
//           "term": { "log_level": 40000 }
//         },
//         {
//           "exists": { "field": "issue_type" }
//         },
//         {
//           "more_like_this": {
//             "fields": ["message"],
//             "like": "xxx",
//             "minimum_should_match" : "90%"
//           }
//         }
//       ],
//       "should": {
//         "term": {
//           "launch_name": {
//             "value": "xxx",
//             "boost": 2.0
//           }
//         }
//       }
//     }
//   }
// }

// {
//   "mappings": {
//     "log": {
//       "properties": {
//         "test_item": {
//           "type": "keyword"
//         },
//         "issue_type": {
//           "type": "keyword"
//         },
//         "message": {
//           "type":     "text",
//           "analyzer": "standard"
//         },
//         "log_level": {
//           "type": "integer"
//         },
//         "launch_name": {
//           "type": "keyword"
//         }
//       }
//     }
//   }
// }
