package notifier

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscordNotifier_Post(t *testing.T) {
	t.Run("post success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent) // Discordは成功時に204を返す
		}))
		defer server.Close()

		notifier := NewDiscordNotifier(server.URL)
		err := notifier.Post("test message")
		if err != nil {
			t.Errorf("Post() should not return an error, but got: %v", err)
		}
	})

	t.Run("post failure due to server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		notifier := NewDiscordNotifier(server.URL)
		err := notifier.Post("test message")
		if err == nil {
			t.Error("Post() should return an error for non-2xx status, but got nil")
		}
	})

	t.Run("post failure due to invalid url", func(t *testing.T) {
		// 無効なURL（例: 存在しないサーバー）
		notifier := NewDiscordNotifier("http://localhost:99999")
		err := notifier.Post("test message")
		if err == nil {
			t.Error("Post() should return an error for invalid URL, but got nil")
		}
	})
}
