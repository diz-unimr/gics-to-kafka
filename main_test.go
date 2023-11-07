package main

import (
	"gics-to-kafka/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadDefaultConfig(t *testing.T) {

	defaultConfig := config.AppConfig{
		App: config.App{
			Name:     "gics-to-kafka",
			LogLevel: "info",
			Http:     config.Http{Port: "8080"},
		},
		Kafka: config.Kafka{
			BootstrapServers: "localhost:9092",
			OutputTopic:      "gics-notification",
			SecurityProtocol: "ssl",
			Ssl: config.Ssl{
				CaLocation:          "/app/cert/kafka-ca.pem",
				CertificateLocation: "/app/cert/app-cert.pem",
				KeyLocation:         "/app/cert/app-key.pem",
			},
		},
	}
	actual := loadConfig()

	assert.Equal(t, defaultConfig, actual)
}

func TestConfigureLogger(t *testing.T) {
	configureLogger(config.App{LogLevel: "debug"})

	assert.Equal(t, log.GetLevel().String(), log.DebugLevel.String())
}
