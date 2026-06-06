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
