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
