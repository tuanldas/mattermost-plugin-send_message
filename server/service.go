package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type User struct {
	Username string `json:"username"`
}

func getUsernameByEmail(appHost string, botToken string, email string) (string, error) {
	url := fmt.Sprintf(appHost+"/api/v4/users/email/%s", email)

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
func sendMessageToChannel(appHost, botToken, channelId, message, link string, emails []string) error {
	url := appHost + `/api/v4/posts`

	// Lấy danh sách username từ email
	var tags []string
	for _, email := range emails {
		username, err := getUsernameByEmail(appHost, botToken, email)
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
