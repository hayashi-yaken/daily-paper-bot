package formatter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
)

func TestFormatPaper(t *testing.T) {
	paper := openreview.Note{
		ID: "testID123",
		Content: openreview.NoteContent{
			Title:    openreview.ValueField[string]{Value: "Test Title"},
			Authors:  openreview.ValueField[[]string]{Value: []string{"Author A", "Author B"}},
			Abstract: openreview.ValueField[string]{Value: "This is a test abstract. It has several words."},
			PDF:      openreview.ValueField[string]{Value: "http://example.com/test.pdf"},
		},
	}
	venue := "ICLR"
	year := 2025

	t.Run("format with full abstract", func(t *testing.T) {
		formatted := FormatPaper(paper, venue, year, 1000)
		expected := fmt.Sprintf(
			"📄 今日の論文 (ICLR 2025)\n\n*Title*: Test Title\n*Authors*: Author A, Author B\n\n*Abstract*:\nThis is a test abstract. It has several words.\n\n*Link*:\nhttp://example.com/test.pdf\n\nID: `%s`",
			"testID123",
		)
		if formatted != expected {
			t.Errorf("formatted text does not match expected.\nGot:\n%s\n\nExpected:\n%s", formatted, expected)
		}
	})

	t.Run("format with truncated abstract", func(t *testing.T) {
		formatted := FormatPaper(paper, venue, year, 20)
		if !strings.Contains(formatted, "This is a test abstr...") {
			t.Errorf("abstract is not truncated correctly. Got: %s", formatted)
		}
		if strings.Contains(formatted, "It has several words.") {
			t.Errorf("truncated abstract should not contain the full text. Got: %s", formatted)
		}
	})

	t.Run("format with no pdf link", func(t *testing.T) {
		paperNoPDF := paper
		paperNoPDF.Content.PDF = openreview.ValueField[string]{Value: ""}
		formatted := FormatPaper(paperNoPDF, venue, year, 1000)
		expectedLink := fmt.Sprintf("https://openreview.net/forum?id=%s", paperNoPDF.ID)
		if !strings.Contains(formatted, expectedLink) {
			t.Errorf("expected link to be forum URL, but it was not. Got: %s", formatted)
		}
	})
}