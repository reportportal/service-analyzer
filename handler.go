package main

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
func (h *RequestHandler) DeleteIndex(id int64) (*Response, error) {
	return h.c.DeleteIndex(id)
}

//CleanIndex cleans index
func (h *RequestHandler) CleanIndex(ci *CleanIndex) (*Response, error) {
	return h.c.DeleteLogs(ci)
}
