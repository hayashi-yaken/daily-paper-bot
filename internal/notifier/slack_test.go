package notifier

import (
	"errors"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/formatter"
	"github.com/slack-go/slack"
)

type mockAPIPoster struct {
	shouldFail bool
	calls      []struct {
		channelID string
		options   []slack.MsgOption
	}
}

func (m *mockAPIPoster) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	m.calls = append(m.calls, struct {
		channelID string
		options   []slack.MsgOption
	}{channelID, options})
	if m.shouldFail {
		return "", "", errors.New("mock post error")
	}
	return channelID, "12345.67890", nil
}

func TestSlackNotifier_Post(t *testing.T) {
	t.Run("post success without Sub calls PostMessage once", func(t *testing.T) {
		mock := &mockAPIPoster{}
		notifier := &SlackNotifier{poster: mock, channelID: "C12345"}

		if err := notifier.Post(formatter.Message{Main: "hello"}); err != nil {
			t.Fatalf("Post returned error: %v", err)
		}
		if len(mock.calls) != 1 {
			t.Errorf("expected 1 PostMessage call, got %d", len(mock.calls))
		}
	})

	t.Run("parent post failure returns error", func(t *testing.T) {
		mock := &mockAPIPoster{shouldFail: true}
		notifier := &SlackNotifier{poster: mock, channelID: "C12345"}

		if err := notifier.Post(formatter.Message{Main: "hello"}); err == nil {
			t.Error("expected error when parent post fails")
		}
	})
}
