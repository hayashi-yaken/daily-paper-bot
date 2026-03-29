# OpenReview Authentication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** GitHub Actions の IP ブロック回避のため OpenReview API v2 の JWT 認証を導入し、認証済みリクエストで論文データを取得できるようにする。

**Architecture:** `openreview.Client` に `token` フィールドと `Login()` メソッドを追加し、`GetNotes()` で自動的に `Authorization: Bearer <token>` ヘッダーを付与する。認証情報は環境変数 `OR_EMAIL` / `OR_PASSWORD` から読み込み、両方セットされている場合のみ `main.go` で `Login()` を呼び出す。

**Tech Stack:** Go 1.22, `net/http`, `net/http/httptest`（テスト用）, GitHub Actions Secrets

---

## File Map

| ファイル | 変更種別 | 内容 |
|---|---|---|
| `internal/config/config.go` | Modify | `OpenReviewEmail`, `OpenReviewPassword` フィールド追加 |
| `internal/config/config_test.go` | Modify | 認証情報の読み込みテスト追加 |
| `internal/openreview/openreview.go` | Modify | `token` フィールド・`Login()` 追加、`GetNotes()` ヘッダー付与 |
| `internal/openreview/openreview_test.go` | Modify | `Login()` と `GetNotes()` のユニットテスト追加 |
| `cmd/dailybot/main.go` | Modify | 認証ブロック追加 |
| `.github/workflows/daily.yml` | Modify | `OR_EMAIL`, `OR_PASSWORD` 環境変数追加 |

---

## Task 1: Config に認証情報フィールドを追加する

**Files:**
- Modify: `internal/config/config_test.go`
- Modify: `internal/config/config.go`

- [ ] **Step 1: テストを追加する**

`internal/config/config_test.go` の末尾に以下を追記する。

```go
func TestLoad_WithOpenReviewCredentials(t *testing.T) {
	t.Run("both credentials set", func(t *testing.T) {
		jsonContent := `[{"name":"ICLR","venue":"ICLR.cc/2025/Conference","year":2025}]`
		cleanup := setupTestConfigFile(t, jsonContent)
		defer cleanup()

		os.Setenv("TARGET_PLATFORM", "slack")
		os.Setenv("SLACK_BOT_TOKEN", "test_token")
		os.Setenv("SLACK_CHANNEL_ID", "test_channel")
		os.Setenv("OR_EMAIL", "user@example.com")
		os.Setenv("OR_PASSWORD", "secret")
		defer os.Unsetenv("TARGET_PLATFORM")
		defer os.Unsetenv("SLACK_BOT_TOKEN")
		defer os.Unsetenv("SLACK_CHANNEL_ID")
		defer os.Unsetenv("OR_EMAIL")
		defer os.Unsetenv("OR_PASSWORD")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}
		if cfg.OpenReviewEmail != "user@example.com" {
			t.Errorf("expected OpenReviewEmail 'user@example.com', got '%s'", cfg.OpenReviewEmail)
		}
		if cfg.OpenReviewPassword != "secret" {
			t.Errorf("expected OpenReviewPassword 'secret', got '%s'", cfg.OpenReviewPassword)
		}
	})

	t.Run("credentials not set", func(t *testing.T) {
		jsonContent := `[{"name":"ICLR","venue":"ICLR.cc/2025/Conference","year":2025}]`
		cleanup := setupTestConfigFile(t, jsonContent)
		defer cleanup()

		os.Setenv("TARGET_PLATFORM", "slack")
		os.Setenv("SLACK_BOT_TOKEN", "test_token")
		os.Setenv("SLACK_CHANNEL_ID", "test_channel")
		os.Unsetenv("OR_EMAIL")
		os.Unsetenv("OR_PASSWORD")
		defer os.Unsetenv("TARGET_PLATFORM")
		defer os.Unsetenv("SLACK_BOT_TOKEN")
		defer os.Unsetenv("SLACK_CHANNEL_ID")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}
		if cfg.OpenReviewEmail != "" {
			t.Errorf("expected empty OpenReviewEmail, got '%s'", cfg.OpenReviewEmail)
		}
		if cfg.OpenReviewPassword != "" {
			t.Errorf("expected empty OpenReviewPassword, got '%s'", cfg.OpenReviewPassword)
		}
	})
}
```

- [ ] **Step 2: テストが失敗することを確認する**

```bash
cd /Users/hayashinaofumi/mywork/hayashi-yaken/daily-paper-bot
go test ./internal/config/... -run TestLoad_WithOpenReviewCredentials -v
```

期待: `FAIL` — `cfg.OpenReviewEmail` フィールドが存在しないコンパイルエラー

- [ ] **Step 3: Config 構造体にフィールドを追加する**

`internal/config/config.go` の `Config` 構造体の末尾（`CustomUserAgent string` の下）に追加する。

```go
	// OpenReview Auth (optional)
	OpenReviewEmail    string
	OpenReviewPassword string
```

同ファイルの `Load()` 関数の末尾（`cfg.CustomUserAgent = ...` の下）に追加する。

```go
	cfg.OpenReviewEmail = os.Getenv("OR_EMAIL")
	cfg.OpenReviewPassword = os.Getenv("OR_PASSWORD")
```

- [ ] **Step 4: テストが通ることを確認する**

```bash
go test ./internal/config/... -run TestLoad_WithOpenReviewCredentials -v
```

期待: `PASS`

- [ ] **Step 5: 既存テストが壊れていないことを確認する**

```bash
go test ./internal/config/... -v
```

期待: 全テスト `PASS`

- [ ] **Step 6: コミットする**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add OpenReviewEmail and OpenReviewPassword to config"
```

---

## Task 2: openreview.Client に Login() メソッドを追加する

**Files:**
- Modify: `internal/openreview/openreview_test.go`
- Modify: `internal/openreview/openreview.go`

- [ ] **Step 1: テストを追加する**

`internal/openreview/openreview_test.go` の冒頭の import を以下に置き換える。

```go
import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)
```

ファイル末尾に以下を追記する。

```go
func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"token": "test-jwt-token", "user": {}}`)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL

	err := client.Login("user@example.com", "password")
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if client.token != "test-jwt-token" {
		t.Errorf("expected token 'test-jwt-token', got '%s'", client.token)
	}
}

func TestLogin_Failure_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL

	err := client.Login("user@example.com", "wrong-password")
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

```bash
go test ./internal/openreview/... -run "TestLogin" -v
```

期待: `FAIL` — `client.Login` が未定義のコンパイルエラー

- [ ] **Step 3: Client 構造体に token フィールドを追加する**

`internal/openreview/openreview.go` の `Client` 構造体を以下のように修正する。

```go
type Client struct {
	httpClient *http.Client
	BaseURL    string
	UserAgent  string
	token      string // 追加: 空文字 = 未認証
}
```

- [ ] **Step 4: Login() メソッドを実装する**

`internal/openreview/openreview.go` の import に `"bytes"` を追加する。

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)
```

同ファイルの末尾（`GetTitle()` の下）に以下を追加する。

```go
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
		return fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.token = loginResp.Token
	return nil
}
```

- [ ] **Step 5: テストが通ることを確認する**

```bash
go test ./internal/openreview/... -run "TestLogin" -v
```

期待: `PASS`

- [ ] **Step 6: コミットする**

```bash
git add internal/openreview/openreview.go internal/openreview/openreview_test.go
git commit -m "feat: add Login() method to openreview.Client"
```

---

## Task 3: GetNotes() に Authorization ヘッダーを付与する

**Files:**
- Modify: `internal/openreview/openreview_test.go`
- Modify: `internal/openreview/openreview.go`

- [ ] **Step 1: テストを追加する**

`internal/openreview/openreview_test.go` の末尾に以下を追記する。

```go
func TestGetNotes_WithAuthToken(t *testing.T) {
	var capturedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"notes": [], "count": 0}`)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL
	client.token = "test-jwt-token"

	_, err := client.GetNotes("TestVenue/Conference")
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if capturedAuthHeader != "Bearer test-jwt-token" {
		t.Errorf("expected Authorization header 'Bearer test-jwt-token', got '%s'", capturedAuthHeader)
	}
}

func TestGetNotes_WithoutAuthToken(t *testing.T) {
	var capturedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"notes": [], "count": 0}`)
	}))
	defer server.Close()

	client := NewClient("test-agent")
	client.BaseURL = server.URL
	// token は空のまま（デフォルト）

	_, err := client.GetNotes("TestVenue/Conference")
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if capturedAuthHeader != "" {
		t.Errorf("expected no Authorization header, got '%s'", capturedAuthHeader)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

```bash
go test ./internal/openreview/... -run "TestGetNotes_With" -v
```

期待: `FAIL` — `Authorization` ヘッダーが付与されていない

- [ ] **Step 3: GetNotes() にヘッダー付与ロジックを追加する**

`internal/openreview/openreview.go` の `GetNotes()` 内、`req.Header.Set("User-Agent", c.UserAgent)` の直後に以下を追加する。

```go
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
```

- [ ] **Step 4: テストが通ることを確認する**

```bash
go test ./internal/openreview/... -run "TestGetNotes_With" -v
```

期待: `PASS`

- [ ] **Step 5: 全テストが通ることを確認する**

```bash
go test ./internal/openreview/... -v
```

期待: 統合テスト以外の全テスト `PASS`（統合テストは CI 環境変数がないためスキップ）

- [ ] **Step 6: コミットする**

```bash
git add internal/openreview/openreview.go internal/openreview/openreview_test.go
git commit -m "feat: add Authorization header to GetNotes() when token is set"
```

---

## Task 4: main.go に認証ブロックを追加する

**Files:**
- Modify: `cmd/dailybot/main.go`

- [ ] **Step 1: 認証ブロックを追加する**

`cmd/dailybot/main.go` の `run()` 関数内、`orClient := openreview.NewClient(cfg.CustomUserAgent)` の直後に以下を追加する。

```go
	if cfg.OpenReviewEmail != "" && cfg.OpenReviewPassword != "" {
		if err := orClient.Login(cfg.OpenReviewEmail, cfg.OpenReviewPassword); err != nil {
			return fmt.Errorf("failed to login to openreview: %w", err)
		}
		log.Println("INFO: Authenticated to OpenReview.")
	}
```

- [ ] **Step 2: ビルドが通ることを確認する**

```bash
go build ./cmd/dailybot
```

期待: エラーなしでビルド完了

- [ ] **Step 3: コミットする**

```bash
git add cmd/dailybot/main.go
git commit -m "feat: add OpenReview authentication block to main.go"
```

---

## Task 5: GitHub Actions ワークフローに環境変数を追加する

**Files:**
- Modify: `.github/workflows/daily.yml`

- [ ] **Step 1: 環境変数を追加する**

`.github/workflows/daily.yml` の `env:` セクション（`CUSTOM_USER_AGENT: ...` の直後）に以下を追加する。

```yaml
          OR_EMAIL: ${{ secrets.OR_EMAIL }}
          OR_PASSWORD: ${{ secrets.OR_PASSWORD }}
```

- [ ] **Step 2: YAML の構文確認**

```bash
cat .github/workflows/daily.yml
```

インデントと構造が崩れていないことを目視確認する。

- [ ] **Step 3: 全テストが通ることを最終確認する**

```bash
go test ./... -v 2>&1 | grep -E "^(ok|FAIL|---)"
```

期待: `FAIL` が含まれないこと（統合テストは `SKIP` として表示される）

- [ ] **Step 4: コミットする**

```bash
git add .github/workflows/daily.yml
git commit -m "ci: add OR_EMAIL and OR_PASSWORD secrets to daily workflow"
```

---

## 事前準備（ユーザー作業・実装前に完了させること）

1. [https://openreview.net](https://openreview.net) でアカウントを作成する
2. GitHub リポジトリの **Settings > Secrets and variables > Actions** で以下を登録する
   - `OR_EMAIL`: 登録したメールアドレス
   - `OR_PASSWORD`: 登録したパスワード
