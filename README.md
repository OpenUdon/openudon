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

## Quick Start

```bash
cd ../ramen
make check
```

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
- `docs/` records architecture and safety boundaries.

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
  --model gemini-2.5-pro \
  --max-attempts 5
```

The command reads `project.md`, discovers or imports OpenAPI documents under `openapi/`, writes
`workflows/intent.hcl`, generates `workflows/workflow.hcl` through udon, exports
`workflows/workflow.uws.yaml`, and writes `expected/plan.json`, `expected/plan.md`,
`expected/discovery.json`, `expected/refinement.json`, `expected/refinement.md`,
`expected/review.md`, `expected/quality.json`, and `expected/quality.md`.

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
go run ./cmd/ramen eval --root ./examples/eval --provider gemini --model gemini-2.5-pro
```

Eval runs synthesize temporary copies of the eval briefs and writes JSON/Markdown summaries under
`eval/runs/`.

LLM credentials must come from provider environment variables such as `GEMINI_API_KEY`,
`OPENAI_API_KEY`, or `ANTHROPIC_API_KEY`; do not place tokens in prompts, commands, examples, or
workflow artifacts.

Before writing a new `project.md`, read `docs/project-authoring.md` and `docs/data-flow.md`, then
start from `templates/project.md`. The brief should include runtime policy, data-flow hints,
credential binding names, safety boundaries, and fallback behavior. For runtime-only projects that
do not need API/OpenAPI integration, include `OpenAPI: none required`.
