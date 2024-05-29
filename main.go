package main

import (
	"gics-to-kafka/pkg/config"
	"gics-to-kafka/pkg/web"
)

func main() {

	appConfig := config.LoadConfig()
	config.ConfigureLogger(appConfig.App)

	server := web.NewServer(appConfig)
	server.Run()
}
