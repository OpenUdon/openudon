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
For real-provider release evidence, copy the Provider Drift Watch section from the eval Markdown
report and follow `docs/xrd-006-provider-drift-watch.md`.
CI and release automation remain disabled until the infra handoff in
`docs/xrd-007-infra-handoff.md` is accepted.

## Trusted Handoff

Ramen emits generated artifacts only. Before any side-effectful execution, the Symphony-owned work
item must move through review and the applicable approval state. The trusted handoff package and
approval-state contract are defined in `docs/cross-repo-contracts.md`; the external Symphony owner
handoff is in `docs/xrd-005-symphony-handoff.md`.
