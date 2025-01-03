package main

import (
	config "afrus-whatsapp-evolution_api-notification/configs"
	"afrus-whatsapp-evolution_api-notification/pkg/db"
	"afrus-whatsapp-evolution_api-notification/pkg/queue"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
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
	}

	if err := dbManager.Connect(db.AfrusDB, afrusConfig); err != nil {
		panic(fmt.Sprintf("Failed to connect to Afrus database: %v", err))
	}

	eventsConfig := db.DBConfig{
		Host:     conf.EventsDBHost,
		User:     conf.EventsDBUser,
		Password: conf.EventsDBPassword,
		DBName:   conf.EventsDBName,
		Port:     conf.EventsDBPort,
	}

	if err := dbManager.Connect(db.EventsDB, eventsConfig); err != nil {
		panic(fmt.Sprintf("Failed to connect to Events database: %v", err))
	}

	// // Gets db instances
	// afrusDB, _ := dbManager.GetDB(db.AfrusDB)
	// eventsDb, _ := dbManager.GetDB(db.EventsDB)

	// fmt.Printf("Afrus DB: %v\n", afrusDB)
	// fmt.Printf("Events DB: %v\n", eventsDb)

	var msgs = make(chan *amqp.Delivery)
	queueCli := queue.NewRabbitMQ(conf, msgs)
	if err := queueCli.Dial(); err != nil {
		panic(fmt.Sprintf("Failed to dial RabbitMQ: %v", err))
	}
	defer queueCli.Close()

	log.Printf("[SERVICE] - Service started on port %s", conf.ServerPort)
}
