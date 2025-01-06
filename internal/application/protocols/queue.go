package protocols

type Queue interface {
	Dial() error
	Setup() error
	Close() error
	Consume(queue string) error
	DeclareQueue(queue string) error
	BindQueue(exchange, routingKey, queue string) error
	Publish(exchange, routingKey string, body []byte) error
	DeclareExchange(exchange, kind string, delayed bool) error
	Schedule(exchange, routingKey string, body []byte, delay int) error
}
