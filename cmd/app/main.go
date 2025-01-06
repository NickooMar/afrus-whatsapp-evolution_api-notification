package main

import (
	config "afrus-whatsapp-evolution_api-notification/configs"
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"afrus-whatsapp-evolution_api-notification/internal/usecase"
	"afrus-whatsapp-evolution_api-notification/pkg/db"
	"afrus-whatsapp-evolution_api-notification/pkg/queue"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

func main() {
	conf := config.LoadConfig(".")
	if conf == nil {
		panic("Failed to load config")
	}

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

	if err := dbManager.Connect(db.AfrusDB, afrusConfig, &models.WhatsappTrigger{}, &models.WhatsappInstance{}, &models.WhatsappTriggerAttachment{}); err != nil {
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

	if err := dbManager.Connect(db.EventsDB, eventsConfig); err != nil {
		panic(fmt.Sprintf("Failed to connect to Events database: %v", err))
	}

	// Gets db instances
	afrusDB, _ := dbManager.GetDB(db.AfrusDB)
	eventsDB, _ := dbManager.GetDB(db.EventsDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var msgs = make(chan *amqp.Delivery)
	queueCli := queue.NewRabbitMQ(conf, msgs)
	if err := queueCli.Dial(); err != nil {
		panic(fmt.Sprintf("Failed to dial RabbitMQ: %v", err))
	}
	defer queueCli.Close()

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	log.Printf("[SERVICE] - Service started on port %s", conf.ServerPort)

	messageHandler(ctx, conf, msgs, queueCli, afrusDB, eventsDB)
}

func messageHandler(ctx context.Context, config *config.Config, msgs <-chan *amqp.Delivery, queue *queue.RabbitMQ, afrusDB, eventsDB *gorm.DB) {
	if err := queue.Consume(config.EvolutionAPINotificationQueue); err != nil {
		log.Printf("Error starting consumer: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down message handler")
			return
		case msg := <-msgs:
			if msg == nil {
				continue
			}

			handler := usecase.NewReceiptWhatsappEventUseCase(ctx, config, queue, afrusDB, eventsDB)
			if err := handler.Execute(string(msg.Body)); err != nil {
				log.Printf("[ERROR] - Error processing message: %v", err)
				continue
			}
		}
	}

}
