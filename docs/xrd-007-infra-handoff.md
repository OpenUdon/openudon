# XRD-007 Infra Handoff

This is the Ramen-owned handoff package for infrastructure owners. It defines the prerequisites for
deterministic CI and future real-provider release automation.

## Current State

- Deterministic GitHub CI is available in `.github/workflows/deterministic.yml`.
- Deterministic checks run locally on a trusted workstation with private siblings checked out.
- Real-provider evals remain local/manual because they need provider credentials and can produce
  generated artifacts, prompts, and model responses that require redaction review.
- CI requires `RAMEN_CI_GENELET_TOKEN` and `RAMEN_CI_TABILET_TOKEN` with read access to private
  dependency repositories.

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

Deterministic CI is allowed only when the workflow can run these commands from a clean private
checkout:

```bash
go test ./...
go vet ./...
make check
git diff --check
```

Deterministic CI must not have provider API keys in its environment. It should not upload generated
workflow artifacts, prompts, model responses, or eval archives. The workflow setup details are in
[`docs/ci.md`](ci.md).

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

Infra owns the token, runner policy, and any future automation expansion. Ramen's deterministic CI
readiness criteria are complete when the private checkout and secret controls above are satisfied.
Real-provider release evals stay local/manual.
