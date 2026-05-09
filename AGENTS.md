# AGENTS.md

## Purpose

Ramen is a private Go integration layer for Symphony-managed UWS projects executed by the private `udon` runtime.

Ramen owns project templates, Symphony workflow policy, example artifacts, validation wrappers, and trusted execution glue.

## Memory Bank First

Before making substantial changes, read in this order:

1. [memory-bank/product.md](memory-bank/product.md)
2. [memory-bank/architecture.md](memory-bank/architecture.md)
3. [memory-bank/tech-stack.md](memory-bank/tech-stack.md)
4. [memory-bank/milestone.md](memory-bank/milestone.md)
5. [memory-bank/status.md](memory-bank/status.md)

Use the memory bank as the active project source of truth. Do not recreate
duplicate root-level product, architecture, roadmap, or status documents.

## Boundaries

- `../uws` is the public UWS specification and Go model. Put public workflow semantics there.
- `../udon` is the private UWS/OpenAPI compiler and runtime. Put generic execution/compiler capabilities there.
- `../symphony` is the work orchestration service. Configure it through Ramen policy and README
  operator guidance; do not fork or modify it from Ramen unless explicitly requested.
- Ramen source code must not import `../udon`, udon's private build-time siblings, or any private
  `genelet/*` executor module. Ramen invokes udon only as an external CLI or Docker executor through
  the trusted run-config handoff.
- `../apitools` owns narrowed OpenAPI tooling only. Ramen owns review state, handoff validation,
  approval templates, package contents, and local trusted-runner enforcement.
- `../openw8m` owns concrete IaC authoring/planning and is parked; it is not a Ramen compatibility
  gate while the OpenAPI-only apitools boundary is active.
- `../ramen` owns only the integration layer above those projects.

Rule of thumb:

- If it changes public workflow semantics, it belongs in `../uws`.
- If it improves generic UWS/OpenAPI execution, it belongs in `../udon`.
- If it manages Symphony-driven project workflow, templates, examples, approval
  routing, or trusted execution glue, it belongs in Ramen.

## Commands

```bash
go test ./...
go run ./cmd/ramen check
go run ./cmd/ramen check-apitools-boundary
go run ./cmd/ramen check-doc-memory
go run ./cmd/ramen validate ./examples
make check
```

## Architecture

Natural-language project brief -> Symphony issue/workspace -> Codex-generated OpenAPI/UWS artifacts -> validation/review -> approved artifact -> udon execution by trusted runner.

Agents may generate and validate artifacts. Production side effects must only happen through an approved trusted runtime path.

## Go Conventions

- Primary language is Go.
- Keep `cmd/ramen` thin.
- Put reusable logic under `internal/`.
- Keep scripts small wrappers around Go behavior when possible.
- Do not add product-specific behavior to `../uws` or core `../udon`.

## Safety

- Do not put secrets in prompts, examples, or committed workflow artifacts.
- Treat generated UWS/OpenAPI/HCL as untrusted until validated.
- Do not execute side-effectful workflows unless explicitly requested.
- Prefer sandbox/test endpoints for proof runs.

## Documentation Rules

- Update [memory-bank/status.md](memory-bank/status.md) after feature implementation changes
  completion state.
- Update [memory-bank/milestone.md](memory-bank/milestone.md) when sequencing, milestone scope,
  acceptance criteria, or cross-repo contracts change.
- Update [memory-bank/product.md](memory-bank/product.md) when product scope, users, workflows,
  concepts, or non-goals change.
- Update [memory-bank/architecture.md](memory-bank/architecture.md) when system boundaries, data
  flow, artifact layout, or security boundaries change.
- Update [memory-bank/tech-stack.md](memory-bank/tech-stack.md) when dependencies, commands,
  runtime assumptions, artifact schemas, or tooling choices change.
- Keep README focused on operator entry points and concise command guidance. Put durable project
  memory in `memory-bank/`.

## Evolution Rules

- Check [evolution/](evolution/) after a major review, milestone, or boundary change.
- Create the next `prompt-vN.md` and `result-vN.md` only when product direction, architecture
  boundary, milestone target, or public/private contract direction materially changes.
- Keep the current evolution version when implementation only advances the existing direction.
- When adding a new evolution version, reconcile it with `memory-bank/` in the same change.
