package notifier

import (
	"errors"
	"testing"

	"github.com/slack-go/slack"
)

// mockAPIPoster は apiPoster インターフェースのモック実装です。
type mockAPIPoster struct {
	shouldFail bool
}

func (m *mockAPIPoster) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	if m.shouldFail {
		return "", "", errors.New("mock post error")
	}
	// 成功した場合は、実際のAPIが返すようなダミーの値を返す
	return channelID, "12345.67890", nil
}

func TestSlackNotifier_Post(t *testing.T) {
	t.Run("post success", func(t *testing.T) {
		notifier := &SlackNotifier{
			poster:    &mockAPIPoster{shouldFail: false},
			channelID: "C12345",
		}
		err := notifier.Post("test message")
		if err != nil {
			t.Errorf("Post() should not return an error, but got: %v", err)
		}
	})

	t.Run("post failure", func(t *testing.T) {
		notifier := &SlackNotifier{
			poster:    &mockAPIPoster{shouldFail: true},
			channelID: "C12345",
		}
		err := notifier.Post("test message")
		if err == nil {
			t.Error("Post() should return an error, but got nil")
		}
	})
}
