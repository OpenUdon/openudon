# M99 — Stage 11 public decoder and review contract remediation

**Stage:** STG-11 post-acceptance remediation. **Owner:** OpenUdon.
**State:** Authorized serial implementation, 2026-10-08; M99.1/M99.2 complete, M99.3 in progress.
**Authority:** The complete review reconciliation was approved, followed by the confirmed serial GOAL request on 2026-10-08: Udon:M49 → UWS:M09 → APItools:M83 → OpenUdon:M99 → Kinet:M49, COMMIT_POLICY: task and EXTERNAL_MUTATIONS: none. This authorizes scoped implementation after accepted/published prerequisites; source publication still requires this owner's separate fresh named grant. The Udon/UWS publication exceptions do not extend to this owner or live operations.
**Review source:** stage11-siblings-review.md — Stage 11 code review — sibling packages; OpenUdon section.
**Review baseline/range:** `7cd7fbb837fb87e1ca4abea2a362790b0f434188` → `f7aa5d874bc474888bac1b43d4112c1faf29d499`.
**Revalidation HEAD:** `f7aa5d874bc474888bac1b43d4112c1faf29d499`; clean worktree, no relevant uncommitted code in the evidence. Approved planning changes are not implementation evidence.
**Lineage:** [M98](../docs/history/status-M98.md) and [P09](../docs/history/status-P09.md); [A31](../docs/history/status-A31.md) trust/compatibility boundary remains frozen. Existing acceptance, review counters, statuses and frozen evidence stay preserved.
**Coordinator:** [Stage 11](../../../kinet/docs/stage11.md#post-acceptance-remediation--2026-10-08); package-local specification and status own acceptance.

## Dependencies and handoff

Accepted M98/P09/A31 contracts, accepted and independently published UWS:M09 root/codec and APItools:M83 before exact SDK adoption/qualification. Serial scheduling follows APItools:M83. No private Udon import is added; runtime proof remains supplied through public host adapters.

**Exact successor acceptance/publication/build identities:** unset; record full independently observed revisions and hashes during the later execution. Never substitute local HEAD, directory replacements or prior consumed publication authority.

**Downstream:** Kinet:M49 public author and separate private exec consumers, exact package/source verification, corrected worker closures and successor bundle. Browser/legacy/frozen consumers keep their existing independently fetchable pins.

## Scope and acceptance

Make strict JSON key handling match typed Go record semantics without over-rejecting free-form data, and retain unsupported symbolic security metadata as indeterminate for review-only packaging. Preserve exact valid wire/schema bytes, public/private import boundaries and no-public-v2-build/synthesis contract.

Struct-typed records reject duplicate aliases under Go Unicode simple field folding, including long-s/K cases, at nested typed paths. Free-form maps permit case-distinct keys while still rejecting exact duplicate keys and retaining number, depth, node, Unicode and trailing-value bounds. Build/Assess/Verify preserve unsupported scheme names as original indeterminate review metadata; broker approval/runtime binding stays refused unless its independently supported/addressable policy is proved. Existing valid historical wire encodings, digest order, source verification, public API manifests and read-only v2/v3 history remain compatible. Ordinary public/private consumers and whole milestone review qualify exact published SDK closure.

## Tasks

| Item | State | Notes |
|---|---|---|
| M99.1 — Apply typed Unicode alias checks without rejecting free-form maps | `[+]` | Make duplicate checking aware of destination struct/map types at nested paths, matching encoding/json Unicode simple field folding for records. Permit id/ID in free-form JSON while rejecting exact duplicates everywhere; preserve bounded decoding, numeric lexemes and trailing/unknown field policy. Test run-evidence v2/v3, package data and protected broker/package boundaries. Sources P3.1/P3.2. |
| M99.2 — Preserve indeterminate symbolic security in review-only packages | `[+]` | Build/Assess/Verify retain original unsupported scheme symbols without silently renaming or treating unknown security as anonymous. Keep credential/addressability/broker approval requirements closed, refusing execution when unsupported names cannot be independently bound. Preserve valid source/shape security semantics and wire/schema contracts. Source P3.3. |
| M99.3 — Qualify the corrected public SDK and source handoff | `[~]` | Explicitly select both exact accepted/published UWS:M09 root and codec with APItools:M83 (standalone codec retains C09), reconcile expression-contract inference to declared step/operation outputs and the active $outputs owner (no operation fallback for absent step outputs), then run public import/API manifests, unchanged valid wire/digest/schema, forged-source verification and ordinary public/private consumer checks. Keep v2 synthesis/build APIs private and A31 browser closure unchanged. Persist pre-publication/closing review counts, publish only under fresh named authority, and hand off exact SDK sources/sums to Kinet:M49. |

## Active finding provenance

Only approved active findings are recorded here. Source priorities and local severity are separate; task references identify one owner for each required outcome. Unsupported and unscheduled findings remain in the conversational handoff; optional directions belong only in milestone.md.

| Source finding | Source priority | Local severity | Disposition | Current repository evidence | Task owner |
|---|---|---|---|---|---|
| P3.1 | P3 | P2 | confirmed | wire/json.go:95 uses strings.ToLower; offline scope/ſcope probe passes but encoding/json selects the alias | M99.1 |
| P3.2 | P3 | P2 | confirmed | packagev3/build.go and wire.DecodeStrictNumbers reject valid id/ID free-form map; offline probe | M99.1 |
| P3.3 | P3 | P2 | confirmed | packagev3/security.go:44 hard-refuses unsupported symbols even for Build/Assess/Verify review-only paths | M99.2 |

## Verification and compatibility

go test ./...; go vet ./...; owner quality/API-surface/public trust-import/wire/schema identity checks; affected wire/packagev3/runevidence races; Unicode struct alias/exact duplicate/map-key/number/depth fixtures; review-only unsupported security with execution refusal; forged/stale source/shape/assessment/authority regressions; exact accepted UWS/APItools standalone module/public/private consumer reproduction with GOWORK=off GOPROXY=off; git diff --check.

Use retained Go 1.26.6 and exact ordinary modules without ambient workspace substitution. Default verification is offline, credential-free and model-free. No live user ledger, host, provider, account, mail, Cloudflare, registration or consumer adoption operation is included.

Public schemas/wires, published grammar/version bytes, accepted historical qualification and independently retained browser/media/legacy/frozen-consumer pins stay preserved. Corrected derived metadata and new worker/package identities require fresh consumer assessment and explicit authority; historical approvals are never upgraded automatically. Source publication requires a new separately named request and independent resolution before downstream adoption. Planning rows may remain pending on this external prerequisite; none is started here.

## Closing review

**Review iterations:** 1/10.
**Review state:** iteration 1 found blocking P2 R99-1; narrow fix and fresh exact-source qualification required before whole review 2.
**Findings/fixes:** R99-1 P2 is independently confirmed and persisted before fixes; no other wire/security blockers. Expression/source/proof reviewer found no other P1/P2; current Build documentation is reconciled from M82 to M83.
**Execution owner:** sole serial OpenUdon M99 implementation owner under the confirmed GOAL; M99.3 is in progress, and the parent makes no ledger writes during this handoff.
**Commit policy:** The user separately authorized a planning commit on 2026-10-08 with “git commit and then report the index refresh issue in ~/skill-index.md”. This authorizes one commit of the approved planning changes in this owner repository; implementation, publication and deployment remain outside this request. Future task commits follow the separately invoked GOAL/request policy.
**Closure:** persist each started review iteration before reviewing; resume an interrupted pass at the same number. No open P1/P2 may remain at acceptance. Required verification, exact downstream reconciliation and owner-specific consolidation/retirement follow implementation; never reopen completed Stage 11 history.

## Accepted UWS:M09 prerequisite — 2026-10-08

All five UWS rows and whole review 7/10 pass; accepted root/codec runtime is
b099f6803277ae94c7e9f1da0904a0140b278f20, independently resolved as
v0.0.0-20261008043726-b099f6803277. Published evidence
51a74545b016b8ab75e30d454338c52f7945a836 is independently observed. Root sum is
h1:4xy+/HBNh1CSJDO+qOzWdV/0zC/yFCKAz2kOBWufA7g=; codec sum is
h1:OJsmDK/RcFpyMAMGy84DjcX6kNalYH/E0Q/C3XwD5Uo=. GoMod sums are
h1:DlqFOnO9lbmYWLLIh5WicNX6NTWIuytU6mIHmxj9BVw= (root) and
h1:0cR/xLzEP8vJ9FAUhsLbaKVkU7UarXPc51nJjEaMP5Q= (codec).
[Ordinary publication proof](../../../uws/docs/m09-publication.md) records
full provenance, exact 360/21 files and complete 52/50/52 selected closures.
These counts describe UWS proofs, not a prescribed downstream closure size.
Normal M09 retirement closure 989e3f2c88cac5c0f5a2911dfe04c36a61e43126
is independently observed on the approved origin/main before adoption. Runtime
b099f6803277ae94c7e9f1da0904a0140b278f20 is its ancestor; UWS worktree is clean.

Corrected binding proofs retain containing constraints, original dialect and
indeterminate outcomes. Flow and strict portability use actual root goto,
trigger/dependency iteration contexts and separate step/operation output owners.
Untrusted HCL/shape parsing refuses depth above 100 before recursive decoding;
HCL views require deterministic canonical bytes plus independent value/numeric
proof. Non-NFC presentation remains fail-closed. Ordinary validation/execution,
wire/schema/digest algorithms, published versions/corpora and frozen consumer
pins stay preserved. Changed diagnostics/derived assessments/package or worker
identities require fresh assessments/confirmations/grants. This prerequisite
note changes no implementation row to complete and supplies no live authority.

## M09 output-path and owner reconciliation — M99.3 qualification

Read-only candidate-bound checks at retained OpenUdon
2b4382011fe0f98b52a8f0c892bd64da78bfa82f isolate the original
TestSourceBackedChainedInputTypesRemainExact failure to accepted M09, independently
of M83: retained M82 plus exact M09 reproduces the same indeterminate path.
The original required integer response minimum 9007199254740993 exceeds every
bounded fallback witness. Native validation admits a value, but failed samples
prove neither reachability nor absence; preserve that original source as truthful
indeterminate evidence and broker refusal under the accepted public contract.

A separate P2 SDK inference defect is established at
packagev3/expression_contracts.go: output falls back from absent Step.outputs to
Operation.outputs, and current $outputs always resolves through the operation.
Exact accepted core ExecuteStep finalizes only the step's declared outputs and
copies only operation Result; the shared evaluator reads successful step records.
Bounded probes show absent step output and self-reference cases can Build/Verify
compatible and derive metadata authority while native execution refuses unresolved
$steps.fetch.outputs.n. A valid explicit step {a: '$response.body.n',
n: '$outputs.a'} executes with exact json.Number yet the old SDK reports
incompatible. This is one owner/context inference defect, not an APItools or UWS
runtime regression. The flow.output_unreferenced note truthfully describes unused
operation output; it is advisory, not runtime proof.

Existing pending M99.3 exact-SDK qualification owns the narrow adapter repair:
retain actual output owner/context and already-resolved output timing, reject
missing/forward/current/cyclic outputs, and prove the valid correctly owned step
chain (including a preceding-output $outputs reference). Keep the original minimum-only path
indeterminate. Add an original-dialect response enum
[9007199254740993, 9007199254740995] while retaining integer/minimum as a positive
finite-witness chain; prove native execution, exact type/number propagation,
forward-step refusal, rounded-number and contradictory-parent controls.
OpenAPI 3.1 const is a separate supported case; 3.0 const stays unproved.
Build/Verify success alone cannot establish native runtime correctness.

These amendments reconcile the existing qualification to accepted UWS/P09
contracts and preserve the three task units, wires/digests, ownership boundary and
frozen P09/M09 history. No grammar/runtime feature or minimum-only compatibility
restoration is authorized. That broader restoration would require fresh UWS-owned
planning. No row starts, counter remains 0/10, and APItools:M83 accepted/published
source is still required before M99 implementation. Exact failed candidate-suite
logs and bounded diagnostic probes remain retained as qualification evidence.

**Governing reconciliation authority:** This is the confirmed GOAL's required
accepted-prerequisite/downstream reconciliation, not a new product direction or
external review launch. Its Milestone Loop step 4 permits discovered in-scope work
in an existing pending owner. M99.3 already owns exact SDK/source/assessment/
authority verification against the changed prerequisites; the narrow inference
repair is necessary to that acceptance. The parent revalidated the analyst's
claims against code and bounded native execution before amending the untouched
pending row. No new milestone, runtime semantics or live authority is introduced.

## Accepted APItools:M83 prerequisite — 2026-10-08

APItools:M83 is accepted/retired after all seven rows and whole review5. Exact
published runtime/source f2c5693ec39ad6981693d2ad126ff26e3fdd564c independently
resolves as v0.0.0-20261008061439-f2c5693ec39a. Archive sum is
h1:+UICyuitKDE3g5rnlCA6Sk+QLIEqqVn8MN0PtOX+NKk=; GoMod sum is
h1:WmUXlfBBoaI6vtv/wzeZfyO8q/pZMhnHiBckkAthYfk=. Published evidence
 e891dfd014821c0ef8b9a29a90e19f6252a48392 is independently observed on main.
[Ordinary proof](../../../apitools/docs/m83-ordinary-proof.json), SHA-256
5241e29354e50da3763fc0b30f6cad9731abe7d91338de61d63e1fd7ae1c31b1, records
all614source files and complete owner77/39/414/16271 and public-consumer
64/33/353/15444 selected/compiled/package/file proofs; all downloaded scopes
pass full tests/races/vet/build/modverify. These are upstream actual counts,
not prescribed downstream closure sizes. Normal M83 retirement closure
de3f16acbf12c7b632ee0ed9be02efaaf2c2be4b is independently observed on approved
APItools origin/main before adoption. Runtime f2c5693ec39ad6981693d2ad126ff26e3fdd564c
is its ancestor; the owner worktree is clean.

Corrected direction/default/member/request/response, canonical response-key
merge and conservative format/serialization metadata require fresh source-backed
assessment/package/worker identities. Keep unknown/indeterminate outcomes honest,
original M82 corpus/history and independent browser/media/PhaseA/legacy pins.
The old candidate OpenUdon suite remains failed evidence: pending M99.3 owns the
narrow SDK declared/current-output owner/timing repair and correctly owned exact
finite-witness/native regressions, without changing accepted UWS semantics or
pretending minimum-only reachability proved. Existing pending rows retain their
source verification, authority and actual-runtime checks. No live authority or
other-owner source publication is supplied by this prerequisite.

## M99.1 verification — 2026-10-08

Destination-aware strict JSON scanning preserves Go exact-name-first/simple-fold
record matching and embedded-field dominance without folding free-form map or
interface keys. Nested pointer/slice/map record aliases, long-s/Kelvin forms,
escaped/exact duplicates, case-distinct large numeric data, retained evidence
v2/v3 and original published fixtures pass. Standalone offline Go1.26.6
`go test ./wire ./packagev3 ./runevidence` and affected `-race` suite pass;
`git diff --check` passes. Includes the parent-authorized exact prerequisite
reconciliation present at handoff; no unrelated worktree change was absorbed.

## M99.2 verification — 2026-10-08

Build/Assess/Verify retain slash, space and Unicode security symbols in exact
source, native shape and execution-plan review metadata. Unaddressable slots
produce binding.security_symbol_unaddressable indeterminate findings; unchanged
handoff credentials include only addressable symbols. OR alternatives are
preserved, unknown security stays unknown and privacy scanning stays closed.
Exact unsupported, missing and renamed credential candidates all refuse broker
authority. Full packagev3 tests, race and vet pass offline under Go1.26.6, with
existing API/wire/golden/schema/source/authority regressions unchanged.

## M99.3 local implementation and qualification — 2026-10-08

Both exact ordinary UWS modules and APItools M83 are explicitly selected.
The SDK adapter now uses declared step outputs without operation fallbacks and
projects only lexically preceding outputs of the active owner. Native public
execution, Build/Verify/Broker matrices cover missing step outputs, operation
borrowing, current/self/forward/cyclic references, valid preceding output chains
and future steps. Owned large-minimum-only sources remain indeterminate; a
finite enum retains exact json.Number 9007199254740993 through both native steps.
Original-dialect 3.0 const stays unproved; 3.1 const, rounded/boolean/contradictory
parent and unknown-parent controls pass. Original failed candidate evidence is
retained at /home/peter/.cache/openudon-m99-proof/baseline-original-preserved-regression.log
(SHA-256 16f93bc25c6ba680336c18c18d9fb995fa6e554d773d323deecc5604b89eca70),
with its original verification/source/owner probes. This does not relabel that
failed P09 regression as a pass. Original M82 fixtures stay frozen; separate
exact M83 eight-family/13-operation source/table reproduction passes.
The current-owner UWS/codec dependency assertion is reconciled; historical E12
and browser locks remain unchanged. Local implementation is ready for exact
source freeze/full closure/consumer proof and persisted whole review. M99.3
stays in progress until its separately granted ordinary publication gate passes.

## Whole review 1 — started 2026-10-08

Read persisted 0/10 and record 1/10 before read-only fan-out. Review the full
M99 implementation range from 2b4382011fe0f98b52a8f0c892bd64da78bfa82f through
qualified runtime 6cf6d9ebc5f38bb7476bccfb1e0c780279ef5937, including the pending
local qualification/proposal records. Owner/public/private selected artifact
closures include every unused module, exact source/ZIP/file reproduction,
native correctness/refusal matrices, source/schema/API/wire/history identity,
public import boundary, privacy, errors and host authority separation.
Exact readonly owner full tests/vet/build and focused races/checker pass;
public/private fixture tests/races/run/vet/build/modverify pass. Reproduced CLI
bytes match. Source-file guards bind exact source before/after final checks.
Local SDK bootstrap is explicitly separated from ordinary upstream publication;
the OpenUdon grant and fresh ordinary SDK/consumer acquisition remain required.
No closing acceptance or retirement is claimed before that independent gate.

### Review 1 finding R99-1 — persisted before fix

P2: wire/json.go:31 supplies only the static destination type. encoding/json
follows a populated interface containing a non-nil record pointer, while the
scanner sees free-form any. Read-only offline Go1.26.6 overlay probes show
DecodeStrict of scope/long-s aliases succeeds with last-value overwrite through
both root any and nested struct.Value any containing *Record. This violates
the typed-record alias contract. Fix scope remains M99.1: carry actual
destination values through pointers/interfaces, existing slice elements and
struct fields, while retaining fresh map values/free-form boxed or nil-pointer
interfaces and exact duplicate bounds. Retain this failed candidate/source
6cf6d9ebc5f38bb7476bccfb1e0c780279ef5937 and its local proof; a fresh source
checkpoint/qualification and whole review 2 must bind the repair. No upstream
semantic, wire/API/schema, authority or new feature change is authorized.

R99-1 repair carries destination reflect.Value alongside declared types,
following non-nil pointer interface contents only where encoding/json does.
Nested struct fields and reused slice elements retain the actual destination;
map elements, boxed values and nil-pointer interfaces stay free-form. The native
self-containing-interface escape hatch is preserved. Root/nested/slice Unicode
alias refusal and positive exact-field/boxed/nil/map/self-reference controls pass.
Both read-only whole-review reports found no other P1/P2; the lower current
Build documentation mismatch is corrected to M83. Fresh full-source gates follow.
