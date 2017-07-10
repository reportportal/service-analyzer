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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/assert"
)

const (
	TwoIndicesRs           = "two_indices_rs.json"
	IndexCreatedRs         = "index_created_rs.json"
	IndexAlreadyExistsRs   = "index_already_exists_rs.json"
	IndexDeletedRs         = "index_deleted_rs.json"
	IndexNotFoundRs        = "index_not_found_rs.json"
	LaunchWoTestItems      = "launch_wo_test_items.json"
	LaunchWTestItemsWoLogs = "launch_w_test_items_wo_logs.json"
	LaunchWTestItemsWLogs  = "launch_w_test_items_w_logs.json"
	IndexLogsRq            = "index_logs_rq.json"
	IndexLogsRs            = "index_logs_rs.json"
	SearchRq               = "search_rq.json"
	NoHitsSearchRs         = "no_hits_search_rs.json"
	OneHitSearchRs         = "one_hit_search_rs.json"
	TwoHitsSearchRs        = "two_hits_search_rs.json"
	ThreeHitsSearchRs      = "three_hits_search_rs.json"
)

func TestListIndices(t *testing.T) {
	tests := []struct {
		params        map[string]interface{}
		expectedCount int
		expectErr     bool
	}{
		{
			params: map[string]interface{}{
				"statusCode": http.StatusOK,
				"response":   "[]",
			},
			expectedCount: 0,
			expectErr:     false,
		},
		{
			params: map[string]interface{}{
				"statusCode": http.StatusOK,
				"response":   getFixture(TwoIndicesRs),
			},
			expectedCount: 2,
			expectErr:     false,
		},
		{
			params: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
			},
			expectedCount: 0,
			expectErr:     true,
		},
	}

	for _, test := range tests {
		ts := startServer(t, "GET", "/_cat/indices?format=json", test.params)
		defer ts.Close()
		c := NewClient(ts.URL)

		indices, err := c.ListIndices()
		if test.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			idxs := *indices
			assert.Equal(t, test.expectedCount, len(idxs))
			for j, idx := range idxs {
				assert.Equal(t, fmt.Sprintf("idx%d", j), idx.Index)
			}
		}
	}
}

func TestCreateIndex(t *testing.T) {
	tests := []struct {
		params    map[string]interface{}
		expectErr bool
	}{
		{
			params: map[string]interface{}{
				"indexName":  "idx0",
				"statusCode": http.StatusOK,
				"response":   getFixture(IndexCreatedRs),
			},
			expectErr: false,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx1",
				"statusCode": http.StatusBadRequest,
				"response":   getFixture(IndexAlreadyExistsRs),
			},
			expectErr: true,
		},
	}

	for _, test := range tests {
		indexName := test.params["indexName"].(string)
		ts := startServer(t, "PUT", "/"+indexName, test.params)
		defer ts.Close()
		c := NewClient(ts.URL)

		rs, err := c.CreateIndex(indexName)
		if test.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.True(t, rs.Acknowledged)
		}
	}
}

func TestIndexExists(t *testing.T) {
	tests := []struct {
		params map[string]interface{}
		exists bool
	}{
		{
			params: map[string]interface{}{
				"indexName":  "idx0",
				"statusCode": http.StatusOK,
			},
			exists: true,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx1",
				"statusCode": http.StatusNotFound,
			},
			exists: false,
		},
	}

	for _, test := range tests {
		indexName := test.params["indexName"].(string)
		ts := startServer(t, "HEAD", "/"+indexName, test.params)
		defer ts.Close()
		c := NewClient(ts.URL)

		exists, err := c.IndexExists(indexName)
		assert.NoError(t, err)
		assert.Equal(t, test.exists, exists)
	}
}

func TestDeleteIndex(t *testing.T) {
	tests := []struct {
		params         map[string]interface{}
		expectedStatus int
	}{
		{
			params: map[string]interface{}{
				"indexName":  "idx0",
				"statusCode": http.StatusOK,
				"response":   getFixture(IndexDeletedRs),
			},
			expectedStatus: 0,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx1",
				"statusCode": http.StatusNotFound,
				"response":   getFixture(IndexNotFoundRs),
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		indexName := test.params["indexName"].(string)
		ts := startServer(t, "DELETE", "/"+indexName, test.params)
		defer ts.Close()
		c := NewClient(ts.URL)

		rs, err := c.DeleteIndex(indexName)
		assert.NoError(t, err)
		assert.Equal(t, test.expectedStatus, rs.Status)
	}
}

func TestIndexLogs(t *testing.T) {
	tests := []struct {
		params           map[string]interface{}
		indexRequest     string
		expectServerCall bool
	}{
		{
			params: map[string]interface{}{
				"indexName": "idx0",
			},
			indexRequest:     getFixture(LaunchWoTestItems),
			expectServerCall: false,
		},
		{
			params: map[string]interface{}{
				"indexName": "idx1",
			},
			indexRequest:     getFixture(LaunchWTestItemsWoLogs),
			expectServerCall: false,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx2",
				"request":    getFixture(IndexLogsRq),
				"response":   getFixture(IndexLogsRs),
				"statusCode": http.StatusOK,
			},
			indexRequest:     getFixture(LaunchWTestItemsWLogs),
			expectServerCall: true,
		},
	}

	for _, test := range tests {
		var esURL string
		indexName := test.params["indexName"].(string)
		if test.expectServerCall {
			ts := startServer(t, "PUT", "/_bulk", test.params)
			defer ts.Close()
			esURL = ts.URL
		}
		c := NewClient(esURL)

		launch := &Launch{}
		err := json.Unmarshal([]byte(test.indexRequest), launch)
		assert.NoError(t, err)

		_, err = c.IndexLogs(indexName, launch)
		assert.NoError(t, err)
	}
}

func TestAnalyzeLogs(t *testing.T) {
	tests := []struct {
		params            map[string]interface{}
		analyzeRequest    string
		expectedIssueType string
		serverCallCount   int
	}{
		{
			params: map[string]interface{}{
				"indexName": "idx0",
			},
			analyzeRequest: getFixture(LaunchWoTestItems),
		},
		{
			params: map[string]interface{}{
				"indexName": "idx1",
			},
			analyzeRequest: getFixture(LaunchWTestItemsWoLogs),
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx2",
				"request":    getFixture(SearchRq),
				"response":   getFixture(NoHitsSearchRs),
				"statusCode": http.StatusOK,
			},
			analyzeRequest:    getFixture(LaunchWTestItemsWLogs),
			expectedIssueType: "",
			serverCallCount:   2,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx2",
				"request":    getFixture(SearchRq),
				"response":   getFixture(OneHitSearchRs),
				"statusCode": http.StatusOK,
			},
			analyzeRequest:    getFixture(LaunchWTestItemsWLogs),
			expectedIssueType: "AB001",
			serverCallCount:   2,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx3",
				"request":    getFixture(SearchRq),
				"response":   getFixture(TwoHitsSearchRs),
				"statusCode": http.StatusOK,
			},
			analyzeRequest:    getFixture(LaunchWTestItemsWLogs),
			expectedIssueType: "AB001",
			serverCallCount:   2,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx4",
				"request":    getFixture(SearchRq),
				"response":   getFixture(ThreeHitsSearchRs),
				"statusCode": http.StatusOK,
			},
			analyzeRequest:    getFixture(LaunchWTestItemsWLogs),
			expectedIssueType: "PB001",
			serverCallCount:   2,
		},
	}

	for _, test := range tests {
		var esURL string
		indexName := test.params["indexName"].(string)
		if test.serverCallCount > 0 {
			ts := startServer(t, "GET", "/"+indexName+"/log/_search", test.params)
			defer ts.Close()
			esURL = ts.URL
		}
		c := NewClient(esURL)

		launch := &Launch{}
		err := json.Unmarshal([]byte(test.analyzeRequest), launch)
		assert.NoError(t, err)

		launch, err = c.AnalyzeLogs(indexName, launch)
		assert.NoError(t, err)

		if test.expectedIssueType != "" {
			assert.Equal(t, test.expectedIssueType, launch.TestItems[0].IssueType)
		}
	}
}

func getFixture(filename string) string {
	f, _ := ioutil.ReadFile("fixtures/" + filename)
	return string(f)
}

func startServer(t *testing.T, expectedMethod string, expectedURI string, params map[string]interface{}) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedMethod, r.Method)
		assert.Equal(t, expectedURI, r.URL.RequestURI())
		expectedRq, ok := params["request"]
		if ok {
			defer r.Body.Close()
			rq, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, expectedRq, string(rq))
		}
		w.WriteHeader(params["statusCode"].(int))
		rs, ok := params["response"]
		if ok {
			w.Write([]byte(rs.(string)))
		}
	}))

	return ts
}
