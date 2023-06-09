package main

import (
	"gics-to-kafka/pkg/config"
	"gics-to-kafka/pkg/web"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {

	appConfig := loadConfig()
	configureLogger(appConfig.App)

	server := web.NewServer(appConfig)
	server.Run()
}

func loadConfig() config.AppConfig {
	c, err := config.LoadConfig(".")
	if err != nil {
		log.WithError(err).Fatal("Unable to load config file")
		os.Exit(1)
	}
	return *c
}

func configureLogger(config config.App) {

	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
	log.SetOutput(os.Stdout)
	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
}
