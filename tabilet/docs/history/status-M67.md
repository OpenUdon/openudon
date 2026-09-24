# Retired milestone M67 - Status M67 - Desired-State Conversion Removal

**Milestone.** M67
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M67.md
**Source status SHA-256.** 8900a9339ce1d48acade6683d43a2fa569104c2b917c1ed40d88bfc45003d8bc
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
# Status M67 - Desired-State Conversion Removal

State of the OpenUdon cleanup that removes retired desired-state conversion
documents and milestone history from OpenUdon.

## Goal

Keep OpenUdon focused on UWS authoring, review, package, approval, and trusted
executor handoff while routing desired-state conversion work to Ramen.

## Tasks

| Item | State | Notes |
|---|---|---|
| Retired conversion docs removed | `[+]` | Removed the stale AWS provider conversion corpus page and public navigation entries. |
| Historical conversion status files removed | `[+]` | Removed OpenUdon status files for the retired conversion adapter and provider conversion tracks. |
| Milestone dashboard cleaned | `[+]` | Removed old conversion sections and status-file index rows; M67 records the ownership cleanup instead. |
| Boundary docs updated | `[+]` | Product, architecture, tech-stack, related-projects, README, AGENTS, and release docs now route desired-state conversion to Ramen without preserving an OpenUdon conversion surface. |
| Regression guard retained | `[+]` | The repository boundary check still rejects parser/conversion imports so conversion code cannot return to OpenUdon accidentally. |

## Acceptance Criteria

- OpenUdon has no desired-state conversion docs, milestone tracks, fixtures,
  command surface, or parser/conversion dependencies.
- Desired-state conversion and provider/resource operation mapping are documented
  as Ramen-owned.
- OpenUdon keeps only negative boundary checks that prevent conversion imports.

## Verification

- `go test ./internal/localcheck ./cmd/openudon`
- `go run ./cmd/openudon check-doc-memory`
- `go run ./cmd/openudon check-apitools-boundary`
- `mkdocs build --strict --site-dir /tmp/openudon-mkdocs-m67`
- `go test ./...`
- `go vet ./...`
- `make check`
- `git diff --check`
- `git -C ../tofu diff --check -- openudon`
~~~~~~~~~~~~~~~~~~~~
