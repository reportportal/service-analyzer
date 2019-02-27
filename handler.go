package main

import (
	"github.com/pkg/errors"
	"gopkg.in/reportportal/commons-go.v5/server"
)

type requestHandler func([]Launch) (interface{}, error)

//RequestHandler handles ES-related requests
type RequestHandler struct {
	c ESClient
}

//NewRequestHandler creates new instance of handler
func NewRequestHandler(c ESClient) *RequestHandler {
	return &RequestHandler{c: c}
}

//IndexLaunches indexes launches
func (h *RequestHandler) IndexLaunches(launches []Launch) (interface{}, error) {
	return h.c.IndexLogs(launches)
}

//AnalyzeLogs analyzes the logs
func (h *RequestHandler) AnalyzeLogs(launches []Launch) (interface{}, error) {
	return h.c.AnalyzeLogs(launches)
}

//DeleteIndex deletes index
func (h *RequestHandler) DeleteIndex(id int64) func(launches []Launch) (interface{}, error) {
	return func(launches []Launch) (interface{}, error) {
		_, err := h.c.DeleteIndex(id)
		return nil, err
	}
}

//CleanIndex cleans index
func (h *RequestHandler) CleanIndex(ci *CleanIndex) (*Response, error) {
	err := server.Validate(ci)
	if nil != err {
		return nil, errors.WithStack(err)
	}
	return h.c.DeleteLogs(ci)
}
