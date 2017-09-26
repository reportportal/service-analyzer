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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

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

type ServerCall struct {
	method string
	uri    string
	rs     string
	rq     string
	status int
}

func TestListIndices(t *testing.T) {
	tests := []struct {
		calls         []ServerCall
		expectedCount int
		expectErr     bool
	}{
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/_cat/indices?format=json",
					rs:     "[]",
					status: http.StatusOK,
				},
			},
			expectedCount: 0,
			expectErr:     false,
		},
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/_cat/indices?format=json",
					rs:     getFixture(TwoIndicesRs),
					status: http.StatusOK,
				},
			},
			expectedCount: 2,
			expectErr:     false,
		},
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/_cat/indices?format=json",
					status: http.StatusInternalServerError,
				},
			},
			expectErr: true,
		},
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient(ts.URL)

		indices, err := c.ListIndices()

		assert.Equal(t, len(test.calls), i)

		if test.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expectedCount, len(indices))
			for j, idx := range indices {
				assert.Equal(t, fmt.Sprintf("idx%d", j), idx.Index)
			}
		}
	}
}

func TestCreateIndex(t *testing.T) {
	tests := []struct {
		calls     []ServerCall
		index     string
		expectErr bool
	}{
		{
			calls: []ServerCall{
				{
					method: "PUT",
					uri:    "/idx0",
					rs:     getFixture(IndexCreatedRs),
					status: http.StatusOK,
				},
			},
			index:     "idx0",
			expectErr: false,
		},
		{
			calls: []ServerCall{
				{
					method: "PUT",
					uri:    "/idx1",
					rs:     getFixture(IndexAlreadyExistsRs),
					status: http.StatusBadRequest,
				},
			},
			index:     "idx1",
			expectErr: true,
		},
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient(ts.URL)

		rs, err := c.CreateIndex(test.index)

		assert.Equal(t, len(test.calls), i)

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
		calls  []ServerCall
		index  string
		exists bool
	}{
		{
			calls: []ServerCall{
				{
					method: "HEAD",
					uri:    "/idx0",
					status: http.StatusOK,
				},
			},
			index:  "idx0",
			exists: true,
		},
		{
			calls: []ServerCall{
				{
					method: "HEAD",
					uri:    "/idx1",
					status: http.StatusNotFound,
				},
			},
			index:  "idx1",
			exists: false,
		},
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient(ts.URL)

		exists, err := c.IndexExists(test.index)

		assert.Equal(t, len(test.calls), i)

		assert.NoError(t, err)
		assert.Equal(t, test.exists, exists)
	}
}

func TestSuccess(t *testing.T) {
	assert.Equal(t, false, success(400))
	assert.Equal(t, true, success(200))
}

func TestDeleteIndex(t *testing.T) {
	tests := []struct {
		calls          []ServerCall
		index          string
		expectedStatus int
	}{
		{
			calls: []ServerCall{
				{
					method: "DELETE",
					uri:    "/idx0",
					rs:     getFixture(IndexDeletedRs),
					status: http.StatusOK,
				},
			},
			index:          "idx0",
			expectedStatus: 0,
		},
		{
			calls: []ServerCall{
				{
					method: "DELETE",
					uri:    "/idx1",
					rs:     getFixture(IndexNotFoundRs),
					status: http.StatusNotFound,
				},
			},
			index:          "idx1",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient(ts.URL)

		rs, err := c.DeleteIndex(test.index)

		assert.Equal(t, len(test.calls), i)

		assert.NoError(t, err)
		assert.Equal(t, test.expectedStatus, rs.Status)
	}
}

func TestIndexLogs(t *testing.T) {
	tests := []struct {
		calls   []ServerCall
		indexRq string
	}{
		{
			calls: []ServerCall{
				{
					method: "HEAD",
					uri:    "/idx0",
					status: http.StatusOK,
				},
			},
			indexRq: getFixture(LaunchWoTestItems),
		},
		{
			calls: []ServerCall{
				{
					method: "HEAD",
					uri:    "/idx1",
					status: http.StatusOK,
				},
			},
			indexRq: getFixture(LaunchWTestItemsWoLogs),
		},
		{
			calls: []ServerCall{
				{
					method: "HEAD",
					uri:    "/idx2",
					status: http.StatusNotFound,
				},
				{
					method: "PUT",
					uri:    "/idx2",
					rs:     getFixture(IndexCreatedRs),
					status: http.StatusOK,
				},
				{
					method: "PUT",
					uri:    "/_bulk",
					rq:     getFixture(IndexLogsRq),
					rs:     getFixture(IndexLogsRs),
					status: http.StatusOK,
				},
			},
			indexRq: getFixture(LaunchWTestItemsWLogs),
		},
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient(ts.URL)

		launches := []Launch{}
		err := json.Unmarshal([]byte(test.indexRq), &launches)
		assert.NoError(t, err)

		_, err = c.IndexLogs(launches)

		assert.Equal(t, len(test.calls), i)
		assert.NoError(t, err)
	}
}

func TestAnalyzeLogs(t *testing.T) {
	tests := []struct {
		calls         []ServerCall
		analyzeRq     string
		expectedIssue string
	}{
		{
			calls:     []ServerCall{},
			analyzeRq: getFixture(LaunchWoTestItems),
		},
		{
			calls:     []ServerCall{},
			analyzeRq: getFixture(LaunchWTestItemsWoLogs),
		},
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(NoHitsSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(NoHitsSearchRs),
					status: http.StatusOK,
				},
			},
			analyzeRq: getFixture(LaunchWTestItemsWLogs),
		},
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(NoHitsSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(OneHitSearchRs),
					status: http.StatusOK,
				},
			},
			analyzeRq:     getFixture(LaunchWTestItemsWLogs),
			expectedIssue: "AB001",
		},
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(OneHitSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(TwoHitsSearchRs),
					status: http.StatusOK,
				},
			},
			analyzeRq:     getFixture(LaunchWTestItemsWLogs),
			expectedIssue: "AB001",
		},
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(TwoHitsSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(ThreeHitsSearchRs),
					status: http.StatusOK,
				},
			},
			analyzeRq:     getFixture(LaunchWTestItemsWLogs),
			expectedIssue: "AB001",
		},
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(NoHitsSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/idx2/log/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(ThreeHitsSearchRs),
					status: http.StatusOK,
				},
			},
			analyzeRq:     getFixture(LaunchWTestItemsWLogs),
			expectedIssue: "PB001",
		},
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient(ts.URL)

		launches := []Launch{}
		err := json.Unmarshal([]byte(test.analyzeRq), &launches)
		assert.NoError(t, err)

		launches, err = c.AnalyzeLogs(launches)
		assert.NoError(t, err)

		if test.expectedIssue != "" {
			assert.Equal(t, test.expectedIssue, launches[0].TestItems[0].IssueType)
		}
	}
}

func getFixture(filename string) string {
	f, _ := ioutil.ReadFile("fixtures/" + filename)
	return string(f)
}

func startServer(t *testing.T, expectedCalls []ServerCall, i *int) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedCall := expectedCalls[*i]
		assert.Equal(t, expectedCall.method, r.Method)
		assert.Equal(t, expectedCall.uri, r.URL.RequestURI())
		if expectedCall.rq != "" {
			defer r.Body.Close()
			rq, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, expectedCall.rq, string(rq))
		}
		w.WriteHeader(expectedCall.status)
		if expectedCall.rs != "" {
			w.Write([]byte(expectedCall.rs))
		}
		*i = *i + 1
	}))

	return ts
}
