package notifier

import (
	"fmt"

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
	_, _, err := n.poster.PostMessage(
		n.channelID,
		slack.MsgOptionText(msg.Main, false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return fmt.Errorf("failed to post message to slack: %w", err)
	}
	return nil
}
