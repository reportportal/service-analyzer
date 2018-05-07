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
		}

		c := &client{searchCfg: cfg}
		launch := Launch{Conf: AnalyzerConf{Mode: SearchModeAll}, LaunchID: "123", LaunchName: "Launch name"}
		q1Struct := c.buildQuery(launch, "unique", "hello world")
		q2Struct := buildDemoQuery(cfg, SearchModeAll, "mylaynch", "unique", "hello world")

		q1B, _ := json.Marshal(q1Struct)
		var q1 map[string]interface{}
		json.Unmarshal(q1B, &q1)

		q2B, _ := json.Marshal(q2Struct)
		var q2 map[string]interface{}
		json.Unmarshal(q2B, &q2)

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
						"issue_type": "TI*",
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
							"minimum_should_match": searchCfg.MinShouldMatch,
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
