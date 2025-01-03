package protocols

type Queue interface {
	Dial() error
	Setup() error
	Consume(queue string) error
	DeclareExchange(exchange, kind string) error
	DeclareQueue(queue string) error
	BindQueue(exchange, routingKey, queue string) error
	Close() error
}
