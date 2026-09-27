# Status M87 — Non-interactive step authoring contract for Kinet

**State:** Pending. Planning approved 2026-09-26; implementation and acceptance
have not begun.

**Specification:** [M87 in milestone.md](milestone.md#m87--non-interactive-step-authoring-contract-for-kinet).
**Direction:** [Evolution v44](../evolution/result-v44.md).
**Priority:** Next OpenUdon implementation milestone, beginning with M87.1.

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

## Task ledger

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed historical with an accepted successor, `[X]` cancelled.
Each row is one implementation commit unit under the governing policy. Keep
one execution owner and at most one general in-progress row. This planning
approval itself authorizes no execution, commit, or publication.

| Item | State | Notes |
|---|---|---|
| M87.1 Define the public step-command contract | `[ ]` | Own bounded versioned requests/results, step-contract identity, source and intent digests, statuses/diagnostics, exit codes, strict JSON behavior, compatibility policy, schemas, and initial consumer fixtures. Define purpose, inputs/outputs, accounts/destinations, and confirmed effect without inventing UWS semantics. Specify explicit scaffold creation, read/write boundaries, conflict and indeterminate outcomes, model configuration, and source-family support. Record the Apitools metadata interface needed by M87.4. No Kinet milestone or new Apitools API is a prerequisite for this row. Shape purpose, inputs, outputs, and effect exactly as UWS C07.2's pending-step declaration; agree it with C07.2 before freezing and record the UWS revision used. OpenUdon-only fields sit alongside the shared ones. Aligned 2026-09-27 at the owner's request (Kinet `docs/kinet-order.md` X2). |
| M87.2 Implement step check | `[ ]` | Depends on M87.1. Read-only validation of one exact selected step against its contract, source operation/digest, required mappings, outputs, authentication alternative, relevant dependencies, and effect constraints. Reuse existing validators and shared logic. Distinguish deterministic results from unresolved semantics, missing evidence, and unsupported capabilities. Add deterministic positive and rejection fixtures; no provider/model operation is needed for structural checks. |
| M87.3 Implement step bind | `[ ]` | Depends on M87.1–M87.2. Create or replace one identified step in intent.hcl using validated explicit input and optimistic revision/source checks. Initial creation requires an explicit workflow scaffold. Preserve unrelated steps, blocks, and content; reuse transactional filesystem protections. Reject stale/malformed/unsafe inputs before replacement, keep rejected writes byte-identical, and distinguish indeterminate outcomes. Test repeat requests, source drift, concurrent edits, and unrelated-content preservation. |
| M87.4 Implement step candidates with Apitools metadata | `[ ]` | Depends on M87.1 and a compatible published Apitools S2a metadata API. Return ranked exact-source operations, consumer-readable summaries, request/output compatibility evidence, OR-of-AND auth needs, and read/write/unknown effects with provenance and visible gaps. Unknown effects are conservative; HTTP verbs alone do not authorize read claims. Fixture adapters may precede upstream delivery but cannot satisfy production integration acceptance. Record upstream version and evidence when available; never duplicate generic classification/ranking locally. |
| M87.5 Expose advisory flow-review | `[ ]` | Depends on M87.1. Extract/reuse existing local and model-assisted review logic for a read-only non-interactive command while preserving current iCoT callers. Require explicit model configuration for model review; return structured completion/skipped/unavailable/failure outcomes and sanitized advisory findings. No file repair, package approval, hidden prompts, or swallowed failure presented as a passed review. Test through fake model responses and cancellation. May proceed before M87.4 when upstream delivery is pending. |
| M87.6 Qualify contracts and consumer handoff | `[ ]` | Depends on M87.1–M87.5 and real upstream metadata integration. Complete published versioned schemas/fixtures, independent consumer parsing examples, CLI and package-composition checks, malformed/stale/digest/auth/effect/write-preservation negatives, and existing iCoT/package regressions. Document operator entry points and Kinet fixture consumption; update current memory only for verified behavior. Run standalone, race, boundary, documentation and repository checks, then the bounded full-milestone review. Prepare a release-ready handoff; remote publication and Kinet W03 adoption remain separately authorized/owned. Include a fixture mapping each step contract's shared fields losslessly onto UWS's pending-step declaration at the recorded UWS revision. |

## Dependency and execution policy

M87.1 is actionable under a later implementation request. Contract-first work
must not wait for Kinet W03's first row. M87.2–M87.3 use contract-bound fixture
metadata until production metadata can be connected, without claiming that
fixture evidence satisfies the upstream integration requirement. M87.5 is
independent of that delivery once M87.1 settles its public shape. If Apitools
delivery remains unavailable when M87.4 is selected, record the external gate
and continue only independent authorized work; M87.6 and overall acceptance
cannot pass with a stub or permanently unknown replacement for the requested
classification/ranking capability.

Apitools owns its own planning and implementation for consumer summaries,
semantic effect metadata and ranking by step purpose/inputs/outputs. M87.1
must identify the exact required API and conformance evidence; M87.4 records
the published compatible revision. This OpenUdon plan allocates no sibling
milestone ID and authorizes no sibling mutation.

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

**Review state:** Not started.
**Review iterations started:** 0 of at most 10.

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
