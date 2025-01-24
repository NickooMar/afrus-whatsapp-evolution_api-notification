package main

import (
	config "afrus-whatsapp-evolution_api-notification/configs"
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"afrus-whatsapp-evolution_api-notification/internal/domain/models/events"
	"afrus-whatsapp-evolution_api-notification/internal/services"
	"afrus-whatsapp-evolution_api-notification/internal/usecase"
	"afrus-whatsapp-evolution_api-notification/pkg/db"
	"afrus-whatsapp-evolution_api-notification/pkg/queue"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conf := config.LoadConfig(".")
	if conf == nil {
		panic("Failed to load config")
	}

	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbManager := db.NewDatabaseManager()
	defer dbManager.CloseAll()

	afrusConfig := db.DBConfig{
		Host:     conf.AfrusDBHost,
		User:     conf.AfrusDBUser,
		Password: conf.AfrusDBPassword,
		DBName:   conf.AfrusDBName,
		Port:     conf.AfrusDBPort,
		SSLMode:  conf.AfrusDBSSLMode,
	}

	if err := dbManager.Connect(db.AfrusDB, afrusConfig, &models.WhatsappTrigger{}, &models.WhatsappTriggerAttachment{}, &models.CommunicationWhatsapp{}, &models.CommunicationWhatsappAttachment{}); err != nil {
		panic(fmt.Sprintf("Failed to connect to Afrus database: %v", err))
	}

	eventsConfig := db.DBConfig{
		Host:     conf.EventsDBHost,
		User:     conf.EventsDBUser,
		Password: conf.EventsDBPassword,
		DBName:   conf.EventsDBName,
		Port:     conf.EventsDBPort,
		SSLMode:  conf.EventsDBSSLMode,
	}

	if err := dbManager.Connect(db.EventsDB, eventsConfig, &events.Sent{}, &events.Accepted{}, &events.Canceled{}, &events.Delivered{}, &events.Failed{}, &events.PartiallyDelivered{}, &events.Queued{}, &events.Read{}, &events.Scheduled{}); err != nil {
		panic(fmt.Sprintf("Failed to connect to Events database: %v", err))
	}

	afrusDB, _ := dbManager.GetDB(db.AfrusDB)
	eventsDB, _ := dbManager.GetDB(db.EventsDB)

	databases := &db.DBConnections{
		Afrus:    afrusDB,
		EventsDB: eventsDB,
	}

	rabbitMQ := queue.NewRabbitMQ(conf)

	blastsMessages, err := rabbitMQ.AddQueue(queue.QueueConfig{
		Name:       conf.EvolutionAPINotificationBlastQueue,
		BufferSize: 10,
		Consumer:   "blast-consumer",
	})
	if err != nil {
		log.Fatalf("[RABBITMQ] - Error adding blast queue: %v", err)
	}

	autoresponderMessages, err := rabbitMQ.AddQueue(queue.QueueConfig{
		Name:       conf.EvolutionAPINotificationAutoresponderQueue,
		BufferSize: 100,
		Consumer:   "autoresponder-consumer",
	})
	if err != nil {
		log.Fatalf("[RABBITMQ] - Error adding autoresponder queue: %v", err)
	}

	if err := rabbitMQ.Connect(); err != nil {
		log.Fatalf("[RABBITMQ] - Error connecting to RabbitMQ: %v", err)
	}
	defer rabbitMQ.Close()

	whatsappSenderService := services.NewWhatsappSenderService(conf)

	go processBlastEvent(conf, blastsMessages, databases, rabbitMQ, whatsappSenderService)
	go processAutoresponderEvent(conf, autoresponderMessages, databases, rabbitMQ, whatsappSenderService)

	if err := waitForShutdown(ctx, cancel, errChan, rabbitMQ); err != nil {
		log.Fatalf("[SHUTDOWN] - Error during shutdown: %v", err)
	}

}

func processAutoresponderEvent(config *config.Config, msgs <-chan *amqp.Delivery, databases *db.DBConnections, rabbitMQ *queue.RabbitMQ, service *services.WhatsappSenderService) {
	var wg sync.WaitGroup
	numWorkers := 300
	workerPool := make(chan struct{}, numWorkers)

	for msg := range msgs {
		wg.Add(1)
		workerPool <- struct{}{}

		go func(msg *amqp.Delivery) {
			defer wg.Done()
			defer func() { <-workerPool }()

			log.Printf("[INFO] - Received an autoresponder message\n")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			handler := usecase.NewReceiptAutoresponderEventUseCase(ctx, config, rabbitMQ, databases.Afrus, databases.EventsDB, service)
			if err := handler.Execute(string(msg.Body)); err != nil {
				log.Printf("[ERROR] - Error processing message: %v", err)
				msg.Nack(false, false)
				return
			}

			if err := msg.Ack(false); err != nil {
				log.Printf("[ERROR] - Error acknowledging message: %v", err)
			}
		}(msg)
	}

	wg.Wait()
	close(workerPool)
}

func processBlastEvent(config *config.Config, msgs <-chan *amqp.Delivery, databases *db.DBConnections, rabbitMQ *queue.RabbitMQ, service *services.WhatsappSenderService) {
	var wg sync.WaitGroup
	numWorkers := 300
	workerPool := make(chan struct{}, numWorkers)

	for msg := range msgs {
		wg.Add(1)
		workerPool <- struct{}{}

		go func(msg *amqp.Delivery) {
			defer wg.Done()
			defer func() { <-workerPool }()

			log.Printf("[INFO] - Received an blast message\n")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			handler := usecase.NewReceiptBlastEventUseCase(ctx, config, rabbitMQ, databases.Afrus, databases.EventsDB, service)
			if err := handler.Execute(string(msg.Body)); err != nil {
				log.Printf("[ERROR] - Error processing message: %v", err)
				msg.Nack(false, false)
				return
			}

			if err := msg.Ack(false); err != nil {
				log.Printf("[ERROR] - Error acknowledging message: %v", err)
			}
		}(msg)
	}

	wg.Wait()
	close(workerPool)
}

func waitForShutdown(ctx context.Context, cancel context.CancelFunc, errChan <-chan error, rabbit *queue.RabbitMQ) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("[SHUTDOWN] - Received shutdown signal")
	case err := <-errChan:
		cancel()
		return err
	}

	cancel()

	// Initiate server shutdown with a timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Wait for RabbitMQ connections and consumers to be cleaned up
	done := make(chan struct{})
	go func() {
		if err := rabbit.Close(); err != nil {
			log.Printf("[SHUTDOWN] - Error closing RabbitMQ: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
		log.Println("[SHUTDOWN] - RabbitMQ closed successfully")
	case <-shutdownCtx.Done():
		log.Println("[SHUTDOWN] - Timeout while waiting for RabbitMQ to close")
	}

	return nil
}
