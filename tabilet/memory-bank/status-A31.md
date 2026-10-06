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
