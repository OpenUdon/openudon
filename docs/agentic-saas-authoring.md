# Agentic SaaS Authoring

M15 focuses OpenUdon on AI-assisted authoring for common SaaS workflows. The
authoring assistant may help clarify goals, choose operations, draft request
mappings, and explain unresolved assumptions. Once `project.md` and
`workflows/intent.hcl` are ready, the rest of the path is deterministic:
validate, build, review, package, approve, and hand off to a trusted executor.

n8n and `../try-n8n` are service-priority evidence for this track. They help
identify useful services, common operations, and provider vocabulary. OpenUdon
does not run n8n workflows, import n8n internals, or add n8n-specific UWS
semantics for this milestone.
The [n8n Pattern Bridge](n8n-pattern-bridge.md) formalizes that evidence as a
small advisory summary format; it is not an executable importer.

## Authoring Contract

The supported authoring flow is:

1. Start from a natural-language goal or guided `cmd/icot` session.
2. Select local OpenAPI documents and listed operation IDs only.
3. Draft `project.md` with inputs, outputs, data flow, runtime policy,
   credential binding names, safety, and fallback behavior.
4. Draft `workflows/intent.hcl` with auditable `with` and `bind` mappings.
5. Run deterministic build and assessment commands.
6. Review generated evidence before any trusted-runner handoff.

AI output is a draft, not an execution permission. Side-effectful calls such as
send, create, update, delete, upload, or post remain review-required until a
human approves the package and the trusted runner receives a valid approval
artifact.

## Service Priority Corpus

The first SaaS corpus is selected from existing OpenUdon evals plus
`../try-n8n` reducibility evidence. These services are common enough to teach
the authoring path without making n8n import the product interface.
See [SaaS Authoring Corpus](saas-authoring-corpus.md) for the M16 fixture
policy, readiness matrix, and graduation criteria.

| Service | Starter Operation | Operation ID | Binding Name | Readiness Notes |
| --- | --- | --- | --- | --- |
| Slack | Post a message | `postMessage` | `slack_bot_token` | Existing native and n8n-derived fixtures cover side-effect review. |
| Gmail | Send a message | `sendMessage` | `gmail_oauth_token` | Existing fixture covers generated send response and audit evidence. |
| Jira | Create or fetch an issue | `createIssue`, `getIssue` | `jira_api_token` | Existing IT Ops fixtures cover Slack-to-Jira handoff. |
| HubSpot | List CRM records | `listDeals`, `listTickets` | `hubspot_private_app_token` | n8n-derived fixtures supply provider vocabulary and OpenAPI slices. |
| Google Drive | Upload a file | `uploadFile` | `google_drive_oauth_token` | Existing IT Ops archive fixture covers document handoff; Drive-specific graduation is deferred. |
| Airtable | Get a record | `getAirtableRecord` | `airtable_api_key` | Existing native and n8n-derived fixtures cover record lookup. |
| PagerDuty | Fetch a user | `getUser` | `pagerduty_api_token` | Existing fixture covers nested response extraction. |
| Trello | List board lists | `listTrelloBoardLists` | `trello_api_token` | Native fixture covers list summarization; n8n evidence owns the provider operation ID. |
| OpenWeatherMap | Fetch current weather | `getOpenWeatherMapCurrentWeather` | `openweathermap_appid` | n8n evidence covers provider-specific weather; generic native fixture covers the read-only pattern. |

## OpenAPI Readiness

Each service slice should be authoring-ready before it graduates from advisory
evidence to a strict golden fixture:

- local OpenAPI file is present under `openapi/`;
- selected operations have stable `operationId` values;
- required path, query, header, and body fields are visible to the prompt;
- request and response schemas expose the fields used in `with`, `bind`, and
  outputs;
- security schemes can be mapped to symbolic binding names;
- server URLs, examples, and enums are present when they affect request shape;
- no credential values or production tenant IDs are stored in examples.

When any of those inputs are missing, the authoring assistant should leave the
field unresolved or explain the assumption instead of inventing unavailable
provider behavior.

## Mapping Rules

For each SaaS API step, make review evidence easy to audit:

- name the OpenAPI file and operation ID;
- map every required request field to an input, safe literal, prior-step output,
  or credential binding name;
- use `bind` for prior-step data flow and keep `depends_on` on the consumer
  step;
- record response paths used by later steps or final outputs;
- identify side-effect scope for send/create/update/delete/upload/post actions;
- keep provider-specific secrets out of prompts, examples, and artifacts.

## Golden Fixtures

OpenUdon-native golden fixtures remain the regression target. n8n-derived
fixtures are advisory evidence until the OpenAPI slices and credential policy
are complete enough for strict comparison.

- `examples/eval/slack-message-audit-log` is the initial native SaaS authoring
  fixture. It documents an AI-authoring contract in
  `reference/authoring.json`.
- `examples/eval/gmail-send-audit-receipt` and
  `examples/eval/itops-slack-jira-issue-intake` join Slack as the M16 initial
  strict SaaS golden set.
- `examples/eval/itops-incident-response-archive` and related fixtures cover
  follow-on service patterns before they graduate into strict provider-specific
  coverage.
- `examples/eval/n8n-*` fixtures preserve n8n provenance in
  `reference/n8n.json`. Selected M22 fixtures also include
  `reference/n8n-bridge.json` summaries for service, operation, credential, and
  unsupported-semantics evidence. They stay diagnostic unless their policy
  graduates them to strict mode.
