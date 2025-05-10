package pubsub

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueType int

const (
	DurableQueue   QueueType = iota // 0
	TransientQueue                  // 1
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	})
	if err != nil {
		return err
	}
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType QueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	newChan, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	isDurable := true
	autoDelete := false
	exclusive := false
	if simpleQueueType == 1 {
		isDurable = false
		autoDelete = true
		exclusive = true
	}

	newQueue, err := newChan.QueueDeclare(queueName, isDurable, autoDelete, exclusive, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = newChan.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	return newChan, newQueue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType QueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) (*amqp.Channel, amqp.Queue, error) {
	AMQPChann, AMPQQueue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	delChan, err := AMQPChann.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	go func() {
		for delivery := range delChan {
			var message T
			err := json.Unmarshal(delivery.Body, &message)
			if err != nil {
				log.Println("Cant Unmarshal in goroutine: ", err)
				continue
			}
			handler(message)
			delivery.Ack(false)
		}
	}()
	return AMQPChann, AMPQQueue, nil
}
