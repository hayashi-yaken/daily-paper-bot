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

	t.Run("post with Sub triggers thread reply", func(t *testing.T) {
		mock := &mockAPIPoster{}
		notifier := &SlackNotifier{poster: mock, channelID: "C12345"}

		err := notifier.Post(formatter.Message{Main: "main text", Sub: "thread text"})
		if err != nil {
			t.Fatalf("Post returned error: %v", err)
		}
		if len(mock.calls) != 2 {
			t.Fatalf("expected 2 PostMessage calls (parent + thread), got %d", len(mock.calls))
		}
		if mock.calls[0].channelID != "C12345" || mock.calls[1].channelID != "C12345" {
			t.Errorf("expected both posts to channel C12345, got %q and %q", mock.calls[0].channelID, mock.calls[1].channelID)
		}
	})

	t.Run("thread post failure does not fail Post", func(t *testing.T) {
		mock := &flakeyPoster{failAfter: 1}
		notifier := &SlackNotifier{poster: mock, channelID: "C12345"}

		err := notifier.Post(formatter.Message{Main: "main text", Sub: "thread text"})
		if err != nil {
			t.Errorf("Post should not return error when only thread reply fails, got: %v", err)
		}
		if mock.callCount != 2 {
			t.Errorf("expected 2 PostMessage attempts, got %d", mock.callCount)
		}
	})
}

type flakeyPoster struct {
	failAfter int
	callCount int
}

func (f *flakeyPoster) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	f.callCount++
	if f.callCount > f.failAfter {
		return "", "", errors.New("mock thread failure")
	}
	return channelID, "12345.67890", nil
}
