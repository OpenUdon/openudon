# Status M91 — iCoT inventory and behavior-preserving extraction

**State:** Approved planning, 2026-09-30; every task pending. No implementation or publication is established by this record.

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
| M91.1 — Inventory journeys and consumers | `[ ]` | Record retained/replaced/discontinued journeys and replacement checks; inventory both W8M capture modes, expert CLI and evaluation surfaces; approve any new discontinuation explicitly. |
| M91.2 — Extract artifact writing and draft review | `[ ]` | Move transactional artifactwriter and shared review/sanitization into neutral implementation; prove byte-equivalent fixtures. |
| M91.3 — Extract discovery and session logic | `[ ]` | Move local/catalog discovery, planning and session types; decouple non-iCoT authoring consumers from Authoring icot. |
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
