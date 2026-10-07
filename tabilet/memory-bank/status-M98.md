# M98 — Public trust libraries

**Stage:** Kinet STG-11, Phase A. **Owner:** OpenUdon.
**State:** Confirmed serial Stage 11 execution; M98.1–M98.3 complete; M98.4 in progress; whole review 0/10 not started.
**Source baseline:** `7cd7fbb837fb87e1ca4abea2a362790b0f434188` (clean at planning).
**Coordinator:** [Stage 11 contract](../../../kinet/docs/stage11.md); the package-local milestone/status owns acceptance.

## Dependencies and handoff

[UWS:C08](../../../uws/tabilet/docs/history/status-C08.md); [Udon:M48](../../../udon/tabilet/memory-bank/status-M48.md).
The serial predecessor is a scheduling gate; direct contract and regression impacts are also listed. Every prerequisite must pass its whole review, and required publication must be independently verified before adoption. Record exact accepted/published sources and fixture/build hashes; no Stage 11 acceptance or future pin is claimed yet.

**Downstream:** [Kinet:M46](../../../kinet/tabilet/memory-bank/status-M46.md), [OpenUdon:P09](status-P09.md), [Kinet:M47](../../../kinet/tabilet/memory-bank/status-M47.md). Reconcile every affected consumer against the accepted prerequisite revision before advancing.

## Tasks

| Item | State | Notes |
|---|---|---|
| M98.1 — Extract handoff digest and authority APIs | `[+]` | Expose deliberate public packages for existing handoff, digest, approval and Authority types. Preserve published discriminators, canonicalization and wire bytes; protect the public/private import boundary. |
| M98.2 — Define format-neutral verification boundaries | `[+]` | Expose bounded format-neutral trust inspection/verification types without making synthesis-coupled v2 construction, assessment or simulation orchestration a supported public API. Retain those legacy implementations privately behind unchanged CLI adapters. Public v3 construction/assessment belongs to OpenUdon:P09; include affected mockruntime simulation regression vectors without broadening the M98 API promise. |
| M98.3 — Expose evidence verification | `[+]` | Expose run-evidence and Udon-report wire verification without importing Udon. Add golden/API-surface fixtures for current approvals, broker identities, reports and uncertainty. |
| M98.4 — Qualify and publish public interfaces | `[~]` | Run public standalone tests, boundary guards, wire vectors and affected consumers. Publish accepted source with named authority; the existing CLI and execution path stay available in Phase A. API-surface/import tests must reject an accidental public dependency on internal/synthesize or its legacy construction types. Source/shape reproduction is owned by P09, not M98. |

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

Udon closure/evidence head 4ec2bdf155ced16dbd3303dc6c45f7359631811f was also
independently observed on its unchanged authorized origin/main after normal
source-only push, containing exact accepted runtime ancestry and passed closing
review. This satisfies the full prerequisite source/closure publication gate.

## M98.1 selection — 2026-10-07

Continue from clean owner head 538f7bc9b97c094176c90c0d336207c629ac4301 after
exact accepted/published Udon M48 reconciliation. Its final factual-summary
closure head 496240b0c7c68a2097ce3ecac89ac9759fc4ea63 is independently observed
on unchanged authorized Udon origin/main; runtime/module/binary source stays
da43e57be37af4e18e633558580f740c525f139d. The sole execution owner now selects
M98.1; no other general task row is in progress. Extract only deliberate
format-neutral public handoff/digest/approval/Authority APIs with unchanged
canonicalization/discriminators/wires. Do not expose synthesis-coupled v2
construction or import private Udon; other M98 rows and whole review stay pending.

## M98.1 completion — 2026-10-07

Public `handoff`, `digest`, `authority` and `approval` packages now own the
existing neutral manifests, canonical self/policy digests, approval wire and
value validators. Internal adapters alias/delegate without changing CLI
construction, assessment or execution. Published fixture JSON hashes and
embedded self/policy digests pass; the public dependency guard excludes all
OpenUdon internals and private genelet modules. No module/pin changed.

Verification passed: public and affected internal tests; `go vet ./...`;
`make check` (standalone CLI/runner build, full tests, sibling and APItools
boundary checks); focused public/broker/evidence race tests; tabilet
`check-doc-memory`; `git diff --check`. Existing mock simulation and browser
adapter unit regressions passed in the full suite. No browser runtime/capture
code changed, so no live browser smoke was selected. Full M98 acceptance and
publication remain pending; whole review remains 0/10.

## M98.2 selection — 2026-10-07

M98.1 task commit is 739b862. Select only M98.2: bounded caller-supplied byte
snapshot verification and public snapshot digest; no filesystem discovery,
construction, assessment, simulation, credentials or executor capability.
Retained private v2 CLI orchestration stays intact.

## M98.2 completion — 2026-10-07

`trust.Inspect` verifies the caller's explicit required byte inventory with safe
canonical paths, unchanged manifest self digest, artifact hashes and neutral
policy. It computes the existing package-digest-v1 envelope through public
`handoff.DigestFiles`. It provides no source assessment, construction,
simulation, discovery or I/O; the host owns format-specific required paths,
stable bytes, safe file reads and isolation. Public `wire` provides bounded
strict JSON and lossless-number decoding. Limits and API responsibilities are
documented in docs/public-trust-api.md. Inspection errors are fixed/value-free.

Public/private legacy package identity parity, digest-v1 golden, mutation,
missing/unlisted inventory, path, byte/node/depth, duplicate/unknown JSON,
large-number and cancellation checks passed. Existing mock runtime simulation
and private v2 orchestration regressions passed. Full `make check`,
`go vet ./...`, focused races including trustedrunner/simulation, tabilet
`check-doc-memory` and `git diff --check` passed. No dependency/pin, browser
runtime or installed behavior changed. Evidence verification, whole review
0/10 and publication remain pending.

## M98.3 selection — 2026-10-07

M98.2 task commit fcdd8fb passed all row verification. Select only M98.3:
public neutral report/run-evidence byte verification and exact-attempt
uncertainty fixtures without private runtime imports. Retained browser
execution remains separately pinned; no executor, provider or live call.

## M98.3 completion — 2026-10-07

Public `udonreport` owns existing v2–v5 report validation and conservative v5
observations; private workflow inventory derivation stays private. Public
`runevidence` owns unchanged run/async/signature/browser metadata wires and
bounded non-browser explicit-byte verification. Internal type aliases and
non-browser validation delegate to the public implementation; browser
execution/verification retain their private exact pinned path. No private
Udon dependency or filesystem/credential/executor operation is exposed.

Verification requires all exact referenced report/async bytes and can bind
independent attempt and approved inventory identities. Missing or untrusted
reports remain conservative unknown observations. Embedded-key signature
integrity and supplied-key trust are separate, neither grants authority.
Legacy v1 stays read-only. docs/public-trust-api.md records limits, provenance
and reduction/redaction responsibilities. The public ObserveV5 entry validates
its expected inventory before treating a report as validated.

Published report/uncertainty fixtures, broker run wire golden, approval/broker
downgrade/drift, report bytes/attempt/approved inventory mismatch, explicit
private/public dry-run/async parity and deterministic signature/trusted-key
tests passed. Full `make check`, `go vet ./...`, focused public/report/CLI/runner
races, tabilet `check-doc-memory` and `git diff --check` passed. One test setup
initially omitted the fixture inventory's explicit not_started outcomes; it
was corrected to the existing qualified inventory contract before passing.
All published fixtures/schemas/module pins remain unchanged. Whole M98 review
0/10 and standalone consumer qualification/publication remain pending.

## M98.4 selection — 2026-10-07

M98.3 task commit b52fa1e passed required row checks. Select only M98.4 for
public standalone/import/surface/wire qualification, affected consumers,
persisted pre-publication review, exact source publication under the existing
STG11_SOURCE_PUBLICATION grant, then closing review and normal retirement.
No Kinet push, deployment, private runtime import or live operation.

M98.4 preliminary standalone public tests passed with GOWORK=off, GOPROXY=off.
API surface fixture SHA-256 is
8ac50a0eb84ba7a28ab2fda375bfc536b0df568272017e3f0674b9ca315b1e7e;
it preserves existing exported signatures/types and rejects legacy construction
names. The module graph contains main plus 89 ordinary modules, no replacements.
Seven uncached exact already-selected dependency archives are being acquired
under the existing build-closure grant, without version changes. Full frozen
source qualification, review and publication remain pending.

The owner docs workflow on main changes uses mkdocs gh-deploy --force; release
publication is tag-only and the public browser workflow is manual. Source-only
publication will use a final [skip ci] commit, as already established for UWS
source publication, to avoid the unrelated docs deployment. Local verification
will be recorded independently; no hosted CI result or docs deployment is claimed.
