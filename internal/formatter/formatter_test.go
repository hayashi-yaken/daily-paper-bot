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

func TestFormatters_PDFLine(t *testing.T) {
	venue := config.VenueConfig{Name: "ICLR", Venue: "ICLR.cc/2025/Conference", Year: 2025}

	t.Run("Slack with PDF shows *PDF* line", func(t *testing.T) {
		paper := &openreview.Note{
			ID: "PID",
			Content: openreview.NoteContent{
				Title:   openreview.ValueField[string]{Value: "T"},
				Authors: openreview.ValueField[[]string]{Value: []string{"A"}},
				PDF:     openreview.ValueField[string]{Value: "/pdf?id=PID"},
			},
		}
		msg := NewSlackFormatter().Format(paper, venue, 100, "")
		if !strings.Contains(msg.Main, "*PDF*: https://openreview.net/pdf?id=PID") {
			t.Errorf("expected Slack output to contain '*PDF*: ...'.\nGot: %s", msg.Main)
		}
		if strings.Contains(msg.Main, "*Link*:") {
			t.Errorf("expected Slack output not to contain legacy '*Link*:'.\nGot: %s", msg.Main)
		}
	})

	t.Run("Slack without PDF omits link line", func(t *testing.T) {
		paper := &openreview.Note{
			ID: "PID",
			Content: openreview.NoteContent{
				Title:   openreview.ValueField[string]{Value: "T"},
				Authors: openreview.ValueField[[]string]{Value: []string{"A"}},
			},
		}
		msg := NewSlackFormatter().Format(paper, venue, 100, "")
		if strings.Contains(msg.Main, "*PDF*:") {
			t.Errorf("expected no *PDF*: line when PDF is missing.\nGot: %s", msg.Main)
		}
		if strings.Contains(msg.Main, "*Link*:") {
			t.Errorf("expected no legacy *Link*: line.\nGot: %s", msg.Main)
		}
	})

	t.Run("Discord with PDF shows *PDF* line", func(t *testing.T) {
		paper := &openreview.Note{
			ID: "PID",
			Content: openreview.NoteContent{
				Title:   openreview.ValueField[string]{Value: "T"},
				Authors: openreview.ValueField[[]string]{Value: []string{"A"}},
				PDF:     openreview.ValueField[string]{Value: "/pdf?id=PID"},
			},
		}
		msg := NewDiscordFormatter().Format(paper, venue, 100, "")
		if !strings.Contains(msg.Main, "*PDF*: https://openreview.net/pdf?id=PID") {
			t.Errorf("expected Discord output to contain '*PDF*: ...'.\nGot: %s", msg.Main)
		}
	})
}
