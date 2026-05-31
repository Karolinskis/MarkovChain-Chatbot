package discord

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

var webhookURL string

// Init sets the Discord webhook URL. If empty, Notify is a no-op.
func Init(url string) {
	webhookURL = url
}

// Notify sends a message to the configured Discord webhook.
// It is a no-op if no webhook URL is configured.
func Notify(message string) {
	if webhookURL == "" {
		return
	}

	go func() {
		payload := map[string]string{"content": message}
		data, err := json.Marshal(payload)
		if err != nil {
			slog.Error("discord: failed to marshal payload", "error", err)
			return
		}

		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
		if err != nil {
			slog.Error("discord: failed to send webhook", "error", err)
			return
		}
		resp.Body.Close()
	}()
}
