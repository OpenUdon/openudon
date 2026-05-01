# XRD-007 Infra Handoff

This is the Ramen-owned handoff package for infrastructure owners. It defines the local readiness
prerequisites, the structured readiness report, and the conditions required before future automation
is reintroduced.

## Current State

- Deterministic checks run locally on a trusted workstation with private siblings checked out.
- Real-provider evals remain local/manual because they need provider credentials and can produce
  generated artifacts, prompts, and model responses that require redaction review.
- GitHub CI has been removed during active development because the private dependency checkout
  layout is still changing.

## Private Checkout Prerequisites

The local workspace must check out Ramen and every private sibling required for local builds:

| Path | Purpose |
| --- | --- |
| `../uws` | Public UWS model, schema, and validation dependency. |
| `../udon` | Private compiler/runtime dependency. |
| `../symphony` | Work orchestration sibling used by the private workspace. |
| `../grand` | Private udon build-time sibling. |
| `../golet` | Private udon build-time sibling. |
| `../hcllight` | Private udon build-time sibling. |
| `../horizon` | Private udon build-time sibling. |
| `../molecule` | Private udon build-time sibling. |
| `../arazzo` | Private udon build-time sibling. |
| `../apitools` | Shared API tools and OpenAPI discovery/import library used by Ramen and udon. |

Readiness check:

```bash
./scripts/check-siblings.sh
```

Structured readiness report:

```bash
go run ./cmd/ramen readiness --out eval/readiness/local.json
```

## Local Deterministic Gate

Run these commands from a clean local private checkout:

```bash
go run ./cmd/ramen readiness --run-gates --out eval/readiness/local.json
```

With `--run-gates`, the readiness report runs `go test ./...`, `go vet ./...`, `make check`, and
`git diff --check`. These deterministic checks do not need provider API keys and should not upload
generated workflow artifacts, prompts, model responses, or eval archives.

The readiness report uses schema `ramen.local-readiness.v1` and records:

- required sibling presence
- deterministic gate results or skipped status
- `git diff --check` and dirty-tree status
- ignored local artifact paths such as `.ramen-run/`, `approvals/`, `eval/runs/`,
  `eval/artifacts/`, and `eval/readiness/`
- provider credential environment variable presence as booleans only, never values
- local/manual automation policy

## Real-Provider Release Gate

Real-provider release automation is not part of the current development setup. It may be added only
after infra has a protected secret store and log/artifact redaction policy.

Allowed command shape:

```bash
go run ./cmd/ramen eval --root ./examples/eval --provider <provider> --model <model> --release-gate
```

Required release evidence:

- Eval JSON and Markdown report paths.
- Local readiness JSON path from `ramen readiness --run-gates`.
- Provider and model.
- Prompt version.
- Commit and dirty state.
- Pass rate.
- Attempts-to-pass and repeated repair loop count.
- Structured fallback count.
- Blocking reference issue count.
- Provider Drift Watch findings from `docs/xrd-006-provider-drift-watch.md`.

## Secret And Artifact Rules

- Provider keys may exist only in a protected secret store or trusted local environment.
- Do not print environment variables or secret values in logs.
- Generated OpenAPI, UWS, HCL, review evidence, eval archives, prompts, provider responses, and
  uploaded logs must not contain literal secrets.
- Generated artifacts may refer to credential binding names only.
- Artifact retention for real-provider runs must stay disabled until redaction review is in place.

## Re-enable Decision

Infra owns any future runner policy and automation expansion. Ramen now owns the local structured
readiness report; readiness criteria remain local until the private checkout layout and secret
controls are stable enough to reintroduce automation. Real-provider release evals stay local/manual.
