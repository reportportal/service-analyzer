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
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/reportportal/commons-go.v5/conf"
	"gopkg.in/reportportal/commons-go.v5/server"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_DeleteIndex(t *testing.T) {
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	mux := chi.NewMux()

	h := &RequestHandler{c: NewClient([]string{}, defaultSearchConfig())}
	mux.Handle("/", server.Handler{H: func(w http.ResponseWriter, rq *http.Request) error {
		if id := chi.URLParam(rq, "index_id"); "" != id {
			return handleHTTPRequest(w, rq, h.DeleteIndex(id))
		}
		return server.ToStatusError(http.StatusBadRequest, errors.New("Index ID is incorrect"))
	}})

	req, _ := http.NewRequest(http.MethodDelete, "/", nil)
	mux.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusBadRequest)
	assert.Contains(t, rr.Body.String(), "Index ID is incorrect")
}

func TestClient_CleanIndex(t *testing.T) {
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	mux := chi.NewMux()

	h := &RequestHandler{c: NewClient([]string{}, defaultSearchConfig())}

	mux.Handle("/_index/{index_id}/delete", server.Handler{H: cleanIndexHttpHandler(h)})

	req, _ := http.NewRequest(http.MethodPut, "/_index/xxx/delete", bytes.NewBufferString(`{"ids" : []}`))
	mux.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusBadRequest)
	assert.Contains(t, rr.Body.String(), "Struct validation has failed")
}

func defaultSearchConfig() *SearchConfig {
	sc := &SearchConfig{}
	conf.LoadConfig(sc)
	return sc
}
