package openreview

import (
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
		client := NewClient()

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
