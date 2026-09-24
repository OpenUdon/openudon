# Retired milestone P04 - P04 Immutable Trusted Execution And Containment Closure

**Milestone.** P04
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-P04.md
**Source status SHA-256.** 2dca95e60bd9f24f9bc5726e82d4b3af8e133e1b696800cae4a7ae4ffbdd088a
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
# P04 Immutable Trusted Execution And Containment Closure

| Item | State | Notes |
| --- | --- | --- |
| P04.1 Manifest-bound byte snapshot | `[+]` | Every required package input is read once; declared input digests, stored quality, package/handoff digests, run config, plan/intent/review/profile parsing, browser protocol, credentials, sessions, and approvals derive from that immutable generation. Current-file staging still rehashes and rejects later drift. |
| P04.2 Process-tree containment | `[+]` | Normal leader exit and cancellation both sweep the process tree. Linux records descendant PID/start-time identities through `/proc`, terminates detached process-group/session children, verifies exit, and avoids PID-reuse kills; other Unix and Windows retain documented fallbacks. |
| P04.3 Docker and outer-runner isolation | `[+]` | Docker `-e` forwards only credentials and sessions, driver environment names resolve from container-owned defaults, host desktop/socket requirements fail closed, outer-runner Udon binary/image overrides propagate, and all credential names use the canonical Udonrunner derivation. |
| P04.4 Snapshot and containment regression proof | `[+]` | Tests mutate inputs after validation, prove config remains snapshot-bound while staging rejects drift, cover Docker argv/environment and external overrides, and cover normal-exit plus detached Linux descendants. |
| P04.5 Release verification | `[+]` | Focused/full/race tests, vet, aggregate release, strict docs, six CGO-disabled cross-build targets, cold-cache standalone verification, clean-root browser matrices, and sandbox-required UI qualification pass under E09. |

P04 does not widen executor authority: production side effects remain behind the
approved trusted runtime boundary.
~~~~~~~~~~~~~~~~~~~~
