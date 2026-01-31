# 今日の論文 Bot（Go / GitHub Actions / OpenReview）仕様書（MVP）

## 1. 概要

OpenReview 上の対象カンファレンス（初期は ICLR）から論文を取得し、1日1本「今日の論文」を選定して Slack または Discord に自動投稿するバッチ型 Bot を実装する。

- フロントエンドなし
- 常駐サーバなし
- 実行基盤は GitHub Actions の定期実行（cron）
- 実装言語は Go

> ※ AI要約拡張予定。現時点では仕様外。

---

## 2. 目的

- 論文キャッチアップの習慣化
- OpenReview の取得・フィルタ・投稿・重複排除が安定稼働する最小構成を確立

---

## 3. スコープ

### 3.1 本仕様に含む

- OpenReview から論文メタデータ取得
- 1日1本の選定
- Slack / Discord への投稿（MVPではどちらか一方でも可）
- 重複投稿防止（永続化）
- GitHub Actions による定期実行

### 3.2 本仕様に含まない（将来拡張）

- PDF本文取得・全文抽出・AI要約
- Web UI
- ユーザー別設定、複数チャンネル配信
- 週次/月次まとめ投稿

---

## 4. 対象データソース

### 4.1 OpenReview

- 初期対象: ICLR（年度は設定で指定）
- 取得対象: OpenReview 上の該当 Venue / Invitation の submissions（ノート）

#### 取得する項目（必須）

- paper_id（OpenReview note ID）
- title
- authors（著者名）
- abstract
- pdf_url（可能な場合）
- venue / year（投稿時に表示するため）

#### 取得する項目（任意）

- keywords / subject areas（取得できる場合）
- publication date / creation date（選定ロジックに利用する場合）

---

## 5. 実行環境

### 5.1 実行基盤

- GitHub Actions

### 5.2 実行スケジュール

- 毎日 09:00 JST
- GitHub Actions cron は UTC 基準で設定する
  - 例: `0 0 * * *`（JST 09:00 = UTC 00:00）
- スケジュール遅延（数分程度）は許容範囲とする

---

## 6. システム構成

### 6.1 リポジトリ構成（例）

```
.
├── cmd/
│ └── dailybot/
│ └── main.go # エントリーポイント
├── internal/
│ ├── openreview/  # OpenReview クライアント
│ ├── selector/    # 論文選定ロジック
│ ├── formatter/   # 投稿文整形
│ ├── notifier/    # Slack/Discord 投稿
│ ├── storage/     # 重複防止ストレージ
│ └── config/      # 設定ロード
├── data/
│ └── posted.json  # 投稿済み記録（MVP）
├── .github/
│ └── workflows/
│ └── daily.yml    # 定期実行
├── go.mod
├── go.sum
└── README.md
```

### 6.2 実行フロー

1. GitHub Actions がジョブを起動
2. 設定を読み込み（対象 venue / year / 投稿先など）
3. OpenReview から論文一覧を取得
4. 投稿済みを除外（paper_id で判定）
5. 選定ロジックで「今日の1本」を決定
6. 投稿文を生成
7. Slack または Discord に投稿
8. 投稿済み記録を保存（paper_id と日付）
9. 終了（ログ出力）

---

## 7. 設定仕様

### 7.1 設定方法

- GitHub Actions の環境変数 + Secrets を使用

### 7.2 設定項目

#### OpenReview（必須）

- `OR_VENUE` : 例 `ICLR.cc/2025/Conference` など
- `OR_YEAR` : 例 `2025`（表示用。取得に使うかは実装依存）

> 注: OpenReview の “venue / invitation” は年によって変わる可能性があるため、
> 初期は **手動で `OR_VENUE` を設定**する方針とする。

#### 投稿先（必須）

- `TARGET_PLATFORM`: `slack` or `discord`

#### Slack（Slackを使う場合）

- `SLACK_BOT_TOKEN`（Secret）
- `SLACK_CHANNEL_ID`

#### Discord（Discordを使う場合）

- `DISCORD_WEBHOOK_URL`（Secret）

#### 選定（任意）

- `SELECT_STRATEGY`: `random`（MVPは固定で可）
- `ABSTRACT_MAX_CHARS`: 抽象の最大文字数（例: 1200）
- `DRY_RUN`: `true` の場合は投稿せずログ出力のみ（任意）

---

## 8. 論文選定仕様（MVP）

### 8.1 前提

- 取得した論文リストから、投稿済みのものは除外する
- 「未投稿」が1本以上存在する場合のみ投稿する

### 8.2 選定ルール（MVP）

- 未投稿論文から **ランダムに1本**選ぶ

### 8.3 未投稿が存在しない場合

- 投稿は行わず正常終了（exit code 0）
- ログに `no candidates` を出力する

---

## 9. 投稿内容仕様

### 9.1 投稿フォーマット（共通）

- 文字は Markdown（Slack/Discordで概ね読める範囲）
- 1メッセージ内に収める
- Abstract は長い場合に切り詰める（`ABSTRACT_MAX_CHARS`）

#### 例

```
📄 今日の論文 (ICLR 2025)

Title: <タイトル>
Authors: <著者1>, <著者2>, …

Abstract:
<abstract（必要なら省略）>

Link:
<pdf_url または OpenReview のリンク>
ID: <paper_id>
```

### 9.2 リンク仕様

- 優先: `pdf_url`
- 取得できない場合:
  - OpenReview の paper / forum URL を構成して提示（実装方針で決定）
- どちらも不可の場合:
  - `paper_id` のみを提示（最低限）

---

## 10. 重複投稿防止（永続化）

### 10.1 方式（MVP）

- `data/posted.json` に投稿済みを記録する
- 形式は **paper_id をキーに保持**し、同一 paper_id の再投稿を禁止

#### 保存例

```json
{
  "posted": {
    "<paper_id>": {
      "date": "2026-01-31",
      "venue": "ICLR.cc/2025/Conference"
    }
  }
}
```

### 10.2 永続化の扱い（GitHub Actions）

    •	posted.json をリポジトリにコミットする方式（MVP）
    •	実行後に bot が commit & push して状態を更新
    •	Push できない場合は失敗扱い（運用方針により変更可）

注: 将来的には DB / KV（例: SQLiteをartifactに保持、S3、GitHub Releases等）へ移行可能。

⸻

## 11. エラーハンドリング

### 11.1 OpenReview取得失敗

    •	例外として終了（exit code != 0）
    •	ログに原因を出す（HTTP status / error message）

### 11.2 投稿失敗（Slack/Discord）

    •	例外として終了（exit code != 0）
    •	失敗時は posted.json を更新しない（再実行で再投稿できるように）

### 11.3 データ不整合

    •	必須項目（paper_id/title）が欠落している場合は候補から除外し、ログ警告

⸻

## 12. ログ仕様

標準出力に以下を出力する（GitHub Actionsログで確認できること）。
• 実行開始時刻（JST/UTCどちらか）
• venue/year
• 取得件数
• 投稿済み除外後の候補件数
• 選定した paper_id
• 投稿先（slack/discord）
• 投稿成功/失敗

⸻

## 13. 非機能要件

    •	1回の実行で完結すること（ステートは posted.json のみ）
    •	実行時間: 5分以内を目安
    •	OpenReview API に過剰なリクエストをしない（必要最小限）
    •	シークレットは GitHub Secrets で管理し、ログに出さない

⸻

## 14. GitHub Actions（MVP要件）

    •	cron での定期実行
    •	workflow_dispatch（手動実行）を有効化
    •	Go のセットアップ（actions/setup-go）
    •	実行後に posted.json を更新した場合は commit & push（MVP）

⸻

## 15. 受け入れ条件（Definition of Done）

    •	ローカルで go run ./cmd/dailybot が実行できる
    •	GitHub Actions が毎日起動し、指定チャンネルへ投稿できる
    •	同じ paper_id が2度投稿されない
    •	OpenReview取得に失敗した場合にジョブが失敗として判定される
    •	投稿失敗時に posted.json が更新されない

```

```
