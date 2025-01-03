package protocols

type Queue interface {
	Dial() error
	Consume(queue string) error
	Close() error
}
