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
