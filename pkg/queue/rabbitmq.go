package queue

import (
	config "afrus-whatsapp-evolution_api-notification/configs"
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueConfig struct {
	Name       string
	BufferSize int
	Consumer   string
}

type RabbitMQ struct {
	Channel    *amqp.Channel
	Connection *amqp.Connection
	Configs    *config.Config
	Queues     map[string]chan *amqp.Delivery
}

func NewRabbitMQ(configs *config.Config) *RabbitMQ {
	return &RabbitMQ{
		Channel:    nil,
		Connection: nil,
		Configs:    configs,
		Queues:     make(map[string]chan *amqp.Delivery),
	}
}

func (r *RabbitMQ) AddQueue(config QueueConfig) (chan *amqp.Delivery, error) {
	msgs := make(chan *amqp.Delivery, config.BufferSize)
	r.Queues[config.Name] = msgs

	if r.Channel != nil {
		if err := r.startConsuming(config, msgs); err != nil {
			return nil, err
		}
	}

	return msgs, nil
}

func (r *RabbitMQ) Connect() error {
	connectionString := fmt.Sprintf("%s://%s:%s@%s:%s",
		getRabbitMQProtocol(r.Configs.Environment),
		r.Configs.RabbitMQUser,
		r.Configs.RabbitMQPassword,
		r.Configs.RabbitMQUrl,
		r.Configs.RabbitMQPort,
	)

	var conn *amqp.Connection
	var err error

	for retries := 0; retries < 5; retries++ {
		conn, err = amqp.Dial(connectionString)
		if err == nil {
			break
		}
		log.Printf("[RABBITMQ] - Failed to connect. Retrying in 2 seconds... (%d/5)\n", retries+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
	}

	r.Connection = conn

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	r.Channel = ch

	log.Printf("[RABBITMQ] - Connected to RabbitMQ")

	// Start consuming for all registered queues
	for name, msgs := range r.Queues {
		config := QueueConfig{
			Name:       name,
			BufferSize: cap(msgs),
			Consumer:   fmt.Sprintf("%s-consumer", name),
		}
		if err := r.startConsuming(config, msgs); err != nil {
			return err
		}
	}

	return nil
}

func (r *RabbitMQ) startConsuming(config QueueConfig, msgs chan<- *amqp.Delivery) error {
	deliveries, err := r.Channel.Consume(
		config.Name,
		config.Consumer,
		false, // Auto-Ack
		false, // Exclusive
		false, // No Local
		false, // No Wait
		nil,   // Args
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range deliveries {
			msgs <- &msg
		}
	}()

	return nil
}

func getRabbitMQProtocol(environment string) string {
	if environment == "development" || environment == "staging" {
		return "amqp"
	}
	return "amqps"
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	err := r.Channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("[RABBITMQ] - failed to publish message: %w", err)
	}
	return nil
}

func (r *RabbitMQ) Schedule(exchange, routingKey string, body []byte, delay int) error {
	err := r.Channel.Publish(
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Headers: amqp.Table{
				"x-delay": delay,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("[RABBITMQ] - failed to schedule message: %w", err)
	}
	return nil
}

func (r *RabbitMQ) Close() error {
	if r.Channel != nil {
		if err := r.Channel.Close(); err != nil {
			return fmt.Errorf("failed to close RabbitMQ channel: %w", err)
		}
		r.Channel = nil
	}
	if r.Connection != nil {
		if err := r.Connection.Close(); err != nil {
			return fmt.Errorf("failed to close RabbitMQ connection: %w", err)
		}
		r.Connection = nil
	}
	for _, msgs := range r.Queues {
		close(msgs)
	}
	return nil
}
