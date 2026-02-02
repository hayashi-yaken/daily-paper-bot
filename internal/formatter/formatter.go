package formatter

import (
	"fmt"
	"strings"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
)

// Formatter は論文情報を文字列に整形するインターフェースです。
type Formatter interface {
	Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int) string
}

// --- Discord Formatter (Standard Markdown) ---

type discordFormatter struct{}

// NewDiscordFormatter はDiscord用のFormatterを生成します。
func NewDiscordFormatter() Formatter {
	return &discordFormatter{}
}

func (f *discordFormatter) Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int) string {
	// ヘッダー部分を生成
	venueLink := fmt.Sprintf("https://openreview.net/group?id=%s", venue.Venue)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("[%s](%s)", headerText, venueLink)

	return formatMessage(paper, header, abstractMaxChars)
}

// --- Slack Formatter (Slack Mrkdwn) ---

type slackFormatter struct{}

// NewSlackFormatter はSlack用のFormatterを生成します。
func NewSlackFormatter() Formatter {
	return &slackFormatter{}
}

func (f *slackFormatter) Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int) string {
	// ヘッダー部分を生成
	venueLink := fmt.Sprintf("https://openreview.net/group?id=%s", venue.Venue)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("<%s|%s>", venueLink, headerText) // Slack形式のリンク

	return formatMessage(paper, header, abstractMaxChars)
}

// --- Helper Function ---

// formatMessage は共通のメッセージ本文を組み立てます。
func formatMessage(paper *openreview.Note, header string, abstractMaxChars int) string {
	// Abstractを指定文字数で切り詰める
	abstract := paper.Content.Abstract.Value
	if abstractMaxChars > 0 && len([]rune(abstract)) > abstractMaxChars {
		abstract = string([]rune(abstract)[:abstractMaxChars]) + "..."
	}

	// 著者リストをカンマ区切りの文字列にする
	authors := strings.Join(paper.Content.Authors.Value, ", ")

	// PDFのリンクを生成する
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

	// メッセージを組み立てる
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
