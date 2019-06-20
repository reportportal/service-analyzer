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
	"context"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

//AmqpClient is useful interface to deal with amqp connection
type AmqpClient struct {
	conn *amqp.Connection
}

//NewAmqpClient is a factory method for AmqpClient
func NewAmqpClient(conn *amqp.Connection) *AmqpClient {
	return &AmqpClient{conn: conn}
}

//DoOnChannel opens amqp channel and execute func
func (a *AmqpClient) DoOnChannel(chCallback func(channel *amqp.Channel) error) error {
	ch, err := a.conn.Channel()
	if err != nil {
		return errors.Wrap(err, "Failed to open a channel")
	}
	defer func() {
		if err := ch.Close(); err != nil {
			log.Errorf("Unable to close opened ampq channel: %v", err)
		}
	}()

	return chCallback(ch)
}

//Receive connects to queue and processes each message with given callback
//Message callback does not result to exit from this method
func (a *AmqpClient) Receive(ctx context.Context, queue string, autoAck, exclusive, noLocal, noWait bool, msgCallback func(amqp.Delivery) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := a.consumeQueue(ctx, queue, autoAck, exclusive, noLocal, noWait, msgCallback); nil != err {
				return err
			}
		}
	}

}

func (a *AmqpClient) consumeQueue(ctx context.Context, queue string, autoAck, exclusive, noLocal, noWait bool, msgCallback func(amqp.Delivery) error) error {
	return a.DoOnChannel(func(ch *amqp.Channel) error {
		msgs, cErr := ch.Consume(
			queue,     // queue
			"",        // consumer
			autoAck,   // auto-ack
			exclusive, // exclusive
			noWait,    // no-local
			noWait,    // no-wait
			nil,       // args
		)
		if cErr != nil {
			return errors.Wrap(cErr, "Failed to register a consumer")
		}

		if err := a.processMessages(ctx, msgs, msgCallback); nil != err {
			return err
		}
		return nil
	})

}

func (a *AmqpClient) processMessages(ctx context.Context, msgs <-chan amqp.Delivery, msgCallback func(amqp.Delivery) error) error {
	for {
		//grab incoming messages
		select {
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			go func() {
				//process incoming message
				if err := msgCallback(msg); nil != err {
					log.Error(err)
				}
			}()
			//if cancel signal has been received
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
