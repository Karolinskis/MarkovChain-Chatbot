// Package llm is a minimal client for the Ollama chat API.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const maxReplyTokens = 80

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	url   string
	model string
	http  *http.Client
}

func New(url, model string) *Client {
	return &Client{
		url:   url,
		model: model,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []Message      `json:"messages"`
	Stream   bool           `json:"stream"`
	Think    bool           `json:"think"`
	Options  map[string]any `json:"options"`
}

type chatResponse struct {
	Message Message `json:"message"`
	Error   string  `json:"error"`
}

// Chat sends messages to the model and returns its reply. Thinking is
// disabled so small reasoning models answer in one short pass.
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: messages,
		Options:  map[string]any{"num_predict": maxReplyTokens},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm chat: %w", err)
	}
	defer resp.Body.Close()

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm chat: status %d: %s", resp.StatusCode, out.Error)
	}
	return out.Message.Content, nil
}
