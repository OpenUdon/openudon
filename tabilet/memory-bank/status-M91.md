# Status M91 — iCoT inventory and behavior-preserving extraction

**State:** M91.1 complete; the user approved the 23-capability inventory. M91.2 is complete. M91.3 awaits accepted/published Authoring M29; later extraction tasks remain pending.

**Goal.** Move shared implementation out of iCoT while keeping all current consumers working.

**Dependencies.** Accepted/published M90 at ed5b206a524e6e193123b2d06714b75160379560; current P07/P08 and E23/E24 behavior. Udon M45 precedes this milestone in the serial launch order, not as an extraction API prerequisite.

**Downstream.** M92–M95; Kinet W08; M93 supplies the W8M replacement contract.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled. (OpenUdon's convention; W8M's differs.)

## Scope and contract

Inventory terminal authoring, ui/control protocols, browser authoring/transactions, lint/repair/reconcile/report, evaluation/variants/scorecards, corpus/provider tooling, worker dispatch, qualification, docs and W8M consumers. Retain capabilities through replacement commands or Kinet; do not silently discontinue an unmatched journey. Keep registration and authenticated/TOTP capture. Present any additional discontinuation for explicit approval before removal. Extract artifactwriter, review/sanitization, source discovery/catalog planning/session types and browser/qualification helpers into neutral packages, without duplicating them. Rebase non-iCoT importers and evaluation; iCoT remains functional in 5A. Preserve P07/P08 v11 dispatch and E23/E24 current-stack inputs, including any already-landed W27 upstream handoffs.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| M91.1 — Inventory journeys and consumers | `[+]` | Record retained/replaced/discontinued journeys and replacement checks; inventory both W8M capture modes, expert CLI and evaluation surfaces; approve any new discontinuation explicitly. |
| M91.2 — Extract artifact writing and draft review | `[+]` | Move transactional artifactwriter and shared review/sanitization into neutral implementation; prove byte-equivalent fixtures. |
| M91.3 — Extract discovery and session logic | `[!]` | Move local/catalog discovery, planning and session types; decouple non-iCoT authoring consumers from Authoring icot. |
| M91.4 — Extract browser worker and qualification helpers | `[ ]` | Move process dispatch/launch and scenario/registration helpers; preserve both capture modes and current-stack inputs. |
| M91.5 — Rebase evaluation | `[ ]` | Move lint/evaluation, variants and scorecard callers off cmd/icot without changing their fixture corpus or expected coverage. |
| M91.6 — Prove equivalence, review and publish | `[ ]` | Check imports, fixtures, evaluation and owner qualification; preserve P07/P08 dispatch; publish accepted source. Consumer qualification belongs to Kinet W08, without weakening its production pin. |

## Acceptance and verification

Step conformance fixtures remain byte-identical and evaluation/qualification results comparable. No non-iCoT importer depends on internal/icot or authoring/icot. Qualify extraction within OpenUdon; Kinet's existing script always checks pinned M90 and is not evidence for this revision. Kinet W08 subsequently qualifies adopted M92 containing this extraction. Run make check, owner fast/smoke gates and required frozen integration qualification; bounded review and publication.

Default checks use fake providers, disposable roots and loopback fixtures. No live target operation or deployment is authorized. Preserve package instructions, one execution owner and exact upstream reconciliation before advancing. Task commits/publication follow only the separately launched goal's explicit policy; this planning approval performs neither.

## Provenance and lineage

Source: “Stage 5 draft reconciliation” (Kinet `stage5/REVIEW.md` findings F01–F10 and its accompanying drafts; refinement findings SR01–SR06; see the findings table in Kinet `docs/kinet-order.md` §6). Source priorities: not supplied. Draft baseline: e12a6488b86cafddb9298c7917de84fbc1cc85ff. Revalidated at this repository's full HEAD `e12a6488b86cafddb9298c7917de84fbc1cc85ff`. The draft source is read-only. Relevant uncommitted evidence: Kinet's `stage5/` drafts and APItools' approved M81/M80 planning changes; no implementation changes were used or made. User approved the complete dispositions and planning-file actions on 2026-09-30.

F03 (not supplied / P1, confirmed) is owned here: stepauthoring/{bind,source,flow_review}.go, browserscenario/, eval/authoring_variants.go and authoring/ import iCoT code. F09 (not supplied / P2, confirmed) is owned here: preserve internal/{trustedrunner,udonrunner}/browser.go and internal/browsersystem/inputs.go. M95 consumes that compatibility evidence.

Lineage: M87–M90 remain accepted step/handoff foundations; E23/E24 are retired. Preserve E21/P07/P08 completed rows and review counters; ordinary closure is separate from this planning intake.

## Closing review

Persisted iteration count: 0/10. Not started; this reconciliation is intake, not a closing-review iteration. Resume any interrupted future review at its persisted number. Acceptance, exact source/build revisions, publication and downstream reconciliation remain pending and must be recorded from observed evidence before normal package retirement.

## Execution reconciliation — 2026-09-30

Udon M45 precedes M91 in the approved serial order and is now accepted with
published closure `6c4fb8c80a06179f8c3e33ebe685b2d44f66d273`. Its qualified
source is `238f2e487d50ffec057b7a109a35c9db03f59c55`; M92 owns adoption,
not this extraction. APItools M81/M80 remain accepted/published prerequisites
for M94. The producer handoffs are recorded in each pending consumer status.

E21/P07/P08 terminal task rows and passed review counts were preserved and their
unfinished normal history retirement completed before M91. They resolve through
OpenUdon's history index; no old task, review, browser run or W8M claim was
reopened. P08's v11 pairing is the current behavior; P07's v10 preparation
result retains its historical scope. No W27 capture/live operation was touched.

The user updated the confirmed launch reference to automatically authorize
scoped pushes, including exact-diff approval, after the execution owner's
recorded diff/remote/verification checks. Other named human checkpoints remain.
M91.1 inventory dispositions still require user approval under that reference;
Gate 5B later also requires accepted replacements and demonstrated journeys.

## M91.1 inventory prepared

[Retained-journey inventory](../../docs/icot-retirement-inventory.md) contains
23 entries covering terminal/seed/replay flows, source discovery, transaction
writing, step commands, both browser captures and TOTP, UI/control, workers,
expert review/repair and evaluation, legacy packages, v11 dispatch,
qualification, W8M authority and release gates. Every entry has an existing
package owner and explicit replacement evidence; all remain available in 5A.
There are zero proposed additional capability discontinuations.

Current import inspection confirms the extraction seams, including Authoring's
neutral readiness/session primitives and transport-specific progressive
adapters. The inventory is reviewable; no source file, provider/account,
workflow artifact or qualification binding changed. M91.1 stays in progress
at the retained inventory-disposition human checkpoint before extraction.
The later Gate 5B still requires completed replacements and both demonstrations.

## M91.1 approved and complete

The user explicitly approved the 23-capability inventory. No capability
removal is authorized beyond the existing later M95 entry-point replacement;
all retained behavior needs replacement evidence before Gate 5B/M95 closure.
Inventory source paths, 23 unique entries, prerequisite retirement envelopes,
documentation-memory checks and diff checks pass. Closing review stays 0/10;
this inventory approval is not milestone acceptance or Gate 5B approval.

## M91.2 selected — shared writer and elicitor review

Move `internal/icot/artifactwriter` to `internal/artifactwriter` and the coupled
elicitor/review implementation to `internal/elicitor`, preserving package names,
APIs and implementation bodies. Moving the coupled elicitor early carries its
session/discovery code into neutral ownership without copying private helpers;
M91.3 still owns removal of its transport-specific Authoring iCoT dependency.
Rebase callers and current test-command paths, fix only changed fixture-relative
paths, preserve historical harness/qualification records, and verify identical
tracked corpus/step fixture bytes. No iCoT command or user capability is removed.

## M91.2 complete — relocation equivalence

The writer and all 78 coupled elicitor files moved without copied implementation.
All 81 moved files match their `968f334` bodies after import-path and two
fixture-relative-path adjustments. All 405 tracked evaluation/step fixture files
match the pre-task digest inventory at
`/var/tmp/openudon-m91-fixtures-before.json`.

Focused writer/elicitor/step/iCoT/browser tests pass, as do transactional writer
and discovery race tests, and step optimistic-write/stale-intent/unsafe-review
race tests. `make fast` passes (full Go tests and documentation-memory checks).
Logs: `/tmp/openudon-m91-2-focused.log`,
`/tmp/openudon-m91-2-race.log`, `/tmp/openudon-m91-2-step-race.log`, and
`/tmp/openudon-m91-2-fast-fixed.log`.

The first full test detected an accidentally rewritten historical integration
selector. Its original command paths were restored; immutable v2/v3 inventory
hashes and existing report validators now pass. M91.5 owns evaluation rebasing
and any needed versioned selector; no historical report, lock, source fixture,
command capability or approval policy was changed. Final import independence,
smoke/frozen qualification and closing review remain milestone work.

## M91.3 dependency discovery — awaiting scope decision

Read-only inspection of Authoring's published HEAD
`b417eb681746476cca80bdb14cde3e3c96de980c` and OpenUdon's pinned
`2f73e3526583d303bd67a505a54035f8b1618ae0` finds the generic progressive
loop, prompt compatibility envelope and clone-based `InterviewBinding` in
`github.com/OpenUdon/authoring/icot`. Neutral interview/readiness/session/prompt
primitives exist, but no neutral public engine entry point exposes that complete
contract. `GOWORK=off go list -deps ./internal/stepauthoring` confirms the
remaining transitive `authoring/icot` dependency after M91.2.

Authoring's instructions assign generic loops, sessions and atomic bindings to
Authoring. A recommended additive prerequisite would relocate the existing
implementation into a neutral Authoring engine and keep the old `icot` API as
compatibility aliases/forwarders. Ramen still imports the old API; its source
and ledger were not changed. This is a compatibility-preserving extraction,
not the deferred Authoring iCoT/icotcli retirement.

Authoring has no pending owner for that prerequisite and is absent from this
run's execution/publication targets. Under Kinet GOAL's scope stop, M91.3 is
blocked awaiting the complete prerequisite proposal's approval and explicit
extension of the goal. No Authoring ID, planning file, code, commit or remote
was changed. M91's review count remains 0/10 and later rows were not started;
completed relocation evidence is preserved in `1a2570232cdbe8cafecda7e580f91b8a2af12746`.

## Upstream prerequisite approved

The user approved Authoring M29, extended the existing goal and authorized
Authoring publication. M91.3 remains blocked on its accepted/published exact
revision, rather than on user approval. Resume after M29 handoff and record
its source/module/qualification; retain M91.1/M91.2 and review count 0/10.
