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

func (rmq *RabbitMQ) Setup() error {
	// Declare exchanges
	exchanges := []struct {
		Name string
		Type string
	}{
		{rmq.Configs.EvolutionAPINotificationExchange, "direct"},
	}

	for _, ex := range exchanges {
		if err := rmq.DeclareExchange(ex.Name, ex.Type); err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", ex.Name, err)
		}
	}

	// Declare queues
	queues := []string{rmq.Configs.EvolutionAPINotificationQueue}

	for _, q := range queues {
		if err := rmq.DeclareQueue(q); err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", q, err)
		}
	}

	// Bind queues to exchanges
	bindings := []struct {
		Queue      string
		Exchange   string
		RoutingKey string
	}{
		{rmq.Configs.EvolutionAPINotificationQueue, rmq.Configs.EvolutionAPINotificationExchange, rmq.Configs.EvolutionAPINotificationRoutingKey},
	}

	for _, b := range bindings {
		if err := rmq.BindQueue(b.Exchange, b.RoutingKey, b.Queue); err != nil {
			return fmt.Errorf("failed to bind queue %s to exchange %s: %w", b.Queue, b.Exchange, err)
		}
	}

	log.Printf("[RABBITMQ] - Setup completed\n")

	return nil
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

	// Declare exchanges, queues and bindings
	if err := rmq.Setup(); err != nil {
		return fmt.Errorf("failed to set up RabbitMQ: %w", err)
	}

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

func (rmq *RabbitMQ) DeclareExchange(exchange, exType string) error {
	err := rmq.Channel.ExchangeDeclare(
		exchange,
		exType,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (rmq *RabbitMQ) DeclareQueue(queue string) error {
	_, err := rmq.Channel.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (rmq *RabbitMQ) BindQueue(exchange, routingKey, queue string) error {
	err := rmq.Channel.QueueBind(
		queue,
		routingKey,
		exchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}
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
