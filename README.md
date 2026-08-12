# Daily Paper Bot

OpenReviewから論文を自動取得し、Slack/Discord に投稿するGo製バッチBot。

## 機能

- 指定したOpenReviewのVenue（ICLR / NeurIPS / ICML の 2023〜2025 年）から採択論文を取得
- 全採択論文の中からランダムに1本を選定
- 選定した論文の情報を整形して Slack または Discord に投稿
- (Optional) Azure AI Translator を用いた Abstract の日本語訳表示
  - Slack: 親メッセージに訳、原文はスレッド返信
  - Discord: 親メッセージに訳と spoiler 化した原文を返信

---

## セットアップ

### 1. リポジトリをクローン

```bash
git clone https://github.com/hayashi-yaken/daily-paper-bot.git
cd daily-paper-bot
```

### 2. Goのバージョン

`go 1.22` 以上を推奨。

### 3. 設定ファイルと環境変数の設定

#### 学会リストの設定

`assets/venues.json` ファイルを開き、対象としたい学会の情報を編集。
デフォルトでは ICLR (2024〜2025) / NeurIPS (2023〜2025) / ICML (2023〜2025) が登録されている。

`venue` フィールドの値は OpenReview API v2 の venueid（例: `ICLR.cc/2024/Conference`）で、採択論文の絞り込みにそのまま使われる。ICLR 2023 以前など旧 API v1 にしかない学会は対象外。

#### 環境変数の設定

プロジェクトのルートにある `.env.sample` ファイルをコピーして `.env` ファイルを作成。

```bash
cp .env.sample .env
```

その後、`.env` ファイルを設定する。以下は必須項目。

- `TARGET_PLATFORM` (`slack` または `discord`)
- 通知先プラットフォームに応じた認証情報 (`SLACK_BOT_TOKEN`, `DISCORD_WEBHOOK_URL` など)

OpenReview API は未認証アクセスがbot対策 (403) で弾かれることがあるため、`OR_EMAIL` / `OR_PASSWORD`(OpenReviewアカウントの認証情報)の設定を推奨。

#### Azure AI Translator（Optional）

Abstract を日本語訳して投稿に含めたい場合は、Azure ポータルで Translator リソースを作成し、以下の環境変数を設定する。

- `TRANSLATE_ENABLED`: `"true"` で機能を有効化（デフォルト `"false"`）
- `AZURE_TRANSLATOR_KEY`: Translator リソースのサブスクリプションキー
- `AZURE_TRANSLATOR_REGION`: リソースのリージョン（例: `japaneast`）
- `AZURE_TRANSLATOR_ENDPOINT`: 任意。デフォルト `https://api.cognitive.microsofttranslator.com`

翻訳 API が失敗した場合は WARN ログを出して原文だけで投稿を続行する。

---

## 実行

### ローカルでの実行

ローカルでBotを一度だけ実行するには、以下のコマンドを使用する。

```bash
go run ./cmd/dailybot
```

`DRY_RUN="true"` を設定すると、実際に投稿せずに動作確認が可能。

### 定時起動の構成

毎日の論文投稿は以下の 2 経路で起動するようになっている。

- **プライマリ**: Google Apps Script の時間トリガーから GitHub の `workflow_dispatch` API を叩いて `daily.yml` を起動する（JST 08:00 起動）
- **フォールバック**: `.github/workflows/daily.yml` の `schedule:` で定義された cron (`0 4 * * *` = UTC 04:00 = JST 13:00 狙い)

運用の際は、リポジトリの `Settings > Secrets and variables > Actions` から Bot 本体が必要とする環境変数を設定すること。

#### GAS トリガーのセットアップ

1. **Fine-grained PAT を発行する**
   - `Settings → Developer settings → Personal access tokens → Fine-grained tokens` で新規発行
   - Repository access: Only select repositories → `daily-paper-bot` のみ
   - Repository permissions: `Actions` を `Read and write`、それ以外は `No access`
   - Expiration はカレンダーに登録しておく（最大 1 年）
2. **Slack Incoming Webhook を発行する**
   - 既存の Slack App の `Features → Incoming Webhooks` を有効化
   - `Add New Webhook to Workspace` で論文投稿チャンネルと同じチャンネルを選択し、URL を発行
3. **GAS プロジェクトを作成する**
   - `https://script.google.com/` で新規プロジェクトを作成
   - リポジトリの `scripts/gas/trigger.gs` の内容をエディタに貼り付け（既存の `Code.gs` を置き換える）
4. **Script Properties を登録する**
   - `Project Settings → Script Properties` で以下を追加
     - `GITHUB_PAT`: 手順 1 で取得した PAT
     - `SLACK_WEBHOOK_URL`: 手順 2 で取得した Webhook URL
5. **タイムゾーンを `Asia/Tokyo` に設定する**
   - `Project Settings → General settings → Time zone` を `Asia/Tokyo` に変更
6. **時間トリガーを登録する**
   - `Triggers → Add Trigger`
   - Function: `trigger`、Deployment: `Head`、Event source: 時間主導型、Type: 日付ベースのタイマー、Hour: 午前 8 時 〜 9 時
7. **動作確認**
   - GAS エディタから `trigger` を手動実行 → `Actions` タブで `Daily Paper Bot` Run が起動することを確認
   - 数分以内に Slack の投稿チャンネルへ論文が投稿されることを確認
   - 失敗時は Slack に `:warning: GAS trigger failed ...` の通知が来ることを確認したい場合は、`GITHUB_PAT` を一度ダミー文字列に書き換えて手動実行する（確認後は必ず元に戻す）

---

## テスト

ユニットテストとインテグレーションテストの2種類がある。

### ユニットテストの実行

外部APIへの通信を伴わない、基本的なロジックのテスト。
CI環境では自動的に実行される。

```bash
go test ./... -v
```

### インテグレーションテストの実行

Slack や Discord の API へメッセージを送信し、通知の疎通を確認するテスト。
このテストを実行するには、事前に `.env` ファイルに有効な `SLACK_BOT_TOKEN` や `DISCORD_WEBHOOK_URL` を設定しておく必要がある。

以下のコマンドで、インテグレーションテストを含む全てのテストを実行できる。

```bash
go test -tags=integration ./... -v
```

**注意**: このコマンドを実行すると、設定したチャンネルやサーバーに実際にテストメッセージが投稿される。
