# Tech Stack

## Memory Bank Index

- This file owns implementation technologies, dependency defaults, and tooling constraints.
- Use [product.md](product.md) for product scope and non-goals.
- Use [architecture.md](architecture.md) for system boundaries and planned structure.
- Use [milestone.md](milestone.md) for milestones.
- Use [status.md](status.md) for current completion state.

OpenUdon is a Go package and CLI that composes sibling modules for public UWS modeling, narrowed
OpenAPI discovery/indexing, and portable trusted executor handoff.

## Language And Runtime

- Primary language: Go.
- Module path: `github.com/OpenUdon/openudon`.
- CLI entrypoints: `cmd/openudon`, `cmd/icot`, and `cmd/udon-runner`.
- Internal implementation: reusable behavior under `internal/`.
- Artifact formats: Markdown, HCL, JSON, YAML, UWS YAML, and review handoff JSON.
- Normal verification: `go test ./...`, `go vet ./...`, `make check`, and `git diff --check`.
  Local repository guard checks, UWS artifact validation, and trusted executor staging are Go CLI
  commands/packages under `cmd/` and `internal/`.
- Release verification: `make release-check` for deterministic gates and `make release-eval` for
  opt-in real-provider eval gates.

## Module Dependencies

OpenUdon is published as a normal Go module. Its committed `go.mod` must use public module versions
for:

- `github.com/OpenUdon/uws` for public UWS model/schema validation.
- `github.com/OpenUdon/apitools` for OpenAPI discovery, import, search, indexing, auth/security summaries, and
  operation ranking.

Local sibling development belongs in the parent `go.work`, not in committed `replace ../...`
directives. Verify public-module behavior with `GOWORK=off go test ./...` and
`GOWORK=off go vet ./...`.

`../symphony` is an operational sibling for work orchestration. OpenUdon configures the workflow policy
but should not import or fork Symphony implementation code.

Udon is not a OpenUdon Go dependency. When used, it is an external trusted executor selected at runtime by `OPENUDON_EXECUTOR`, `OPENUDON_UDON_BIN`, `OPENUDON_UDON_RUNNER`, or `OPENUDON_UDON_IMAGE`.
`OPENUDON_EXECUTOR`, `OPENUDON_UDON_BIN`, and `OPENUDON_UDON_RUNNER` must be absolute executable paths.
`OPENUDON_UDON_IMAGE` remains a Docker image reference chosen by the operator.

## Preferred Dependency Direction

- Use `github.com/OpenUdon/uws` and sibling schemas for public UWS parsing and validation.
- Use narrowed `github.com/OpenUdon/apitools` APIs only for OpenAPI discovery/import/search,
  indexing, summaries, and ranking.
- Keep OpenUdon-owned authoring, iCoT, review evidence, approval, credential scanning, package digest,
  and handoff helpers under `internal/`.
- Non-OpenAPI `apitools` APIs have been removed from the active boundary. Do not add temporary
  lifecycle compatibility shims back to apitools.
- OpenUdon generates public UWS documents directly and invokes executors only through UWS Document +
  OpenAPI files + non-secret run config + runtime credential resolver.
- Keep udon runtime-plan helpers and private HCL body representations such as `hcllight/light.Body`
  behind udon APIs. OpenUdon may consume public maps and UWS documents, but must not inspect udon's
  private runtime-plan or HCL AST types directly.
- Keep public workflow semantics out of OpenUdon and put them in `../uws`.
- Keep generic execution/compiler behavior out of OpenUdon and put it in `../udon`.
- Keep concrete IaC models and `.tf` rendering out of OpenUdon; use `apitools` only for
  OpenAPI tooling.

## Artifact Contracts

- OpenUdon examples use `project.md`, `openapi/`, `workflows/`, and `expected/`.
- `workflows/intent.hcl` is OpenUdon's structured authoring contract.
- `workflows/workflow.hcl` is public UWS HCL.
- `workflows/workflow.uws.yaml` is equivalent public UWS YAML.
- `expected/plan.json` and `expected/plan.md` describe the expected workflow behavior.
- `expected/refinement.json` and `expected/refinement.md` record bounded repair attempts.
- `expected/review.md` records human review evidence and trusted-runner command text.
- `expected/quality.json` and `expected/quality.md` record deterministic gate results.
- `expected/symphony-handoff.json` uses `apitools.review-handoff.v1`; legacy
  `openudon.symphony-handoff.v1` is accepted only for read compatibility.

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

`openudon approval-template` prints approval JSON with:

- `version`: `openudon.approval.v1`
- `scope`: example path relative to repo root
- `state`: `approved_for_sandbox` or `approved_for_production`
- `reviewer`, `approved_at`, optional `expires_at`, optional `notes`
- `package_sha256`: canonical digest of required handoff inputs

`openudon run` validates approval, package digest, stored and current quality, manifest policy,
credential-value prohibition, direct-production prohibition, and tier/state compatibility before
writing `openudon.executor-run.v1`. The run config includes sorted `package_paths` for every
digest-covered required handoff input. The digest covers the reviewed UWS artifacts and every
regular OpenAPI file staged for execution. The shim stages the reviewed package paths into a fresh
executor-visible directory under the configured run workdir, recomputes the approved package digest
from the staged copy, fails if it does not match `package_sha256`, fails if a declared credential
binding is missing from the local `UDON_CREDENTIAL_*` environment, then invokes either a trusted
absolute-path udon-compatible binary by argv or a Docker image via `OPENUDON_UDON_IMAGE`. Docker mode
passes only declared credential env names, not all host environment variables.

## Tooling Constraints

- Keep `cmd/openudon` and `cmd/icot` thin.
- Keep scripts small wrappers around Go behavior where practical. Prefer `cmd/openudon` subcommands
  for local repository checks.
- Do not add product-specific behavior to `../uws` or core `../udon`.
- Prefer deterministic checks and synthetic fixtures over live-provider tests during development.
- Keep generated eval outputs, readiness reports, approvals, transcripts, autosaves, and run
  workdirs ignored unless explicitly converted into reviewed fixtures.
- Keep release note evidence in Markdown templates and ignored report paths; do not commit
  real-provider JSON/Markdown eval outputs.
