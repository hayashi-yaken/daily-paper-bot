package notifier

import "github.com/hayashi-yaken/daily-paper-bot/internal/formatter"

// Notifier はメッセージを通知する責務を持つインターフェースです。
type Notifier interface {
	Post(msg formatter.Message) error
}
