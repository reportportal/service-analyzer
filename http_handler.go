package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gopkg.in/reportportal/commons-go.v5/server"
	"net/http"
)

func cleanIndexHttpHandler(h *RequestHandler) func(w http.ResponseWriter, rq *http.Request) error {
	return func(w http.ResponseWriter, rq *http.Request) error {
		var ci CleanIndex
		err := server.ReadJSON(rq, &ci)
		if nil != err {
			return server.ToStatusError(http.StatusBadRequest, errors.Wrap(err, "Cannot read request body"))
		}

		rs, err := h.CleanIndex(&ci)
		if nil != err {
			return server.ToStatusError(http.StatusBadRequest, errors.WithStack(err))
		}
		return server.WriteJSON(http.StatusOK, rs, w)
	}
}

func handleHTTPRequest(w http.ResponseWriter, rq *http.Request, handler requestHandler) error {
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

func handleAmqpRequest(ch *amqp.Channel, d amqp.Delivery, handler requestHandler) error {
	var launches []Launch
	err := json.Unmarshal(d.Body, &launches)
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

	rsBody, err := json.Marshal(rs)
	if err != nil {
		return err
	}

	return ch.Publish(
		"",        // exchange
		d.ReplyTo, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: d.CorrelationId,
			Body:          rsBody,
		})

}
