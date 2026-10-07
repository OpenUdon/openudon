# P09 — Package v3

**Stage:** Kinet STG-11, Phase B. **Owner:** OpenUdon.
**State:** Approved planning on 2026-10-06; 5 pending rows, no implementation or acceptance.
**Source baseline:** `7cd7fbb837fb87e1ca4abea2a362790b0f434188` (clean at planning).
**Coordinator:** [Stage 11 contract](../../../kinet/docs/stage11.md); the package-local milestone/status owns acceptance.

## Dependencies and handoff

[UWS:C09](../../../uws/tabilet/docs/history/status-C09.md); [APItools:M82](../../../apitools/tabilet/docs/history/status-M82.md); [OpenUdon:M98](../docs/history/status-M98.md); [Kinet:W17](../../../kinet/tabilet/memory-bank/status-W17.md).
The serial predecessor is a scheduling gate; direct contract and regression impacts are also listed. Every prerequisite must pass its whole review, and required publication must be independently verified before adoption. Record exact accepted/published sources and fixture/build hashes; no Stage 11 acceptance or future pin is claimed yet.

**Downstream:** [Kinet:A15](../../../kinet/tabilet/memory-bank/status-A15.md), [Kinet:W18](../../../kinet/tabilet/memory-bank/status-W18.md), [Kinet:M47](../../../kinet/tabilet/memory-bank/status-M47.md), [Kinet:W19](../../../kinet/tabilet/memory-bank/status-W19.md). Reconcile every affected consumer against the accepted prerequisite revision before advancing.

## Tasks

| Item | State | Notes |
|---|---|---|
| P09.1 — Define v3 package and review records | `[ ]` | Define package/handoff/assessment versions covering approved YAML bytes, data.json, source artifacts and operation shapes. Exclude authored intent.hcl and packaged workflow.hcl; retain the existing package digest algorithm. |
| P09.2 — Build directly from UWS | `[ ]` | Build and assess v3 packages from standard UWS without intent synthesis. Preserve source-family limits, pending refusals, credential filtering and public package policy. This is the first supported public v3 construction/assessment surface; do not require stable public v2 synthesis APIs from M98. |
| P09.3 — Verify sources and derive authority | `[ ]` | Provide library verification that reproduces or validates shapes against exact source artifacts before approval; the consuming author/execution worker supplies isolation, bounded source access and lifecycle controls. Derive exact operation/input/worker authority and reject forged tables, provenance, security alternatives or stale sources. |
| P09.4 — Preserve v2 and evidence readers | `[ ]` | Keep historical v2 inspection, approval and report readers and the legacy browser path. Converted bytes get new identities; no reader silently upgrades a package or carries a grant forward. Kinet cut-over retains read-only non-browser v2 history; future runs need explicit conversion and fresh approval. Preserve the independently pinned browser path without introducing a dual non-browser executor. |
| P09.5 — Qualify and publish v3 | `[ ]` | Exercise tampering, unsupported versions, missing artifacts, privacy and old/new compatibility. Publish exact public trust APIs/schema fixtures under named authority before Kinet adoption. |

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

- Review iteration: **0/10**; not started.
- Closing-review findings: none; the whole-milestone review has not started. Approved intake requirements above remain pending.
- Accepted revision: not available.
- Published revision / artifact evidence: not available.
- Verification: pending implementation; no test result is claimed by this planning record.

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
