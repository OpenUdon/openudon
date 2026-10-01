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

## APItools producer reconciliation — 2026-09-30

APItools M81/M80 qualified source is published at `fb132631c9827eae5f2ec4503d03f21eabfb4113`
(`github.com/OpenUdon/apitools v0.0.0-20260930205753-fb132631c982`). Its observed Go
`Origin.Hash` equals the full source revision. Source review and all required
producer/consumer checks passed; package-local retirement evidence resolves
through APItools `tabilet/docs/history/status-M80.md` and `status-M81.md`.
The cross-package goal owns this downstream reconciliation; APItools performed
no sibling implementation or dependency edit. Every task here remains pending.

M94.1 must pin that exact published module, not a sibling replacement, and
consume `CatalogDiscoveryRequest`/`CatalogDiscoveryReport` with
`apitools.catalog-discovery/v1`. Installation `CatalogIndexOptions` supplies
explicit root/catalog plus read-only registrations; no root is metadata-only.
Nil provider keys search open scope; an explicit empty list never broadens it.
Keep all five outcomes and positive scope gaps. Strong documented purpose needs
at least two terms and half the requested terms, with typed input/output and
requested-effect compatibility; scores alone do not select a match. Critical
selected-field loss cannot prove absence. No-reference providers, stale/missing
metadata, work/link/prompt/context limits and cancellation stay incomplete.

M94.2 uses `CatalogArtifactReference` with exact raw SHA/bytes and native
selector, and `ExportCatalogArtifacts`; copy only selected raw identity and
applicable provider/selected-spec overlays, preserving all selected provenance.
OpenUdon still verifies selector binding and owns source confirmation. Keep
ephemeral remote evidence separate until explicit provisioning/registration.
Remote lookup requires both request and installation opt-in, retains the
eight-second/three-document/20-MiB bounds and public-catalog digest/final URL,
and never writes the local index or proves absence.

M94.3 reuses producer source-backed conformance/round-trip fixtures and adds
its own command/package checks. Existing consumers passed full workspace and
standalone actual-source adoption with disposable modfiles; that compatibility
does not qualify future `step discover`. M93 remains the upstream prerequisite
and no command, source confirmation, integration test or publication is marked
complete by this handoff. The rollback pin remains the original published
APItools M79 revision until M94's own implementation adopts this release.

## M91 exact producer reconciliation — 2026-10-01

Accepted extraction application source: `3fd40d3f874bdcf668a018550112a02cd0d02409`;
qualified review/source publication: `c8f2de71d983bea93dc1c045568c397a10a4eb56`.
Resolve the producer through OpenUdon's history index and its permanent M91
record; no producer ledger is merged here. Review 1 passed, 405 fixture bytes
and three UI assets stayed identical, integration v5 passed 16 required gates
with three unrequested optional gates, and native current-stack qualification
passed three fresh complete repeats (39 stages). Summary SHA-256:
`9a2524deddccc400457ae76d2e432a205898a181bcfe8bdc84f62348b4c28075`.

The single implementations now live in `internal/artifactwriter`, `elicitor`,
`browserauthor`, `browserauthoring`, `authoringengine`, `authoringui` and
`authoringcli`; Authoring is pinned to its published neutral `engine` source
`18056cb6b0c1007dd567a4a825a6b4311a357185`. `internal/icot` is a temporary
legacy forwarding adapter, and the old UI/control/terminal still works during
5A. All current public approval, credentials, cancellation, recovery and
v10/v11 browser dispatch boundaries remain unchanged. Historical report
selectors/locks and `.icot` package data stay frozen. These source facts satisfy
the extraction prerequisite only; every task in this consumer remains pending.

Current discovery helpers are under neutral elicitor/sourcecatalog, and expert
evaluation is available through `openudon authoring`. APItools adoption remains
M94.1 at the separately published producer pin; reuse its metadata/index/rank
logic rather than copying the neutral helper algorithms again.
