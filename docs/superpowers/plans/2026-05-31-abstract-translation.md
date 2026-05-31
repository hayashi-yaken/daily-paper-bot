# Abstract 翻訳表示（Azure AI Translator）実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** OpenReview の論文 Abstract を Azure AI Translator で日本語訳し、Slack ではスレッドに原文、Discord では spoiler で原文を併記する。あわせてヘッダリンクを論文ページに切り替え、本文末尾の Link 表現を `*PDF*:` に整理する。

**Architecture:** `internal/translator` パッケージを新設し Azure AI Translator v3.0 `/translate` を直接叩く。`formatter.Format` の戻り値を `Message{Main, Sub}` 構造体に変え、Slack/Discord それぞれが Sub の意味を解釈する。Notifier の `Post` シグネチャを `Post(formatter.Message)` に変更し、Slack 実装だけ親メッセージ投稿後の `thread_ts` を使った 2 段投稿に拡張する。

**Tech Stack:** Go 1.25 / 標準ライブラリ `net/http` のみ（Azure SDK 不使用） / `slack-go/slack` v0.17.3 / `httptest` でモック / TDD

**Spec:** [docs/superpowers/specs/2026-05-31-abstract-translation-design.md](../specs/2026-05-31-abstract-translation-design.md)

---

## File Structure

| ファイル | 種別 | 責務 |
|---|---|---|
| `internal/translator/translator.go` | 新規 | `Translator` インターフェースと `azureTranslator` 実装。Azure AI Translator v3.0 `/translate` の呼び出しに専念 |
| `internal/translator/translator_test.go` | 新規 | `httptest.NewServer` で正常系・異常系・空入力・JSON 異常を網羅 |
| `internal/config/config.go` | 修正 | `TranslateEnabled` / `AzureTranslatorEndpoint` / `AzureTranslatorRegion` / `AzureTranslatorKey` の追加とバリデーション |
| `internal/config/config_test.go` | 修正 | バリデーションテスト追加 |
| `internal/formatter/formatter.go` | 修正 | `Message{Main, Sub}` 型、ヘッダリンク差し替え、PDF 行条件出力、訳ありレイアウト |
| `internal/formatter/formatter_test.go` | 修正 | 訳あり/なし × Slack/Discord × PDF あり/なしの組合せで書き換え |
| `internal/notifier/notifier.go` | 修正 | `Post(formatter.Message) error` に変更 |
| `internal/notifier/slack.go` | 修正 | 親投稿後 `thread_ts` でスレッド子を投稿。子失敗は WARN 吸収 |
| `internal/notifier/slack_test.go` | 修正 | スレッド呼び出し回数の検証を追加 |
| `internal/notifier/discord.go` | 修正 | `Message` 受け取り（`Sub` は無視） |
| `internal/notifier/discord_test.go` | 修正 | 新シグネチャに合わせて更新 |
| `cmd/dailybot/main.go` | 修正 | 翻訳ステップ・DryRun 表示更新 |
| `.env.sample` | 修正 | 翻訳関連の環境変数追加 |
| `.github/workflows/daily.yml` | 修正 | Secret 参照を `env:` に追加 |
| `README.md` | 修正 | 翻訳機能の設定説明 |
| `GEMINI.md` | 修正 | 環境変数と挙動の追補 |
| `docs/tasks/v2/DPB-014.md` | 修正 | 完了チェックボックスとステータス更新（最終タスク） |

各ステップでは「失敗テスト → 実装 → グリーン → コミット」を 1 サイクルとする。コミットメッセージは Conventional Commits 形式（`feat:` `refactor:` `test:` `chore:` `docs:`）。

---

## Task 1: Config に翻訳関連フィールドとバリデーションを追加

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: 失敗テストを追加（`TRANSLATE_ENABLED=true` で Key 未設定 → エラー）**

`internal/config/config_test.go` の末尾に追加：

```go
func TestLoad_WithTranslator(t *testing.T) {
	jsonContent := `[{"name":"ICLR","venue":"ICLR.cc/2025/Conference","year":2025}]`

	setBasicEnv := func() {
		os.Setenv("TARGET_PLATFORM", "slack")
		os.Setenv("SLACK_BOT_TOKEN", "test_token")
		os.Setenv("SLACK_CHANNEL_ID", "test_channel")
	}
	unsetBasicEnv := func() {
		os.Unsetenv("TARGET_PLATFORM")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL_ID")
	}
	unsetTranslatorEnv := func() {
		os.Unsetenv("TRANSLATE_ENABLED")
		os.Unsetenv("AZURE_TRANSLATOR_KEY")
		os.Unsetenv("AZURE_TRANSLATOR_REGION")
		os.Unsetenv("AZURE_TRANSLATOR_ENDPOINT")
	}

	t.Run("enabled but key missing fails", func(t *testing.T) {
		cleanup := setupTestConfigFile(t, jsonContent)
		defer cleanup()
		setBasicEnv()
		defer unsetBasicEnv()
		os.Setenv("TRANSLATE_ENABLED", "true")
		os.Setenv("AZURE_TRANSLATOR_REGION", "japaneast")
		os.Unsetenv("AZURE_TRANSLATOR_KEY")
		defer unsetTranslatorEnv()

		if _, err := Load(); err == nil {
			t.Fatal("expected error when TRANSLATE_ENABLED=true but AZURE_TRANSLATOR_KEY is missing")
		}
	})

	t.Run("enabled but region missing fails", func(t *testing.T) {
		cleanup := setupTestConfigFile(t, jsonContent)
		defer cleanup()
		setBasicEnv()
		defer unsetBasicEnv()
		os.Setenv("TRANSLATE_ENABLED", "true")
		os.Setenv("AZURE_TRANSLATOR_KEY", "k")
		os.Unsetenv("AZURE_TRANSLATOR_REGION")
		defer unsetTranslatorEnv()

		if _, err := Load(); err == nil {
			t.Fatal("expected error when TRANSLATE_ENABLED=true but AZURE_TRANSLATOR_REGION is missing")
		}
	})

	t.Run("enabled with all required env succeeds", func(t *testing.T) {
		cleanup := setupTestConfigFile(t, jsonContent)
		defer cleanup()
		setBasicEnv()
		defer unsetBasicEnv()
		os.Setenv("TRANSLATE_ENABLED", "true")
		os.Setenv("AZURE_TRANSLATOR_KEY", "secret-key")
		os.Setenv("AZURE_TRANSLATOR_REGION", "japaneast")
		defer unsetTranslatorEnv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}
		if !cfg.TranslateEnabled {
			t.Errorf("expected TranslateEnabled=true")
		}
		if cfg.AzureTranslatorKey != "secret-key" {
			t.Errorf("expected key 'secret-key', got %q", cfg.AzureTranslatorKey)
		}
		if cfg.AzureTranslatorRegion != "japaneast" {
			t.Errorf("expected region 'japaneast', got %q", cfg.AzureTranslatorRegion)
		}
		if cfg.AzureTranslatorEndpoint != "https://api.cognitive.microsofttranslator.com" {
			t.Errorf("expected default endpoint, got %q", cfg.AzureTranslatorEndpoint)
		}
	})

	t.Run("disabled passes through without keys", func(t *testing.T) {
		cleanup := setupTestConfigFile(t, jsonContent)
		defer cleanup()
		setBasicEnv()
		defer unsetBasicEnv()
		unsetTranslatorEnv()
		defer unsetTranslatorEnv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}
		if cfg.TranslateEnabled {
			t.Errorf("expected TranslateEnabled=false")
		}
	})
}
```

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/config/... -run TestLoad_WithTranslator -v`
Expected: ビルドエラー（`cfg.TranslateEnabled` 等のフィールドが未定義）

- [ ] **Step 3: 実装する**

`internal/config/config.go` の `Config` 構造体に以下を追加：

```go
	// Translation
	TranslateEnabled        bool
	AzureTranslatorEndpoint string
	AzureTranslatorRegion   string
	AzureTranslatorKey      string
```

`Load()` の末尾、`cfg.OpenReviewPassword = os.Getenv("OR_PASSWORD")` の **直後** に以下を追記：

```go
	// Translation
	translateEnabledStr := os.Getenv("TRANSLATE_ENABLED")
	if translateEnabledStr == "" {
		cfg.TranslateEnabled = false
	} else {
		cfg.TranslateEnabled, err = strconv.ParseBool(translateEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TRANSLATE_ENABLED: %w", err)
		}
	}

	cfg.AzureTranslatorEndpoint = os.Getenv("AZURE_TRANSLATOR_ENDPOINT")
	if cfg.AzureTranslatorEndpoint == "" {
		cfg.AzureTranslatorEndpoint = "https://api.cognitive.microsofttranslator.com"
	}
	cfg.AzureTranslatorRegion = os.Getenv("AZURE_TRANSLATOR_REGION")
	cfg.AzureTranslatorKey = os.Getenv("AZURE_TRANSLATOR_KEY")

	if cfg.TranslateEnabled {
		if cfg.AzureTranslatorKey == "" || cfg.AzureTranslatorRegion == "" {
			return nil, fmt.Errorf("AZURE_TRANSLATOR_KEY and AZURE_TRANSLATOR_REGION are required when TRANSLATE_ENABLED=true")
		}
	}
```

- [ ] **Step 4: テストがグリーンになることを確認**

Run: `go test ./internal/config/... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add Azure Translator settings and validation"
```

---

## Task 2: Translator パッケージを新設

**Files:**
- Create: `internal/translator/translator.go`
- Create: `internal/translator/translator_test.go`

- [ ] **Step 1: テストファイルを作成（正常系、空入力、4xx/5xx、JSON 異常）**

`internal/translator/translator_test.go`:

```go
package translator

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAzureTranslator_Translate(t *testing.T) {
	t.Run("success returns translation", func(t *testing.T) {
		var receivedAuthKey, receivedRegion, receivedQuery string
		var receivedBody []byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuthKey = r.Header.Get("Ocp-Apim-Subscription-Key")
			receivedRegion = r.Header.Get("Ocp-Apim-Subscription-Region")
			receivedQuery = r.URL.RawQuery
			receivedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"translations":[{"text":"こんにちは","to":"ja"}]}]`))
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "secret-key")
		got, err := tr.Translate("hello", "ja")
		if err != nil {
			t.Fatalf("Translate failed: %v", err)
		}
		if got != "こんにちは" {
			t.Errorf("expected translation 'こんにちは', got %q", got)
		}
		if receivedAuthKey != "secret-key" {
			t.Errorf("expected Ocp-Apim-Subscription-Key=secret-key, got %q", receivedAuthKey)
		}
		if receivedRegion != "japaneast" {
			t.Errorf("expected Ocp-Apim-Subscription-Region=japaneast, got %q", receivedRegion)
		}
		if !strings.Contains(receivedQuery, "api-version=3.0") || !strings.Contains(receivedQuery, "to=ja") {
			t.Errorf("expected query to contain api-version=3.0 and to=ja, got %q", receivedQuery)
		}
		var bodyPayload []struct{ Text string }
		if err := json.Unmarshal(receivedBody, &bodyPayload); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if len(bodyPayload) != 1 || bodyPayload[0].Text != "hello" {
			t.Errorf("expected body [{Text:hello}], got %+v", bodyPayload)
		}
	})

	t.Run("empty text skips API call", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "k")
		got, err := tr.Translate("", "ja")
		if err != nil {
			t.Fatalf("Translate returned error for empty input: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty translation, got %q", got)
		}
		if callCount != 0 {
			t.Errorf("expected API not to be called, but got %d call(s)", callCount)
		}
	})

	t.Run("non-2xx returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":401000,"message":"invalid key"}}`))
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "k")
		_, err := tr.Translate("hello", "ja")
		if err == nil {
			t.Fatal("expected error for 401 response")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("expected error to mention status 401, got %v", err)
		}
	})

	t.Run("malformed response returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "k")
		_, err := tr.Translate("hello", "ja")
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("empty translations array returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`[]`))
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "k")
		_, err := tr.Translate("hello", "ja")
		if err == nil {
			t.Fatal("expected error for empty top-level array")
		}
	})
}
```

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/translator/... -v`
Expected: ビルドエラー（パッケージ未定義）

- [ ] **Step 3: 実装する**

`internal/translator/translator.go`:

```go
package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Translator は文字列を指定言語に翻訳します。
type Translator interface {
	Translate(text, targetLang string) (string, error)
}

type azureTranslator struct {
	httpClient *http.Client
	endpoint   string
	region     string
	key        string
}

// NewAzureTranslator は Azure AI Translator v3.0 を叩く Translator を返します。
func NewAzureTranslator(endpoint, region, key string) Translator {
	return &azureTranslator{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoint:   endpoint,
		region:     region,
		key:        key,
	}
}

type translateRequestItem struct {
	Text string `json:"Text"`
}

type translateResponseItem struct {
	Translations []struct {
		Text string `json:"text"`
		To   string `json:"to"`
	} `json:"translations"`
}

func (t *azureTranslator) Translate(text, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	payload, err := json.Marshal([]translateRequestItem{{Text: text}})
	if err != nil {
		return "", fmt.Errorf("failed to marshal translator request: %w", err)
	}

	q := url.Values{}
	q.Set("api-version", "3.0")
	q.Set("to", targetLang)
	reqURL := t.endpoint + "/translate?" + q.Encode()

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create translator request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Ocp-Apim-Subscription-Key", t.key)
	req.Header.Set("Ocp-Apim-Subscription-Region", t.region)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute translator request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read translator response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("translator returned status %d: %s", resp.StatusCode, string(body))
	}

	var items []translateResponseItem
	if err := json.Unmarshal(body, &items); err != nil {
		return "", fmt.Errorf("failed to decode translator response: %w", err)
	}
	if len(items) == 0 || len(items[0].Translations) == 0 || items[0].Translations[0].Text == "" {
		return "", fmt.Errorf("translator response missing translation: %s", string(body))
	}
	return items[0].Translations[0].Text, nil
}
```

- [ ] **Step 4: テストがグリーンになることを確認**

Run: `go test ./internal/translator/... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add internal/translator/
git commit -m "feat(translator): add Azure AI Translator v3.0 client"
```

---

## Task 3: Formatter に Message 型を導入し、シグネチャを変更

このタスクでは出力レイアウトは現行のまま保ち、戻り値の型と引数だけを変える。既存テストは新シグネチャに合わせて修正する。後続タスクでレイアウトを段階的に変えていく。

**Files:**
- Modify: `internal/formatter/formatter.go`
- Modify: `internal/formatter/formatter_test.go`

- [ ] **Step 1: 既存テストを新シグネチャに合わせて修正**

`internal/formatter/formatter_test.go` を以下に **置換**：

```go
package formatter

import (
	"strings"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
)

func TestFormatters_LegacyLink(t *testing.T) {
	paper := &openreview.Note{
		ID: "testID123",
		Content: openreview.NoteContent{
			Title:   openreview.ValueField[string]{Value: "Test Title"},
			Authors: openreview.ValueField[[]string]{Value: []string{"Author A", "Author B"}},
		},
	}
	venue := config.VenueConfig{
		Name:  "ICLR",
		Venue: "ICLR.cc/2025/Conference",
		Year:  2025,
	}

	t.Run("DiscordFormatter returns Message with empty Sub", func(t *testing.T) {
		formatter := NewDiscordFormatter()
		msg := formatter.Format(paper, venue, 100, "")
		expectedLink := "[📄 今日の論文 (ICLR 2025)](https://openreview.net/group?id=ICLR.cc/2025/Conference)"
		if !strings.Contains(msg.Main, expectedLink) {
			t.Errorf("Discord link format is incorrect.\nGot: %s\nExpected to contain: %s", msg.Main, expectedLink)
		}
		if msg.Sub != "" {
			t.Errorf("Discord Sub should always be empty, got %q", msg.Sub)
		}
	})

	t.Run("SlackFormatter returns Message with empty Sub when no translation", func(t *testing.T) {
		formatter := NewSlackFormatter()
		msg := formatter.Format(paper, venue, 100, "")
		expectedLink := "<https://openreview.net/group?id=ICLR.cc/2025/Conference|📄 今日の論文 (ICLR 2025)>"
		if !strings.Contains(msg.Main, expectedLink) {
			t.Errorf("Slack link format is incorrect.\nGot: %s\nExpected to contain: %s", msg.Main, expectedLink)
		}
		if msg.Sub != "" {
			t.Errorf("Slack Sub should be empty when jaAbstract is empty, got %q", msg.Sub)
		}
	})
}
```

注: ここではヘッダリンクの URL を旧仕様（`group?id=...`）で書く。後続 Task 4 でリンク仕様を変更するとともにテスト期待値も更新する。

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/formatter/... -v`
Expected: ビルドエラー（`Format` の引数不一致、`Message` 未定義）

- [ ] **Step 3: 実装する**

`internal/formatter/formatter.go` を以下に **置換**：

```go
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
	venueLink := fmt.Sprintf("https://openreview.net/group?id=%s", venue.Venue)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("[%s](%s)", headerText, venueLink)
	return Message{Main: formatMessage(paper, header, abstractMaxChars)}
}

// --- Slack Formatter (Slack Mrkdwn) ---

type slackFormatter struct{}

// NewSlackFormatter は Slack 用の Formatter を返します。
func NewSlackFormatter() Formatter {
	return &slackFormatter{}
}

func (f *slackFormatter) Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int, jaAbstract string) Message {
	venueLink := fmt.Sprintf("https://openreview.net/group?id=%s", venue.Venue)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("<%s|%s>", venueLink, headerText)
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
```

- [ ] **Step 4: テストがグリーンになることを確認**

Run: `go test ./internal/formatter/... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add internal/formatter/
git commit -m "refactor(formatter): return Message{Main, Sub} from Format"
```

---

## Task 4: ヘッダリンクを論文 forum URL に変更

**Files:**
- Modify: `internal/formatter/formatter.go`
- Modify: `internal/formatter/formatter_test.go`

- [ ] **Step 1: 失敗テストを追加（ヘッダリンクが `forum?id={paperID}` を含む）**

`internal/formatter/formatter_test.go` に追加：

```go
func TestFormatters_HeaderLinkPointsToPaper(t *testing.T) {
	paper := &openreview.Note{
		ID: "ABC123",
		Content: openreview.NoteContent{
			Title:   openreview.ValueField[string]{Value: "T"},
			Authors: openreview.ValueField[[]string]{Value: []string{"A"}},
		},
	}
	venue := config.VenueConfig{Name: "ICLR", Venue: "ICLR.cc/2025/Conference", Year: 2025}

	t.Run("Slack header links to forum page", func(t *testing.T) {
		msg := NewSlackFormatter().Format(paper, venue, 100, "")
		wantLink := "<https://openreview.net/forum?id=ABC123|📄 今日の論文 (ICLR 2025)>"
		if !strings.Contains(msg.Main, wantLink) {
			t.Errorf("Slack header link wrong.\nGot: %s\nWant contains: %s", msg.Main, wantLink)
		}
	})

	t.Run("Discord header links to forum page", func(t *testing.T) {
		msg := NewDiscordFormatter().Format(paper, venue, 100, "")
		wantLink := "[📄 今日の論文 (ICLR 2025)](https://openreview.net/forum?id=ABC123)"
		if !strings.Contains(msg.Main, wantLink) {
			t.Errorf("Discord header link wrong.\nGot: %s\nWant contains: %s", msg.Main, wantLink)
		}
	})
}
```

そして既存の `TestFormatters_LegacyLink` を **削除**する（リンク仕様が変わるため Legacy テストの期待値は無効になる）。

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/formatter/... -run TestFormatters_HeaderLinkPointsToPaper -v`
Expected: FAIL（古い `group?id=...` がまだ使われている）

- [ ] **Step 3: 実装する**

`internal/formatter/formatter.go` の Slack/Discord 両方の `Format` メソッドを修正。`venueLink := ...` を以下に置き換える：

Discord 側：

```go
	paperLink := fmt.Sprintf("https://openreview.net/forum?id=%s", paper.ID)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("[%s](%s)", headerText, paperLink)
```

Slack 側：

```go
	paperLink := fmt.Sprintf("https://openreview.net/forum?id=%s", paper.ID)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("<%s|%s>", paperLink, headerText)
```

- [ ] **Step 4: テストがグリーンになることを確認**

Run: `go test ./internal/formatter/... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add internal/formatter/
git commit -m "feat(formatter): point header link to paper forum page"
```

---

## Task 5: 本文末尾の Link 行を PDF 専用に整理

PDF が取得できているときのみ `*PDF*: {pdfURL}` を末尾に出し、無いときは行ごと省略する。

**Files:**
- Modify: `internal/formatter/formatter.go`
- Modify: `internal/formatter/formatter_test.go`

- [ ] **Step 1: 失敗テストを追加**

`internal/formatter/formatter_test.go` に追加：

```go
func TestFormatters_PDFLine(t *testing.T) {
	venue := config.VenueConfig{Name: "ICLR", Venue: "ICLR.cc/2025/Conference", Year: 2025}

	t.Run("Slack with PDF shows *PDF* line", func(t *testing.T) {
		paper := &openreview.Note{
			ID: "PID",
			Content: openreview.NoteContent{
				Title:   openreview.ValueField[string]{Value: "T"},
				Authors: openreview.ValueField[[]string]{Value: []string{"A"}},
				PDF:     openreview.ValueField[string]{Value: "/pdf?id=PID"},
			},
		}
		msg := NewSlackFormatter().Format(paper, venue, 100, "")
		if !strings.Contains(msg.Main, "*PDF*: https://openreview.net/pdf?id=PID") {
			t.Errorf("expected Slack output to contain '*PDF*: ...'.\nGot: %s", msg.Main)
		}
		if strings.Contains(msg.Main, "*Link*:") {
			t.Errorf("expected Slack output not to contain legacy '*Link*:'.\nGot: %s", msg.Main)
		}
	})

	t.Run("Slack without PDF omits link line", func(t *testing.T) {
		paper := &openreview.Note{
			ID: "PID",
			Content: openreview.NoteContent{
				Title:   openreview.ValueField[string]{Value: "T"},
				Authors: openreview.ValueField[[]string]{Value: []string{"A"}},
			},
		}
		msg := NewSlackFormatter().Format(paper, venue, 100, "")
		if strings.Contains(msg.Main, "*PDF*:") {
			t.Errorf("expected no *PDF*: line when PDF is missing.\nGot: %s", msg.Main)
		}
		if strings.Contains(msg.Main, "*Link*:") {
			t.Errorf("expected no legacy *Link*: line.\nGot: %s", msg.Main)
		}
	})

	t.Run("Discord with PDF shows *PDF* line", func(t *testing.T) {
		paper := &openreview.Note{
			ID: "PID",
			Content: openreview.NoteContent{
				Title:   openreview.ValueField[string]{Value: "T"},
				Authors: openreview.ValueField[[]string]{Value: []string{"A"}},
				PDF:     openreview.ValueField[string]{Value: "/pdf?id=PID"},
			},
		}
		msg := NewDiscordFormatter().Format(paper, venue, 100, "")
		if !strings.Contains(msg.Main, "*PDF*: https://openreview.net/pdf?id=PID") {
			t.Errorf("expected Discord output to contain '*PDF*: ...'.\nGot: %s", msg.Main)
		}
	})
}
```

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/formatter/... -run TestFormatters_PDFLine -v`
Expected: FAIL

- [ ] **Step 3: 実装する**

`internal/formatter/formatter.go` の `formatMessage` を以下に **置換**：

```go
func formatMessage(paper *openreview.Note, header string, abstractMaxChars int) string {
	abstract := truncateRunes(paper.Content.Abstract.Value, abstractMaxChars)
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
		"%s\n\n*Title*: %s\n*Authors*: %s\n\n*Abstract*:\n%s%s\n\nID: `%s`",
		header,
		paper.Content.Title.Value,
		authors,
		abstract,
		pdfLine,
		paper.ID,
	)
}
```

- [ ] **Step 4: テストがグリーンになることを確認**

Run: `go test ./internal/formatter/... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add internal/formatter/
git commit -m "feat(formatter): replace *Link*: row with conditional *PDF*:"
```

---

## Task 6: Slack 訳ありレイアウト（親=日本語、Sub=原文）

**Files:**
- Modify: `internal/formatter/formatter.go`
- Modify: `internal/formatter/formatter_test.go`

- [ ] **Step 1: 失敗テストを追加**

`internal/formatter/formatter_test.go` に追加：

```go
func TestSlackFormatter_WithTranslation(t *testing.T) {
	paper := &openreview.Note{
		ID: "PID",
		Content: openreview.NoteContent{
			Title:    openreview.ValueField[string]{Value: "T"},
			Authors:  openreview.ValueField[[]string]{Value: []string{"A"}},
			Abstract: openreview.ValueField[string]{Value: "english abstract"},
		},
	}
	venue := config.VenueConfig{Name: "ICLR", Venue: "ICLR.cc/2025/Conference", Year: 2025}

	msg := NewSlackFormatter().Format(paper, venue, 100, "日本語訳テスト")

	if !strings.Contains(msg.Main, "*Abstract (日本語)*:\n日本語訳テスト") {
		t.Errorf("expected Main to contain Japanese abstract heading.\nGot: %s", msg.Main)
	}
	if strings.Contains(msg.Main, "english abstract") {
		t.Errorf("expected Main not to contain original abstract when translated.\nGot: %s", msg.Main)
	}
	if !strings.Contains(msg.Sub, "*Original Abstract*:\nenglish abstract") {
		t.Errorf("expected Sub to contain original abstract block.\nGot: %s", msg.Sub)
	}
}

func TestSlackFormatter_WithoutTranslation_LegacyHeading(t *testing.T) {
	paper := &openreview.Note{
		ID: "PID",
		Content: openreview.NoteContent{
			Title:    openreview.ValueField[string]{Value: "T"},
			Authors:  openreview.ValueField[[]string]{Value: []string{"A"}},
			Abstract: openreview.ValueField[string]{Value: "english abstract"},
		},
	}
	venue := config.VenueConfig{Name: "ICLR", Venue: "ICLR.cc/2025/Conference", Year: 2025}

	msg := NewSlackFormatter().Format(paper, venue, 100, "")

	if !strings.Contains(msg.Main, "*Abstract*:\nenglish abstract") {
		t.Errorf("expected Main to show original abstract under *Abstract*: when no translation.\nGot: %s", msg.Main)
	}
	if msg.Sub != "" {
		t.Errorf("expected empty Sub when no translation, got %q", msg.Sub)
	}
}
```

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/formatter/... -run "TestSlackFormatter_With" -v`
Expected: FAIL（日本語見出しが未実装）

- [ ] **Step 3: 実装する**

`internal/formatter/formatter.go` の構造を以下のように整理する。`formatMessage` は本文の共通部（タイトル・著者・PDF・ID）に絞り、アブストラクト見出しは呼び出し側で組み立てる。

`formatMessage` を以下に **置換**：

```go
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

func abstractBlock(originalAbstract, jaAbstract string, abstractMaxChars int) string {
	if jaAbstract != "" {
		return fmt.Sprintf("*Abstract (日本語)*:\n%s", truncateRunes(jaAbstract, abstractMaxChars))
	}
	return fmt.Sprintf("*Abstract*:\n%s", truncateRunes(originalAbstract, abstractMaxChars))
}
```

Slack 側 `Format` を以下に **置換**：

```go
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
```

Discord 側 `Format` は次の Task 7 で訳に対応するため、ここでは `formatMessage` の新シグネチャに合わせるだけ：

```go
func (f *discordFormatter) Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int, jaAbstract string) Message {
	paperLink := fmt.Sprintf("https://openreview.net/forum?id=%s", paper.ID)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("[%s](%s)", headerText, paperLink)

	main := formatMessage(paper, header, abstractBlock(paper.Content.Abstract.Value, jaAbstract, abstractMaxChars))
	return Message{Main: main}
}
```

- [ ] **Step 4: テストがグリーンになることを確認**

Run: `go test ./internal/formatter/... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add internal/formatter/
git commit -m "feat(formatter): slack uses Japanese abstract in main and original in Sub"
```

---

## Task 7: Discord 訳ありレイアウト（spoiler で原文同梱）

**Files:**
- Modify: `internal/formatter/formatter.go`
- Modify: `internal/formatter/formatter_test.go`

- [ ] **Step 1: 失敗テストを追加**

`internal/formatter/formatter_test.go` に追加：

```go
func TestDiscordFormatter_WithTranslation(t *testing.T) {
	paper := &openreview.Note{
		ID: "PID",
		Content: openreview.NoteContent{
			Title:    openreview.ValueField[string]{Value: "T"},
			Authors:  openreview.ValueField[[]string]{Value: []string{"A"}},
			Abstract: openreview.ValueField[string]{Value: "english abstract"},
		},
	}
	venue := config.VenueConfig{Name: "ICLR", Venue: "ICLR.cc/2025/Conference", Year: 2025}

	msg := NewDiscordFormatter().Format(paper, venue, 100, "日本語訳テスト")

	if !strings.Contains(msg.Main, "*Abstract (日本語)*:\n日本語訳テスト") {
		t.Errorf("expected Main to contain Japanese abstract heading.\nGot: %s", msg.Main)
	}
	if !strings.Contains(msg.Main, "*Original Abstract*:\n||english abstract||") {
		t.Errorf("expected Main to contain spoilered original abstract.\nGot: %s", msg.Main)
	}
	if msg.Sub != "" {
		t.Errorf("Discord Sub must remain empty, got %q", msg.Sub)
	}
}
```

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/formatter/... -run TestDiscordFormatter_WithTranslation -v`
Expected: FAIL（spoiler ブロックがまだ無い）

- [ ] **Step 3: 実装する**

`internal/formatter/formatter.go` の Discord 側 `Format` を以下に **置換**：

```go
func (f *discordFormatter) Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int, jaAbstract string) Message {
	paperLink := fmt.Sprintf("https://openreview.net/forum?id=%s", paper.ID)
	headerText := fmt.Sprintf("📄 今日の論文 (%s %d)", venue.Name, venue.Year)
	header := fmt.Sprintf("[%s](%s)", headerText, paperLink)

	abs := abstractBlock(paper.Content.Abstract.Value, jaAbstract, abstractMaxChars)
	if jaAbstract != "" {
		abs += fmt.Sprintf("\n\n*Original Abstract*:\n||%s||", truncateRunes(paper.Content.Abstract.Value, abstractMaxChars))
	}
	return Message{Main: formatMessage(paper, header, abs)}
}
```

- [ ] **Step 4: テストがグリーンになることを確認**

Run: `go test ./internal/formatter/... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add internal/formatter/
git commit -m "feat(formatter): discord inlines original abstract as spoiler when translated"
```

---

## Task 8: Notifier の Post シグネチャを `Post(formatter.Message)` に変更

このタスクではインターフェース変更と Discord/Slack の最小対応のみ行う。Slack のスレッド対応は Task 9 に分離する。

**Files:**
- Modify: `internal/notifier/notifier.go`
- Modify: `internal/notifier/slack.go`
- Modify: `internal/notifier/discord.go`
- Modify: `internal/notifier/slack_test.go`
- Modify: `internal/notifier/discord_test.go`

- [ ] **Step 1: テストを新シグネチャに合わせて修正**

`internal/notifier/discord_test.go` を以下に **置換**：

```go
package notifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/formatter"
)

func TestDiscordNotifier_Post(t *testing.T) {
	t.Run("post success sends Main as content", func(t *testing.T) {
		var receivedBody []byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		notifier := NewDiscordNotifier(server.URL)
		err := notifier.Post(formatter.Message{Main: "hello", Sub: "ignored"})
		if err != nil {
			t.Errorf("Post() should not return an error, but got: %v", err)
		}

		var payload struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(receivedBody, &payload); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if payload.Content != "hello" {
			t.Errorf("expected content 'hello', got %q", payload.Content)
		}
	})

	t.Run("post failure due to server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		notifier := NewDiscordNotifier(server.URL)
		err := notifier.Post(formatter.Message{Main: "test"})
		if err == nil {
			t.Error("Post() should return an error for non-2xx status, but got nil")
		}
	})

	t.Run("post failure due to invalid url", func(t *testing.T) {
		notifier := NewDiscordNotifier("http://localhost:99999")
		err := notifier.Post(formatter.Message{Main: "test"})
		if err == nil {
			t.Error("Post() should return an error for invalid URL, but got nil")
		}
	})
}
```

`internal/notifier/slack_test.go` を以下に **置換**：

```go
package notifier

import (
	"errors"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/formatter"
	"github.com/slack-go/slack"
)

type mockAPIPoster struct {
	shouldFail bool
	calls      []struct {
		channelID string
		options   []slack.MsgOption
	}
}

func (m *mockAPIPoster) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	m.calls = append(m.calls, struct {
		channelID string
		options   []slack.MsgOption
	}{channelID, options})
	if m.shouldFail {
		return "", "", errors.New("mock post error")
	}
	return channelID, "12345.67890", nil
}

func TestSlackNotifier_Post(t *testing.T) {
	t.Run("post success without Sub calls PostMessage once", func(t *testing.T) {
		mock := &mockAPIPoster{}
		notifier := &SlackNotifier{poster: mock, channelID: "C12345"}

		if err := notifier.Post(formatter.Message{Main: "hello"}); err != nil {
			t.Fatalf("Post returned error: %v", err)
		}
		if len(mock.calls) != 1 {
			t.Errorf("expected 1 PostMessage call, got %d", len(mock.calls))
		}
	})

	t.Run("parent post failure returns error", func(t *testing.T) {
		mock := &mockAPIPoster{shouldFail: true}
		notifier := &SlackNotifier{poster: mock, channelID: "C12345"}

		if err := notifier.Post(formatter.Message{Main: "hello"}); err == nil {
			t.Error("expected error when parent post fails")
		}
	})
}
```

注: スレッド呼び出し回数を検証するテストは Task 9 で追加する。

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/notifier/... -v`
Expected: ビルドエラー（`Post(string)` のまま）

- [ ] **Step 3: インターフェースを変更**

`internal/notifier/notifier.go` を以下に **置換**：

```go
package notifier

import "github.com/hayashi-yaken/daily-paper-bot/internal/formatter"

// Notifier はメッセージを通知する責務を持つインターフェースです。
type Notifier interface {
	Post(msg formatter.Message) error
}
```

`internal/notifier/discord.go` の `Post` を以下に **置換**：

```go
func (n *DiscordNotifier) Post(msg formatter.Message) error {
	payload := discordPayload{Content: msg.Main}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post message to discord: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned non-2xx status: %d", resp.StatusCode)
	}
	return nil
}
```

import に `"github.com/hayashi-yaken/daily-paper-bot/internal/formatter"` を追加。

`internal/notifier/slack.go` の `Post` を以下に **置換**：

```go
func (n *SlackNotifier) Post(msg formatter.Message) error {
	_, _, err := n.poster.PostMessage(
		n.channelID,
		slack.MsgOptionText(msg.Main, false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return fmt.Errorf("failed to post message to slack: %w", err)
	}
	return nil
}
```

import に `"github.com/hayashi-yaken/daily-paper-bot/internal/formatter"` を追加。

- [ ] **Step 4: cmd/dailybot/main.go の `paperNotifier.Post(message)` 呼び出しは次の Task 10 で更新する。先にビルドだけ通すため、main.go の `message` 変数の使い方を一時的に直す**

`cmd/dailybot/main.go` の以下の行を：

```go
message := paperFormatter.Format(selectedNote, selectedVenue, cfg.AbstractMaxChars)
```

以下に変更（Step 4 ではビルドを通すだけ、翻訳ロジックの追加は Task 10）：

```go
message := paperFormatter.Format(selectedNote, selectedVenue, cfg.AbstractMaxChars, "")
```

そして `paperNotifier.Post(message)` はそのままで OK（`message` がすでに `formatter.Message` 型になっている）。

DryRun の `log.Printf("--- Message to be posted ---\n%s\n--------------------------", message)` 行を以下に置換：

```go
log.Printf("--- Message to be posted ---\n%s\n--------------------------", message.Main)
```

- [ ] **Step 5: 全テストがグリーンになり、ビルドも通ることを確認**

Run: `go build ./... && go test ./... -v`
Expected: 全テスト PASS

- [ ] **Step 6: コミット**

```bash
git add internal/notifier/ cmd/dailybot/main.go
git commit -m "refactor(notifier): change Post signature to accept formatter.Message"
```

---

## Task 9: Slack で Sub をスレッド子として投稿

**Files:**
- Modify: `internal/notifier/slack.go`
- Modify: `internal/notifier/slack_test.go`

- [ ] **Step 1: 失敗テストを追加**

`internal/notifier/slack_test.go` の `TestSlackNotifier_Post` に追加（既存の 2 ケースの下）：

```go
	t.Run("post with Sub triggers thread reply", func(t *testing.T) {
		mock := &mockAPIPoster{}
		notifier := &SlackNotifier{poster: mock, channelID: "C12345"}

		err := notifier.Post(formatter.Message{Main: "main text", Sub: "thread text"})
		if err != nil {
			t.Fatalf("Post returned error: %v", err)
		}
		if len(mock.calls) != 2 {
			t.Fatalf("expected 2 PostMessage calls (parent + thread), got %d", len(mock.calls))
		}
		// 親も子も同一チャンネル
		if mock.calls[0].channelID != "C12345" || mock.calls[1].channelID != "C12345" {
			t.Errorf("expected both posts to channel C12345, got %q and %q", mock.calls[0].channelID, mock.calls[1].channelID)
		}
	})

	t.Run("thread post failure does not fail Post", func(t *testing.T) {
		mock := &flakeyPoster{failAfter: 1} // 親は成功、子は失敗
		notifier := &SlackNotifier{poster: mock, channelID: "C12345"}

		err := notifier.Post(formatter.Message{Main: "main text", Sub: "thread text"})
		if err != nil {
			t.Errorf("Post should not return error when only thread reply fails, got: %v", err)
		}
		if mock.callCount != 2 {
			t.Errorf("expected 2 PostMessage attempts, got %d", mock.callCount)
		}
	})
}

type flakeyPoster struct {
	failAfter int // この回数を超えた呼び出しで失敗を返す
	callCount int
}

func (f *flakeyPoster) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	f.callCount++
	if f.callCount > f.failAfter {
		return "", "", errors.New("mock thread failure")
	}
	return channelID, "12345.67890", nil
}
```

- [ ] **Step 2: テスト失敗を確認**

Run: `go test ./internal/notifier/... -run TestSlackNotifier_Post -v`
Expected: FAIL（スレッド呼び出しが行われていない）

- [ ] **Step 3: 実装する**

`internal/notifier/slack.go` の `Post` を以下に **置換**：

```go
func (n *SlackNotifier) Post(msg formatter.Message) error {
	_, parentTS, err := n.poster.PostMessage(
		n.channelID,
		slack.MsgOptionText(msg.Main, false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return fmt.Errorf("failed to post message to slack: %w", err)
	}

	if msg.Sub == "" {
		return nil
	}

	if _, _, threadErr := n.poster.PostMessage(
		n.channelID,
		slack.MsgOptionText(msg.Sub, false),
		slack.MsgOptionAsUser(true),
		slack.MsgOptionTS(parentTS),
	); threadErr != nil {
		log.Printf("WARN: failed to post thread reply to slack (parent succeeded): %v", threadErr)
	}
	return nil
}
```

import に `"log"` を追加。

- [ ] **Step 4: 全テストがグリーンになることを確認**

Run: `go test ./internal/notifier/... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add internal/notifier/slack.go internal/notifier/slack_test.go
git commit -m "feat(notifier/slack): post Sub as a threaded reply"
```

---

## Task 10: main.go に翻訳ステップを追加し DryRun 表示を更新

**Files:**
- Modify: `cmd/dailybot/main.go`

- [ ] **Step 1: 翻訳ステップを追加**

`cmd/dailybot/main.go` の import に追加：

```go
	"github.com/hayashi-yaken/daily-paper-bot/internal/translator"
```

論文選定後の `// 6. 投稿メッセージを生成` の直前に以下を挿入：

```go
	// 5.5. アブストラクトの翻訳（任意）
	var jaAbstract string
	if cfg.TranslateEnabled {
		tr := translator.NewAzureTranslator(
			cfg.AzureTranslatorEndpoint,
			cfg.AzureTranslatorRegion,
			cfg.AzureTranslatorKey,
		)
		translated, err := tr.Translate(selectedNote.Content.Abstract.Value, "ja")
		if err != nil {
			log.Printf("WARN: translation failed, falling back to original abstract only: %v", err)
		} else {
			jaAbstract = translated
			log.Printf("INFO: Translated abstract (len=%d chars).", len([]rune(jaAbstract)))
		}
	} else {
		log.Println("INFO: Translation disabled.")
	}
```

- [ ] **Step 2: Format 呼び出しに `jaAbstract` を渡す**

`cmd/dailybot/main.go` の以下の行：

```go
	message := paperFormatter.Format(selectedNote, selectedVenue, cfg.AbstractMaxChars, "")
```

を以下に置換：

```go
	message := paperFormatter.Format(selectedNote, selectedVenue, cfg.AbstractMaxChars, jaAbstract)
```

- [ ] **Step 3: DryRun 表示を Main/Sub に対応**

`cmd/dailybot/main.go` の DryRun ブロックを以下に **置換**：

```go
	if cfg.DryRun {
		log.Println("INFO: Dry run mode is enabled. Skipping post.")
		log.Printf("--- Main ---\n%s\n------------", message.Main)
		if message.Sub != "" {
			log.Printf("--- Sub (thread) ---\n%s\n--------------------", message.Sub)
		}
		return nil
	}
```

- [ ] **Step 4: ビルドと全テストを実行**

Run: `go build ./... && go test ./... -v`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add cmd/dailybot/main.go
git commit -m "feat(cmd): translate abstract and pass it through formatter/notifier"
```

---

## Task 11: 環境変数サンプルと GitHub Actions 設定の追加

**Files:**
- Modify: `.env.sample`
- Modify: `.github/workflows/daily.yml`

- [ ] **Step 1: `.env.sample` に翻訳関連の環境変数を追加**

`.env.sample` の末尾に以下を追記：

```
# --- Azure AI Translator (任意, abstract の日本語訳) ---
# TRANSLATE_ENABLED="false"
# AZURE_TRANSLATOR_KEY=""
# AZURE_TRANSLATOR_REGION="japaneast"
# 通常はデフォルトのままで OK
# AZURE_TRANSLATOR_ENDPOINT="https://api.cognitive.microsofttranslator.com"
```

- [ ] **Step 2: `.github/workflows/daily.yml` の `env:` に翻訳関連の Secret を追加**

`OR_PASSWORD: ${{ secrets.OR_PASSWORD }}` の **直後** に以下を追記：

```yaml
          TRANSLATE_ENABLED: ${{ secrets.TRANSLATE_ENABLED }}
          AZURE_TRANSLATOR_KEY: ${{ secrets.AZURE_TRANSLATOR_KEY }}
          AZURE_TRANSLATOR_REGION: ${{ secrets.AZURE_TRANSLATOR_REGION }}
```

- [ ] **Step 3: ビルドと全テストを実行（変更が他に波及していないか）**

Run: `go build ./... && go test ./... -v`
Expected: 全テスト PASS

- [ ] **Step 4: コミット**

```bash
git add .env.sample .github/workflows/daily.yml
git commit -m "chore: wire Azure Translator env vars in sample and GitHub Actions"
```

---

## Task 12: README と GEMINI.md の追補

**Files:**
- Modify: `README.md`
- Modify: `GEMINI.md`

- [ ] **Step 1: README.md に翻訳機能の説明を追加**

`README.md` の「主な機能」リストの末尾に以下を追加：

```
- (任意) Azure AI Translator を用いた Abstract の日本語訳表示
  - Slack: 親メッセージに訳、原文はスレッド返信
  - Discord: 親メッセージに訳と spoiler 化した原文を同梱
```

`README.md` の「環境変数の設定」セクションの最低限項目リストの直後に、以下のサブセクションを追加：

```
#### Azure AI Translator（任意）

Abstract を日本語訳して投稿に含めたい場合は、Azure ポータルで Translator リソースを作成し、以下の環境変数を設定します。

- `TRANSLATE_ENABLED`: `"true"` で機能を有効化（デフォルト `"false"`）
- `AZURE_TRANSLATOR_KEY`: Translator リソースのサブスクリプションキー
- `AZURE_TRANSLATOR_REGION`: リソースのリージョン（例: `japaneast`）
- `AZURE_TRANSLATOR_ENDPOINT`: 任意。デフォルト `https://api.cognitive.microsofttranslator.com`

翻訳 API が失敗した場合は WARN ログを出して原文だけで投稿を続行します（投稿はスキップしません）。
```

- [ ] **Step 2: GEMINI.md の環境変数リストに追加**

`GEMINI.md` の「5.2. 環境変数」リストの末尾に以下を追加：

```
- **`TRANSLATE_ENABLED`**: (任意) `true` で Azure AI Translator による日本語訳を有効化。デフォルト `false`。
- **`AZURE_TRANSLATOR_KEY`**: (Secret, `TRANSLATE_ENABLED=true` のとき必須) Translator のサブスクリプションキー。
- **`AZURE_TRANSLATOR_REGION`**: (Secret, `TRANSLATE_ENABLED=true` のとき必須) Translator リソースのリージョン (例: `japaneast`)。
- **`AZURE_TRANSLATOR_ENDPOINT`**: (任意) Translator エンドポイント。通常はデフォルトで OK。
```

`GEMINI.md` の「3. プロジェクト構造」の `internal/` 配下のリストに以下を追加：

```
  - `translator/`: Azure AI Translator を用いた Abstract の翻訳処理。
```

- [ ] **Step 3: コミット**

```bash
git add README.md GEMINI.md
git commit -m "docs: document Azure Translator integration"
```

---

## Task 13: 仕上げ確認とチケット更新

- [ ] **Step 1: 全テストとビルドが通ることを最終確認**

Run: `go build ./... && go test ./... -v && go vet ./...`
Expected: 全コマンド成功（exit 0）

- [ ] **Step 2: DryRun で動作確認（翻訳 OFF）**

`.env` を一時的に以下にして実行：

```
TARGET_PLATFORM="slack"
SLACK_BOT_TOKEN="xoxb-dummy"
SLACK_CHANNEL_ID="C00000000"
DRY_RUN="true"
TRANSLATE_ENABLED="false"
```

Run: `go run ./cmd/dailybot`
Expected:
- `INFO: Translation disabled.` が出る
- `--- Main ---` ブロックに既存と同等のメッセージ（`*Abstract*:` 見出し、ヘッダリンクが `forum?id={paperID}`、PDF があれば `*PDF*:` 行）
- `--- Sub (thread) ---` ブロックは出ない

- [ ] **Step 3: DryRun で動作確認（翻訳 ON、Azure キー有り）**

`.env` に Azure Translator の有効なキー/リージョンを追加し、`TRANSLATE_ENABLED="true"` に変更。

Run: `go run ./cmd/dailybot`
Expected:
- `INFO: Translated abstract (len=... chars).` が出る
- `--- Main ---` に `*Abstract (日本語)*:` 見出し
- Slack 設定なら `--- Sub (thread) ---` に `*Original Abstract*:` ブロック
- Discord 設定なら Sub は出ず、Main 内に `*Original Abstract*:` の spoiler ブロック

実 API キーが手元になければこのステップは省略可（ユーザに任せる）。

- [ ] **Step 4: DPB-014 チケットのチェックボックスとステータスを更新**

`docs/tasks/v2/DPB-014.md` の以下を更新：

- 「やること」セクションの全 `- [ ]` を `- [x]` に
- 「成果物」セクションの全 `- [ ]` を `- [x]` に
- 「基本情報」のステータスを「未着手」→「完了」に

- [ ] **Step 5: 最終コミット**

```bash
git add docs/tasks/v2/DPB-014.md
git commit -m "docs(DPB-014): mark ticket completed"
```

- [ ] **Step 6: 完了報告**

完了をユーザに報告。プッシュや PR 作成はユーザ判断（`superpowers:finishing-a-development-branch` スキルを参照）。

---

## Notes for the implementer

- **TDD 厳守**: 各タスクは「失敗テスト → 実装 → グリーン → コミット」のサイクルを崩さない。
- **コミット粒度**: タスクごとに 1 コミット（Task 8 のみ複数ファイルにまたがるが 1 コミット）。コミットメッセージは Conventional Commits 形式。
- **ビルド維持**: 中間ステップでも `go build ./...` が必ず通る順序にしてある（Task 8 で main.go を一時的に修正するのはこのため）。
- **テストの独立性**: `os.Setenv` を使うテストは必ず `defer os.Unsetenv` をペアで書く。並列実行されると壊れるので `t.Parallel()` は使わない。
- **依存追加なし**: `go.mod` への新規依存は不要。Azure SDK は使わない。
- **疑問が出たら**: 設計書（`docs/superpowers/specs/2026-05-31-abstract-translation-design.md`）を最優先で参照。
