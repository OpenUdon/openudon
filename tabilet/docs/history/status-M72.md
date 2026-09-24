# Retired milestone M72 - Status M72 - Structured API Security Alternatives

**Milestone.** M72
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M72.md
**Source status SHA-256.** 4db45681139241f663b98ff5ab442acff3406c8fc06e09c6440d328177e81aab
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
# Status M72 - Structured API Security Alternatives

State: Complete

| Item | State | Notes |
|---|---|---|
| M72 security-alternative authoring | `[+]` | Commit `00a5a4e` migrates to Apitools security requirement sets and `authoring.prompt-context.v2`; prompts retain OR/AND/anonymous structure; iCoT forces a stable indexed choice, persists it and safety evidence across resume, requires only the chosen credentials, and rejects unselected request fields. Incomplete discovery, prompt-budget deferral, network-policy retention, and source target collision regressions are covered; full tests and vet pass. |

## Boundary Checks

- Selection is authoring evidence, not runtime credential resolution or API
  execution approval.
- Candidate workflows receive no source/operation/security breakdown.
- Final intent approval remains non-deferrable.
~~~~~~~~~~~~~~~~~~~~
