# Retired milestone M61 - M61 Status - Release-Note Generation

**Milestone.** M61
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M61.md
**Source status SHA-256.** a71d074889c5965a6bb55bef083e7543b6db36d4a7698c63d7efd8e91609b80d
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
# M61 Status - Release-Note Generation

State markers and commit rules are defined in [milestone.md](milestone.md).

| Item | State | Notes |
|---|---|---|
| Draft helper | `[+]` | Added release-note draft generation from verified run evidence, current commit, gates, verifier output, and evidence paths. |
| CLI command | `[+]` | Added `openudon release-notes draft --run-evidence ... --out ...`. |
| Verification | `[+]` | CLI smoke confirms draft output includes gate and async sidecar evidence paths. |
~~~~~~~~~~~~~~~~~~~~
