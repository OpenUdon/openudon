# M98 — Public trust libraries

**Stage:** Kinet STG-11, Phase A. **Owner:** OpenUdon.
**State:** Approved planning on 2026-10-06; 4 pending rows, no implementation or acceptance.
**Source baseline:** `7cd7fbb837fb87e1ca4abea2a362790b0f434188` (clean at planning).
**Coordinator:** [Stage 11 contract](../../../kinet/docs/stage11.md); the package-local milestone/status owns acceptance.

## Dependencies and handoff

[UWS:C08](../../../uws/tabilet/docs/history/status-C08.md); [Udon:M48](../../../udon/tabilet/memory-bank/status-M48.md).
The serial predecessor is a scheduling gate; direct contract and regression impacts are also listed. Every prerequisite must pass its whole review, and required publication must be independently verified before adoption. Record exact accepted/published sources and fixture/build hashes; no Stage 11 acceptance or future pin is claimed yet.

**Downstream:** [Kinet:M46](../../../kinet/tabilet/memory-bank/status-M46.md), [OpenUdon:P09](status-P09.md), [Kinet:M47](../../../kinet/tabilet/memory-bank/status-M47.md). Reconcile every affected consumer against the accepted prerequisite revision before advancing.

## Tasks

| Item | State | Notes |
|---|---|---|
| M98.1 — Extract handoff digest and authority APIs | `[ ]` | Expose deliberate public packages for existing handoff, digest, approval and Authority types. Preserve published discriminators, canonicalization and wire bytes; protect the public/private import boundary. |
| M98.2 — Define format-neutral verification boundaries | `[ ]` | Expose bounded format-neutral trust inspection/verification types without making synthesis-coupled v2 construction, assessment or simulation orchestration a supported public API. Retain those legacy implementations privately behind unchanged CLI adapters. Public v3 construction/assessment belongs to OpenUdon:P09; include affected mockruntime simulation regression vectors without broadening the M98 API promise. |
| M98.3 — Expose evidence verification | `[ ]` | Expose run-evidence and Udon-report wire verification without importing Udon. Add golden/API-surface fixtures for current approvals, broker identities, reports and uncertainty. |
| M98.4 — Qualify and publish public interfaces | `[ ]` | Run public standalone tests, boundary guards, wire vectors and affected consumers. Publish accepted source with named authority; the existing CLI and execution path stay available in Phase A. API-surface/import tests must reject an accidental public dependency on internal/synthesize or its legacy construction types. Source/shape reproduction is owned by P09, not M98. |

## Acceptance and verification

Consumers use supported format-neutral trust APIs without internal/private runtime imports or a stable synthesis-coupled v2 construction surface. Existing CLI/trust bytes remain compatible; P09 owns public v3 construction.

go test ./...; go vet ./...; make check; API/import-boundary and trust-wire fixtures; affected exact-pin consumer checks; git diff --check. Use owner-required offline browser smoke/qualification only for affected retained browser paths.
Use only disposable roots and fixtures. Preserve published schemas/wires, historic evidence, current runtime capability restrictions and the installed M44 service. Changed v3/package/worker identities require fresh approval; they do not preserve old grants.

## Execution policy

One execution owner, serial execution and task commits under the later confirmed goal. Planning authorizes no code execution, commit, publication or external operation. Source publication requires separately named authority; a status marker or local build is not publication. Consumers must record exact accepted and published prerequisites before adoption. Default checks are offline, credential-free and model-free. No deployment, live ledger migration, real API/model/mail action or registration change.

## Approved review-intake amendments — 2026-10-06

Source: **Stage 11 planning review — cross-package refactoring** (2026-10-06). Review baseline and full local revalidation HEAD: `7cd7fbb837fb87e1ca4abea2a362790b0f434188`; relevant uncommitted Stage 11 planning changes were included. The review covers the five owner baselines recorded in the coordinator. This is approved intake, not a closing review iteration; the persisted counter remains 0/10.

- **P2-8** — source P2; local P2; confirmed. Evidence: internal/trustedrunner/trustedrunner.go AssessCurrent dependency; original M98.2. Ownership/lineage: M98.2 narrows public compatibility to format-neutral verification; P09 owns v3 construction and A31 respects the resulting surface.

## Accepted expression prerequisite — 2026-10-06

UWS:C08 is accepted and independently observed on origin/main at `0411eea6fc84fbd6aa97cef94f53f301260f4844`, whole review 2/10. [Qualification](../../../uws/docs/c08-qualification.md) and the [supplement manifest](../../../uws/docs/examples/expressions/v1/manifest.json) identify exact source/vector bytes. Ordinary validation and frozen published artifacts are unchanged; strict portability is opt-in, required for new Kinet:W18 packages. Legacy mock numeric/encoded-root adapters remain explicit. Values must use lossless json.Number projections before constructing snapshots because outer UseNumber does not override legacy custom model decoders. C08 supplies no source parsing, credential/provider I/O or execution authority. Retirement closure `5c0c74f48d84588e3ff4f994f0713f199cfcc67c` was independently observed on origin/main; that publication gate is satisfied. [Publication evidence](../../../uws/docs/c08-publication.md) records the accepted-source ancestry.

## Persisted review

- Review iteration: **0/10**; not started.
- Closing-review findings: none; the whole-milestone review has not started. Approved intake requirements above remain pending.
- Accepted revision: not available.
- Published revision / artifact evidence: not available.
- Verification: pending implementation; no test result is claimed by this planning record.

After all tasks finish, perform the whole-milestone review with persisted iteration/finding state and fix every P1/P2 before acceptance. Resume an interrupted pass at the same counter. Consolidate current facts, reconcile downstream work and retire under this package’s normal procedure.

## Accepted/published Udon M48 prerequisite — 2026-10-07

Udon:M48 is accepted after all four tasks, pre-publication review 1 and closing
review 2 at exact qualified runtime
da43e57be37af4e18e633558580f740c525f139d. Normal authorized origin/main
publication e338c6bb542c927f7eb5ba559616fbfc7add6aa8 was independently observed
and contains that source. Exact private module
v0.0.0-20261007023819-da43e57be37a resolves to the full origin hash.
[Handoff](../../../udon/docs/m48-release-handoff.md),
[qualification](../../../udon/docs/m48-qualification.md) and
[publication](../../../udon/docs/m48-publication.md) record full ordinary
337-module owner graph/336 dependency archives, independent no-directory
private consumer and reproduced CLI SHA-256
5de04144464357d5f300e2a051bdf218068aea08aad88e46792ffa2aa426d021.

Public OpenUdon must not import Udon or its private dependencies. M98 evidence
verification consumes neutral immutable wire/identity data and preserves its
existing CLI execution adapter/pins; new v3 construction remains P09. The
private runtime's function catalog/shape table are exact reproduced metadata
with incomplete native invocation/domain constraints; metadata grants no
authority. Shared C08/M08 grammar/import/provenance and existing report schemas
are retained, while supported private execution uses explicit capabilities and
scoped lossless response snapshots. No current OpenUdon/Kinet/browser/media
pin is moved by this prerequisite. All M98 rows/review remain pending.
