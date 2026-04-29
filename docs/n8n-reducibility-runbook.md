# n8n Reducibility Runbook

## Goal

Evaluate whether selected n8n workflow examples can be expressed as Ramen `intent.hcl`, then lowered through Ramen into udon `workflow.hcl` and UWS artifacts.

Use `../w8m` only as a read-only scanner and evidence source. Keep actual Ramen examples, generated artifacts, quality reports, and follow-up work in this repo.

## Source Inputs

`../w8m/reducibility/specs/` currently contains scanner-owned OpenAPI evidence for these providers:

- `airtable.json`
- `google_drive.json`
- `jira.json`
- `pagerduty.json`
- `trello.json`
- `gmail.json`
- `hubspot.json`
- `openweathermap.json`
- `slack.json`

Upstream n8n workflow examples exist in `../n8n` for eight of those nine families: Airtable, Google Drive, PagerDuty, Trello, Gmail, HubSpot, OpenWeatherMap, and Slack. Jira has node schemas and tests in `../n8n`, but no workflow JSON fixture was found in the initial pass.

Useful fixture locations:

- `../n8n/packages/nodes-base/nodes/Airtable/test/workflow.json`
- `../n8n/packages/nodes-base/nodes/Google/Drive/v2/test/*.workflow.json`
- `../n8n/packages/nodes-base/nodes/Google/Gmail/test/v1/*.workflow.json`
- `../n8n/packages/nodes-base/nodes/Google/Gmail/test/v2/*.workflow.json`
- `../n8n/packages/nodes-base/nodes/Hubspot/__test__/*.workflow.json`
- `../n8n/packages/nodes-base/nodes/OpenWeatherMap/test/workflow.json`
- `../n8n/packages/nodes-base/nodes/Slack/test/v1/**/*.workflow.json`
- `../n8n/packages/nodes-base/nodes/Slack/test/v2/**/*.workflow.json`
- `../n8n/packages/testing/playwright/tests/cli-workflows/workflows/10.json` for PagerDuty
- `../n8n/packages/testing/playwright/tests/cli-workflows/workflows/56.json` for Trello

## First Slice

The first curated eval samples cover every OpenAPI evidence file under `../w8m/reducibility/specs/`:

- `n8n-airtable-record-get`: Airtable `record/get` -> `getAirtableRecord`
- `n8n-gmail-message-send`: Gmail `message/send` -> `sendMessage`
- `n8n-google-drive-file-upload`: Google Drive `file/upload` -> `uploadFile`
- `n8n-hubspot-deal-list`: HubSpot `deal/getAll` -> `listDeals`
- `n8n-jira-issue-get`: scanner-backed Jira `issue/get` -> `getIssue`
- `n8n-openweathermap-current-weather`: OpenWeatherMap current weather -> `getOpenWeatherMapCurrentWeather`
- `n8n-pagerduty-user-get`: PagerDuty `user/get` -> `getUser`
- `n8n-slack-message-post`: Slack `message/post` -> `postMessage`
- `n8n-trello-list-get-all`: Trello `list/getAll` -> `listTrelloBoardLists`

All n8n reducibility fixtures start with advisory reference policy while the fixture shape settles.

Avoid helper-heavy or n8n-specific behavior in the first pass, such as `sendAndWait`, trigger registration, binary follow-up downloads, UI lookup sidecars, or paired-item semantics.

## Manual Evaluation Flow

From `../w8m`, scan a source n8n workflow:

```bash
go run ./cmd/w8mscan \
  --workflow ../n8n/packages/nodes-base/nodes/Slack/test/v1/node/message/post.workflow.json \
  --format json
```

Create or inspect the Ramen eval example:

```bash
cd ../ramen
find examples/eval/n8n-slack-message-post -maxdepth 3 -type f | sort
```

For local validation, copy the reference intent into a temporary working artifact path and remove generated artifacts afterward:

```bash
mkdir -p examples/eval/n8n-slack-message-post/workflows
cp examples/eval/n8n-slack-message-post/reference/intent.hcl \
  examples/eval/n8n-slack-message-post/workflows/intent.hcl

go run ./cmd/ramen build --example examples/eval/n8n-slack-message-post
go run ./cmd/ramen promote --example examples/eval/n8n-slack-message-post
go run ./cmd/ramen assess --example examples/eval/n8n-slack-message-post

rm -rf examples/eval/n8n-slack-message-post/workflows \
  examples/eval/n8n-slack-message-post/expected
```

Run the corpus eval command when LLM-backed eval credentials are available:

```bash
go run ./cmd/ramen eval --root examples/eval --name n8n-slack-message-post --no-compare
```

## Evaluation Matrix

Track each attempted n8n fixture with:

| Field | Meaning |
| --- | --- |
| Source workflow | Path under `../n8n` |
| Provider | Slack, Gmail, etc. |
| n8n node type | Example: `n8n-nodes-base.slack` |
| n8n version | `typeVersion` from the node |
| n8n resource | `parameters.resource`, if present |
| n8n operation | `parameters.operation`, if present |
| OpenAPI file | Copied file under example `openapi/` |
| Operation ID | Ramen `operation` value |
| Intent result | Parses / build fails / needs manual mapping |
| Workflow result | `workflow.hcl` generated / UWS generated / assess pass |
| Notes | Missing schema, binary mapping, pagination, trigger lifecycle, etc. |

## Interpretation Rules

- If a hand-authored `intent.hcl` builds and assesses, the n8n example is likely expressible as Ramen intent for the tested slice.
- If `intent.hcl` needs a local `fnct` step for data shaping, record that as helper-heavy rather than an OpenAPI failure.
- If the OpenAPI evidence is missing an operation used by an n8n example, either extend the local OpenAPI file for the Ramen example or record it as an evidence gap.
- If the n8n example relies on trigger setup, credentials UI, dynamic lookups, binary envelopes, send-and-wait approval, or paired-item semantics, treat that as Ramen modeling work beyond a direct OpenAPI operation.
- Do not add n8n-specific runtime behavior to Ramen or udon. Prefer explicit intent, OpenAPI, and generic `fnct` or control-flow modeling.
