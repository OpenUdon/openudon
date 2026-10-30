# Retired milestone P17 - Prove runtime function package contracts

**Milestone.** P17
**Outcome.** completed
**Retired.** 2026-10-30
**Source status.** tabilet/memory-bank/status-P17.md
**Source specification.** tabilet/memory-bank/milestone.md#stg15-p17
**Evidence.** 874ef77bcf3ee3416ad2da68b10a1eb249d1d88f
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Offline package tests, race, isolated public-boundary and full check passed; task reviews 2/2/1. Retained qualification is `.openudon-run/p17-qualification-20261030/`; the original failed boundary log is a labeled transcription. Post-commit proxy materialization and consumer adoption remain downstream gates.
**Consolidated into.** [product](../../memory-bank/product.md), [architecture](../../memory-bank/architecture.md), [lessons](../../memory-bank/lessons.md).

## Milestone specification

`````markdown
<a id="stg15-p17"></a>

### P17 — Prove runtime function package contracts

**Owner/stage.** OpenUdon; corrective prerequisite for Kinet STG-15, parallel-safe no. **Outcome.** Package assessment and expression checking can independently prove supported inline runtime functions against independently verified catalog and shapes, while unknown, forged, incomplete or mismatched evidence remains non-compatible. A compatible assessment grants neither runtime admission nor execution authority.

**Evidence and lineage.** Kinet M67.2 runtime assessment diagnosis, F01 (source priority not supplied; local P1, confirmed), revalidated at OpenUdon `874ef77bcf3ee3416ad2da68b10a1eb249d1d88f` with Kinet partial uncommitted reproducer. `packagev3/build.go` marks unbound profiles unproved; `packagev3/expression_contracts.go` cannot resolve their output. Existing retired records remain frozen. The owner explicitly requested this plan and local implementation/reviews on 2026-10-30; the selected Kinet goal's original boundaries and counters otherwise remain.

**Scope and tasks.** P17.1 implements independently verified runtime function input/output contract assessment and expression projections with exact source/catalog, revision, selector, complete shape and request/response identity checks. Keep source-bound and runtime-extension semantics distinct; unknown profiles and incompatible or missing proof must refuse conservatively. P17.2 verifies build and verification/reassessment parity, valid synthetic composed cases and negative cases for forged/stale revision, selector, source, shape, incomplete mappings, output references and malformed inputs; preserve existing API/browser behavior and runtime admission. P17.3 runs required public-boundary and full offline verification, prepares exact local module handoff and reconciles Kinet consumer expectations. Kinet owns actual author/executor pairing and sandboxed integration; no private executor imports or live effects.

**Acceptance and verification.** Independently verified complete runtime metadata allows a genuinely compatible public package with supported function request and output references; false completeness and identity/shape drift fail closed. Build and verify reproduce assessment, existing compatibility and authorization remain intact. Focused packagev3 tests, affected race tests, `make public-boundary` and `GOWORK=off make check` using existing cached dependencies and toolchain, no network. Persist task reviews and whole review at Sol/xhigh; ten-iteration milestone limit, no counter reset. Kinet consumes the exact accepted local commit via a retained local proxy, with no publication claim. The coordinator is sole ledger writer, local milestone commit only; no push, downloads, deployment, live services/models, physical browser, sudo or SSH; audit stays disabled.

**Downstream.** Accepted exact local P17 revision → Kinet M67.2, then M67.3/.4 and the existing ordered horizon; Kinet retains its original verification and review requirements.
`````

## Status record

`````markdown
# P17 — Prove runtime function package contracts

**Owner:** OpenUdon; Kinet-coordinated STG-15 corrective prerequisite. **State:** Accepted; 3 tasks complete; retirement underway. **Parallel-safe:** no.
**Specification:** [P17](milestone.md#stg15-p17). **Baseline:** `874ef77bcf3ee3416ad2da68b10a1eb249d1d88f`; clean worktree before planning.
**Approval:** Owner explicitly requested preparation and execution of the OpenUdon local corrective plan and reviews, then resumption of the existing goal. Existing local-only restrictions, review models and limits remain.
**Evidence:** Kinet M67.2 runtime assessment diagnosis, F01; source priority not supplied, local P1, confirmed against the baseline and Kinet's partial uncommitted reproducer. P17 is a prerequisite, not a substitute for Kinet acceptance or its review.

## Tasks

| Item | State | Notes |
|---|---|---|
| P17.1 — Prove runtime input and output contracts | `[+]` | Task review 1 found P2: whole-body expression could be compatible despite Udon requiring an explicit body object. Corrected and task review 2 passed with no findings; offline packagev3 tests passed. |
| P17.2 — Verify reproduction and refusal parity | `[+]` | Task review 1 found P2: nondeterministic output diagnostics could break Build/Assess/Verify parity. Corrected with deterministic ordering and 24-cycle regression; task review 2 passed with no findings. |
| P17.3 — Qualify and hand off exact local revision | `[+]` | Race, public-boundary and full check passed; task review 1 passed. Exact local-module handoff procedure is prepared; materialization and independent resolution follow the reviewed closure commit before M67 resumes. |

## Review and verification

Whole-milestone review 1/10 passed with no P1, P2 or lower findings across all five lenses. Task reviews passed 2/2/1; their two P2 findings were corrected and re-reviewed. The review covered the uncommitted implementation atop the full baseline, not a future commit.

Race `go test -race ./packagev3` passed (170.092s), SHA-256 `b6ff92a639ae574be11521f32b666df3b5f30732772bf5d8a15f3fe7c1c66535`. Isolated source snapshot `make public-boundary` and `GOWORK=off make check` passed, logs SHA-256 `b6906a3ddad1d437de78c8c8325e0c5f2debe5f3552264b5e10159f6d4a91fb3` and `17b6a15f826c40f9e0f2dcb1f5f0b12c2b677f641e3e9c1b01be83a1cfca5056`. Retained evidence: `../../.openudon-run/p17-qualification-20261030/`. The original worktree boundary attempt failed due to retained ignored historical files; its raw log was overwritten inadvertently, so only a labeled transcript remains. The successful snapshot contained every tracked baseline file plus exact P17 source/test/fixtures; independent review verified its coverage. All reported original processes settled.

The post-commit exact local module archive, proxy, isolated resolution and Kinet adoption have not yet occurred; they are downstream handoff gates before Kinet M67 resumes. Their procedure is retained as `handoff-procedure.txt` SHA-256 `18f9348a676d121a097e90be1a1a135ea6c5844ed0e1bb8bdfd97b9731828fab`. No source publication, runtime admission or execution is claimed. Preserve the interrupted Kinet M67.2 work and its task review counter.
`````
