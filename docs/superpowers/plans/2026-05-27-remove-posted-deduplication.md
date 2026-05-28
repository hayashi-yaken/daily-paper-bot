# Remove Posted-Paper Deduplication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** "一度投稿された論文は再投稿しない" 重複防止機構を完全に廃止する。`internal/storage` パッケージ・`data/posted.json`・GitHub Actions のコミットステップを含め、関連コード/データ/ドキュメントを一式撤去する。

**Architecture:** `storage.JSONStorage` は `cmd/dailybot/main.go` で生成され、(1) `selector.NewRandomSelector` に `IsPosted` をコールバック注入し、(2) 投稿成功後に `Add` + `Save` で `data/posted.json` を更新する、という形で参照されている。GitHub Actions ワークフローは更新された JSON を自動コミット & プッシュしている。これら 3 つの依存リンクを断ち切り、`internal/storage` パッケージと `data/posted.json` を物理削除する。`selector.RandomSelector` は「全候補からランダムに 1 本選ぶ」だけのシンプルな実装に縮退させる。

**Tech Stack:** Go 1.22, 標準ライブラリ中心, `joho/godotenv`, GitHub Actions (`stefanzweifel/git-auto-commit-action`)

---

## File Structure

**削除（パッケージ丸ごと）:**
- `internal/storage/json_storage.go`
- `internal/storage/json_storage_test.go`
- `internal/storage/record.go`
- `data/posted.json`
- `data/` ディレクトリ自体（中身が空になるため）

**修正:**
- `internal/selector/random.go` — `isPosted` コールバックを除去、`NewRandomSelector` を引数なしに、Select ロジックを単純化
- `internal/selector/random_test.go` — `isPosted` 関連テストを削除、新シグネチャに合わせて書き直し
- `cmd/dailybot/main.go` — `storage` import / 初期化 / 投稿後の Add+Save を削除、`NewRandomSelector()` 呼び出しを引数なしに
- `.github/workflows/daily.yml` — `Commit and push if changed` ステップと `permissions.contents: write` を削除
- `README.md` — 「投稿済み除外」の記述を更新
- `GEMINI.md` — 永続化セクションと storage 関連の記述を更新
- `docs/PROJECT_SPEC.md` — 重複防止セクション・関連記述を「廃止済」に更新
- `.serena/memories/code_layout.md` — `data/` 説明から `posted.json` への言及を削除

各タスクは自己完結 (一度のコミット可能) になるよう構成。

---

### Task 1: selector を独立した「投稿済みフィルタ無し」実装にリファクタリング (TDD)

**Files:**
- Modify: `internal/selector/random.go`
- Test: `internal/selector/random_test.go`

- [ ] **Step 1: 既存テストを新仕様に書き直し（赤）**

`internal/selector/random_test.go` を以下の内容に置き換える。`isPosted` 関連ヘルパーとテストケースを削除し、「不正データ除外」「ランダム選定」「候補ゼロ」のみ残す。

```go
package selector

import (
	"errors"
	"math/rand"
	"testing"
)

// MockPaper はテスト用のPaper実装です。
type MockPaper struct {
	id    string
	title string
}

func (m *MockPaper) GetID() string {
	return m.id
}

func (m *MockPaper) GetTitle() string {
	return m.title
}

func TestRandomSelector_Select(t *testing.T) {
	papers := []Paper{
		&MockPaper{id: "p1", title: "Title 1"},
		&MockPaper{id: "p2", title: "Title 2"},
		&MockPaper{id: "p3", title: "Title 3"},
		&MockPaper{id: "p4", title: ""},      // Invalid title
		&MockPaper{id: "", title: "Title 5"}, // Invalid ID
	}

	t.Run("select one from valid candidates", func(t *testing.T) {
		selector := NewRandomSelector()
		// 乱数を固定
		selector.rand = rand.New(rand.NewSource(0))

		selected, err := selector.Select(papers)
		if err != nil {
			t.Fatalf("Select() returned an error: %v", err)
		}

		// 有効候補は p1, p2, p3 の 3 件。seed 0 では Intn(3)=0 なので p1 が選ばれる
		expectedID := "p1"
		if selected.GetID() != expectedID {
			t.Errorf("expected paper %s, but got %s", expectedID, selected.GetID())
		}
	})

	t.Run("no candidates because of invalid data", func(t *testing.T) {
		papersOnlyInvalid := []Paper{
			&MockPaper{id: "", title: "Title 1"},
			&MockPaper{id: "p2", title: ""},
		}
		selector := NewRandomSelector()

		_, err := selector.Select(papersOnlyInvalid)
		if !errors.Is(err, ErrNoCandidates) {
			t.Errorf("expected ErrNoCandidates, but got %v", err)
		}
	})

	t.Run("no papers provided", func(t *testing.T) {
		selector := NewRandomSelector()
		_, err := selector.Select([]Paper{})
		if !errors.Is(err, ErrNoCandidates) {
			t.Errorf("expected ErrNoCandidates, but got %v", err)
		}
	})
}
```

- [ ] **Step 2: テストを走らせて失敗を確認**

Run: `go test ./internal/selector/... -v`
Expected: コンパイルエラー (`NewRandomSelector` の引数不一致 / `ErrNilIsPosted` 参照削除など) で FAIL。

- [ ] **Step 3: `random.go` を新仕様に書き換える**

`internal/selector/random.go` 全体を以下に置き換える。

```go
package selector

import (
	"errors"
	"math/rand"
	"time"
)

var ErrNoCandidates = errors.New("no candidates to select from")

// RandomSelector はランダムに論文を選定するセレクターです。
type RandomSelector struct {
	// rand はテストで乱数生成を固定できるようにインターフェースとして保持します。
	rand *rand.Rand
}

// NewRandomSelector は新しいRandomSelectorを生成します。
func NewRandomSelector() *RandomSelector {
	return &RandomSelector{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select は論文リストから必須項目が揃ったものをフィルタリングし、ランダムに1本を選定します。
func (s *RandomSelector) Select(papers []Paper) (Paper, error) {
	var candidates []Paper

	// 必須項目をチェックします。
	for _, p := range papers {
		if p == nil {
			continue
		}
		if p.GetID() == "" || p.GetTitle() == "" {
			continue // データ不整合はスキップ
		}
		candidates = append(candidates, p)
	}

	if len(candidates) == 0 {
		return nil, ErrNoCandidates
	}

	// ランダムに1本選定します。
	selectedIndex := s.rand.Intn(len(candidates))

	return candidates[selectedIndex], nil
}
```

- [ ] **Step 4: テストを走らせて緑を確認**

Run: `go test ./internal/selector/... -v`
Expected: 3 件すべて PASS。

- [ ] **Step 5: コミット**

```bash
git add internal/selector/random.go internal/selector/random_test.go
git commit -m "refactor(selector): drop isPosted filter from RandomSelector"
```

---

### Task 2: `cmd/dailybot/main.go` から storage 依存を撤去

**Files:**
- Modify: `cmd/dailybot/main.go`

- [ ] **Step 1: import から storage を削除**

`cmd/dailybot/main.go` の import ブロック内の以下の行を削除する。

```go
"github.com/hayashi-yaken/daily-paper-bot/internal/storage"
```

- [ ] **Step 2: storage 初期化と selector 注入を書き換え**

`cmd/dailybot/main.go` の以下のブロック (Step 1 適用前の 53〜57 行目相当):

```go
	jsonStorage, err := storage.NewJSONStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	paperSelector := selector.NewRandomSelector(jsonStorage.IsPosted)
```

を、以下の 1 行に置き換える。

```go
	paperSelector := selector.NewRandomSelector()
```

- [ ] **Step 3: 投稿成功後の保存処理を削除**

`cmd/dailybot/main.go` の以下のブロック (投稿成功ログの直後):

```go
	log.Println("INFO: Saving posted record...")
	jsonStorage.Add(selectedPaper.GetID(), selectedVenue.Venue)
	if err := jsonStorage.Save(); err != nil {
		return fmt.Errorf("failed to save posted record: %w", err)
	}
	log.Println("INFO: Record saved.")
```

を完全に削除する（後続の `return nil` まで何も挟まないようにする）。

- [ ] **Step 4: ビルドが通ることを確認**

Run: `go build ./...`
Expected: エラーなし (`storage` 未使用 import の警告も出ない)。

- [ ] **Step 5: 単体テストが通ることを確認**

Run: `go test ./... -count=1`
Expected: `internal/storage` パッケージのテストはまだ通る (Task 3 で削除する)。それ以外もすべて PASS。

- [ ] **Step 6: コミット**

```bash
git add cmd/dailybot/main.go
git commit -m "refactor(cmd): remove storage wiring from dailybot entrypoint"
```

---

### Task 3: `internal/storage` パッケージを物理削除

**Files:**
- Delete: `internal/storage/json_storage.go`
- Delete: `internal/storage/json_storage_test.go`
- Delete: `internal/storage/record.go`

- [ ] **Step 1: パッケージディレクトリごと削除**

```bash
rm -rf internal/storage
```

- [ ] **Step 2: ディレクトリが削除されたことを確認**

```bash
ls internal/
```

Expected: `storage` が表示されないこと。

- [ ] **Step 3: 全テスト + ビルド再実行**

```bash
go build ./... && go test ./... -count=1
```

Expected: ビルド成功、`storage` への参照が残っていないので全テスト PASS。

- [ ] **Step 4: コミット**

```bash
git add -A internal/storage
git commit -m "refactor(storage): delete posted-paper persistence package"
```

---

### Task 4: GitHub Actions ワークフローから posted.json 関連を撤去

**Files:**
- Modify: `.github/workflows/daily.yml`

- [ ] **Step 1: `permissions` ブロックを削除**

`.github/workflows/daily.yml` の以下のブロック (10〜12 行目):

```yaml
    permissions:
      contents: write # posted.json をコミットするためにリポジトリへの書き込み権限が必要

```

を完全に削除する（書き込み権限が不要になるため）。

- [ ] **Step 2: 自動コミットステップを削除**

`.github/workflows/daily.yml` の以下のブロック (41〜48 行目相当):

```yaml
      - name: Commit and push if changed
        uses: stefanzweifel/git-auto-commit-action@v5
        with:
          commit_message: "chore(bot): Update posted papers"
          file_pattern: "data/posted.json"
          commit_user_name: "github-actions[bot]"
          commit_user_email: "github-actions[bot]@users.noreply.github.com"
          commit_author: "github-actions[bot] <github-actions[bot]@users.noreply.github.com>"
```

を完全に削除する。

- [ ] **Step 3: 最終的なワークフロー内容を確認**

Run: `cat .github/workflows/daily.yml`
Expected: `permissions:` ブロックと `Commit and push if changed` ステップが存在しないこと。`Run Bot` ステップが最後のステップになっていること。

- [ ] **Step 4: コミット**

```bash
git add .github/workflows/daily.yml
git commit -m "ci: drop posted.json auto-commit step and write permission"
```

---

### Task 5: `data/posted.json` と `data/` ディレクトリを削除

**Files:**
- Delete: `data/posted.json`
- Delete: `data/`

- [ ] **Step 1: ファイルとディレクトリを削除**

```bash
rm -rf data
```

- [ ] **Step 2: 削除確認**

```bash
ls -la | grep -E "^d.* data$" || echo "data dir gone"
```

Expected: `data dir gone` と表示される。

- [ ] **Step 3: コミット**

```bash
git add -A data
git commit -m "chore: remove obsolete data/posted.json store"
```

---

### Task 6: README / GEMINI / PROJECT_SPEC / serena memory を更新

**Files:**
- Modify: `README.md`
- Modify: `GEMINI.md`
- Modify: `docs/PROJECT_SPEC.md`
- Modify: `.serena/memories/code_layout.md`

- [ ] **Step 1: README.md の機能説明を更新**

`README.md` の 9 行目周辺を確認:

```bash
grep -n "投稿済み" README.md
```

該当行 (現状):

```
- 投稿済みの論文を除外して、未投稿の中から1本をランダムに選定
```

を以下に置き換える:

```
- 取得した論文の中からランダムに1本を選定
```

- [ ] **Step 2: GEMINI.md の永続化記述を削除/更新**

`GEMINI.md` の以下の行を削除する:

- 17 行目: `- **永続化**: リポジトリにコミットされる単一の `data/posted.json` ファイル`
- 28 行目: `  - `storage/`: 重複投稿を防ぐため、`data/posted.json` の読み書きを管理。`
- 34 行目: `- `data/`: 投稿済み論文を記録する `posted.json` ファイルを格納します。`
- 105 行目: `- 実行成功後、ワークフローは `data/posted.json` への変更を自動的にコミット＆プッシュし、状態を更新します。`

(削除位置は `grep -n "posted\|永続化\|storage" GEMINI.md` で再確認してから編集する)

- [ ] **Step 3: docs/PROJECT_SPEC.md に廃止メモを追記し、矛盾箇所を更新**

`docs/PROJECT_SPEC.md` の冒頭 (1〜2 行目の見出し直下) に以下のセクションを追加する。

```markdown
> **更新 (2026-05-27)**: 「10. 重複投稿防止 (永続化)」は廃止されました。以下の項目は本仕様から除外されます:
> - `data/posted.json` による重複防止
> - 投稿済みフィルタリング
> - GitHub Actions による `data/posted.json` の自動コミット&プッシュ
> Bot は実行のたびに全候補からランダムに 1 本を選定します (再投稿が許容される運用)。
```

- [ ] **Step 4: .serena/memories/code_layout.md を更新**

`.serena/memories/code_layout.md` の以下の行:

```
- **`data/`**: Stores persistent data, such as the `posted.json` file for deduplication.
```

を削除する。

- [ ] **Step 5: 更新後の grep 確認**

Run: `grep -rn -E "posted\.json|IsPosted|投稿済み" README.md GEMINI.md docs/PROJECT_SPEC.md .serena/memories/code_layout.md`
Expected: 残存ヒットは `docs/PROJECT_SPEC.md` 内の「廃止済み」と明記された箇所のみ。それ以外に "投稿済みを除外する" など現役感のある記述が残っていないこと。

- [ ] **Step 6: コミット**

```bash
git add README.md GEMINI.md docs/PROJECT_SPEC.md .serena/memories/code_layout.md
git commit -m "docs: remove references to deprecated posted.json deduplication"
```

---

### Task 7: 仕上げの検証 (ビルド / 単体テスト / dry-run)

**Files:** (検証のみ、修正なし)

- [ ] **Step 1: 全ビルド**

Run: `go build ./...`
Expected: エラーなし。

- [ ] **Step 2: 全単体テスト**

Run: `go test ./... -count=1`
Expected: すべて PASS。`internal/storage` パッケージが消えているので対象テストは出てこない。

- [ ] **Step 3: dry-run 実行**

`.env` に `DRY_RUN="true"` が設定されている前提で以下を実行する (設定されていない場合は一時的に追加)。

Run: `DRY_RUN=true go run ./cmd/dailybot`
Expected:
- ログに `INFO: Selecting a paper...` と `INFO: Selected paper: ...` が出る
- ログに `INFO: Saving posted record...` が **出ない**
- ログに `Dry run mode is enabled. Skipping post and save.` が出て正常終了

- [ ] **Step 4: 作業ツリー確認**

Run: `git status`
Expected: untracked / modified が無いか、想定外の変更が無いことを確認。

- [ ] **Step 5: ワーキングブランチ全体の差分確認**

Run: `git log --oneline main..HEAD`
Expected: Task 1〜6 で作った 6 つのコミットが綺麗に並んでいる。

---

## Self-Review チェック結果

- **Spec coverage:** Plan B で挙げた範囲 (selector / storage / main.go / GitHub Actions / posted.json / docs 4 ファイル / dry-run 動作確認) を Task 1〜7 で全カバー済み。
- **Placeholder scan:** "TBD" / "後で書く" / 抽象的な "適切なエラーハンドリング" 等は無し。各コード変更は実コードを記載済み。
- **Type consistency:** `NewRandomSelector()` (引数なし) が Task 1 / Task 2 で一貫している。`ErrNoCandidates` は残し、`ErrNilIsPosted` は Task 1 で削除して以降は参照無し。
