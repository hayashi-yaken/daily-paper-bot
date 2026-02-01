package formatter

import (
	"fmt"
	"strings"

	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
)

// FormatPaper は論文情報から投稿用のメッセージ文字列を生成します。
func FormatPaper(paper *openreview.Note, venue string, year int, abstractMaxChars int) string {
	// Abstractを指定文字数で切り詰める
	abstract := paper.Content.Abstract.Value
	if abstractMaxChars > 0 && len([]rune(abstract)) > abstractMaxChars {
		abstract = string([]rune(abstract)[:abstractMaxChars]) + "..."
	}

	// 著者リストをカンマ区切りの文字列にする
	authors := strings.Join(paper.Content.Authors.Value, ", ")

	// PDFのURLを取得する。なければOpenReviewのフォーラムURLを生成する。
	link := paper.Content.PDF.Value
	if link == "" {
		link = fmt.Sprintf("https://openreview.net/forum?id=%s", paper.ID)
	}

	// メッセージを組み立てる
	return fmt.Sprintf(
		"📄 今日の論文 (%s %d)\n\n*Title*: %s\n*Authors*: %s\n\n*Abstract*:\n%s\n\n*Link*:\n%s\n\nID: `%s`",
		venue,
		year,
		paper.Content.Title.Value,
		authors,
		abstract,
		link,
		paper.ID,
	)
}
