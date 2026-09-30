# Status M94 — Catalog discovery and digest-bound source provisioning

**State:** Approved planning, 2026-09-30; every task pending. No implementation or publication is established by this record.

**Goal.** Expose APItools catalog discovery and artifact provisioning without broadening evidence or authority.

**Dependencies.** M93 accepted/published; APItools existing M81 then M80 accepted/published. Consume the exact approved M81.1 contract and M80 release; never edit their plan.

**Downstream.** Kinet W10; M95 removal precondition; W8M W28.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled. (OpenUdon's convention; W8M's differs.)

## Scope and contract

step discover consumes explicit catalog-root/index configuration and returns APItools' five outcomes: match, ambiguous, no qualifying API within checked scope, insufficient evidence, blocked. Preserve coverage, exclusions, digest/staleness evidence, exact multiword provider keys, authority/license unknowns and bounded rank evidence. Missing roots/documents or unexamined scope are not definitive no-match. No implicit sibling root and no per-call reimplementation of APItools indexing/ranking. Catalog step source add uses M81 artifact-scoped export/materialization with stable native selectors and digest checks; confirmation and package rules remain OpenUdon-owned. Default offline; remote lookup only explicitly enabled, using APItools bounds and provenance.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| M94.1 — Adopt APItools and expose discovery | `[ ]` | Pin the accepted M80 release including M81; require explicit root/index; preserve all five outcomes and producer conformance fixtures. |
| M94.2 — Provision selected catalog artifacts | `[ ]` | Round-trip stable references through artifact-scoped export, verify digests/native selectors and preserve source confirmation/security-overlay provenance. |
| M94.3 — Qualify outcomes and provisioning, review and publish | `[ ]` | Test scoped outcomes, index failures, root relocation and provisioning; publish fixtures and exact accepted revision for Kinet W10. |

## Acceptance and verification

Conformance tests cover all five outcomes, missing/stale index, root relocation, unknown licenses, provider constraints, cancellation and matching discovery-to-export digest/selector identity. Only scoped no-match signals automatic browser fallback; ambiguity asks for intent, missing evidence requests configuration, blockers explain refusal. make check, owner compatibility checks, bounded review and publication; do not make APItools publication depend on this future implementation.

Default checks use fake providers, disposable roots and loopback fixtures. No live target operation or deployment is authorized. Preserve package instructions, one execution owner and exact upstream reconciliation before advancing. Task commits/publication follow only the separately launched goal's explicit policy; this planning approval performs neither.

## Provenance and lineage

Source: “Stage 5 draft reconciliation” (Kinet `stage5/REVIEW.md` findings F01–F10 and its accompanying drafts; refinement findings SR01–SR06; see the findings table in Kinet `docs/kinet-order.md` §6). Source priorities: not supplied. Draft baseline: e12a6488b86cafddb9298c7917de84fbc1cc85ff. Revalidated at this repository's full HEAD `e12a6488b86cafddb9298c7917de84fbc1cc85ff`. The draft source is read-only. Relevant uncommitted evidence: Kinet's `stage5/` drafts and APItools' approved M81/M80 planning changes; no implementation changes were used or made. User approved the complete dispositions and planning-file actions on 2026-09-30.

F02 (not supplied / P2, confirmed) and SR01 (not supplied / P1, confirmed) are owned here. Evidence: internal/icot/elicitor/progressive.go (CatalogPlan), stepauthoring/source.go, and APItools' existing status-M81.md/status-M80.md five-outcome, root/index and export contracts. Relevant APItools uncommitted plans were read-only evidence and must remain unchanged.

Lineage: Promotes G1/S2d, retaining M89 source/confirmation guarantees. APItools M79 is accepted history; M81/M80 are existing separate owners.

## Closing review

Persisted iteration count: 0/10. Not started; this reconciliation is intake, not a closing-review iteration. Resume any interrupted future review at its persisted number. Acceptance, exact source/build revisions, publication and downstream reconciliation remain pending and must be recorded from observed evidence before normal package retirement.
