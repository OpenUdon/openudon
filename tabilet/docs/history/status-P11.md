# Retired milestone P11 - Browser output binding correction

**Milestone.** P11
**Outcome.** completed
**Retired.** 2026-10-13
**Source status.** tabilet/memory-bank/status-P11.md
**Source specification.** tabilet/memory-bank/milestone.md#p11--browser-output-binding-correction
**Evidence.** b5fba07aae9b50338ae44345be13eb0b99b5387b
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Focused/race/vet/static/tidy/verify/full Make PASS; public compiled/dependency closure; exact published Git/ZIP/extracted files/native H1; docs/p11-qualification.json and docs/p11-publication.json. Normal closure publication checked afterward before dependents.
**Consolidated into.** tabilet/memory-bank/product.md, architecture.md, tech-stack.md, lessons.md; pending A32 preservation and Kinet W20/M52 handoff.

## Milestone specification

`````markdown
### P11 — Browser output binding correction

**Stage/owner.** STG-12 corrective public-package work; OpenUdon.
**Lineage.** Accepted/retired P10; W20 browser-binding review F01.
**Goal.** Make native browser named-output expression resolution consistent
with exact M33 output shapes and C10 binding, without consumer schema rewriting.
**Depends on.** Accepted/published OpenUdon:P10, Browsertools:M33 and UWS:C10.
**Downstream impacts.** Kinet:W20, Kinet:M52 and their dependent release chain.
**Scope.** Native `packagev3` expression/assessment output-reference resolution;
focused regression/compatibility tests, ordinary publication proof and handoff.
Preserve complete native declarations, numeric lexemes, escaped pointers,
unknownness, HTTP behavior, authentication security diagnostics and native
formats. No private Udon import, new browser runtime or consumer-authority claim.
**Acceptance.** Actual M33/C10/P10 operation exports, step exports and downstream
references resolve consistently; missing or malformed outputs refuse. HTTP
fixtures remain unchanged. Historical package bytes remain readable and no
historical approval migrates; document any generated-proof compatibility
change explicitly rather than rewriting immutable fixtures.
**Verification.** Cheap offline fixture/readiness checks first; focused pure Go
regressions target30s/cap2m, affected race cap5m, scoped vet/static cap2m.
Run the existing full offline regression/closure checks once per unchanged
closing candidate, with20m outer cap and honest timing. Adapted immutable
consumer fixture/build checks precede source publication; W20/M52 separately
prove actual adoption of the accepted published revision. No browser, display,
supervisor daemon, snapshot, new copied cache or automatic dependency download.
**Write set.** OpenUdon implementation/tests/ordinary docs and required manifests
within this correction; coordinator alone owns ledgers/shared memory/closure.
**Contracts read.** Exact published M33/C10/P10, immutable Kinet source2 manifest
and synthetic fixture hashes. No live changing consumer checkout as evidence.
**Parallel-safe.** no. Serialize against the dependent W20 recovery lease.
**Tasks/review.** [status-P11.md](status-P11.md),4rows,whole review1/10 passed; all task rows qualified.

AUTHORIZATION_REQUIREMENTS:
  git.commit:
    via: goal-policy
    executor: coordinator
  git.push:
    via: explicit
    executor: coordinator
    scope:
      repository: /home/peter/Workspace/openudon
      remote: git@github.com-tabilet:OpenUdon/openudon.git
      ref: refs/heads/main
      mode: fast-forward
      invocation_limit: 2
`````

## Status record

`````markdown
# P11 — Browser output binding correction

**Approved P11 correction — 2026-10-13.** The user approved the proposed
OpenUdon:P11 browser output binding correction and its planning, local execution,
bounded publication and ordinary published-module acquisition scopes. Source:
W20 browser-binding review F01; source priority not supplied, local P2, confirmed.
Published M33/C10/P10 tests show named `result_count` output binding compatible
in UWS but P10 assessment incompatible (`binding.output_reference_incompatible`
and `binding.output_missing`). Revalidation: Kinet
`8d318aab9f435fe65e716366208511bcd23e43c3` with uncommitted W20.4 source2,
OpenUdon `78f8609d2e21976fa1311aa914cf2e4170b895ef` and exact published modules.
P11 is new work; accepted P10 history and all existing row outcomes/review
counters remain unchanged. This intake starts no closing review. Audit disabled.

**State:** P11.1–.3 complete; P11.4 complete; normal closure publication follows. Whole review1/10 PASS.
**Source baseline:** `78f8609d2e21976fa1311aa914cf2e4170b895ef` before planning.
**Coordinator:** [Stage12](../../../kinet/docs/stage12.md).
**Depends on:** accepted/published OpenUdon:P10, Browsertools:M33 and UWS:C10.
**Downstream:** Kinet:W20 and Kinet:M52.
**Authority:** requirements belong in the [owning specification](milestone.md#p11--browser-output-binding-correction).
The approved goal request owns effective scopes; this status grants nothing.

## Tasks

| Item | State | Notes |
|---|---|---|
| P11.1 — Freeze named-output regressions | `[+]` | Reproduce exact M33/C10/P10 mismatch with immutable synthetic action bytes; operation and step exports and downstream consumers, missing/escaped/nested references and unchanged HTTP controls. Pure offline readiness first. |
| P11.2 — Correct native output resolution | `[+]` | Align browser response expressions and assessment references with reproduced named output declarations. Preserve full values, numeric spellings, unknownness and generic HTTP semantics. Do not rename shapes or suppress diagnostics in Kinet. |
| P11.3 — Qualify compatibility and public closure | `[+]` | Focused/race/vet/static/full offline checks, immutable historical readability and explicit generated-proof compatibility decision, public-only compiled/module closure and adapted frozen consumer fixture checks. No real browser or private-runtime imports. |
| P11.4 — Review, publish and hand off | `[+]` | Persist bounded independent whole review; coordinator publishes reviewed source/closure by at most two fast-forward pushes under the exact scope, verifies ordinary published module revision/files/H1 and exact remote closure, consolidates and retires. W20/M52 adopt only accepted/published proof; no prior grant transfers. |

## Acceptance and verification

Use the owning specification. All four rows and whole review qualify before
closure. Publication verifies exact immutable revisions; no directory/workspace
substitution or changing consumer inputs. Producer fixture proof and real
consumer adoption are separate. Required unperformed proof prevents closure.
Audit disabled; no browser/live/deployment/general sudo/SSH authority.

## Review

Whole-milestone review:1/10, PASS; no P1/P2/P3. Intake is not review1.

## P11.1 verified characterization — 2026-10-13

Exact immutable action SHA56b48d53… and independent actual M33 Build/Verify/C10
controls preserve named outputs and complete declarations.29subcases reproduce
P10 operation/step/workflow exports, typed downstream ownership, invalid/missing/
escaped/nested references and unchanged HTTP behavior through Build/Assess/Verify.
The incompatible expectations explicitly characterize the defect; P11.2 must
replace valid cases with corrected support. No production or frozen fixture
changed. Coordinator source review passes; focused combined checks pass3.261s
(4.38s wall), earlier focused1.919s and malformed.205s; formatting passes.
All are offline pinned checks with120s outer bounds. Whole review stays0/10.

## P11.2 task review1 started — 2026-10-13

Native correction source1 SHA256 `79e89ff4f55d2f35d297760d69301cfcf11e1fc32bc315e690108bdfe9a0bad6` binds6files.
Focused output/ownership/HTTP/numeric tests pass4.451s, race30.080s,
vet3.14s, static2.54s and formatting pass. Root starts independent full task
review1 before row completion; whole P11 review stays0/10. Separate old P10
fixture preserves readable historical bytes and strict stale-assessment refusal.
All source writes stopped; no publication or physical action. Audit disabled.

### P11.2 task review1 finding

Full independent task review1 found one P2, P11.2-R1-F01: ranging
`op.Outputs` collects distinct browser output diagnostics in random order,
changing canonical proof bytes and potentially failing unchanged Verify.
Sort browser action output keys before reference collection while preserving
HTTP behavior; add repeated heterogeneous Build/Assess/Verify regression.
No P1/P3 or other native/history/authority defects found; all six source hashes
match before/after. Finding is unresolved before correction. Row2 remains
in progress; whole review0/10. Audit disabled.

### P11.2 correction and task review2 started

P11.2-R1-F01 is corrected by browser-action-only sorted output keys; HTTP
ordering remains unchanged. Heterogeneous diagnostics and32repeated native
Build/Assess/Verify rounds retain identical canonical proof/files/package IDs.
Focus5.802s, HTTPgolden.165s, race22.897s, vet/static/format pass.
Only build and regression files changed; other4source1files remain exact.
Source2 SHA256 `3674d8d6b12a7947bd4b2a32a6977419f49c6812c5f377cf7ef1c11465fc51b8` binds6files for full task review2,
which root starts now. Finding awaits independent confirmation; whole0/10.
No publication or physical action; audit disabled.

### P11.2 task review2 PASS and verified correction

Full independent corrected task review2 passes with no P1/P2/P3 findings;
P11.2-R1-F01 is resolved. Source2 and6files/79553bytes match before/after.
Native named outputs, ownership/timing, malformed/scalar refusal, unknownness
and complete declarations remain consistent; HTTP and authentication controls
are preserved. Historical inspection stays read-only; stale assessment refuses
current verification, and no old approval transfers. Focused/race/vet/static/
format and exact HTTP golden pass at the recorded bounds. This qualifies row2
only; P11.3/whole qualification and source publication remain pending.
Whole review remains0/10; audit disabled.

## P11.3 offline qualification PASS — 2026-10-13

Qualification baseline `22beb2a7b7ff9661b991423a5b54c86d1b4f9f7f`
includes the new frozen Kinet adapter test; the qualification document was
written afterward. [Record](../../docs/p11-qualification.json) SHA256
`e075575a43c1aabd04568d8d748512aa3c264417fc691c09caf66a20b3748d0b`
records focused/public boundary, vet/static, tidy/verify and compiled closure
checks. Full owner Make attempt1 passed: actual command177.85s; invocation
intent to receipt persistence322.0612s, both within1200s. No unchanged rerun.
All1673 inventoried inputs and24 frozen consumer files remained exact. Native
H1 checks cover90 dependency archives;373 compiled packages,372 exports and
1812 source fields verified with no replacements/private Udon imports. Twenty
legacy goldens stay immutable. Adapted producer fixture proof does not replace
ordinary published-module adoption by W20/M52. Whole review remains0/10;
publication and acceptance pending. Audit disabled; no physical browser action.

## Whole P11 review1 started — 2026-10-13

Root starts the independent deep review of the complete milestone diff from
lease base `ff841e9c5d01a783205a58a699adcc4fafefa472` through
`634a882e670cfb8c0877f45a5e2b012b17b4809e`, with this row/review selection
included uncommitted. All implementation writes are stopped. Review covers
native semantics, authority/history compatibility, deterministic proof,
regression coverage and qualification/compiled closure evidence. Prior task
reviews are retained; this is whole iteration1, not their continuation.
Publication remains unused0/2. Audit disabled.

## Whole P11 review1 PASS — 2026-10-13

Independent semantics/security/compatibility and tests/documentation/evidence
lenses reviewed the complete milestone; both pass with no P1/P2/P3 findings.
All9 frozen files/104361bytes matched before/after. Native named outputs,
ownership, malformed/scalar refusal, unknownness, deterministic diagnostics,
HTTP/auth controls and immutable historical readability remain qualified.
Actual dependency ZIP/H1/extracted bytes, compiled source/exports,20goldens and
24 frozen consumer inputs match. Make process/receipt timing is honest and
its shim invokes the freshly built candidate. No unchanged full rerun.
This passes whole iteration1; row4 remains in progress for required exact
publication/module proof and consolidation/retirement. Audit disabled.

## Reviewed source publication and ordinary module proof PASS

Coordinator source push1/2 settled successfully and independently observed
remote refs/heads/main equals `b5fba07aae9b50338ae44345be13eb0b99b5387b`. Ordinary direct Go VCS resolves
`v0.1.1-0.20261013095009-b5fba07aae9b`; all1684 regular Git/ZIP/extracted
files are equal, full Origin.Hash is exact, and native module H1
`h1:HmshTq+aGymIejVT/8csvA8ZSu3wLpH4HFnJusYP4uY=` / go.mod H1
`h1:lZmzCmDQ5hOA2C/gj/aucrl5YfJq+YxnsBzGhedKB7g=` reproduce. ZIP SHA256
`2dd4eb4a12e0ae34e4dcb403376682ae49a6490b93319e7e59c91572c6c58340`.
GOSUMDB was off; no signed checksum-database proof is claimed.
[Publication proof](../../docs/p11-publication.json) binds exact source and
private complete-file proof. Independent operational review corrected extra
extracted-file admission before acquisition; symlinks/nonregular files refuse.
All four task outcomes and whole review1 pass. Root consolidates current
contracts/lesson, preserves full specification/status and pending A32 support,
then publishes normal retirement closure using the remaining second invocation.
Dependents wait for independently verified closure and exact ordinary successor
module proof; actual W20/M52 adoption remains separate. Audit disabled.
`````
