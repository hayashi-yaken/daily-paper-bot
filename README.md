# Daily Paper Bot

OpenReviewから論文を自動取得し、Slack/Discordに投稿するGo製バッチBotです。
GitHub Actionsによる定期実行を想定して設計されています。

## 主な機能

- 指定したOpenReviewのVenueから論文リストを取得
- 取得した論文の中からランダムに1本を選定
- 選定した論文の情報を整形してSlackまたはDiscordに投稿
- (任意) Azure AI Translator を用いた Abstract の日本語訳表示
  - Slack: 親メッセージに訳、原文はスレッド返信
  - Discord: 親メッセージに訳と spoiler 化した原文を同梱

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

### 3. 設定ファイルと環境変数の設定

#### 学会リストの設定

`assets/venues.json` ファイルをエディタで開き、対象としたい学会の情報を編集します。

#### 環境変数の設定

プロジェクトのルートにある `.env.sample` ファイルをコピーして `.env` ファイルを作成します。

```bash
cp .env.sample .env
```

その後、`.env` ファイルをエディタで開き、ご自身の環境に合わせて各値を設定してください。最低限、以下の項目が必要です。

- `TARGET_PLATFORM` (`slack` または `discord`)
- 通知先プラットフォームに応じた認証情報 (`SLACK_BOT_TOKEN`, `DISCORD_WEBHOOK_URL` など)

#### Azure AI Translator（任意）

Abstract を日本語訳して投稿に含めたい場合は、Azure ポータルで Translator リソースを作成し、以下の環境変数を設定します。

- `TRANSLATE_ENABLED`: `"true"` で機能を有効化（デフォルト `"false"`）
- `AZURE_TRANSLATOR_KEY`: Translator リソースのサブスクリプションキー
- `AZURE_TRANSLATOR_REGION`: リソースのリージョン（例: `japaneast`）
- `AZURE_TRANSLATOR_ENDPOINT`: 任意。デフォルト `https://api.cognitive.microsofttranslator.com`

翻訳 API が失敗した場合は WARN ログを出して原文だけで投稿を続行します（投稿はスキップしません）。

`.env` ファイルは `.gitignore` に登録されているため、誤ってリポジトリにコミットされることはありません。

---

## 実行

### ローカルでの実行

ローカルでBotを一度だけ実行するには、以下のコマンドを使用します。
`.env` ファイルが自動的に読み込まれます（`godotenv` を利用）。

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
