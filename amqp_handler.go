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
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gopkg.in/go-playground/validator.v9"
)

var validate = validator.New()

func handleAmqpRequest(ch *amqp.Channel, d amqp.Delivery, handler requestHandler) (err error) {
	var launches []Launch
	err = json.Unmarshal(d.Body, &launches)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	defer func() {
		if err == nil {
			d.Ack(false)
		}
	}()

	for i, l := range launches {
		if err = validate.Struct(l); nil != err {
			err = errors.Wrapf(err, "Validation failed on Launch[%d]", i)
			return
		}
	}

	rs, err := handler(launches)
	if err != nil {
		return errors.WithStack(err)
	}

	rsBody, err := json.Marshal(rs)
	if err != nil {
		return
	}

	err = ch.Publish(
		"",        // exchange
		d.ReplyTo, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: d.CorrelationId,
			Body:          rsBody,
		})
	return
}

func handleSearchRequest(ch *amqp.Channel, d amqp.Delivery, h searchRequestHandler) error {
	var request SearchLogs
	err := json.Unmarshal(d.Body, &request)
	if err != nil {
		return errors.WithStack(err)
	}

	response, err := h(request)
	if err != nil {
		return errors.WithStack(err)
	}

	rsBody, err := json.Marshal(response)
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

func handleDeleteRequest(d amqp.Delivery, h *RequestHandler) (err error) {
	var id int64
	err = json.Unmarshal(d.Body, &id)

	if err != nil {
		err = errors.WithStack(err)
		return err
	}

	defer func() {
		if err == nil {
			d.Ack(false)
		}
	}()

	_, err = h.DeleteIndex(id)
	if err != nil {
		err = errors.WithStack(err)
		return err
	}
	return
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
