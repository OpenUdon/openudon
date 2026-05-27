# Product Smoke Matrix

M37 adds a product smoke matrix for the improved `v0.1.2-a.1` release candidate.
It is release evidence, not a new workflow feature. The matrix starts from reviewed
eval fixtures and natural-language requests, builds ignored scratch packages, and
uses the normal approval plus trusted-runner handoff path.

## Commands

Provider-free package and dry-run evidence:

```bash
make product-smoke-check
```

Opt-in local live evidence:

```bash
OPENUDON_EXECUTOR=/absolute/path/to/udon make product-smoke-live
```

Both commands write ignored evidence under `.openudon-run/product-smoke/`. Do not
commit approval JSON, run configs, provider responses, or real-provider output.
For targeted diagnosis, pass one or more `--scenario <id>` flags to
`openudon smoke-matrix`.

## Scenarios

| Scenario | Natural-language request | Basis | Live policy |
| --- | --- | --- | --- |
| Slack post | Post `OpenUdon v0.1.2-a.1 Slack smoke test` to my Slack sandbox channel and return the Slack response metadata. | `slack-message-audit-log` | Required live smoke for tagging. |
| Weather read | Fetch the current weather for Toronto and prepare a short audit summary. | `weather-toronto` | Runs live only when the OpenWeatherMap credential env exists. |
| Gmail audit receipt | Send an audit receipt email through Gmail with the approved package digest. | `gmail-send-audit-receipt` | Credential-backed examples exist; this matrix records dry-run evidence unless an operator runs a separate reviewed Gmail proof. |
| Slack/Jira intake | Read a Slack incident report, create a Jira issue, and post a Slack confirmation. | `itops-slack-jira-issue-intake` | Jira has fixture/dry-run coverage but no recorded real-key proof; matrix records dry-run evidence. |
| Order fulfillment | Look up a customer, check inventory, and create an order if stock is available. | `order-fulfillment-chain` | Local stub-backed live smoke. |
| Header API key report | Fetch a compliance report using a header API key. | `api-header-key-report` | Local stub-backed live smoke. |
| Bearer profile fetch | Fetch a directory profile using bearer authorization. | `api-oauth-profile-fetch` | Local stub-backed live smoke. |
| Inventory API key | Read inventory details using an API key credential binding. | `inventory-api-key-binding` | Local stub-backed live smoke. |
| Runtime-only render | Render a local audit note without calling an external API. | `runtime-only-render` | Trusted-runner dry-run only. |

## Environment

Required for Slack live tagging evidence:

```bash
export OPENUDON_SLACK_CHANNEL_ID=...
export UDON_CREDENTIAL_SLACK_BOT_TOKEN=...
export OPENUDON_EXECUTOR=/absolute/path/to/udon
```

The smoke command maps the operator-owned Slack bot token to the package-local
`slackBearer` credential binding without writing the token to artifacts. Optional
weather live evidence uses the OpenWeatherMap OpenAPI credential binding name:

```bash
export UDON_CREDENTIAL_OPENWEATHERAPIKEY=...
```

This env name is deliberate: the M37 live overlay declares the OpenWeatherMap
credential as `OpenWeatherAPIKey`, matching the OpenAPI document instead of the
older package-local `weather_appid` fixture shorthand. The current live overlay
passes the value through reviewed runtime data as
`{ ENVIRONMENT = "UDON_CREDENTIAL_OPENWEATHERAPIKEY" }` and maps it to the normal
`appid` query field on both the geocoding and weather requests, because the
current udon/OpenUdon compatibility path would otherwise duplicate the API-key
security field and the normal query parameter.
The smoke runner may set an internal `UDON_CREDENTIAL_INPUTS_APPID` alias for
the trusted-runner credential-like-field guard, but operators should configure
only `UDON_CREDENTIAL_OPENWEATHERAPIKEY`.

Local stub-backed scenarios set only synthetic child-process credential values
needed by the trusted-runner preflight. They do not require real provider secrets.

## Real-Provider Status

OpenUdon keeps provider credentials operator-owned and out of committed
artifacts. Current real-provider posture is:

- Slack: required `product-smoke-live` provider proof for the v0.1.2-a.1 tag
  gate, using `OPENUDON_SLACK_CHANNEL_ID` and
  `UDON_CREDENTIAL_SLACK_BOT_TOKEN`.
- OpenWeatherMap: optional `product-smoke-live` provider proof when
  `UDON_CREDENTIAL_OPENWEATHERAPIKEY` is present.
- Gmail: credential-backed Gmail/weather-to-Gmail examples use reviewed
  environment markers such as `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`,
  `GOOGLE_OAUTH_REDIRECT_URL`, and `GOOGLE_REFRESH_TOKEN`, but the product smoke
  matrix does not claim committed live Gmail evidence.
- Jira: strict fixtures cover Jira issue creation and Slack/Jira workflows, but
  there is no recorded real-key Jira proof in this release evidence.

## Tag Gate

Tag `v0.1.2-a.1` only after:

- `make product-smoke-check` passes;
- `make product-smoke-live` passes with Slack live evidence;
- local stub-backed live scenarios pass when a trusted executor is available;
- optional provider scenarios with complete envs pass, or missing envs are recorded
  as `skipped_missing_env`;
- normal M35/M36 provider-free release gates remain green.
