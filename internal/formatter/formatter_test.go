package formatter

import (
	"strings"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
)

func TestFormatters_LegacyLink(t *testing.T) {
	paper := &openreview.Note{
		ID: "testID123",
		Content: openreview.NoteContent{
			Title:   openreview.ValueField[string]{Value: "Test Title"},
			Authors: openreview.ValueField[[]string]{Value: []string{"Author A", "Author B"}},
		},
	}
	venue := config.VenueConfig{
		Name:  "ICLR",
		Venue: "ICLR.cc/2025/Conference",
		Year:  2025,
	}

	t.Run("DiscordFormatter returns Message with empty Sub", func(t *testing.T) {
		formatter := NewDiscordFormatter()
		msg := formatter.Format(paper, venue, 100, "")
		expectedLink := "[📄 今日の論文 (ICLR 2025)](https://openreview.net/group?id=ICLR.cc/2025/Conference)"
		if !strings.Contains(msg.Main, expectedLink) {
			t.Errorf("Discord link format is incorrect.\nGot: %s\nExpected to contain: %s", msg.Main, expectedLink)
		}
		if msg.Sub != "" {
			t.Errorf("Discord Sub should always be empty, got %q", msg.Sub)
		}
	})

	t.Run("SlackFormatter returns Message with empty Sub when no translation", func(t *testing.T) {
		formatter := NewSlackFormatter()
		msg := formatter.Format(paper, venue, 100, "")
		expectedLink := "<https://openreview.net/group?id=ICLR.cc/2025/Conference|📄 今日の論文 (ICLR 2025)>"
		if !strings.Contains(msg.Main, expectedLink) {
			t.Errorf("Slack link format is incorrect.\nGot: %s\nExpected to contain: %s", msg.Main, expectedLink)
		}
		if msg.Sub != "" {
			t.Errorf("Slack Sub should be empty when jaAbstract is empty, got %q", msg.Sub)
		}
	})
}
