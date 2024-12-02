package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"io/ioutil"
	"log"
	"net/http"
)

type User struct {
	Username string `json:"username"`
}

var channels []ChannelAction

func (p *Plugin) OnActivate() error {
	channels = p.configuration.channels
	log.Print(p.configuration.ChannelNewLead)
	p.connectRabbitMQ()
	p.consumeMessages("test", processMessage)
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.closeRabbitMQ()
	log.Printf("RabbitMQ connection closed")
	return nil
}

type ActionMessage struct {
	Action  string   `json:"action"`
	Message string   `json:"message"`
	Link    string   `json:"link"`
	Emails  []string `json:"emails"`
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
			err := sendMessageToChannel("17mxst3bz78r9p5itzc8btgi9a", channel.ChannelID, data.Message, data.Link, data.Emails)
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

func getUsernameByEmail(botToken string, email string) (string, error) {
	url := fmt.Sprintf("http://localhost:8065/api/v4/users/email/%s", email)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+botToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error fetching user: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var user User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response: %v", err)
	}

	return user.Username, nil
}

// Hàm gửi tin nhắn đến channel Mattermost
func sendMessageToChannel(botToken, channelId, message, link string, emails []string) error {
	url := "http://localhost:8065/api/v4/posts"

	// Lấy danh sách username từ email
	var tags []string
	for _, email := range emails {
		username, err := getUsernameByEmail(botToken, email)
		if err != nil {
			log.Printf("Failed to get username for email %s: %v", email, err)
			continue
		}
		tags = append(tags, "@"+username)
	}

	// Tạo nội dung tin nhắn kèm link và tag người dùng
	formattedMessage := formatMessage(message, link, tags)

	// Tạo body request
	reqBody := map[string]interface{}{
		"channel_id": channelId,
		"message":    formattedMessage,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshalling request body: %v", err)
	}

	// Tạo request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+botToken)
	req.Header.Set("Content-Type", "application/json")

	// Gửi request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Kiểm tra kết quả
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error sending message: %s", resp.Status)
	}

	log.Println("Message sent successfully!")
	return nil
}

// Hàm định dạng tin nhắn với link và tag người dùng
func formatMessage(message, link string, tags []string) string {
	taggedUsers := ""
	for _, tag := range tags {
		taggedUsers += tag + " "
	}

	// Định dạng tin nhắn kèm link và tag
	formattedMessage := fmt.Sprintf("%s\n\nLink: [%s](%s)\n%s", message, link, link, taggedUsers)
	return formattedMessage
}
