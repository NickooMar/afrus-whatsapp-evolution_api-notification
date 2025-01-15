package protocols

import "context"

type Queue interface {
	Dial() error
	Setup() error
	Close() error
	Consume(queue string) error
	DeclareQueue(queue string) error
	BindQueue(exchange, routingKey, queue string) error
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error
	DeclareExchange(exchange, kind string, delayed bool) error
	Schedule(exchange, routingKey string, body []byte, delay int) error
}
