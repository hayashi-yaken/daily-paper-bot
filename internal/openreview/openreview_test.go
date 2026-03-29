package openreview

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestGetNotes_Integration は、実際のOpenReview APIにアクセスしてデータを取得する統合テストです。
// このテストを実行するには、インターネット接続が必要です。
// `go test -v -tags=integration` のようにビルドタグを使って、普段のユニットテストと分けて実行するのが一般的です。
func TestGetNotes_Integration(t *testing.T) {
	// CI環境など、特定の条件下でのみ実行したい場合
	if os.Getenv("CI") == "" {
		t.Skip("Skipping integration test; set CI environment variable to run.")
	}

	t.Run("fetch from live server", func(t *testing.T) {
		client := NewClient("daily-paper-bot-integration-test/1.0")

		// ICLR 2024 のような、確実にデータが存在する過去のカンファレンスを対象とする
		venue := "ICLR.cc/2024/Conference"
		notes, err := client.GetNotes(venue)

		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}

		if len(notes) == 0 {
			t.Fatalf("expected to fetch at least one note, but got 0")
		}

		// 最初の1件のデータが基本的なフィールドを持っているかを確認
		firstNote := notes[0]
		if firstNote.ID == "" {
			t.Error("expected first note to have an ID, but it was empty")
		}
		if firstNote.Content.Title.Value == "" {
			t.Error("expected first note to have a title, but it was empty")
		}

		t.Logf("Successfully fetched %d notes. First note ID: %s, Title: %s", len(notes), firstNote.ID, firstNote.Content.Title.Value)
	})
}

func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"token": "test-jwt-token", "user": {}}`)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL

	err := client.Login("user@example.com", "password")
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if client.token != "test-jwt-token" {
		t.Errorf("expected token 'test-jwt-token', got '%s'", client.token)
	}
}

func TestLogin_Failure_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL

	err := client.Login("user@example.com", "wrong-password")
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
}

func TestLogin_Failure_EmptyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"token":""}`)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL

	err := client.Login("user@example.com", "password")
	if err == nil {
		t.Fatal("expected an error for empty token, but got nil")
	}
}

func TestGetNotes_WithAuthToken(t *testing.T) {
	var capturedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"notes": [], "count": 0}`)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL
	client.token = "test-jwt-token"

	_, err := client.GetNotes("TestVenue/Conference")
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if capturedAuthHeader != "Bearer test-jwt-token" {
		t.Errorf("expected Authorization header 'Bearer test-jwt-token', got '%s'", capturedAuthHeader)
	}
}

func TestGetNotes_WithoutAuthToken(t *testing.T) {
	var capturedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"notes": [], "count": 0}`)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL
	// token は空のまま（デフォルト）

	_, err := client.GetNotes("TestVenue/Conference")
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if capturedAuthHeader != "" {
		t.Errorf("expected no Authorization header, got '%s'", capturedAuthHeader)
	}
}
