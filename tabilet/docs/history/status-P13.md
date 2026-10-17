# Retired milestone P13 - Public saved-session authority and fallback evidence

**Milestone.** P13
**Outcome.** completed
**Retired.** 2026-10-17
**Source status.** tabilet/memory-bank/status-P13.md
**Source specification.** tabilet/memory-bank/milestone.md#p13--public-saved-session-authority-and-fallback-evidence
**Evidence.** 95e066ea526d91c9bdd60faa78bab1763c149294
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Fifteen recorded invocations preserve setup/red/environment failures; focused/compatibility/full-owner Make/vet/public compiled closure PASS. Exact published source 5f2104652a7ef40e60cc04ce70e4b63206864f87, complete1697Git/ZIP/extracted/nativeH1 proof PASS; normal closure publication follows before Kinet:M53.22. No signed checksum-database or physical claim. docs/p13-qualification.json retains all evidence.
**Consolidated into.** docs/browser-package-v3.md, tabilet/memory-bank/architecture.md, tech-stack.md, lessons.md; pending A32 preservation and Kinet:M53.22 adoption.

## Milestone specification

`````markdown
### P13 — Public saved-session authority and fallback evidence

**Stage/owner:** Kinet STG-12 Phase B; OpenUdon. Required corrective producer.
**Depends on:** accepted P10/P11/P12 contracts; immutable Kinet M53.21 and
adopted native Udon source `46e526a44d3cadcbcd8eccff9aae8d8bb0aa9f07` as
read-only compatibility inputs. No dependency on completed Kinet:M53.
**Downstream impacts:** Kinet:M53.22 and original M53.4; OpenUdon:A32.
**Parallel-safe:** no; serialize its source/contract adoption. Coordinator-only
ledger writes; implementation in an isolated `goal/P13` worktree under the
resolved Stage 12 goal. Capture actual primary/ref/full baseline before dispatch.
**Write set:** public packagev3/browsercontract/runevidence implementation and
focused tests, ordinary public contract docs/qualification, and coordinator-owned
P13/shared-memory planning records. No private-runtime import, legacy removal,
browser harness, consumer pin change or sibling implementation.
**Contracts read:** existing public config/receipt/authority/report schemas,
immutable P12 and native fallback source, Kinet approval/session consumer snapshot.

M53 contract/final-qualification recovery review, 2026-10-17.
Source priorities were not supplied. Revalidated at Kinet
`68c3c08a03b063789e70f1e37039c09c3fe7f598` with existing unfinished M53.4
fixtures/records, and OpenUdon `53191bc0d9fb6bfdae130e3af41f28beaa3e39b2`
(clean primary). Its `packagev3/browser_authority.go` matches the adopted P12
module. Human approval: “I approve this contract-repair and
qualification-replanning work first.” This is review intake, not a task or
whole-milestone review iteration.

F01 (confirmed P2) repairs `packagev3/browser_authority.go`'s unconditional
saved-session expiry refusal while preserving complete immutable config/hash,
receipt, current execution clock and calls. The authority is review metadata,
not saved-state access. Missing/expired payload may select fresh only before
dispatch, with both observed reuse/save rights false and unchanged binding.
Validate independent report/evidence attenuation in `runevidence/browser.go`,
including forged retained rights, fallback-after-send and save-on-fallback
negative cases. Do not add a new schema or delete a guard without the full
contract proof. Kinet retains custody/revocation and native runtime access.

**Acceptance:** fixed-time before/at/after-expiry authority and receipt tests;
current execution deadline/approval, creation time, exact hash/calls and unchanged
generation checks; missing/expired fresh result/evidence validation; forged
rights/save/post-send rejection; unchanged valid fresh/live reuse and existing
public golden/wire/history/default behavior. Actual Kinet adoption is M53.22,
physical proof M53.4. Public owner tests never import private Udon.
**Expensive checks:** cheapest offline deterministic reproduction first, then
focused affected packages (30-second target/2-minute cap); race only for changed
concurrency (5-minute cap); one full owner `make check`/required closure candidate
within 20 minutes after focused proof. Use cached tools/modules with network off,
no browser/display/snapshots/new caches. Cheap local diagnostics are separate
from counted expensive checks; preserve failed attempts and stop gates.
**Review/publication:** normal task reviews plus fresh whole review, initially
0/10; no substitution for M53 review0. Exact ordinary published producer proof
is a consumer prerequisite, performed only under independently matching existing
grants or a separately prepared/approved scope. No exhausted P12 grant transfer.
**Tasks:** [status-P13.md](status-P13.md), three rows; all three rows pass fresh task review1; whole review1/10 PASS.

When `tabilet/stages.md` exists, it records stable `STG-` IDs, the current
stage, and provisional later ideas. Give each active milestone specification
its stage ID. An absent stages file means one implicit stage. Stage labels do
not determine task readiness or prove milestone acceptance.

Each active milestone specification defines scope, acceptance criteria, and
optional boundaries and parallel safety: `Depends on`, `Downstream impacts`,
`Write set`, `Contracts read`, and `Parallel-safe` (defaulting to `no`). A
milestone is parallel-safe only when unordered against other active work, its
write set is disjoint, and its inputs are frozen. Parallel safety never prunes
downstream impact reconciliation.

`````

## Status record

`````markdown
# P13 — Public saved-session authority and fallback evidence

**State:** P13.1 accepted at `7356d428a406100105d843bbf526ae1b5867d47d`/task review1; P13.2 complete at fresh task review1; P13.3 complete at fresh task review1. Whole review1/10 PASS; accepted locally, normal closure publication pending.
**Baseline:** `53191bc0d9fb6bfdae130e3af41f28beaa3e39b2`, clean primary at intake.
**Coordinator:** Kinet Stage12; one coordinator writes all ledgers. Audit disabled.
Read the owning [specification](milestone.md#p13--public-saved-session-authority-and-fallback-evidence)
for scope, dependencies, verification and action requirements. P13 is distinct
from the historical Kinet physical attempt P13 and its narrow recovery exception.

M53 contract/final-qualification recovery review, 2026-10-17.
Source priorities were not supplied. Revalidated at Kinet
`68c3c08a03b063789e70f1e37039c09c3fe7f598` with existing unfinished M53.4
fixtures/records, and OpenUdon `53191bc0d9fb6bfdae130e3af41f28beaa3e39b2`
(clean primary). At intake, `packagev3/browser_authority.go` matched the adopted
P12 module. Human approval: “I approve this contract-repair and
qualification-replanning work first.” This is review intake, not a task or
whole-milestone review iteration.

## Tasks

| Item | State | Notes |
|---|---|---|
| P13.1 — Repair authority and observed fallback rights with deterministic regressions | `[+]` | Reproduce fixed-clock expiry rejection, then preserve immutable configuration and current execution approval while permitting native pre-dispatch missing/expired-to-fresh attenuation. Verify complete report/evidence rights, no saved access or save on fallback, post-send refusal and all exact binding/hash/clock negatives. Implementation and focused regression coverage form one coherent unit; no browser/private import/schema widening. |
| P13.2 — Qualify public worker and consumer contract compatibility | `[+]` | Exercise finalized receipt/Derive/Check and Observe/Verify transitions in disposable offline consumer fixtures. Preserve existing public golden bytes/history/defaults and expose exact compatibility expectations for Kinet currentness/native adoption; fixtures do not substitute for M53.22 actual published adoption. |
| P13.3 — Qualify exact closure and hand off the producer | `[+]` | Required full owner checks, public/private closure, immutable source compatibility evidence and exact publication/ordinary-module handoff under matching authority. Reconcile M53.22 and A32; preserve every M53.4 obligation. Fresh whole review remains mandatory before acceptance/retirement. |

## Verification and review

Fifteen local verification invocations are retained in the qualification.
All three task reviews1 pass; whole review1/10 passes.
Owner check1 failed only under wrapper umask077; focused correction11 passes,
and independently reviewed owner check2 passes59.10s under022. Public compiled
closure/module integrity13 passes5.82s. Source publication and ordinary module14
pass exact1697file/nativeH1 proof; P13.3 task review1 passes. Whole review1 passes.
Do not reset originating M53.4 counts or infer physical acceptance. Requirements,
status and planning approval do not grant new external actions. The original
goal's model/integration/commit policies and mandatory fresh reviews apply.

## Current lease and execution

Primary `/home/peter/Workspace/openudon`, integration `refs/heads/main`,
branch `goal/P13`, worktree `../openudon.goal/P13`, full dispatch base
`7a7e24973b016a4772634313e2ce8b743371f7aa`. Coordinator owns all status/Git/shared-memory writes.
P13.1/.2 used offline library and compatibility checks. P13.3 used only the
exact approved source publication/module acquisition; acquisition ended at
first verified success. No browser ran.
Original M53.4 remains blocked at physical failure3. Audit disabled.

## Historical P13.1 verification checkpoint

Seven local verification invocations are recorded in [qualification](../../docs/p13-qualification.json): two environment setup failures (no tests), one compile-only pass, two deterministic red checks (including a corrected positive-fixture equality assertion), four-regression green3.65s and all affected package tests PASS20.15s. Exact six-file green inputs are frozen; fresh task review1 PASS, no findings across all five lenses; whole review0/10. No concurrency change requires race. This is public-library behavior, not Kinet worker/native adoption or physical proof. Audit disabled.

## Historical P13.2 compatibility checkpoint

Public compatibility batch invocation8 PASS1.76s and scoped vet9 PASS1.87s/no diagnostics; exact source/test/contract inputs remain frozen. Existing strict schema, wire/golden/history/default and public boundary fixtures pass. P13.1 behavioral evidence is reused without repeating it. These are worker-facing public API fixtures, not an actual Kinet worker/native admission: M53.22 owns that after published adoption. Fresh task review1 PASS across all five lenses/no findings; no new implementation or physical evidence is claimed.

## P13.3 owner check1

Owner full check1 (local verification10) FAIL123.27s/exit2, original command positively waited. One legacy authoringengine permission fixture expected an unsafe root but the wrapper umask077 made it private. All other owner packages passed; trailing sibling/boundary targets were not reached. This historical failure/log is retained; focused normal-umask proof11 and independently reviewed owner successor12 subsequently passed without source changes or extra browser authority.

## P13.3 current owner/closure checkpoint

Full owner successor12 PASS59.10s/exit0 at clean primary `5f2104652a7ef40e60cc04ce70e4b63206864f87`; original command waited. All owner tests, required sibling check and APItools boundary pass. Independent readiness confirmed focused correction11 PASS1.79s/0.079pkg; failure10 is retained and no source changed. Closure13 PASS5.82s: 373 compiled public packages, 372 export hashes, 923 nonstandard source hashes, no private runtime/owner-internal package or replacement; native Go module integrity passes. Module locks match accepted P12. Actual published P13 module14 is verified and its acquisition ended; P13.3 fresh task review1 passes. Whole review1 passes; normal closure publication is required before consumer dispatch.

## Current handoff boundary

Exact qualified source `5f2104652a7ef40e60cc04ce70e4b63206864f87` is independently published. Its ordinary module `v0.1.1-0.20261017191049-5f2104652a7e` passes all 1,697 Git/ZIP/extracted file comparisons and independent native archive/go.mod H1 proof. The human-approved acquisition succeeded once in6.43s; first-success termination ends the remaining invocation (1used/1unusedterminated/0remaining). No signed checksum-database claim. P13.3 fresh task review1 passes, including independent complete module/closure proof; whole review1/10 passes, so no milestone acceptance/retirement. Kinet:M53.22 still waits for the accepted producer. All M53.4 physical limits/unknowns remain unchanged; audit disabled.

## Whole-milestone review

Iteration1/10 PASS with a fresh read-only Sol6.1/xhigh reviewer covering all five lenses against complete P13 scope at `95e066ea526d91c9bdd60faa78bab1763c149294` plus the disclosed qualification records. No unresolved findings. Task reviews1/1/1 and all original M53 counters remain preserved. Normal closure publication precedes consumer dispatch.

Required full-owner `go vet ./...` invocation15 PASS10.01s/exit0/zero diagnostics within this same whole review1. The earlier scoped vet9 is retained; source/tests, deadline and cache/network bounds did not change.

## Accepted producer and normal closure

All three rows and fresh whole review1 PASS across allfive lenses, no unresolved findings. Full-owner vet15 PASS10.01s/zero diagnostics fills the one missing required gate within that same iteration. Complete verification and all failed/setup/red outcomes remain in docs/p13-qualification.json. Exact accepted source is `5f2104652a7ef40e60cc04ce70e4b63206864f87`, ordinary module `v0.1.1-0.20261017191049-5f2104652a7e`; all1697Git/ZIP/extracted bytes and nativeH1 pass independently. The module acquisition ended1used/1unusedterminated/0remaining. Consolidated source contract, stack and lessons preserve requested-versus-observed rights and qualification lineage. The coordinator retires this complete status/specification, publishes normal closure under the existing exact global OpenUdon source grant, independently verifies it, then dispatches Kinet:M53.22. No physical acceptance, deployment, audit activation or M53 counter change follows.
`````
