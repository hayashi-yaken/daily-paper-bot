//go:build integration

package notifier

import (
	"os"
	"testing"
)

func TestSlackNotifier_Integration_Post(t *testing.T) {
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	channelID := os.Getenv("SLACK_CHANNEL_ID")

	if botToken == "" || channelID == "" {
		t.Skip("SLACK_BOT_TOKEN and SLACK_CHANNEL_ID must be set for integration tests")
	}

	notifier := NewSlackNotifier(botToken, channelID)
	message := "This is an integration test message for Slack from the Daily Paper Bot."

	err := notifier.Post(message)
	if err != nil {
		t.Fatalf("Failed to post message to Slack: %v", err)
	}

	t.Log("Successfully posted a test message to Slack.")
}
