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
	return Message{Main: formatMessage(paper, header, abstractMaxChars)}
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
	return Message{Main: formatMessage(paper, header, abstractMaxChars)}
}

// --- Helper Function ---

func truncateRunes(s string, max int) string {
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "..."
}

func formatMessage(paper *openreview.Note, header string, abstractMaxChars int) string {
	abstract := truncateRunes(paper.Content.Abstract.Value, abstractMaxChars)
	authors := strings.Join(paper.Content.Authors.Value, ", ")

	var link string
	pdfPath := paper.Content.PDF.Value
	if pdfPath != "" {
		if !strings.HasPrefix(pdfPath, "http") {
			link = "https://openreview.net" + pdfPath
		} else {
			link = pdfPath
		}
	} else {
		link = fmt.Sprintf("https://openreview.net/forum?id=%s", paper.ID)
	}

	return fmt.Sprintf(
		"%s\n\n*Title*: %s\n*Authors*: %s\n\n*Abstract*:\n%s\n\n*Link*:\n%s\n\nID: `%s`",
		header,
		paper.Content.Title.Value,
		authors,
		abstract,
		link,
		paper.ID,
	)
}
