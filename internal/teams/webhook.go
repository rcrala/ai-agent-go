package teams

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type TeamsMessage struct {
	Text string `json:"text"`
}

func SendMessage(webhookURL, message string) error {
	payload := TeamsMessage{Text: message}
	data, _ := json.Marshal(payload)

	_, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	return err
}
