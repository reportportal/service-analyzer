package main

import (
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"gopkg.in/reportportal/commons-go.v5/server"
	"net/http"
)

func deleteIndexHandler(c ESClient) func(w http.ResponseWriter, rq *http.Request) error {
	return func(w http.ResponseWriter, rq *http.Request) error {
		if id := chi.URLParam(rq, "index_id"); "" != id {
			_, err := c.DeleteIndex(id)
			return err
		}
		return server.ToStatusError(http.StatusBadRequest, errors.New("Index ID is incorrect"))
	}
}

func cleanIndexHandler(c ESClient) func(w http.ResponseWriter, rq *http.Request) error {
	return func(w http.ResponseWriter, rq *http.Request) error {
		var ci CleanIndex
		err := server.ReadJSON(rq, &ci)
		if nil != err {
			return server.ToStatusError(http.StatusBadRequest, errors.Wrap(err, "Cannot read request body"))
		}
		err = server.Validate(ci)
		if nil != err {
			return server.ToStatusError(http.StatusBadRequest, errors.WithStack(err))
		}

		rs, err := c.DeleteLogs(&ci)
		if nil != err {
			return server.ToStatusError(http.StatusBadRequest, errors.WithStack(err))
		}
		return server.WriteJSON(http.StatusOK, rs, w)
	}
}

type requestHandler func([]Launch) (interface{}, error)

func handleRequest(w http.ResponseWriter, rq *http.Request, handler requestHandler) error {
	var launches []Launch
	err := server.ReadJSON(rq, &launches)
	if err != nil {
		return server.ToStatusError(http.StatusBadRequest, errors.WithStack(err))
	}

	for i, l := range launches {
		if valErr := server.Validate(l); nil != valErr {
			return server.ToStatusError(http.StatusBadRequest, errors.Wrapf(valErr, "Validation failed on Launch[%d]", i))
		}
	}

	rs, err := handler(launches)
	if err != nil {
		return server.ToStatusError(http.StatusInternalServerError, errors.WithStack(err))
	}
	return server.WriteJSON(http.StatusOK, rs, w)
}
