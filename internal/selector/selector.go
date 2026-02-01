package selector

// Paper は selector が要求する論文のインターフェースです。
// 外部の具体的な論文構造体（例: openreview.Note）は、このインターフェースを実装する必要があります。
type Paper interface {
	GetID() string
	GetTitle() string
}

// Selector は論文を選定するロジックのインターフェースです。
type Selector interface {
	Select(papers []Paper) (Paper, error)
}
