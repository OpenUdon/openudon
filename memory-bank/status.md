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
- [x] Cross-repo runtime boundary clarified: public `x-uws-runtime` is a slim non-HTTP invocation
  selector with no HTTP/OpenAPI metadata, provider/security configuration, or request/response
  schemas; `type: http` is rejected while udon's legacy private `x-udon-runtime` HTTP projection
  remains compatibility-only pending a separate migration. Runtime auth/security shapes for the
  supported non-HTTP runtime types are implementation-specific and typically belong in arguments or
  private runtime configuration, not a public runtime config object.
- [x] Local readiness report implemented for sibling checkouts, deterministic gates, git state,
  ignored artifacts, provider env presence booleans, and local/manual automation policy.
- [x] `expected/symphony-handoff.json` implemented with the stable `apitools.review-handoff.v1`
  wire version and legacy read compatibility; Ramen owns validation and lifecycle behavior.
- [x] Approval template generation and local trusted-runner validation implemented.
- [x] Ramen authoring compatibility adapter completed without importing concrete IaC
  models.
- [x] OpenAPI discovery remains available through `apitools/openapidisco`. Ramen now owns
  `.icot/session.yaml` draft persistence, the progressive iCoT loop, transcript lifecycle, and JSON
  completion fallback locally under `internal/authoring`.
- [x] Ramen progressive iCoT uses `apitools` only for OpenAPI authoring documents, operation
  summaries, and ranking while keeping rollout-specific prompts, sanitization, readiness checks,
  final edit/explain confirmation, artifact rendering, and lifecycle plumbing local.
- [x] Ramen Symphony handoff assembly and trusted-runner package digest now use local Ramen handoff
  input, binding contract, validation, and digest helpers while keeping the
  `apitools.review-handoff.v1` wire shape stable.
- [x] Ramen quality checks no longer depend on udon runtime plans. Request evidence is projected from public UWS operation request maps with indexed expression precision, and Ramen implementation imports no udon Go packages.
- [x] Udon HTTP credential binding resolution implemented upstream: OpenAPI security schemes carry
  non-secret binding names from `x-udon-config.security[].binding` or default to the scheme name,
  the default resolver reads `UDON_CREDENTIAL_<BINDING>` at execution time, literal secret fields in
  `x-udon-config.security` are rejected, and resolved values stay out of Ramen artifacts and
  persisted udon output.
- [x] Hosted CI intentionally disabled during active private-sibling development.
- [x] Roadmap, XRD, onboarding, operator, and safety docs consolidated into memory-bank and README.
- [x] Direction change recorded: Ramen is now the public UWS authoring, review, package, and
  executor-handoff tool; Symphony is optional orchestration; `apitools` is being narrowed back to
  OpenAPI document tooling; udon remains a private executor boundary.
- [x] First Ramen-local lifecycle migration implemented: progressive iCoT loop, draft/transcript
  lifecycle, prompt replay types, JSON completion fallback, artifact/review metadata, handoff
  validation, package digest, symbolic binding contract, and credential scanning now live under
  `internal/authoring`.
- [x] Ramen lifecycle migration split from the broader apitools narrowing blocker. Ramen now keeps
  product lifecycle behavior local and should use `../apitools` only for OpenAPI discovery,
  import/search, operation indexing, prompt-safe summaries, security/auth summaries, ranking, and
  cache-backed CLI support.
- [x] `../apitools` hard-narrowed to OpenAPI tooling: discovery, search/import/download, local file
  scanning, validation, operation inventory/indexing/summaries, auth/security summaries, operation
  ranking, CLI search/import, `openapidisco`, and `sqlitecache`. Old lifecycle APIs, LLM providers,
  Context7, iCoT helpers, review handoff/state machine, package digest, credential scan, binding
  contract, and leaf/review package helpers are removed.
- [x] `../udon` migrated off non-OpenAPI apitools APIs. Udon now owns rollout LLM provider plumbing
  and runtime-plan review/handoff helper types locally while keeping only OpenAPI search/import
  usage from apitools.
- [x] The IaC sibling is parked and is not a compatibility gate for the apitools narrowing.
- [x] Split Ramen's remaining udon executor coupling into a CLI/Docker-compatible trusted executor
  handoff based on UWS Document, OpenAPI files, non-secret run config, and runtime credential
  resolution.
- [x] Trusted-runner hardening added: OpenAPI files staged for execution are required handoff
  inputs covered by package digests, symlinked OpenAPI artifacts are rejected, execution uses a
  fresh staged workdir, Docker execution passes only declared `UDON_CREDENTIAL_*` names, and invalid
  OpenAPI operation IDs fail generation instead of dropping request bindings.
- [x] Review follow-up safety hardening is complete. The first Ramen-owned package artifact
  hardening pass is implemented for required handoff files, and `../apitools` local OpenAPI reads
  now fail closed on symlinked roots/paths/parents, directories, special files, and oversized
  path-backed documents. `scripts/run-udon.sh` no longer uses line-delimited run-config parsing;
  config validation and workflow/OpenAPI staging happen in Python before executor argv handoff. The
  `workflowintent` package is split by responsibility into intent/HCL, provider-client, OpenAPI,
  and helper files without changing its package boundary or exported API.

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
- `ramen run` is the only Ramen-owned path that invokes a trusted executor shim, and it requires
  approval JSON plus a valid handoff package. It writes a non-secret `ramen.executor-run.v1` run
  config over UWS YAML, packaged OpenAPI files, package digest, tier, workdir, and credential
  binding names. Docker execution receives only the declared `UDON_CREDENTIAL_*` environment names.
- Symphony managed reviewer routing remains optional external integration. Ramen owns local package
  evidence and trusted-runner enforcement only.
- Concrete IaC behavior remains outside Ramen. That sibling is parked during this narrowing and is not
  a release compatibility gate.
- Udon consumer migration for the narrowed apitools boundary is complete.
- Keep `apitools.review-handoff.v1` only as a stable wire compatibility string while downstream
  artifacts still need it; do not treat it as active `../apitools` lifecycle ownership.
- Ramen no longer imports udon as a Go module; udon is an optional external trusted executor behind
  the run-config handoff.
- Review regressions in the udon separation slice are closed locally: provider-native structured
  output and copilot GPT-5 Responses routing are restored, OpenAPI request placement is inferred
  from public `apitools` summaries and fails on missing operation IDs, trigger routes/options are
  preserved, UWS 1.1 is selected for timeout/idempotency artifacts, and the trusted runner stages
  digest-covered artifacts into a fresh workdir before CLI/Docker invocation.
- Remaining detailed docs are intentionally narrow working references: intent contract, data-flow
  examples, project authoring guide, eval gallery, release-note template, and safety guide.
- Update this file after feature implementation changes completion state.
- Update [milestone.md](milestone.md) in the same change when a high-level plan or acceptance
  criterion changes.
- After a major review or milestone, check whether [evolution/](../evolution/) needs a new
  prompt/result version.
