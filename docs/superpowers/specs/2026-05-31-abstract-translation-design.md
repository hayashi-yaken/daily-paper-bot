# Abstract 翻訳表示（Azure AI Translator）設計書

**作成日:** 2026-05-31
**ステータス:** 実装済み
**対象ブランチ:** docs/dpb-014-abstract-translation

---

## 1. 背景・目的

現状、Daily Paper Bot は OpenReview から取得した論文の Abstract を英語のまま Slack / Discord に投稿している。日本語話者の読者にとっては「何の論文か」を瞬時に把握しづらく、興味の入口として機能しにくい。

Azure AI Translator を用いて Abstract を日本語に機械翻訳し、投稿の主役を訳に置きつつ原文も別表現で確認できる仕組みを導入する。あわせて、ヘッダリンクが常にカンファレンスTOPを指す現行仕様を、その日の論文ページを指すように修正する。

---

## 2. スコープ

### 含む

- Azure AI Translator (v3.0 `/translate`) を叩く `internal/translator` パッケージの新設
- 翻訳結果を投稿メッセージに含めるための `formatter` / `notifier` のシグネチャ拡張
  - Slack: 親メッセージに訳、スレッド子に原文
  - Discord: 親メッセージに訳と原文の spoiler を同梱
- 翻訳機能の ON/OFF を制御する環境変数（`TRANSLATE_ENABLED`）と、Azure リソース認証情報の追加
- 翻訳失敗時のフォールバック（原文だけで投稿し WARN ログ）
- ヘッダリンクを `forum?id={paperID}` に変更
- 本文末尾の `*Link*:` を `*PDF*:` に改名し、PDF が取れているときのみ出力

### 含まない

- LLM (Azure OpenAI) を使った翻訳・要約
- 翻訳結果のキャッシュ・永続化（毎回 API を呼ぶ）
- リトライ・指数バックオフ
- 日本語以外の対象言語切り替え（コード上は `ja` 固定）
- スレッド投稿失敗時のリトライ（WARN ログのみで吸収）

---

## 3. 設計

### 3.1 設定（`internal/config/config.go`）

`Config` に以下のフィールドを追加する。

| フィールド | 環境変数 | 必須 | デフォルト | 説明 |
|---|---|---|---|---|
| `TranslateEnabled` | `TRANSLATE_ENABLED` | 任意 | `false` | 翻訳機能の ON/OFF。`strconv.ParseBool` で解釈 |
| `AzureTranslatorEndpoint` | `AZURE_TRANSLATOR_ENDPOINT` | 任意 | `https://api.cognitive.microsofttranslator.com` | Translator エンドポイント |
| `AzureTranslatorRegion` | `AZURE_TRANSLATOR_REGION` | `TRANSLATE_ENABLED=true` のとき必須 | なし | リソースのリージョン (例: `japaneast`) |
| `AzureTranslatorKey` | `AZURE_TRANSLATOR_KEY` | `TRANSLATE_ENABLED=true` のとき必須 | なし | Translator のサブスクリプションキー |

バリデーション：

- `TRANSLATE_ENABLED=true` のとき、`AzureTranslatorRegion` と `AzureTranslatorKey` が空ならロード失敗
- `TRANSLATE_ENABLED=false` のときは Key/Region 未設定でも問題なし
- `Endpoint` は未設定時に常にデフォルトを当てる

### 3.2 Translator パッケージ（`internal/translator/translator.go` 新設）

#### インターフェース

```go
type Translator interface {
    Translate(text, targetLang string) (string, error)
}
```

#### Azure 実装

```go
type azureTranslator struct {
    httpClient *http.Client
    endpoint   string
    region     string
    key        string
}

func NewAzureTranslator(endpoint, region, key string) Translator
```

#### API 仕様 (Azure AI Translator v3.0)

```
POST {endpoint}/translate?api-version=3.0&to={targetLang}
Ocp-Apim-Subscription-Key: {key}
Ocp-Apim-Subscription-Region: {region}
Content-Type: application/json

[{"Text": "..."}]
```

レスポンス:

```json
[{"translations": [{"text": "...", "to": "ja"}]}]
```

挙動：

- `text == ""` のときは API を呼ばず `("", nil)` を返す
- HTTP タイムアウトは 30 秒（既存 `openreview.Client` と同等）
- 2xx 以外はステータスコードとレスポンスボディを含むエラーを返す
- レスポンスが想定外（配列が空、`translations` が空、`text` が空）の場合はエラー
- リトライはしない
- `targetLang` はインターフェース上は引数で受け取るが、本スコープでは呼び出し側 (`main.go`) が `"ja"` を直接渡す（多言語対応はスコープ外）

### 3.3 Formatter（`internal/formatter/formatter.go`）

#### 戻り値の型

```go
type Message struct {
    Main string // 親メッセージ（または単発メッセージ）
    Sub  string // Slack のスレッド子。Discord は無視
}

type Formatter interface {
    Format(paper *openreview.Note, venue config.VenueConfig, abstractMaxChars int, jaAbstract string) Message
}
```

#### ヘッダの仕様変更（翻訳とは独立した改修）

- リンク先を `https://openreview.net/group?id={venue}` から `https://openreview.net/forum?id={paperID}` に変更
- ラベル文言 `📄 今日の論文 ({Name} {Year})` は据え置き

#### 本文末尾のリンク表現

- PDF が取得できている → `*PDF*: {pdfURL}` を末尾に出す
- PDF が取得できていない → 行ごと省略

#### 翻訳結果がある場合（Slack）

```
<{forumURL}|📄 今日の論文 (ICLR 2025)>

*Title*: ...
*Authors*: ...

*Abstract (日本語)*:
{jaAbstract をトリム}

*PDF*: {pdfURL}   ← PDF があれば

ID: `{paperID}`
```

スレッド子 (`Sub`):

```
*Original Abstract*:
{原文をトリム}
```

#### 翻訳結果がある場合（Discord）

```
[📄 今日の論文 (ICLR 2025)]({forumURL})

*Title*: ...
*Authors*: ...

*Abstract (日本語)*:
{jaAbstract をトリム}

*Original Abstract*:
||{原文をトリム}||

*PDF*: {pdfURL}   ← PDF があれば

ID: `{paperID}`
```

`Sub` は常に空。

#### 翻訳結果が無い場合（`jaAbstract == ""`）

- 「日本語」見出しは出さず、`*Abstract*:` で原文を出す（現行と同じ表示）
- Slack の `Sub` は空、Discord の spoiler ブロックも出さない
- ヘッダ改修と PDF 行の改修は翻訳の有無と独立に常時適用

#### トリム

- 原文・訳ともに `ABSTRACT_MAX_CHARS` を rune ベースで適用
- 既存の切り詰めロジックを共通ヘルパに抽出

### 3.4 Notifier（`internal/notifier/`）

#### インターフェース

```go
type Notifier interface {
    Post(msg formatter.Message) error
}
```

#### Slack 実装

1. `chat.postMessage` で `msg.Main` を投稿し、レスポンスから親 `ts` を取得
2. `msg.Sub != ""` のとき、`thread_ts=ts` を付けて `chat.postMessage` で子を投稿
3. 子投稿の失敗は WARN ログを出して `nil` を返す（親はすでに届いている）
4. 親投稿の失敗は `error` を返し、`main.go` 側で FATAL になる

`slack-go/slack` の `PostMessage(channelID, slack.MsgOptionText(...), slack.MsgOptionTS(parentTS))` を使う。

#### Discord 実装

- `msg.Main` を Webhook に投げるだけ。`msg.Sub` は無視
- 既存ロジックからの差分は引数型のみ

### 3.5 main.go の組み立て（`cmd/dailybot/main.go`）

論文選定の直後に翻訳ステップを追加する。

```go
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

message := paperFormatter.Format(selectedNote, selectedVenue, cfg.AbstractMaxChars, jaAbstract)
```

DryRun 表示は Sub があれば続けて出す。

```go
log.Printf("--- Main ---\n%s", message.Main)
if message.Sub != "" {
    log.Printf("--- Sub (thread) ---\n%s", message.Sub)
}
```

---

## 4. データフロー

```
config.Load
  ├─ TRANSLATE_ENABLED, AZURE_TRANSLATOR_{KEY,REGION,ENDPOINT} を解釈
  └─ ENABLED=true でキー欠落ならロード失敗
       ▼
venueselector.Select ─ openreview.GetNotes ─ selector.Select
                                                  ▼
                              cfg.TranslateEnabled ?
                                ├─ yes → translator.Translate(abstract, "ja")
                                │           ├─ 成功 → jaAbstract
                                │           └─ 失敗 → jaAbstract="" + WARN
                                └─ no  → jaAbstract=""
                                                  ▼
                          formatter.Format(note, venue, max, jaAbstract)
                                                  ▼
                                       Message{Main, Sub}
                                                  ▼
                                       notifier.Post(Message)
                                          ├─ Slack:   Main 親 + Sub をスレッド
                                          └─ Discord: Main のみ
```

---

## 5. エラーハンドリング

| ケース | 挙動 |
|---|---|
| `TRANSLATE_ENABLED=true` で Key/Region 未設定 | `config.Load` がエラー → 起動失敗 |
| `TRANSLATE_ENABLED=false` | 翻訳ステップをスキップ。INFO ログを 1 行 |
| Translator API 4xx/5xx | WARN ログを出し、訳なしで投稿継続 |
| Translator のネットワークエラー / タイムアウト | WARN ログを出し、訳なしで投稿継続 |
| Translator レスポンス構造異常（空配列など） | WARN ログを出し、訳なしで投稿継続 |
| Slack 親投稿失敗 | FATAL（既存と同じ） |
| Slack スレッド子投稿失敗 | WARN ログのみ。Bot は正常終了 |
| Discord 投稿失敗 | FATAL（既存と同じ） |

---

## 6. テスト方針

| 層 | テスト | 内容 |
|---|---|---|
| `translator` | ユニット | `httptest.NewServer` でモック。200 正常系、4xx/5xx、ネットワークエラー、レスポンス JSON 異常、空入力で API 呼ばない |
| `formatter` | ユニット | Slack/Discord × 訳あり/なし × PDF あり/なし。ヘッダリンクが `forum?id={paperID}` を指すこと。Slack の Sub に原文、Discord の Main に spoiler が入ること。訳なし時は現行レイアウト相当 |
| `notifier/slack` | ユニット | `httptest.NewServer` でモック。親のみ投稿（Sub 空）、親 + スレッド両方投稿、スレッド失敗時に `Post` が `nil` |
| `notifier/discord` | ユニット | `Message{Main, Sub}` を渡しても Sub が無視されてペイロードに含まれないこと |
| `config` | ユニット | `TRANSLATE_ENABLED=true` で Key 未設定 → エラー、`false` で問題なし、デフォルト値が当たる |
| インテグレーション | 任意 | `go test -tags=integration` で Slack/Discord に実投稿（既存方針踏襲）。Translator のインテグレーションは追加しない |

`cmd/dailybot/main.go` のユニットテストは追加しない（既存方針踏襲）。

---

## 7. 変更ファイル一覧

| ファイル | 変更内容 |
|---|---|
| `internal/translator/translator.go` | 新規（Translator インターフェース + Azure 実装） |
| `internal/translator/translator_test.go` | 新規 |
| `internal/config/config.go` | 翻訳関連フィールド追加と env 読み込み・バリデーション |
| `internal/config/config_test.go` | 新フィールド検証ケース追加 |
| `internal/formatter/formatter.go` | `Message` 型導入。ヘッダリンクを forum URL に。PDF 行の条件出力。訳ありレイアウト |
| `internal/formatter/formatter_test.go` | 訳あり/なし × Slack/Discord × PDF あり/なしの組合せを追加 |
| `internal/notifier/notifier.go` | `Post(Message)` にシグネチャ変更 |
| `internal/notifier/slack.go` | スレッド投稿対応。子投稿失敗は WARN 吸収 |
| `internal/notifier/discord.go` | `Message` 受け取りに変更（Sub は無視） |
| `internal/notifier/slack_test.go` / `discord_test.go` | 上記対応 |
| `cmd/dailybot/main.go` | 翻訳ステップ追加、DryRun 表示更新 |
| `.env.sample` | `TRANSLATE_ENABLED`, `AZURE_TRANSLATOR_KEY`, `AZURE_TRANSLATOR_REGION`, `AZURE_TRANSLATOR_ENDPOINT` を追加 |
| `.github/workflows/daily.yml` | Secret 参照を `env:` に追加 |
| `README.md` / `GEMINI.md` | 設定説明と挙動説明を追補 |

---

## 8. 事前準備（ユーザー作業）

1. Azure ポータルで「Translator」リソースを作成する（無料枠 F0 で 200 万文字/月）
2. リージョンと Key 1 を控える
3. GitHub リポジトリの **Settings > Secrets and variables > Actions** に以下を登録する
   - `AZURE_TRANSLATOR_KEY`: 取得したサブスクリプションキー
   - `AZURE_TRANSLATOR_REGION`: リソースのリージョン (例: `japaneast`)
4. `TRANSLATE_ENABLED=true` を `.env` または GitHub Secrets に追加する
