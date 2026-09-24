# Retired milestone M12 - Status M12 — Package Artifact And Local OpenAPI Safety Hardening

**Milestone.** M12
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M12.md
**Source status SHA-256.** 5c7a1a171696070c7be801d31c5205d62426bfd78a5951ddcc3d6dc3102548d2
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
# Status M12 — Package Artifact And Local OpenAPI Safety Hardening

State of M12 items. See [milestone.md](milestone.md) for milestone scope and
acceptance criteria.

Status markers:

| Symbol | Suggested Status | Interpretation |
|---|---|---|
| `[ ]` | Pending | Item not started or pending action. |
| `[+]` | Completed | Item finished or done. |
| `[~]` | In Progress | Item is being worked on. |
| `[!]` | Blocked | Item requires attention or is on hold. |
| `[X]` | Cancelled | Item is no longer needed. |

| Item | State | Notes |
|---|---|---|
| Package root validation hardened | `[+]` | Required handoff inputs share safe relative path validation, package-root validation, and regular-file checks. |
| Required handoff inventory hardened | `[+]` | Manifest inventory checks reject unsafe, missing, symlinked, directory, special-file, and unstated required inputs. |
| Package digest inputs hardened | `[+]` | Digest input validation covers reviewed UWS artifacts and staged OpenAPI files. |
| Trusted-runner staging guards hardened | `[+]` | Staging rejects symlinks and verifies staged package digest before executor invocation. |
| apitools local OpenAPI reads hardened | `[+]` | Local OpenAPI reads reject symlinked roots/paths/parents, directories, special files, and oversized path-backed documents. |
| Shell/Python runner replaced | `[+]` | Go run-config parsing and staging live in `internal/udonrunner` and `cmd/udon-runner`. |
| Executor selector simplified | `[+]` | `OPENUDON_EXECUTOR` is canonical and accepts absolute binary paths or `docker://<image>`. |
| Direct runner fail-closed checks added | `[+]` | Direct `cmd/udon-runner` rejects missing package digest, missing package paths, and direct production run configs. |
| workflowintent package split | `[+]` | Intent/HCL, authoring adapter, chat adapter, provider client, OpenAPI, and helper files replace the monolith without changing the package API. |
~~~~~~~~~~~~~~~~~~~~
