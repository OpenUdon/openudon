# Retired milestone M87 - Non-interactive step authoring contract for Kinet

**Milestone.** M87
**Outcome.** completed
**Retired.** 2026-09-27
**Source status.** tabilet/memory-bank/status-M87.md
**Source specification.** tabilet/memory-bank/milestone.md#m87--non-interactive-step-authoring-contract-for-kinet
**Evidence.** 8178e7ead454b766cdef4ae09e48d7ca457a9ef8
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 9
**Verification.** `go test -count=1 ./...` and `env GOWORK=off go test -count=1 ./...`; workspace and standalone `go vet`; affected standalone race tests; `make check`; tagged UWS C07.2 mapping test; Apitools boundary check; `check-doc-memory`; and `git diff --check` all passed. The pinned Apitools M77 pseudo-version resolved to `v0.0.0-20260927073943-30c6f3bd5700` at `30c6f3bd57001e521804a520e5aa21cb7dd95047`. `check-doc-memory` emitted only its expected reminder that evolution v44 already records this direction.
**Consolidated into.** Current behavior and boundaries are documented in `tabilet/memory-bank/product.md`, `architecture.md`, and `tech-stack.md`. No new reusable lesson or evolution version was needed; M87 implementation advances the approved v44 direction.

## Milestone specification

~~~~~~~~markdown
### M87 — Non-interactive step authoring contract for Kinet

**Goal.** Deliver `openudon step candidates`, `openudon step bind`,
`openudon step check`, and `openudon flow-review` as non-interactive commands
with stable, versioned JSON results and published conformance fixtures.
OpenUdon owns the command and intent-authoring contracts; Kinet W03 owns its
consumer adapter, workflow planning loop, confirmations, ledger, and repair
orchestration. Public workflow semantics remain owned by UWS.

**Authority and lineage.** The owner approved this four-file planning change
on 2026-09-26 after inspection of Kinet `docs/ideas.md` section 10 and
`docs/icot.md` sections 4–7. Kinet's pending W03 first row is consumer
requirements/feedback, not a prerequisite for OpenUdon to draft and own the
contract. This is new cross-cutting public-contract work, not a reopening of
retired iCoT milestones. The planning decision is recorded in
[evolution v44](../evolution/result-v44.md).

**Scope and contract requirements.**

- Define bounded versioned requests, results, schemas, stable statuses and
  diagnostic codes, exit-code behavior, and compatibility rules. Machine
  output uses clean JSON stdout for success and failure; terminal prompts,
  progress text, and provider error bodies must not contaminate it.
- A step contract declares purpose, inputs, outputs, account/destination
  constraints, and confirmed effect class. It is OpenUdon authoring metadata,
  not a new UWS operation or pending-step extension. Its purpose, inputs,
  outputs, and effect fields use exactly the declaration shape of UWS C07.2's
  pending step, which UWS owns; OpenUdon-only fields (contract identity,
  digests, and account/destination constraints) sit alongside them, so a step
  contract maps losslessly onto a UWS pending step when package-level
  placeholders arrive (S2b). Operation/source identity,
  content digests, contract identity, and expected intent revision bind the
  candidate, check, and write to the same reviewed inputs.
- `step candidates` consumes Apitools discovery, ranking, consumer-readable
  summaries, auth needs, and effect metadata. Preserve auth alternatives as
  OR-of-AND sets with symbolic bindings only, exact source identity, visible
  ambiguity/truncation, and unsupported capability diagnostics. Missing effect
  evidence is `unknown`, treated conservatively like `write`; an HTTP method
  alone must not establish a `read` claim. Ranking is advisory and confers no
  operation, account, destination, or execution approval.
- `step check` is read-only and checks one selected step against its contract,
  exact source operation, request/output mappings, authentication requirements,
  relevant dependencies, and effect constraints. Reject self-output references
  and dependency cycles involving the selected step. Report unresolved semantic
  questions separately from deterministic checks. Structural compatibility
  does not prove that arbitrary natural-language intent has been fulfilled.
- `step bind` explicitly creates or replaces one identified step in
  `workflows/intent.hcl`, reusing the checks before an atomic write. Preserve
  unrelated steps, blocks, and content. Initial intent creation requires an
  explicit workflow scaffold; never silently invent workflow-wide policy.
  Stale source/contract/intent inputs, unsafe paths, or invalid bindings fail
  before replacement. Reject self-output references and dependency cycles
  before replacement. Rejected writes leave existing bytes unchanged; an
  indeterminate filesystem outcome must be reported distinctly for recovery.
- `flow-review` exposes today's advisory flow review through shared logic,
  including relevant local checks and explicitly configured model review.
  It is read-only, performs no automatic repair, and reports unavailable,
  skipped, or failed model review distinctly from a completed review. Findings
  remain advisory and cannot replace build, assessment, or digest-bound
  approval. Published fixtures cover model-free completion, model findings,
  provider unavailability, model failure, and cancellation.
- Publish consumer-usable schemas and conformance fixtures with valid,
  malformed, unsupported-version, stale-revision, and digest-mismatched cases
  plus auth/effect/mapping and write-preservation cases. Kinet can independently
  validate the public wire without importing OpenUdon internal packages.

**Order and dependencies.** M87.1 owns contract/schema/initial-fixture design
first. M87.2 and M87.3 supply check and bind against the agreed contract;
M87.4 supplies candidates integration; M87.5 exposes flow review; M87.6
qualifies and documents the complete surface. Keep one execution owner and at
most one general in-progress row. If upstream metadata is unavailable, finish
independent contract, fixture, check/bind, and review work rather than making
all of OpenUdon wait; record the unresolved external gate explicitly.

| Dependency or consumer | Owner and required evidence | Effect on M87 |
|---|---|---|
| S2a operation metadata | APItools owns source-backed consumer summaries, effect classification with evidence, and ranking by purpose/inputs/outputs. M77 is complete, committed and pushed at `30c6f3bd57001e521804a520e5aa21cb7dd95047`, and published as `v0.0.0-20260927073943-30c6f3bd5700`. The previously reported OpenUdon shared-workspace regression for legitimate `projectKey` and `idempotencyKey` fields is fixed; workspace and standalone consumers pass against this exact revision. M87.1 reconciles the DTO field shape without importing APItools internals. | M87.4 production integration is complete against the published API; no remaining APItools publication gate blocks M87.6. Do not duplicate generic ranking/classification here or call a stub delivered. |
| Kinet W03 | Consumes OpenUdon's published contract and fixtures through an external CLI adapter; retains independent validation and its own `make openudon-check`. Kinet M05/A03 remain its internal sequencing dependencies. | No reverse dependency on Kinet starting or completing W03. OpenUdon supplies early fixtures and release-ready contracts; Kinet later verifies adoption. |
| Existing OpenUdon authoring/package behavior | Reuse current intent validation, source/auth checks, atomic writing, and advisory review. Preserve iCoT callers, existing packages, build/assess, approval, and trusted-runner behavior. | Regression responsibility, not retirement authority or an instruction to reopen completed history. |
| UWS S1, Browsertools S2b, and Udon S2c | Public pending-step/mock semantics, acquisition/snapshot checks, and live hybrid execution remain with their owners. UWS C07.2 also owns the declaration shape of the shared purpose/inputs/outputs/effect fields. | Not M87 implementation prerequisites or deliverables. M87.1 agrees the shared field shape with UWS C07.2 before freezing (a design synchronization point, not a wait for UWS publication) and records the UWS revision used; no new runtime semantics or live calls are implied. |

**Acceptance and verification.** All four commands implement their published
versioned success/failure contracts and pass the conformance corpus. A fixture
maps each step contract's shared fields losslessly onto UWS's pending-step
declaration at a recorded UWS revision. A
credential-free local workflow can select a candidate, bind/check one step,
review the assembled flow, and pass existing build/assessment without an
interactive session. Tests prove malformed input and stale/digest-mismatched
selection rejection, OR-of-AND authentication, conservative unknown effects,
missing mappings/dependencies, cycle rejection, unchanged unrelated steps, concurrent-edit
conflicts, and rejected-write byte preservation. Fake model responses cover
flow-review success, findings, failure, and cancellation without claiming live
model evidence. Preserve supported existing source families or report explicit
unsupported capabilities; do not silently drop a source kind.

Run focused CLI, schema/conformance, intent, source/auth, writer and review
tests, affected race tests, `GOWORK=off go test ./...`,
`GOWORK=off go vet ./...`, `make check`, the Apitools boundary check,
document-memory validation, and `git diff --check`. Follow the existing
affected-smoke policy only if shared UI/runtime behavior changes; full browser
qualification and live-model/provider runs are not default S2a checks.
Document supported versions and fixture consumption for Kinet; its later real
consumer check does not block OpenUdon's own contract drafting or qualification.
Remote publication follows separate authority. Complete the persisted
maximum-ten-iteration milestone review before declaring M87 accepted.

**Compatibility and exclusions.** Add the commands alongside current iCoT
interfaces. Extract only shared logic needed for these commands and keep old
callers working. Kinet-owned placeholders stay in its ledger at this stage;
M87 neither emits executable placeholders nor weakens approval gates.
`simulate`, supervised `browser acquire`, package-level pending steps, runtime
effect enforcement, and iCoT/UI retirement are outside scope. No sibling files,
UWS semantics, credentials, live provider/browser behavior, deployment, or
runtime adoption are changed by this planning approval.

~~~~~~~~

## Status record

~~~~~~~~markdown
# Status M87 — Non-interactive step authoring contract for Kinet

**State:** Complete; all implementation, qualification, and review acceptance
criteria passed. Planning was approved 2026-09-26; the combined M77 -> M87 run was confirmed
2026-09-27 with `COMMIT_POLICY: milestone`. Push authority is limited to
APItools and OpenUdon `origin/main`; no tag is planned. UWS, Kinet, Ramen, and
other repositories remain outside commit/push scope. The Kinet consumer remains
separately owned; this release supplies its published contract and commands.

**Specification:** [M87 in milestone.md](milestone.md#m87--non-interactive-step-authoring-contract-for-kinet).
**Direction:** [Evolution v44](../evolution/result-v44.md).
**Priority:** M87.1–M87.6 are complete against the published APItools M77
revision. The review gate passed at iteration 9; archival retirement and the
authorized OpenUdon push remain.

## Evidence and decisions

- The owner requests sibling S2a from Kinet `docs/ideas.md` section 10 and
  `docs/icot.md` sections 4–7: `step candidates`, `step bind`, `step check`,
  and `flow-review`, stable versioned JSON, and published conformance fixtures.
- OpenUdon owns and drafts the final command contract independently of Kinet's
  M05 -> A03 -> W03 sequence. Kinet W03 consumes it and retains independent
  parsing, approval checks, user-ledger decisions, and repair orchestration.
- Inspection baseline: clean OpenUdon
  `55b24d29279c8efe67ae931f9d73f094f929efab`. Relevant existing code is
  `internal/workflowintent/intent.go`, `internal/icot/artifactwriter/writer.go`,
  `internal/icot/elicitor/draft_review.go`,
  `internal/icot/elicitor/extractor.go`, and `cmd/openudon/main.go`.
  None of the four new command routes exists at that baseline.
- Apitools `inventory_types.go` and `authoring_api.go` provide operation and
  request/response summaries, security alternatives, and text ranking. Explicit
  effect classification and structured step-contract ranking are not yet in
  the inspected API. They remain an Apitools-owned external dependency, not
  permission to duplicate that policy inside OpenUdon.
- Kinet evidence was read at HEAD
  `9719a364904743c0d2cb8b3f47cb4e50ed6322e8` with uncommitted planning changes,
  including `status-W03.md`; the proposed design is not claimed to be delivered
  or committed. No sibling file is changed by this plan.
- Existing iCoT and package behavior remain supported. S2b simulation/browser
  acquisition and S3 retirement stay unnumbered in Candidate Directions.
- The step contract's purpose, inputs, outputs, and effect fields share one
  shape with UWS C07.2's pending-step declaration, which UWS owns. M87.1 and
  C07.2 agree that shape before either freezes, so stage 5 needs no translation
  layer. Aligned 2026-09-27 at the owner's request (Kinet `docs/kinet-order.md` X2).
- Goal execution baseline: OpenUdon `8178e7ead454b766cdef4ae09e48d7ca457a9ef8`,
  clean worktree, branch one local commit ahead of `origin/main`. The resolved
  status order is M87. The task order follows Kinet's cross-package note:
  M87.1 -> M87.2 -> M87.3 -> M87.5 -> M87.4 -> M87.6. Apitools M77.1 is
  currently in progress in its own checkout; UWS C07.1 is in progress while
  C07.2 remains pending. These observations establish coordination points,
  not permission to edit sibling repositories.
- Combined-run handoff (2026-09-27): the user quit the prior OpenUdon session
  while preserving its worktree. The existing M87 edits are intentionally
  retained and are the starting point; no clean, reset, or checkout operation
  is authorized. APItools M77 was committed and pushed as
  `30c6f3bd57001e521804a520e5aa21cb7dd95047`; Go resolved the exact public
  module revision `v0.0.0-20260927073943-30c6f3bd5700`. M87.4 is now active and
  must verify this pinned revision in standalone mode as well as the local
  workspace. The user authorized the OpenUdon milestone closure commit and
  `origin/main` push, but no UWS/Kinet/Ramen release or mutation.
- UWS read-only dependency refresh (2026-09-27): local UWS HEAD is
  `7f843af78e508fee140b3b43f28a6b73b37a66a8`, clean and nine commits ahead of
  its remote. It contains the C07.2 `uws1.PendingStep` declaration; remote
  `origin/main` remains `1d5535ec75d5693a66bcced5bd98f5c4c824fb2a`. No UWS push
  is authorized. M87.6 can validate the lossless field mapping against this
  recorded workspace revision while standalone checks continue to validate
  against OpenUdon's published UWS pin.
- M87.1 draft progress (2026-09-27): added
  `docs/step-authoring-contract-v1.md`, the Draft 2020-12
  `openudon.step-authoring.v1` schema, and synthetic request/result and
  rejection fixtures with an independent schema-conformance test. The
  owner selected one recursive UWS ParamSchema object per shared field set
  (inputs and outputs), with object roots, named properties, and required-name
  arrays. The draft schema and candidate fixture now use this shape, including
  nested properties, array items, and UWS-style extensions. The candidate
  result preserves the currently inspected APItools metadata dimensions:
  exact source identity, consumer summary/evidence/gaps, per-dimension match
  evidence and scores, authentication alternatives, effect evidence, and
  source capability reports. The draft maps APItools' sha256 field to the
  OpenUdon sha256: digest form and omits paths and URLs from results. Focused
  verification: go test ./internal/stepauthoringcontract passes.
- Cross-contract recheck (2026-09-27): APItools is at
  `e3b4b6ec343a18c48fa93a971a69930b993203db`, with M77.1 and M77.2 marked
  complete and M77.3 active; its working tree contains uncommitted M77 code
  and documentation, which this task did not edit. Its draft candidate DTO
  includes source identity, summary evidence/gaps, typed input/output
  summaries, effect evidence/reasons, per-dimension match evidence/scores,
  source capability reports, and the existing operation security sets; the
  OpenUdon candidate fixture/schema retains those consumer-relevant fields.
  This is a shape reconciliation, not a released dependency. Kinet is at
  `d3e48aca53214f35064e022ac10501230b640025`; its W03 consumer row and X3
  requirements are recorded, and its untracked `internal/appstore/` is
  preserved. UWS is at
  `9ab397864d3e3f501c4fbd699584ebf63dca1ea4`; C07.1 is complete and C07.2
  remains pending. The owner selected one ParamSchema object per field set.
  With `GOWORK=off`, the focused fixture test uses the OpenUdon-pinned UWS
  module `v0.0.0-20260925154821-80ee9bfb24a6` and confirms the recursive
  ParamSchema wire data survives a Go JSON round trip. C07.2 still needs to
  implement the selected pending-step wrapper before OpenUdon can verify the
  wrapper-level lossless mapping fixture in M87.6. That fixture is not an
  M87.1 prerequisite. At the latest read-only refresh, UWS remained at HEAD
  `9ab397864d3e3f501c4fbd699584ebf63dca1ea4` with C07.2 in progress in its
  worktree; OpenUdon remains pinned to the published module above. Kinet is at
  HEAD `3d2f1bdb0fff0d95a373f09fc4eb409afd2c9c79` with separate A03 audit
  changes in progress; W03 row 1 still requires adopting OpenUdon's published
  contract and fixtures, and X3 says its requirements feed M87.1 before freeze.
  APItools remains at HEAD `e3b4b6ec343a18c48fa93a971a69930b993203db` with
  uncommitted M77 implementation; its candidate DTO shape was inspected but
  not treated as a released dependency. No sibling worktree was changed.
- Final contract review found that the first `step.check` draft did not carry
  the selected candidate's `operation_ref`; its intent digest alone could not
  detect source drift. M87.1 now adds that source identity/digest to the check
  request and echoes it in the result, with a negative fixture for the missing
  reference. This closes the accepted candidate/check/bind continuity
  requirement. `go test ./internal/stepauthoringcontract`,
  `GOWORK=off go test ./internal/stepauthoringcontract`,
  `go run ./cmd/openudon check-doc-memory`, and `git diff --check` passed.
- M87.2 implementation (2026-09-27): added the strict versioned `step.check`
  request/result path, safe bounded reads of the intent and exact source,
  exact source-operation matching through the pinned APItools inventory,
  mapping/output/dependency checks, OR-of-AND authentication evidence, and
  conservative `indeterminate` effect assessment because the pinned metadata
  does not classify effects. Added a runnable OpenAPI/intent example, expected
  result, rejection fixtures, and tests for stale digests, unsafe/symlink source
  paths, missing mappings, output references, and auth alternatives. The
  runnable fixture returns one JSON result and no source paths. Verification
  passed: focused race tests for `internal/stepauthoring`,
  `internal/stepauthoringcontract`, and `cmd/openudon`; focused `go vet`; the
  runnable `step check` CLI fixture; `check-doc-memory` (with the expected
  reminder to confirm no evolution change is needed); and `git diff --check`.
- M87.2 follow-up during M87.3: corrected selected-source resolution so an
  explicit per-step source overrides the workflow-level default, enabling
  multi-service workflows without weakening safe relative-path checks. The
  focused regression test confirms the exact step source wins.
- M87.3 implementation (2026-09-27): added `openudon step bind` with strict
  v1 JSON input/result handling, exact source-ID/digest lookup over bounded
  regular files in the source-kind directory, and exact source operation/key/
  selector verification via pinned Apitools inventory. It checks the selected
  operation's required request mappings, response-backed output mappings,
  declared workflow inputs, dependencies, and one explicitly selected
  OR-of-AND authentication alternative. Symbolic credential slots map to
  `credentials.<name>` in the step's native `with` map; `none`, `clear`, and
  credential-looking values are rejected before mutation. Missing intent
  requires explicit workflow metadata/input/output scaffold. Existing intent
  receives a token-preserving in-place step update (or one top-level step
  addition), with stale browser/function-only fields removed and downstream
  output references rechecked. Only `workflows/intent.hcl` is written through
  the shared atomic artifact writer using digest preconditions and create-only
  installation. Added success, stale revision/source, invalid credential,
  missing mapping, symlink, repeat/unchanged, unrelated-step preservation, and
  concurrent-edit tests plus runnable request/result/HCL fixtures. Focused
  `GOWORK=off` tests and race tests for step-authoring, contract fixtures, and
  CLI passed; focused vet, document-memory, and diff checks passed. The doc
  checker emitted only its expected reminder to confirm that this implementation
  does not require a new evolution version.

- M87.5 implementation (2026-09-27): added the read-only `openudon flow-review`
  command, reusing iCoT's deterministic local checks and chat-extractor model
  review while leaving interactive callers unchanged. Requests bind to the
  exact intent digest, use bounded no-symlink reads, and require explicit
  provider/model names when model review is enabled. Runtime credentials stay
  in the process environment; credential-shaped draft context is not sent, and
  local paths/provenance are removed from the provider payload. Findings are
  advisory, bounded, and filtered; source paths, provider errors, evidence,
  repair suggestions, and unsafe findings are not emitted. Results distinguish
  model `skipped`, `unavailable`, `completed`, and `failed`; cancellation is a
  blocked command outcome. Added valid completion/finding/unavailable/failure/
  cancellation fixtures, an invalid provider fixture, schema checks, CLI JSON
  tests, fake-reviewer success/failure/cancellation tests, unsafe input/output
  tests, size-bound and stale-digest tests. Verification passed:
  `GOWORK=off go test -race -count=1 ./internal/stepauthoring ./internal/icot/elicitor ./cmd/openudon ./internal/stepauthoringcontract`;
  `GOWORK=off go test ./...`; `GOWORK=off go vet ./...`; runnable fixture CLI;
  document-memory and diff checks. The model-free command fixture exactly
  matches its published result.
- M87.4 external gate recheck (2026-09-27): Apitools remains at
  `e3b4b6ec343a18c48fa93a971a69930b993203db` with M77.1–M77.5 implementation
  files uncommitted and M77.6 explicitly blocked pending this OpenUdon session
  handoff. No compatible published M77 revision is available to pin. This
  checkout was inspected read-only; M87.4 and dependent M87.6 remain gated, and
  no sibling files or remotes were changed.
- M77.6 OpenUdon consumer recheck (2026-09-27): `GOWORK=off go test -count=1
  ./...` and both `go vet ./...` / `GOWORK=off go vet ./...` pass against
  OpenUdon's published Apitools pin. The shared-workspace `go test ./...`
  fails three existing iCoT seed/build matrix cases against local Apitools
  `e3b4b6ec343a18c48fa93a971a69930b993203db`: `projectKey` is absent in two
  Jira cases, and `idempotencyKey` is absent in the webhook case. A focused
  uncached repro confirms the failure. The local Apitools `looksLikeCredentialName`
  change tokenizes camelCase and treats any `key` token as credential-shaped,
  so these ordinary request fields are filtered from inventory; this is a
  downstream compatibility regression, not an OpenUdon fixture failure.
  M77.1–M77.5 are reported complete by the sibling handoff, but remain
  uncommitted/unpublished. Do not edit that sibling here. M77.6 remains blocked
  until the owning Apitools session fixes and verifies the compatibility issue;
  M87.4 remains blocked until a compatible published API is available. The
  documented APItools candidate DTO otherwise matches M87.1's shared purpose,
  object-root ParamSchema inputs/outputs, and effect field at the declared
  boundary; unsupported schema constructs must continue to surface as gaps.
- M77.6 follow-up (2026-09-27; supersedes the shared-workspace failure above):
  APItools corrected the false-positive credential filtering for camel-case
  `projectKey` and `idempotencyKey`, retaining conservative filtering for
  standalone/delimiter-separated `key` names and explicit credential contexts.
  APItools reports its uncached suite and the OpenUdon shared-workspace suite
  passing against the local APItools tree. This resolves the local consumer
  regression; M87.4 and dependent M87.6 remain gated because no compatible
  published Apitools API is available. No sibling code, commit, or publication
  is included in this update.

## Task ledger

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed historical with an accepted successor, `[X]` cancelled.
Each row is one implementation commit unit under the governing policy. Keep
one execution owner and at most one general in-progress row. The confirmed
combined goal authorizes the M77 and M87 milestone commits and pushes to
APItools and OpenUdon `origin/main` only; it authorizes no tag or Kinet, UWS,
Ramen, or other repository mutation.

| Item | State | Notes |
|---|---|---|
| M87.1 Define the public step-command contract | `[+]` | Settled v1 request/result schemas, bounded transport and exit/status rules, diagnostics, compatibility, command contracts, synthetic consumer fixtures, and positive/negative schema tests. Reconciled APItools M77 candidate metadata dimensions and Kinet W03 X3 requirements from the inspected snapshots recorded above. The owner selected one recursive UWS `ParamSchema` object each for `inputs` and `outputs`, rooted at `type: object`, with named `properties` and `required` arrays. OpenUdon's pinned UWS module `v0.0.0-20260925154821-80ee9bfb24a6` supports that recursive wire shape and round-trip test. X2 design sync is complete; actual pending-wrapper mapping remains M87.6 and does not block this row. `step.check` carries and echoes `operation_ref` so source identity/digest continuity is bound across candidate, check, and bind. |
| M87.2 Implement step check | `[+]` | Depends on M87.1. Implemented read-only validation of one exact selected step against its contract, source operation/digest, required mappings, outputs, authentication alternative, dependencies, and effect evidence. Final exact source-operation matching reuses the pinned APItools operation-candidate API and native selector tuple; absent explicit effect metadata remains indeterminate rather than inferred from an HTTP verb. Added runnable source/intent/request/result fixtures and deterministic rejection tests. Focused race tests, vet, CLI fixture, document-memory check, and diff check passed. |
| M87.3 Implement step bind | `[+]` | Depends on M87.1–M87.2. Implemented strict versioned binding of one exact source operation after source/intent digest checks, required request/output/auth/dependency validation, and explicit scaffold for absent intent. Writes only the selected HCL step via the shared rollback-capable writer; stale/malformed/unsafe requests remain byte-identical, repeats report unchanged, and uncertain transaction errors report indeterminate. Final selector resolution reuses the exact APItools candidate tuple. Runnable request/result/HCL fixtures and tests cover source/intent drift, secrets/sentinels, symlinks, mappings, dependencies/output references, unrelated content, repeat requests, and concurrent edits. Focused tests/race, vet, document-memory, and diff checks passed. |
| M87.4 Implement step candidates with Apitools metadata | `[+]` | Complete against Apitools `v0.0.0-20260927073943-30c6f3bd5700` at commit `30c6f3bd57001e521804a520e5aa21cb7dd95047`; `GOWORK=off go list -m` resolved the pushed revision from GitHub. Implemented bounded exact-source local discovery across all eight source families plus the legacy `discovery/` alias, and the production adapter, preserving consumer summaries, match evidence, auth alternatives, capabilities, effects, limits, and diagnostics while omitting filesystem paths/URLs. The camel-case field regression remains covered. Workspace, standalone, and candidate/check/bind source-reference tests pass against the pin. |
| M87.5 Expose advisory flow-review | `[+]` | Depends on M87.1. Added read-only `openudon flow-review` over the existing local checks and model-assisted iCoT review while preserving current iCoT callers. Model review requires explicit provider/model names; it uses process-environment credentials only, rejects credential-shaped context, strips local source paths/provenance, and emits bounded sanitized advisory findings. Skipped, unavailable, completed, failed, and cancellation outcomes are distinct; provider errors and unsafe findings are not echoed. No file repair, package approval, or hidden response fields. Published fixtures cover model-free completion, a model finding, unavailable provider, model failure, and cancellation. Focused race tests, vet, schema, CLI, and fake-reviewer tests passed. |
| M87.6 Qualify contracts and consumer handoff | `[+]` | Depends on completed M87.1–M87.5 and published APItools M77. Published schemas/fixtures, independent JSON Schema consumer validation, CLI composition, eight-family candidate/check/bind resolution, malformed/stale/digest/auth/effect/write-preservation cases, and the end-to-end local build/assessment workflow are implemented. The UWS C07.2 shared-field fixture is validated at recorded local revision `7f843af78e508fee140b3b43f28a6b73b37a66a8`; do not push UWS. Workspace and standalone tests/vet, affected race tests, tagged UWS test, `make check`, boundary/doc-memory checks, and diff validation pass. The full-milestone review passed at iteration 9. Kinet W03 remains separately owned and its real-tool check can resume after publication; no Kinet/UWS/Ramen changes are included. |

## Dependency and execution policy

M87.1–M87.6 are complete under the approved goal run. M87.4 uses the compatible
published APItools M77 revision recorded above; its workspace and standalone
consumer checks pass. The whole-milestone review passed at iteration 9. Final
archive retirement and the authorized OpenUdon publication remain.
Contract-first work did not wait for Kinet W03's first row. M87.2–M87.3 used
contract-bound fixtures until production metadata became available; their
fixtures did not substitute for M87.4's published integration. M87.5 completed
independently once M87.1 settled the public shape, following the verified
cross-package order in Kinet's `docs/kinet-order.md`. Kinet W03 remains a later
consumer check and does not block OpenUdon acceptance.

APItools owns consumer summaries, semantic effect metadata, and ranking by step
purpose/inputs/outputs; its published M77 implementation is consumed directly
by M87.4. This OpenUdon plan allocates no sibling milestone ID. The combined
run authorizes APItools and OpenUdon work and pushes only; it does not authorize
UWS/Kinet/Ramen changes or publication.

OpenUdon qualifies its commands using independent CLI consumers and fixtures;
Kinet W03 later follows the published version and runs its real-tool check.
Kinet's M05/A03 sequencing and user-ledger placeholders do not create an
OpenUdon implementation prerequisite. Package-level pending steps, new UWS
effects/mock runtime, live-read execution, browser acquisition, and iCoT
retirement remain outside M87.

## Acceptance and planned verification

- All four commands emit the agreed versioned result for successful and failed
  requests, with documented exit codes and no incidental stdout text.
- The public corpus includes valid, malformed, unsupported-version,
  stale-revision and digest-mismatched cases plus auth alternatives, unknown
  and incompatible effects, mapping gaps, and writer-preservation failures.
  Consumers can validate fixtures without importing internal Go packages.
- A model-free local CLI scenario selects an operation, binds/checks a step,
  reviews a multi-step flow, and passes existing build/assess. Structural
  success is labeled accurately and does not claim live execution or proof of
  all natural-language semantics.
- Candidates preserve exact source identity and auth requirements, expose
  evidence gaps, and consume the published Apitools capability. An unknown
  effect is never silently treated as read; ranking grants no authority.
- Failed binds preserve original bytes. Tests cover unrelated step/content
  preservation, duplicate/repeated operations, stale source/intent data,
  concurrent mutation, unsafe targets, and distinct indeterminate outcomes.
- Flow-review fixtures cover advisory findings, model failure/unavailability,
  and cancellation. The command writes no intent, selects no credentials, and
  grants no approval. Existing iCoT review behavior remains covered.
- Run focused command, conformance/schema, intent, source/auth, writer and
  review tests; affected race tests; `GOWORK=off go test ./...`;
  `GOWORK=off go vet ./...`; `make check`;
  `go run ./cmd/openudon check-apitools-boundary`;
  `go run ./cmd/openudon check-doc-memory`; and `git diff --check`.
  Shared UI/runtime changes additionally follow the existing one-affected-smoke
  policy. No live model, provider, browser, or full qualification run is a
  default requirement for this additive command scope.
- Step contracts' shared purpose/inputs/outputs/effect fields map losslessly
  onto UWS's pending-step declaration at the recorded UWS revision.
- Existing iCoT-authored packages and supported source families retain their
  behavior. Unsupported command capabilities are explicit. Final package
  approval and trusted execution retain their current independent gates.

## Review and closure record

**Review state:** Passed at iteration 9; no P1/P2-or-higher findings remain.
**Review iterations started:** 9 of at most 10.

### Bounded review iteration 1 — 2026-09-27

- **P1 — bind can write a step without the checker's auth/effect guarantees.**
  `Bind` writes after mapping and operation lookup but does not require the
  selected operation's effect evidence to agree with the declared contract.
  It also treats an empty `SecurityRequirementSets` list as unauthenticated,
  while `step check` correctly treats that absence as unknown. A mismatched
  write operation, or one with unresolved authentication evidence, can therefore
  be bound even though check would reject or mark it indeterminate. Reuse the
  source-backed effect/auth checks before atomic replacement and preserve the
  no-write guarantee on conflict or unknown evidence.
- **Fix and focused verification:** `Bind` now requires a complete APItools
  candidate report, a known authentication alternative, and effect evidence
  compatible with the declared `read`/`write` contract before any write. An
  explicit anonymous alternative remains valid; absent auth evidence, unknown
  effects, unknown contracts, and conflicting effects return `needs_input`.
  Added rejection/no-mutation and explicit-anonymous tests. Focused workspace
  and standalone bind tests pass; full regression rerun is pending.
- **P2 — check/bind CLI invocations do not propagate cancellation.** The
  candidates and flow-review routes use signal-derived contexts, but check and
  bind pass `context.Background()`. Their package APIs also do not consistently
  prioritize a cancelled context over source-inspection diagnostics. Align all
  four routes so cancellation returns the documented structured outcome and a
  bind cancelled before commit cannot mutate the intent.
- **P2 fix and focused verification:** `step check` and `step bind` now use
  signal-derived CLI contexts and return `request.cancelled` from the package
  APIs before source diagnostics can mask cancellation. Direct cancellation
  tests verify structured blocking outcomes and no intent write. Focused
  workspace and standalone command/package tests pass.
- **Iteration 1 full regression:** `make check`, `GOWORK=off go test -count=1
  ./...`, `GOWORK=off go vet ./...`, `GOWORK=off go test -race -count=1
  ./internal/stepauthoring ./internal/icot/elicitor ./cmd/openudon
  ./internal/stepauthoringcontract`, the tagged UWS C07.2 mapping test,
  document-memory validation, and `git diff --check` pass after these fixes.
- **P2 — adapter family coverage is incomplete.** `step candidates` declares
  all eight APItools source families, but the OpenUdon integration tests cover
  only OpenAPI. A mixed-directory reproduction returned `result.invalid`
  because the identity regex rejects valid punctuation in native selectors or
  Smithy operation keys. Add an all-family adapter test and validate native
  identity strings using bounded printable UTF-8 rather than an OpenAPI-shaped
  character allowlist, so supported source kinds cannot silently disappear.
- **P2 fix and focused verification:** operation keys and native selectors now
  retain bounded printable UTF-8 identities instead of an OpenAPI-only
  punctuation allowlist. A mixed-root integration test covers OpenAPI, Google
  Discovery, AWS Smithy, AsyncAPI, GraphQL, OpenRPC, gRPC/protobuf, and OData;
  all eight produce candidates with source digests and native selectors. The
  focused family test passes.

### Bounded review iteration 2 — 2026-09-27

- **P1 — check/bind can accept schema type or requiredness mismatches.** Their
  deterministic mapping checks verify that names are present, but do not
  compare the step's recursive input/output declarations with the selected
  operation's APItools contract match. A step can therefore be reported
  compatible or written even when, for example, the contract declares a string
  where the operation requires an integer or returns an array. Reuse exact
  source-candidate input/output match evidence; check should report conflicts
  and unsupported cases distinctly, and bind must not write unless these
  dimensions are compatible.
- **P1 — bind does not revalidate the source revision at commit time.** It
  verifies source bytes before building the intent, but only rechecks the
  intent revision inside the atomic writer. A source can change after its
  digest is checked and before the intent replacement, leaving the new intent
  bound to stale operation evidence. Recheck the exact selected source digest
  in the writer's pre-replacement callback and report a source conflict without
  changing intent bytes.
- **P1 fixes and focused verification:** `step check` now reports explicit
  input/output compatibility checks from the exact APItools operation
  candidate, preserving known type/requiredness conflicts as failures and
  unsupported dimensions as indeterminate. `step bind` refuses to write unless
  both dimensions are compatible. Its transactional pre-replacement guard now
  rechecks the exact source digest alongside the intent revision and returns
  `source.stale` on drift. Added type-mismatch, unsupported-schema,
  no-mutation, and pre-replacement source-change tests. Focused workspace tests
  pass; the full regression rerun is pending.
- **P1 — bind's `needs_input` outcomes return the invalid-request exit code.**
  Several mapping, dependency, output, and composite-step branches emit status
  `needs_input` with exit code 2, although the published v1 contract assigns
  exit code 4 to needs-input/blocked outcomes. Correct every branch and add CLI
  coverage so consumers can distinguish a missing decision from malformed
  input.
- **P2 — conflicting credential aliases can be silently ignored.** A scheme
  whose source name and normalized credential slot are both valid symbols can
  receive different symbolic bindings under each alias; `bindStep` currently
  selects the scheme-name entry and ignores the slot entry. Reject conflicting
  aliases before write and test byte preservation.
- **P1/P2 fixes and focused verification:** all `step bind` `needs_input`
  branches now return exit code 4, matching the v1 transport table; a CLI test
  verifies clean JSON, the exact status/code, and no intent creation. Credential
  bindings supplied under both a source scheme name and its normalized slot
  now fail if their symbols disagree. A source fixture using `api-key` proves
  the alias conflict is rejected without an intent write. Focused package and
  CLI tests pass; full regression rerun remains pending.

### Bounded review iteration 3 — 2026-09-27

- **P2 — `step check` can accept a mapping to an undeclared workflow input.**
  Required-input checking only verifies that a mapping key or value is present;
  unlike `step bind`, it does not validate `inputs.<name>` references against
  the current intent's declared inputs. A step mapped from a missing workflow
  input can therefore receive a `compatible` assessment even though binding
  the same mapping returns `needs_input`. Reuse the bounded input-reference
  validation for the selected step's `with`/field mappings and report the
  unresolved reference as a deterministic failure.
- **P2 fix and focused verification:** `step check` now validates each selected
  step's `with` and binding-field references against declared workflow inputs,
  reporting `mapping.workflow_inputs` as a deterministic failure. A fixture
  regression proves an undeclared input cannot receive a compatible
  assessment. The v1 contract lists the new stable check code. Focused
  `go test ./internal/stepauthoring -count=1` and the full workspace, standalone,
  race, tagged-UWS, boundary, memory, vet, and diff checks pass after the fix.

### Bounded review iteration 4 — 2026-09-27

- **P1 — native-source operation references cannot round-trip through check or
  bind.** `step candidates` returns APItools' native selector for each source
  family, but check and bind reconstruct selectors from `OperationSummary.Path`
  for every non-OpenAPI family. Google Discovery, Smithy, AsyncAPI, and gRPC
  selectors are not their HTTP/RPC paths, so a valid candidate can be rejected
  as stale/not found and cannot be checked or bound. Resolve the exact operation
  using the candidate API's source selector and operation identity, and add
  candidate-to-check/bind round-trip tests for native families.
- **P1 fix and focused verification:** check and bind now resolve the operation
  from the exact APItools candidate tuple (source kind/id/digest, native
  selector, operation key/id) instead of reconstructing a selector from its
  path. The mixed-source fixture now passes each of its eight family candidates
  through check and bind's operation-resolution path. Focused authoring and CLI
  tests pass. Iteration 4 full regression also passes: `make check`, standalone
  `GOWORK=off go test -count=1 ./...` and vet, workspace vet, affected race
  tests, the tagged UWS C07.2 mapping test, boundary check, document-memory
  validation, and `git diff --check`.

### Bounded review iteration 5 — 2026-09-27

- **P2 — candidate discovery and bind do not share the package's source-path
  inventory.** The project and `sourcecatalog` retain `discovery/` as a legacy
  Google Discovery directory, but `step candidates` scans only
  `google-discovery/`, silently omitting those existing documents. Also,
  candidate discovery excludes supported security sidecars while `step bind`
  can resolve a caller-supplied source ID to one. Share the family-directory
  alias rules and sidecar exclusion across discovery and bind resolution, and
  test the legacy path through candidate/check/bind plus direct sidecar rejection.
- **P2 fix and focused verification:** candidate discovery and exact-source
  lookup now recognize both `google-discovery/` and the package's legacy
  `discovery/` alias as the `google-discovery` family. Candidate discovery,
  check, and bind all apply the shared advisory-security-sidecar exclusion.
  Contract docs list the alias; tests cover candidate/check/bind round-trip
  through it and reject both direct bind lookup and intent selection of a
  sidecar. Focused authoring tests and the full workspace, standalone, race,
  vet, tagged-UWS, boundary, document-memory, and diff checks pass after the fix.

### Bounded review iteration 6 — 2026-09-27

- **P2 — active M87 memory still describes the published dependency as blocked.**
  The milestone dashboard and dependency row continue to say APItools M77 is
  uncommitted/unpublished and M87.4 is gated, even though the compatible
  revision `v0.0.0-20260927073943-30c6f3bd5700` is published and M87.4 is
  implemented. The tech-stack note also says check/bind use the APItools
  inventory, while they now resolve the exact operation-candidate tuple. Update
  the active dashboard/status, dependency and execution notes, and stack
  description to the verified release and implemented resolver; retain the
  Kinet/UWS ownership boundaries.
- **P2 fix and focused verification:** the active roadmap, dependency/dashboard
  rows, M87 task ledger, stack description, README entry point, and v1 contract
  now name the published M77 revision, completed M87.4, current candidate-based
  selector matching, the legacy Google Discovery directory, and sidecar policy.
  Kinet and UWS ownership/release limits remain explicit. `check-doc-memory`
  passes with only the expected note that v44 already records this direction;
  `git diff --check` passes. Full implementation regressions passed immediately
  before these documentation-only reconciliations.

### Bounded review iteration 7 — 2026-09-27

- **P2 — unsupported schema details can hide a known candidate mismatch.**
  `applyUnsupportedDimensions` unconditionally changes a match status to
  `indeterminate`. If APItools has already established a concrete input or
  output type conflict while another schema construct is unsupported, the
  candidate result erases that conflict. Preserve `incompatible` while adding
  the unsupported-evidence gap, and test the mixed known-conflict/unsupported
  case.
- **P2 fix and focused verification:** unsupported schema gaps now move a match
  to `indeterminate` only when no known `incompatible` result exists. The new
  candidate integration regression combines a concrete type conflict with an
  unsupported schema reference and verifies both the conflict and gap remain
  visible. The published candidate-result fixture now records the preserved
  `incompatible` status. Focused tests, `make check`, uncached standalone
  `GOWORK=off go test -count=1 ./...`, workspace and standalone vet, the tagged
  UWS C07.2 mapping test, document-memory check, diff check, and affected race
  tests all pass.

### Bounded review iteration 8 — 2026-09-27

- **P2 — check and bind accept self-references and dependency cycles.** Their
  dependency checks verify referenced steps and declared dependencies, but do
  not reject a selected step that consumes its own output or a new dependency
  that closes a cycle through an existing step. Check can therefore report a
  structurally unusable step as compatible, and bind can write a cyclic intent.
  Reject self-references and detect dependency cycles involving the selected
  step before reporting compatibility or writing; add check and bind
  regressions, including an existing predecessor that depends on the selected
  step.
- **P2 fix and focused verification:** check and bind now detect self-output
  references and dependency cycles along the selected step's prerequisite
  graph. Check reports `dependency.cycle` as a failed check; bind returns
  `needs_input` before mutation. Tests cover self-reference and a transitive
  back-edge in both commands, and assert rejected binds preserve the intent
  bytes. Both focused regression tests pass; full regression is pending.
- **P2 — missing and unsupported version diagnostics are inconsistent.** The
  public v1 registry distinguishes a missing/invalid request from a provided
  unsupported version, but candidates and bind classify an empty version as
  `request.unsupported_version`, while flow-review reports a provided unknown
  version as `request.invalid`. Apply the same distinction to all commands:
  an absent version is invalid and a non-empty unknown version is unsupported;
  add command-level package tests for both cases.
- **P2 fix and focused verification:** candidates, check, bind, and flow-review
  now consistently classify a missing version as `request.invalid` and a
  provided unsupported version as `request.unsupported_version`. Focused
  package tests pass for the changed candidates, bind, and flow-review paths;
  full regression passes: workspace and standalone `go test -count=1 ./...`,
  workspace and standalone `go vet ./...`, affected standalone race tests,
  `make check`, tagged UWS C07.2 mapping test, Apitools boundary check,
  document-memory validation, and `git diff --check`.

### Bounded review iteration 9 — 2026-09-27

- Reviewed the complete M87 implementation and worktree diff, including the
  public JSON schema and fixtures, all four CLI routes, source discovery and
  operation identity, authentication/effect checks, intent transaction and
  rollback behavior, cycle rejection, model-review boundaries, docs, and the
  fixes from iterations 1–8. No P1/P2-or-higher finding remains. The review
  gate passes at iteration 9.
- Final verification: workspace and standalone `go test -count=1 ./...`,
  workspace and standalone `go vet ./...`, affected standalone race tests,
  `make check`, tagged UWS C07.2 mapping test, `check-apitools-boundary`,
  `check-doc-memory`, and `git diff --check` all pass. APItools resolves to the
  published M77 revision recorded above. Kinet W03 remains an independently
  owned consumer, not an M87 acceptance prerequisite; no Kinet/UWS/Ramen files
  or remotes were changed.

Before the first full-milestone review, persist iteration 1 and its baseline.
Resume an interrupted iteration without resetting the count. Fix all P1/P2
or higher findings and rerun affected verification before reviewing the full
milestone again. At the limit, retain blocked findings and request direction;
never declare acceptance merely because rows are terminal. Consolidate current
facts and reconcile Kinet/Apitools handoff requirements before normal retirement.

## Planning verification

The approved file scope is exactly `milestone.md`, this status file, and
`prompt-v44.md` / `result-v44.md`. Planning checks cover the installed runner's
read-only ledger parser, ID uniqueness, file links, document-memory validation,
and whitespace. These checks validate planning structure only, not command
implementation or milestone acceptance.

Verified 2026-09-26: the installed runner parsed both active ledgers, with six
pending M87 rows, no in-progress rows, and E21 unchanged as complete. Active
and retired IDs do not overlap; all 17 local links/anchors in the four changed
documents resolve. `go run ./cmd/openudon check-doc-memory` and
`git diff --check` passed; whitespace was also checked in the three new files.
No implementation tests, commit, or publication were performed.
~~~~~~~~
