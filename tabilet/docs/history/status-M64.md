# Retired milestone M64 - M64 Status - Release Evidence Workflow Consolidation

**Milestone.** M64
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M64.md
**Source status SHA-256.** f5e5162d7eb68fa3d6e7d7bbc52ac9fced0cf904f3c0392ee3e72d9afdc95ed0
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
# M64 Status - Release Evidence Workflow Consolidation

State markers and commit rules are defined in [milestone.md](milestone.md).

| Item | State | Notes |
|---|---|---|
| Single release-evidence entrypoint | `[+]` | Added `openudon release-evidence` and `make release-evidence` to run local udon smoke, archive verification, release-note draft generation, and gate capture. |
| Summary artifact | `[+]` | Emits `openudon.release-evidence-summary.v1` JSON plus Markdown with commit, paths, gates, verifier output, archive refs, and sidecar/report counts. |
| Documentation | `[+]` | README and release stewardship docs describe the local-only flow and confirm artifacts remain ignored unless explicitly promoted. |
| Verification | `[+]` | Passed focused CLI tests, `go test ./...`, `go vet ./...`, `make check`, `check-doc-memory`, boundary check, and diff checks. |
~~~~~~~~~~~~~~~~~~~~
