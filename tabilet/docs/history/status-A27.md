# Retired milestone A27 - A27 Status - Registration Terminal Delivery And Global Containment Propagation

**Milestone.** A27
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A27.md
**Source status SHA-256.** 284e5f895e12a654ce8daa0ad42e481c9c96fa14ba8d2f1ebaa885fda2fb3ce9
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
# A27 Status - Registration Terminal Delivery And Global Containment Propagation

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Complete. The containment correction and exact Browsertools A11 adoption are
published at OpenUdon `552ee3a88a595cb25af1d089a0d3beeebba2a24f`;
downstream W8M pin adoption remains separate.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A27.1 Retain and prioritize terminal teardown failure | `[+]` | Joined process-tree timeout now maps to fixed `worker_teardown`, `RegistrationSession` retains the last value-free terminal event independently of its bounded stream, and context cancellation cannot mask process-tree timeout. |
| A27.2 Propagate the registration containment latch globally | `[+]` | The UI reconciles retained terminal state after stream close and one shared predicate blocks every later registration, capture/preflight/staging, package/resume, browser-transaction, and ordinary authoring mutation gate. |
| A27.3 Complete local verification and bounded review | `[+]` | Full tests, vet, `make check`, repeated process-tree tests, focused browserauthor/UI/processgroup race, formatting, diff, privacy, and documentation review pass. Recorded Linux PID/start-time identities are immutable, cancellation takes a verified live-leader group kill before reap, procfs health loss fails containment, and no post-reap numeric process-group signal remains on Linux. No P1/P2 or higher-severity finding remains in the local A27 diff. |
| A27.4 Publish and hand off exact adoption | `[+]` | Browsertools A11 is published at `ce06b13bfef8d1776c3aa019322619c90dacbbd2` (`v0.0.0-20260902183222-ce06b13bfef8`); OpenUdon adopted that exact module, requalified the complete A27 change, and published `552ee3a88a595cb25af1d089a0d3beeebba2a24f`. W8M adoption remains separately owned. |

## Review provenance

- Source: deep W03/W04 implementation review; source priority and baseline were
  not supplied.
- Local severity: P1.
- Revalidated against published A26 before the local correction.
- Implementation commits are
  `7c2dc3048e443c8cae9fd6f48dd4a62e606706b1` and
  `b570e8f48986afcef3bd5a9682e4c988f04ecb6e`.
- Exact Browsertools adoption is commit
  `552ee3a88a595cb25af1d089a0d3beeebba2a24f`; the complete A27 series is
  published at that revision.
- No browser, iCoT launch, target request, candidate, or runtime action was
  part of implementation, adoption, or publication.

## Verification

- Pinned Go 1.26.6 focused browserauthor and UI tests plus focused race.
- Full tests, vet, `make check`, JavaScript syntax, documentation-memory,
  sibling boundary, formatting, diff, and privacy checks.
- Non-Linux processgroup shim compile check after the verified Linux
  live-leader group-kill correction.
- Bounded review of the complete A27 implementation and all downstream gates.
- Exact published Browsertools A11 module resolution, full tests, vet,
  `make check`, and diff checks before OpenUdon publication.

## Publication

- Browsertools A11 was independently confirmed at published `origin/main`
  `ce06b13bfef8d1776c3aa019322619c90dacbbd2`.
- OpenUdon was pushed by normal fast-forward and independently confirmed at
  `origin/main` `552ee3a88a595cb25af1d089a0d3beeebba2a24f`.
- W8M pin adoption and operational browser authorization remain separate.

## Bounded review-fix gate

Maximum 10 iterations. No P1/P2 or higher-severity finding may remain before
A27.3 completes locally.

- Review iteration 1 closed terminal-event loss, teardown classification, and
  the initially identified cross-plane gates.
- Review iteration 2 found and closed capture-stage, authoring-resume, and
  browser-transaction mutation gaps.
- Review iteration 3 found and closed PID/start-time replacement, unsafe
  post-reap group signaling, procfs-health false success, and cancellation
  masking of teardown timeout. The repeated containment and race gates pass.
- Review iteration 4 found and closed a pre-reap cancellation race by taking a
  process-group kill only after the immutable live leader and its group are
  revalidated; identity-only cleanup remains authoritative after reap.
~~~~~~~~~~~~~~~~~~~~
