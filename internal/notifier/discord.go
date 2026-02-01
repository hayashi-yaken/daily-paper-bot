package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DiscordNotifier はDiscordのWebhookにメッセージを投稿します。
type DiscordNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// NewDiscordNotifier は新しいDiscordNotifierを生成します。
func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// discordPayload はDiscord Webhookに送信するJSONの構造体です。
type discordPayload struct {
	Content string `json:"content"`
}

// Post は指定されたメッセージをDiscordのWebhookに投稿します。
func (n *DiscordNotifier) Post(message string) error {
	payload := discordPayload{Content: message}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post message to discord: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
