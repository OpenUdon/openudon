# Retired milestone A26 - A26 Status - One-Attempt Registration Authoring And Terminal Failure Retention

**Milestone.** A26
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A26.md
**Source status SHA-256.** 456a68c6a08431cd23e5373473a0ece33633839813d776f32fd64f46c108e400
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
# A26 Status - One-Attempt Registration Authoring And Terminal Failure Retention

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Complete and published at OpenUdon
`e40f56fc71b9e336f270118dde9cc1830326b489`, against published A25 base
`561fd933097abe90cdbe2b59b56e0fa33d4d41c1`. A downstream authorized session
showed that the final iCoT state replaced the transient closed worker failure
code with generic prose, while the same UI process admitted additional Launch
requests after the first authority boundary was consumed. Every observed
attempt failed closed, all descendants stopped, and no candidate or private
result survived. The remediation, verification, and publication were
target-offline.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A26.1 Enforce one process-local registration Launch | `[+]` | Published at OpenUdon `e40f56fc71b9e336f270118dde9cc1830326b489`. The authenticated start endpoint consumes its attempt immediately before worker construction. All later valid starts return `registration_authorization_consumed` before revision, workspace, or worker processing; API state exposes `attempt_consumed`, and the UI remains locked independently of client activity state. |
| A26.2 Retain one closed terminal failure class | `[+]` | Published at OpenUdon `e40f56fc71b9e336f270118dde9cc1830326b489`. Final state carries only an allowlisted `failure_code`; unknown input collapses to `worker_failed`, and raw worker errors, paths, output, and target observations cannot enter the state. |
| A26.3 Publish and qualify the exact implementation | `[+]` | Complete. OpenUdon `origin/main` independently resolved to exact `e40f56fc71b9e336f270118dde9cc1830326b489` after publication. Downstream W8M adoption, preflight, and target authority remain separate. |

## Acceptance

- One attempt is consumed before worker construction, including a construction
  failure, and no later start in the same iCoT process can create a worker.
- Rapid duplicate requests carrying the original revision, active-session
  repeats, and post-terminal repeats return the fixed consumed-authority code.
- Terminal state preserves only a closed value-free failure class; unknown or
  unsafe error values collapse to `worker_failed` without disclosure.
- UI form controls lock from server-owned `attempt_consumed` state and explain
  that another attempt requires a fresh preflight, authorization, and process.
- Focused/full offline tests, race tests, vet, repository checks,
  documentation-memory validation, JavaScript syntax, diff checks, and bounded
  review pass.

## Non-Goals

- No UWS, Browsertools, dependency, public stable API, runtime, workflow, or
  qualification-report change.
- No target contact, browser session, registration submission, candidate,
  transaction adoption, credential entry, account action, runtime execution,
  downstream pin adoption, release, or deployment.

## Verification

- Installed pinned Go 1.26.6 with `GOWORK=off`, `GOPROXY=off`, `GOSUMDB=off`,
  `GOENV=off`, and `GOTOOLCHAIN=local`: focused UI/browserauthor tests, focused
  UI race tests, full `go test ./...`, full `go vet ./...`, and `make check`
  pass.
- `node --check internal/icot/ui/assets/app.js`, sibling/API boundary checks,
  documentation-memory validation, example validation, and OpenUdon/Tofu diff
  checks pass.
- Review iteration 1 found two P2s: consumed-authority rejection followed
  workspace inspection, so an unavailable inspection could obscure the fixed
  retry result; and status wording implied all safe final-state fields vanished.
  Rejection now precedes inspection, and the wording distinguishes the closed
  failure class from safe message/timestamp fields. Review iteration 2 found no
  unresolved P1/P2 or higher-severity issue.
- Final inspection found no iCoT, Playwright-driver, or worker-Chromium process.
  No target contact, browser session, candidate, package write, execution,
  release, deployment, or provider mutation occurred.
- Separately authorized publication advanced OpenUdon `origin/main` from A25
  `561fd933097abe90cdbe2b59b56e0fa33d4d41c1` to exact A26
  `e40f56fc71b9e336f270118dde9cc1830326b489`; independent resolution returned
  that exact commit.

## Publication State

- Independent `origin/main` resolution returned exact OpenUdon commit
  `e40f56fc71b9e336f270118dde9cc1830326b489`. This Tofu closeout records that
  publication; downstream pin adoption remains a separate decision.
~~~~~~~~~~~~~~~~~~~~
