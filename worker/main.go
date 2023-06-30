package main

import (
	"github.com/SparkSecurity/wakizashi/worker/config"
	"github.com/SparkSecurity/wakizashi/worker/scrape"
	"github.com/SparkSecurity/wakizashi/worker/storage"
)

func main() {
	config.LoadConfig()
	storage.CreateStorage()
	MQConnect()
	defer MQDisconnect()
	scrape.Init()
	defer scrape.Close()
	MQConsume(scrape.Handler)
}
