package formatter

import (
	"strings"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
)

func TestFormatters(t *testing.T) {
	paper := &openreview.Note{
		ID: "testID123",
		Content: openreview.NoteContent{
			Title:   openreview.ValueField[string]{Value: "Test Title"},
			Authors: openreview.ValueField[[]string]{Value: []string{"Author A", "Author B"}},
		},
	}
	venue := "ICLR.cc/2025/Conference"
	year := 2025

	t.Run("DiscordFormatter", func(t *testing.T) {
		formatter := NewDiscordFormatter()
		formatted := formatter.Format(paper, venue, year, 100)
		expectedLink := "[📄 今日の論文 (ICLR 2025)](https://openreview.net/group?id=ICLR.cc/2025/Conference)"
		if !strings.Contains(formatted, expectedLink) {
			t.Errorf("Discord link format is incorrect.\nGot: %s\nExpected to contain: %s", formatted, expectedLink)
		}
	})

	t.Run("SlackFormatter", func(t *testing.T) {
		formatter := NewSlackFormatter()
		formatted := formatter.Format(paper, venue, year, 100)
		expectedLink := "<https://openreview.net/group?id=ICLR.cc/2025/Conference|📄 今日の論文 (ICLR 2025)>"
		if !strings.Contains(formatted, expectedLink) {
			t.Errorf("Slack link format is incorrect.\nGot: %s\nExpected to contain: %s", formatted, expectedLink)
		}
	})
}
