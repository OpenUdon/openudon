# Ramen Operator Checklist

Use this path when preparing generated artifacts or release evidence.

## Author

- Start from `templates/project.md`.
- Follow `docs/project-authoring.md` and `docs/data-flow.md`.
- For side-effectful work, follow `docs/safety.md` and name approval, sandbox proof-run, trusted-runner, and credential-binding policy.
- Check the eval fixture patterns in `docs/eval-gallery.md` before adding or changing corpus samples.

## Validate Locally

Daily deterministic check:

```bash
make check
```

Explicit vet parity:

```bash
make vet
```

Deterministic release readiness:

```bash
make release-check
```

## Manual Release Eval

Real-provider evals stay local/manual. Set provider credentials in the environment and run:

```bash
make release-eval
```

`make release-eval` uses `RAMEN_PROVIDER` and `RAMEN_MODEL`, defaulting to `gemini` and
`gemini-2.5-flash`. It runs `ramen eval --release-gate`, so absolute release criteria and
comparison regressions fail the command. It also passes a minimum brief count for the current
committed eval corpus; see `docs/xrd-009-expanded-corpus-release-evidence.md`.

## Release Notes

Use `docs/release-note-template.md`. Record the comparison baseline, eval JSON/Markdown paths,
commit and dirty state, release-gate result, deterministic checks, and known external blockers.
For real-provider release evidence, copy the eval JSON `provider_drift_watch` status plus the
Provider Drift Watch section from the eval Markdown report, then follow
`docs/xrd-006-provider-drift-watch.md`.
Development gates and real-provider release automation remain local/manual.

## Trusted Handoff

Before any side-effectful execution, review the generated artifacts and create approval JSON from
the current handoff package digest:

```bash
mkdir -p approvals
go run ./cmd/ramen approval-template \
  --example examples/support-email \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/support-email-sandbox.json
```

Then run the trusted wrapper. Use `--dry-run` first when checking the handoff without invoking udon:

```bash
go run ./cmd/ramen run \
  --example examples/support-email \
  --tier sandbox \
  --approval approvals/support-email-sandbox.json \
  --dry-run
```

The wrapper validates `expected/symphony-handoff.json`, `expected/quality.json`, current in-memory
quality gates, approval scope, approval expiry, package digest, and tier/state compatibility before
calling `scripts/run-udon.sh`. The approval-state contract remains compatible with Symphony
work-item routing, but Ramen owns this trusted local execution gate. See `SYMPHONY_WRAPPER.md` and
`docs/xrd-005-symphony-handoff.md`.
