# P13 — Public saved-session authority and fallback evidence

**State:** P13.1 complete at fresh task review1; P13.2/.3 pending. Whole review0/10.
**Baseline:** `53191bc0d9fb6bfdae130e3af41f28beaa3e39b2`, clean primary at intake.
**Coordinator:** Kinet Stage12; one coordinator writes all ledgers. Audit disabled.
Read the owning [specification](milestone.md#p13--public-saved-session-authority-and-fallback-evidence)
for scope, dependencies, verification and action requirements. P13 is distinct
from the historical Kinet physical attempt P13 and its narrow recovery exception.

M53 contract/final-qualification recovery review, 2026-10-17.
Source priorities were not supplied. Revalidated at Kinet
`68c3c08a03b063789e70f1e37039c09c3fe7f598` with existing unfinished M53.4
fixtures/records, and OpenUdon `53191bc0d9fb6bfdae130e3af41f28beaa3e39b2`
(clean primary). Its `packagev3/browser_authority.go` matches the adopted P12
module. Human approval: “I approve this contract-repair and
qualification-replanning work first.” This is review intake, not a task or
whole-milestone review iteration.

## Tasks

| Item | State | Notes |
|---|---|---|
| P13.1 — Repair authority and observed fallback rights with deterministic regressions | `[+]` | Reproduce fixed-clock expiry rejection, then preserve immutable configuration and current execution approval while permitting native pre-dispatch missing/expired-to-fresh attenuation. Verify complete report/evidence rights, no saved access or save on fallback, post-send refusal and all exact binding/hash/clock negatives. Implementation and focused regression coverage form one coherent unit; no browser/private import/schema widening. |
| P13.2 — Qualify public worker and consumer contract compatibility | `[ ]` | Exercise finalized receipt/Derive/Check and Observe/Verify transitions in disposable offline consumer fixtures. Preserve existing public golden bytes/history/defaults and expose exact compatibility expectations for Kinet currentness/native adoption; fixtures do not substitute for M53.22 actual published adoption. |
| P13.3 — Qualify exact closure and hand off the producer | `[ ]` | Required full owner checks, public/private closure, immutable source compatibility evidence and exact publication/ordinary-module handoff under matching authority. Reconcile M53.22 and A32; preserve every M53.4 obligation. Fresh whole review remains mandatory before acceptance/retirement. |

## Verification and review

No implementation or test has run for this new milestone. Expensive attempts0;
task reviews0; whole review0/10. Reuse valid immutable prerequisite evidence.
Do not reset originating M53.4 counts or infer physical acceptance. Requirements,
status and planning approval do not grant new external actions. The original
goal's model/integration/commit policies and mandatory fresh reviews apply.

## Current lease and execution

Primary `/home/peter/Workspace/openudon`, integration `refs/heads/main`,
branch `goal/P13`, worktree `../openudon.goal/P13`, full dispatch base
`7a7e24973b016a4772634313e2ce8b743371f7aa`. Coordinator owns all status/Git/shared-memory writes.
P13.1 uses deterministic offline red→green tests only; no browser or network.
Original M53.4 remains blocked at physical failure3. Audit disabled.

## P13.1 verification checkpoint

Seven local verification invocations are recorded in [qualification](../../docs/p13-qualification.json): two environment setup failures (no tests), one compile-only pass, two deterministic red checks (including a corrected positive-fixture equality assertion), four-regression green3.65s and all affected package tests PASS20.15s. Exact six-file green inputs are frozen; fresh task review1 PASS, no findings across all five lenses; whole review0/10. No concurrency change requires race. This is public-library behavior, not Kinet worker/native adoption or physical proof. Audit disabled.
