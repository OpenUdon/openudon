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

Normal development gates are:

```bash
go test ./...
go vet ./...
make check
```

## Provider Credentials

Real LLM synthesis and evals need provider credentials from environment variables:

```bash
export GEMINI_API_KEY=...
```

Do not put provider keys, API tokens, SMTP passwords, or other secret values in prompts, examples,
`project.md`, generated artifacts, or eval fixtures. Use credential binding names only.

## Author A Project

Start from `templates/project.md`, then read:

- `docs/project-authoring.md`
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
go run ./cmd/ramen eval --root ./examples/eval --provider gemini --model gemini-2.5-flash --release-gate
```

Release-gated evals require all briefs to pass, zero legacy extraction fallbacks, no brief above two
attempts, zero blocking reference issues, zero secret-scan failures, and no regression against the
selected comparison run. Treat a single real-provider pass as a smoke result, not proof of long-term
stability.
