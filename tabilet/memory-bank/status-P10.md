# P10 — Package v3 browser supplement

**Stage:** Kinet STG-12, Phase A. **Owner:** OpenUdon.
**State:** P10.1–4 verified; P10.5 resumed final qualification/publication handoff. Review 1/10 failed; repairs requalified, review2/publication pending.
**Source baseline:** `477bf1a53591da70c98979a7a351db4fafac0ff9` (clean at planning).
**Coordinator:** [Stage 12 contract](../../../kinet/docs/stage12.md). This
package-local milestone and status own acceptance. Planning was approved on
2026-10-09 (Kinet R65). It authorizes these planning files only.

**Planning reconciliation.** Stage 12 planning review, 2026-10-09; source priority not supplied. Review baseline and current revalidation: `477bf1a53591da70c98979a7a351db4fafac0ff9`; includes the uncommitted Stage 12 planning files. F02 (confirmed P1; runtime owner Udon:M53) requires independent evidence checks in P10.4. Evidence: `../browserdriver/src/driver.ts` executes the action sequence before extracting outputs; a closed failure code alone supplies no non-dispatch witness. This is approved review intake, not a closing-review iteration; all rows and review counters remain pending/0.

**Parallel planning provenance.** Stage 12 parallel execution review,
2026-10-09; source priority and separate review baseline not supplied. Current
revalidation `477bf1a53591da70c98979a7a351db4fafac0ff9`, including uncommitted planning files. F01 (confirmed local P2) adopts scoped workflow ownership; F02 (confirmed Lower) replaces strict ordering with readiness; F04 (confirmed local P2) requires frozen consumer checks. F03 (partially confirmed Lower; owner-selected early freeze) moves report/host contract definition into existing M51 rows and joins actual producer qualification later.
Evidence: the old Kinet goal/launch rules, owning agent rules, existing pending
dependencies and sibling consumer checks; Udon `go.mod`/`pkg/execute` import no
OpenUdon code. This approved intake changes no row state or review counter.

## Dispatch and lease boundaries

**Depends on.** Kinet:M51, UWS:C10, Browsertools:M33. All required prerequisites must have
accepted closure at exact revisions; sibling producer adoption also needs
independently verified publication. A priority position never supplies authority.

**Downstream impacts.** Kinet:M56, Kinet:W20, Kinet:M52, OpenUdon:A32.

**Write set.** The owning `openudon/` package's implementation, tests, ordinary
documentation, manifests and qualification outputs only as required by this
milestone's existing scope, plus `tabilet/memory-bank/status-P10.md` in its
assigned worktree. Excludes `AGENTS.md`, `tabilet/GOAL.md`, shared memory-bank
files, other statuses, evolution, stages, history/knowledge, the package audit
database/sidecars, coordination docs and launch input. The coordinator alone applies shared-memory and closure
changes serially; no child writes a sibling repository or user ledger.

**Contracts read.** Immutable exact prerequisite artifacts listed above, the
M51 native-owner-reviewed contract/fixtures when applicable, the assigned
package baseline and frozen shared-memory/consumer snapshots captured at
dispatch. Cross-package checks use read-only exact snapshots or approved
published module inputs, never changing sibling checkouts. Record full source,
artifact and fixture hashes in the later execution brief; contract drift pauses
affected leases for coordinator reconciliation. Existing no-workspace/no-directory
substitution requirements for ordinary published adoption remain in force.

**Parallel-safe.** yes. Eligible only under the explicit Stage 12 lease opt-in, with no dependency path or bidirectional read/write conflict against any running lease.
At most one live milestone per package. All tests use private lease ports,
disposable stores/caches/browser profiles and unique output directories.

## Dependencies and handoff

**Upstream.**

- [Kinet:M51](../../../kinet/tabilet/docs/history/status-M51.md) consumer contract.
- Accepted and published
  [UWS:C10](../../../uws/tabilet/docs/history/status-C10.md) and
  [Browsertools:M33](../../../browsertools/tabilet/memory-bank/status-M33.md).
- Builds on accepted [P09](../docs/history/status-P09.md) package v3. Today it
  refuses browser paths, the `browser-profile` kind and non-http/fnct leaves
  (`packagev3/records.go:89-104`, `execution_plan.go:248-257`).

**Downstream.**

- [Kinet:M56](../../../kinet/tabilet/memory-bank/status-M56.md)
- [Kinet:W20](../../../kinet/tabilet/memory-bank/status-W20.md)
- [Kinet:M52](../../../kinet/tabilet/memory-bank/status-M52.md)
- [OpenUdon:A32](status-A32.md)

Consumers adopt only the exact accepted and independently published revision and
SDK version.

## Tasks

| Item | State | Notes |
|---|---|---|
| P10.1 — Versioned browser supplement | `[+]` | Admit browser-profile, authentication and registration source artifacts and browser leaves in a versioned package v3 browser supplement. Browser shape tables are untrusted until reproduced through the published Browsertools:M33 verifier. Non-browser v3 package, approval and run-evidence identities stay byte-identical. |
| P10.2 — Public browser verification subset | `[+]` | Promote only the needed `browserverify` / `browsertransaction` behavior to public packages: profile transaction receipts v1–v4 and capture review evidence. Import nothing from synthesize, workflowintent, elicitor, projectwizard, udonrunner or trustedrunner. |
| P10.3 — Browser approval and Authority | `[+]` | Approval and Authority bind browser actions, origins, side effects, confirmation policy, credential-slot names and saved-session reuse permission. Changed bytes refuse. No value is carried. |
| P10.4 — Browser run evidence | `[+]` | Independently verify browser reports/run evidence, including `runevidence.BrowserConfig`, against exact action/dispatch identity. A typed error after authentication, registration or action dispatch cannot establish no effect. Preserve unknown after successful write then failed extraction, lost response, cancellation or crash; reject evidence that relabels uncertainty as safe retry or continuation. Positive non-dispatch proof is explicit and cannot transfer old authority. Evidence stays value-free; no private runtime import. Use M51 frozen native-report/interface fixtures for independent implementation beside Udon:M53. Preserve public/private isolation; M56/M52 subsequently prove actual producer interoperability. Contract drift pauses affected leases for coordinator reconciliation. |
| P10.5 — Qualify and publish | `[~]` | No Udon import; v2 and v3 history readers and existing wires unchanged; consumer builds (Kinet author/exec workers); SDK publication handoff under named authority. |

## Acceptance and verification

**Acceptance.**

- A browser package with verified shapes builds, assesses and yields exact
  browser Authority.
- Tampered profiles, shapes or approvals refuse.
- Non-browser identities are unchanged.
- The public closure contains no retained-set package.

**Verification.**

- `GOWORK=off go test ./...` and `go vet ./...`.
- `make check`.
- Public-closure and boundary guard.
- Wire and immutability fixtures.
- `git diff --check`.

## Execution policy

One coordinator owns the integrated ledgers, shared memory and serialized
integration/closure. Serial execution remains the default. Concurrent leases
require this milestone's declared safety, frozen inputs, a complete explicit
Kinet goal request and the Stage 12 agent-rule opt-in. Each lease has one
in-progress row and one assigned milestone; its persisted review count survives
resume/rebase. Commit policy comes from that later request. Source publication,
deployment and live operations retain separate named authority. Planning and
status markers grant none; audit stays disabled.

## Review

Whole-milestone review: 1/10 FAIL — 3 P2, 0 P1/P3; repairs and subsequent full review pending. Counter never resets.

## Frozen M51 producer input checkpoint — 2026-10-09

Kinet:M51.3 completed at local commit `589b844628b358167fe0a0137eded11c1028875c`. The [native-owner-reviewed contract](../../../kinet/docs/stage12-browser-contract.md), [host ABI](../../../kinet/docs/stage12-browser-host.go.txt), [public declarations](../../../kinet/docs/stage12-browser-public.go.txt) and [fixture manifest](../../../kinet/fixtures/stage12-browser-v1/manifest.json) SHA256 `0c445a5c90d2c09be561e713c364747f4ab9a3698ea8ab46e7b7b774ccf16bad` are exact producer inputs. Both native/public contract reviews pass with zero P1/P2/P3. This is input review, not this producer's implementation, publication or closing review. Kinet:M51 is now accepted locally after all five rows, review1, clean source-built checks and consumed fixture qualification; only that complete retired dependency satisfies its dispatch gate. Rows and persisted review counters remain unchanged.

Use the exact source/extraction/identity maps, per-leaf old driver protocols, complete-plan credential lease, private registration inputs, automatic TOTP versus claimed push continuations, original admitted deadline and separate bounded teardown. The supported consumer profile has one durable session binding per execution and permits other fresh named contexts. M16's immutable host-private save plan preserves v2–v11; candidate creation precedes Join and encrypted host acceptance follows Join/current-generation checks. Preserve report-v5 uncertainty and independently reproduce canonical/golden digests and positive/negative host witnesses. No current artifact claims real conformance or adoption.

**Accepted local M51 prerequisite.** [Retired M51](../../../kinet/tabilet/docs/history/status-M51.md) at observed task/review evidence `33980aafdc614424f61c32a2d79171689c7952bc` (reviewed implementation874c89e, declared coordinator closure) qualifies the unchanged frozen consumer manifest0c445a5c…. Browser qualification1+3 is consumed. This producer remains pending with its original row count/review counter; exact accepted publication and its own browser authority, where required, remain separate.

## Accepted UWS:C10 prerequisite — 2026-10-09

Exact public implementation f01a2542410c583d0ea909dadd8f17527cc0d27d resolves as v0.0.0-20261009220914-f01a2542410c, archive h1:TTjAn0++TdGHY0vahbXw3XsICQaavBytoqTSHu0K6o0= and GoMod h1:DlqFOnO9lbmYWLLIh5WicNX6NTWIuytU6mIHmxj9BVw=. Whole review1 and independently fetched closing publication 60e1baa0470cd7dca5d3166d344896bc0689abf7 qualify this input. See [public proof](../../../uws/docs/c10-publication.md). Own adoption, other prerequisites, custody and qualification remain pending; no task/review/pin change. Scoped source-publication authority is separate conversation approval; this record grants none. Audit stays disabled.

## Accepted Browsertools:M33 source prerequisite — 2026-10-10

Browsertools:M33 completes four rows and whole review2,0 remainingP1/P2/P3.
Exact source/evidence/closure1859f5e34367cb0a3fbc34ddbc2446b59ce3f2bc is
independently published; full351entry tree13dd5ca876591f55154e5723d7e16d43bde260e5
matches, final receiptSHA25612c8fb83e7f25da5548d14ed66e6bd9a1b870fe37fb7b6c94c59e5eaf1b0227b.
[Producer closure](../../../browsertools/docs/m33-closure.md) binds whole review2,
workspace repair, tests/vet/tidy, races, consumers and primary integration.
Ordinary versionv0.0.0-20261010115646-1859f5e34367 is a source-bound candidate
only: public ZIP/native sums are not yet acquired or qualified. Root has prepared
an exact bounded download scope; no directory/workspace replacement supplies
actual adoption. Owning task rows/review counters remain unchanged.

Public APIs are BuildBrowserShapeTable(context.Context,BrowserShapeOptions) and
VerifyBrowserShapeTable(context.Context,BrowserShapeOptions,binding.ShapeTable).
Options contain exact sourceID/URL/content and bounded byte/operation limits;
complete selected subtrees/numeric lexemes survive independent reproduction.
Host-neutral authorworker.Launch/registrationauthorworker.Launch and capture-only
workerhost.RunManaged use Host Launch/Join, exact closure/nonce/origin/original
admitted deadline and mandatory current-authority/durable-claim sends. Trusted
host flags/fake witnesses are not actual M56 containment or M52 custody proof.
Existing Run/CLI/wires and frozen M51 contracts remain unchanged. Native Udon
host ABI/report-v5 semantics are independently implemented from M51; never copy
capture semantics as a substitute for native execution evidence.

The human explicitly approved P10 and Udon:M53 as next execution units; they
remain independent siblings under the original goal. Before ordinary M33 adoption,
complete the separate exact download/checksum proof. P10 reconciles the stale
Stage11 pin assertion to current coreC10f01 with retained independent hcl/b099;
M33's disposable fixture patch349586c3… preserved every behavioral assertion.
Do not modify retained codec/native Udon formats or claim actual M56/M52
interoperability from synthetic fixtures. Audit stays disabled.

## Ordinary M33 prerequisite qualified — 2026-10-10

The coordinator acquired exact ordinary published Browsertools
v0.0.0-20261010115646-1859f5e34367 under actual human-approved21bd4d2f scope.
Module sum h1:xjsFcA8O/fP/D20rxonAzWKnFvxtzYVOJQmwXEb4Nbk=; GoMod sum
h1:XWKEmtkaiTWi/xQiI0RA6Y7LpwvGI8dRq5KkT57y8FI=; ZIP SHA256
c3234a3d63fa624bef8e571ca042324548a7dea1cdea2955473ad1b93621b946.
All351files match independently fetched accepted1859 source; native signed
checksum verification and actual-module offline tests/vet pass. [Proof](../../../kinet/docs/stage12-m33-ordinary-proof.md)
records two settled invocations, localreservation correction and ended grant.
This satisfies only the M33 input gate. Owning rows/review0 remain pending until
lease execution; exactcachedreuse is offline, never new download/browser authority.
P10 and M53 remain independently implementable from frozen M51 inputs.
Audit stays disabled.

## P10 execution lease — 2026-10-10

Governing Kinet `tabilet/GOAL.md` SHA256 `cad15b1c175094505d380c581f577bde7ced4109ffda40b4b39d5ffe6ae88113`; complete frozen request SHA256 `dcb1e1f89ca30750456ab791f098fe000a1c45660bba14a98cf1055457b28665`. Assigned `OpenUdon:P10` only, branch `goal/P10`, worktree `/home/peter/Workspace/openudon.goal/P10`; captured primary `/home/peter/Workspace/openudon`, integration ref `refs/heads/main`, original goal base `c0b0d7aae77a9e68630f9e1d8bfe2a9525a0ddb6`, dispatch/review base `f9f48a007d721b984bd26358b300cb5748f63754`. Task commits, parallelism3, local-rebase-ff; effective protected child grants `{}`. Coordinator owns publication/integration/shared-memory/closure. Offline exact acquired M33/C10 reuse only; audit disabled. Frozen M51 manifest remains `0c445a5c90d2c09be561e713c364747f4ab9a3698ea8ab46e7b7b774ccf16bad`. No changing sibling inputs are consumed.

## P10.1 interrupted verification — 2026-10-10

The isolated draft adopts exact ordinary C10/M33 and adds optional browser
manifest/supplement inputs, complete native shape reproduction, native source
families, per-leaf protocol/call inventory and isolated runtime plan admission.
The eleven inert native source vectors and six frozen M51 files hash-match the
exact frozen corpus; formatting and `git diff --check` pass. These structural
checks are not implementation acceptance. The Go behavioral checks have not run.

Offline tidy first failed expanding cached libc under the private `/dev/shm`
quota. Root preserved and deduplicated immutable input bytes, then assigned
private `GOCACHE=/tmp/kinet-stage12-parallel-build-20261010/openudon/gocache` and
`TMPDIR=GOTMPDIR=/tmp/kinet-stage12-parallel-build-20261010/openudon/tmp`.
Updated trusted context SHA256 is
`402b45c17a4cbacd53c9e2803d844a1f1dfa281577c7fe2aa5be5ac5f8093054`;
assignment/approval/ref/base remain unchanged. Focused
`go test -mod=mod ./packagev3 ./browsercontract` also failed writing standard
library compiler/vet/importcfg outputs with `disk quota exceeded` under `/tmp`,
before compiling P10 or running tests. No failed compile is qualification.

Tidy additionally awaits already-cached exact modernc opt0.1.4, sortutil1.2.1,
strutil1.2.1, token1.1.0, gc/v2 2.6.5 and gc/v3 3.1.2 artifacts.
Root is resolving headroom/cached inputs. There is no new network/browser
request or expanded authority. Draft source remains uncommitted in this lease;
no task is complete and formal review stays0/10. Resume this same row after
usable isolated compiler capacity is supplied; preserve all source/fixture/
module/provenance paths. Coordinator publication, consumer builds, complete
checks, shared-memory proposals, review and closure remain unperformed gates.

Resumed the same P10.1 lease after coordinator-proved isolated compiler headroom. Prior failed compiles remain failed evidence; no test/run identity is relabeled. No authority or baseline changes.

## P10.1 verified row

Focused `go test -mod=mod ./packagev3 ./browsercontract` and `go vet ./packagev3 ./browsercontract` pass using the assigned offline Go1.26.6 environment and repaired private `/tmp` compiler resources. All11 native source-family vectors, whole-selected canonical/source hashes, frozen M51 canonical/config goldens, immutable snapshots, rehashed shape refusal, runtime-admission constraints and prior non-browser goldens pass. Ordinary C10/M33 sums equal their accepted input proofs; retained codec b099 is independently asserted. `git diff --check` passes. New fields omit on non-browser records; all old fixtures remain unchanged. Task implementation advances the approved direction; no shared memory or evolution is edited by this lease. Formal review stays0/10.

## P10.2 verified row

Public `browsertransaction` and `browserverify` own the unchanged pure transaction/report implementation; retained CLI packages are thin aliases. All original transaction v1-v4/report/private-file regressions remain active at public owner paths, with the Make selector updated. New explicit-byte reports and capture receipt checks require independently retained exact start/receipt/transaction identities and complete declared source/review bytes, retaining only metadata and no capture/run authority. Focused public/retained engine/candidate/package/synthesis/packagev3 tests and vet pass; public subset closure151 packages contains no OpenUdon internal/retained or private Udon import. `git diff --check` passes. No old wire/schema/fixture was rewritten. Formal review stays0/10.

Coordinator preparation comparison of immutable first2-task source52b057d/tree and manifestSHA256fcac7ab668594cc5696183fbef41a7586671d08a833113c4a9b865bec7ac042a passes0P1/P2/P3:47postimages,16frozenM51publictype declarations and12unchanged nonbrowsergoldens match. This is preparation only; formal counter remains0/10.

## P10.3 verified row

Additive sandbox-only browser approval v1 finalizes exact independent human receipt bytes before Config/Authority hashing. Public Config/Authority strict bounded decoders preserve frozen declarations/canonical identity, complete calls/origin/credential unions, protocol pairs, one durable binding and at most one save permission. Derivation independently reproduces verified package/plan/calls under actual consumer runtime admission and explicit host time; changed actions/effects/policy/origins/worker/driver/inputs/deadline/reuse/credentials or receipt bytes refuse. Existing approval v1/v2 APIs refuse all browser leaves, including mixed sequences. Focused packagev3/browsercontract/approval/authority/browserverify tests and vet pass; frozen config golden and privacy/refusal checks pass; old nonbrowser goldens remain unchanged. `git diff --check` passes. Config/receipt validation remains metadata only, with actual human confirmation/grants/current claims/custody owned by Kinet. Formal review0/10.

Coordinator scope reconciliation during P10.4: exact frozen M51 host ABI
`BrowserExecutionBinding.Inventory []BrowserLeaf`, closed `BrowserLeaf.Purpose`
action/authentication/registration, and public complete ordered Calls select a
browser-only executable inventory. Mixed HTTP/fnct/browser artifacts may remain
review packages, but SDK execution/Authority must refuse them before runtime
admission or host access. Existing pure non-browser profiles/goldens stay
unchanged. This corrects unsupported-profile enforcement within approved P10
scope, without new ABI/kind/edge or new milestone. Udon:M53 independently
implements that same selected profile; this lease reads no changing checkout.
Formal review remains0/10.

Preparation review of immutable task1–3 sourceaf8e945 identified one P2:
current browser Authority compared session expiry to explicit Now but omitted
creation/current-time comparison. Exact frozen session contract requires
creation not in the future. P10.4 fixes creation comparison in the owning
current-time Authority path and verifies before/at-creation boundaries; historical
Config validation remains clock-free. Review preparation manifestSHA256
c41e1fb607dd178e2c8f671a5acfdedc5a49008696e4fad84fd195227d79a53f.
Formal count stays0/10; this is scoped pre-acceptance implementation repair.

The task1–3 preparation report consolidates1P2/0P1/P3 at
SHA256dce7648c5afb3e7985e9035be2c14c72d508422a8c4f0fce0907d16baa7196de,
with56manifest postimages verified. The future-created session finding is fixed:
current-time derivation refuses Now before CreatedAt, accepts the exact creation
boundary, and clock-free metadata validation preserves canonical UTC timestamps.
Affected focused packagev3/approval/authority/report suites pass.

## P10.4 verified row

Public explicit-byte browser report/run-evidence verification independently passes all34 exact frozen M51 metadata cases, plus current identity/claim/question/protocol/deadline/positive-join/safe-successor/privacy regressions. Initial and effect-capable continuation sends require a immediately preceding exact claim; typed errors/extraction/loss/cancellation/crash remain unknown and stop later sends, retry and fallback. Complete non-dispatch evidence supports only a distinct newly confirmed successor; old authority never transfers. Known completed leaves survive later checkpoint failure. Candidate/save ordering, one-use permission/current generation and join/release are checked. New browser evidence v1 is closed/value-free; legacy report-v5/RunEvidence/BrowserConfig/schema/fixture bytes remain unchanged. Mixed browser+HTTP/nonsequence-fnct plans now refuse before native admission; standalone nonbrowser fixtures and pure-browser positives pass. Opaque native credential revision IDs and registration fresh-access metadata match M51 without inventing private input values. Future-created session P2 is repaired and focused before/at-creation regressions pass. Focused tests/vet and `git diff --check` pass; no private runtime import or actual browser/process/conformance claim. Formal0/10.

## P10.5 preparation repairs and qualification progress

Second preparation review of immutable f79fa0a source, manifestSHA256
724c613ca350dc7cf8f11a7e6b6b4f139795837a1ce700ceb61e02859983750c,
consolidates1P1/5P2 at reportSHA256
08d52a1fd846569c917324d510158d4fe54d4ae4dd21f7e14f0c6431004dc614.
All are repaired in this pending qualification checkpoint: per-leaf terminal
response/uncertainty latch; pre-send exact runtime confirmation; each started
leaf before the original deadline; positive complete symbolic credential lease;
irreversible denial/checkpoint stop; supplied answer/recheck/private checkpoint
question identity. Frozen omitted question fields retain implicit current trusted
identity. Native coordinator optional-access reconciliation permits exact empty
access identity when no durable reuse/save is requested, with positive fresh
no-session auth/action/registration controls. All34 frozen cases and new adverse/
positive focused regressions pass. No old report/wire/private snapshot changes.
Formal review remains0/10; preparation checks are not closing acceptance.

Initial full tests exposed the expected additive API surface guard and tidy's
unused-codec removal. P09 original surface/goldens remain frozen; P10 adds a
versioned current surface manifest/guard. A meaningful test imports the exact
retained ordinary b099 codec and render/verifies an existing immutable P09 YAML
fixture, preserving real test dependency/pin without production HCL changes.
Final full tests and vet, tidy and modverify pass. Frozen Kinet author/exec workers
from09834b6 copy/hash verification and unpublished local SDK compatibility builds
pass; no actual adoption/native/browser conformance is claimed. Public closure
373 packages/owner91 modules has no internal/retained/private import or directory/
workspace replacement;20 legacy wire/golden files remain byte-identical.

Required make check subsequently encountered isolated `/tmp` disk quota during
linking/SQLite fixtures; focused races encountered the same quota during stdlib
compile. Those failed invocations are not qualified or waived. Their logs and all
source/module/consumer/proof outputs are preserved. Coordinator headroom repair
is requested before rerunning only the affected gates. Publication, authoritative
whole review and closure remain pending. Evidence progress artifact is
`/tmp/kinet-stage12-parallel-build-20261010/openudon/tmp/p10-proof/qualification-progress.json`
SHA2565ee58679fada77d6afc4b35d0752cb46e1b420ee2f5ec68a6663d6145f5f24e5.

Coordinator isolated disk cache repair contextSHA256af3b593ca2d166877a37156926b76b0423bc458287c4f1a983351606116e7d18 and receiptSHA256b682484546fbe5baec1d63fb7e366549b592c0637d7b0c8eb3a643a6753913cd preserve source/module/proof/grant identity. New private cache/temp roots under `/home/peter/Workspace/openudon.goal/verification-P10-4m_d2i4b/` have9993hash-verified cache files/960583768bytes and positive32MiB fsynced probe. Resume only the affected makecheck/race gates sequentially; all earlier failures remain failed evidence.

## P10.5 successful local qualification checkpoint

Under repaired isolated disk cache, required make check and focused seven-package
races both pass. All final full tests/vet/tidy/modverify, public boundary/closure,
wire/golden immutability, additive public API and retained codec checks, and frozen
Kinet author/exec compatibility builds pass. Failed prior runs stay failed evidence;
[qualification](../../docs/p10-qualification.md) and its machine-readable log/hash
manifest bind the results and exact fixture wiring. Captured primary main and this
lease's main reference still equal dispatch basef9f48a007d721b984bd26358b300cb5748f63754;
no rebase or history rewrite is needed. This local qualification/source checkpoint
leaves P10.5 in progress for root-owned source publication and independent proof.
Formal review stays0/10 until the candidate is frozen and the coordinator is
notified before reviewer dispatch; no whole acceptance or closure is claimed.

## Whole-milestone review 1 — STARTED

Started2026-10-10 after all required local checks pass. Full reviewed source
checkpointde2e405e488e3c90e2cc1c29d99b9901cce90f07, tree97e07fc4fe43f078ad55f43f8fb2dac4db4b9061,
diff baselinef9f48a007d721b984bd26358b300cb5748f63754; captured primary
`refs/heads/main` is unchanged and clean. Original goal base remains
c0b0d7aae77a9e68630f9e1d8bfe2a9525a0ddb6. The complete immutable66postimage
handoff manifestSHA256475b521a254bc6dfb882ef5995e0cab41317bda0ca803f93727cd063c1eb4924
and actual source diffSHA2563b5097e157d4ff11227139145867696e3b0ffef4cbdf19ed971a055e9d634121
bind the candidate and all successful evidence. Coordinator notified before
reviewer dispatch; read-only preparation reviews remain separate historical
provenance, not earlier closing iterations. This iteration reviews the full
milestone and any joined read-only coordinator reviews. Required P10.5 source
publication/independent Git proof remain unperformed; keep this iteration pending
until those actual actions/evidence are reviewed. No whole acceptance is claimed.

### Review1 own findings persisted before fixes

Full own review checks all five rows against the66postimage immutable candidate,
complete approved scope and frozen M51 contracts. Two reproduced P2 findings:
P10-R1-01: Config.LaunchNonce has no independently projected launch nonce/closure
comparison in the out-of-band public witness; unchanged launch/join/trace accepts
a changed expected/host Config nonce. This awaits coordinator contract
reconciliation before an ABI-adjacent fix. P10-R1-02: new run-evidence serializer
closed-shape decoding alone emits arbitrary private RawMessage Effects instead
of requiring valid closed Config/evidence metadata. Fix requires semantic
metadata validation before serialization. Both reproductions use a private test
overlay without source mutation; evidence at
`/tmp/kinet-stage12-parallel-build-20261010/openudon/tmp/p10-proof/review1-own.json`.
No other P1/P2/P3 found in this own pass; coordinator fanout/publication evidence
remain pending. Root notified to hold candidate publication. Findings/counter
are persisted before fixes; iteration1 does not pass or reset.

Review1 qualification correction: recursive audit found a nested Moby replacement
metadata error in the earlier171-module frozen exec listing despite its actual
consumer build passing. The coordinator supplied the exact already-cached four
Moby artifacts to the private cache (receiptSHA25601917b09651b94461be13209757a3be0983adf90c42bbca114e606b2d46958a0).
Refreshed exec listing has171modules/zero recursive metadata errors, command
receiptSHA256ec98d54b312b17dee712cf690466636058ebe95906a313c2e613689971f36b61.
Old incomplete listing/proof stay unchanged; new files are
`consumer-exec-modules-fixed.json`, `consumer-proof-fixed.json` and the exact
command receipt under the private p10-proof directory. No source/runtime/pin
change, network retrieval or new authority. Final qualification must use the
corrected listing/proof while retaining the earlier evidence's limitation.

Joined coordinator review1 confirms additional P2 P10-R1-03: denied in-flight
authentication can still consume a matching initial response:success after
Interact:deny because stopped is not a per-leaf terminal latch. Reproduction
retains claimed-push initial authenticate claim/send/question, denies, omits
recheck/continuation sends and supplies original initial success response plus
success report. Repair must latch denied unknown in-flight leaf as terminal
uncertainty, refuse later response upgrade, and preserve already completed
leaf proof before a later checkpoint failure. Coordinator full review continues;
collect remaining findings before applying this same iteration1 repair set.
Publication remains held; counter never resets.

### Review1 complete — FAIL,3P2/0P1/P3

Joined coordinator full-scope review completes at reportSHA256
98b85e62bff439a7aa4034d8312949c712df664b372b3d105a6b595127c1c108
(`/dev/shm/kinet-stage12-leases-f8qot0xc/resources/coordinator/p10-formal-review1-de2e405e488e/review-result.json`).
All63live postimages/3moved-deleted paths/25proof artifacts and20unchanged
goldens verified. Exactly P10-R1-01 launch nonce/closure binding, P10-R1-02
serializer semantic validation and P10-R1-03 denied in-flight success upgrade
remain open P2s. The separate nested Moby proof limitation is repaired, with
zero recursive errors and original evidence retained. Findings persisted before
repairs. Coordinator confirms fixes stay inside approved P10 and preserve all16
frozen public wire declarations/native ABI. Actual launch witness fields must
project observed native containment, never copy expected Config as proof.
Native session Acquire precedes Launch; retain same-binding expired→fresh before
launch, optional fresh-only profiles may omit Acquire, and reuse/save must bind
positive exact observed durable access. No new mandatory fresh lease.
Publication remains held; after repairs and verification start full review2
without resetting this counter or treating local rows as acceptance.

Review1 authorized repairs implemented. Actual value-free launch witness nonce/
worker/driver closure now match Config independently. Durable reuse/save require
positive observed exact access identity/generation/creation/expiry; observed
permissions may narrow approved rights, including same-binding missing/expired
to fresh before Launch. Narrowed save forbids candidate/save. Fresh-only work
requires no durable lease, and extra or post-launch acquisition refuses. The new
serializer/verifier share semantic closed metadata validation before emitting
RawMessage fields. Denial latches the in-flight unknown leaf terminal while
already completed checkpoint outcomes stay known. All34 frozen cases and new
nonce/worker/driver, access/fallback/narrowing/extra-access, denial and serializer
positive/adverse focused controls pass. Frozen source bytes and16public wire
declarations remain unchanged; current additive API guard reflects out-of-band
witness fields only. Required requalification and full review2 remain pending.

Review1 requalification output interruption: the old `/tmp` proof directory
hit Errno122 disk quota when persisting the first full-test command receipt.
Its redirected log is empty and process exit receipt unavailable, so that
invocation is unqualified/unknown, never a pass. New disk compiler isolation
is unchanged and healthy. Preserve those earlier paths and put only new proof/
log/receipt outputs under assigned isolated disk verification root
`/home/peter/Workspace/openudon.goal/verification-P10-4m_d2i4b/proof-review1/`.
No source/module/proof deletion or external action; repeat the incomplete
qualification there before review2.

## Review1 repairs fully requalified

All required corrected full tests, vet, tidy, modverify, makecheck and focused
races pass with complete disk command/exit receipts. Fresh frozen72source-file
author/exec fixtures compile successfully, retain exact original inputs/pins
with only explicit local SDK compatibility overlays, and have zero recursive
metadata errors (author85/exec171). Owner closure remains373public packages/
91modules with no retained/private import or directory/workspace replacement;
20legacy wires/goldens and all16frozen public declarations are unchanged.
Private disk proof root and effective p2-to-p1 concurrency transition are bound
in [qualification](../../docs/p10-qualification.json); original receipts and
unqualified/failed old attempts stay preserved. Captured OpenUdon primary
refs/heads/main is clean and stillf9f48a007d721b984bd26358b300cb5748f63754.
All three review1 P2 repairs and coordinator access reconciliation are ready
for next full review. No publication, whole acceptance or closure is claimed;
P10.5 stays in progress and counter1 never resets.
