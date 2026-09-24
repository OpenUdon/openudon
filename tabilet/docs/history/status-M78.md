# Retired milestone M78 - M78 Status - Trusted Browser Registration Handoff

**Milestone.** M78
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M78.md
**Source status SHA-256.** 424352e12a1c79202024ba441edeb915ebd77606d6733697d1ebcc93d336b141
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
# M78 Status - Trusted Browser Registration Handoff

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Complete locally. Publication is not authorized; standalone module resolution
remains intentionally pending until the Browsertools dependency commit is
published.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| M78.1 Pin runtime contracts and add transaction v2 | `[+]` | Commit `94831c8` pins Browsertools `d26f298`, preserves exact transaction-v1 output and unchanged BAP/legacy-BRP production, and adds the registration-only `openudon.browser-profile-transaction.v2` contract for exact registration-authoring-v2 provenance. Focused transaction, adoption, engine, presentation, schema, and legacy-byte tests pass. |
| M78.2 Add registration attestation artifact | `[+]` | Commit `4c07309` adds the closed `openudon.browser-registration-attestation.v1` schema and reader. It requires an absolute canonical owner-only file outside the repository, exact package/profile/operation/flow/zero-attempt/dedicated-test/cleanup bindings, a symbolic reviewer, and an expiry within 24 hours; strict decoding rejects account/value fields, duplicates, symlinks, public modes, and drift. |
| M78.3 Enable exact trusted registration handoff | `[+]` | Commit `a2534f4` enables only one registration-only Browserdriver-v4/Udon-report-v3 handoff. OpenUdon independently validates the private package/profile/operation attestation, requires a separate exact `--approve-browser-registration OP_ID`, keeps values in inherited credential environments, emits Udon attestation and submit flags, revalidates through the outer runner without persisting the private path, and keeps v1-v3/incomplete registration configs fail-before-executor. Existing BAP report-v2/v3 handling remains compatible. |
| M78.4 Qualify compatibility and review | `[+]` | Commit `a94267d` extends the adversarial gate for v1-byte stability, v2 composition/schema, owner-only attestation drift, exact v4/report-v3 construction, Docker flags, closed outer-runner environment, missing attestation/submit approval, and value/path non-disclosure. Full workspace `go test ./...`, `go vet ./...`, focused race tests, the expanded browser-transaction adversarial matrix, and the four-commit gitleaks scan pass. Whole-tree gitleaks retains 10 historical findings and staticcheck retains the documented baseline. Review 1 separated runtime submit approval from the inert authoring symbol; review 2 required report v3 even when a v4 executor returns failure. `make release-check` reaches the expected `GOWORK=off` missing-go.sum boundary because Browsertools `d26f298` is deliberately unpushed. No publication occurred. |

Evolution prompt v32 is present; its result remains absent until E11 passes.
~~~~~~~~~~~~~~~~~~~~
