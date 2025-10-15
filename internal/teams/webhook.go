package teams

import (
	logger "ai-agent-go/internal/logger"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type TeamsMessage struct {
	Text string `json:"text"`
}

func SendMessage(webhookURL, message string) error {
	lg := logger.NewLogger()
	lg.Info("teams", "SendMessage", fmt.Sprintf("Enviando mensaje a Teams: %s", webhookURL))
	payload := TeamsMessage{Text: message}
	data, err := json.Marshal(payload)
	if err != nil {
		lg.Error("teams", "SendMessage", fmt.Sprintf("error serializando payload: %v", err))
		return fmt.Errorf("error al serializar mensaje Teams: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		lg.Error("teams", "SendMessage", fmt.Sprintf("error enviando POST a Teams: %v", err))
		return fmt.Errorf("error al enviar mensaje a Teams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		lg.Error("teams", "SendMessage", fmt.Sprintf("Teams devolvió código HTTP %d", resp.StatusCode))
		return fmt.Errorf("teams devolvió código HTTP %d", resp.StatusCode)
	}
	lg.Info("teams", "SendMessage", "Mensaje enviado correctamente")
	return nil
}
