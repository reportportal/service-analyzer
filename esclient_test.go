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
	"fmt"
	"github.com/reportportal/commons-go/conf"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reportportal/commons-go/server"
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
	LaunchWTestItemsWLogsDifferentLogLevel  = "launch_w_test_items_w_logs_different_log_level.json"
	IndexLogsRqDifferentLogLevel            = "index_logs_rq_different_log_level.json"
	IndexLogsRsDifferentLogLevel            = "index_logs_rs_different_log_level.json"
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
		c := NewClient([]string{ts.URL}, defaultSearchConfig())

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
		c := NewClient([]string{ts.URL}, defaultSearchConfig())

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
		c := NewClient([]string{ts.URL}, defaultSearchConfig())

		exists, err := c.IndexExists(test.index)

		assert.Equal(t, len(test.calls), i)

		assert.NoError(t, err)
		assert.Equal(t, test.exists, exists)
	}
}

func TestDeleteIndex(t *testing.T) {
	tests := []struct {
		calls          []ServerCall
		index          int64
		expectedStatus int
	}{
		{
			calls: []ServerCall{
				{
					method: "DELETE",
					uri:    "/1",
					rs:     getFixture(IndexDeletedRs),
					status: http.StatusOK,
				},
			},
			index:          1,
			expectedStatus: 0,
		},
		{
			calls: []ServerCall{
				{
					method: "DELETE",
					uri:    "/2",
					rs:     getFixture(IndexNotFoundRs),
					status: http.StatusNotFound,
				},
			},
			index:          2,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient([]string{ts.URL}, defaultSearchConfig())

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
					uri:    "/1",
					status: http.StatusOK,
				},
			},
			indexRq: getFixture(LaunchWoTestItems),
		},
		{
			calls: []ServerCall{
				{
					method: "HEAD",
					uri:    "/1",
					status: http.StatusOK,
				},
			},
			indexRq: getFixture(LaunchWTestItemsWoLogs),
		},
		{
			calls: []ServerCall{
				{
					method: "HEAD",
					uri:    "/2",
					status: http.StatusNotFound,
				},
				{
					method: "PUT",
					uri:    "/2",
					rs:     getFixture(IndexCreatedRs),
					status: http.StatusOK,
				},
				{
					method: "PUT",
					uri:    "/_bulk?refresh",
					rq:     getFixture(IndexLogsRq),
					rs:     getFixture(IndexLogsRs),
					status: http.StatusOK,
				},
			},
			indexRq: getFixture(LaunchWTestItemsWLogs),
		},
		{
			calls: []ServerCall{
				{
					method: "HEAD",
					uri:    "/2",
					status: http.StatusNotFound,
				},
				{
					method: "PUT",
					uri:    "/2",
					rs:     getFixture(IndexCreatedRs),
					status: http.StatusOK,
				},
				{
					method: "PUT",
					uri:    "/_bulk?refresh",
					rq:     getFixture(IndexLogsRqDifferentLogLevel),
					rs:     getFixture(IndexLogsRsDifferentLogLevel),
					status: http.StatusOK,
				},
			},
			indexRq: getFixture(LaunchWTestItemsWLogsDifferentLogLevel),
		}
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient([]string{ts.URL}, defaultSearchConfig())

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
					uri:    "/2/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(NoHitsSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/2/_search",
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
					uri:    "/2/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(NoHitsSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/2/_search",
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
					uri:    "/2/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(OneHitSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/2/_search",
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
					uri:    "/2/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(TwoHitsSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/2/_search",
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
					uri:    "/2/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(NoHitsSearchRs),
					status: http.StatusOK,
				},
				{
					method: "GET",
					uri:    "/2/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(ThreeHitsSearchRs),
					status: http.StatusOK,
				},
			},
			analyzeRq:     getFixture(LaunchWTestItemsWLogs),
			expectedIssue: "PB001",
		},
		{
			calls: []ServerCall{
				{
					method: "GET",
					uri:    "/2/_search",
					rq:     getFixture(SearchRq),
					rs:     getFixture(TwoHitsSearchRs),
					status: http.StatusOK,
				}
			},
			analyzeRq:     getFixture(LaunchWTestItemsWLogsDifferentLogLevel),
			expectedIssue: "AB001",
		},
	}

	for _, test := range tests {
		i := 0
		ts := startServer(t, test.calls, &i)
		defer ts.Close()
		c := NewClient([]string{ts.URL}, defaultSearchConfig())

		launches := []Launch{}
		err := json.Unmarshal([]byte(test.analyzeRq), &launches)
		assert.NoError(t, err)

		results, err := c.AnalyzeLogs(launches)
		assert.NoError(t, err)

		if test.expectedIssue != "" {
			assert.Equal(t, test.expectedIssue, results[0].IssueType)
		}
	}
}

func TestClearIndex(t *testing.T) {
	assert.Error(t, server.Validate(&CleanIndex{}), "Incorrect struct validation")
	assert.NoError(t, server.Validate(&CleanIndex{
		Project: 1,
	}), "Incorrect struct validation")

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
			defer func() {
				if cErr := r.Body.Close(); cErr != nil {
					log.Error(cErr)
				}
			}()
			rq, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, expectedCall.rq, string(rq))
		}
		w.WriteHeader(expectedCall.status)
		if expectedCall.rs != "" {
			if _, wErr := w.Write([]byte(expectedCall.rs)); wErr != nil {
				log.Error(wErr)
			}
		}
		*i = *i + 1
	}))

	return ts
}

func Test_findNth(t *testing.T) {
	type args struct {
		str string
		f   string
		n   int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "one occurrence",
			args: args{str: "search", f: "se", n: 1},
			want: 0,
		},
		{
			name: "multiple occurrence",
			args: args{str: "search search", f: "se", n: 2},
			want: 7,
		},
		{
			name: "multiple occurrence - third",
			args: args{str: "ok ok ok ok ok", f: "k", n: 3},
			want: 7,
		},
		{
			name: "not found",
			args: args{str: "search search", f: "se", n: 3},
			want: -1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := findNth(tt.args.str, tt.args.f, tt.args.n); got != tt.want {
				t.Errorf("findNth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_firstLines(t *testing.T) {
	type args struct {
		str string
		n   int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "pos",
			args: args{str: `hello
world`, n: 1},
			want: "hello",
		},
		{
			name: "pos",
			args: args{str: `hello
world  
hello`, n: 2},
			want: "hello\nworld  ",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := firstLines(tt.args.str, tt.args.n); got != tt.want {
				t.Errorf("firstLines() = %v, want %v", got, tt.want)
			}
		})
	}
}

func defaultSearchConfig() *SearchConfig {
	sc := &SearchConfig{}
	if err := conf.LoadConfig(sc); err != nil {
		log.Error(err)
		return &SearchConfig{}
	}
	return sc
}
