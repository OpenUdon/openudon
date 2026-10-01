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

## APItools producer reconciliation — 2026-09-30

APItools M81/M80 qualified source is published at `fb132631c9827eae5f2ec4503d03f21eabfb4113`
(`github.com/OpenUdon/apitools v0.0.0-20260930205753-fb132631c982`). Its observed Go
`Origin.Hash` equals the full source revision. Source review and all required
producer/consumer checks passed; package-local retirement evidence resolves
through APItools `tabilet/docs/history/status-M80.md` and `status-M81.md`.
The cross-package goal owns this downstream reconciliation; APItools performed
no sibling implementation or dependency edit. Every task here remains pending.

The M94 discovery/provisioning replacement must qualify against that exact
APItools source and its five-outcome/explicit-root/native-reference contract
before removal. Producer publication satisfies the metadata prerequisite only;
it establishes no iCoT replacement journey, Gate 5B approval or W28 acceptance.
Retain all existing removal gates and pending outcomes.

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

M95.2 must remove obsolete UI/control/interactive code and assets now under
`authoringui` and `authoringcli`, not merely delete the old `internal/icot`
facade. Expert scorecards and seed/replay cases currently invoke the shared
legacy Main internally; preserve their noninteractive draft/core behavior
through an explicit neutral entry when removing interactive transports. Retain
all inventory evidence gates, and qualify replacement commands independently
before deleting anything. This reconciliation authorizes no early removal.

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

Retain the new pending/simulation commands, schemas, public UWS mock semantics
and qualification markers when removing iCoT transports. M95 removal still
waits for its own stated M93/M94, U07, Gate5B and W28 evidence. Preserve old readers,
locks and package fixtures; replacement runtime adoption requires fresh
owner/consumer qualification rather than reusing M92's binary identity.

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

Preserve the new public capture command and shared embedded worker when removing
iCoT; never remove native validation, private input, canonical reconstruction or
profile import. Inventory, Kinet U07, Gate5B and W8M W28 remain unsatisfied.
