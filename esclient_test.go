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
	TwoIndicesRs = `
	[
		{
			"health": "yellow",
			"status": "open",
			"index": "idx0",
			"uuid": "sGD-VQy5StS1jIUbuo3R7A",
			"pri": "1",
			"rep": "1",
			"docs.count": "353400",
			"docs.deleted": "0",
			"store.size": "37.9mb",
			"pri.store.size": "37.9mb"
		},
		{
			"health": "yellow",
			"status": "open",
			"index": "idx1",
			"uuid": "DoA20IojS72IdaFSN8CX9Q",
			"pri": "1",
			"rep": "1",
			"docs.count": "38771",
			"docs.deleted": "0",
			"store.size": "11.2mb",
			"pri.store.size": "11.2mb"
		}
	]`
	IndexCreatedRs = `
	{
		"acknowledged" : true,
		"shards_acknowledged" : true
	}`
	IndexAlreadyExistsRs = `
	{
		"error" : {
			"root_cause" : [
				{
					"type" : "index_already_exists_exception",
					"reason" : "index [idx1/DoA20IojS72IdaFSN8CX9Q] already exists",
					"index_uuid" : "DoA20IojS72IdaFSN8CX9Q",
					"index" : "idx1"
				}
			],
			"type" : "index_already_exists_exception",
			"reason" : "index [idx1/DoA20IojS72IdaFSN8CX9Q] already exists",
			"index_uuid" : "DoA20IojS72IdaFSN8CX9Q",
			"index" : "idx1"
		},
		"status" : 400
	}`
	IndexDeletedRs = `
	{
		"acknowledged" : true
	}`
	IndexNotFoundRs = `
	{
		"error" : {
			"root_cause" : [
			{
				"type" : "index_not_found_exception",
				"reason" : "no such index",
				"resource.type" : "index_or_alias",
				"resource.id" : "idx1",
				"index_uuid" : "_na_",
				"index" : "idx1"
			}
			],
			"type" : "index_not_found_exception",
			"reason" : "no such index",
			"resource.type" : "index_or_alias",
			"resource.id" : "idx1",
			"index_uuid" : "_na_",
			"index" : "idx1"
		},
		"status" : 404
	}`
	LaunchWoTestItems = `
	{
		"launchId": "1234567890",
		"launchName": "Launch without test items",
		"testItems": []
	}`
	LaunchWTestItemsWoLogs = `
	{
		"launchId": "1234567891",
		"launchName": "Launch with test items without logs",
		"testItems": [
			{
				"testItemId": "0001",
				"issueType": "TI001",
				"logs": []
			}
		]
	}`
	LaunchWTestItemsWLogs = `
	{
		"launchId": "1234567892",
		"launchName": "Launch with test items with logs",
		"testItems": [
			{
				"testItemId": "0002",
				"issueType": "TI001",
				"logs": [
					{
						"logId": "0001",
						"logLevel": 40000,
						"message": "Message 1"
					},
					{
						"logId": "0002",
						"logLevel": 40000,
						"message": "Message 2"
					}
				]
			}
		]
	}`
	IndexLogsRq = `{"index":{"_id":"0001","_index":"idx2","_type":"log"}}
{"issue_type":"TI001","launch_name":"Launch with test items with logs","log_level":40000,"message":"Message ","test_item":"0002"}
{"index":{"_id":"0002","_index":"idx2","_type":"log"}}
{"issue_type":"TI001","launch_name":"Launch with test items with logs","log_level":40000,"message":"Message ","test_item":"0002"}
`
	IndexLogsRs = `
	{
		"took" : 63,
		"errors" : false,
		"items" : [
			{
				"index" : {
					"_index" : "idx2",
					"_type" : "log",
					"_id" : "0001",
					"_version" : 1,
					"result" : "created",
					"_shards" : {
						"total" : 2,
						"successful" : 1,
						"failed" : 0
					},
					"created" : true,
					"status" : 201
				}
			}
		]
	}`
	SearchRq = `{"query":{"bool":{"must":[{"term":{"log_level":40000}},{"exists":{"field":"issue_type"}},{"more_like_this":{"fields":["message"],"like":"Message ","minimum_should_match":"90%"}}],"must_not":{"wildcard":{"issue_type":"TI*"}},"should":{"term":{"launch_name":{"boost":2,"value":"Launch with test items with logs"}}}}},"size":10}
`
	OneHitSearchRs = `
	{
		"took" : 13,
		"timed_out" : false,
		"_shards" : {
			"total" : 1,
			"successful" : 1,
			"failed" : 0
		},
		"hits" : {
			"total" : 1,
			"max_score" : 10,
			"hits" : [
				{
					"_index" : "idx2",
					"_type" : "log",
					"_id" : "0001",
					"_score" : 10,
					"_source" : {
						"issue_type" : "AB001",
						"launch_name" : "Launch 1",
						"log_level" : 40000,
						"message" : "Message AB",
						"test_item" : "0001"
					}
				}
			]
		}
	}`
	TwoHitsSearchRs = `
	{
		"took" : 13,
		"timed_out" : false,
		"_shards" : {
			"total" : 1,
			"successful" : 1,
			"failed" : 0
		},
		"hits" : {
			"total" : 2,
			"max_score" : 15,
			"hits" : [
				{
					"_index" : "idx3",
					"_type" : "log",
					"_id" : "0001",
					"_score" : 15,
					"_source" : {
						"issue_type" : "AB001",
						"launch_name" : "Launch 1",
						"log_level" : 40000,
						"message" : "Message AB",
						"test_item" : "0001"
					}
				},
				{
					"_index" : "idx3",
					"_type" : "log",
					"_id" : "0002",
					"_score" : 10,
					"_source" : {
						"issue_type" : "PB001",
						"launch_name" : "Launch 1",
						"log_level" : 40000,
						"message" : "Message PB",
						"test_item" : "0001"
					}
				}
			]
		}
	}`
	ThreeHitsSearchRs = `
	{
		"took" : 13,
		"timed_out" : false,
		"_shards" : {
			"total" : 1,
			"successful" : 1,
			"failed" : 0
		},
		"hits" : {
			"total" : 2,
			"max_score" : 20,
			"hits" : [
				{
					"_index" : "idx4",
					"_type" : "log",
					"_id" : "0001",
					"_score" : 15,
					"_source" : {
						"issue_type" : "AB001",
						"launch_name" : "Launch 1",
						"log_level" : 40000,
						"message" : "Message AB",
						"test_item" : "0001"
					}
				},
				{
					"_index" : "idx4",
					"_type" : "log",
					"_id" : "0002",
					"_score" : 10,
					"_source" : {
						"issue_type" : "PB001",
						"launch_name" : "Launch 1",
						"log_level" : 40000,
						"message" : "Message PB",
						"test_item" : "0001"
					}
				},
				{
					"_index" : "idx4",
					"_type" : "log",
					"_id" : "0003",
					"_score" : 10,
					"_source" : {
						"issue_type" : "PB001",
						"launch_name" : "Launch 1",
						"log_level" : 40000,
						"message" : "Message PB",
						"test_item" : "0001"
					}
				}
			]
		}
	}`
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
				"response":   TwoIndicesRs,
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
				"response":   IndexCreatedRs,
			},
			expectErr: false,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx1",
				"statusCode": http.StatusBadRequest,
				"response":   IndexAlreadyExistsRs,
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
				"response":   IndexDeletedRs,
			},
			expectedStatus: 0,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx1",
				"statusCode": http.StatusNotFound,
				"response":   IndexNotFoundRs,
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
			indexRequest:     LaunchWoTestItems,
			expectServerCall: false,
		},
		{
			params: map[string]interface{}{
				"indexName": "idx1",
			},
			indexRequest:     LaunchWTestItemsWoLogs,
			expectServerCall: false,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx2",
				"request":    IndexLogsRq,
				"response":   IndexLogsRs,
				"statusCode": http.StatusOK,
			},
			indexRequest:     LaunchWTestItemsWLogs,
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
			analyzeRequest: LaunchWoTestItems,
		},
		{
			params: map[string]interface{}{
				"indexName": "idx1",
			},
			analyzeRequest: LaunchWTestItemsWoLogs,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx2",
				"request":    SearchRq,
				"response":   OneHitSearchRs,
				"statusCode": http.StatusOK,
			},
			analyzeRequest:    LaunchWTestItemsWLogs,
			expectedIssueType: "AB001",
			serverCallCount:   2,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx3",
				"request":    SearchRq,
				"response":   TwoHitsSearchRs,
				"statusCode": http.StatusOK,
			},
			analyzeRequest:    LaunchWTestItemsWLogs,
			expectedIssueType: "AB001",
			serverCallCount:   2,
		},
		{
			params: map[string]interface{}{
				"indexName":  "idx4",
				"request":    SearchRq,
				"response":   ThreeHitsSearchRs,
				"statusCode": http.StatusOK,
			},
			analyzeRequest:    LaunchWTestItemsWLogs,
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
