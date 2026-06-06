# DPB-015 GAS workflow_dispatch Trigger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** GitHub Actions のスケジュール遅延を回避するため、Google Apps Script から `workflow_dispatch` で `daily.yml` を起動する構成に切り替え、既存 cron は JST 13:00 狙いのフォールバックとして残す。

**Architecture:** リポジトリには GAS にコピペするためのリファレンス実装 `scripts/gas/trigger.gs` を新規追加するだけで、Go アプリ本体には一切手を入れない。`.github/workflows/daily.yml` は `schedule:` の cron を後ろ倒しするだけ。残りは README / GEMINI.md / DPB-015.md の文書更新で完結する。

**Tech Stack:** Google Apps Script (V8 / ES6+), GitHub Actions, GitHub REST API v3 (`POST /repos/{owner}/{repo}/actions/workflows/{workflow_id}/dispatches`), Slack Incoming Webhooks。

---

## ファイル構成

| 状態 | パス | 役割 |
| ---- | ---- | ---- |
| 新規 | `scripts/gas/trigger.gs` | GAS にコピペする時間トリガー本体。冒頭にセットアップ手順をコメントで含める |
| 修正 | `.github/workflows/daily.yml` | `schedule:` の cron を `0 4 * * *` (JST 13:00 狙い) に変更し、コメントで「フォールバック」と明記 |
| 修正 | `README.md` | `## 実行` 配下に「定時起動の構成」サブセクションと GAS セットアップ手順を追加 |
| 修正 | `GEMINI.md` | `## 6. デプロイ` を GAS + cron フォールバックの構成に更新 |
| 修正 | `docs/tasks/v2/DPB-015.md` | 全チェックボックスを `- [x]` に、ステータスを `完了` に更新 |

### 自動テストの方針

- GAS スクリプトは `UrlFetchApp` / `PropertiesService` など Google Apps Script ランタイム固有の API に依存するため、Go のテストツリーや Node では実行できない。リポジトリ側で自動テストは追加しない。
- 代わりに **構文チェック** （`node --check` で関数定義レベルの parse 確認）と、**実機での手動検証**（GAS エディタからの手動実行 → `Actions` タブで Run が起動 → Slack に論文が投稿されることの確認）を Task 1 と Task 6 に含める。

### コミット粒度

Task 単位で 1 コミット。各タスク末尾の Commit ステップに具体的なメッセージ案を載せる。

---

## Task 1: GAS トリガースクリプトを追加する

**Files:**
- Create: `scripts/gas/trigger.gs`

- [ ] **Step 1: ディレクトリ確認**

Run: `ls scripts 2>/dev/null || echo "scripts directory does not exist"`
Expected: ディレクトリが無ければ `scripts directory does not exist`。あれば中身が一覧される。

- [ ] **Step 2: `scripts/gas/trigger.gs` を作成**

以下の内容で新規ファイルを作成する。冒頭の長いコメントブロックは GAS エディタに貼り付けた人がそのままセットアップ手順として読めるよう、削らずに保持する。

```javascript
/**
 * Daily Paper Bot — GitHub Actions trigger from Google Apps Script.
 *
 * このスクリプトは GitHub Actions の scheduled workflow 遅延を回避するため、
 * GAS の時間トリガーから `workflow_dispatch` API を叩いて `daily.yml` を
 * 起動するためのものです。リポジトリ側にはコピペ用のリファレンス実装として
 * 置いてあるだけで、CI からは参照されません。
 *
 * Setup:
 *   1. Create a new Google Apps Script project at https://script.google.com/.
 *   2. Paste the contents of this file into the editor (replacing Code.gs).
 *   3. Open Project Settings → Script Properties and add:
 *        - GITHUB_PAT         : Fine-grained PAT with `Actions: Read and write`
 *                               on the `daily-paper-bot` repository only.
 *        - SLACK_WEBHOOK_URL  : Slack Incoming Webhook URL for the channel
 *                               that receives the daily paper post.
 *   4. Project Settings → General settings: set timezone to "Asia/Tokyo".
 *   5. Triggers → Add Trigger:
 *        - Function:   trigger
 *        - Deployment: Head
 *        - Event:      Time-driven → Day timer → 8am to 9am
 *
 * On failure (non-204 HTTP status or thrown exception), the script posts a
 * short error message to the configured Slack Incoming Webhook so the
 * operator notices the outage on the same day.
 */

const REPO = 'hayashi-yaken/daily-paper-bot';
const WORKFLOW_FILE = 'daily.yml';
// Dispatch against the repository's default branch (currently `develop`),
// which is the branch the scheduled `daily.yml` already runs on. If the
// default branch changes on GitHub, update this to match.
const REF = 'develop';

function trigger() {
  const props = PropertiesService.getScriptProperties();
  const pat = props.getProperty('GITHUB_PAT');
  const webhookUrl = props.getProperty('SLACK_WEBHOOK_URL');

  if (!pat) {
    notifySlack(webhookUrl, ':warning: GAS trigger: GITHUB_PAT is not set in Script Properties.');
    return;
  }

  const url = 'https://api.github.com/repos/' + REPO + '/actions/workflows/' + WORKFLOW_FILE + '/dispatches';
  const options = {
    method: 'post',
    contentType: 'application/json',
    headers: {
      Authorization: 'Bearer ' + pat,
      Accept: 'application/vnd.github+json',
      'X-GitHub-Api-Version': '2022-11-28',
    },
    payload: JSON.stringify({ ref: REF }),
    muteHttpExceptions: true,
  };

  const ts = Utilities.formatDate(new Date(), 'Asia/Tokyo', 'yyyy-MM-dd HH:mm:ss z');

  try {
    const res = UrlFetchApp.fetch(url, options);
    const code = res.getResponseCode();
    if (code !== 204) {
      const body = res.getContentText().slice(0, 500);
      notifySlack(
        webhookUrl,
        ':warning: GAS trigger failed at ' + ts + ': HTTP ' + code + '\n```' + body + '```'
      );
    }
  } catch (e) {
    const message = e && e.message ? e.message : String(e);
    notifySlack(webhookUrl, ':warning: GAS trigger threw at ' + ts + ': ' + message);
  }
}

function notifySlack(webhookUrl, text) {
  if (!webhookUrl) {
    return;
  }
  try {
    UrlFetchApp.fetch(webhookUrl, {
      method: 'post',
      contentType: 'application/json',
      payload: JSON.stringify({ text: text }),
      muteHttpExceptions: true,
    });
  } catch (e) {
    // Final fallback: GAS execution log retains the failure trace.
  }
}
```

- [ ] **Step 3: 構文チェック**

Run: `node --check scripts/gas/trigger.gs`
Expected: 何も出力されず exit code 0。`UrlFetchApp` などの未定義参照は構文エラーにはならない（実行時に解決されるグローバル）。

- [ ] **Step 4: コミット**

```bash
git add scripts/gas/trigger.gs
git commit -m "$(cat <<'EOF'
feat(DPB-015): add GAS workflow_dispatch trigger script

Add scripts/gas/trigger.gs as a reference implementation to paste into a
Google Apps Script project. The script reads GITHUB_PAT and
SLACK_WEBHOOK_URL from Script Properties, calls workflow_dispatch on
daily.yml, and posts a Slack notification on non-204 responses or
exceptions.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: `daily.yml` の cron をフォールバック用に後ろ倒しする

**Files:**
- Modify: `.github/workflows/daily.yml:6`

- [ ] **Step 1: 現在の cron 行を確認**

Run: `grep -n "cron:" .github/workflows/daily.yml`
Expected: `6:    - cron: "0 0 * * *" # 毎日 00:00 UTC (JST 09:00) に実行`

- [ ] **Step 2: cron 行を `0 4 * * *` に変更し、コメントを更新**

Edit `.github/workflows/daily.yml` to replace:
```yaml
    - cron: "0 0 * * *" # 毎日 00:00 UTC (JST 09:00) に実行
```
with:
```yaml
    - cron: "0 4 * * *" # 毎日 04:00 UTC (JST 13:00) — フォールバック実行（プライマリは GAS からの workflow_dispatch）
```

- [ ] **Step 3: 変更を確認**

Run: `grep -n "cron:" .github/workflows/daily.yml`
Expected: `6:    - cron: "0 4 * * *" # 毎日 04:00 UTC (JST 13:00) — フォールバック実行（プライマリは GAS からの workflow_dispatch）`

- [ ] **Step 4: ワークフロー全体に構文エラーが入っていないことを確認**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/daily.yml'))" && echo OK`
Expected: `OK`

- [ ] **Step 5: コミット**

```bash
git add .github/workflows/daily.yml
git commit -m "$(cat <<'EOF'
chore(DPB-015): move schedule cron to JST 13:00 as fallback

Now that the GAS time trigger fires daily.yml at JST 08:00 via
workflow_dispatch, the schedule: cron exists only as a safety net for
days when GAS itself is broken. Move it from JST 09:00 to JST 13:00 so
that the two trigger paths don't fight for the same time slot.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: README に定時起動の構成と GAS セットアップ手順を追加する

**Files:**
- Modify: `README.md:79-82`

- [ ] **Step 1: 現状のセクション境界を確認**

Run: `sed -n '79,86p' README.md`
Expected:
```
### GitHub Actionsによる定期実行

`.github/workflows/daily.yml` に、毎日定刻にBotを実行するワークフローが定義されています。
本番運用では、リポジトリの `Settings > Secrets and variables > Actions` で必要な環境変数を設定してください。

---

## テスト
```

- [ ] **Step 2: `### GitHub Actionsによる定期実行` セクションを差し替える**

Edit `README.md` を以下のように変更する。`### GitHub Actionsによる定期実行` の行から、その下の `---` 行の直前までを丸ごと差し替える。

```markdown
### 定時起動の構成

毎日の論文投稿は以下の 2 経路で起動します。

- **プライマリ**: Google Apps Script の時間トリガーから GitHub の `workflow_dispatch` API を叩いて `daily.yml` を起動する（JST 08:00 起動）
- **フォールバック**: `.github/workflows/daily.yml` の `schedule:` で定義された cron (`0 4 * * *` = UTC 04:00 = JST 13:00 狙い)。Actions のスケジュール遅延を見越して GAS が止まった日でも午後には投稿が来るようにしてある

本番運用では、リポジトリの `Settings > Secrets and variables > Actions` で Bot 本体が必要とする環境変数を設定してください。

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
   - Function: `trigger`、Event source: 時間主導型、Type: 日付ベースのタイマー、Hour: 午前 8 時 〜 9 時
7. **動作確認**
   - GAS エディタから `trigger` を手動実行 → `Actions` タブで `Daily Paper Bot` Run が起動することを確認
   - 数分以内に Slack の投稿チャンネルへ論文が投稿されることを確認
   - 失敗時は Slack に `:warning: GAS trigger failed ...` の通知が来ることを確認したい場合は、`GITHUB_PAT` を一度ダミー文字列に書き換えて手動実行する（確認後は必ず元に戻す）
```

- [ ] **Step 3: 差し替えが正しいことを確認**

Run: `grep -n "^### \|^#### " README.md`
Expected: 一覧の中に `### 定時起動の構成` と `#### GAS トリガーのセットアップ` が含まれ、元の `### GitHub Actionsによる定期実行` は消えていること。

- [ ] **Step 4: コミット**

```bash
git add README.md
git commit -m "$(cat <<'EOF'
docs(DPB-015): document GAS trigger architecture and setup in README

Replace the brief "GitHub Actions schedule" subsection with a detailed
"定時起動の構成" section that explains both the GAS primary path and the
JST 13:00 cron fallback, then walks through the seven setup steps users
need to perform once in the Google Apps Script console.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: GEMINI.md のデプロイセクションを更新する

**Files:**
- Modify: `GEMINI.md:103-108`

- [ ] **Step 1: 現状のセクションを確認**

Run: `sed -n '103,108p' GEMINI.md`
Expected:
```
## 6. デプロイ

- デプロイは `.github/workflows/daily.yml` に定義されたGitHub Actionsによって完全に処理されます。
- ワークフローは毎日定刻に実行 (`cron`) されるほか、手動での実行 (`workflow_dispatch`) も可能です。
- 全てのシークレットは、リポジトリの「Settings > Secrets and variables > Actions」で設定する必要があります。
```

- [ ] **Step 2: セクションを差し替える**

Edit `GEMINI.md` の `## 6. デプロイ` セクションを以下に置き換える（直後の `## 7. 作業プロトコル` の手前まで）。

```markdown
## 6. デプロイ

- デプロイは `.github/workflows/daily.yml` に定義された GitHub Actions によって完全に処理されます。
- 定時起動は **プライマリ: Google Apps Script の時間トリガー → `workflow_dispatch` で `daily.yml` を起動 (JST 08:00)** という構成です。GAS にコピペするためのスクリプトは `scripts/gas/trigger.gs` を参照してください。
- フォールバックとして `schedule:` の cron が `0 4 * * *` (UTC 04:00 = JST 13:00 狙い) で残してあり、GAS が止まった日でも遅延込みで午後には投稿が来るようになっています。
- 手動実行は GitHub UI 上で `workflow_dispatch` から起動できます。
- Bot 本体のシークレットは、リポジトリの「Settings > Secrets and variables > Actions」で設定します。GAS 側で使う秘密情報 (`GITHUB_PAT` / `SLACK_WEBHOOK_URL`) は GAS の Script Properties で個別に管理します。
```

- [ ] **Step 3: 差し替え確認**

Run: `sed -n '103,112p' GEMINI.md`
Expected: 新しい本文が出力され、`## 7. 作業プロトコル` の見出しが新しい本文の直後に続いていること。

- [ ] **Step 4: コミット**

```bash
git add GEMINI.md
git commit -m "$(cat <<'EOF'
docs(DPB-015): update GEMINI deploy section for GAS-driven schedule

Reflect the new architecture where GAS is the primary trigger and the
GitHub Actions schedule cron is a JST 13:00 fallback, and add a pointer
to scripts/gas/trigger.gs for future agents that need to find the
trigger code.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: DPB-015 チケットを完了状態に更新する

**Files:**
- Modify: `docs/tasks/v2/DPB-015.md`

- [ ] **Step 1: ステータス行を確認**

Run: `grep -n "ステータス\|やること\|成果物" docs/tasks/v2/DPB-015.md`
Expected: 各見出しの行番号が表示される。

- [ ] **Step 2: ステータスを `完了` に変更**

Edit `docs/tasks/v2/DPB-015.md` で `| ステータス | 未着手` を `| ステータス | 完了` に置き換える（`replace_all` は使わず、ステータス行の周辺の table 行を含めて一意に特定して差し替える）。

- [ ] **Step 3: 「やること」のチェックボックスを完了にする**

Edit `docs/tasks/v2/DPB-015.md` で `## やること` セクション配下の全 `- [ ]` を `- [x]` に変更する。`Edit` ツールの `replace_all` で `- [ ]` → `- [x]` をかけると後段の「成果物」セクションも同時に切り替わるので、それで OK。

- [ ] **Step 4: 「成果物」セクションが `- [x]` 化されたことを確認**

Run: `grep -n "^- \[\]\|^- \[ \]" docs/tasks/v2/DPB-015.md`
Expected: 何も出力されない（残っている未完了チェックボックスがゼロ）。

- [ ] **Step 5: コミット**

```bash
git add docs/tasks/v2/DPB-015.md
git commit -m "$(cat <<'EOF'
docs(DPB-015): mark ticket completed

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: 手動検証（ユーザー作業）

このタスクはローカルでの自動化ができない実機確認なので、リポジトリへの変更はない。実装担当はユーザーに以下の検証を依頼し、結果を Slack 等で確認してから PR をマージする。

- [ ] **Step 1: GAS エディタから手動実行**

GAS のエディタで `trigger` 関数を選択し、`実行` ボタンを押す。

- [ ] **Step 2: GitHub Actions が起動したことを確認**

`https://github.com/hayashi-yaken/daily-paper-bot/actions/workflows/daily.yml` を開き、最新の Run が「Triggered via API」もしくは `workflow_dispatch` で数秒前に起動していることを確認する。

- [ ] **Step 3: Slack 投稿を確認**

設定したチャンネルに 1〜2 分以内に論文の投稿が来ることを確認する。

- [ ] **Step 4: 時間トリガーを有効化したまま翌朝に確認**

翌日の JST 08:00 〜 08:15 の間に自動で投稿が来ることを確認する。来なかった場合は GAS の `Executions` ログを開き、`trigger` 関数の失敗内容を見る。

- [ ] **Step 5: 失敗時の Slack 通知パスを軽く検証（任意）**

`GITHUB_PAT` をダミー文字列 (`dummy_token_for_test`) に一時的に書き換えて `trigger` を手動実行し、Slack に `:warning: GAS trigger failed ... HTTP 401` の通知が来ることを確認する。確認後は元の値に戻す。

---

## Self-Review チェックリスト（実装担当向け）

最終 PR を作る前に以下を確認すること。

- [ ] `scripts/gas/trigger.gs` がリポジトリに含まれ、`node --check` で構文エラーが出ない
- [ ] `.github/workflows/daily.yml` の `cron` が `0 4 * * *` でコメントが「フォールバック」と明記されている
- [ ] `README.md` の `### 定時起動の構成` セクションが存在し、GAS セットアップ手順 7 ステップが揃っている
- [ ] `GEMINI.md` の `## 6. デプロイ` がプライマリ/フォールバックの構成を述べている
- [ ] `docs/tasks/v2/DPB-015.md` のステータスが `完了`、全チェックボックスが `- [x]`
- [ ] `git log --oneline` で Task 1〜5 の 5 コミットが並んでいる
- [ ] 手動検証（Task 6）が実機で完了している
