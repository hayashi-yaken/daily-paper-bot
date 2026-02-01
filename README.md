# Daily Paper Bot

OpenReviewから論文を自動取得し、Slack/Discordに投稿するGo製バッチBotです。
GitHub Actionsによる定期実行を想定して設計されています。

## 主な機能

- 指定したOpenReviewのVenueから論文リストを取得
- 投稿済みの論文を除外して、未投稿の中から1本をランダムに選定
- 選定した論文の情報を整形してSlackまたはDiscordに投稿
- 投稿記録をリポジトリ内のJSONファイルで管理し、重複投稿を防止

---

## 開発セットアップ

このリポジトリをフォークまたはクローンして開発を始めるための手順です。

### 1. リポジトリをクローン

```bash
git clone https://github.com/hayashi-yaken/daily-paper-bot.git
cd daily-paper-bot
```

### 2. Goのバージョン

このプロジェクトは `go 1.22` 以上を推奨しています。

### 3. 環境変数の設定

プロジェクトのルートにある `.env.sample` ファイルをコピーして `.env` ファイルを作成します。

```bash
cp .env.sample .env
```

その後、`.env` ファイルをエディタで開き、ご自身の環境に合わせて各値を設定してください。最低限、以下の項目が必要です。

- `OR_VENUE`, `OR_YEAR`
- `TARGET_PLATFORM` (`slack` または `discord`)
- 通知先プラットフォームに応じた認証情報 (`SLACK_BOT_TOKEN`, `DISCORD_WEBHOOK_URL` など)

`.env` ファイルは `.gitignore` に登録されているため、誤ってリポジトリにコミットされることはありません。

---

## 実行

### ローカルでの実行

ローカルでBotを一度だけ実行するには、以下のコマンドを使用します。
`.env` ファイルが自動的に読み込まれます（`go-dotenv` を利用）。

```bash
go run ./cmd/dailybot
```

`DRY_RUN="true"` を設定すると、実際に投稿せずに動作確認ができます。

### GitHub Actionsによる定期実行

`.github/workflows/daily.yml` に、毎日定刻にBotを実行するワークフローが定義されています。
本番運用では、リポジトリの `Settings > Secrets and variables > Actions` で必要な環境変数を設定してください。

---

## テスト

このプロジェクトには、ユニットテストとインテグレーションテストの2種類があります。

### ユニットテストの実行

外部APIへの通信を伴わない、基本的なロジックのテストです。
CI環境では、このテストが自動的に実行されます。

```bash
go test ./... -v
```

### インテグレーションテストの実行

実際にSlackやDiscordのAPIへメッセージを送信し、通知の疎通を確認するテストです。
このテストを実行するには、事前に `.env` ファイルに有効な `SLACK_BOT_TOKEN` や `DISCORD_WEBHOOK_URL` を設定しておく必要があります。

以下のコマンドで、インテグレーションテストを含む全てのテストを実行できます。

```bash
go test -tags=integration ./... -v
```

**注意**: このコマンドを実行すると、設定したチャンネルやサーバーに実際にテストメッセージが投稿されます。