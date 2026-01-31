# Geminiエージェント向け指示書: "Daily Paper Bot"

このドキュメントは、Geminiエージェントがこのプロジェクトで作業するための、プロジェクト固有のコンテキストと指示を記載します。

## 1. プロジェクト概要

これはGo言語で記述されたバッチ型のBotです。OpenReviewから学術論文を自動的に取得し、「今日の論文」を1本選定して、指定されたプラットフォーム（SlackまたはDiscord）に投稿します。常駐サーバを持たず、GitHub Actionsの定期実行によって動作するように設計されています。

## 2. 技術スタック

- **言語**: Go
- **実行環境**: GitHub Actions (cronによる定期実行)
- **主要な外部ライブラリ**:
  - `slack-go/slack` (Slack通知用)
  - その他は標準ライブラリ（HTTP, JSONなど）を中心に使用
- **永続化**: リポジトリにコミットされる単一の `data/posted.json` ファイル

## 3. プロジェクト構造

このプロジェクトは、標準的なGoアプリケーションのレイアウトに従います。

- `cmd/dailybot/`: アプリケーションのメインエントリーポイント (`main.go`)。
- `internal/`: アプリケーションのコアロジック全体を格納します。外部パッケージからの利用は想定していません。
  - `config/`: 環境変数からの設定読み込みを処理します。
  - `openreview/`: OpenReview APIから論文データを取得するためのクライアントです。
  - `storage/`: 重複投稿を防ぐため、`data/posted.json` の読み書きを管理します。
  - `selector/`: 候補リストから論文を1本選定するロジックを実装します。
  - `formatter/`: 論文情報を投稿用のメッセージ文字列に整形します。
  - `notifier/`: SlackまたはDiscordへメッセージを送信する処理を担当します。
- `data/`: 投稿済み論文を記録する `posted.json` ファイルを格納します。
- `.github/workflows/`: 定期実行のためのGitHub Actionsワークフローファイル (`daily.yml`) を格納します。

## 4. 開発ワークフロー

### ローカルでの実行

ローカルでBotを実行するには、環境変数で必要な設定を提供する必要があります。

```bash
# Slackに投稿する場合の実行例
export OR_VENUE="ICLR.cc/2025/Conference"
export OR_YEAR="2025"
export TARGET_PLATFORM="slack"
export SLACK_BOT_TOKEN="xoxb-your-token"
export SLACK_CHANNEL_ID="C12345678"

go run ./cmd/dailybot
```

実際に投稿せず、ログ出力のみを行うドライランを実行する場合：

```bash
export DRY_RUN="true"
go run ./cmd/dailybot
```

### テストの実行

プロジェクトのルートディレクトリから全てのユニットテストを実行します。

```bash
go test ./... -v
```

### 依存関係の管理

標準的なGoモジュールのコマンドを使用します。

- 新しい依存関係を追加する場合: `go get github.com/new/package`
- 依存関係を整理する場合: `go mod tidy`

## 5. 設定

アプリケーションは以下の環境変数によって設定されます。本番環境では、シークレットはGitHub Secretsで管理してください。

- `OR_VENUE`: (必須) OpenReviewのvenue/invitation ID。
- `OR_YEAR`: (必須) カンファレンスの年（表示用）。
- `TARGET_PLATFORM`: (必須) `slack` または `discord`。
- `SLACK_BOT_TOKEN`: (Secret) Slack API用のBotトークン。
- `SLACK_CHANNEL_ID`: (Secret) 投稿先のチャンネルID。
- `DISCORD_WEBHOOK_URL`: (Secret) Discord用のWebhook URL。
- `SELECT_STRATEGY`: (任意) デフォルトは `random`。
- `ABSTRACT_MAX_CHARS`: (任意) Abstractの最大文字数。デフォルトは `1200`。
- `DRY_RUN`: (任意) `true` の場合、Botは投稿も結果の保存も行いません。

## 6. デプロイ

- デプロイは `.github/workflows/daily.yml` に定義されたGitHub Actionsによって完全に処理されます。
- ワークフローは毎日定刻に実行 (`cron`) されるほか、手動での実行 (`workflow_dispatch`) も可能です。
- 実行成功後、ワークフローは `data/posted.json` への変更を自動的にコミット＆プッシュし、状態を更新します。
- 全てのシークレットは、リポジトリの「Settings > Secrets and variables > Actions」で設定する必要があります。

## 7. 作業プロトコル (Working Protocol)

- **チケットの更新**: `docs/tasks` 内のチケットに記載されたタスクを完了した場合、必ず該当のマークダウンファイルを編集し、「やること」および「成果物」セクションのチェックボックス (`- [ ]`) を完了状態 (`- [x]`) に変更してください。また、基本情報の「ステータス」を「完了」に更新してください。これは作業の進捗を正確に追跡するために不可欠です。
- **ドキュメントの鮮度維持**: 実装がチケットの計画やコード例から大幅に逸脱した場合、完了後にチケットの内容を最終的な実装に合わせて更新してください。これにより、ドキュメントとコードの乖離を防ぎます。
