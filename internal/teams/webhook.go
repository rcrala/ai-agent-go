package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type TeamsMessage struct {
	Text string `json:"text"`
}

func SendMessage(webhookURL, message string) error {
	payload := TeamsMessage{Text: message}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error al serializar mensaje Teams: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error al enviar mensaje a Teams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("teams devolvió código HTTP %d", resp.StatusCode)
	}
	return nil
}