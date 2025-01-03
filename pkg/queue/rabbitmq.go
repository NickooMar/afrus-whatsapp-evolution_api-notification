package queue

import (
	config "afrus-whatsapp-evolution_api-notification/configs"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Channel    *amqp.Channel
	Connection *amqp.Connection
	Configs    *config.Config
	OutChannel chan<- *amqp.Delivery
}

func NewRabbitMQ(configs *config.Config, outChannel chan<- *amqp.Delivery) *RabbitMQ {
	return &RabbitMQ{
		Channel:    nil,
		Connection: nil,
		Configs:    configs,
		OutChannel: outChannel,
	}
}

func (rmq *RabbitMQ) Dial() error {
	var connectionString string
	if rmq.Configs.Environment == "development" || rmq.Configs.Environment == "staging" {
		connectionString = fmt.Sprintf("amqp://%s:%s@%s:%s", rmq.Configs.RabbitMQUser, rmq.Configs.RabbitMQPassword, rmq.Configs.RabbitMQUrl, rmq.Configs.RabbitMQPort)
	} else {
		connectionString = fmt.Sprintf("amqps://%s:%s@%s:%s", rmq.Configs.RabbitMQUser, rmq.Configs.RabbitMQPassword, rmq.Configs.RabbitMQUrl, rmq.Configs.RabbitMQPort)
	}

	connection, err := amqp.Dial(connectionString)
	if err != nil {
		return fmt.Errorf("failed to dial RabbitMQ: %w", err)
	}
	rmq.Connection = connection
	channel, err := rmq.Connection.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	rmq.Channel = channel
	log.Printf("[RABBITMQ] - Connection established \n")
	return nil
}

func (rmq *RabbitMQ) Consume(queue string) error {
	msgs, err := rmq.Channel.Consume(
		queue,
		"whatsapp-evolution_api-consumer",
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	go func() {
		for msg := range msgs {
			rmq.OutChannel <- &msg
		}
		close(rmq.OutChannel)
	}()

	return nil
}

func (rmq *RabbitMQ) Close() error {
	if rmq.Channel != nil {
		if err := rmq.Channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
		rmq.Channel = nil
	}
	if rmq.Connection != nil {
		if err := rmq.Connection.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
		rmq.Connection = nil
	}
	return nil
}
