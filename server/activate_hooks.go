package main

import (
	"log"
)

func (p *Plugin) OnActivate() error {
	p.connectRabbitMQ()
	p.consumeMessages("test", processMessage)
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.closeRabbitMQ()
	log.Printf("RabbitMQ connection closed")
	return nil
}

func processMessage(body []byte) {
	log.Printf("Received message: %s", string(body))
}
