# XRD-006 Provider Drift Watch

XRD-006 is closed in Ramen as a structured reporting capability. Provider behavior can still change
without a Ramen code change, so release evidence must separate external drift from deterministic
Ramen regressions.

## When To Attach A Watch Report

Attach provider drift evidence to release notes whenever a real-provider eval is used for release
confidence:

```bash
make release-eval
```

The eval JSON report includes a `provider_drift_watch` block, and the eval Markdown report renders
the same Provider Drift Watch findings. Copy the JSON path and Markdown findings into
[`docs/release-note-template.md`](release-note-template.md) when preparing a release note.

## Signals To Monitor

| Signal | Source | Drift trigger |
| --- | --- | --- |
| Structured fallback count | `provider_drift_watch.structured_fallbacks` and `.structured_fallback_delta`. | Any legacy fallback in a release-gated run, or an increase against the comparison run. |
| Rate and transient failures | `provider_drift_watch.provider_failures[]` from failed results with provider/model errors, timeout text, HTTP 429/5xx text, or unavailable service text. | Any new rate, timeout, transient, or provider-service failure. |
| Model availability | `provider_drift_watch.model_availability` and `.model_availability_detail`. | Requested model is missing, renamed, retired, permission-gated, or unavailable in the configured region/account. |
| Attempts-to-pass | `provider_drift_watch.max_attempts_to_pass`, `.repeated_repair_loops`, and `.attempt_regressions[]`. | Any brief needs more than two attempts in a release gate, or attempts increase versus baseline. |
| Release-gate failures | `provider_drift_watch.release_gate_failures[]`, `ramen eval --release-gate` exit status, and the eval report summary. | Any absolute release criterion fails, or a comparison regression fails the release gate. |

## Triage Guidance

- Treat deterministic failures that reproduce without provider calls as Ramen issues.
- Treat fallback increases, model-class failures, provider HTTP errors, timeout errors, or sudden
  model-unavailable errors as provider drift until a deterministic repro proves otherwise.
- Record the provider, model, date, commit, comparison baseline, `provider_drift_watch.status`,
  fallback count, maximum attempts, release-gate result, and any observed provider error text.
- Do not loosen release criteria from a single transient run. Rerun once from a trusted workstation
  if the error looks external, then document both attempts.

## Boundary

Ramen owns reporting these signals in eval JSON, eval Markdown, and release notes. Provider owners
own service availability, schema dialect behavior, rate limits, transient failures, and model
lifecycle changes.
