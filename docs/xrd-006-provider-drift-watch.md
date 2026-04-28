# XRD-006 Provider Drift Watch

XRD-006 is a watch plan, not an implementation plan. Provider behavior can change without a Ramen
code change, so release evidence must separate external drift from deterministic Ramen regressions.

## When To Attach A Watch Report

Attach provider drift evidence to release notes whenever a real-provider eval is used for release
confidence:

```bash
make release-eval
```

The eval Markdown report includes a Provider Drift Watch section. Copy its findings into
[`docs/release-note-template.md`](release-note-template.md) when preparing a release note.

## Signals To Monitor

| Signal | Source | Drift trigger |
| --- | --- | --- |
| Structured fallback count | Eval summary `legacy_fallbacks` and result `mode` fields. | Any legacy fallback in a release-gated run, or an increase against the comparison run. |
| Rate and transient failures | Failed results with provider/model errors, timeout text, HTTP 429/5xx text, or unavailable service text. | Any new rate, timeout, transient, or provider-service failure. |
| Model availability | Eval metadata provider/model fields and model-unavailable error text. | Requested model is missing, renamed, retired, permission-gated, or unavailable in the configured region/account. |
| Attempts-to-pass | Per-brief `attempt_count`, `attempts_to_pass`, and repeated-repair summary. | Any brief needs more than two attempts in a release gate, or attempts increase versus baseline. |
| Release-gate failures | `ramen eval --release-gate` exit status plus the eval report summary. | Any absolute release criterion fails, or a comparison regression fails the release gate. |

## Triage Guidance

- Treat deterministic failures that reproduce without provider calls as Ramen issues.
- Treat fallback increases, model-class failures, provider HTTP errors, timeout errors, or sudden
  model-unavailable errors as provider drift until a deterministic repro proves otherwise.
- Record the provider, model, date, commit, comparison baseline, fallback count, maximum attempts,
  release-gate result, and any observed provider error text.
- Do not loosen release criteria from a single transient run. Rerun once from a trusted workstation
  if the error looks external, then document both attempts.

## Boundary

Ramen owns reporting these signals in eval evidence and release notes. Provider owners own service
availability, schema dialect behavior, rate limits, transient failures, and model lifecycle changes.
