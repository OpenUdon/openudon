# Retired milestone M60 - M60 Status - Executor-Report Archive Command

**Milestone.** M60
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M60.md
**Source status SHA-256.** 9b0ec19856ae8791af112de3edd6c21a8a8f37dc0d3e0f5920eda9b4648a23be
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
# M60 Status - Executor-Report Archive Command

State markers and commit rules are defined in [milestone.md](milestone.md).

| Item | State | Notes |
|---|---|---|
| Archive helper | `[+]` | Added archive copying for run evidence, async sidecars, and executor report files when present. |
| CLI command | `[+]` | Added `openudon run-evidence archive --file ... --out ...`. |
| Verification | `[+]` | Archive helper verifies source and archived evidence; CLI smoke covers archive output. |
~~~~~~~~~~~~~~~~~~~~
