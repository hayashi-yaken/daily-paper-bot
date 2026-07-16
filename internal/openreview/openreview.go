package openreview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client はOpenReview APIと通信するためのクライアントです。
type Client struct {
	httpClient *http.Client
	BaseURL    string
	UserAgent  string
	token      string // 追加: 空文字 = 未認証
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

// GetAcceptedNotes は指定されたVenueの採択論文をlimit/offset指定で取得します。
// content.venueidフィルタを使うため、採択済みの論文だけが対象になります。
// 返り値のcountはフィルタに一致する全体の論文数です（取得した件数ではありません）。
func (c *Client) GetAcceptedNotes(venue string, limit, offset int) ([]Note, int, error) {
	// APIエンドポイントを構築
	endpoint := fmt.Sprintf("%s/notes?content.venueid=%s&limit=%d&offset=%d",
		c.BaseURL, url.QueryEscape(venue), limit, offset)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response body: %w", err)
	}

	return apiResponse.Notes, apiResponse.Count, nil
}

// GetID はPaperインターフェースを満たすためにNoteのIDを返します。
func (n *Note) GetID() string {
	return n.ID
}

// GetTitle はPaperインターフェースを満たすためにNoteのタイトルを返します。
func (n *Note) GetTitle() string {
	return n.Content.Title.Value
}

// loginRequest は /login エンドポイントへのリクエストボディです。
type loginRequest struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

// loginResponse は /login エンドポイントのレスポンスボディです。
type loginResponse struct {
	Token string `json:"token"`
}

// Login は OpenReview API で認証し、取得したトークンをクライアントに保存します。
func (c *Client) Login(email, password string) error {
	payload, err := json.Marshal(loginRequest{ID: email, Password: password})
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/login", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("login failed with status code: %d, body: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	if loginResp.Token == "" {
		return fmt.Errorf("login succeeded but response contained empty token")
	}

	c.token = loginResp.Token
	return nil
}
