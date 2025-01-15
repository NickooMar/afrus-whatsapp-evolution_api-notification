package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort                         string `mapstructure:"SERVER_PORT" default:"3008"`
	Environment                        string `mapstructure:"ENVIRONMENT" default:"development"`
	RabbitMQUrl                        string `mapstructure:"RABBITMQ_URL" default:"amqp://guest:guest@localhost:5672/"`
	RabbitMQPort                       string `mapstructure:"RABBITMQ_PORT" default:"5672"`
	RabbitMQManagerPort                string `mapstructure:"RABBITMQ_MANAGER_PORT"`
	RabbitMQUser                       string `mapstructure:"RABBITMQ_USER"`
	RabbitMQPassword                   string `mapstructure:"RABBITMQ_PASSWORD"`
	EvolutionAPINotificationQueue      string `mapstructure:"RABBITMQ_EVOLUTION_API_NOTIFICATION_QUEUE"`
	EvolutionAPINotificationExchange   string `mapstructure:"RABBITMQ_EVOLUTION_API_NOTIFICATION_EXCHANGE"`
	EvolutionAPINotificationRoutingKey string `mapstructure:"RABBITMQ_EVOLUTION_API_NOTIFICATION_ROUTING_KEY"`
	RabbitMQBillingExchange            string `mapstructure:"RABBITMQ_BILLING_EXCHANGE"`
	RabbitMQBillingQueue               string `mapstructure:"RABBITMQ_BILLING_QUEUE"`
	RabbitMQBillingRoutingKey          string `mapstructure:"RABBITMQ_BILLING_ROUTING_KEY"`
	AfrusDBHost                        string `mapstructure:"AFRUS_DB_HOST"`
	AfrusDBPort                        string `mapstructure:"AFRUS_DB_PORT"`
	AfrusDBName                        string `mapstructure:"AFRUS_DB_NAME"`
	AfrusDBUser                        string `mapstructure:"AFRUS_DB_USER"`
	AfrusDBPassword                    string `mapstructure:"AFRUS_DB_PASSWORD"`
	AfrusDBSSLMode                     string `mapstructure:"AFRUS_DB_SSL_MODE"`
	EventsDBHost                       string `mapstructure:"EVENTS_DB_HOST"`
	EventsDBPort                       string `mapstructure:"EVENTS_DB_PORT"`
	EventsDBName                       string `mapstructure:"EVENTS_DB_NAME"`
	EventsDBUser                       string `mapstructure:"EVENTS_DB_USER"`
	EventsDBPassword                   string `mapstructure:"EVENTS_DB_PASSWORD"`
	EventsDBSSLMode                    string `mapstructure:"EVENTS_DB_SSL_MODE"`
	EvolutionAPIBaseURL                string `mapstructure:"EVOLUTION_API_BASE_URL"`
	EvolutionAPIKey                    string `mapstructure:"EVOLUTION_API_KEY"`
}

func LoadConfig(path string) *Config {
	var cfg Config
	viper.SetConfigName("app_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		if err != err.(*os.PathError) {
			return nil
		}
		cfg = Config{
			ServerPort:                         os.Getenv("SERVER_PORT"),
			Environment:                        os.Getenv("ENVIRONMENT"),
			RabbitMQUrl:                        os.Getenv("RABBITMQ_URL"),
			RabbitMQPort:                       os.Getenv("RABBITMQ_PORT"),
			RabbitMQManagerPort:                os.Getenv("RABBITMQ_MANAGER_PORT"),
			RabbitMQUser:                       os.Getenv("RABBITMQ_USER"),
			RabbitMQPassword:                   os.Getenv("RABBITMQ_PASSWORD"),
			EvolutionAPINotificationQueue:      os.Getenv("RABBITMQ_EVOLUTION_API_NOTIFICATION_QUEUE"),
			EvolutionAPINotificationExchange:   os.Getenv("RABBITMQ_EVOLUTION_API_NOTIFICATION_EXCHANGE"),
			EvolutionAPINotificationRoutingKey: os.Getenv("RABBITMQ_EVOLUTION_API_NOTIFICATION_ROUTING_KEY"),
			RabbitMQBillingExchange:            os.Getenv("RABBITMQ_BILLING_EXCHANGE"),
			RabbitMQBillingQueue:               os.Getenv("RABBITMQ_BILLING_QUEUE"),
			RabbitMQBillingRoutingKey:          os.Getenv("RABBITMQ_BILLING_ROUTING_KEY"),
			AfrusDBHost:                        os.Getenv("AFRUS_DB_HOST"),
			AfrusDBPort:                        os.Getenv("AFRUS_DB_PORT"),
			AfrusDBName:                        os.Getenv("AFRUS_DB_NAME"),
			AfrusDBUser:                        os.Getenv("AFRUS_DB_USER"),
			AfrusDBPassword:                    os.Getenv("AFRUS_DB_PASSWORD"),
			AfrusDBSSLMode:                     os.Getenv("AFRUS_DB_SSL_MODE"),
			EventsDBHost:                       os.Getenv("EVENTS_DB_HOST"),
			EventsDBPort:                       os.Getenv("EVENTS_DB_PORT"),
			EventsDBName:                       os.Getenv("EVENTS_DB_NAME"),
			EventsDBUser:                       os.Getenv("EVENTS_DB_USER"),
			EventsDBPassword:                   os.Getenv("EVENTS_DB_PASSWORD"),
			EventsDBSSLMode:                    os.Getenv("EVENTS_DB_SSL_MODE"),
			EvolutionAPIBaseURL:                os.Getenv("EVOLUTION_API_BASE_URL"),
			EvolutionAPIKey:                    os.Getenv("EVOLUTION_API_KEY"),
		}
	} else {
		err = viper.Unmarshal(&cfg)
		if err != nil {
			return nil
		}
	}
	return &cfg
}
