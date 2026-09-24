# Retired milestone M74 - M74 Consolidation And v0.2 Closure

**Milestone.** M74
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M74.md
**Source status SHA-256.** 233d5bd255c751152a33dc9aa63a4afb61767362478c07c6a3c86c0ee1efbf26
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
# M74 Consolidation And v0.2 Closure

| Item | State | Notes |
| --- | --- | --- |
| M74.1 Durable atomic evidence writes | `[+]` | Shared writes sync file data, rename atomically, sync parent directories where supported, clean failures, and cover replacement/rename faults. |
| M74.2 Reusable CLI policy extraction | `[+]` | Build-info parsing, public UWS validation, quality remediation, and catalog selection moved from `cmd/openudon` into focused internal packages. |
| M74.3 Shared evidence helpers | `[+]` | Bounded regular-file reads, duplicate/unknown/trailing-safe JSON, digest sidecars, and full 40/64-character Git IDs share one implementation. |
| M74.4 Dead-code and schema cleanup | `[+]` | Pinned `deadcode v0.47.0 -test ./...` reports no functions; unused UWS schema snapshots and obsolete non-atomic APIs were deleted. |
| M74.5 File responsibility split and regression tests | `[+]` | iCoT, progressive elicitation, and OpenAPI quality files are split below 1,000 lines; iCot reports, atomic files, endpoints, verbs, and evidence helpers have dedicated tests. |
| M74.6 Documentation and release-ready tree | `[+]` | README, compatibility, safety, handoff, memory, status, and evolution describe v0.2 without tagging or publishing it. |
| M74.7 Hosted sandboxed Phase C proof | `[-]` | Remains owned by A11.5. The 2026-08-20 local sandbox-enabled attempt failed because the host has no usable Chromium sandbox; no sandbox-disable override was used, and hosted release-runner proof remains required. Successor: A11.5. |
~~~~~~~~~~~~~~~~~~~~
