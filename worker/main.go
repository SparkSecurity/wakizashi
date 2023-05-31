package main

func main() {
	LoadConfig()
	MQConnect()
	defer MQDisconnect()
	ScrapeInit()
	defer ScrapeClose()
	MQConsume(ScrapeHandler)
}
