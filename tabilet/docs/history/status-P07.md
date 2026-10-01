# Retired milestone P07 - Browser 1.10 trusted package dispatch compatibility

**Milestone.** P07
**Outcome.** completed
**Retired.** 2026-09-30
**Source status.** tabilet/memory-bank/status-P07.md
**Source specification.** tabilet/memory-bank/milestone.md#p07---browser-110-trusted-package-dispatch-compatibility
**Evidence.** 5a809e98bd4608fe42d9449ca3f8f33904370861
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 2
**Verification.** Original recorded owner checks, source-bound synthetic qualification and bounded review passed; original evidence and counters preserved. Current upstream ancestry and downstream W8M history were inspected for normal closure, without a new browser run.
**Consolidated into.** [architecture](../../memory-bank/architecture.md), [tech-stack](../../memory-bank/tech-stack.md), [lessons](../../memory-bank/lessons.md), and [milestone dashboard](../../memory-bank/milestone.md).

## Milestone specification

````markdown
### P07 — Browser 1.10 trusted package dispatch compatibility

Repair OpenUdon's existing trusted-runner dispatch so the completed E22
Browser 1.10 count profile can pass package preparation and trusted dry-run.
Map it to the existing rank-10/v10 handoff and preserve Browser 1.8/1.9
behavior, approval checks, and executor boundaries. Do not change UWS, Udon,
Browsertools, Browserdriver, or other package source. E22 remains completed
history. P07 revalidates the exact W8M staged workflow/profile package only in
a fresh restricted preparation path; the separate W8M `session_posture: none`
input correction remains W8M-owned downstream work.

P07.1 implements and regression-tests the trusted dispatch repair and the
restricted exact-package dry-run without invoking an executor or browser.
P07.2 runs fresh current v4 qualification on the exact clean P07 source,
independently verifies the reports, rechecks retained E22 evidence without
changing its bytes, and closes the bounded review gate. P07 depends on completed
E22 and precedes W8M W22.3 source-bound smoke/qualification and later adoption
or its separately gated operation. Use local task commits only; no push,
publication, target contact, or runtime adoption is part of P07.

````

## Status record

````markdown
# Status P07 — Browser 1.10 trusted package dispatch compatibility

**State:** Complete locally after bounded review iteration 2. P07.1 and P07.2
are complete on local branch `P07`; no push or publication.

**Goal.** Repair OpenUdon's trusted-runner rank-10 dispatch so the existing
Browser 1.10 count profile passes the complete package preparation and
trusted-dry-run lifecycle while preserving Browser 1.8/1.9 behavior.

**Review source and finding.** F01, W8M W22 bounded review iteration 4,
`tabilet/memory-bank/status-W22.md` in the W8M repository, W22.5 offline
package-preflight investigation. Source priority P2; local severity P2;
disposition confirmed; owner OpenUdon P07. The review baseline commit was not
supplied. The W8M status-file evidence was uncommitted in the inspected W8M
main worktree. OpenUdon revalidation used clean commit
`55b24d29279c8efe67ae931f9d73f094f929efab`: its trusted runner accepted only
Browser 1.8/1.9 in the rank-10 switch and rejected other discriminators. Its
existing focused tests covered 1.8/1.9 but not 1.10. The uncached
`internal/trustedrunner` baseline test passed with Go 1.26.6.

**Dependencies and downstream impact.** Completed E22 supplies the published
Browser 1.10 count contract and remains immutable history. P07 repairs the
source-owner dispatch. W8M W22.2 remains blocked on its current OpenUdon pin
until W8M binds the exact local P07 commit through its local replacement and
passes the exact package lifecycle; W22.3 must then bind that commit and source
digest and run fresh consumer smoke and qualification. W8M remains responsible
for its `session_posture: none` package-review input correction. W22.4 adoption
and W22.5's single read-only operation remain separately gated by W8M
acceptance. P07 does not change other package sources or grant target,
executor, adoption, push, or publication authority.

**Evolution.** No evolution pair is needed: P07 completes trusted-package
compatibility for the existing Browser 1.10 count direction and changes no
product boundary or public/private contract direction.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| P07.1 Implement Browser 1.10 trusted dispatch and exact restricted package check | `[+]` | Commit after focused tests, repository gates, and the exact restricted package lifecycle pass. The independent package report and source-byte comparisons are recorded below. |
| P07.2 Fresh v4 qualification, independent verification, and bounded review | `[+]` | Fresh current v4 integration, loopback, journey, and three-repeat native qualification passed from exact clean P07.1 commit `87df787c7737cc669f98c3b4462d7151db3e6b68`. All four reports independently verify; all 19 native source bindings match independent recomputation and exact locks. Retained E22 report hashes and bindings are unchanged. Bounded review iteration 2 passed with no unresolved P1/P2; required repository and documentation checks pass. |

### P07.1 Verification

The source change adds `uws.browser.1.10` to the existing active-profile rank-10
mapping. Focused uncached `go test -count=1 ./internal/trustedrunner`,
`make fast`, `go vet ./...`, `make check`, `check-doc-memory`, and
`git diff --check` pass with Go 1.26.6. The memory check emits its normal
warning that the milestone changed without an evolution pair; the evolution
rationale above records why no new direction version is needed.

The fresh restricted W22 package check is preserved at
`/home/peter/.local/state/openudon/p07-package-20260928/attempt-02/prepare.json`.
Its input SHA-256 is
`9a90d3b679f77bfff294038a6f0f59438238c5fc13d0713420fe4ef08e367be3`, package
SHA-256 is `6f374ea4b80f786af23615975fa189c0345ec989f17de951644bd94b13ed4d43`,
quality SHA-256 is
`20f2c97a1db16a7145ce16e970b550e2da0f049be68850ce8ad14ae6c9fd3f29`,
manifest SHA-256 is
`sha256:43553e8fc9d331087c88259c584c03df4e22a9535aca6511b4ddd857b7a4bcf9`,
and qualification SHA-256 is
`sha256:a22c73cbc6912ee9b9ca1a9057337c6c601afe3d0420cf1640ca72ee9cbd1f69`.
All five gates pass, including quality/secret scan and trusted dry-run; the
report says executor not invoked. No browser ran. The run used
`session_posture: none` in `.icot/browser-sources.json` (SHA-256
`43a55405dc0e44857028d0671f719cadf555434686aaef2c1543d1690ddb79fe`).
Its browser profile SHA-256
`f2f48267b31d2efdd8828ff504f1a5d82013338406f7b783ceedb657363606c5` and
workflow SHA-256
`bb4b85bedce6ff8bddb89657ed5f09bd58f3ff47455c655df80b6734944d998f` match the
corresponding W8M candidate source files byte-for-byte. W8M's supplied checkout
and these source files were not modified. A preceding fresh CLI invocation
stopped before lifecycle gates because its optional expected-input digest was
passed with the wrong prefix; it is preserved at `attempt-01` and was not a
package or source failure. W8M still owns binding the corrected posture metadata
in its candidate.

### P07.2 Verification

The fresh qualification ran from an isolated clean clone of P07.1 commit
`87df787c7737cc669f98c3b4462d7151db3e6b68`, using the exact current v4 locks
and clean source revisions. Integration passed 19/19 with no skips, loopback
passed 23/23 with no skips or quarantines, and journey passed 14/14 with no
skips or quarantines. Their reports independently verify and their recorded
digests are:

- `integration-v4-sandbox-helper.json` —
  `c1c90912963fc9bca08606ab1d2770bf1a05847e7dd7979a7b3e9071cfe1b368`.
- `loopback-v4.json` —
  `eb098351d1597833c5f804e262598dd75b10e18d9869332540a349daa3a48477`.
- `journey-v4.json` —
  `b8f3aaed7b3e3db5d452af9bd7860c294e8f391807ed13fd2e220f7cecfcc011`.

The fresh current-stack native loopback qualification passed three repeats of
13 stages each. The v4 report independently verifies; all 19 source entries,
including the fourteen `udon_build_*` aliases, match independent `SourceDigest`
recomputation against the exact locked commits. The source inventory and
recomputation are preserved at
`/home/peter/.local/state/openudon/p07-qualification-20260928/attempt-01/evidence/native-v4.json`,
`source-bindings-independent.py`, and
`source-bindings-independent.json`. The report SHA-256 is
`e9f443b649f48da7bdfff4f3a6e54d7b57384c046762ac895bd44a334d4f4431`.
Every source tree remained clean; the external read-only Browserdriver Node
modules matched the lock. The qualification used the already-installed secure
Chrome sandbox helper through `CHROME_DEVEL_SANDBOX`; it changed no sysctl and
used no sudo. The disposable browser and scenario directories were removed.

Retained E22 integration, loopback, journey, and native reports still verify.
Their original SHA-256 values remain, respectively,
`cb380a6f63951c1e8ec542507ce196a9a60296419a23e8d73e506d46f04035f2`,
`a9549e8b95b990d9676468cad3cfcc7cebbbdf0f0f2f447d243ead90329b4c93`,
`e5b21a5c88add31f5aa8141bd59352210e1a2cbdcdfef07432af333a5d1deb37`, and
`3e0e26a1350f9c2c2121173d0efbe53818161792926481b1e02cf2caa205cf12`. The
retained native report's nineteen bindings independently match its original
clean source trees. New comparison artifacts are under the fresh P07 evidence
directory; the retained E22 reports and source trees were left unchanged.

An auxiliary optional `browser-system-input` identity export failed with
`qualification_input` because it expects Browserdriver's `node_modules` inside
the source checkout. Its empty output is preserved. A temporary symlink used
to diagnose that helper was removed; the native qualification instead used the
validated separately supplied, read-only dependency tree and passed its own
source checks. No report or qualification input was edited to obtain a pass.

Repository checks on the implementation branch passed with Go 1.26.6:
`make fast` (including `check-doc-memory`), `make check` (standalone iCoT build,
all Go tests, sibling checks and Apitools boundary), `make vet`, and
`git diff --check`. The documentation-memory check reported the existing
milestone-without-evolution warning; P07 changes implementation and current
facts without changing product or architecture direction, as recorded above.

### Bounded review

Iteration 1 reviewed the complete P07 change from baseline
`55b24d29279c8efe67ae931f9d73f094f929efab`, including the trusted-runner
dispatch, regression tests, exact-package evidence, current facts, lesson,
knowledge preservation, milestone specification, and qualification record.
The imported F01 P2 is resolved by mapping Browser 1.10 to the existing
rank-10/v10 handoff and covering supported and unsupported versions.

Iteration 1 found P2 P07-R01: the OpenUdon active-track summary said no W8M
target contact had occurred, but W8M W22's log records an earlier
unauthenticated campaign-route probe. That probe read no authenticated count
and created no packet or attempt; P07 itself made no target contact, and the
single authorized count remains unused. Correct the summary to preserve those
distinctions. No code finding remains. Iteration 2 is required after the
documentation correction and focused memory/diff checks.

Iteration 2 is recorded before the full review pass. P07-R01 is corrected in
`milestone.md`; `check-doc-memory` and `git diff --check` pass. Re-review the
complete baseline-to-branch change, including the correction and its review
record, updated completion summary, and downstream W8M local-replacement
requirement. Iteration 2 found no unresolved P1/P2-or-higher finding; the
bounded review gate passed after two iterations. P07-R01 is closed, and the
imported F01 P2 remains resolved by the existing rank-10/v10 mapping and
supported/unsupported-version regressions. The fresh three-repeat v4 evidence,
independent 19-source digest comparison, unchanged retained E22 evidence, full
repository checks, and downstream W8M local-replacement requirement were
rechecked. No evolution version was needed.

## Acceptance

- Browser 1.10 uses the established rank-10/v10 trusted handoff without
  changing executor protocol, approval, authentication, or runtime behavior;
  Browser 1.8/1.9 remain compatible and unsupported profile versions remain
  rejected.
- The exact W22 workflow/profile passes OpenUdon package quality, prepare,
  and trusted dry-run using corrected session-posture metadata, with no executor
  or browser invoked and no supplied source modified.
- Fresh current v4 integration, loopback, journey, and three-repeat native
  qualification pass on clean exact P07 source; reports independently verify.
- Retained E22 reports verify with unchanged digests and bindings.
- The bounded review gate passes with no unresolved P1/P2-or-higher finding.
- Work remains local on branch `P07`, with one verified commit per task row;
  no push, publication, W8M runtime adoption, or live target operation occurs.

## Verification commands

Use the repository-pinned Go 1.26.6 toolchain and the normal OpenUdon current
v4 qualification commands. Preserve the original W8M and E22 evidence; write
new reports to fresh private output paths. Run focused trusted-runner tests,
the exact restricted package lifecycle check, independent report verifiers,
`make fast`, `go vet ./...`, `make check`, documentation-memory checks, and
`git diff --check`. Do not fetch, install, upgrade, or mutate supplied source
worktrees during qualification.

## Normal closure reconciliation — 2026-09-30

The confirmed Stage 5 goal authorizes completing genuinely unfinished
prerequisite closure before M91. All original task outcomes, consumed attempts
and review counters above are preserved; no completed task or review is rerun.
Observed OpenUdon origin/main is
`e12a6488b86cafddb9298c7917de84fbc1cc85ff`; original accepted E21, P07 and
P08 source commits are ancestors of that published revision. Earlier local-only
and downstream-pending statements above retain their original context.

W8M's completed W24 now owns its independently qualified P08.1 v11 source
`5cad6ce55e0f615a8e754f468614cadbe565790b` and its accepted count/teardown;
resolve W21/W22/W24 through W8M's history index. P07's v10 preparation result
remains recorded acceptance of its original scope; P08 owns the corrected v11
executor pairing. Current Stage 5 extraction preserves that v11 pairing and
E23/E24 inputs. No W8M record, old report, live claim or adopted runtime is
modified here. The normal retirement preserves the complete specification and
status, repairs maintained evidence links, and retains reusable lessons.
````
