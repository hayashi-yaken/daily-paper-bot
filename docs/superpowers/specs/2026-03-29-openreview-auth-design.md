# OpenReview 認証対応 設計書

**作成日:** 2026-03-29
**ステータス:** 承認済み
**対象ブランチ:** develop

---

## 1. 背景・目的

GitHub Actions 上で `go run ./cmd/dailybot` を実行したところ、OpenReview API から `403 Forbidden` が返るようになった。ブラウザからは同じエンドポイントへのアクセスが可能であることから、**GitHub Actions の IP レンジが OpenReview 側でブロックされている**と判断した。

対策として、OpenReview API v2 の JWT 認証を導入し、認証済みリクエストとして論文データを取得する。

---

## 2. スコープ

### 含む
- OpenReview API v2 `/login` エンドポイントを使った JWT 認証
- `openreview.Client` への `Login()` メソッド追加
- `GetNotes()` への `Authorization` ヘッダー付与
- `Config` への認証情報フィールド追加
- GitHub Actions ワークフローへの Secret 追加

### 含まない
- トークンのキャッシュ・永続化（毎回ログインする方式）
- リトライ・フォールバック（認証失敗時は即 FATAL）
- その他エンドポイントへの認証対応

---

## 3. 設計

### 3.1 設定（`internal/config/config.go`）

`Config` 構造体に以下の2フィールドを追加する。

| フィールド | 環境変数 | 必須 | 説明 |
|---|---|---|---|
| `OpenReviewEmail` | `OR_EMAIL` | 任意 | OpenReview ログイン用メールアドレス |
| `OpenReviewPassword` | `OR_PASSWORD` | 任意 | OpenReview ログイン用パスワード |

- 両方セットされている場合のみ認証を試みる
- どちらかが空の場合は認証をスキップする（ローカル開発・DRY_RUN との互換性維持）
- 認証失敗時は即 FATAL で終了する

### 3.2 OpenReview クライアント（`internal/openreview/openreview.go`）

#### `Client` 構造体の変更

```go
type Client struct {
    httpClient *http.Client
    BaseURL    string
    UserAgent  string
    token      string  // 追加: 空文字 = 未認証
}
```

#### `Login()` メソッドの追加

```
POST https://api2.openreview.net/login
Content-Type: application/json

{"id": "<email>", "password": "<password>"}
```

レスポンスの形式は以下のとおり。

```json
{"token": "<JWT>", "user": {...}}
```

- レスポンスから `token` フィールドを取得し `c.token` に保存する
- ステータスコードが 200 以外の場合はエラーを返す

#### `GetNotes()` の変更

- `c.token` が空でない場合、リクエストヘッダーに以下を付与する
  ```
  Authorization: Bearer <token>
  ```
- `c.token` が空の場合は従来どおり（ヘッダーなし）

### 3.3 エントリーポイント（`cmd/dailybot/main.go`）

`openreview.NewClient()` の直後に以下の認証ブロックを追加する。

```go
if cfg.OpenReviewEmail != "" && cfg.OpenReviewPassword != "" {
    if err := orClient.Login(cfg.OpenReviewEmail, cfg.OpenReviewPassword); err != nil {
        log.Fatalf("FATAL: failed to login to openreview: %v", err)
    }
    log.Println("INFO: Authenticated to OpenReview.")
}
```

### 3.4 GitHub Actions（`.github/workflows/daily.yml`）

`env:` セクションに以下を追加する。

```yaml
OR_EMAIL: ${{ secrets.OR_EMAIL }}
OR_PASSWORD: ${{ secrets.OR_PASSWORD }}
```

リポジトリの **Settings > Secrets and variables > Actions** に `OR_EMAIL` と `OR_PASSWORD` を登録する。

---

## 4. データフロー

```
main.go
  └─ cfg.OpenReviewEmail / cfg.OpenReviewPassword が両方あれば
       └─ orClient.Login(email, password)
            └─ POST /login → token を c.token に保存
  └─ orClient.GetNotes(venue)
       └─ GET /notes?invitation=... + Authorization: Bearer <token>
```

---

## 5. エラーハンドリング

| ケース | 挙動 |
|---|---|
| `OR_EMAIL` または `OR_PASSWORD` が未設定 | 認証スキップ（警告なし） |
| `/login` がステータス 200 以外を返した | `FATAL: failed to login to openreview: ...` で即終了 |
| ネットワークエラー（ログイン時） | `FATAL: failed to login to openreview: ...` で即終了 |
| 認証後の `GetNotes` が 403 を返した | 既存の `FATAL: failed to get notes from openreview: ...` で終了 |

---

## 6. テスト方針

- `Login()` のユニットテスト: `httptest.NewServer` でモックサーバーを立て、正常系・異常系（4xx）を検証する
- `GetNotes()` のヘッダー付与テスト: トークンがセットされた場合に `Authorization` ヘッダーが付与されることを検証する
- `Config` のロードテスト: `OR_EMAIL` / `OR_PASSWORD` の有無による挙動差分を検証する

---

## 7. 変更ファイル一覧

| ファイル | 変更内容 |
|---|---|
| `internal/config/config.go` | `OpenReviewEmail`, `OpenReviewPassword` フィールドと読み込み追加 |
| `internal/openreview/openreview.go` | `token` フィールド・`Login()` メソッド追加、`GetNotes()` にヘッダー付与 |
| `cmd/dailybot/main.go` | 認証ブロック追加 |
| `.github/workflows/daily.yml` | `OR_EMAIL`, `OR_PASSWORD` の環境変数追加 |

---

## 8. 事前準備（ユーザー作業）

1. [https://openreview.net](https://openreview.net) でアカウントを作成する
2. GitHub リポジトリの **Settings > Secrets and variables > Actions** で以下を登録する
   - `OR_EMAIL`: 登録したメールアドレス
   - `OR_PASSWORD`: 登録したパスワード
