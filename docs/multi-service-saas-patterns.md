# Multi-Service SaaS Patterns

M18 documents the repeatable SaaS workflow chains OpenUdon should keep
reviewable and deterministic. The goal is not to add new runtime semantics; it
is to make cross-service bindings, credential scopes, side effects, and
graduation policy explicit enough for authoring and review.

## Pattern Inventory

| Pattern | Strict Or Reference Fixture | What It Proves | Status |
| --- | --- | --- | --- |
| Lookup then notify | `response-field-ticket-alert` | API lookup response fields feed an approved alert adapter. | Strict single-service plus adapter pattern. |
| Ticket then message | `itops-slack-jira-issue-intake` | Slack fetch and parse output feeds Jira create, then Jira response feeds Slack confirmation. | Strict multi-service SaaS golden fixture. |
| Send then audit | `gmail-send-audit-receipt`, `slack-message-audit-log` | Provider send/post response feeds local audit rendering. | Strict single-service authoring pattern. |
| Ticket, alert, then archive | `itops-incident-response-archive` | Jira issue response feeds Slack alert and Drive archive upload. | Strict multi-service SaaS golden fixture. |
| Paginate then summarize | `cursor-pagination-report`, `page-token-pagination-export`, `offset-pagination-export` | Page/cursor/offset response data feeds later reads and local merge/report steps. | Strict data-flow pattern coverage. |
| Webhook send, then later create/update | `retry-idempotent-webhook-send` | Idempotent outbound webhook send with timeout and approval posture. | Strict send coverage now; inbound webhook-to-create/update remains future scope. |
| Lookup then create | `order-fulfillment-chain` | Customer and inventory reads feed sandbox fulfillment order creation. | Strict multi-service SaaS golden fixture. |

## Strict Multi-Service Set

The M18 strict multi-service set is:

- `itops-slack-jira-issue-intake`
- `itops-incident-response-archive`
- `order-fulfillment-chain`

These fixtures are release-relevant because they exercise different cross-step
contracts:

- one service's response fields become another service's request fields;
- each service keeps its own symbolic credential binding;
- side-effectful create/post/upload steps stay behind review and trusted-runner
  approval;
- local `fnct` steps are only deterministic transforms, renderers, or approved
  adapters.

## Cross-Service Data Flow

Cross-service data flow should be visible in both `project.md` and
`reference/intent.hcl`.

| Fixture | Cross-Service Mapping |
| --- | --- |
| `itops-slack-jira-issue-intake` | `get_slack_message.received_body.message.text` feeds `parse_issue_report`; parsed title/description/priority/type feed `create_jira_issue`; `create_jira_issue.received_body.key` feeds Slack confirmation. |
| `itops-incident-response-archive` | `create_jira_incident.received_body.key/self` feed Slack alert and timeline rendering; rendered report name/content feed Drive upload. |
| `order-fulfillment-chain` | customer email/address and inventory SKU/warehouse feed fulfillment order creation. |

If a later service request depends on a prior service response, use explicit
`bind` blocks and keep `depends_on` on the consumer step. Do not encode hidden
cross-service coupling only in prose.

## Credential Scopes

Multi-service workflows must keep per-service credential bindings distinct.

| Fixture | Credential Bindings |
| --- | --- |
| `itops-slack-jira-issue-intake` | `slack_bot_token`, `jira_api_token` |
| `itops-incident-response-archive` | `jira_api_token`, `slack_bot_token`, `google_drive_oauth_token` |
| `order-fulfillment-chain` | `customers_bearer_token`, `inventory_api_key`, `orders_bearer_token` |

Do not reuse one binding name across services unless the project brief
explicitly documents a shared credential boundary. Never include credential
values in project briefs, reference intents, metadata, review evidence, or
approval files.

## Side-Effect Policy

The strict multi-service fixtures cover these side effects:

- Jira issue create;
- Slack message post;
- Google Drive file upload;
- fulfillment order create.

All are review-first. The fixtures may be built, assessed, and linted without
real provider credentials. Execution requires an approved package, symbolic
bindings resolved by the trusted runner, and the usual `openudon run` approval
checks.

## Known Gaps

- Inbound webhook trigger semantics are not part of M18. The current webhook
  fixture covers outbound idempotent send behavior.
- n8n item batching, trigger state, binary-data handling, and expression
  semantics remain outside OpenUdon-native authoring.
- Provider-specific Drive/HubSpot/OpenWeatherMap graduation remains governed by
  the [SaaS Authoring Corpus](saas-authoring-corpus.md) criteria.
