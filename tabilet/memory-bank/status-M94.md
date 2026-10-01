# Status M94 — Catalog discovery and digest-bound source provisioning

**State:** M94.1 complete; M94.2/M94.3 pending. M93 is accepted, retired and published; no M94 acceptance or publication is claimed.

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
| M94.1 — Adopt APItools and expose discovery | `[+]` | Pin the accepted M80 release including M81; require explicit root/index; preserve all five outcomes and producer conformance fixtures. |
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

## M92 exact producer reconciliation — 2026-10-01

Final qualified application source: `96c16acacc7f442858dac8a0fcb36c84991ebddf`;
source/qualification publication independently verified at `bbb03effbe4215cf15473c4dfec561b64b80a122`.
Review 4/10 passed with no open findings. Frozen final producer bundle:
`/var/tmp/openudon-m92-corrected-producer-z89k3b36`, summary SHA-256 `a87277a65341e17b3f2e40daf275197cc02ff4377d160fdd0ba383fb7684ec30`;
CLI SHA-256 `dd109478d24321733fc63b20c163340e4786727fe18f39146c69be714d8edbef`. Published UWS source is
`a7688f54c68f5a75c7cc95aa2b31cea98b31af41` (1.12); exact M45 executor source,
binary and fourteen-source closure remain accepted: source
`238f2e487d50ffec057b7a109a35c9db03f59c55`, executor SHA-256
`cb4b94c968aa3f3de4106a440fdcd02e6c210941eb25e6666b84cfbc7f63868b`,
closure SHA-256
`10d4c613c4882365f2799789d456e8a3b15484b1995a052e334616cad9d0fd59`.
Final frozen full checks and integration v6 passed seventeen required gates
(fourteen named producer tests), zero failed, three optional unrequested.
Fresh native v5 passed39/39 at `7efb58678954a037a54e5d5874020258ce98cdca`.
Its evidence retains that actual source; the final delta changes only pure
simulation inventory, its test/marker, current documents and a new example's
formatting/digest, with browser/runtime/authoring/pin scope proven identical.
Both reports independently verify; temporary display teardown is verified.
Never relabel native evidence or treat a same-version binary as adoption.

Additive contracts and local conformance fixtures are
`openudon.step-pending.v1`, `openudon.simulate-input.v1`, and
`openudon.simulate.v1`. The legacy step-authoring v1 remains unchanged. New
packages default to UWS1.12, existing declarations remain unchanged. Pending
contracts refuse every approval/dry/real path across both artifacts and all
branches/workflows before credential/executor dispatch. Simulation is pure
public mock orchestration and in-memory projection, with no network, browser
worker, credentials or executor. Browser results are mocked contracts, not
page verification; previews grant no action authority. Every task in this
consumer remains pending; resolve producer closure through OpenUdon's history
index after normal retirement, preserving each package's own ledger.

Retain M92's new qualification contexts when separately adopting published
APItools fb132631c9827eae5f2ec4503d03f21eabfb4113. Its catalog/discovery pin
change requires its own exact-source producer and affected compatibility
checks; it is not covered by M92's old APItools executor closure. No copied
index/rank semantics, implicit root or broadened absence claim is authorized.

## M93 exact producer reconciliation — 2026-10-01

Qualified application source: `f1273b622445d60dc7f3ea849e5b7f1a1f1e733a`;
source/qualification publication independently verified at `d5b483afc93dc5b25ac319b1ce590f5d4d6fd682`.
CLI SHA-256 `50ed529b5375c01bc8aab0ef91b2910c55672074794b10e0dbcbcd70c10987e9`.
Qualification: `/var/tmp/openudon-m93-qualified-8heukane/qualification-summary.json`;
SHA-256 `abdb8490206a3e734f009a6f9430d903017635270327366b4e40bdf95e625da6`. Full check/vet/affected race and conformance passed;
native39 in three fresh repeats, integration17/0/3 optional unrequested and
independent verifiers passed. Both actual visible loopback consumers passed,
and the user explicitly confirmed seeing and accepting BOTH journeys.
Review1/10 passed without open findings. Resolve the producer via OpenUdon's
history index and permanent M93 record; do not merge or recreate its ledger.

The actual public command is `openudon browser-capture` with an exact approved
closed `openudon.browser-capture-start.v1` file SHA, safe disjoint package and
private roots, bounded NDJSON `openudon.browser-capture.v1`, random issued
session/event/action references, current revision and immutable command SHA.
Proposal executes nothing; exact approval/refusal consumes one card. Native
origin/action/completion gates remain separate. A successful joined native
capture then requires a separate `import_review` decision and independent
canonical validation before atomic profile/review/receipt publication. The
receipt is `expected/browser-capture/<transaction-id>.json`; result exposes
only profile ID, transaction SHA and read/write effect. Login/submission
recipes classify as write. Worker errors, stale input, expiry, cancellation or
lost output grant no automatic retry. Native registration v4 is the default;
reviewed v1–v3 remain explicit retained choices. iCoT remains on this same
implementation until M95. No hosted capture, live target or runtime authority.

Credential/code values enter the private headed browser only. Reduced labels,
structural URLs, canonical profiles and full event/command/view bodies may be
personal data: keep them transient, never generic durable job/audit payloads.
Model disclosure requires an exact observation-bound user consent. All rows
in this downstream consumer remain pending until its own acceptance checks.

M93 now satisfies the serial producer prerequisite. Catalog implementation still
uses the separately accepted APItools contract; never copy browser or catalog
semantics. M94.1 is next only after M93 normal retirement/publication closure.

## M94.1 execution selected — 2026-10-01

M93 closure was independently verified on origin/main at
`ca3b805456a459d83431cbc6ef3126015a575772`; its qualified source and protocol
remain the exact producer recorded above. APItools source
`fb132631c9827eae5f2ec4503d03f21eabfb4113` and closure
`8a52c3f602988b945a4b5c1960bce8c04170c63d` remain published and unchanged.
No sibling plan/source edit is needed. Implement the adapter over native
DiscoverCatalogOperations and DecodeCatalogDiscoveryRequest, explicit
installation catalog root/registry/index and read-only registration access.
Missing installation configuration is incomplete evidence; never infer global
absence or build an index implicitly. Optional remote lookup remains dual
opt-in and bounded by APItools. Retain the five native outcomes unchanged.

## M94.1 delivered — 2026-10-01

Adopted published APItools module `v0.0.0-20260930205753-fb132631c982`,
Origin.Hash `fb132631c9827eae5f2ec4503d03f21eabfb4113`, module sum
`h1:ELxWOW2xW+3ahSrArRi78JD8kVKDFI7TduZrpBBWgwE=`. The first offline download
found only metadata cached; explicit retrieval of this published revision
succeeded, after which checks ran offline with GOWORK off. No sibling changed.

`step discover` consumes/returns native APItools catalog-discovery/v1;
explicit CLI root/registry/index and optional installation metadata use native
read-only sqlitecache/index validation. Missing configuration/index remains
insufficient evidence; all five outcomes, multiword/empty provider scope,
coverage, relocation, license unknowns and references are preserved. Default
is offline; optional remote requires both installation and request opt-in.
No index build, copied ranking, source write or API/browser execution.
Shared strict JSON preflight additionally refuses duplicate request keys;
initial conformance failure exposed native acceptance of duplicate fields,
then the affected check passed without relaxing semantic validation.

Source-backed conformance copies three exact upstream synthetic fixture files
with full-source/hash provenance and builds native indexes freshly. Complete
CLI/native reports match for all five outcomes and relocated roots. Malformed
requests/flags are refused without echo, and request-only remote opt-in fails.
Focused command/candidate/source tests passed. Full make fast passed with
actual published module and offline environment:
`/tmp/openudon-m94-1-fast.log`; focused vet, document, gofmt and diff checks
passed. Updated product/architecture/stack and operator documentation; old
current-pin wording preserved in the append-only knowledge journal.
M94.2 still owns selected artifact export/provisioning and provenance, and
M94.3 owns final qualification/review/publication. Review remains0/10.
