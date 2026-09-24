# Retired milestone E13 - E13 — Focused browser development checks

**Milestone.** E13
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-E13.md
**Source status SHA-256.** 9206eaad5aed7a5f5ce993412981a848507e0242f9354918ac1b6f93919af48c
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# E13 — Focused browser development checks

| Item | State | Notes |
| --- | --- | --- |
| E13.1 Add fast/smoke entry points and timing | `[+]` | Separate development reports from native qualification; keep exact legacy report semantics. OpenUdon `3c9bebb`. |
| E13.2 Cache unchanged development work | `[+]` | Bind source bytes, tool/runtime and fixture inputs; preserve original execution identity; cache only successful reduced evidence and immutable builds. OpenUdon `d32e036` and `3c9bebb`. |
| E13.3 Verify and document | `[+]` | Invalidation, corruption, failure, private-state exclusion and actual fresh/cached smoke checks; W8M owns acceptance v2. OpenUdon docs `836053f`; canonical guidance `aea4e90`. |

Owner-approved successor for faster feature feedback. No live target, provider,
deployment or account operation. W15/A29 qualified history stays frozen. Review
iteration 3 of maximum ten completed with no open P1/P2 findings.


Complete, with implementation and planning commits published. OpenUdon owns
the thin `browser-system-dev` CLI, closed fast/smoke stage selection, input
fingerprints, development-only evidence,
immutable build/result reuse and private diagnostic timing. Custom reuse is
limited to the controlled in-process registration and BAP/BCP components; other
stages remain fresh. Fast works without a private executor checkout. Native
qualification versions and three-repeat inventory remain unchanged. W8M owns
its v2 aggregate and consumer smoke. No new runtime was adopted or published.

Full OpenUdon unit suite and document checks passed; affected package tests/vet,
cache drift/corruption/permissions/failure/staleness checks and both development
and native evidence isolation passed. Final fresh typed registration smoke took
95.70 s wall (69.289 s flow); explicit reuse took 9.87 s and preserved original
execution identity/time. Earlier forced-fresh build-cache reuse also passed:
Browserdriver/Udon/Browsertools restore phases measured 77/232/172 ms versus
8406/4152/2978 ms for initial compilation. The final source change caused a cache
miss and a fresh flow before the successful reuse check.

W8M's fresh consumer smoke passed in 409.238 s; its warm fast gate took 9.54 s.
Its original W15 v1 aggregate still verifies unchanged. Reduced development
reports and timing are retained privately under
`/home/peter/.local/state/w8m-browser/w17-development-20260912`. Fresh complete
qualification remains required for a future frozen integration/adoption
candidate; cached development results never satisfy it. Review corrections
covered parent aliases, child build-environment scope, separate Playwright-Go
runtime inputs, deferred cache publication, and standalone fast checks.


## Source commit split

OpenUdon: `d32e036` cache/timing support, `3c9bebb` focused commands/reuse,
`836053f` operator documentation. W8M: `1010057` acceptance v2, `ae3abe8`
development checks, `e2fe70b` policy and W17 evidence. Tofu `aea4e90` records
the canonical development guidance; `43c503a` records completion and
cross-repository source references. Independent staged compilation and the
relocated consumer refusal regression passed. Subsequent owner-authorized
pushes published these commits through OpenUdon `aaf3c0f`, W8M `e2fe70b` and
Tofu `43c503a`. Independent remote checks confirmed each implementation and
planning commit is reachable from its repository's `origin/main`. Qualification
publication locks and the adopted W15 runtime remain unchanged; source
publication does not claim a fresh v2 qualification.
~~~~~~~~~~~~~~~~~~~~
