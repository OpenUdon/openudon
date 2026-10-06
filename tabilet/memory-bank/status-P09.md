# P09 — Package v3

**Stage:** Kinet STG-11, Phase B. **Owner:** OpenUdon.
**State:** Approved planning on 2026-10-06; 5 pending rows, no implementation or acceptance.
**Source baseline:** `7cd7fbb837fb87e1ca4abea2a362790b0f434188` (clean at planning).
**Coordinator:** [Stage 11 contract](../../../kinet/docs/stage11.md); the package-local milestone/status owns acceptance.

## Dependencies and handoff

[UWS:C09](../../../uws/tabilet/memory-bank/status-C09.md); [APItools:M82](../../../apitools/tabilet/memory-bank/status-M82.md); [OpenUdon:M98](status-M98.md); [Kinet:W17](../../../kinet/tabilet/memory-bank/status-W17.md).
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

## Persisted review

- Review iteration: **0/10**; not started.
- Closing-review findings: none; the whole-milestone review has not started. Approved intake requirements above remain pending.
- Accepted revision: not available.
- Published revision / artifact evidence: not available.
- Verification: pending implementation; no test result is claimed by this planning record.

After all tasks finish, perform the whole-milestone review with persisted iteration/finding state and fix every P1/P2 before acceptance. Resume an interrupted pass at the same counter. Consolidate current facts, reconcile downstream work and retire under this package’s normal procedure.
