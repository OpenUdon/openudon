# Ramen

Ramen is a private integration layer for Symphony-managed UWS projects executed by the private
`udon` runtime.

It connects three sibling projects:

- `../symphony` manages work items, isolated agent workspaces, and Codex app-server sessions.
- `../uws` defines the public UWS workflow document model, schema, validation, and execution
  semantics.
- `../udon` validates, lowers, and executes approved UWS/OpenAPI workflow artifacts.

Ramen should stay thin. It owns project templates, Symphony workflow policy, example artifacts,
and trusted execution glue. Generic workflow semantics belong in `../uws`; reusable execution
capabilities belong in `../udon`.

## Status

Early private scaffold. Do not treat this repository as a public API yet.
See `TODO.md` for the post-POC hardening roadmap and current product-readiness gaps.

## Quick Start

```bash
cd ../ramen
make check
```

New trusted operators should start with `docs/onboarding.md`, then use `templates/project.md` when
authoring a new project brief.
For the short operator path from authoring through release evidence, see
`docs/operator-checklist.md`.

The scaffold expects the following sibling directories:

```text
../symphony
../uws
../udon
```

Because Ramen imports udon packages directly, local builds also need udon's private build-time
siblings used by `../udon/go.mod`, including `../grand`, `../golet`, `../hcllight`, `../horizon`,
`../molecule`, and `../arazzo`.

## Layout

- `WORKFLOW.md` is the Symphony workflow prompt/config template for UWS/OpenAPI work.
- `cmd/ramen` is a small Go CLI for local checks.
- `examples/support-email` is the first natural-language-to-UWS example.
- `scripts/` contains local validation and execution wrappers.
- `docs/` records architecture, safety boundaries, operator checklist, and cross-repo contracts.
- `templates/project.md` is the starting point for new project briefs.

## Execution Boundary

The intended lifecycle is:

```text
natural-language project brief
  -> Symphony-managed implementation issue
  -> generated OpenAPI/UWS artifacts
  -> validation and review
  -> approved artifact
  -> udon execution by a trusted runner
```

Agents may generate and validate artifacts. Production side effects should happen only through an
approved trusted runtime path.

## Synthesize An Example

Ramen can turn an example brief into reviewed artifacts:

```bash
export GEMINI_API_KEY=...
go run ./cmd/ramen synthesize \
  --example ./examples/support-email \
  --provider gemini \
  --model gemini-2.5-flash \
  --max-attempts 5
```

The command reads `project.md`, discovers or imports OpenAPI documents under `openapi/`, writes
`workflows/intent.hcl`, generates `workflows/workflow.hcl` through udon, exports
`workflows/workflow.uws.yaml`, and writes `expected/plan.json`, `expected/plan.md`,
`expected/discovery.json`, `expected/refinement.json`, `expected/refinement.md`,
`expected/review.md`, `expected/symphony-handoff.json`, `expected/quality.json`, and
`expected/quality.md`.

The expected plan records inferred steps, OpenAPI operations, required request fields,
step-to-step bindings, and credential-like parameters. `ramen assess` compiles the final
`workflow.hcl` through udon and checks it still matches that plan.

`synthesize` and `build` run a bounded refinement loop, defaulting to five attempts. The loop records
which stage was retried, which checks failed, and why it stopped in `expected/refinement.json`.
Use narrower stages after editing artifacts:

```bash
go run ./cmd/ramen build --example ./examples/support-email --max-attempts 5  # intent.hcl -> workflow/UWS
go run ./cmd/ramen promote --example ./examples/support-email  # workflow.hcl -> UWS
go run ./cmd/ramen assess --example ./examples/support-email   # quality reports only
```

Use the eval harness when changing prompts or synthesis behavior:

```bash
go run ./cmd/ramen eval --root ./examples/eval --provider gemini --model gemini-2.5-flash
```

Eval runs synthesize temporary copies of the eval briefs and write JSON/Markdown summaries under
`eval/runs/`. Reports include run metadata, pass/fail summaries, provider/model/mode/prompt-version
breakdowns, approximate prompt-token totals, generated workspace paths, and comparison against the
previous report in the output directory when one exists.

### Real LLM Eval Smoke

Real-provider evals are useful smoke tests for prompt, model, and refinement-loop changes, but they
are not part of the normal development loop. They require provider credentials, spend external model
quota, can vary run to run, and may fail for transient provider reasons. For routine code changes,
prefer deterministic checks:

```bash
go test ./...
make check
```

Use `make vet` when you need explicit vet parity with release documentation. For deterministic
release readiness, run:

```bash
make release-check
```

For XRD-007 local/private checkout evidence, write a structured readiness report:

```bash
go run ./cmd/ramen readiness --run-gates --out eval/readiness/local.json
```

The readiness report records sibling checkouts, deterministic gate results, git state, ignored local
artifact paths, provider credential environment presence as booleans only, and the current
local/manual automation policy.

Run a real LLM eval when changing prompt templates, synthesis/refinement behavior, model defaults, or
quality gates that could affect generated artifacts:

```bash
export GEMINI_API_KEY=...
go run ./cmd/ramen eval --root ./examples/eval --provider gemini --model gemini-2.5-flash
```

Use `--compare eval/runs/<previous>.json` to compare against a specific run, or `--no-compare` for
an isolated smoke. Normal evals print comparison regressions but do not fail the command for them.
Release-gated evals fail on both absolute release criteria and comparison regressions.

Temporary generated workspaces are listed in the report for manual inspection. To preserve them
under the repo-local ignored artifact directory, add:

```bash
go run ./cmd/ramen eval --root ./examples/eval --provider gemini --model gemini-2.5-flash --archive-dir eval/artifacts
```

Ramen records approximate prompt-token counts and has report fields for provider-reported token
usage or cost when a provider path exposes them. It does not hardcode provider pricing tables.
Eval JSON reports include a `provider_drift_watch` block, and Markdown reports render the same
XRD-006 signals: structured fallback count, rate/transient failures, model availability,
attempts-to-pass, and release-gate failures.

For a candidate release smoke, add the local release gate:

```bash
make release-eval
```

The release gate requires a 100% pass rate, structured-mode usage with zero legacy extraction
fallbacks, no brief above two refinement attempts, no blocking reference issues after each fixture's
`reference/policy.json` is applied, and zero `artifacts.no_secrets` failures.
`make release-eval` uses `RAMEN_PROVIDER` and `RAMEN_MODEL`, defaulting to `gemini` and
`gemini-2.5-flash`. It also passes the current eval corpus size as a minimum brief gate; see
`docs/xrd-009-expanded-corpus-release-evidence.md`.

Development gates and real-provider release automation remain local/manual through `make check` and
`make release-eval`.

## Trusted Execution Wrapper

After artifacts pass review, generate approval JSON with the current handoff package digest:

```bash
go run ./cmd/ramen approval-template \
  --example ./examples/support-email \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/support-email-sandbox.json
```

Then validate approval, quality, handoff policy, package digest, and tier compatibility before udon
execution:

```bash
go run ./cmd/ramen run \
  --example ./examples/support-email \
  --tier sandbox \
  --approval approvals/support-email-sandbox.json
```

Use `--dry-run` to validate the gates without invoking udon. The full wrapper plan and approval
schema are in `SYMPHONY_WRAPPER.md`.

Last full passing real-LLM smoke: 2026-04-28, `gemini-2.5-flash`, prompt `intent.v3`, structured
output path with `0` legacy extraction fallbacks.

| Eval brief | Result | Mode | Attempts | Reference issues (A/W/B) |
| --- | --- | --- | ---: | --- |
| `cmd-allowed-deploy` | pass | structured | 1 | 2/2/0 |
| `cmd-disallowed-deploy` | pass | structured | 1 | 2/2/0 |
| `crm-note-write` | pass | structured | 1 | 1/0/0 |
| `customer-export-two-pages` | pass | structured | 2 | 4/0/0 |
| `inventory-api-key-binding` | pass | structured | 2 | 0/0/0 |
| `paginated-list` | pass | structured | 1 | 0/0/0 |
| `runtime-only-render` | pass | structured | 1 | 0/0/0 |
| `support-email` | pass | structured | 1 | 5/0/0 |
| `support-priority-routing` | pass | structured | 1 | 5/0/1 |
| `weather-toronto` | pass | structured | 1 | 2/0/0 |

Summary: all ten examples passed deterministic quality gates. Eval reports classify reference drift
as advisory/warning/blocking and apply each fixture's `reference/policy.json`. Runtime type,
selected OpenAPI operation, and parse/compare failures are behavioral drift; step names, output
names, request literal names, and bind field names are semantic hints by default.

Structured-output smoke on 2026-04-28 recovered the ten-example quality baseline with
`gemini-2.5-flash`: 10/10 passed, all in structured mode, with `0` legacy fallbacks.

### Model Selection

Use `gemini-2.5-flash` as the default Gemini model for synthesis. Ramen's task is mostly structured
extraction and artifact generation, and reliability comes from prompt preprocessing, structured
output when available, deterministic quality gates, and bounded repair attempts. In local smoke
runs, Flash was fast enough to complete the `runtime-only-render` eval after validation fixes, while
Gemini Pro preview models were slower and more prone to timeout or temporary capacity errors.

Escalate to a larger model such as `gemini-2.5-pro` only after Flash fails deterministic checks
after one or two attempts. Prefer a fast validated retry over a slow preview model as the default.

LLM credentials must come from provider environment variables such as `GEMINI_API_KEY`,
`OPENAI_API_KEY`, or `ANTHROPIC_API_KEY`; do not place tokens in prompts, commands, examples, or
workflow artifacts.

Before writing a new `project.md`, read `docs/project-authoring.md` and `docs/data-flow.md`, then
start from `templates/project.md`. The brief should include runtime policy, data-flow hints,
credential binding names, safety boundaries, and fallback behavior. For runtime-only projects that
do not need API/OpenAPI integration, include `OpenAPI: none required`.
