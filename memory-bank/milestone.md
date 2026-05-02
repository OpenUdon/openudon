# Milestone

## Memory Bank Index

- This file owns milestones, work sequencing, and acceptance criteria.
- Use [product.md](product.md) for product scope and non-goals.
- Use [architecture.md](architecture.md) for system boundaries and planned structure.
- Use [tech-stack.md](tech-stack.md) for dependency and tooling defaults.
- Use [status.md](status.md) for current completion state.

## Delivery Strategy

Keep Ramen as a thin private integration layer. Build in verifiable slices: authoring and examples,
deterministic synthesis, quality gates, eval and release evidence, trusted handoff, readiness, and
cross-repo compatibility. Push public workflow semantics to `../uws`, reusable execution to
`../udon`, domain-neutral review/authoring helpers to `../apitools`, and Symphony orchestration to
`../symphony`.

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
  can be expressed as Ramen intent without adding n8n-specific runtime behavior.
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

Acceptance: release evidence can distinguish deterministic Ramen regressions from provider drift.

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

Acceptance: Ramen can validate richer workflow artifacts without moving public semantics or runtime
execution behavior into Ramen.

### 6. iCoT Authoring

- Guide operators from broad project ideas to `project.md` and `workflows/intent.hcl`.
- Support optional LLM kickoff/refine/disambiguate roles while keeping offline manual authoring.
- Autosave incomplete sessions, save transcripts, support reconcile/lint/replay, and write final
  artifacts atomically.
- Improve OpenAPI operation ranking, request mapping inference, readiness checks, grouped
  questions, and confidence/evidence classification.

Acceptance: a trusted user can author a reviewable Ramen workflow package without reading
implementation code.

### 7. Safety And Trusted Execution

- Define the minimum review package for trusted execution.
- Emit `expected/symphony-handoff.json` using the public apitools handoff schema.
- Generate approval JSON from the current package digest.
- Validate handoff manifest, stored/current quality, approval scope, expiry, digest, and tier/state
  compatibility before udon invocation.
- Keep synthesis, build, promote, assess, iCoT, and eval free of production side effects.

Acceptance: side-effectful execution happens only through approved local trusted-runner gates.

### 8. Local Checks And Release Process

- Keep `go test ./...`, `go vet ./...`, `make check`, and `git diff --check` as normal
  deterministic gates.
- Keep `make release-check` as deterministic release readiness.
- Keep `make release-eval` as opt-in real-provider release evidence with expanded-corpus minimum
  brief count.
- Keep release notes recording model, prompt version, corpus size, pass rate, comparison baseline,
  provider drift, and known gaps.

Acceptance: routine development is fast and deterministic, while release confidence can include
manual provider evidence.

### 9. Product Usability

- Keep CLI help, operator checklist, onboarding, project template, eval gallery, and quality repair
  hints aligned with current behavior.
- Keep README as the concise operator entrypoint and memory-bank as project source of truth.

Acceptance: a new trusted operator can author, synthesize, assess, evaluate, and prepare a handoff
from documented commands.

### 10. Cross-Repo Dependency Stewardship

- Track UWS semantics, udon lowering/runtime compatibility, Symphony approval handoff, provider
  drift, private checkout readiness, runtime/profile evals, and expanded release evidence.
- Close Ramen-owned slices with regression coverage and open sibling work only when a reusable
  upstream gap is proven.

Acceptance: Ramen remains thin and does not absorb sibling ownership.

## Closed XRD Regression Matrix

| ID | Status | Regression owner | Boundary |
| --- | --- | --- | --- |
| XRD-001 | Closed | udon / Ramen | Structured output fallback regressions only. |
| XRD-002 | Closed | udon / Ramen | UWS structural/action preservation regressions only. |
| XRD-003 | Closed | UWS / udon / Ramen | UWS 1.1 timeout/idempotency public contract is done; runtime enforcement remains udon work. |
| XRD-004 | Closed | Ramen eval first, udon if needed | Pagination, request bodies, security, writes, response extraction, and multi-service eval coverage. |
| XRD-005 | Closed | Ramen / optional Symphony owner | Ramen emits handoff evidence and trusted wrapper; Symphony owns managed routing if needed. |
| XRD-006 | Closed | Ramen release owner / provider owners | Provider drift reporting in eval evidence and release notes. |
| XRD-007 | Closed | Ramen / infra | Local readiness reporting; hosted automation deferred. |
| XRD-008 | Closed | Ramen eval first, UWS/udon if needed | Runtime/profile coverage as policy/eval evidence. |
| XRD-009 | Closed | Ramen release owner | Expanded-corpus minimum brief release gate. |

## Runtime/Profile Eval Coverage Detail

Runtime/profile semantics and generic execution remain upstream in `../udon` or `../uws`. Ramen's
XRD-008 coverage is policy language, reference intent shape, review evidence, and fixture
regression only.

| Category | Fixtures | Boundary proven |
| --- | --- | --- |
| Approved function runtime | `runtime-only-render`, `support-priority-routing`, `profile-boundary-manifest` | Trusted local adapters and renderers can be generated when declared in project policy and Function Contracts. |
| Approved command runtime | `cmd-allowed-deploy` | `cmd` is allowed only when project policy explicitly permits a sandbox command. |
| Denied command runtime | `cmd-disallowed-deploy` | A generated `cmd` step remains a policy failure when the project denies command execution. |
| Denied SSH/runtime profiles | `cmd-disallowed-deploy`, `profile-boundary-manifest` | Ramen policy keeps `ssh` and future profile runtimes out of generated intent unless an upstream public/runtime contract exists and project policy approves it. |
| Future profile boundary | `profile-boundary-manifest` | SQL-style profile work is represented as a trusted `fnct` manifest request instead of inventing `sql`, `ssh`, or `x-udon-*` runtime semantics in Ramen. |

Ramen reference intents must not invent unsupported runtime types such as `sql`, `smtp`, `llm`, or
profile-specific `x-udon-*` payloads. Any future fixture that needs real profile semantics should
open upstream work in `../udon` or `../uws` before Ramen prompt defaults emit those fields.

## Documentation Maintenance

- Update [status.md](status.md) whenever completion state changes.
- Update this file when sequencing, contracts, milestones, or acceptance criteria change.
- After each major milestone or boundary change, decide whether [evolution/](../evolution/) needs a
  new prompt/result version.
