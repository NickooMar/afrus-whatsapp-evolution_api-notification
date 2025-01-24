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
}
