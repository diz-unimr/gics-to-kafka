package main

import (
	"gics-to-fhir/pkg/config"
	client "gics-to-fhir/pkg/web"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {

	appConfig := loadConfig()
	configureLogger(appConfig.App)

	server := client.NewServer(appConfig)
	server.Run()
}

func loadConfig() config.AppConfig {
	c, err := config.LoadConfig(".")
	if err != nil {
		log.WithError(err).Fatal("Unable to load config file")
	}
	return c
}

func configureLogger(config config.App) {
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
}
