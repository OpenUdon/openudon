# M40 - iCoT Natural-Language Authoring Reliability Expansion

## Goal

Make iCoT reliability evidence cover realistic operator language without adding a new execution
surface or requiring live provider credentials. M40 stays provider-free and focuses on Slack, Gmail,
OpenWeatherMap, weather-to-Gmail, missing-detail prompts, unsafe negative prompts, and grouped
scorecard evidence.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| Variant corpus format | `[+]` | Added `openudon.icot-authoring-variants.v1` metadata under fixture `reference/authoring-variants.json`. |
| Slack variants | `[+]` | Added direct, business, provider-as-verb, missing-channel, review-bypass, and inline-token Slack variants. |
| Gmail/weather variants | `[+]` | Added Gmail send/audit variants and OpenWeatherMap weather phrasing variants. |
| Weather-to-Gmail fixture | `[+]` | Promoted the reviewed weather-to-Gmail package into eval seed/build coverage with natural-language variants. |
| Negative prompt diagnostics | `[+]` | iCoT readiness now blocks inline-secret and review-bypass prompts with credential or side-effect failure families. |
| Scorecard grouping | `[+]` | `icot scorecard --include-variants` adds provider-family, variant-class, and provider/failure-family summaries. |
| Reliability hardening | `[+]` | Distinguished provider-free variant package evidence from optional real-LLM `icot authoring-eval`, reference-seeded missing-detail variants now remove only intended slots, failure-family classification is centralized, scorecard `needs_input` results include top issue diagnostics, and exact top issue regressions are enforced from variant metadata. |
| Variant metadata validation | `[+]` | Added `icot variants validate` and `make icot-variants-validate` for fast schema, expected-family, expected-top-issue, duplicate-ID, and clear-slot checks before scorecard runs. |
| Variant coverage gate | `[+]` | Added `icot variants coverage` and `make icot-variants-coverage` so each provider family represented in authoring variants must carry positive, missing-detail, and unsafe-negative coverage before release scorecard evidence. |
| Real-LLM evidence redaction | `[+]` | `icot authoring-eval` now scans generated project, intent, transcript, and report JSON output for credential-like literal values and fails or redacts unsafe evidence. |
| Report triage and provenance | `[+]` | `icot authoring-eval` now reports structured failure categories, provenance, and local ephemeral provider-output retention metadata, while `icot scorecard` records run ID/commit/prompt/readiness provenance, release-evidence retention metadata, and explicit missing-detail, unsafe false-pass, and needs-input diagnostic-gap counters; both report lanes write SHA-256 digest sidecars after consistency validation, `icot report verify` revalidates archived report JSON plus digest sidecars and retention metadata, and provider-free release targets verify the scorecard report. |
| Release command wiring | `[+]` | Added `make icot-authoring-scorecard` plus `make icot-variants-validate` and included both in `make release-saas-check`. |
| Enterprise positioning docs | `[+]` | Added documentation for LLM-assisted authoring, spec contract, deterministic execution, auditability, and low token use. |
| Verification | `[+]` | Focused tests, full Go tests/vet, `make icot-authoring-scorecard`, `make release-saas-check`, strict MkDocs, doc-memory, and `git diff --check` passed locally. |

## Acceptance

- M40 gates are provider-free.
- Positive variants build from reviewed package-local artifacts.
- Missing-detail and unsafe-negative variants stop with declared outcomes and failure families.
- Provider-free variant scorecards are documented as deterministic package evidence, while optional
  `icot authoring-eval` records real provider/model authoring evidence outside release gates.
- Live Slack/Gmail/weather execution remains optional manual evidence outside M40.
- Future UWS source-profile roadmap remains deferred.
