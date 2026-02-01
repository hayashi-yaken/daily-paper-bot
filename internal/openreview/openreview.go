package openreview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client はOpenReview APIと通信するためのクライアントです。
type Client struct {
	httpClient *http.Client
	BaseURL    string
	UserAgent  string
}

// NewClient は新しいOpenReviewクライアントを生成します。
func NewClient(userAgent string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:    "https://api2.openreview.net",
		UserAgent:  userAgent,
	}
}

// --- API Response Structures ---

// APIResponse は/notesエンドポイントのトップレベルのレスポンス構造です。
type APIResponse struct {
	Notes []Note `json:"notes"`
	Count int    `json:"count"`
}

// Note は論文一件の情報を保持するオブジェクトです。
type Note struct {
	ID      string      `json:"id"`
	CDate   int64       `json:"cdate"`
	Content NoteContent `json:"content"`
}

// NoteContent は論文の具体的な内容を保持します。
// 各フィールドは {"value": ...} という構造になっています。
type NoteContent struct {
	Title    ValueField[string]   `json:"title"`
	Authors  ValueField[[]string] `json:"authors"`
	Abstract ValueField[string]   `json:"abstract"`
	PDF      ValueField[string]   `json:"pdf,omitempty"`
	Bibtex   ValueField[string]   `json:"_bibtex,omitempty"`
}

// ValueField は {"value": T} の構造を表現するためのジェネリックな型です。
type ValueField[T any] struct {
	Value T `json:"value"`
}

// GetNotes は指定されたVenueの論文リストを取得します。
func (c *Client) GetNotes(venue string) ([]Note, error) {
	// APIエンドポイントを構築
	endpoint := fmt.Sprintf("%s/notes?invitation=%s/-/Submission", c.BaseURL, url.QueryEscape(venue))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return apiResponse.Notes, nil
}

// GetID はPaperインターフェースを満たすためにNoteのIDを返します。
func (n *Note) GetID() string {
	return n.ID
}

// GetTitle はPaperインターフェースを満たすためにNoteのタイトルを返します。
func (n *Note) GetTitle() string {
	return n.Content.Title.Value
}