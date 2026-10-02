# Status M95 — Remove iCoT after consumer migration

**State:** Active, 2026-10-02; M95.1 verifies accepted consumer migration/dispositions. No removal or final adoption is accepted.

**Goal.** Remove OpenUdon's iCoT terminal, UI, control and planner after their replacements qualify.

**Dependencies.** M91–M94 and M96 accepted/published; Kinet U07 acceptance; approved Gate 5B and inventory dispositions; W8M W28 accepted/published at exact Kinet/OpenUdon revisions.

**Downstream.** Kinet M20, then W8M W29 final adoption. Earlier W28 qualification cannot qualify new M95 binaries.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled. (OpenUdon's convention; W8M's differs.)

## Scope and contract

Remove cmd/icot and remaining internal/icot surfaces, embedded UI assets, application/registration control entry points, iCoT-only planner/reporting, release binary and obsolete CI gates. Remove OpenUdon's remaining Authoring icot dependency, not Authoring's packages. Retain neutral shared implementation, replacement commands, evaluation/scorecards and qualification coverage from M91. Retain build/assess/approval-template/run/package on legacy packages without editing historical .icot files. No unnoticed journey loss; extra discontinuations need approval. Preserve P07/P08 browser v11 dispatch and E23/E24 qualification inputs. Source history and old adopted binaries remain available.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| M95.1 — Verify consumer migration and dispositions | `[+]` | Exact W28 source/build/qualification/closure and U07/Gate5B/23 retained dispositions verified. Existing native/neutral seams mapped; closed draft and current neutral qualification still require M95.2–.4. Historical artifacts and authority preserved. |
| M95.2 — Remove iCoT code and surfaces | `[+]` | Removed actual terminal/UI/control/assets and facade; neutral closed draft/registration builder/public capture/package qualification replace callers. Full offline/unit/vet/focused race, retained native v5 verification and docs passed. Fresh frozen qualification and closing review remain M95.4/.5. |
| M95.3 — Update CI and release gates | `[+]` | Two CLI release builds on all six OS/architecture targets; neutral expert/scorecard/seed and actual draft-to-approved-dry-run passed. Strict docs, affected race and additive current offline/native gate checks passed; frozen qualification remains M95.4. |
| M95.4 — Qualify legacy packages and retained gates | `[+]` | Frozen bb9863 current full/offline4/integration17/native39 and independent verifiers passed; declared legacy1.11 rebuild/assess/approval/dry-run preserved .icot bytes. Separate CI/doc-only correction qualified; no runtime evidence relabeled. |
| M95.5 — Document, review and publish | `[~]` | Name replacements and explicit discontinuations; persist bounded review and publish accepted source for M20 and W29. |

## Acceptance and verification

Verify W8M no longer launches iCoT, all inventory dispositions have evidence, no retained command/gate depends on removed code, and old packages still work. Run owner checks and required integration/browser qualification on frozen source, review with no open P1/P2, update install/startup/operator/tutorial docs and publish. Publication is producer acceptance; W29 separately qualifies final consumer pins.

Default checks use fake providers, disposable roots and loopback fixtures. No live target operation or deployment is authorized. Preserve package instructions, one execution owner and exact upstream reconciliation before advancing. Task commits/publication follow only the separately launched goal's explicit policy; this planning approval performs neither.

## Provenance and lineage

Source: “Stage 5 draft reconciliation” (Kinet `stage5/REVIEW.md` findings F01–F10 and its accompanying drafts; refinement findings SR01–SR06; see the findings table in Kinet `docs/kinet-order.md` §6). Source priorities: not supplied. Draft baseline: e12a6488b86cafddb9298c7917de84fbc1cc85ff. Revalidated at this repository's full HEAD `e12a6488b86cafddb9298c7917de84fbc1cc85ff`. The draft source is read-only. Relevant uncommitted evidence: Kinet's `stage5/` drafts and APItools' approved M81/M80 planning changes; no implementation changes were used or made. User approved the complete dispositions and planning-file actions on 2026-09-30.

F04 (source priority not supplied; local P2, confirmed) is owned here. Evidence: Makefile, .github/workflows/{test,release}.yml, internal/eval/authoring_variants.go and cmd/icot. F03/F09 extraction ownership stays M91; F07 consumer compatibility ownership stays Kinet M20.

Lineage: Consumes M91 extraction and W28 migration without reopening their histories; Authoring retirement remains deferred pending Ramen.

## Closing review

Persisted iteration count: 2/10. Iteration2 FINDINGS, 2026-10-02. M95-R1 (P2): combined --agent --print --yes can enter the agent publisher, bypassing print-only rendering; incomplete --print with --report can write a report. Reject conflicting print/output modes before source discovery or effects and add no-write regression. No acceptance; fix and affected verification precede iteration2.

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

## M94 exact producer reconciliation — 2026-10-01

Qualified application/test source `ee49fe433a4d476f8d28d3d888352293c490dfc6`; source/qualification
publication independently verified at `5a2a2643ff71831ae72cdeca57baf197c1afbf16`.
CLI SHA-256 `2586eccc26088abd08a5448d88c821e50ed05e6b71f2ceab2eeb12c0f3faac1a`. Frozen summary:
`/var/tmp/openudon-m94-qualified-k0khdqy5/qualification-summary.json`, SHA-256 `36445e7dec47d757656dab0eee0db23265112423cfabdff01f4f72fde8f8dc4b`.
Full offline make check/vet, affected race, real-main-dispatch conformance,
legacy compatibility and document/format/diff passed; review1/10 passed with
no open P1/P2. APItools pin is its published M81/M80 source
`fb132631c9827eae5f2ec4503d03f21eabfb4113`. Old fixture/UI/lock bytes unchanged.
Resolve the complete M94 record through OpenUdon's history index; never merge
its ledger or infer this consumer's acceptance from the producer checks.

`openudon step discover` retains native `apitools.catalog-discovery/v1` with
all five outcomes, scope/coverage/license unknowns, exact native references and
explicit root/registry/index/optional metadata flags. No implicit indexing,
root or remote authority. Native discovery request is64KiB, report2MiB,
installation metadata2MiB. Optional remote requires both CLI installation and
request opt-in; native limits/provenance remain, and remote leads are not
exportable registered artifacts without separate explicit acquisition.

Confirmed provisioning is `step source add --catalog --example DIR --request
FILE|-` plus the explicit installation flags. Closed additive request version
is `openudon.step-source-catalog.v1`, command `step.source.add`, kind request,
confirmed true, exact manifest revision, and source IDs with unchanged selected
native references. It privately stages native selected export, independently
checks native selector/raw identity and disjoint final roots, then atomically
publishes raw API bytes, existing source manifest, all provider-link provenance
and applicable advisory overlay files with the one existing writer. Result
retains source-add fields under the additive version and adds provenance_path.
No-confirmation/drift/collision/cancel failures publish no package files;
indeterminate/lost output requires inspection, never automatic replay. Advice
grants no runtime authority. Legacy local source-add v1 remains unchanged.

Public request/report/source/provenance fixtures and full hash manifest are at
`docs/fixtures/catalog-discovery-v1` in the exact producer. References/URLs and
full reports may contain personal data: audit only an allowlisted bounded
projection of producer-declared metadata/digests/outcomes, never full payloads,
private catalog roots, raw source or reconstructed request/body summaries.

Retain both new catalog commands, all native conformance fixtures and shared
source writer/provenance when removing iCoT. U07, Gate5B and W28 still gate
removal; accepted M94 does not authorize early retirement.

## Approved replacement prerequisite — 2026-10-01

M96 exposes reviewed capture adoption and ordinary package authoring without iCoT. All M95 rows remain pending and review stays0/10. M95.1 must verify M96's actual accepted/published contract and downstream M19/U07/W28 adoption; M95.2 must retain its neutral implementation/CLI and fixtures when removing old transports. Do not infer migration from capture receipt import. Reconcile exact producer revision before removal. The additional W27-status pause is cancelled after both approved integrations were independently verified; U07.4 human acceptance and explicit Gate5B remain. Frozen W27 history grants no new live authority.

## M96 exact producer reconciliation — 2026-10-01

Qualified application source `eed683f27d448ca96af90e7bc5987967a6cd0335`, CLI SHA-256 `da4f12772200e7c45a404829ab8485c221d80b05b536bf1fef9d982fe5a45bd3`, build-closure SHA-256 `6b985c2de5bf8483948d5e4de85355d1587163e2867524828643cda8b08c3bf9`. Source/review plus W27 handoff integration independently verified published at `d77f6d51262d0f311910070bc4a43f662260cd7e`. Review1/10 passed, DOC1 resolved. Native39/three fresh repeats and independent verifier, integration17/0failed/3optional unrequested and independent verifier, both capture modes/TOTP public capture→author→build→prepare/promote/inspect/recovery and private display teardown passed. The final permanent build-failure regression is separately full-check/race/vet qualified; production source/dependency/contract/old-fixture bytes remain identical to the actual qualified application. Metadata/test-only later commits are not relabeled as native executions.

`openudon browser-author plan/apply` reuses native reviewed receipt/source/start/operation checks and one neutral lowering/writer/build path. Strict `openudon.browser-author.v1` request uses base64 of exact native start bytes; bounded symbolic declarations/bindings and exact current inventory/request/plan/receipt/transaction digests bind separate ordinary authoring confirmation. Plan is read-only; apply never recaptures, promotes or executes. Structured `build_failed` may follow committed authoring; inspect/reconcile, never infer no-write or replay. Exact original capture receipt digest is required; unsigned replaced receipts are not new capture attestations. Registration retains native approved-origin navigation semantics, not invented profile/initial-URL equality.

M19.3 adopts the published qualified application source in the one shared pin, with its own adapter/worker/native checks; narrowly admits only validated public review artifacts, implements durable prepare/promote/inspect/recover and single-use backup/restore, and leaves frozen external v1 unchanged. M95/M20 retain this replacement; W28 binds accepted M19/U07 plus M96 and qualifies its own consumer, W29 later qualifies final M95/M20. No downstream row, review counter, runtime adoption or live authority is completed by producer acceptance. Normal M96 retirement/closure resolves through OpenUdon's permanent history index once recorded.

## M19 exact accepted-source reconciliation — 2026-10-01

Kinet M19 accepted application/source publication is independently verified at
`d712c1081487ab6fae9a580195d5072f84d43d9f`; review1/10 passed with no open
findings. Clean exact-source CLI SHA-256
`6611df798c6fa7d7811070a9d7e19fba28b3533775bfe546af5c5015d45e740a`;
build-closure SHA-256
`f8b5657d646cd0e316d3f99191b593ee0666e501b0fd0a6e394a3db8f0c34b90`.
Evidence: `/var/tmp/kinet-m19-4-qualified-15rjw7ay/qualification-summary.json`,
SHA-256 `ff8a657a3195e1ac7a7fa4bfcbcf443a6c0e8cb0d1db2ba6266b690073da1661`.
Both fresh native external modes/TOTP, registration verification refusal/approval,
native author/build/delivery, lost promotion output, deleted requester result,
physical process interruption/replay and private browser/display teardown passed.
A later unsafe post-effect destination fix has separate final default/race
coverage; the earlier native attempt is not relabeled as containing it.
Resolve complete acceptance through Kinet `tabilet/docs/history/status-M19.md`.

The frozen `kinet.external-authoring.v1` binds request/owner/destinations/expiry,
exact original input and both modes. Single-use synchronized intent precedes
capture and possible promotion. Native `browser-author plan/apply` and package
prepare/promote/inspect/recover remain OpenUdon semantics at qualified M96
`eed683f27d448ca96af90e7bc5987967a6cd0335`. Separate issued capture/import,
ordinary author/build and exact delivery approvals are retained; no runtime
operation is delegated to Kinet. `--recover` never captures/authors/promotes
again, including after expiry. Possible delivery stays `recovery_required`
without exact native selection proof, even when unsafe publication is refused.
Durable terminal records reconstruct lost results; backup preserves consumed
records. W8M must independently validate current native package evidence and
matching terminal result before use; historical outcome alone is not freshness.

All consumer rows/review counters remain pending and unchanged by reconciliation.
U07 owns readable owner/deadline/issued questions, cancellation, no-store transient
presentation and reconnect without automatic decision replay. M95/M20 retain
the public native replacement through final pin adoption. W28 qualifies its
accepted M19/U07/M96 stack; W29 separately qualifies final M95/M20. Fresh U07.4
human-visible acceptance and explicit Gate5B still precede W28. The additional
W27-status pause is cancelled after the verified integrations; no frozen W27
record or expired M93 desktop is reopened.

## U07 exact accepted-source reconciliation — 2026-10-01

Kinet U07 accepted after review1/10 with no open P1/P2/higher findings.
Qualified application/task source and independently verified publication:
`d3589d4742272b3d024328192768374da4c0c637`. Clean CLI SHA-256 `c69f3ddfa80d5e25ff94a049d3579218c5a0bad8f6ab7601b3b323ba20d2673f`;
build-closure SHA-256 `21cf7ea4a26d066e7ba3fa7b5607860e01d9d92cf3d2a54dfca55aad7a0f0772`. Qualification and build contexts:
`/var/tmp/kinet-u07-4-qualified-zk6meqih/{qualification-summary.json,build-closure.json}`.
Full default, UI85/embed and jobs/server race passed. Fresh local/rootless
catalog and native pending/refusal/resolution passed; bound read/write mock
preview made0 HTTP reads/writes/executor calls. Both fresh embedded UI/native
login/TOTP and registration verification-refusal-approval-review-import journeys
passed twice with distinct capture IDs. The user explicitly accepted both after
the slower repeat. Private launch/display/browser roots were joined and auth
removed; no registration submission or real account/target/model was used.

Resolve complete U07 acceptance via Kinet's package-local history index and
`tabilet/docs/history/status-U07.md` after closure. Production bytes qualified
from190cca6 plus an explicit manual test fixture are committed at the exact
source above; the clean detached CLI build has separate recorded provenance.
M19 external v1 semantics and the shared accepted M96 producer pin remain
unchanged. UI controls delegate exact current owner/native decisions, preserve
separate capture/import/plan/delivery authority and current deadlines, and never
replay an action on reconnect. Hosted external/capture remains refused.

This is producer acceptance only. Explicit Gate5B remains unapproved and no
consumer row/review count is advanced. W28 must qualify both modes and validate
original bindings, current native package evidence and terminal result before
use; old W27/M19/U07 evidence is not fresh W28 runtime adoption. M95 still waits
for W28 acceptance; M20 and W29 independently qualify the final changed pins.
Authoring and udon-ui retirement remain deferred.

U07 normal retirement/closure publication independently verified at Kinet
`29d0a0bf47ccbd3de1db670be89d7a357c1a39b0`. The qualified application/task
source remains `d3589d4742272b3d024328192768374da4c0c637`; closure changes
only documentation and retirement records. Its permanent history record exists
and literal source/status equality is validated. Gate5B is still unapproved;
this reconciliation advances no consumer task or operational authority.

## Explicit Gate5B approval — 2026-10-01

User answered “Approve Gate5B and continue” after accepted/published U07,
approved23-capability inventory and both retained journey demonstrations.
Resume the existing order W28 → M95 → M20 → W29 with task commits and standing
scoped normal publication. Preserve package-local ledgers, source/build evidence,
no-automatic-expensive-cache-fallback and single execution ownership. No live
account, target operation, M17 deployment or extra discontinuation is authorized.
Receipt: `/var/tmp/kinet-stage5-gate5b-receipt.json`.

## W28 exact accepted consumer reconciliation — 2026-10-02

W28 acceptance/publication independently verified at
`7cbea4933adf6a8c55864ddb6257e8d33edd0950`. Executed clean W8M source is
`46accdb39f57597c3dd50640e8c2b2f13a834e69`; the acceptance commit changes
only documentation/records and must never be relabeled as executed source.
Review2/10 passed, W28-R1 operator guidance resolved, no open P1/P2.

Selected Kinet source `d3589d4742272b3d024328192768374da4c0c637`, CLI SHA256
`c69f3ddfa80d5e25ff94a049d3579218c5a0bad8f6ab7601b3b323ba20d2673f`;
OpenUdon source `eed683f27d448ca96af90e7bc5987967a6cd0335`, clean rebuilt CLI
SHA256 `2d6e75fa6db07c3d2e22703ccc42da94fdb7a373d640faecc6cfc92350b09020`.
This is an explicitly different build from M96's original da4f127 CLI; both
retain their actual evidence identities. Exact19-source/tool/environment closure
is in the frozen W8M selection and private records, not inferred from versions.

Private evidence root: `/var/tmp/w8m-w28-4-authorized-20261002-m1l8cy05`.
Summary SHA256 `6ec63e46ac139a4be985e8292d111f2142730eb2f5c11283d717253cde58314a`;
fresh-v4 SHA256 `bec9c641c7af0b87d8488bdcdc97e5506d0247de6b8925e9db29c5d2af4ca9e8`;
reuse-v5 SHA256 `d2b03f66793ec366b106770699ecad74c304a53447c701dd84671cfa698881f5`.
Native39/three passes, fresh and reuse each three new authenticated/TOTP and
actual embedded registration pairs, separate verification refusal/approval,
native selected delivery and counts0/1/3 independently passed. Native proof
was not rerun for reuse. All worker/display owners joined, zero survivors/forced
cleanup; outer auth removed. Total40.343minutes versus estimate40–45minutes.

W28's contract preserves packet/agent/origin/goal/dashboard/scope/destination/
diagnostic/deadline/TOTP bindings and independent terminal/package inspection.
It launches Kinet serve only, never iCoT. This satisfies the old-entry consumer
migration prerequisite, not final M95/M20/W29 adoption or live action authority.
All tasks in this consumer remain pending until their own verified execution.
Preserve retained23 dispositions, historical .icot artifacts/readers/locks and
W27's frozen outcomes. M95 changes invalidate earlier producer identity; M20
must run its exact-pin gates, and W29 freshly qualifies final consumers, with
explicit seed selection if current cache bindings fail. No automatic fallback.

W28 normal literal retirement/closure is independently verified published at
`fd571ef0db8ab438a33cf454dd859f0fbb91fc5d`. Resolve the permanent producer at
W8M `tabilet/docs/history/status-W28.md`; source46accdb and actual runtime
identities above remain unchanged. Original specification SHA256
`c3194567a7a6635073ade10fd126b6661ec1e4d4137106ac171ce5e91e93def9` and
status SHA256 `a24c8455de65b2872a9e88a6273fe5e1e8fd66170a5a93e1e6369514fe97c806`
were compared before removal. W29 is the next planning owner, awaiting M95/M20
with no task/operation in progress. Remaining verified order:M95 → M20 → W29.

## M95.1 migration/disposition inspection — 2026-10-02

One execution owner; worktree contains only this session’s exact W28 downstream
reconciliation. Actual owner planning/instructions are regular tracked files,
not symlinks to Tofu. M91–M94/M96, Kinet U07 and W28 resolve through their own
accepted/retired records. Gate5B and all23 retained/replaced dispositions were
explicitly approved. W28 final closurefd571ef independently published.

Authoring’s remaining icot dependency is already absent: all three internal
authoring adapters use published `authoring/engine`. Preserve that dependency;
no sibling Authoring change is needed. Actual remaining removal seams include
cmd/icot, internal/icot, authoringcli’s legacy Main/UI/control/browser entry
wrappers, authoringui assets/transport, native qualification and scenario callers
and CI/release/docs. Do not delete only the compatibility facade.

Two replacement contracts still require M95 implementation before removal:
J02/J17 need a closed noninteractive draft entry for seeded/from-example/print/
report/replay use; old Main is still called internally by scorecards. J21 needs
neutral current qualification selectors and fixtures in place of legacy UI/control
gates, preserving frozen historical version readers. These are explicit M95.2–.4
work, not an extra capability discontinuation or upstream blocker. Public
capture/browser-author/package/step and expert commands already cover retained
native authority. Kinet owns interactive UI/session/delivery replacements.

M95.1 verification passed: all23 entries have explicit retained owners and
required checks; all five native prerequisite retired records have completed/
passed metadata; actual U07/W28 closure commits contain permanent records;
Authoring icot imports absent; check-doc-memory and git diff --check passed.
No new evolution version: implementation advances the approved retirement
boundary, not a changed direction or capability disposition. Task1 completes
inspection only; every subsequent implementation/qualification row pending.

M95.1 source/reconciliation publication independently verified at
c6b590cf0642bd81a5802fa860e3a6b48f2287c4. M95.2 is now the sole in-progress
row across the combined goal. Establish and check closed seeded draft/plan and
neutral qualification replacements before deleting their callers/transports.
No live/model/native run is started by this implementation transition.

### M95.2 neutral replacement preparation — in progress

Added closed `openudon authoring draft` and non-executing `browser-plan`;
scorecard callers now use the noninteractive draft path. Missing/partial input
returns the existing structured frontier without terminal reads or publication;
print is read-only and publication requires --yes. Focused tests caught JSON
stdout contamination by a progress preamble; corrected and rerun passed.

Moved the single pure registration field-definition builder and its tests into
`internal/registrationdraft`, with temporary forwarding glue in the old UI until
replacement qualification is ready. The neutral registration-draft command
retains typed/conditional fields, reviewed query disclosure, symbolic bindings
and explicit deferred success proof; it starts no browser and writes no package.
Existing builder and unsafe-source checks passed, plus old UI consumer-focused
checks while its transport is still present. A first new command compile error
was fixed before successful focused checks; neither failed run is acceptance.

No iCoT transport or asset has yet been deleted; native/report selector and
actual public capture/package qualification replacements remain in progress.
Temporary forwarding and test fixture glue must be removed with the old UI,
never committed as a second implementation. No task2 commit/qualification or
whole-milestone review is claimed. Source worktree changes belong to this owner.

M95.2 focused development smoke first attempt `/var/tmp/openudon-m95-neutral-smoke-1ozuganu`: failed at120s, display joined/auth removed. Second diagnostic attempt `/var/tmp/openudon-m95-neutral-smoke-opzu0771` reached public capture imported frame22, exposing a harness EOF handshake error: after the terminal result the caller must close input before draining stdout/Wait. Fixed in the same neutral qualification owner; neither attempt is accepted qualification and no operation/receipt is replayed. New focused run uses a new disposable workspace. Whole review remains0/10.

### M95.2 verified removal — 2026-10-02

Removed cmd/icot, internal/icot, authoringui/assets and legacy terminal/UI/control
adapters. One pure registration builder moved to registrationdraft; actual public
capture/browser-author/package adapters replace old UI qualification helpers.
No source is copied from Authoring and no sibling is changed. Native current v6
and integration v7 preserve old version selectors/locks/readers. Scenario/journey
v5 and UWS1.12/M45 closure stay unchanged. Development v2 separates dirty-source
smoke from clean frozen qualification and binds actual current dependency inputs.
Minimal CI/build removals are prerequisites here; M95.3 completes operator/gate
migration before qualification. No removal acceptance or final adoption is claimed.

Development attempts are retained, not relabeled as frozen qualification. Failed
registration attempts 1ozuganu/opzu0771 reached an EOF handshake defect; after the
fix, 6cpjky85/beluohr3/irtwnf8j exposed the package root restriction. The caller now
runs actual package CLI commands from the containing root rather than weakening
native containment. Every display joined/auth removed. Successful registration
8q04ittm: binary40811ab168e8aa99dce6f6154422f70f4ddfe5644cfd9c1e11987cb804cab0a5,
11.604seconds; authenticated bu0p9l1q:
5cc8c917788a446f3abf1199df84184cb861ea83117b2a3660d250a262905156,
11.108seconds. Both private roots are under /var/tmp/openudon-m95-*. Actual capture
issued decisions/refusal/grant, typed inputs/TOTP, reviewed receipts and independent
native package selection passed. These occurred before final UI deletion and are
honestly development evidence; M95.4 must execute final frozen source freshly.

Task2 verification log /var/tmp/openudon-m95-task2-check.log: make check, go vet,
focused race (authoringcli/browsercapture/browserpackage/registrationdraft/
capturequalification/browsertransaction engine/browsersystem/browserintegrationeval),
unchanged W28 native v5 report verification, check-doc-memory and diff-check passed.
Retained reference seed matrix passed without caller terminal reads; print/partial
inputs make no state writes. Whole review remains0/10. Evolution checked: this
advances the already approved retirement boundary, no new version/horizon.

Task2 log SHA256 `82f8808757c1b2dacc7f6b0df0be6266483a02f870ea898fcde26adb59e9f6d1`. Final renamed neutral draft files and current report packages passed focused unit checks; documentation refinements preserve reusable core policy and literal superseded wording. check-doc-memory and diff-check passed on the final task unit.

M95.2 publication independently verified at68118634de49ef1ec48dbcd8824dd9467ea2650a. M95.3 is the sole in-progress row; neutral release/operator/expert gates are being reconciled. No browser qualification is launched by this transition.

M95.3 first release check /var/tmp/openudon-m95-release-check-wwfc9dm5 passed
Linux and macOS builds, then found preexisting M96 browserpackage syscall.Stat_t
references that prevented Windows builds. Kept the exact Unix owner/mode/hard-link
rules in a platform helper; non-Unix reviewed capture-package operations now fail
closed rather than claiming unqualified ACL equivalence. This is scoped release
compatibility work, not a new native authority. Failure/build evidence retained;
resume the remaining affected release checks without repeating passing platforms.

M95.3 self-review preserves the former four-stage offline aggregate before the
39 fresh loopback stages: current native v6 now admits offline with current
locks, while v1–v5 readers/loopback contracts remain untouched. make qualify
runs/verifies both reports. A version/omitted-stage regression and affected
report tests establish this additive contract; actual frozen execution is M95.4.

### M95.3 verified release and operator migration — 2026-10-02

Current release/CI builds only openudon and udon-runner. Expert scorecard/replay/
variants use neutral authoring commands; historical .icot/report schemas stay
unchanged. make qualify retains four current offline gates before39 fresh
loopback stages. New native v6 adds current offline coverage without changing
v1–v5 contracts. Current docs direct interaction to Kinet; old UI/control guides
are explicitly historical. Literal superseded current truth and candidate
wording remain in the knowledge journal. The drafting candidate stays deferred.

Release qualification /var/tmp/openudon-m95-release-corrected-0lg1gmma passed
all12 cross-compiles, variants validate/coverage, and provider-free scorecard
103/103 with failed0 and missing-detail/unsafe false passes0. Its initial strict
doc check found seven external/missing receiving links; corrected verified source
links and explicitly unavailable receiving-checkout context, preserving W27
lessons. No external receipt or unavailable revision is invented.

/var/tmp/openudon-m95-release-remaining-26_ii_s4 passed strict docs and a
disposable neutral draft → build → assess → exact sandbox approval → dry-run;
no executor/model/live target was invoked. Final source after the additive
offline gate: /var/tmp/openudon-m95-release-final-mz94nwnu passed affected race,
all12 builds, strict docs, memory docs and diff checks. Final docs-only current
truth refinements reran strict docs/memory/diff successfully. An attempted
make check-doc-memory was an unavailable target; the actual owner command
(cd tabilet && go run ../cmd/openudon check-doc-memory) then passed. Unix capture
package ownership/permissions/hard-link checks remain equivalent; non-Unix
package operations explicitly refuse unqualified ACL semantics. Windows CLI
builds pass, without claiming a Windows browser journey. Failed attempts remain
recorded separately and are not acceptance. Whole review remains0/10.

final-release-summary SHA256 `39a7ad1e2704e0863665432d2b6a642d3ec2a42fb95b7356f400588301ffd843`.

corrected-release-summary SHA256 `325f4539fef02e2c5026ad60bd9301ba2c684992b6dae328680623ba390e1034`.

remaining-release-summary SHA256 `37c546fe7bfce916e79fc874562e61e5d5eddacc61ca3a69390af9cc753db12b`.

### M95.4 deliberate fresh frozen selection — 2026-10-02

M95.3 independently published at cf4e25d22f62355a5b591cb082e5b5775ebe879f.
M95.4 is the sole in-progress row. Frozen producer source is that exact task
commit; status selection remains outside the clean clone. Required qualification
uses all18 frozen repositories and exact tool/dependency inputs, offline native v6 four gates,
integration v7, and native v6 loopback39 stages in three fresh passes. Both
authenticated/TOTP and registration public package journeys and actual BRP
execution are required. Estimated whole owner run25–35minutes; no live target,
model, account, installer, listener exposure or deployment. Installed private
TCP-disabled Xvfb with private temporary authentication is selected and joined.

Rationale: M95 removes producer transports and changes native qualification
inventory; prior M96/W28 binaries/reports cannot qualify this implementation.
Focused development checks and cache reuse cannot establish full retained
producer acceptance. The confirmed goal requires this fresh producer gate;
select it once, retain any failure, stop without automatic cache/fresh fallback.
Historical W28 v5 report will be independently read without relabeling or replay.
Final M20/W29 consumer adoption remains separate.

M95.4 pre-launch self-check found and corrected two transport leftovers:
reconcile could still request overwrite input; it now requires --yes and never
reads stdin. Closed draft claimed a terminal transcript without creating one;
removed that claim, preserving historical histories and Kinet-owned conversation
persistence. Explicit model/repair draft options now refuse and name retained
expert evaluation rather than silently ignore them. Added zero-read/no-write
regressions; current authoring guide no longer recommends removed iCoT.
The prepared cf4e25d clone /var/tmp/openudon-m95-qualified-wgiawsyl was NOT
executed; retain it as superseded preparation, never relabel as qualification.
Commit and refreeze these corrections before the one required fresh run.
Whole closing review remains0/10.

Pre-launch corrections passed the full authoringcli reference/regression suite, strict Mkdocs, actual check-doc-memory and diff checks. The frozen fresh gate has not yet started; preserve qualified prior release platform checks as their actual earlier source, then execute full/default/offline/native/integration on the corrected source.

Corrected prequalification source/publication independently verified at
34556808c4b4d2211a9c9b72948ca8d0ffd9f065. This supersedes the unexecuted
cf4e25d preparation for the same deliberate fresh selection; no expensive run
was launched or consumed. Freeze and execute that exact corrected source now.

M95.4 first frozen execution /var/tmp/openudon-m95-qualified-5zlscy6u at
34556808c4b4d2211a9c9b72948ca8d0ffd9f065 passed build/full make check then
stopped at current offline admission (current_stack_source_state),53.886seconds.
No browser/display/native/integration stage launched; failure summary retained.
Independent ignored-status inspection found empty .openudon-run created by old
smokematrix/releaseevidence tests. Moved only test-owned working roots to unique
complete disposable directories inside the same containing source root; native
containment and source guard remain unchanged. Preserve the failed attempt,
check the smallest affected suites/cleanliness first, commit/refreeze before
selecting the required corrected qualification. This is not automatic fallback
or reuse of a consumed operation. Whole review remains0/10.

Affected smokematrix and releaseevidence suites passed uncached in0.437/0.289seconds. Final cleanup has one owner per root, with no shared ignored parent or suppressed cleanup errors. Only test fixtures change; the current source guard is not relaxed. Required corrected selection remains fresh offline/integration/native with the same25–35minute estimate, clean exact source and all prior failure records preserved.

Cleanup-only source independently published at
fe4a55eedbbf699bd090e8dbc9ea3a3b5aae6ee4. Select the corrected required
frozen qualification from this exact commit; first assert current cleanliness
after the two formerly leaking suites. Native browsers were never launched in
the failed53.886second preflight; no completed seed is replayed.

Corrected frozen run /var/tmp/openudon-m95-qualified-ryjt_hd_ atfe4a55e passed
full make check and allfour current offline v6 gates plus independent verifier.
Integration v7 passed16 required gates/failed1/optional3 unrequested; stopped
before any display/browser/native stage. Failed boundary inherited the old UI's
Browsertools prohibition but scanned browsercapture, the M93-approved embedded
worker owner. Keep that native dispatch; apply the full forbidden list to the
pure registrationdraft seam and retain authoringengine's separate capture/
Playwright/iCoT prohibition. This corrects the gate's owner, not a policy bypass
or new worker dependency. Actual Go dependency scan and forbidden-import
regressions must pass before fresh exact-source qualification resumes. Historical
v1–v6 gate selectors/readers remain unchanged. Preserve the failed report and
passing offline context without relabeling; no browser seed has been consumed.

Focused integration regressions passed. Actual pure registrationdraft closure
contains no forbidden capture/Playwright/UI/iCoT imports; the new actual public
CLI dependency-closure regression separately excludes retired OpenUdon and
Authoring transports while preserving the approved native worker. First new
regression compile attempt used the wrong private evaluator function name;
corrected to evaluateGate before successful checks. Gate version changes only
current v7; frozen older selectors/readers and all existing prohibited imports
remain intact. Commit/refreeze then select the required current offline, full
integration17 and native39 qualification; no native browser stage has launched
in either retained failed preflight. Whole review remains0/10.

Focused boundary correction independently published at
62a8b594e74d31493f0656fa01c82264de2c265d. Select the corrected required
frozen producer run at this exact source, preserving both previous pre-browser
failures. Same25–35minute rationale/estimate; no native seed yet consumed and
no automatic cache/qualification fallback. M95.4 remains sole in-progress row.

### M95.4 native failure and bounded diagnosis — 2026-10-02

Frozen run /var/tmp/openudon-m95-qualified-4cpsy2wp at
62a8b594e74d31493f0656fa01c82264de2c265d passed full checks, offline
v6 four gates and independent verification, integration v7 17/0failed/3optional
and independent verification. Native pass1 completed all13 stages; pass2
stopped at registration_capture_handoff after its first10 stages. No third pass
or acceptance claimed. Failed native SHA256
894b1a19ac0e36bc76f4affdebd548a05604e2b07897696a9b82c66bd3510cdf;
integration SHA256656647cc43e744ddf08de2e81d8ed60a02f32506887c8735c291e8f361768a60.
Whole attempt1134.364seconds; native912.568seconds. Private display joined and
auth removed. Failure retained unchanged, no fallback or relabeled result.

Select one fresh bounded development registration_capture_handoff diagnosis
on that exact frozen source, expected1–4minutes. The development diagnostic
retains the underlying closed component failure unavailable in the aggregate
report. New private TCP-disabled display/authentication and verified teardown;
no cache, full native replay, live target, model or account. This diagnostic
cannot establish acceptance. M95.4 remains sole in-progress row; review0/10.

Bounded diagnosis /var/tmp/openudon-m95-brp-diagnosis-pq7w1tgw passed
in66.876seconds at the actual62a8b594 source; display joined/auth removed.
Development report SHA2568c2f548948bf31068225b097fb8da908d10ca748a17ab0b3649f7d14e6805b62.
It does not reproduce or explain the earlier native failure. Self-inspection
found the private-input fixture fills disabled fields before the runtime's first
Apply checkpoint, relying on a15-second browser timeout despite package/runtime
startup work. Add an explicit bounded Apply-readiness wait; retain workflow
deadlines/approval and exactly-one-POST policy. Add fixed component phase codes
without dynamic paths, values or raw errors in public output. Verify diagnostics
privacy regressions and one fresh affected development handoff before selecting
a new exact-source full acceptance run. No earlier result is relabeled.

Affected browsersystem/browserscenario/public CLI suites passed, including fixed
phase privacy regressions; diff check passed. Self-review confirmed the wait
changes only synthetic input sequencing, not executor approvals, workflow
deadlines, submit counts or native report evidence. Publish/refreeze the focused
correction, then run one bounded affected development handoff on exact source.
Full acceptance remains pending regardless of that focused result.

Focused correction source bb9863ed097a56460082640668f7cc0a121002ac
is independently published. Select one fresh development handoff from a new
frozen exact-source closure, expected1–4minutes; preserve previous failures.
No cache, consumer adoption or automatic expensive fallback.

Corrected focused development handoff passed onbb9863ed097a56460082640668f7cc0a121002ac
in81.82seconds, private display joined/auth removed. Evidence root
/var/tmp/openudon-m95-brp-diagnosis-o766j7ip; report SHA256
78b19ae3ec8763f3a5aec4049637ced2a28b9f323484a5f7373dca63f1f87f16. Diagnostic-only context explicitly
corrects the helper's inherited full-qualification label/estimate without
overwriting original records. Neither development attempt qualifies acceptance.

Select the required fresh full owner qualification on the corrected exact clean
sourcebb9863ed097a56460082640668f7cc0a121002ac in
/var/tmp/openudon-m95-qualified-4z1c8jd5: full/default, offline4+verify,
integration17+verify, native39/three fresh passes+verify and retained v5
read-only verification. Expected25–35minutes. Prior actual native failure
remains failed, not resumed or relabeled; focused readiness correction and
safe phase evidence justify this distinct exact-source candidate selection.
No cache/reuse, automatic fallback, live targets/accounts/models or deployment.
Join all private process/display owners and remove temporary auth. M95.4 sole
in-progress row; closing review0/10, M20/W29 pending.

M95.4 release gate self-check found a material preexisting CI prerequisite gap:
release.yml used nonexistent current-compatibility-lock.json and staged only
four siblings instead of the M45 build closure. Current local qualification
uses the valid v5 snapshots and is unaffected. Reconcile CI preparation to
those actual locks, all16 distinct browser/executor repositories (14 closure
plus Browserdriver/Udon), exact clean detached heads and ephemeral read-only
Git credential headers. No token persisted or printed; no release/tag or
remote CI execution claimed. This CI-only correction is separately verified
and cannot relabel the currently executingbb9863 native source.

CI local preparation verification /var/tmp/openudon-m95-ci-layout-i9vjdbww
passed: actual inline preparation body with Git clone URLs mapped only to
existing local exact-source objects staged16 distinct repos; all14 Udon
replacements resolved offline, conflicting locks refused before clone, no
credential header persisted. This is local layout/conformance, not remote CI.
Current architecture/tooling self-check also found stale three-binary release,
UI/control inventory ownership and open M93 claims plus duplicated neutral
policy. Preserve complete old excerpts in knowledge.md and consolidate current
truth; historical snapshots/readers/evidence remain unchanged. No native code
or dependency changes from these CI/document corrections.

Native full selection is still in progress. Select independent cheap legacy
package check on the same already-built exact frozen producer, using a separate
outer disposable package root (no source/binary mutation): explicit1.11
rebuild, assess, exact sandbox approval and no-executor dry-run; compare opaque
.icot bytes/modes. Expected1–5seconds. This independent package evidence does
not replace full native acceptance. Strict owner docs/memory/diff checks passed
for the separate CI/document delta at /var/tmp/openudon-m95-current-docs-87nkyebx.

Independent public legacy package check passed in1.767seconds at actual
sourcebb9863ed097a56460082640668f7cc0a121002ac, binary SHA256
dfb0e9f0764aecf7ee448d5c6d19adea6945bf79244123231c5f39524176ece3.
It retained declared1.11 through rebuild, assessment and exact sandbox approval
with dry-run executor.invoked=false; opaque historical .icot file bytes/modes
were identical. Summary /var/tmp/openudon-m95-qualified-4z1c8jd5/legacy-package-4c2prmv5/summary.json,
SHA256090f045315248b3b11ee2e56f6023d6525cf5626457a709305095ef4b116870f.
Actual CI-layout proof SHA2564728c093cca63e76105ede5952830e1f5708cad4787319398e66d60d098afd8d;
strict owner-doc proof SHA256a6e3f2b8214afa96cdfd386a7b8844fd57026acbe5d52da10f484ff989aa68ae.
Whole required native selection remains in progress; closing review not started.

### M95.4 required frozen acceptance evidence — 2026-10-02

Exact qualified application source: bb9863ed097a56460082640668f7cc0a121002ac.
Full make check, offline v6 four gates, integration v7 seventeen required gates
with0failed/3optional unrequested, native v6 all39 stages in three fresh passes,
and each independent verifier passed. Retained W28 native v5 report independently
verified without replay/relabeling. Exact original summary:
/var/tmp/openudon-m95-qualified-4z1c8jd5/qualification-summary.json,
SHA256ea80c3d40ce3c544f33845e931d50dc17cb8a318e466ba9fd9c04ef6b601e201. Native SHA256
267927b434595133afc3e562f821d8e8217b3980fde17b4a0dcd228756cd34a1; integration SHA256
59d516e790271b3a50534aa2ee2eff70759e08fee85d54c0b6696975ca2d064b; offline SHA256
3586591379f07940c8f250a5b295b039d105cebd0876b2fb3914f00342b9d9fe. CLI SHA256
dfb0e9f0764aecf7ee448d5c6d19adea6945bf79244123231c5f39524176ece3; build closure SHA2560b1ec5a0a26b707a304159eba34da0ff0b538374f9bc41e7ead484c414337bd0.
Whole1631.471seconds (27.191minutes versus25–35estimate); native1426.547seconds.
Private TCP-disabled display joined and auth removed; source closure remained
clean after all stages. Both authenticated/TOTP and registration verification
refusal/grant/typed history/review/import/native package selection passed; BRP
proved separately approved exactly-one synthetic POST. Earlier failed/superseded
contexts remain preserved with their original outcomes.

The subsequent delta changes only CI preparation, documentation/history and
status; its local16-repo/14-replacement/refusal/no-header-persistence proof and
strict docs/memory/diff checks are recorded above. It changes no runtime code,
module, fixture or lock and cannot relabelbb9863 qualification as a later commit.
Task4 verification complete; whole acceptance/review/publication/downstream
reconciliation remain M95.5.

M95.4 independently published atc03221b059d7601e12e22f195cf0a0a1ce0d3c0c.
M95.5 is sole in-progress row; whole review iteration1 is persisted STARTED
before the full closing review. Exact qualified application remainsbb9863.

### Whole review iteration1 finding M95-R1 (P2)

Reviewed the full M95 diff, boundaries, native/legacy/CLI/evaluation/CI and
current documents. RunDraft dispatches --agent before print handling, so an
explicit --yes combined with --print reaches the publisher. Incomplete print
can also enter report-file writing. Resolve by refusing --print combined with
--agent --yes or --report before any reads/effects. Positive draft, capture, package,
worker, executor and schema behavior remains unchanged. Preserve actual
qualifiedbb9863 native context, never relabel it as containing this later fix.
Verify the new refusal path and frozen full/default/integration on the final
source, plus static unchanged-native-owner/fixture/lock equivalence. No expensive
native replay is selected for an early closed-draft flag refusal; final M20/W29
retain their own exact changed binary/native adoption gates.

M95-R1 first regression run exposed an existing compatibility requirement:
read-only --agent --print must retain source validation/frontier reporting.
Narrow the refusal to --agent --print --yes and any --print --report; retain
read-only agent/print and test both complete and incomplete frontiers, plus
all conflicting modes before missing-source discovery. The failed overbroad
first check is not acceptance evidence. No native owner/fixture/lock changes.

M95-R1 corrected focused checks passed: go test ./internal/authoringcli
./cmd/openudon, including complete/incomplete agent-print frontier, inactive
browser source refusal and conflicting output zero-write checks. Runtime delta
is four early validation lines only; all protected native owners/entry/fixtures,
locks/modules match qualifiedbb9863. Commit this verified review fix, then
freeze the final source for required offline/default/integration verification.

Final frozen first regression /var/tmp/openudon-m95-review-final-l_4drobv
at8ce4dd2 failed full make check only in the old release checkout assertion:
TestBrowserScenarioWorkflowsUseLockedPrivateUdonCheckout expects the retired
shell/action preparation, not the verified current16-repository Python closure.
M95-R2 (P2 verification mismatch) discovered before review2. Preserve this failed
context; align only the release assertion to exact v5 locks, complete closure,
revision/dirty refusal and ephemeral/no-persistence credentials. Historical
browser-scenario-public workflow assertions remain unchanged. No native runtime
changes or new native selection; rerun frozen default/offline/integration after
the focused CI regression passes.

M95-R2 focused internal/eval tests passed with complete closure and ephemeral
credential assertions; original browser-scenario-public historical requirements
remain. The final23-entry replacement map explicitly separates producer native
bb9863, later early-refusal application8ce4dd2, and pending M20/W29 consumer
acceptance. Publish this test/document/status-only reconciliation; run one clean
frozen final regression selection, no native replay/fallback.

### Whole review iteration2 STARTED — 2026-10-02

Resume persisted review count2/10 after R1/R2 fixes and all affected frozen
checks passed; examine the whole M95 scope again, not just these fixes.
Final exact source94ef1d7f26120005e6b3d6be51c892032e4f9f71,
frozen root /var/tmp/openudon-m95-review-final-huhpxlon: full default make check,
offline4/integration17 and independent verifiers passed; original nativev6
and historical W28v5 independently verified, not re-executed or relabeled.
Legacy1.11 public rebuild/assessment/exact sandbox approval/dry-run and .icot
bytes/modes passed on the final exact CLI. Protected native owners/fixtures/
locks/modules equal qualifiedbb9863. Strict owner docs/memory/diff passed.
Final regression summary SHA2561593a95f736d332af5885857d5578f21e540ba056689d2b956df5a17c5139de8; CLI SHA256a1d4529c44663240ee93848205959dcf6ed22a56ac11f8e49f8f6f2ae7e4ba76; build closure SHA2564561dc76c07b08a160235787616f830ed61bd6580319bce03247837140dc718a.

Iteration2 finding M95-R3 (P2): reinspection of the complete runAgentAuthor
and pre-M95 source proves agent mode NEVER publishes deliverables, including
--yes. Iteration1's alleged agent publisher was an incorrect review statement;
only incomplete --print --report reaches an actual write. The new prohibition
of --agent --print --yes unnecessarily changed a retained read-only invocation;
the added operator sentence describing agent publication is also incorrect.
Correct both: reject only print with a report path; preserve complete/incomplete
agent print with or without --yes, all with zero package writes. Preserve the
incorrect earlier observation as corrected evidence, not proof of a real bug.
Final full/default and affected CLI checks must pass; protected browser/gate/
fixture/lock code stays unchanged, so existing nativebb9863 and integration94ef1d7
remain separate valid contexts without another native/integration replay.

M95-R3 focused CLI/authoring tests passed, preserving all four combinations
of complete/incomplete agent-print with/without --yes and refusing only report
output in print mode. Actual baseline repro at
/var/tmp/openudon-m95-print-report-baseline-eu83jaw7 confirms bb9863 --print
--report wrote a report with exit0, no package; this is the observed P2 defect.
Commit corrected minimal guard and truthful evidence; final frozen full/default
verification follows. Integration94ef1d7 and nativebb9863 retain actual scope;
no protected native/gate/dependency differences, no new native replay selected.
