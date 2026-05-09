# Milestone

## Memory Bank Index

- This file owns milestones, work sequencing, and acceptance criteria.
- Use [product.md](product.md) for product scope and non-goals.
- Use [architecture.md](architecture.md) for system boundaries and planned structure.
- Use [tech-stack.md](tech-stack.md) for dependency and tooling defaults.
- Use [status.md](status.md) for current completion state.

## Delivery Strategy

Keep OpenUdon as the public UWS authoring, review, package, and executor-handoff tool. Build in
verifiable slices: authoring and examples, deterministic synthesis, quality gates, eval and release
evidence, trusted handoff, readiness, and cross-repo compatibility. Push public workflow semantics
to `../uws`, OpenAPI search/discovery/import/indexing to `../apitools`, reusable execution to
private executors such as `../udon`, and optional orchestration to `../symphony`.

## Milestones

### 1. Post-POC Baseline

- Preserve the known-good eval and synthesis baseline.
- Keep deterministic quality gates covering project policy, OpenAPI availability, intent validity,
  workflow compilation, expected-plan matching, UWS validation, review evidence, and secret scanning.
- Document deterministic checks versus optional real-provider smoke tests.

Acceptance: committed docs and eval reports describe current behavior without overstating public API
stability or release readiness.

### 2. Eval Corpus And Reference Discipline

- Expand curated eval briefs across OpenAPI auth schemes, pagination, request bodies, response
  extraction, writes, multi-service chains, runtime-only functions, approved/denied runtimes, and
  negative policy cases.
- Keep advisory n8n reducibility fixtures for Airtable, Gmail, Google Drive, HubSpot, Jira,
  OpenWeatherMap, PagerDuty, Slack, and Trello as evidence that scanner/OpenAPI-backed workflows
  can be expressed as OpenUdon intent without adding n8n-specific runtime behavior.
- Classify reference issues as advisory, warning, or blocking.
- Keep per-fixture reference policies and triage notes.
- Gate release evals on pass rate, structured mode, attempts, blocking reference issues, and
  secret-scan failures.

Acceptance: eval reports identify behavioral regressions separately from acceptable naming or
review-text drift.

### 3. Structured Output And Provider Drift

- Use provider-native structured generation where supported and keep legacy extraction as fallback.
- Report structured fallback count, provider failures, model availability, attempts-to-pass, release
  gate failures, and comparison deltas.
- Keep real-provider evals local/manual until protected secret and redaction automation exists.

Acceptance: release evidence can distinguish deterministic OpenUdon regressions from provider drift.

### 4. Quality Gate Hardening

- Tighten data-flow checks for missing dependencies, ambiguous sources, invalid response paths, and
  undeclared function inputs.
- Harden credential checks around binding declarations, OpenAPI security schemes, request placement,
  and secret-value leakage.
- Expand side-effect checks for write operations, customer communications, command/SSH runtimes, and
  production endpoint language.
- Require review evidence for side effects, unresolved risks, skipped execution, credential binding
  names, approval states, sandbox proof runs, and trusted-runner handoff.

Acceptance: common artifact mistakes fail with precise quality codes and concrete repair guidance.

### 5. Workflow Artifact Power

- Preserve richer UWS-compatible workflow patterns through intent, workflow HCL, UWS export, plan,
  review, and quality checks.
- Cover switch, loop, structural results, success criteria, failure actions, retries, explicit
  timeout metadata, and workflow idempotency metadata where public sibling contracts exist.
- Keep prompt defaults constrained to explicit project/intent requests for retries, timeouts, and
  idempotency.

Acceptance: OpenUdon can validate richer workflow artifacts without moving public semantics or runtime
execution behavior into OpenUdon.

### 6. iCoT Authoring

- Guide operators from broad project ideas to `project.md` and `workflows/intent.hcl`.
- Support optional LLM kickoff/refine/disambiguate roles while keeping offline manual authoring.
- Autosave incomplete sessions, save transcripts, support reconcile/lint/replay, and write final
  artifacts atomically.
- Improve OpenAPI operation ranking, request mapping inference, readiness checks, grouped
  questions, and confidence/evidence classification.

Acceptance: a trusted user can author a reviewable OpenUdon workflow package without reading
implementation code.

### 7. Safety And Trusted Execution

- Define the minimum review package for trusted execution.
- Emit `expected/symphony-handoff.json` using the stable `apitools.review-handoff.v1` wire
  version, with OpenUdon-owned validation and lifecycle behavior.
- Generate approval JSON from the current package digest.
- Validate handoff manifest, stored/current quality, approval scope, expiry, digest, and tier/state
  compatibility before udon invocation.
- Keep synthesis, build, promote, assess, iCoT, and eval free of production side effects.

Acceptance: side-effectful execution happens only through approved local trusted-runner gates.

### 8. Local Checks And Release Process

- Keep `go test ./...`, `go vet ./...`, `make check`, and `git diff --check` as normal
  deterministic gates.
- Keep `make release-check` as deterministic pre-tag release readiness.
- Keep `make release-eval` separate as opt-in real-provider release evidence with expanded-corpus
  minimum brief count.
- Keep release notes recording model, prompt version, corpus size, pass rate, comparison baseline,
  provider drift, and known gaps.

Acceptance: routine development and deterministic release readiness stay fast and provider-free,
while release confidence can include separately recorded manual provider evidence.

### 9. Product Usability

- Keep CLI help, operator checklist, onboarding, project template, eval gallery, and quality repair
  hints aligned with current behavior.
- Keep README as the concise operator entrypoint and memory-bank as project source of truth.

Acceptance: a new trusted operator can author, synthesize, assess, evaluate, and prepare a handoff
from documented commands.

### 10. Cross-Repo Dependency Stewardship

- Track UWS semantics, udon lowering/runtime compatibility, Symphony approval handoff, provider
  drift, private checkout readiness, runtime/profile evals, and expanded release evidence.
- Close OpenUdon-owned slices with regression coverage and open sibling work only when a reusable
  upstream gap is proven.

Acceptance: OpenUdon remains thin and does not absorb sibling ownership.

### 11. Public OpenUdon Package Boundary

- Complete the OpenUdon lifecycle migration. Status: done. OpenUdon now owns iCoT/progressive loop,
  prompt transcript/replay, JSON completion fallback, artifact sets, review handoff validation,
  package digest, symbolic binding contracts, credential scanning, and review metadata locally.
- Preserve the final `../apitools` keep boundary: OpenAPI search, discovery, import, download,
  local scanning, validation, operation indexing, operation summaries, auth/security summaries,
  operation ranking, CLI search/import, and cache support. Status: done.
- Remove or move downstream all non-OpenAPI `../apitools` lifecycle APIs: generic authoring
  structs/flows, iCoT loop/session/transcript helpers, JSON completion fallback, review
  handoff/state machine, package digest, credential scans, binding contracts, leaf adapter/review
  package helpers, LLM provider helpers, and Context7/documentation authoring context. Status: done.
- The IaC sibling is parked and is not a compatibility gate for this narrowing. It must move any
  lifecycle dependencies into its own packages before it resumes tracking current apitools.
- Migrate `../udon` before deleting APIs. Udon must move runtime-plan leaf/review helpers and
  `apitools/llm` usage into udon-owned code and keep only OpenAPI search/import/index usage.
  Status: done.
- Keep OpenUdon on local `internal/authoring` lifecycle helpers and add or keep a static guard that
  OpenUdon production packages do not import non-OpenAPI `apitools` APIs. Status: done.
- Hard-narrow `../apitools` only after downstream consumers compile without those APIs: delete or
  move non-OpenAPI packages/files, rewrite README/docs around the OpenAPI-only boundary, and keep
  authoring/review/handoff material only as historical migration notes when needed. Status: done.
- Keep `apitools.review-handoff.v1` only as a wire compatibility string while downstream artifacts
  still need it, not as active `../apitools` lifecycle ownership.
- Split OpenUdon's remaining udon executor integration into a trusted executor handoff based on UWS Document, OpenAPI files, non-secret run config, and runtime credential resolution. Status: done; OpenUdon stages reviewed artifacts into a fresh executor-visible directory under the run workdir and invokes udon only as a CLI or Docker process.
- Harden the trusted executor handoff so every staged OpenAPI file is digest-covered, symlinked
  OpenAPI artifacts are rejected, Docker receives only declared credential env names, and bad
  OpenAPI operation IDs fail generation instead of producing partial request maps. Status: done.

Acceptance: OpenUdon owns lifecycle APIs locally, `../apitools` contains only OpenAPI tooling, `../udon`
compiles without non-OpenAPI apitools APIs, the IaC sibling is explicitly parked, static guards
prevent regression, and OpenUdon's public build no longer relies on broad shared apitools product workflow APIs or udon Go packages.

Verification plan:

- OpenUdon: `go test ./...`, `go vet ./...`, `make check`, and `git diff --check`.
- `../apitools`: `go test ./...`, `go vet ./...`, `git diff --check`, plus CLI smoke coverage for
  `search` and `import`.
- Downstreams: `(cd ../udon && go test ./...)`. The IaC sibling is parked and not a gate.
- Static guards: `rg` for removed non-OpenAPI symbols in OpenUdon, udon, and apitools docs.

### 12. Package Artifact And Local OpenAPI Safety Hardening

- Harden OpenUdon package artifact validation and digest inputs for all required handoff files. Status:
  done; required OpenUdon handoff inputs now share safe relative path validation, manifest inventory
  checks, package-root and regular-file validation, digest input validation, and trusted-runner
  staging guards.
- Harden `../apitools` local OpenAPI reads with symlink, type, and size checks. Status: done;
  `LocalFiles`, `BuildOperationInventory`, and `LoadOperationIndex` now share bounded local file
  reads that reject symlinked roots/paths/parents, directories, special files, and oversized
  path-backed documents.
- Replace the trusted executor shell/Python runner with Go run-config parsing and staging. Status:
  done; `internal/udonrunner` and `cmd/udon-runner` now validate config JSON, stage
  workflow/OpenAPI inputs, and exec the configured binary or Docker executor by argv.
- Split `workflowintent` into intent model/HCL, provider clients, and OpenAPI adapter modules.
  Status: done; the package name and exported API remain unchanged while the deleted monolith is
  replaced by focused `intent.go`, `provider_client.go`, `openapi.go`, and shared helpers.

Acceptance: package roots and required handoff inputs cannot be symlinks, directories, special
files, unsafe relative paths, or digest/staging bypasses; sibling-owned hardening remains tracked
without moving ownership into OpenUdon; the review-follow-up hardening group is closed.

## Closed XRD Regression Matrix

| ID | Status | Regression owner | Boundary |
| --- | --- | --- | --- |
| XRD-001 | Closed | udon / OpenUdon | Structured output fallback regressions only. |
| XRD-002 | Closed | udon / OpenUdon | UWS structural/action preservation regressions only. |
| XRD-003 | Closed | UWS / udon / OpenUdon | UWS 1.1 timeout/idempotency public contract is done; runtime enforcement remains udon work. |
| XRD-004 | Closed | OpenUdon eval first, udon if needed | Pagination, request bodies, security, writes, response extraction, and multi-service eval coverage. |
| XRD-005 | Closed | OpenUdon / optional Symphony owner | OpenUdon emits handoff evidence and trusted wrapper; Symphony owns managed routing if needed. |
| XRD-006 | Closed | OpenUdon release owner / provider owners | Provider drift reporting in eval evidence and release notes. |
| XRD-007 | Closed | OpenUdon / infra | Local readiness reporting; hosted automation deferred. |
| XRD-008 | Closed | OpenUdon eval first, UWS/udon if needed | Runtime/profile coverage as policy/eval evidence. |
| XRD-009 | Closed | OpenUdon release owner | Expanded-corpus minimum brief release gate. |

## Runtime/Profile Eval Coverage Detail

Runtime/profile semantics and generic execution remain upstream in `../udon` or `../uws`. OpenUdon's
XRD-008 coverage is policy language, reference intent shape, review evidence, and fixture
regression only.

| Category | Fixtures | Boundary proven |
| --- | --- | --- |
| Approved function runtime | `runtime-only-render`, `support-priority-routing`, `profile-boundary-manifest` | Trusted local adapters and renderers can be generated when declared in project policy and Function Contracts. |
| Approved command runtime | `cmd-allowed-deploy` | `cmd` is allowed only when project policy explicitly permits a sandbox command. |
| Denied command runtime | `cmd-disallowed-deploy` | A generated `cmd` step remains a policy failure when the project denies command execution. |
| Denied SSH/runtime profiles | `cmd-disallowed-deploy`, `profile-boundary-manifest` | OpenUdon policy keeps `ssh` and future profile runtimes out of generated intent unless an upstream public/runtime contract exists and project policy approves it. |
| Future profile boundary | `profile-boundary-manifest` | SQL-style profile work is represented as a trusted `fnct` manifest request instead of inventing `sql`, `ssh`, or `x-udon-*` runtime semantics in OpenUdon. |

OpenUdon reference intents must not invent unsupported runtime types such as `sql`, `smtp`, `llm`, or
profile-specific `x-udon-*` payloads. Any future fixture that needs real profile semantics should
open upstream work in `../udon` or `../uws` before OpenUdon prompt defaults emit those fields.
HTTP is not a runtime-profile type in UWS extension payloads: HTTP/OpenAPI operations must bind
through core UWS OpenAPI fields, and `type: http` in public `x-uws-runtime` payloads is rejected.
The public runtime supplement is intentionally small: runtime selector fields only, with no
provider/security configuration, HTTP metadata, or request/response schema projection. Udon's legacy
private `x-udon-runtime` remains a compatibility surface until a separate udon DTO and export
migration removes its raw HTTP projection.

## Documentation Maintenance

- Update [status.md](status.md) whenever completion state changes.
- Update this file when sequencing, contracts, milestones, or acceptance criteria change.
- After each major milestone or boundary change, decide whether [evolution/](../evolution/) needs a
  new prompt/result version.
