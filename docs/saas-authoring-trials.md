# SaaS Authoring Trials

M20 measures the end-to-end authoring path after the SaaS authoring, corpus,
guided UX, multi-service pattern, and trusted-handoff milestones.

The trial target is deterministic package evidence, not live-provider
execution. Trial runs use existing `project.md` briefs and checked-in
`reference/intent.hcl` files as the documented manual equivalent of a completed
guided authoring session, then run normal OpenUdon build, review, approval, and
dry-run gates in ignored local workdirs.

## Trial Harness

Use ignored local output under `.openudon-run/m20-trials`:

```bash
mkdir -p .openudon-run/m20-trials/approvals
cp -R examples/eval/<fixture> .openudon-run/m20-trials/<fixture>
mkdir -p .openudon-run/m20-trials/<fixture>/workflows
cp .openudon-run/m20-trials/<fixture>/reference/intent.hcl \
  .openudon-run/m20-trials/<fixture>/workflows/intent.hcl

go run ./cmd/icot lint --example ./examples/eval/<fixture>
go run ./cmd/openudon build --example ./.openudon-run/m20-trials/<fixture>
go run ./cmd/openudon approval-template \
  --example ./.openudon-run/m20-trials/<fixture> \
  --state approved_for_sandbox \
  --reviewer "M20 Trial" \
  > .openudon-run/m20-trials/approvals/<fixture>.json
go run ./cmd/openudon run \
  --example ./.openudon-run/m20-trials/<fixture> \
  --tier sandbox \
  --approval .openudon-run/m20-trials/approvals/<fixture>.json \
  --workdir .openudon-run/m20-trials/workdir-<fixture> \
  --dry-run
```

Do not commit `.openudon-run` output or approval JSON. The committed evidence is
the matrix below plus any fixture/doc repairs promoted from the trial.

## Trial Matrix

| Fixture | Service Family | Policy | iCoT Lint | Build/Assess | Approval Dry Run | Result |
| --- | --- | --- | --- | --- | --- | --- |
| `slack-message-audit-log` | Slack | strict native | pass | pass | pass | Converges after fixing renderer input wording in `project.md`. |
| `gmail-send-audit-receipt` | Gmail | strict native | pass | pass | pass | Converges; good single-service side-effect demo. |
| `weather-toronto` | OpenWeatherMap | native/reference | pass | pass | pass | Converges; useful read-style credential binding demo. |
| `itops-slack-jira-issue-intake` | Slack/Jira | strict multi-service | pass | fail | not run | OpenAPI/security mapping gap: `Authorization` is not declared by `getSlackMessage`. |
| `itops-incident-response-archive` | Jira/Slack/Drive | strict multi-service | pass | fail | not run | OpenAPI/security mapping gap: `Authorization` is not declared by `createIssue`. |
| `order-fulfillment-chain` | Customer/Inventory/Order | strict multi-service | pass | fail | not run | OpenAPI/security mapping gap: `Authorization` is not declared by `getCustomer`. |
| `n8n-hubspot-deal-list` | HubSpot | advisory n8n-derived | pass | fail | not run | Copied OpenAPI/request mapping gap: `limit` is not declared by `listDeals`. |
| `n8n-airtable-record-get` | Airtable | advisory n8n-derived | pass | fail | not run | Copied OpenAPI/request mapping gap: `baseId` is not declared by `getAirtableRecord`. |

## Gap Matrix

| Gap Class | Evidence | Follow-Up |
| --- | --- | --- |
| Prompt/readiness | All selected briefs pass `icot lint`; no M20 blocking prompt/readiness gap was found. | Keep current guided questions as the baseline for M21. |
| Function contract wording | `slack-message-audit-log` initially failed because `Inputs: ok, channel, and ts` did not parse as the explicit `ts` input. | Fixed wording to `Inputs: ok, channel, ts`; keep terse comma-separated function inputs in strict fixtures. |
| OpenAPI security mapping | Slack/Jira, incident archive, and order fulfillment fail when reference intents bind `Authorization` but request placement does not expose it as an operation field. | M21 should normalize security binding conventions in strict multi-service fixtures or improve security-field placement support. |
| Copied OpenAPI parameter shape | HubSpot and Airtable advisory fixtures fail because copied OpenAPI operation metadata does not expose the request fields used by the reference intent. | Keep advisory; graduate only after local OpenAPI slices and request mappings are normalized. |
| Review/handoff | Passing package trials have M19 review checks and trusted-runner dry-run evidence. | Reuse M19 evidence shape for promoted M21 fixtures. |
| Live-provider behavior | Not tested. | Remains local/manual release evidence only. |

## Promotion Candidates

Immediate M21 candidates:

- `slack-message-audit-log`
- `gmail-send-audit-receipt`
- `weather-toronto`

Repair-first M21 candidates:

- `itops-slack-jira-issue-intake`
- `itops-incident-response-archive`
- `order-fulfillment-chain`

Keep advisory until normalized:

- `n8n-hubspot-deal-list`
- `n8n-airtable-record-get`

The main M21 implementation priority should be security/request-field
normalization for strict multi-service packages before increasing fixture
volume.
