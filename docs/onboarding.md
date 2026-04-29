# Ramen Operator Onboarding

Ramen is for trusted internal users who generate, validate, review, and hand off workflow artifacts.
It is not a direct production execution tool.

## Environment Setup

Required repositories must sit next to `ramen`:

```text
../ramen
../uws
../udon
../symphony
../grand
../golet
../hcllight
../horizon
../molecule
../arazzo
```

Check the local workspace:

```bash
make check
```

Daily deterministic gates are:

```bash
go test ./...
go vet ./...
make check
git diff --check
```

Deterministic release readiness is:

```bash
make release-check
```

XRD-007 local/private checkout readiness evidence is:

```bash
go run ./cmd/ramen readiness --run-gates --out eval/readiness/local.json
```

## Provider Credentials

Real LLM synthesis and evals need provider credentials from environment variables:

```bash
export GEMINI_API_KEY=...
```

Do not put provider keys, API tokens, SMTP passwords, or other secret values in prompts, examples,
`project.md`, generated artifacts, or eval fixtures. Use credential binding names only.

## Author A Project

For guided authoring, run:

```bash
go run ./cmd/icot --example ./examples/<name>
```

The wizard creates `project.md`, `openapi/`, `workflows/`, and `expected/`, then leaves synthesis to
the normal `ramen build --example ./examples/<name>` step when `workflows/intent.hcl` is already
authored.

Useful deterministic options:

- `--print` renders `project.md` and `workflows/intent.hcl` without writing files.
- `--from-example ./examples/eval/<name>` seeds prompts from an existing brief.
- `--answers ./answers.yaml` renders from YAML or JSON without prompts.
- Interrupted interactive sessions resume from `.icot/session.yaml`; successful save or `cancel`
  removes the draft.
- `reconcile --example ./examples/<name>` regenerates only `project.md` from existing
  `workflows/intent.hcl` while preserving local policy text.
- `lint --example ./examples/<name>` checks authoring sections, obvious secret-like content,
  intent parsing, and advisory project/intent drift.

For manual authoring, start from `templates/project.md`, then read:

- `docs/project-authoring.md`
- `docs/intent.md`
- `docs/data-flow.md`
- `docs/safety.md`
- `docs/cross-repo-contracts.md`

At minimum, include the goal, integration/OpenAPI policy, runtime policy, credential binding policy,
safety boundary, and fallback behavior. Side-effectful workflows must declare approval/trusted-runner
policy and sandbox/test proof-run policy.

## Generate And Review

```bash
go run ./cmd/ramen synthesize --example ./examples/support-email --provider gemini --model gemini-2.5-flash
go run ./cmd/ramen assess --example ./examples/support-email
```

Inspect:

- `workflows/intent.hcl`
- `workflows/workflow.hcl`
- `workflows/workflow.uws.yaml`
- `expected/plan.md`
- `expected/review.md`
- `expected/symphony-handoff.json`
- `expected/quality.md`

For side-effectful workflows, do not execute production effects from synthesis. Use the trusted
handoff command recorded in `expected/review.md` only after human approval.

## Real-LLM Eval Policy

Real-provider evals are local/manual smoke checks, not routine development checks. Run them when
changing prompts, synthesis/refinement behavior, model defaults, release gates, or broad quality
checks:

```bash
go run ./cmd/ramen eval --root ./examples/eval --provider gemini --model gemini-2.5-flash
```

Eval reports are written under `eval/runs/` and compare against the previous run by default. Use
`--compare eval/runs/<previous>.json` for an explicit baseline, `--no-compare` for an isolated
experiment, and `--archive-dir eval/artifacts` when generated workspaces need to be preserved for
manual inspection. Comparison regressions are visible in every compared report, but only
release-gated runs fail because of comparison regression.

For release candidates:

```bash
make release-eval
```

Release-gated evals require all briefs to pass, zero legacy extraction fallbacks, no brief above two
attempts, zero blocking reference issues, zero secret-scan failures, and no regression against the
selected comparison run. Treat a single real-provider pass as a smoke result, not proof of long-term
stability.
`make release-eval` remains local/manual, uses `RAMEN_PROVIDER` and `RAMEN_MODEL`, and may require
provider credentials.

## Operator Checklist

Use `docs/operator-checklist.md` as the compact path for authoring, deterministic checks, manual
release eval, release notes, and trusted handoff expectations.
