package main

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
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

func processMessage(d amqp.Delivery) {
	log.Printf("Received a message: %s", d.Body)
	// Process the message here

	// Acknowledge the message
	if err := d.Ack(false); err != nil {
		log.Printf("Failed to acknowledge message: %s", err)
	}
}
