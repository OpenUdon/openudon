# M45 Release Readiness And Documentation

## Status

Complete.

## Task State

| Item | State | Notes |
|---|---|---|
| README and docs terminology | `[+]` | Updated core README, authoring, synthesize, intent, review handoff, related-project, and selected tutorial wording from API-source-focused to API/event sources. |
| Document `asyncapi/` package directory | `[+]` | README, synthesize docs, review handoff docs, and memory bank now list `asyncapi/`. |
| Clarify runtime ownership | `[+]` | Docs state OpenUdon validates/packages AsyncAPI workflows but does not execute AsyncAPI protocols. |
| Release checklist and full gates | `[+]` | Full provider-free release gates pass with public UWS 1.3 and apitools AsyncAPI module revisions. |
| Post-review hardening | `[+]` | AsyncAPI refs now resolve against local source documents, advisory sidecars are skipped during iCoT source discovery, and API/event source docs use consistent directory wording. Independent review follow-ups added negative AsyncAPI content-sniff coverage, escaped JSON Pointer coverage, and AsyncAPI operation-index parse diagnostics. |

## Verification

- Focused Go tests and the AsyncAPI fixture build pass in workspace and public-module modes.
- Full release gates passed after publishing the required UWS and apitools revisions.
