## 本タスクのチケット（Ticket）

- このタスクのチケットへのリンクを記載してください。

## 関連チケット (Related Ticket)

- 関連するチケットへのリンクを記載してください。 (例: `closes #1`, `refs #2`)

## 概要 (Overview)

- このPull Requestが解決する課題や目的を簡潔に説明してください。

## 主な変更点 (Main Changes)

- 具体的な変更内容を箇条書きで記載してください。
  - 例: OpenReviewクライアントに論文取得機能を追加
  - 例: `internal/formatter` パッケージを新設
  - 例: `daily.yml` のcronスケジュールを修正

## テスト (Testing)

- 実施したテストにチェックを入れてください。
- [ ] ユニットテスト実行 (`go test ./...`)
- [ ] ローカルでの実行確認 (`go run ./cmd/dailybot`)
- [ ] GitHub Actions上での動作確認

## 設定の変更 (Configuration Changes)

- 環境変数の追加・変更・削除があった場合に記載してください。ない場合は「なし」と記載してください。
  - 例: `FOO_BAR` 環境変数を追加

## 備考 (Remarks)

- その他、特記すべき点があれば記載してください。
