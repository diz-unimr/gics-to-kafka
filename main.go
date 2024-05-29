package main

import (
	"gics-to-kafka/pkg/config"
	"gics-to-kafka/pkg/web"
	"os"
)

func main() {

	appConfig := config.LoadConfig(".")
	if appConfig == nil {
		os.Exit(1)
	}
	config.ConfigureLogger(appConfig.App)

	server := web.NewServer(*appConfig)
	server.Run()
}
