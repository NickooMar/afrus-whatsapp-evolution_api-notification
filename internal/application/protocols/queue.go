package protocols

import (
	"afrus-whatsapp-evolution_api-notification/pkg/queue"
	"context"
)

type Queue interface {
	Close() error
	Connect() error
	getRabbitMQProtocol(environment string) string
	AddQueue(config queue.QueueConfig) (chan *interface{}, error)
	startConsuming(config queue.QueueConfig, msgs chan *interface{}) error
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error
	Schedule(exchange, routingKey string, body []byte, delay int) error

	// Dial() error
	// Setup() error
	// Close() error
	// Consume(queue string) error
	// DeclareQueue(queue string) error
	// BindQueue(exchange, routingKey, queue string) error
	// Publish(ctx context.Context, exchange, routingKey string, body []byte) error
	// DeclareExchange(exchange, kind string, delayed bool) error
	// Schedule(exchange, routingKey string, body []byte, delay int) error
}
