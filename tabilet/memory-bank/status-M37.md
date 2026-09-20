# M37 - Product Smoke Matrix and v0.1.2-a.1 Readiness

## Goal

Make an improved `v0.1.2-a.1` release candidate credible for product review by adding a broad
natural-language smoke matrix across major OpenUdon artifacts, while keeping provider-free release
gates deterministic and live provider execution local/manual.

## Task State

| Item | State | Notes |
|---|---|---|
| Product smoke matrix command | `[+]` | `openudon smoke-matrix --mode dry-run\|live` builds ignored scratch packages from reviewed eval fixtures. |
| Make targets | `[+]` | `make product-smoke-check` and `make product-smoke-live` wrap the CLI command. |
| Natural-language scenario coverage | `[+]` | Matrix covers Slack, weather, Gmail, Slack/Jira, order fulfillment, header API key, bearer profile, inventory API key, and runtime-only render cases. |
| Trusted-runner evidence | `[+]` | Each executable scenario creates sandbox approval JSON and normal `openudon run` evidence under `.openudon-run/product-smoke/`. |
| Live policy | `[+]` | Slack live env is required for the tag gate; OpenWeatherMap live proof runs when its credential env exists; Gmail has credential-backed examples/manual proof support but product smoke records dry-run evidence; Jira has fixture/dry-run coverage with no recorded real-key proof. |
| Documentation | `[+]` | Release stewardship, release note template, and product smoke docs describe commands, env, evidence, and tag gate. |
| Package hygiene | `[+]` | Smoke output stays under ignored `.openudon-run/product-smoke/`; no provider output is tracked. |
| OpenWeatherMap credential naming | `[+]` | Weather live smoke uses `OpenWeatherAPIKey` from the OpenWeatherMap OpenAPI security scheme and expects `UDON_CREDENTIAL_OPENWEATHERAPIKEY`, not the older fixture shorthand `UDON_CREDENTIAL_WEATHER_APPID`. |
| Weather live proof | `[+]` | Targeted `weather-read` live smoke passed with digest `50a3588b8160aac7197ce6330e152b177b7076b76740a73143b9860181a3676f` after adding `appid` to both OpenWeatherMap geocoding and weather requests. |
| Final release evidence | `[+]` | Final gates and full live product smoke passed on 2026-05-26; `v0.1.2-a.1` is ready to tag from the recorded OpenUdon commit. |

## Acceptance Criteria

- `make product-smoke-check` passes.
- `make product-smoke-live` passes with a trusted executor and required Slack env.
- Slack channel visibly contains the `OpenUdon v0.1.2-a.1 Slack smoke test` message.
- Local stub-backed live scenarios pass or a concrete executor compatibility blocker is recorded.
- Optional external provider scenarios with missing env are listed as `skipped_missing_env`.
- M35/M36 release gates remain green before tagging `v0.1.2-a.1`.

## Notes

- Real credentials remain operator-owned environment variables.
- Current real-provider evidence distinguishes recorded live smoke from
  credential-backed examples: Slack and OpenWeatherMap have recorded product
  smoke live proof, Gmail has reviewed environment-marker examples/manual proof
  support, and Jira has no recorded real-key proof.
- Weather live smoke must follow the provider OpenAPI credential name: `OpenWeatherAPIKey` maps to
  `UDON_CREDENTIAL_OPENWEATHERAPIKEY`. The committed `weather-toronto` eval fixture still uses the
  older `weather_appid` shorthand for provider-free fixture compatibility. The M37 live overlay
  passes `{ ENVIRONMENT = "UDON_CREDENTIAL_OPENWEATHERAPIKEY" }` through reviewed runtime data and
  maps it to the normal `appid` query field on both the geocoding and weather requests, because the
  current udon/OpenUdon compatibility path would otherwise duplicate the API-key security field and
  the normal query parameter. The smoke harness
  may set internal `UDON_CREDENTIAL_INPUTS_APPID` as a trusted-runner guard alias, but the operator
  contract remains `UDON_CREDENTIAL_OPENWEATHERAPIKEY`.
- Raw provider responses, approval JSON, run configs, and run evidence stay ignored.
- OpenUdon continues to call udon only as an external trusted executor through `OPENUDON_EXECUTOR`.

## Final Evidence - 2026-05-26

- Public dependency compatibility was restored by updating `github.com/OpenUdon/apitools` to
  `v0.0.0-20260525034344-3df8f5e2a23d`, which includes the helper packages used by the public
  OpenUdon tree.
- Public-module gates passed with `GOWORK=off go test ./... -count=1 -timeout=5m` and
  `GOWORK=off go vet ./...`.
- Local gates passed with `go test ./... -count=1 -timeout=5m`, `go vet ./...`, `make check`,
  `make eval-seed-build`, `make release-saas-check`, and `make product-smoke-check`.
- `make release-saas-check` covered strict MkDocs, UWS validation, doc-memory validation,
  n8n bridge validation, eval seed build, boundary checks, and trusted-runner dry-run handoff.
- Full live smoke passed with
  `OPENUDON_EXECUTOR=/home/peter/Workspace/udon/dist/udon-openudon-m36 make product-smoke-live`.
  Live summary: `.openudon-run/product-smoke/summary.json`.
- Live smoke results:
  - `slack-post`: pass, digest `1be73ca3b64e9865d50a3976455090d01704af0cfd5399852019254c5728303a`.
  - `weather-read`: pass, digest `802c8bf84bc46ab9871588ca6efd66524885a64d46d23b5ba7d36ff59694dc6f`.
  - `order-fulfillment`: pass, digest `20a725d8ad28a69f5b49439941332bc65b4db0b07137f757dc07a75d2d0d98aa`.
  - `header-api-key-report`: pass, digest `f02aad79e4b76844c7f2b021e525ba944a6fd2067f5f3575bc356615369b73db`.
  - `bearer-profile-fetch`: pass, digest `5eee59299d81b7f6e187ab30e917a301794de17089f190aba8114e0150e9e491`.
  - `inventory-api-key`: pass, digest `992d36c9c2b0876943b1743e04d3c70cb8bc2b11b5329c03fd46cb5d5e659160`.
  - `gmail-audit-receipt`: `skipped_manual_provider`, digest
    `c14beb4c46a308c455bef05aaf7db48608f70c4cb6040d3bf98bc09102b031db`.
  - `slack-jira-intake`: `skipped_manual_provider`, digest
    `97d1aa71ff78db5844ad6f200c118ef5c5c66f7a34018e1651d97edbef5651a3`.
  - `runtime-only-render`: `dry_run_only`, digest
    `c69581252830c30e4c67595c0219156c618747556686efa7fef4bfbce5a65ddf`.
- Slack live smoke required a live overlay correction: host-only server `https://slack.com`,
  explicit path `/api/chat.postMessage`, and direct response-metadata output. This keeps the
  release smoke focused on the provider post without requiring a private runtime render helper.
- Failure details are redacted and capped in smoke summaries so provider HTML/error bodies do not
  dominate release evidence.
