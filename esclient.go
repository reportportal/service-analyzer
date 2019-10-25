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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

//ErrorLoggingLevel is integer representation of ERROR logging level
const ErrorLoggingLevel int = 40000

// ESClient interface
type ESClient interface {
	ListIndices() ([]Index, error)
	CreateIndex(name string) (*Response, error)
	IndexExists(name string) (bool, error)
	DeleteIndex(name int64) (*Response, error)

	IndexLogs(launches []Launch) (*BulkResponse, error)
	DeleteLogs(ci *CleanIndex) (*Response, error)
	AnalyzeLogs(launches []Launch) ([]AnalysisResult, error)
	SearchLogs(request SearchLogs) ([]int64, error)

	Healthy() bool

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
	LaunchID   int64        `json:"launchId,required" validate:"required"`
	Project    int64        `json:"project,required" validate:"required"`
	LaunchName string       `json:"launchName,omitempty"`
	Conf       AnalyzerConf `json:"analyzerConfig"`
	TestItems  []struct {
		TestItemID        int64  `json:"testItemId,required" validate:"required"`
		UniqueID          string `json:"uniqueId,required" validate:"required"`
		IsAutoAnalyzed    bool   `json:"isAutoAnalyzed,required" validate:"required"`
		IssueType         string `json:"issueType,omitempty"`
		OriginalIssueType string `json:"originalIssueType,omitempty"`
		Logs              []struct {
			LogID    int64  `json:"logId,required" validate:"required"`
			LogLevel int    `json:"logLevel,omitempty"`
			Message  string `json:"message,required" validate:"required"`
		} `json:"logs,omitempty"`
	} `json:"testItems,omitempty"`
}

// AnalyzerConf struct
type AnalyzerConf struct {
	MinDocFreq      float64    `json:"minDocFreq,omitempty"`
	MintTermFreq    float64    `json:"minTermFreq,omitempty"`
	MinShouldMatch  int        `json:"minShouldMatch,omitempty"`
	LogLines        int        `json:"numberOfLogLines,omitempty"`
	AAEnabled       bool       `json:"isAutoAnalyzerEnabled"`
	Mode            SearchMode `json:"analyzerMode"`
	IndexingRunning bool       `json:"indexingRunning"`
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
		Total    Total   `json:"total,omitempty"`
		MaxScore float64 `json:"max_score,omitempty"`
		Hits     []Hit   `json:"hits,omitempty"`
	} `json:"hits,omitempty"`
}

// Total struct
type Total struct {
	Value    int    `json:"value,omitempty"`
	Relation string `json:"relation,omitempty"`
}

//Hit is a single result from search index
type Hit struct {
	Index  string  `json:"_index,omitempty"`
	Type   string  `json:"_type,omitempty"`
	ID     string  `json:"_id,omitempty"`
	Score  float64 `json:"_score,omitempty"`
	Source struct {
		TestItem   int64  `json:"test_item,omitempty"`
		IssueType  string `json:"issue_type,omitempty"`
		Message    string `json:"message,omitempty"`
		LogLevel   int    `json:"log_level,omitempty"`
		LaunchName string `json:"launch_name,omitempty"`
	} `json:"_source,omitempty"`
}

//AnalysisResult represents result of analyzes which is basically array of found matches (predicted issue type and ID of most relevant Test Item)
type AnalysisResult struct {
	TestItem     int64  `json:"testItem,omitempty"`
	IssueType    string `json:"issueType,omitempty"`
	RelevantItem int64  `json:"relevantItem,omitempty"`
}

//CleanIndex is a request to clean index
type CleanIndex struct {
	IDs     []int64 `json:"ids,omitempty"`
	Project int64   `json:"project,required" validate:"required"`
}

//Search logs request
type SearchLogs struct {
	LaunchID          int64    `json:"launchId,omitempty"`
	LaunchName        string   `json:"launchName,omitempty"`
	ItemID            int64    `json:"itemId,omitempty"`
	ProjectID         int64    `json:"projectId,omitempty"`
	FilteredLaunchIds []int64  `json:"filteredLaunchIds,omitempty"`
	LogMessages       []string `json:"logMessages,omitempty"`
	LogLines          int      `json:"logLines"`
}

//Search logs config
type SearchLogConfig struct {
	Mode     string `json:"searchMode,omitempty"`
	LogLines int    `json:"numberOfLogLines,omitempty"`
}

type client struct {
	hosts     []string
	re        *regexp.Regexp
	hc        *http.Client
	searchCfg *SearchConfig
}

// NewClient creates new ESClient
func NewClient(hosts []string, searchCfg *SearchConfig) ESClient {
	return &client{
		hosts:     hosts,
		searchCfg: searchCfg,
		re:        regexp.MustCompile(`\d+`),
		hc:        &http.Client{},
	}
}

func (rs *Response) String() string {
	s, err := json.Marshal(rs)
	if err != nil {
		s = []byte{}
	}
	return fmt.Sprintf("%v", string(s))
}

//Healthy returns TRUE if cluster in operational state
func (c *client) Healthy() bool {
	var rs map[string]interface{}
	err := c.sendOpRequest("GET", c.buildURL("_cluster/health"), &rs, nil)
	if nil != err {
		return false
	}
	status := rs["status"]
	return "yellow" == status || "green" == status
}

func (c *client) ListIndices() ([]Index, error) {
	url := c.buildURL("_cat", "indices?format=json")

	indices := []Index{}

	err := c.sendOpRequest("GET", url, &indices)
	if err != nil {
		return nil, err
	}

	return indices, nil
}

func (c *client) CreateIndex(name string) (*Response, error) {
	log.Debugf("Creating index %s", name)

	body := map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards": 1,
		},
		"mappings": map[string]interface{}{
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
				"is_auto_analyzed": map[string]interface{}{
					"type": "keyword",
				},
			},
		},
	}

	url := c.buildURL(name)

	rs := &Response{}

	return rs, c.sendOpRequest(http.MethodPut, url, rs, body)
}

func (c *client) IndexExists(name string) (bool, error) {
	log.Debugf("Checking index %s", name)

	url := c.buildURL(name)

	httpClient := &http.Client{}
	rs, err := httpClient.Head(url)
	if err != nil {
		return false, errors.WithStack(err)
	}
	defer rs.Body.Close()
	return rs.StatusCode == http.StatusOK, nil
}

func (c *client) DeleteIndex(name int64) (*Response, error) {
	log.Debugf("Deleting index %d", name)
	url := c.buildURL(strconv.FormatInt(name, 10))
	rs := &Response{}
	return rs, c.sendOpRequest(http.MethodDelete, url, rs)
}

func (c *client) DeleteLogs(ci *CleanIndex) (*Response, error) {
	log.Debugf("Deleting logs %v", ci.IDs)
	url := c.buildURL("_bulk")
	url = url + "?refresh"
	rs := &Response{}
	bodies := make([]interface{}, len(ci.IDs))
	for i, id := range ci.IDs {
		bodies[i] = map[string]interface{}{
			"delete": map[string]interface{}{
				"_id":    id,
				"_index": ci.Project,
			},
		}
	}
	return rs, c.sendOpRequest(http.MethodPost, url, rs, bodies...)
}

func (c *client) IndexLogs(launches []Launch) (*BulkResponse, error) {
	log.Debugf("Indexing logs for %d launches", len(launches))

	var bodies []interface{}

	for _, lc := range launches {
		if err := c.createIndexIfNotExists(strconv.FormatInt(lc.Project, 10)); nil != err {
			return nil, errors.Wrap(err, "Cannot index logs")
		}
		for _, ti := range lc.TestItems {
			for _, l := range ti.Logs {

				op := map[string]interface{}{
					"index": map[string]interface{}{
						"_id":    l.LogID,
						"_index": lc.Project,
					},
				}

				bodies = append(bodies, op)

				message := c.sanitizeText(firstLines(l.Message, lc.Conf.LogLines))

				body := map[string]interface{}{
					"launch_id":        lc.LaunchID,
					"launch_name":      lc.LaunchName,
					"test_item":        ti.TestItemID,
					"unique_id":        ti.UniqueID,
					"is_auto_analyzed": ti.IsAutoAnalyzed,
					"issue_type":       ti.IssueType,
					"log_level":        l.LogLevel,
					"message":          message,
				}

				bodies = append(bodies, body)
			}
		}
	}

	rs := &BulkResponse{}

	if len(bodies) == 0 {
		return rs, nil
	}

	url := c.buildURL("_bulk?refresh")

	return rs, c.sendOpRequest(http.MethodPut, url, rs, bodies...)
}

func (c *client) AnalyzeLogs(launches []Launch) ([]AnalysisResult, error) {
	log.Debugf("Starting analysis for %d launches", len(launches))

	result := []AnalysisResult{}
	for _, lc := range launches {
		url := c.buildURL(strconv.FormatInt(lc.Project, 10), "_search")

		for _, ti := range lc.TestItems {
			issueTypes := make(map[string]*score)

			for _, l := range ti.Logs {
				message := c.sanitizeText(firstLines(l.Message, lc.Conf.LogLines))

				query := c.buildAnalyzeQuery(lc, ti.UniqueID, message)

				rs := &SearchResult{}
				err := c.sendOpRequest(http.MethodGet, url, rs, query)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				calculateScores(rs, 10, issueTypes)
			}

			var predictedIssueType string

			if len(issueTypes) > 0 {
				max := 0.0
				for k, v := range issueTypes {
					if v.score > max {
						max = v.score
						predictedIssueType = k
					}
				}
			}
			if "" != predictedIssueType {
				result = append(result, AnalysisResult{
					TestItem:     ti.TestItemID,
					RelevantItem: issueTypes[predictedIssueType].mrHit.Source.TestItem,
					IssueType:    predictedIssueType,
				})
			}

		}
	}
	log.Debugf("Analysis has found %d matches", len(result))

	return result, nil
}

func (c *client) SearchLogs(request SearchLogs) ([]int64, error) {
	set := make(map[int64]bool)
	for _, message := range request.LogMessages {
		url := c.buildURL(strconv.FormatInt(request.ProjectID, 10), "_search")
		sanitizedMsg := c.sanitizeText(firstLines(message, request.LogLines))
		query := c.buildSearchQuery(request, sanitizedMsg)

		response := &SearchResult{}
		err := c.sendOpRequest(http.MethodGet, url, response, query)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for _, log := range response.Hits.Hits {
			logIndex, err := strconv.ParseInt(log.ID, 10, 64)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			set[logIndex] = true
		}
	}
	keys := make([]int64, len(set))

	i := 0
	for k := range set {
		keys[i] = k
		i++
	}

	return keys, nil
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

func (c *client) buildAnalyzeQuery(launch Launch, uniqueID, logMessage string) interface{} {
	minDocFreq := launch.Conf.MinDocFreq
	if 0 == minDocFreq {
		minDocFreq = c.searchCfg.MinDocFreq
	}
	minTermFreq := launch.Conf.MintTermFreq
	if 0 == minTermFreq {
		minTermFreq = c.searchCfg.MinTermFreq
	}
	var minShouldMatch string
	if 0 == launch.Conf.MinShouldMatch {
		minShouldMatch = c.searchCfg.MinShouldMatch
	} else {
		minShouldMatch = fmt.Sprintf("%s%%", strconv.Itoa(launch.Conf.MinShouldMatch))
	}

	q := EsQueryRQ{
		Size: 10,
		Query: &EsQuery{
			Bool: &BoolCondition{
				MustNot: &Condition{
					Wildcard: map[string]interface{}{"issue_type": "ti*"},
				},
				Must: []Condition{
					{
						Range: map[string]interface{}{"log_level": map[string]interface{}{"gte": ErrorLoggingLevel}},
					},
					{
						Exists: &ExistsCondition{
							Field: "issue_type",
						},
					},
				},
				Should: []Condition{
					{
						Term: map[string]TermCondition{"unique_id": {uniqueID, NewBoost(math.Abs(c.searchCfg.BoostUniqueID))}},
					},
					{
						Term: map[string]TermCondition{"is_auto_analyzed": {strconv.FormatBool(c.searchCfg.BoostAA < 0), NewBoost(math.Abs(c.searchCfg.BoostAA))}},
					},
				},
			},
		}}
	switch launch.Conf.Mode {
	case SearchModeAll, SearchModeNotFound:
		q.Query.Bool.Should = append(q.Query.Bool.Should, Condition{
			Term: map[string]TermCondition{"launch_name": {launch.LaunchName, NewBoost(math.Abs(c.searchCfg.BoostLaunch))}},
		})
		q.Query.Bool.Must = append(q.Query.Bool.Must, c.buildMoreLikeThis(minDocFreq, minTermFreq, minShouldMatch, logMessage))
	case SearchModeLaunchName:
		q.Query.Bool.Must = append(q.Query.Bool.Must, Condition{
			Term: map[string]TermCondition{"launch_name": {Value: launch.LaunchName}},
		})
		q.Query.Bool.Must = append(q.Query.Bool.Must, c.buildMoreLikeThis(minDocFreq, minTermFreq, minShouldMatch, logMessage))
	case SearchModeCurrentLaunch:
		q.Query.Bool.Must = append(q.Query.Bool.Must, Condition{
			Term: map[string]TermCondition{"launch_id": {Value: launch.LaunchID}},
		})
		q.Query.Bool.Must = append(q.Query.Bool.Must, c.buildMoreLikeThis(float64(1), minTermFreq, minShouldMatch, logMessage))
	}

	return q
}

func (c *client) buildSearchQuery(request SearchLogs, logMessage string) interface{} {
	q := EsQueryRQ{
		Size: 500,
		Query: &EsQuery{
			Bool: &BoolCondition{
				MustNot: &Condition{
					Term: map[string]TermCondition{"test_item": {request.ItemID, NewBoost(1.0)}},
				},
				Must: []Condition{
					{
						Range: map[string]interface{}{"log_level": map[string]interface{}{"gte": ErrorLoggingLevel}},
					},
					{
						Exists: &ExistsCondition{
							Field: "issue_type",
						},
					},
					{
						Wildcard: map[string]interface{}{"issue_type": "ti*"},
					},
				},
				Should: []Condition{
					{
						Term: map[string]TermCondition{"is_auto_analyzed": {"false", NewBoost(1.0)}},
					},
				},
			},
		}}

	q.Query.Bool.Must = append(q.Query.Bool.Must, Condition{
		Terms: map[string][]int64{"launch_id": request.FilteredLaunchIds},
	})
	q.Query.Bool.Must = append(q.Query.Bool.Must, c.buildMoreLikeThis(1, 1, c.searchCfg.SearchLogsMinShouldMatch, logMessage))

	return q
}

func (c *client) buildMoreLikeThis(minDocFreq, minTermFreq float64, minShouldMatch, logMessage string) Condition {
	return Condition{
		MoreLikeThis: &MoreLikeThisCondition{
			Fields:         []string{"message"},
			Like:           logMessage,
			MinDocFreq:     minDocFreq,
			MinTermFreq:    minTermFreq,
			MinShouldMatch: minShouldMatch,
		},
	}
}

//score represents total score for defect type
//mrHit is hit with highest score found (most relevant hit)
type score struct {
	score float64
	mrHit Hit
}

func calculateScores(rs *SearchResult, k int, scores map[string]*score) {
	if rs.Hits.Total.Value > 0 {
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

			//find the hit with highest score for each defect type.
			//item from the hit will be used as most relevant of request is analyzed successfully
			if typeScore, ok := scores[h.Source.IssueType]; ok {
				if h.Score > typeScore.mrHit.Score {
					typeScore.mrHit = h
				}
			} else {
				scores[h.Source.IssueType] = &score{mrHit: h}
			}

		}
		for _, h := range hits {
			typeScore, ok := scores[h.Source.IssueType]
			currScore := h.Score / totalScore
			if ok {
				typeScore.score += currScore
			} else {
				//should never happen
				log.Errorf("Internal error during AA score calculation. Missed issue type: %s", h.Source.IssueType)
				scores[h.Source.IssueType] = &score{currScore, h}
			}
		}
	}
}

func (c *client) sendOpRequest(method, url string, response interface{}, bodies ...interface{}) error {
	rs, err := c.sendRequest(method, url, bodies...)
	if err != nil {
		return errors.WithStack(err)
	}

	err = json.Unmarshal(rs, &response)
	if err != nil {
		return errors.Wrap(err, "Cannot unmarshal ES OP response")
	}

	return nil
}

func (c *client) sendRequest(method, url string, bodies ...interface{}) ([]byte, error) {
	var rdr io.Reader

	nl := []byte("\n")
	if len(bodies) > 0 {
		buff := bytes.NewBuffer([]byte{})
		for _, body := range bodies {
			rqBody, err := json.Marshal(body)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			// nolint
			buff.Write(rqBody)
			// nolint
			buff.Write(nl)
		}
		rdr = buff
	}

	rq, err := http.NewRequest(method, url, rdr)
	log.Debugf("Request to ES - method: %q;\n url: %q;\n body: %v", method, url, rdr)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot build request to ES")
	}
	rq.Header.Set("Content-Type", "application/json")

	rs, err := c.hc.Do(rq)
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

	if rs.StatusCode > http.StatusCreated && rs.StatusCode < http.StatusNotFound {
		body := string(rsBody)
		log.Errorf("ES communication error. Status code %d, Body %s", rs.StatusCode, body)
		return nil, errors.New(body)
	}

	log.Debugf("Response from ES - %v", string(rsBody))

	return rsBody, nil
}

// findNth searches for the nth occurrence of string
func findNth(str, f string, n int) int {
	i := 0
	for m := 1; m <= n; m++ {
		x := strings.Index(str[i:], f)
		if x < 0 {
			return x
		}
		if m == n {
			return x + i
		}
		i += x + len(f)
	}
	return -1
}

// findNth searches for the nth occurrence of string
func firstLines(str string, n int) string {
	sep := findNth(str, "\n", n)
	if sep > 0 {
		return str[0:sep]
	}
	return str
}
