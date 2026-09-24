# Retired milestone E07 - E07 Authentication-Real Browser Evidence

**Milestone.** E07
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-E07.md
**Source status SHA-256.** 7fc94da2f45ab1e65798738afc4837227ff703791acef113be3c23c5ad6cdf14
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
# E07 Authentication-Real Browser Evidence

| Item | State | Notes |
| --- | --- | --- |
| E07.1 Stateful authentication fixture | `[+]` | Loopback goal pages require a random HttpOnly/SameSite session cookie; exact password and SMS/email/voice/TOTP/non-input MFA challenges are verified server-side. |
| E07.2 Replay success assertion | `[+]` | A scenario cannot pass unless the fixture records authenticated replay, so navigation-shaped bypasses and wrong or absent challenge values fail. |
| E07.3 Compatibility lock v2 | `[+]` | The lock pins the corrected Browsertools revision, exact Playwright package, and exact Chromium browser version, and rejects dirty or revision-mismatched pinned sibling worktrees. |
| E07.4 Shared integration policy | `[+]` | Browser integration evaluation imports the same repository-state validator and Playwright contract; scenario preparation owns the actual installed Playwright/Chromium launch comparison. |
| E07.5 Evidence hygiene | `[+]` | Structured failure classes replace broad context substrings, expected outputs derive from fixture facts, journey responses use bounded server headers/timeouts, and loopback evaluation does not inherit proxy variables. |

`not_run` remains an inspectable non-pass status when readiness is not required;
release targets continue to require actual ready execution.

Implementation commit: OpenUdon `ad66c6a`.
~~~~~~~~~~~~~~~~~~~~
