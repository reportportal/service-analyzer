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
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"math"
	"strconv"
)

var _ = Describe("ES Query Struct", func() {
	It("should build correct search request", func() {
		cfg := &SearchConfig{
			MinShouldMatch: "80%",
			MinTermFreq:    25,
			MinDocFreq:     30,
			BoostAA:        10,
			BoostLaunch:    5,
			BoostUniqueID:  3,
			MaxQueryTerms:  50
		}

		c := &client{searchCfg: cfg}
		launch := Launch{Conf: AnalyzerConf{Mode: SearchModeAll}, LaunchID: 123, LaunchName: "Launch name"}
		q1Struct := c.buildAnalyzeQuery(launch, "unique", "hello world")
		q2Struct := buildDemoQuery(cfg, SearchModeAll, "mylaynch", "unique", "hello world")

		q1B, _ := json.Marshal(q1Struct)
		var q1 map[string]interface{}
		if err := json.Unmarshal(q1B, &q1); err != nil {
			log.Error(err)
		}

		q2B, _ := json.Marshal(q2Struct)
		var q2 map[string]interface{}
		if err := json.Unmarshal(q2B, &q2); err != nil {
			log.Error(err)
		}

		Expect(q2).To(BeEquivalentTo(q2))
	})
})

func buildDemoQuery(searchCfg *SearchConfig, mode SearchMode, launchName, uniqueID, logMessage string) interface{} {
	return map[string]interface{}{
		"size": 10,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{
					"wildcard": map[string]interface{}{
						"issue_type": "ti*",
					},
				},
				"must": []interface{}{
					map[string]interface{}{
						"range": map[string]interface{}{
							"log_level": map[string]interface{}{
								"gte": ErrorLoggingLevel,
							},
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
							"min_doc_freq":         searchCfg.MinDocFreq,
							"min_term_freq":        searchCfg.MinTermFreq,
							"minimum_should_match": "5<"+searchCfg.MinShouldMatch,
							"max_query_terms":      searchCfg.MaxQueryTerms
						},
					},
				},
				"should": []map[string]interface{}{
					{"term": map[string]interface{}{
						"launch_name": map[string]interface{}{
							"value": launchName,
							"boost": math.Abs(searchCfg.BoostLaunch),
						},
					}},
					{"term": map[string]interface{}{
						"unique_id": map[string]interface{}{
							"value": uniqueID,
							"boost": math.Abs(searchCfg.BoostUniqueID),
						},
					}},
					{"term": map[string]interface{}{
						"is_auto_analyzed": map[string]interface{}{
							"value": strconv.FormatBool(searchCfg.BoostAA < 0),
							"boost": math.Abs(searchCfg.BoostAA),
						},
					}},
				},
			},
		},
	}
}
