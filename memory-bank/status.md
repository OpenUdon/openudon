# Status

## Memory Bank Index

- This file owns current completion state.
- Use [milestone.md](milestone.md) for milestone detail and acceptance criteria.
- Use [product.md](product.md), [architecture.md](architecture.md), and [tech-stack.md](tech-stack.md)
  for current project direction.

## Current State

- [x] Initial product direction documented.
- [x] Memory bank scaffold created.
- [x] Evolution v1 scaffold created.
- [x] Go module and private sibling dependency layout established.
- [x] Thin `cmd/ramen` CLI implemented for local checks, synthesis, build, promote, assess, eval,
  readiness, approval template, and trusted run commands.
- [x] Guided `cmd/icot` authoring CLI implemented.
- [x] `project.md` plus `workflows/intent.hcl` authoring model implemented.
- [x] Deterministic artifact generation implemented for workflow HCL, UWS, plans, discovery,
  refinement, review, handoff, and quality reports.
- [x] Bounded refinement loop implemented and recorded in `expected/refinement.json`.
- [x] Deterministic quality gates implemented for project policy, OpenAPI availability, intent,
  data flow, workflow compilation, expected plan, UWS validation, review evidence, handoff policy,
  credentials, side effects, and secret scanning.
- [x] Eval harness implemented with reference comparison, run comparison, release gates, provider
  drift reporting, and ignored run artifacts.
- [x] Expanded eval corpus and per-fixture reference policy support implemented.
- [x] Structured-output reporting and fallback regression detection implemented.
- [x] iCoT autosave, transcript, reconcile, lint, replay, OpenAPI ranking, and atomic writes
  implemented.
- [x] UWS 1.1 timeout and workflow idempotency opt-in preservation covered by Ramen eval/quality.
- [x] Runtime/profile eval coverage implemented for approved `fnct`, approved `cmd`, denied
  command/SSH, and future profile-boundary behavior.
- [x] Local readiness report implemented for sibling checkouts, deterministic gates, git state,
  ignored artifacts, provider env presence booleans, and local/manual automation policy.
- [x] `expected/symphony-handoff.json` implemented on the public `apitools.review-handoff.v1`
  schema with legacy read compatibility.
- [x] Approval template generation and local trusted-runner validation implemented.
- [x] Ramen adoption of shared `apitools` authoring core compatibility adapter completed without
  importing OpenUdon concrete IaC models.
- [x] OpenAPI discovery + `.icot/session.yaml` draft persistence relocated to
  `apitools/openapidisco` and `apitools/icot` (2026-05-07). Ramen keeps thin shims at
  `internal/openapidisco` and `internal/icot/elicitor/draft.go` so existing call sites compile
  unchanged. The conversation engine itself was NOT extracted: `apitools.RunProgressiveICOT[S, D, A]`
  + `ProgressiveLoopHooks[S, D, A]` already implement the bound-runtime pattern, so `ramen`'s
  rollout-shaped `Extractor`, `classification`, `progressive`, `loop`, `session`, `api` files
  remain in `internal/icot/elicitor/` as the rollout binding of those generic hooks.
- [x] Ramen progressive iCoT now inherits shared `apitools` authoring OpenAPI documents,
  draft/transcript lifecycle, and JSON completion fallback while keeping rollout-specific prompts,
  sanitization, readiness checks, final edit/explain confirmation, and artifact rendering local.
- [x] Ramen Symphony handoff assembly and trusted-runner package digest now inherit shared `apitools`
  handoff input, binding contract, and digest helpers while keeping Ramen owner split, approval
  policy, quality gates, trusted-runner command text, and udon invocation local.
- [x] Ramen quality checks no longer import `hcllight` directly. Compiled request evidence is
  projected through `udon/pkg/runtimeplan` as plain recursive request maps with indexed expression
  precision, keeping `github.com/genelet/udon` as the only private Go module named by Ramen
  implementation imports.
- [x] Hosted CI intentionally disabled during active private-sibling development.
- [x] Roadmap, XRD, onboarding, operator, and safety docs consolidated into memory-bank and README.

## Notes

- Historical full real-LLM smoke baseline in README: 2026-04-28, `gemini-2.5-flash`, structured
  output path, ten original examples passed with zero legacy extraction fallbacks. Current local
  real-LLM defaults use `copilot-api` with `gpt-5.4-mini`.
- Current release evidence should use the expanded eval corpus through `make release-eval`; real
  provider outputs remain ignored and local/manual.
- Advisory n8n reducibility fixtures remain part of the eval corpus. They should not introduce
  n8n-specific runtime behavior into Ramen or udon; use explicit intent, OpenAPI, and generic
  `fnct` or control-flow modeling.
- Normal deterministic gates remain `go test ./...`, `go vet ./...`, `make check`, and
  `git diff --check`.
- Ramen synthesis commands generate and validate artifacts only. They do not execute production
  workflows.
- `ramen run` is the only Ramen-owned path that invokes udon, and it requires approval JSON plus a
  valid handoff package.
- Symphony managed reviewer routing remains optional external integration. Ramen owns local package
  evidence and trusted-runner enforcement only.
- OpenUdon remains the owner of concrete IaC behavior. Ramen uses generic `apitools` authoring
  abstractions only for private workflow/iCoT reuse.
- Remaining detailed docs are intentionally narrow working references: intent contract, data-flow
  examples, project authoring guide, eval gallery, release-note template, and safety guide.
- Update this file after feature implementation changes completion state.
- Update [milestone.md](milestone.md) in the same change when a high-level plan or acceptance
  criterion changes.
- After a major review or milestone, check whether [evolution/](../evolution/) needs a new
  prompt/result version.
