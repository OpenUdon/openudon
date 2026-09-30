# Status M95 — Remove iCoT after consumer migration

**State:** Approved planning, 2026-09-30; every task pending. No implementation or publication is established by this record.

**Goal.** Remove OpenUdon's iCoT terminal, UI, control and planner after their replacements qualify.

**Dependencies.** M91–M94 accepted/published; Kinet U07 acceptance; approved Gate 5B and inventory dispositions; W8M W28 accepted/published at exact Kinet/OpenUdon revisions.

**Downstream.** Kinet M20, then W8M W29 final adoption. Earlier W28 qualification cannot qualify new M95 binaries.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled. (OpenUdon's convention; W8M's differs.)

## Scope and contract

Remove cmd/icot and remaining internal/icot surfaces, embedded UI assets, application/registration control entry points, iCoT-only planner/reporting, release binary and obsolete CI gates. Remove OpenUdon's remaining Authoring icot dependency, not Authoring's packages. Retain neutral shared implementation, replacement commands, evaluation/scorecards and qualification coverage from M91. Retain build/assess/approval-template/run/package on legacy packages without editing historical .icot files. No unnoticed journey loss; extra discontinuations need approval. Preserve P07/P08 browser v11 dispatch and E23/E24 qualification inputs. Source history and old adopted binaries remain available.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| M95.1 — Verify consumer migration and dispositions | `[ ]` | Check exact W28 acceptance, Kinet U07 and M91 journey replacement evidence; retain supported journeys and approved historical assets. |
| M95.2 — Remove iCoT code and surfaces | `[ ]` | Delete only retired OpenUdon entry points/assets/planner and remaining Authoring icot use after dependencies pass. |
| M95.3 — Update CI and release gates | `[ ]` | Remove obsolete standalone/UI/variants iCoT gates and release binary; keep equivalent neutral evaluation and qualification gates. |
| M95.4 — Qualify legacy packages and retained gates | `[ ]` | Exercise package/build/assessment/approval/run, browser dispatch and current-stack qualification; preserve .icot artifacts and legacy defaults. |
| M95.5 — Document, review and publish | `[ ]` | Name replacements and explicit discontinuations; persist bounded review and publish accepted source for M20 and W29. |

## Acceptance and verification

Verify W8M no longer launches iCoT, all inventory dispositions have evidence, no retained command/gate depends on removed code, and old packages still work. Run owner checks and required integration/browser qualification on frozen source, review with no open P1/P2, update install/startup/operator/tutorial docs and publish. Publication is producer acceptance; W29 separately qualifies final consumer pins.

Default checks use fake providers, disposable roots and loopback fixtures. No live target operation or deployment is authorized. Preserve package instructions, one execution owner and exact upstream reconciliation before advancing. Task commits/publication follow only the separately launched goal's explicit policy; this planning approval performs neither.

## Provenance and lineage

Source: “Stage 5 draft reconciliation” (Kinet `stage5/REVIEW.md` findings F01–F10 and its accompanying drafts; refinement findings SR01–SR06; see the findings table in Kinet `docs/kinet-order.md` §6). Source priorities: not supplied. Draft baseline: e12a6488b86cafddb9298c7917de84fbc1cc85ff. Revalidated at this repository's full HEAD `e12a6488b86cafddb9298c7917de84fbc1cc85ff`. The draft source is read-only. Relevant uncommitted evidence: Kinet's `stage5/` drafts and APItools' approved M81/M80 planning changes; no implementation changes were used or made. User approved the complete dispositions and planning-file actions on 2026-09-30.

F04 (source priority not supplied; local P2, confirmed) is owned here. Evidence: Makefile, .github/workflows/{test,release}.yml, internal/eval/authoring_variants.go and cmd/icot. F03/F09 extraction ownership stays M91; F07 consumer compatibility ownership stays Kinet M20.

Lineage: Consumes M91 extraction and W28 migration without reopening their histories; Authoring retirement remains deferred pending Ramen.

## Closing review

Persisted iteration count: 0/10. Not started; this reconciliation is intake, not a closing-review iteration. Resume any interrupted future review at its persisted number. Acceptance, exact source/build revisions, publication and downstream reconciliation remain pending and must be recorded from observed evidence before normal package retirement.
