package formatter

import (
	"fmt"
	"strings"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
)

// Message は投稿メッセージのペアを表します。
// Main は親メッセージ（または単発メッセージ）、Sub は Slack スレッド子用の補助メッセージ。
// Discord は Sub を無視します。
type Message struct {
	Main string
	Sub  string
}

// Formatter は論文情報をプラットフォーム別のメッセージに整形するインターフェースです。
type Formatter interface {
	Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int, jaAbstract string) Message
}

// --- Discord Formatter (Standard Markdown) ---

type discordFormatter struct{}

// NewDiscordFormatter は Discord 用の Formatter を返します。
func NewDiscordFormatter() Formatter {
	return &discordFormatter{}
}

func (f *discordFormatter) Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int, jaAbstract string) Message {
	paperLink := fmt.Sprintf("https://openreview.net/forum?id=%s", paper.ID)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("[%s](%s)", headerText, paperLink)

	main := formatMessage(paper, header, abstractBlock(paper.Content.Abstract.Value, jaAbstract, abstractMaxChars))
	return Message{Main: main}
}

// --- Slack Formatter (Slack Mrkdwn) ---

type slackFormatter struct{}

// NewSlackFormatter は Slack 用の Formatter を返します。
func NewSlackFormatter() Formatter {
	return &slackFormatter{}
}

func (f *slackFormatter) Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int, jaAbstract string) Message {
	paperLink := fmt.Sprintf("https://openreview.net/forum?id=%s", paper.ID)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("<%s|%s>", paperLink, headerText)

	main := formatMessage(paper, header, abstractBlock(paper.Content.Abstract.Value, jaAbstract, abstractMaxChars))

	var sub string
	if jaAbstract != "" {
		sub = fmt.Sprintf("*Original Abstract*:\n%s", truncateRunes(paper.Content.Abstract.Value, abstractMaxChars))
	}
	return Message{Main: main, Sub: sub}
}

// --- Helper Function ---

func truncateRunes(s string, max int) string {
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "..."
}

func abstractBlock(originalAbstract, jaAbstract string, abstractMaxChars int) string {
	if jaAbstract != "" {
		return fmt.Sprintf("*Abstract (日本語)*:\n%s", truncateRunes(jaAbstract, abstractMaxChars))
	}
	return fmt.Sprintf("*Abstract*:\n%s", truncateRunes(originalAbstract, abstractMaxChars))
}

func formatMessage(paper *openreview.Note, header, abstractBlock string) string {
	authors := strings.Join(paper.Content.Authors.Value, ", ")

	var pdfLine string
	if pdfPath := paper.Content.PDF.Value; pdfPath != "" {
		pdfURL := pdfPath
		if !strings.HasPrefix(pdfPath, "http") {
			pdfURL = "https://openreview.net" + pdfPath
		}
		pdfLine = fmt.Sprintf("\n\n*PDF*: %s", pdfURL)
	}

	return fmt.Sprintf(
		"%s\n\n*Title*: %s\n*Authors*: %s\n\n%s%s\n\nID: `%s`",
		header,
		paper.Content.Title.Value,
		authors,
		abstractBlock,
		pdfLine,
		paper.ID,
	)
}
