//go:build integration

package notifier

import (
	"os"
	"testing"
)

func TestDiscordNotifier_Integration_Post(t *testing.T) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")

	if webhookURL == "" {
		t.Skip("DISCORD_WEBHOOK_URL must be set for integration tests")
	}

	notifier := NewDiscordNotifier(webhookURL)
	message := "This is an integration test message for Discord from the Daily Paper Bot."

	err := notifier.Post(message)
	if err != nil {
		t.Fatalf("Failed to post message to Discord: %v", err)
	}

	t.Log("Successfully posted a test message to Discord.")
}
