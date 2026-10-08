# A31 — Transition cleanup

**Stage:** Kinet STG-11, Phase B. **Owner:** OpenUdon.
**State:** Approved planning on 2026-10-06; 3 pending rows, no implementation or acceptance.
**Source baseline:** `7cd7fbb837fb87e1ca4abea2a362790b0f434188` (clean at planning).
**Coordinator:** [Stage 11 contract](../../../kinet/docs/stage11.md); the package-local milestone/status owns acceptance.

## Dependencies and handoff

[Kinet:U14](../../../kinet/tabilet/memory-bank/status-U14.md).
The serial predecessor is a scheduling gate; direct contract and regression impacts are also listed. Every prerequisite must pass its whole review, and required publication must be independently verified before adoption. Record exact accepted/published sources and fixture/build hashes; no Stage 11 acceptance or future pin is claimed yet.

**Downstream:** [Kinet:M48](../../../kinet/tabilet/memory-bank/status-M48.md). Reconcile every affected consumer against the accepted prerequisite revision before advancing.

## Tasks

| Item | State | Notes |
|---|---|---|
| A31.1 — Inventory complete remaining consumers | `[ ]` | Scan the full transitive import/command closure, not just named browser entrypoints. Record consumers of synthesize, workflowintent, elicitor, projectwizard, udonrunner/trustedrunner and legacy HCL support. Distinguish M98 format-neutral public compatibility promises, P09 v3 construction and private legacy synthesis adapters; no deletion may break the retained browser profile. |
| A31.2 — Retire only safe superseded surfaces | `[ ]` | Remove obsolete non-browser adapters/surfaces where the remaining tree and accepted consumers allow it. Retain and document browser-required compatibility code; preserve frozen W8M/Ramen pins and public/private boundaries. |
| A31.3 — Qualify publish and hand off deferred cleanup | `[ ]` | Build/test the complete remaining tree and affected browser compatibility fixtures. Publish under named authority, and give Stage 12 a precise dependency/removal checklist; do not mark blocked removals delivered. Reconcile the narrowed M98 surface and P09 consumer closure into Kinet:M48 before final pin freeze; publication still needs the separately granted envelope. |

## Acceptance and verification

The new primary path has clear ownership and dead code is removed only when safe. Every browser-dependent remainder has an explicit Stage 12 owner and deletion gate.

go test ./...; go vet ./...; make check; API/import-boundary and trust-wire fixtures; affected exact-pin consumer checks; git diff --check. Use owner-required offline browser smoke/qualification only for affected retained browser paths.
Use only disposable roots and fixtures. Preserve published schemas/wires, historic evidence, current runtime capability restrictions and the installed M44 service. Changed v3/package/worker identities require fresh approval; they do not preserve old grants.

## Execution policy

One execution owner, serial execution and task commits under the later confirmed goal. Planning authorizes no code execution, commit, publication or external operation. Source publication requires separately named authority; a status marker or local build is not publication. Consumers must record exact accepted and published prerequisites before adoption. Default checks are offline, credential-free and model-free. No deployment, live ledger migration, real API/model/mail action or registration change.

## Reconciled consumer contract — 2026-10-06

Consume OpenUdon:M98’s narrowed public surface and P09’s v3 APIs; retained browser compatibility remains a Stage 12 deletion gate.

## Persisted review

- Review iteration: **0/10**; not started.
- Closing-review findings: none; the whole-milestone review has not started. Approved intake requirements above remain pending.
- Accepted revision: not available.
- Published revision / artifact evidence: not available.
- Verification: pending implementation; no test result is claimed by this planning record.

After all tasks finish, perform the whole-milestone review with persisted iteration/finding state and fix every P1/P2 before acceptance. Resume an interrupted pass at the same counter. Consolidate current facts, reconcile downstream work and retire under this package’s normal procedure.

## Accepted P09 public-surface lineage — 2026-10-07

P09 accepts qualified/public source a6a3ef010fe27f277f8191204c1c81ea1cc0334b,
SDK v0.1.1-0.20261007094531-a6a3ef010fe2 after closing review 4. Public
packagev3 supplies explicit construction/assessment/source/plan/authority and
read-only v2/v3 history, with public credentialpolicy and runtime-owned trusted
catalog/admission adapters. Existing M98 neutral surface and old wires stay
frozen; no private Udon import or stable legacy v2 synthesis contract enters
public SDK. A31 inventory must retain this full public closure and all pending
Kinet accepted consumers before deleting private legacy/browser compatibility.
[Contract](../../docs/package-v3.md) and [qualification](../../docs/p09-qualification.md)
record exact source/owner/consumer scope. Browser-dependent retirement remains
Stage 12; no row/review starts here before Kinet U14.


## Accepted U14 prerequisite — 2026-10-08

U14 completes all four task units and whole review2 at exact qualified code
bd5b6b8281288aa50f0ceb5e0e6c55e0b1fc2ac9. [Kinet qualification](../../../kinet/docs/u14-qualification.md)
records owner native diagnostics/actual A15 progress, exact transient conversion
question/source/artifact diffs, independent verified/packaged-HCL provenance,
read-only runtime/history/fresh authority/disabled schedule and desktop/mobile/
keyboard/focus/viewport/embed behavior. Review1 cache-policy P2 is fixed and
verified; no open P1/P2 or waiver. Current author0e1aafa/private10c7a063/browser
c2f161d7 and full unchanged independent pins remain in qualified manifests.
No Kinet publication, live conversion or deployment authority follows.

Consume existing qualified public SDK P09 and W19/M47 contracts; UI adds no
runtime leaf capability, authority transfer or automatic fallback/replay. A31
retains the full public packagev3/trust closure, current Kinet public author/
private exec imports and separately pinned browser dependencies. Safe removal
requires complete transitive owner inventory; browser-gated adapters stay for
Stage12. M48 freezes all accepted/exact-published sources and independent pins,
qualifies final release including this UI and current deletion/pre-v3 exact-M44
checkpoint rollback, then creates the separate concrete deployment proposal.
A31/M48 rows and whole reviews remain pending; no prerequisite reconciliation
marks those tasks delivered. Complete all18 endpoint remains authorized.
