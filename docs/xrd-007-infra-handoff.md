# XRD-007 Infra Handoff

This is the Ramen-owned handoff package for infrastructure owners. It defines the prerequisites for
future deterministic CI and real-provider release automation without enabling GitHub CI from this
repository.

## Current State

- Hosted GitHub CI is disabled for Ramen.
- Deterministic checks run locally on a trusted workstation with private siblings checked out.
- Real-provider evals remain local/manual because they need provider credentials and can produce
  generated artifacts, prompts, and model responses that require redaction review.
- This repository currently has no `.github/workflows` automation to re-enable.

## Private Checkout Prerequisites

The runner workspace must check out Ramen and every private sibling required for local builds:

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

Readiness check:

```bash
./scripts/check-siblings.sh
```

## Deterministic Runner Gate

Enable deterministic CI only after a self-hosted private runner can run these commands from a clean
checkout:

```bash
go test ./...
go vet ./...
make check
git diff --check
```

Deterministic CI must not have provider API keys in its environment. It should not upload generated
workflow artifacts, prompts, model responses, or eval archives.

## Real-Provider Release Gate

Real-provider release automation remains a future manual or protected workflow. It may be added only
after infra has a protected secret store and log/artifact redaction policy.

Allowed command shape:

```bash
go run ./cmd/ramen eval --root ./examples/eval --provider <provider> --model <model> --release-gate
```

Required release evidence:

- Eval JSON and Markdown report paths.
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
- Deterministic CI must not receive provider keys.
- Do not print environment variables or secret values in logs.
- Generated OpenAPI, UWS, HCL, review evidence, eval archives, prompts, provider responses, and
  uploaded logs must not contain literal secrets.
- Generated artifacts may refer to credential binding names only.
- Artifact retention for real-provider runs must stay disabled until redaction review is in place.

## Re-enable Decision

Infra owns the decision to add `.github/workflows` or another automation entry point. Ramen's
readiness criteria are complete when the private checkout, self-hosted deterministic runner, and
secret/artifact controls above are satisfied. Until then, CI remains disabled and release evals stay
local/manual.
