package notifier

// Notifier はメッセージを通知する責務を持つインターフェースです。
type Notifier interface {
	Post(message string) error
}
