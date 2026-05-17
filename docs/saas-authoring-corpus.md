# SaaS Authoring Corpus

M16 turns the M15 SaaS priority list into a clearer regression corpus. The
corpus has two fixture classes:

- **Strict native**: OpenUdon-owned examples with strict reference policy. These
  are the first regression targets for operation selection, request mapping,
  data flow, side-effect policy, and review evidence.
- **Advisory evidence**: n8n-derived or partial examples that preserve provider
  vocabulary, operation names, and OpenAPI slices. These guide authoring but do
  not define runtime behavior or release-blocking exact intent shape.

## Strict Native Set

The initial strict set is intentionally small. It covers write side effects,
read-to-function data flow, service-to-service binding, and credential binding
names without requiring real provider credentials.

| Fixture | Services | Operations | Policy | Coverage |
| --- | --- | --- | --- | --- |
| `slack-message-audit-log` | Slack-like chat | `postMessage` | strict | Message post request fields, response binding, side-effect review. |
| `gmail-send-audit-receipt` | Gmail-like messages | `sendMessage` | strict | Send-message request body, response binding, audit receipt rendering. |
| `itops-slack-jira-issue-intake` | Slack and Jira | `getSlackMessage`, `createIssue`, `postMessage` | strict | Cross-service data flow, bearer binding names, create/post side effects. |

Native fixtures may use sandbox-local OpenAPI slices instead of full public
provider specifications. That keeps deterministic gates provider-free while
still testing the OpenUdon authoring contract.

M18 extends the strict set with multi-service pattern coverage. See
[Multi-Service SaaS Patterns](multi-service-saas-patterns.md) for cross-service
bindings, credential scopes, side-effect posture, and known gaps.

## Advisory Evidence

Advisory fixtures stay useful because they show which provider operations users
are likely to ask for. They must not introduce n8n runtime concepts into UWS or
OpenUdon.

| Fixture | Service | Operation | Binding Name | Current Role |
| --- | --- | --- | --- | --- |
| `n8n-airtable-record-get` | Airtable | `getAirtableRecord` | `airtable_api_key` | Advisory provider vocabulary plus native pattern coverage in `airtable-record-normalize`. |
| `n8n-gmail-message-send` | Gmail | `sendMessage` | `gmail_oauth_token` | Advisory n8n provenance; strict native Gmail fixture owns release coverage. |
| `n8n-google-drive-file-upload` | Google Drive | `uploadFile` | `google_drive_oauth_token` | Advisory upload shape; strict native Drive coverage is not selected yet. |
| `n8n-hubspot-deal-list` | HubSpot | `listDeals` | `hubspot_private_app_token` | Advisory CRM list shape; strict native HubSpot coverage is not selected yet. |
| `n8n-jira-issue-get` | Jira | `getIssue` | `jira_api_token` | Advisory read shape; strict native Jira coverage currently uses `createIssue`. |
| `n8n-openweathermap-current-weather` | OpenWeatherMap | `getOpenWeatherMapCurrentWeather` | `openweathermap_appid` | Advisory provider-specific weather shape; generic native weather coverage is separate. |
| `n8n-pagerduty-user-get` | PagerDuty | `getUser` | `pagerduty_api_token` | Advisory provider vocabulary plus native pattern coverage in `pagerduty-user-contact-card`. |
| `n8n-slack-message-post` | Slack | `postMessage` | `slack_bot_token` | Advisory n8n provenance; strict native Slack fixture owns release coverage. |
| `n8n-trello-list-get-all` | Trello | `listTrelloBoardLists` | `trello_api_token` | Advisory provider vocabulary plus native pattern coverage in `trello-list-summary`. |

## OpenAPI Readiness Matrix

| Service | Fixture Evidence | Request Fields | Response Fields | Security Evidence | Readiness |
| --- | --- | --- | --- | --- | --- |
| Slack | `slack-message-audit-log`, `itops-slack-jira-issue-intake`, `n8n-slack-message-post` | `channel`, `text`; `channel`, `ts` for fetch | `ok`, `channel`, `ts`, `message.text` | `slack_bot_token` in IT Ops fixture; sandbox Slack post has no auth scheme | strict native for post and IT Ops flow; advisory n8n provenance retained. |
| Gmail | `gmail-send-audit-receipt`, `n8n-gmail-message-send` | `to`, `subject`, `message` | `id`, `threadId` | `gmail_oauth_token` convention; sandbox native slice has no auth scheme | strict native for send-message authoring. |
| Jira | `itops-slack-jira-issue-intake`, `n8n-jira-issue-get` | `projectKey`, `summary`, `description`, `issueType`, optional `priority` | `key`, `self` | `jira_api_token` through bearer binding | strict native for create issue; advisory for get issue. |
| HubSpot | `n8n-hubspot-deal-list` | no required request fields in copied list slice | response schema is sparse in copied slice | `hubspot_private_app_token` convention | advisory; needs native project brief and stricter schema before graduation. |
| Google Drive | `itops-incident-response-archive`, `n8n-google-drive-file-upload` | `parentId`, `name`, `content` in native archive; sparse provider upload slice in n8n fixture | uploaded file `id` and `name` in native archive | `google_drive_oauth_token` convention | native multi-service archive exists; M16 strict set defers Drive-specific graduation. |
| Airtable | `airtable-record-normalize`, `n8n-airtable-record-get` | `baseId`, `tableId`, `recordId` | `id`, `fields`, `createdTime` | `airtable_api_key` convention; native sandbox slice has no auth scheme | native pattern ready; not in initial strict SaaS golden set. |
| PagerDuty | `pagerduty-user-contact-card`, `n8n-pagerduty-user-get` | `userId` | `user.id`, `user.name`, `user.email` | `pagerduty_api_token` convention; native sandbox slice has no auth scheme | native pattern ready; not in initial strict SaaS golden set. |
| Trello | `trello-list-summary`, `n8n-trello-list-get-all` | `boardId` | `lists[].id`, `lists[].name`, `lists[].closed` | `trello_api_token` convention; native sandbox slice has no auth scheme | native pattern ready; provider operation ID mismatch remains before graduation. |
| OpenWeatherMap | `n8n-openweathermap-current-weather`, `weather-enrichment-advice` | `city` in generic native fixture; provider-specific query in n8n fixture | `city`, `tempC`, `condition` in generic native fixture | `openweathermap_appid` convention | advisory for provider-specific OpenWeatherMap; generic weather fixture remains pattern coverage. |

## Graduation Criteria

An advisory fixture can graduate to strict native coverage when it has:

- an OpenUdon-owned `project.md` that does not depend on n8n runtime behavior;
- a local OpenAPI slice with stable operation IDs, required request fields,
  response fields used by outputs/binds, and security schemes when credentials
  are part of the provider call;
- a `reference/intent.hcl` with explicit `with` and `bind` mappings for every
  required request field and every consumed response field;
- symbolic credential binding names in project text and intent, never values;
- side-effect language for send/create/update/delete/upload/post operations;
- `reference/policy.json` in strict mode with `max_blocking` set to `0`;
- optional `reference/authoring.json` metadata documenting the authoring
  contract, mappings, response paths, credential binding, and fixture role.

If any of those inputs are incomplete, keep the fixture advisory and use it as
service-priority evidence only.
