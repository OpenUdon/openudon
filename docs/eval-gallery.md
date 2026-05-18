# Eval Sample Gallery

The eval corpus under `examples/eval` is intentionally curated. Each sample should demonstrate a
specific workflow behavior or failure class rather than merely adding volume. For the M16 SaaS
fixture policy, readiness matrix, and advisory-to-strict graduation criteria, see
[SaaS Authoring Corpus](saas-authoring-corpus.md). For M18 cross-service patterns, see
[Multi-Service SaaS Patterns](multi-service-saas-patterns.md). For M20 trial results, see
[SaaS Authoring Trials](saas-authoring-trials.md).

| Sample | Purpose |
| --- | --- |
| `cmd-allowed-deploy` | Approved `cmd` runtime for a sandbox deployment status command. |
| `cmd-disallowed-deploy` | Negative runtime-policy coverage for disallowed command execution. |
| `api-header-key-report` | Header API-key credential binding for an OpenAPI read operation. |
| `api-nested-user-create` | Nested request-body write operation with bearer auth and approval boundary. |
| `api-oauth-profile-fetch` | Bearer/OAuth-style credential binding for a path-parameter read. |
| `airtable-record-normalize` | Multi-step data-passing fixture: fetched record response feeds local normalization. |
| `array-response-summary` | Array response extraction into an approved local summary function. |
| `compliance-report-summary` | Multi-step data-passing fixture: fetched compliance report feeds local summary rendering. |
| `crm-note-write` | Side-effectful write operation with trusted-runner and sandbox policy. |
| `cursor-pagination-report` | Cursor pagination, bearer security, response cursor extraction, and local report rendering. |
| `customer-export-two-pages` | Multi-step pagination and merge-style function handling. |
| `fallback-cache-read-through` | Primary API read with explicit local cached-fallback preparation and selection. |
| `gmail-send-audit-receipt` | Initial OpenUdon-native SaaS authoring fixture: send-message response feeds local audit receipt rendering. |
| `incomplete-brief-repair` | Negative fixture that renders clarifying questions instead of inventing missing workflow behavior. |
| `inventory-api-key-binding` | Credential binding names for API-key-style request parameters. |
| `inventory-reorder-decision` | Multi-step data-passing fixture: inventory response feeds local reorder decision rendering. |
| `itops-incident-response-archive` | Strict multi-service SaaS pattern: create Jira issue, alert Slack, and archive a Drive timeline report. |
| `itops-slack-jira-issue-intake` | Initial multi-service SaaS authoring fixture: parses Slack report text, creates Jira, and confirms in Slack. |
| `itops-workflow-backup-github` | n8n IT Ops-inspired workflow backup from n8n API to GitHub Contents API. |
| `missing-credential-policy-negative` | Negative fixture that reports missing credential policy instead of issuing unaudited API calls. |
| `missing-openapi-capability-negative` | Negative fixture that reports missing OpenAPI capability instead of inventing provider calls. |
| `n8n-airtable-record-get` | Advisory n8n reducibility sample for Airtable `record/get` mapped to OpenUdon `getAirtableRecord`. |
| `n8n-gmail-message-send` | Advisory n8n reducibility sample for Gmail `message/send` mapped to OpenUdon `sendMessage`. |
| `n8n-google-drive-file-upload` | Advisory n8n reducibility sample for Google Drive `file/upload` mapped to OpenUdon `uploadFile`. |
| `n8n-hubspot-deal-list` | Advisory n8n reducibility sample for HubSpot `deal/getAll` mapped to OpenUdon `listDeals`. |
| `n8n-jira-issue-get` | Advisory n8n reducibility sample for scanner-backed Jira `issue/get` mapped to OpenUdon `getIssue`. |
| `n8n-openweathermap-current-weather` | Advisory n8n reducibility sample for OpenWeatherMap current weather mapped to OpenUdon `getOpenWeatherMapCurrentWeather`. |
| `n8n-pagerduty-user-get` | Advisory n8n reducibility sample for PagerDuty `user/get` mapped to OpenUdon `getUser`. |
| `n8n-slack-message-post` | Advisory n8n reducibility sample for Slack `message/post` mapped to OpenUdon `postMessage`. |
| `n8n-trello-list-get-all` | Advisory n8n reducibility sample for Trello `list/getAll` mapped to OpenUdon `listTrelloBoardLists`. |
| `order-fulfillment-chain` | Strict multi-service SaaS pattern: customer and inventory lookups feed sandbox fulfillment order creation. |
| `offset-pagination-export` | Offset pagination with two fixed pages and a local merge step. |
| `pagerduty-user-contact-card` | Multi-step data-passing fixture: nested user response feeds local contact-card rendering. |
| `paginated-list` | Simple OpenAPI list operation with bounded request parameters. |
| `page-token-pagination-export` | Page-token pagination with second-page token binding and local merge. |
| `profile-boundary-manifest` | Future runtime/profile boundary coverage: renders a local manifest with `fnct` instead of inventing SQL, SSH, or `x-udon-*` profile semantics. |
| `profile-fetch-access-card` | Multi-step data-passing fixture: fetched employee profile feeds local access-card rendering. |
| `response-field-ticket-alert` | Nested response-field extraction into an approved side-effectful alert adapter. |
| `retry-idempotent-webhook-send` | Idempotent side-effectful webhook send with workflow timeout/idempotency controls. |
| `runtime-only-render` | No-OpenAPI runtime-only `fnct` rendering workflow. |
| `slack-message-audit-log` | Initial OpenUdon-native SaaS authoring fixture: post-message response feeds local audit-log rendering. |
| `support-email` | API lookup plus approved side-effectful email adapter and safety boundary. |
| `support-priority-routing` | Function-backed classification/routing with explicit contracts. |
| `timeout-idempotency-controls` | Runtime-only workflow with explicit workflow timeout, step timeout, and workflow idempotency metadata. |
| `trello-list-summary` | Multi-step data-passing fixture: array response feeds local list summarization. |
| `unsafe-side-effect-boundary-negative` | Negative fixture that prepares an approval package instead of executing unsafe deployment side effects. |
| `user-create-welcome-message` | Multi-step data-passing fixture: created user response feeds local welcome rendering. |
| `webhook-validation-fnct` | Runtime-only webhook payload validation and normalization through an approved function. |
| `weather-enrichment-advice` | Multi-step data-passing fixture: weather response feeds local advice rendering. |
| `weather-toronto` | Strict native SaaS authoring fixture: hidden technical step expansion from city lookup to weather lookup with credential binding. |

## Adding Samples

Add samples only when they strengthen coverage:

- Prefer one clear purpose per sample.
- Include `reference/intent.hcl` when reference comparison should detect drift.
- Add `reference/policy.json` when the reference is illustrative or needs per-fixture triage notes.
  Use `mode: "strict"` for golden references and `mode: "advisory"` when deterministic quality
  gates are authoritative but exact intent shape may drift.
- Treat step names, output names, request literal names, and bind field names as semantic hints by
  default. They should help diagnose drift, but they are not release-blocking by themselves.
- Treat wrong runtime type, wrong selected OpenAPI operation, and reference parse/compare failures
  as behavioral drift. These are blocking unless a fixture policy deliberately overrides them.
- For n8n reducibility samples, keep upstream n8n and w8m inputs hermetic by copying OpenAPI
  evidence into the fixture and recording provenance in `reference/n8n.json`. Treat them as
  service-priority and mapping evidence for agentic SaaS authoring unless their fixture policy
  explicitly graduates them to strict OpenUdon-native coverage.
- For M22 n8n bridge samples, keep `reference/n8n-bridge.json` as authoring-assistance evidence
  only. It validates services, nodes, operation candidates, symbolic credentials, and unsupported
  semantics, but it is not an executable import.
- Keep `max_blocking` at `0` unless the fixture is intentionally tracking a temporary known gap.
- Keep secret-shaped values fake and avoid real provider data.
- Document credential bindings by name only.
- For side-effectful workflows, include approval/trusted-runtime policy and sandbox/test proof-run
  policy.
- Keep the corpus curated; grow size only after current samples remain stable.
