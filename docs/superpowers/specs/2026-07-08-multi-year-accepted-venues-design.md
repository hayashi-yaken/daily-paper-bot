# 設計書: 過年度会場の追加 + 採択論文からの公平なランダム選定

日付: 2026-07-08
関連チケット: DPB-016

## 背景と目的

現在のボットは `assets/venues.json` に ICLR / NeurIPS / ICML の 2025 年会場のみが登録されており、2025 年の論文しか投稿されない。これを 2023〜2024 年の会場にも広げる。

あわせて、調査で判明した既存の 2 つの問題をこの機会に解消する:

1. **不採択論文も投稿対象になっている**: 現在の取得クエリ `invitation=<venue>/-/Submission` は全投稿(採択+不採択+取り下げ)を返す。終了済みの会議ではランダム選定の結果、不採択論文が投稿されることが頻繁に起きる。
2. **先頭 1000 件バイアス**: OpenReview の `/notes` API はデフォルトで先頭 1000 件しか返さない。ICLR 2025 の投稿は 1 万件超のため、現状は「最初の 1000 件」からしか論文が選ばれていない。採択論文に絞っても各会議 1800〜4600 件程度あり、1000 件を超える。

## 対象範囲

`assets/venues.json` を以下の 8 エントリに拡張する(スキーマ変更なし):

| 会議    | 年               |
| ------- | ---------------- |
| ICLR    | 2025, 2024       |
| NeurIPS | 2025, 2024, 2023 |
| ICML    | 2025, 2024, 2023 |

**ICLR 2023 は対象外**とする。ICLR 2023 のみ旧 OpenReview API v1(別エンドポイント・別 invitation 形式・別レスポンス形式)にあり、対応コストが見合わないため(実測: API v1 の `Blind_Submission` に 3,792 件存在することは確認済み)。上記 8 会場はすべて現行の API v2 で取得できる。

## 設計

### 1. OpenReview クライアント (`internal/openreview`)

`GetNotes(venue)` を以下のメソッドに置き換える:

```go
// GetAcceptedNotes は venueid で採択論文を limit/offset 指定で取得し、
// notes と全体件数 (count) を返します。
func (c *Client) GetAcceptedNotes(venue string, limit, offset int) ([]Note, int, error)
```

- クエリ: `/notes?content.venueid=<venue>&limit=<limit>&offset=<offset>`
  - 採択論文だけが会議本体の venueid(例: `ICLR.cc/2024/Conference`)を持つため、`content.venueid` フィルタで採択絞り込みが実現できる。`venues.json` の `venue` フィールドの値がそのまま使える。
- レスポンスの `count` フィールド(フィルタ一致の全体件数)を返り値に含める(`APIResponse` 構造体に既存)。
- 認証(`Login`)・User-Agent の扱いは現行のまま。

### 2. 選定フロー (`cmd/dailybot/main.go`)

1. `GetAcceptedNotes(venue, 1, 0)` で **count** を取得する。count が 0 なら「候補なし」として正常終了(既存の `ErrNoCandidates` 時と同じ挙動)。
2. ランダムな offset を `[0, max(0, count-20)]` から選び、`GetAcceptedNotes(venue, 20, offset)` で **20 件の窓**を取得する。
3. 取得した 20 件を既存の `selector.RandomSelector` に渡してランダムに 1 件選定する。タイトル欠損などの無効データ除外は既存ロジックがそのまま担う。

窓方式(20 件)にする理由: 1 件だけ取得すると無効データ時のリトライ処理が必要になるが、20 件窓なら既存 selector の除外ロジックで自然に吸収でき、リトライ不要。通信量は数十 KB 程度で済む。

### 3. エラー処理

- count == 0: 候補なしログを出して正常終了(Slack への投稿なし)。
- API エラー(HTTP 非 200 等): 現行どおりエラーで異常終了し、Actions のログで気付く。

### 4. テスト

- `openreview_test.go`: `httptest` を使い、新メソッドのクエリパラメータ(`content.venueid`, `limit`, `offset`)と count の取り扱いを検証する(既存テストのパターンを踏襲)。
- 既存の `GetNotes` 用テストは新メソッド用に書き換える。
- `go test ./...` 全パスを確認する。

## 動作検証の制約

未認証の API v2 アクセスは bot 対策(ChallengeRequiredError)で弾かれる場合がある。実際の疎通確認には次のいずれかが必要:

- `.env` に `OR_EMAIL` / `OR_PASSWORD` を設定してローカルで `DRY_RUN=true` 実行(推奨)
- GitHub Actions での確認(ただし現行 workflow は `DRY_RUN: "false"` 固定のため本番投稿される)

## 影響しないもの

- `venues.json` のスキーマ、`config` パッケージ
- `formatter`(ヘッダーの `会議名 + 年` 表示は `VenueConfig` 由来のため過年度でもそのまま正しく動く)
- `venueselector`, `notifier`, `translator`
