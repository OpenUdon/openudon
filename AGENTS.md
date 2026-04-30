# AGENTS.md

## Purpose

Ramen is a private Go integration layer for Symphony-managed UWS projects executed by the private `udon` runtime.

Ramen owns project templates, Symphony workflow policy, example artifacts, validation wrappers, and trusted execution glue.

## Boundaries

- `../uws` is the public UWS specification and Go model. Put public workflow semantics there.
- `../udon` is the private UWS/OpenAPI compiler and runtime. Put generic execution/compiler capabilities there.
- `../symphony` is the work orchestration service. Configure it through `WORKFLOW.md`; do not fork or modify it from Ramen unless explicitly requested.
- Udon's private build-time siblings (`../grand`, `../golet`, `../hcllight`, `../horizon`,
  `../molecule`, and `../arazzo`) and the shared `../openapisearch` module must be present for
  local Go builds.
- `../ramen` owns only the integration layer above those projects.

Rule of thumb:

- If it changes public workflow semantics, it belongs in `../uws`.
- If it improves generic UWS/OpenAPI execution, it belongs in `../udon`.
- If it manages Symphony-driven project workflow, templates, examples, or trusted execution glue, it belongs in Ramen.

## Commands

```bash
go test ./...
go run ./cmd/ramen check
./scripts/check-siblings.sh
./scripts/validate-uws.sh ./examples
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
