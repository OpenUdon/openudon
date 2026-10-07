# P09 — Package v3

**Stage:** Kinet STG-11, Phase B. **Owner:** OpenUdon.
**State:** Confirmed serial Stage 11 execution; all five task rows complete; whole closing review pending; pre-publication review 3/10 passed; closing publication review pending.
**Source baseline:** `7cd7fbb837fb87e1ca4abea2a362790b0f434188` (clean at planning).
**Coordinator:** [Stage 11 contract](../../../kinet/docs/stage11.md); the package-local milestone/status owns acceptance.

## Dependencies and handoff

[UWS:C09](../../../uws/tabilet/docs/history/status-C09.md); [APItools:M82](../../../apitools/tabilet/docs/history/status-M82.md); [OpenUdon:M98](../docs/history/status-M98.md); [Kinet:W17](../../../kinet/tabilet/docs/history/status-W17.md).
The serial predecessor is a scheduling gate; direct contract and regression impacts are also listed. Every prerequisite must pass its whole review, and required publication must be independently verified before adoption. Record exact accepted/published sources and fixture/build hashes; no Stage 11 acceptance or future pin is claimed yet.

**Downstream:** [Kinet:A15](../../../kinet/tabilet/memory-bank/status-A15.md), [Kinet:W18](../../../kinet/tabilet/memory-bank/status-W18.md), [Kinet:M47](../../../kinet/tabilet/memory-bank/status-M47.md), [Kinet:W19](../../../kinet/tabilet/memory-bank/status-W19.md). Reconcile every affected consumer against the accepted prerequisite revision before advancing.

## Tasks

| Item | State | Notes |
|---|---|---|
| P09.1 — Define v3 package and review records | `[+]` | Define package/handoff/assessment versions covering approved YAML bytes, data.json, source artifacts and operation shapes. Exclude authored intent.hcl and packaged workflow.hcl; retain the existing package digest algorithm. |
| P09.2 — Build directly from UWS | `[+]` | Build and assess v3 packages from standard UWS without intent synthesis. Preserve source-family limits, pending refusals, credential filtering and public package policy. This is the first supported public v3 construction/assessment surface; do not require stable public v2 synthesis APIs from M98. |
| P09.3 — Verify sources and derive authority | `[+]` | Provide library verification that reproduces or validates shapes against exact source artifacts before approval; the consuming author/execution worker supplies isolation, bounded source access and lifecycle controls. Derive exact operation/input/worker authority and reject forged tables, provenance, security alternatives or stale sources. |
| P09.4 — Preserve v2 and evidence readers | `[+]` | Keep historical v2 inspection, approval and report readers and the legacy browser path. Converted bytes get new identities; no reader silently upgrades a package or carries a grant forward. Kinet cut-over retains read-only non-browser v2 history; future runs need explicit conversion and fresh approval. Preserve the independently pinned browser path without introducing a dual non-browser executor. |
| P09.5 — Qualify and publish v3 | `[+]` | Exercise tampering, unsupported versions, missing artifacts, privacy and old/new compatibility. Publish exact public trust APIs/schema fixtures under named authority before Kinet adoption. |

## Acceptance and verification

V3 has an independently checked source-to-shape-to-authority chain and exact digest-bound inputs. V2 history stays readable and no authority is inferred from conversion.

go test ./...; go vet ./...; make check; API/import-boundary and trust-wire fixtures; affected exact-pin consumer checks; git diff --check. Use owner-required offline browser smoke/qualification only for affected retained browser paths.
Use only disposable roots and fixtures. Preserve published schemas/wires, historic evidence, current runtime capability restrictions and the installed M44 service. Changed v3/package/worker identities require fresh approval; they do not preserve old grants.

## Execution policy

One execution owner, serial execution and task commits under the later confirmed goal. Planning authorizes no code execution, commit, publication or external operation. Source publication requires separately named authority; a status marker or local build is not publication. Consumers must record exact accepted and published prerequisites before adoption. Default checks are offline, credential-free and model-free. No deployment, live ledger migration, real API/model/mail action or registration change.

## Approved review-intake amendments — 2026-10-06

Source: **Stage 11 planning review — cross-package refactoring** (2026-10-06). Review baseline and full local revalidation HEAD: `7cd7fbb837fb87e1ca4abea2a362790b0f434188`; relevant uncommitted Stage 11 planning changes were included. The review covers the five owner baselines recorded in the coordinator. This is approved intake, not a closing review iteration; the persisted counter remains 0/10.

- **P3-7** — source P3; local Lower; confirmed. Evidence: status-P09.md original P09.3; ../kinet/tabilet/memory-bank/status-M46.md. Ownership/lineage: P09.3 supplies verification APIs; consuming workers supply isolation.

## Accepted binding prerequisite — 2026-10-06

UWS:C09 is accepted and independently observed on origin/main at `6a267306032edc687a298cefc8bba7019d3ad059`, whole review 3/10. [Public contract](../../../uws/docs/binding-reference.md) and [qualification](../../../uws/docs/c09-qualification.md) pin source-neutral ShapeTable/Resolver APIs, metadata-only binding checks and deterministic flow observations. Known/unknown evidence remains explicit; nested/typed templates use exact projections and unproved constraints stay indeterminate. Metadata producer claims are independently reproduced/verified before authority. No APItools/private-runtime import, source parser, credential/provider I/O, ordinary-validator/schema change or execution permission is supplied by C09. Retirement closure `8e5be730aa68aa4cb4f6c591a3a9425c6b8bd55c` was independently observed on authorized origin/main, satisfying the prerequisite publication gate. [Publication evidence](../../../uws/docs/c09-publication.md) records accepted-source ancestry.

## Accepted shape producer prerequisite — 2026-10-06

APItools:M82 is accepted after whole review 3 at exact public source
`54583f9b2f452b7cc522360c5aeeff29ca22f96c`, independently observed on the
unchanged authorized origin/main. The configured registry resolves
`v0.0.0-20261006210844-54583f9b2f45` to that full origin hash.
[Contract](../../../apitools/docs/operation-shapes.md) and
[qualification](../../../apitools/docs/m82-qualification.md) define the additive
BuildOperationShapeTable/VerifyOperationShapeTable APIs over accepted UWS C09
`6a267306032edc687a298cefc8bba7019d3ad059`. The eight-family/twelve-operation
fixture is 9,829 bytes, SHA-256
`dc20d2287322a4b20d5d92b1a0d1ec8036f1bec853b642d7df8797ed096c3ba1`.

Consumers must independently reproduce exact source/shape claims, preserve
native selectors/protocols and OR-of-AND security, and keep partial dialect/
wire/presence/auth evidence unknown. Source URLs are sanitized provenance only;
no fetching, credential resolution or execution is supplied. Bounds are 32
sources, 20 MiB each/64 MiB total raw, 10,000 operations, 8 MiB table/aggregate
projected schemas, 256 KiB per schema and 100,000 projection nodes total/10,000
per schema/depth 50. Incremental expansion/serialization checks refuse without
partial tables; hard CPU/RSS/deadline/mount/network controls remain with workers.
The source tooling still carries the public UWS Horizon/HCL closure; no HCL-free
or private runtime claim is made. Current consumer/browser pins remain unchanged
until this milestone's explicit adoption; Retirement closure `fae9982e42d6b16fe7a5ebfd342a016613a62adb` was independently
observed on authorized APItools origin/main, satisfying the publication gate.

## Persisted review

- Review iteration: **3/10**; pre-publication whole review passed at fixed source a6a3ef010fe27f277f8191204c1c81ea1cc0334b; closing publication/consumer acceptance remains pending.
- Closing-review findings: R1-F01/R1-F02 and R2-F01 are fixed with regressions/full checks. Reviews 1 and 2 did not pass their initial gates; complete pre-publication review 3 passed with no remaining P1/P2.
- Accepted revision: not available.
- Published revision / artifact evidence: not available.
- Verification: P09.1–.4 and P09.5 candidate fixture/default checks are recorded below; clean ordinary source/consumer qualification and publication remain pending.

After all tasks finish, perform the whole-milestone review with persisted iteration/finding state and fix every P1/P2 before acceptance. Resume an interrupted pass at the same counter. Consolidate current facts, reconcile downstream work and retire under this package’s normal procedure.

## Accepted/published OpenUdon M98 prerequisite — 2026-10-07

OpenUdon M98 accepted all four tasks after pre-publication review 2 and closing
review 3 at exact qualified source
08a3839f357ec40c7e50c8668e4bd7c8d86bb55a. Normal origin/main publication
642fddbcf960ff5d23edace6cbdc4d1202e1e291 was independently observed and
contains that source. Ordinary module
v0.1.1-0.20261007040813-08a3839f357e resolves to the full origin hash, with
module sum h1:Gn8HnUc5gPa7DnSTEnF2LbaRS5y/5NWfiiqVxutNy/g= and go.mod sum
h1:gYKkottLX/IoTptmggqMl1dMHIi/cg+Tg01OCXhzcGs=.

[API contract](../../docs/public-trust-api.md),
[qualification](../../docs/m98-qualification.md) and
[publication](../../docs/m98-publication.md) bind exact source,
90-module/89 ordinary archive owner closure, standalone checks and reproduced
CLI hashes. Independent published consumer uses pinned Go 1.26.6, 57 modules
and no directory replacements. Public handoff/digest/approval/authority/wire/
trust/report/run-evidence APIs import no OpenUdon internal/private executor.
Neutral verification does not assess source semantics, reproduce shapes,
construct v2 packages or grant authority. Host isolation, stable snapshot
custody, independent provenance and value-free reduction remain consumer-owned.
Legacy browser verification/execution stay on the separately pinned CLI path.
No current Kinet execution/browser/media pin, schema, hosted capability or
installed service changed; all tasks/review in this consumer remain pending.

P09 owns supported public v3 construction/assessment and exact source-to-shape
verification. Extend API/import guards for its new surface while preserving
M98's frozen existing declarations and wires. M98 supplies no stable legacy
synthesis types; the consumer worker owns isolation and lifecycle. W17 remains
the serial predecessor, so no P09 task is selected by this reconciliation.

M98 acceptance/retirement closure `b8eaf1626037561b19f62642329a5c0b5f928f45` is independently observed
on unchanged authorized OpenUdon origin/main. It contains exact qualified
source/module ancestry and passed closing review 3; the full prerequisite
source/closure publication gate is satisfied. P09 still waits for W17.

## Accepted Kinet W17 Phase A prerequisite — 2026-10-07

Kinet W17 is accepted after whole review 2 at fixed consumer source
c508e97e6099936286c9746660427e8af0a3930a.
[Phase A checkpoint](../../../kinet/docs/w17-phase-a-checkpoint.md),
[comparable profile](../../../kinet/docs/w17-profile-comparison.json) and
[view qualification](../../../kinet/docs/w17-views-qualification.md) establish
matched old authoring/corpus/pin behavior, qualified advisory/verified views
and actual local/rootless/CLI/API/privacy/ownership/sandbox evidence. All
original M45 investigation budgets pass; dirty capture context is derived
truthfully and original baseline evidence remains frozen.

The public-only author module uses exact accepted APItools/UWS/SDK dependencies;
current immutable worker source d557eaddd232c446d272f6881e70c9726899ff26,
binary 0708704481cef72ff0cec2b090ec2171ab6e02013326d8f1c6bc1d7dc55d3897 and
image sha256:5924cf6f3e5196a0b9b7f3d245122e7c42ad0cd5597ff05fa248036306db8225
are reproduced and qualified in the [pin/closure](../../../kinet/docs/w17-author-worker-pin.json).
Host resource admission is fixed before buffering; source parsing/reproduction/
codec verification remain worker-owned. Root imports no source parser/private
runtime. Old execution/browser/media pins, schema/wires and installed M44 stay
unchanged; no authority is derived from metadata/views or audit.

P09 remains the next separately owned required milestone under the original
confirmed goal: public v3 construction/assessment, exact source-shape proof
and concrete authority. Do not require public v2 synthesis APIs or broaden
hosted support. Kinet owns private model/confirmation/ledger publication and
consuming worker isolation; A15 follows P09 before W18/M47 integration.
Both phases remain required with exact named source-publication gates; this
handoff grants no deployment/live operation or grant carryover. P09 task rows
and persisted whole review remain 0/10 pending their own execution.

## P09.1 selection — 2026-10-07

Continue from clean OpenUdon c468d9a27c9b42f9cfa4c266c403f1321edec89e after
Kinet W17 acceptance/review2 and literal retirement closure
83ddbc3518e31bb07ccf63d84819c0460eba65dd. Exact accepted consumer is c508e97;
its published upstream gates and qualified public worker are reconciled above.
Select only v3 package/handoff/assessment records over exact approved YAML,
data.json, source bytes and operation shapes, excluding authored/packaged HCL.
Retain digest-v1 algorithm and all old wire/read/browser paths. No construction
acceptance, authority, publication, live action or consumer pin is inferred;
P09.2–.5 and whole review remain required.

## P09.1 implementation checkpoint — 2026-10-07

Public packagev3 defines exact YAML/data/shapes/source input manifest and
separately linked handoff/assessment records, with additive v3 schemas.
Canonical scope/path/SHA/role/source inventory and symbolic-only review-required
metadata are validated. Strict decoders reject unknown/duplicate/case aliases,
missing/null fields and trailing documents under existing public wire limits.
InputDigest delegates unchanged digest-v1 identity/scope/sort rules; reports
have separate identities and final package construction remains P09.2.
Authored/packaged HCL/private/browser paths are excluded, and no record
manufactures source proof, assessment success or approval. Existing wire/schema/
private legacy readers and root dependency versions are unchanged.

Focused public API/schema/closed-wire/HCL/duplicate/source/identity and old
digest parity tests pass; compatible focused staticcheck passed. Full owner
`GOWORK=off GOPROXY=off make check`, `go vet ./...` and focused public
`go test -race ./packagev3 ./handoff ./trust ./wire` passed. The default public
import guard now covers packagev3 and rejects internal/private runtime imports.
P09.1 is complete as structural records only; construction, source proof,
authority, publication and downstream adoption remain P09.2–.5. Persisted
whole review remains 0/10.

## P09.2 selection — 2026-10-07

P09.1 structural record task is committed at d5b4e33. Correct its completed
marker to this owner’s `[+]` convention. Select public explicit-byte construction
and assessment; adopt only the exact accepted/published APItools M82 and UWS
M08 closure already consumed by the qualified Kinet worker. No directory
replacements, old schema edits, credential loading, network or execution.
Source reproduction is mandatory; independently checked execution authority
remains P09.3 and publication remains P09.5. Whole review stays 0/10.

## P09.2 completion — 2026-10-07

Public Build/Assess now construct exact byte snapshots without intent synthesis,
ambient file/network/credential access or runtime binding. Independent APItools
shape reproduction, exact artifact hashes, closed embedded core schemas,
lossless json.Number restoration, strict portability, pending/executable and
entrypoint/source/binding checks precede review assessment. Partial runtime,
step/type/output/work/effect evidence remains indeterminate; incompatible
packages remain reviewable but cannot qualify approval. Handoff stays
review_required, final digest uses unchanged digest-v1 and no grant is created.

Current owner adopts only accepted/published APItools M82
54583f9b2f452b7cc522360c5aeeff29ca22f96c and UWS root/codec M08
c0b19385a3b034cd45de16726668b9150f0633f2, using ordinary exact pseudo-versions
without replacements. The qualified public Horizon/HCL transitive subtree is
allowed by the new surface's import guard; private executor and OpenUdon
internal imports still refuse. Public credentialpolicy shares the unchanged
literal scan through legacy wrappers; browser artifact masking stays private.
The current-owner content-trust pin assertion follows M08; historical evidence,
browser/consumer locks, old schemas/wires and installed M44 stay frozen.

Offline tests prove exact immutable YAML/data/source bytes, deterministic
identities, all eight native families against the unchanged 9,829-byte M82
fixture, 17 exact embedded core schema entries, precise numeric mismatches,
pending/unsupported/stale/forged/privacy/alias/duplicate/size/cancellation
refusals and unknown output/type evidence. GOWORK=off GOPROXY=off make check,
full go vet, focused staticcheck and public/policy/legacy-content-trust races
passed. The first full check caught the stale current-owner UWS pin assertion;
it was reconciled before the final full check passed. No publication or
milestone acceptance is claimed. P09.3–.5 and whole review 0/10 remain required.

## P09.3 selection — 2026-10-07

Continue from clean task source 157a31e5638e42739994b0ebb2eeb94731c8fff4.
Select independent closed snapshot verification and exact source/shape/operation,
input and worker-bound authority derivation. Retain broker-authority v1 wire;
current supported authority profile stays bounded, concrete HTTP sequence.
Runtime function catalogs remain runtime-owned, with explicit independent
consumer verification rather than private OpenUdon imports. Unknown metadata
or producer flags never create authority. P09.4/.5 and whole review remain
required. Doc-memory warning was reviewed against v50: implementation advances
its approved direction, so no new evolution version is warranted.

## P09.3 verification checkpoint — 2026-10-07

Uncommitted public Verify now owns an independent bounded copied snapshot;
callers supply exact expected scope/package digest. It checks canonical closed
manifest/handoff, exact complete inventory, every input/report hash and linked
identity, reproduces API-source shapes and reruns assessment. VerifiedPackage
has private state; its zero value is unverified and Snapshot/Assessment return
independent copies. Extra/private/HCL/missing artifacts, stale/forged shapes,
rehashed report/scope/credential claims and noncanonical records refuse with
fixed errors. Current constructor declares no credential slots; verifier rejects
invented ones until P09.3's independent symbolic security derivation supplies
that contract. No verification flag becomes approval or execution authority.

Focused verification/import tests and compatible staticcheck pass. `go test -race ./packagev3 ./internal/publicapi` also passed. Concrete operation/input/
worker authority and runtime-owned function catalog integration are unfinished;
P09.3 remains the sole in-progress row, whole review remains 0/10, and no
publication/consumer adoption or external operation is claimed.

## P09.3 implementation and preliminary consumer evidence — 2026-10-07

Verify now independently derives selected known symbolic credential inventory
and rejects fabricated handoff names. Up to 32 API sources plus one reserved
runtime-function catalog are admitted. Exact API shapes are reproduced in the
public SDK; runtime catalog claims require the trusted implementing worker's
independent verifier. Catalog identity/revision, security, native selector and
partial type/invocation metadata cannot be replaced by producer booleans.
Unknown source security remains indeterminate and does not invent credentials.

ExecutionPlan binds full package/handoff/inputs, exact source/native operation,
raw operation/step/data constraints and worker binary/closure/runtime revision.
Concrete profile remains the qualified bounded sequence; pending, unknown
controls/overrides/browser/non-HTTP and ambiguity refuse. Pure functions require
catalog revision equality and the selected runtime's non-effectful admission
adapter, preserving partial metadata. Broker authority v1 is derived only from
compatible complete HTTP review, one fixed server/security alternative, native
APItools bearer/API-key evidence and exact current symbolic revisions. Basic,
OAuth/scopes, altered placements, extra credentials, old populated seeds and
changed package/input/worker authorities refuse. Existing wires are unchanged.

Execution approval additionally requires the trusted host's exact confirmed
PlanSHA256 and current explicit-expiry scope/package approval; HTTP needs broker
v2 and pure functions use v1 after native admission. Host owner/grant custody,
revocation, destinations, actual compiled worker identity and lifecycle remain
independently enforced by Kinet. No library flag, audit or artifact is authority.

The ordinary published Udon M48 module independently generated and verified
catalog.json (3,468 bytes/SHA 67933ec02e3b8808ef32637272295908661a39555c13f9f0bfb057af54097812)
and shapes.json (4,189 bytes/SHA 7b8c176eb07315ca076a282887c4f9a54cda810f49a8820eed21bde303bde35e).
Preliminary private consumer /var/tmp/openudon-p09-private-adapter-prepub-20261007
passed actual VerifyRuntimeFunctionCatalog and Compile/CheckSupported adapters,
public Build/Verify/plan and exact approval checks with zero effects. It uses
one temporary local SDK replacement and synthetic worker hashes, so it is not
final published consumer/worker qualification. P09.5 supplies ordinary published
SDK resolution; Kinet M47 qualifies actual worker provenance/isolation.

All eight selected native-family operations can be packaged for review without
invented HTTP/known security; separate pending fixture preserves the exact M82
golden. Full make check/vet, focused staticcheck and public trust/authority races
pass. Latest focused regression, vet and public trust/authority races after the
unknown-security inventory correction also pass. Doc-memory check passes;
its evolution reminder was checked against approved v50, with no direction
change or new evolution version. P09.3 is complete as public verification and
exact constraints/authority APIs, not downstream worker acceptance. P09.4/.5
and persisted whole review 0/10 remain required; no publication or live action.

## P09.4 selection — 2026-10-07

Continue from clean task source 4de9271f371ca34d88163aff7aab0b150e3e12c5.
Select compatibility readers/fixtures and explicit format dispatch; preserve
existing v1/v2 handoff/approval/report surfaces and separately pinned browser
path. No conversion, v2 authority carryover or legacy non-browser execution
fallback is introduced. Kinet owns cut-over/conversion and fresh owner approval
in W18/W19/U14; P09.5 still owns whole review/publication qualification.

## P09.4 completion — 2026-10-07

Explicit InspectHistory dispatches selected v2/v3 format/scope/digest without
conversion or runtime invocation. It returns read-only byte identities and
SourceProof=false; function/unsupported-execution v3 history needs no private
runtime adapter. Private snapshot-integrity sharing cannot construct a verified
proof: only source/shape/assessment Verify creates VerifiedPackage. Unsupported
formats, mismatched selections, stale scope/digest/bytes, privacy and cancellation
refuse. Tests prove old HCL/handoff bytes stay unchanged, v2 cannot enter v3
verification, old valid approval remains readable but cannot authorize a new
v3 successor, and historical report/evidence/approval golden readers are stable.

Owner make check/vet, focused staticcheck, affected public/history/trust/approval/
report races, private legacy identity/broker regressions and diff checks pass.
No old public schema/wire fixture, legacy CLI reader, browser code/lock or Kinet
path changed. Read-only image inspection and an actual bounded rootless
network-none/read-only CLI version probe independently retain browser image
sha256:a88e1c3deb570350661d4522b9b0c8540e17b56f019695d61023c79a673b8e8d,
source c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0, binary
072cd762973804db7c72355b838be077b93d55463c9c331e877382b2db8297a4,
go1.26.6, vcs.modified=false. This is pin/probe evidence, not new capture
qualification; frozen M46/W17 capture qualification is preserved.

Kinet W18/W19/U14 still own cut-over/conversion and fresh exact owner/worker/
schedule authority. No legacy non-browser fallback, second executor, silent
migration or carried grant is introduced. P09.5 and whole review 0/10 remain
required; no source publication or external live operation is claimed.

## P09.5 selection — 2026-10-07

Continue from clean task source f8822b5fdaec5546346cf5a872941465a4627162.
Select full public contract/schema/tamper/privacy/compatibility qualification,
clean ordinary owner/SDK/private-adapter build closure, persisted bounded whole
review and named normal origin/main source publication. Preserve exact existing
schemas/wires/pins and qualified current restrictions. Final consumer proof must
resolve the independently observed published SDK source without directory
replacements; preliminary local bootstrap results are not that proof. Persist
review starts/findings before reviewing/fixing, then reconcile every pending
consumer to exact accepted/published sources and retire the complete envelope.
No deployment, live package inventory, provider/model/API/mail, credential or
registration operation is included. Whole review remains 0/10, not started;
first prepare the final public fixture/API/build gate before starting review.

## Persisted whole review 1 started — 2026-10-07

Review the entire P09 implementation from prerequisite handoff c468d9a through
current f8822b5 plus the P09.5 candidate API/schema/contract fixtures. The stored
count advances from 0 to 1 before review. Inspect correctness, authority/source
proof, privacy/bounds/cancellation, schema/wire compatibility, source-family and
runtime adapters, historical readers, docs and all owning acceptance criteria.
Source publication waits for all P1/P2 to close and a passing pre-publication
pass; final clean/published consumer evidence and closing acceptance follow.
No finding is closed or acceptance inferred by starting this pass.

## Whole review 1 findings persisted before fixes — 2026-10-07

- **R1-F01 (P2, open):** assessment checks only a subset of operation response
  spellings and no semantic step/workflow output-reference proof. Public core
  grammar uses `$response.headers` (plural) and 1.11+ body dot paths; current
  code checks singular header and only body/pointer forms. A workflow output
  `$steps.missing.outputs.value` passes strict syntax and yields only advisory
  output_unreferenced; P09 reports compatible and can derive concrete authority.
  Reproduced by failing TestMissingFlowReferenceCannotQualifyAuthority at the
  current candidate. Add complete core response/reference admission over exact
  reviewed source/output contracts; unresolved/ambiguous/unproved references
  cannot qualify authority. Preserve shared UWS advisory semantics and old wires.
- **R1-F02 (P2, open):** P09 never supplies independently derived ExpressionTypes
  to C09 binding checks. Exact reviewed data input `$variables.inputs.n` with
  data.json n=9007199254740993 becomes binding.expression_type_unknown despite
  the known matching source schema. This breaks required input/data-flow parity
  and prevents later direct authoring/execution adoption. Reproduced by failing
  TestExactReviewedVariableInputCanQualifyBinding. Derive contracts only from
  exact reviewed variable/input bytes and source-backed output schemas, retain
  numeric lexemes and null/presence/constraint unknownness, and never accept
  caller/model type assertions. Qualification must cover chained inputs and
  every core response spelling, missing/optional paths and stale inputs.

The whole pass continues at iteration 1; these findings are not accepted or
carried forward. No publication occurs while either remains open.

## Review 1 fixes verified — 2026-10-07

R1-F01/R1-F02 are fixed by exact source-backed response/step/output contracts
and literal input evidence. The helper uses only the published UWS expression
parser and independently verified shapes; it does not execute an operation or
accept model/caller type assertions. Known reviewed data values emit exact const
schemas with json.Number retained; known required source fields retain their
constraints. Optional/open/nullable/array paths, component/top-variable collision
precedence and unproved scope remain indeterminate. Missing/forward/cyclic
output references cannot qualify compatible review or broker authority. Body,
pointer, 1.11+ dot and plural headers spellings are checked; old source artifacts
and all public wire declarations stay unchanged.

The two regressions failed before fixes; after fixes, they and exact chained
source/output types, missing headers/closed fields, optional/nullable presence,
future-step and all previous package/security/privacy/history tests pass.
Full make check, full vet, focused staticcheck and public package/trust/authority/
approval/evidence races passed. No P1 or additional P2 was found in the rest of
this complete review pass. Review 1 itself remains failed due to those original
findings; a new whole pass is required after the fixed source is committed.
Final clean build/module closure, accepted publication and consumer proof remain
pending, and no P09 acceptance is claimed.

## Persisted whole review 2 started — 2026-10-07

Review 1 findings/fixes and final checks were read before advancing the stored
counter to 2. Review the full milestone from c468d9a through affd575ecaa2043da29badd599870efd7102614e,
including all contracts, trust/privacy/bounds, ordinary dependency adoption,
old/new compatibility and the exact expression/source proof fixes. Clean build,
publication and final ordinary published consumer gates remain pending. This
start does not infer a passed review or milestone acceptance.

## Review 2 finding persisted before fixes — 2026-10-07

- **R2-F01 (P2, open):** the new response child projection discards restricting
  parent/sibling schema keywords. A known response schema requiring n while
  setting maxProperties=0 becomes a simple child integer contract; the chained
  input is assessed compatible despite the impossible parent. Reproduced by
  failing TestParentResponseConstraintsCannotDisappearDuringProjection at
  affd575. Preserve supported parent constraints or retain explicit unknownness
  when projection cannot prove them. Const projection must likewise validate
  the entire retained schema rather than dropping restricting siblings. Keep
  exact numeric handling and all existing public wires. Fix/reverify before
  beginning review 3; no source publication while this finding remains open.

The separate exact-input contradictory-schema regression already passes, so
its full C09 validation is not a new finding. Review 2 continues across the
whole scope; fixture-only source regeneration/build evidence remains pending.

## Review 2 fix verified — 2026-10-07

R2-F01 is fixed: child projection admits only its proved structural parent
subset; any additional restricting parent keyword remains unknown rather than
being discarded. A const source is checked against the entire retained schema
with the existing closed resource loader before its child value is projected.
Numeric values still use strict json.Number decoding. The parent-constraint
regression failed before and passes after; the direct contradictory-input
schema regression, all review-1 regressions and prior fixture suites pass.

Full owner make check/vet, focused staticcheck and public package/trust/authority/
approval/evidence races pass. The exact published Udon M48 preliminary private
adapter again passes catalog reproduction, native admission, plan and approval
with zero effects. This local SDK bootstrap remains preliminary, not published
consumer or actual worker qualification. No additional P1/P2 was found in the
rest of this whole pass. Review 2 did not pass because R2-F01 was originally
open; review 3 must re-examine the full milestone after the fixed commit.

## Persisted whole review 3 started — 2026-10-07

Read review-2 finding/fix/verification before advancing the counter to 3. Review
all P09 source/contracts/fixtures and earlier fixes at exact clean
 a6a3ef010fe27f277f8191204c1c81ea1cc0334b (including ordinary dependency closure,
privacy, authority, supported/unknown semantics, old/new readers and source/schema
constraint preservation). Fresh clean ordinary owner/build qualification is
selected as final pre-publication evidence. No passed review, publication or
milestone acceptance is inferred by this start.

## Pre-publication review 3 and clean owner qualification — 2026-10-07

The complete whole review passes at a6a3ef010fe27f277f8191204c1c81ea1cc0334b
with no remaining P1/P2. Earlier source/authority/output/type/parent fixes,
privacy/bounds/context and old/new compatibility were re-examined across the
full milestone; current capability/unknownness and host trust split remain
explicit. Frozen earlier schemas/wires and browser/runtime/media pins remain.

Clean ordinary owner make check/vet and reproduced CLI builds pass; actual VCS
probe names the full source, Go 1.26.6 and vcs.modified=false. Complete closure
is 91 modules/90 hashed ordinary archives, no replacements or missing archives.
[Qualification](../../docs/p09-qualification.md), [build](../../docs/p09-qualified-build.json)
and [archives](../../docs/p09-module-archives.json) persist exact identities.
Openudon CLI SHA ffa7dc3be990755da7ee9af1fa9fd08642bb423e4491a9cfae564925270aefdb;
runner SHA f089f467142952fc0f7b84e412af41ae59e9cf37e6bc9539776c24a1bb0836eb.

Pre-publication review passing does not close P09.5 or the milestone: normal
named origin/main source/closure publication, independently resolved ordinary
published public/private SDK consumers, closing review, downstream exact-source
reconciliation and literal retirement remain required. No deployment/live action
or consumer adoption is authorized by this preliminary source qualification.

## P09.5 published-source completion — 2026-10-07

Normal named origin/main publication da6653145c8624f862e3c21adb413345d0cb4be2
is independently observed by ls-remote/fetch at the unchanged authorized target
and contains qualified a6a3ef010fe27f277f8191204c1c81ea1cc0334b. Ordinary SDK
v0.1.1-0.20261007094531-a6a3ef010fe2 resolves to that full origin hash, sum
h1:Tomt06dU+DCTEqrTNlDlcFnFZGl/r8fvpIXaSUOwu7s=, go.mod sum
h1:kol8tnW9phAZtwTfHJ2z9Oa8eUnfUYcQIwcxPcyvl4g=.
[Publication](../../docs/p09-publication.md) and [consumer proof](../../docs/p09-consumer-proof.json)
record exact archive/closure evidence and the no-deployment skip-CI source push.

Ordinary public consumer passes package/plan/broker/approval/history/tamper
with 73 modules and no replacements. Ordinary private consumer passes actual
M48 catalog reproduction/native admission plus published SDK package/plan/
approval with 171 modules and no directory replacements. It retains the exact
M48-required docker->moby v24.0.7 version replacement; no new one is introduced.
Both use controlled synthetic worker/time identities, zero effects and pinned
Go 1.26.6; actual Kinet worker qualification remains M47. Full graph observation
needed one already-selected historical x/telemetry archive/metadata fetch under
existing build-closure authority, then passed offline without version change.

All task rows are terminal, but terminal rows do not prove acceptance. Closing
whole review, current-fact consolidation, exact downstream reconciliation and
literal retirement remain required before advancing. No deployment/live action.
