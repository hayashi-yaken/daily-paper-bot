package notifier

import (
	"fmt"
	"log"

	"github.com/hayashi-yaken/daily-paper-bot/internal/formatter"
	"github.com/slack-go/slack"
)

// apiPoster は slack.Client.PostMessage を抽象化し、テストでモックできるようにするためのインターフェースです。
type apiPoster interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

// SlackNotifier はSlackにメッセージを投稿します。
type SlackNotifier struct {
	poster    apiPoster
	channelID string
}

// NewSlackNotifier は新しいSlackNotifierを生成します。
func NewSlackNotifier(botToken, channelID string) *SlackNotifier {
	client := slack.New(botToken)
	return &SlackNotifier{
		poster:    client,
		channelID: channelID,
	}
}

// Post は指定されたメッセージをSlackチャンネルに投稿します。
func (n *SlackNotifier) Post(msg formatter.Message) error {
	_, parentTS, err := n.poster.PostMessage(
		n.channelID,
		slack.MsgOptionText(msg.Main, false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return fmt.Errorf("failed to post message to slack: %w", err)
	}

	if msg.Sub == "" {
		return nil
	}

	if _, _, threadErr := n.poster.PostMessage(
		n.channelID,
		slack.MsgOptionText(msg.Sub, false),
		slack.MsgOptionAsUser(true),
		slack.MsgOptionTS(parentTS),
	); threadErr != nil {
		log.Printf("WARN: failed to post thread reply to slack (parent succeeded): %v", threadErr)
	}
	return nil
}
