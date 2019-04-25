package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gopkg.in/go-playground/validator.v9"
)

var validate = validator.New()

func handleAmqpRequest(ch *amqp.Channel, d amqp.Delivery, handler requestHandler) error {
	var launches []Launch
	err := json.Unmarshal(d.Body, &launches)
	if err != nil {
		return errors.WithStack(err)
	}

	for i, l := range launches {
		if err = validate.Struct(l); nil != err {
			return errors.Wrapf(err, "Validation failed on Launch[%d]", i)
		}
	}

	rs, err := handler(launches)
	if err != nil {
		return errors.WithStack(err)
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

func handleDeleteRequest(d amqp.Delivery, h *RequestHandler) error {
	var id int64
	err := json.Unmarshal(d.Body, &id)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = h.DeleteIndex(id)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func handleCleanRequest(d amqp.Delivery, h *RequestHandler) error {
	var ci CleanIndex
	err := json.Unmarshal(d.Body, &ci)
	if err != nil {
		return errors.WithStack(err)
	}

	if err = validate.Struct(ci); nil != err {
		return errors.Wrapf(err, "Validation failed on CleanIndex")
	}

	_, err = h.CleanIndex(&ci)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
