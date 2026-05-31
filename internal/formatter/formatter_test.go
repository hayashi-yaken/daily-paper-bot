package formatter

import (
	"strings"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
)

func TestFormatters_HeaderLinkPointsToPaper(t *testing.T) {
	paper := &openreview.Note{
		ID: "ABC123",
		Content: openreview.NoteContent{
			Title:   openreview.ValueField[string]{Value: "T"},
			Authors: openreview.ValueField[[]string]{Value: []string{"A"}},
		},
	}
	venue := config.VenueConfig{Name: "ICLR", Venue: "ICLR.cc/2025/Conference", Year: 2025}

	t.Run("Slack header links to forum page", func(t *testing.T) {
		msg := NewSlackFormatter().Format(paper, venue, 100, "")
		wantLink := "<https://openreview.net/forum?id=ABC123|📄 今日の論文 (ICLR 2025)>"
		if !strings.Contains(msg.Main, wantLink) {
			t.Errorf("Slack header link wrong.\nGot: %s\nWant contains: %s", msg.Main, wantLink)
		}
	})

	t.Run("Discord header links to forum page", func(t *testing.T) {
		msg := NewDiscordFormatter().Format(paper, venue, 100, "")
		wantLink := "[📄 今日の論文 (ICLR 2025)](https://openreview.net/forum?id=ABC123)"
		if !strings.Contains(msg.Main, wantLink) {
			t.Errorf("Discord header link wrong.\nGot: %s\nWant contains: %s", msg.Main, wantLink)
		}
	})
}
