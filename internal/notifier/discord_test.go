package notifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/formatter"
)

func TestDiscordNotifier_Post(t *testing.T) {
	t.Run("post success sends Main as content", func(t *testing.T) {
		var receivedBody []byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		notifier := NewDiscordNotifier(server.URL)
		err := notifier.Post(formatter.Message{Main: "hello", Sub: "ignored"})
		if err != nil {
			t.Errorf("Post() should not return an error, but got: %v", err)
		}

		var payload struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(receivedBody, &payload); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if payload.Content != "hello" {
			t.Errorf("expected content 'hello', got %q", payload.Content)
		}
	})

	t.Run("post failure due to server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		notifier := NewDiscordNotifier(server.URL)
		err := notifier.Post(formatter.Message{Main: "test"})
		if err == nil {
			t.Error("Post() should return an error for non-2xx status, but got nil")
		}
	})

	t.Run("post failure due to invalid url", func(t *testing.T) {
		notifier := NewDiscordNotifier("http://localhost:99999")
		err := notifier.Post(formatter.Message{Main: "test"})
		if err == nil {
			t.Error("Post() should return an error for invalid URL, but got nil")
		}
	})
}
