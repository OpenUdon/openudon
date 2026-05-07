# Tech Stack

## Memory Bank Index

- This file owns implementation technologies, dependency defaults, and tooling constraints.
- Use [product.md](product.md) for product scope and non-goals.
- Use [architecture.md](architecture.md) for system boundaries and planned structure.
- Use [milestone.md](milestone.md) for milestones.
- Use [status.md](status.md) for current completion state.

Ramen is a private Go integration package that composes sibling modules for UWS, udon runtime
behavior, OpenAPI discovery, and shared review/authoring contracts.

## Language And Runtime

- Primary language: Go.
- Module path: `github.com/genelet/ramen`.
- CLI entrypoints: `cmd/ramen` and `cmd/icot`.
- Internal implementation: reusable behavior under `internal/`.
- Artifact formats: Markdown, HCL, JSON, YAML, UWS YAML, and apitools review handoff JSON.
- Normal verification: `go test ./...`, `go vet ./...`, `make check`, and `git diff --check`.
- Release verification: `make release-check` for deterministic gates and `make release-eval` for
  opt-in real-provider eval gates.

## Local Workspace Dependencies

Ramen expects sibling modules through local `replace` directives:

- `../uws` for public UWS model/schema validation.
- `../udon` for generic UWS/OpenAPI compilation and execution.
- `../grand`, `../golet`, `../hcllight`, `../horizon`, `../molecule`, and `../arazzo` as udon
  build-time siblings.
- `../apitools` for OpenAPI discovery/import helpers, generic authoring primitives, review-only
  leaf adapters, public review state machine, and handoff schema.

`../symphony` is an operational sibling for work orchestration. Ramen configures the workflow policy
but should not import or fork Symphony implementation code.

## Preferred Dependency Direction

- Use `github.com/OpenUdon/apitools` for domain-neutral OpenAPI discovery/import, shared authoring
  mechanics, review state names, review handoff validation, and review-only leaf-adapter contracts.
- Use `github.com/OpenUdon/uws` and sibling schemas for public UWS validation.
- Use `github.com/genelet/udon` packages for generic workflow compilation, UWS export, runtime-plan
  validation, and trusted execution invocation.
- Keep public workflow semantics out of Ramen and put them in `../uws`.
- Keep generic execution/compiler behavior out of Ramen and put it in `../udon`.
- Keep OpenUdon concrete IaC models and `.tf` rendering out of Ramen; use shared `apitools`
  authoring concepts only.

## Artifact Contracts

- Ramen examples use `project.md`, `openapi/`, `workflows/`, and `expected/`.
- `workflows/intent.hcl` is Ramen's structured authoring contract.
- `workflows/workflow.hcl` is udon workflow source.
- `workflows/workflow.uws.yaml` is exported public UWS.
- `expected/plan.json` and `expected/plan.md` describe the expected workflow behavior.
- `expected/refinement.json` and `expected/refinement.md` record bounded repair attempts.
- `expected/review.md` records human review evidence and trusted-runner command text.
- `expected/quality.json` and `expected/quality.md` record deterministic gate results.
- `expected/symphony-handoff.json` uses `apitools.review-handoff.v1`; legacy
  `ramen.symphony-handoff.v1` is accepted only for read compatibility.

## LLM And Provider Policy

- Local real-LLM synthesis defaults to the `copilot-api` OpenAI-compatible proxy at
  `http://localhost:4141` with model `gpt-5.4-mini`.
- Escalate to a larger model only after the default proxy model fails deterministic checks.
- Provider credentials and proxy endpoints come from environment variables such as
  `COPILOT_API_BASE_URL`, `COPILOT_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, or
  `ANTHROPIC_API_KEY`.
- Never paste credentials into prompts, commands, examples, review evidence, or eval artifacts.
- Real-provider evals are local/manual because they spend quota and may produce artifacts that need
  redaction review.

Provider drift release evidence should record provider, model, commit, comparison baseline,
`provider_drift_watch.status`, structured fallback count, maximum attempts, release-gate result,
and any provider error text. Do not loosen release criteria from a single transient run; rerun once
from a trusted workstation if the error looks external.

## Trusted Runner Contract

`ramen approval-template` prints approval JSON with:

- `version`: `ramen.approval.v1`
- `scope`: example path relative to repo root
- `state`: `approved_for_sandbox` or `approved_for_production`
- `reviewer`, `approved_at`, optional `expires_at`, optional `notes`
- `package_sha256`: canonical digest of required handoff inputs

`ramen run` validates approval, package digest, stored and current quality, manifest policy,
credential-value prohibition, direct-production prohibition, and tier/state compatibility before
invoking udon by argv.

## Tooling Constraints

- Keep `cmd/ramen` and `cmd/icot` thin.
- Keep scripts small wrappers around Go behavior where practical.
- Do not add product-specific behavior to `../uws` or core `../udon`.
- Prefer deterministic checks and synthetic fixtures over live-provider tests during development.
- Keep generated eval outputs, readiness reports, approvals, transcripts, autosaves, and run
  workdirs ignored unless explicitly converted into reviewed fixtures.
- Keep release note evidence in Markdown templates and ignored report paths; do not commit
  real-provider JSON/Markdown eval outputs.
