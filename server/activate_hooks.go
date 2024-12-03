package main

import (
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

type ActionMessage struct {
	Action  string   `json:"action"`
	Message string   `json:"message"`
	Link    string   `json:"link"`
	Emails  []string `json:"emails"`
}

var channels []ChannelAction
var botToken string
var appHost string

func (p *Plugin) OnActivate() error {
	channels = p.configuration.channels
	botToken = p.configuration.BotToken
	appHost = p.configuration.AppHost

	p.connectRabbitMQ()
	p.consumeMessages(p.configuration.RabbitmqQueueName, processMessage)
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.closeRabbitMQ()
	log.Printf("RabbitMQ connection closed")
	return nil
}

func (am *ActionMessage) UnmarshalJSON(data []byte) error {
	var temp struct {
		Action  string          `json:"action"`
		Message string          `json:"message"`
		Link    string          `json:"link"`
		Emails  json.RawMessage `json:"emails"` // Sử dụng RawMessage để xử lý dữ liệu chưa biết
	}

	// Giải mã JSON vào struct tạm
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	am.Action = temp.Action
	am.Message = temp.Message
	am.Link = temp.Link

	// Xử lý trường Emails có thể là object hoặc mảng
	var emails []string

	// Nếu emails là object (ví dụ {"0": "admin@codegym.vn"})
	var emailsMap map[string]string
	if err := json.Unmarshal(temp.Emails, &emailsMap); err == nil {
		for _, email := range emailsMap {
			emails = append(emails, email)
		}
	} else {
		// Nếu emails là mảng (ví dụ ["admin@codegym.vn"])
		if err := json.Unmarshal(temp.Emails, &emails); err != nil {
			return err
		}
	}

	am.Emails = emails
	return nil
}

func processMessage(d amqp.Delivery) {
	log.Printf("Received a message: %s", d.Body)
	// Process the message here

	var data ActionMessage
	// Giải mã JSON vào mảng leads
	if err := json.Unmarshal([]byte(d.Body), &data); err != nil {
		log.Fatalf("Error unmarshaling JSON: %s", err)
	}

	for _, channel := range channels {
		if channel.Action == data.Action {
			err := sendMessageToChannel(appHost, botToken, channel.ChannelID, data.Message, data.Link, data.Emails)
			if err != nil {
				return
			}
		}
	}
	// Acknowledge the message
	if err := d.Ack(false); err != nil {
		log.Printf("Failed to acknowledge message: %s", err)
	}
}
