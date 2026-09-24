# Retired milestone A09 - Status A09 - Enhanced Phase B Reliability And Status UX

**Milestone.** A09
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A09.md
**Source status SHA-256.** 4494ba3e42439b65f4985a6135edba669a592046b120556fe388ea40ed911dba
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
# Status A09 - Enhanced Phase B Reliability And Status UX

## Goal

Close the remaining Phase B mutation-lifecycle, optimistic workspace,
transport-hardening, and read-only status UX gaps without widening the product
beyond one loopback workspace and one trusted operator.

## State

Complete.

## Dependencies

- A07 headless authoring engine and shared artifact writer.
- A08 loopback-only local transport and embedded read-only shell.
- M70 adaptive iCoT v2 plus A01-A06/P01 reviewed source contracts.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A09.1 prospective engine transactions | `[+]` | `ApplyRound` builds refreshed state/snapshot before atomic draft persistence and finishes after persistence without request-context interruption. Approval returns `ApprovalResult{Snapshot, WriteResult}` with no post-commit refresh or redundant draft deletion. |
| A09.2 typed failure and writer boundary | `[+]` | Added rejected, conflict, operational, and indeterminate engine classes. The shared writer resolves outputs beneath the canonical example root, rejects descendant symlinks/swaps, cleans backups after success/rollback, reports rollback failure as indeterminate, and returns post-commit cleanup failures as non-fatal `cleanup_warnings`. |
| A09.3 optimistic workspace protection | `[+]` | Sorted SHA-256 fingerprints cover fixed project/draft/metadata paths, selected materialized sources, and snapshot actions. A pre-refresh workspace observation binds paths that become newly selected during a round. Drift latches `externally_modified`, preserves cached inspection, and blocks every mutation until restart, including forced overwrite and competing-engine cases. |
| A09.4 experimental API v2 | `[+]` | Replaced v1 with snapshot/round/approve v2, workspace-aware revisions, conditional GET, typed status/error envelopes, exact bootstrap scope, recursive duplicate-name and UTF-8 validation, media/body/header/time limits, COOP/CORP, and sanitized 500-only request-ID logging. |
| A09.5 polling read-only shell | `[+]` | The separate embedded assets poll every two seconds while visible, refresh on visibility, back off to 30 seconds, preserve cached JSON, and show revision, refresh, source/action counts, readiness, issue, frontier, preview, completion, and restart-required drift state. |
| A09.6 reliability regression evidence | `[+]` | Tests cover byte-stable round failure, post-persistence cancellation, drift during and between mutations across owned path families, two-engine contention, backup cleanup, rollback classification, symlinks/root escape, strict and slow requests, conditional snapshots, safe logs, real loopback bootstrap/round/approval/freeze, HTTP/direct parity, race behavior, and standalone builds. |
| A09.7 docs and evolution | `[+]` | Updated README, CLI help, operator docs, product/architecture/tech-stack/milestone memory, and evolution v24 while retaining the experimental, read-only, loopback-only Phase B boundary. |

## Acceptance Criteria

- One process still owns one explicit example for one trusted operator; restart
  is the recovery path after external modification.
- The API and shell add no LAN/remote service, persistent lease/token, account
  system, folder browser, React/Node, workflow execution, LLM drafting, or live
  Browsertools orchestration.
- Successful artifact commits always return their exact frozen inspectable
  snapshot, and errors cannot advertise state that differs from durable draft
  or artifact outcomes.

## Verification

- Focused engine, writer, UI, CLI, and race suites pass.
- Workspace and `GOWORK=off` tests/vet, `make check`, strict MkDocs,
  doc-memory, boundary, and diff checks pass.
- CGO-disabled Linux plus Windows/macOS iCoT builds pass with embedded assets.
- The provider-free iCoT authoring scorecard and report verification pass all
  103 cases.
~~~~~~~~~~~~~~~~~~~~
