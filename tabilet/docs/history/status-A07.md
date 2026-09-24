# Retired milestone A07 - Status A07 - Headless iCoT Authoring Engine

**Milestone.** A07
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A07.md
**Source status SHA-256.** 4b88f52acaf85d5666c7dd93a3ac87a2d137cc14eb7543d479a7d407a2bcda89
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
# Status A07 - Headless iCoT Authoring Engine

## Goal

Create an internal, driver-agnostic iCoT authoring engine that can support a
future local UI while preserving the current terminal workflow and artifact
contracts.

## State

Complete.

OpenUdon implementation commit:
`d3ec89967adbf0605e34eb31389255dfb9589d66`.

## Dependencies

- M70 adaptive evidence-grounded iCoT v2 session, frontier, source-discovery,
  draft, and approval lifecycle.
- M53 shared Authoring frontier-round and atomic lifecycle primitives.
- A01-A06 and P01 reviewed browser source, authentication, and value-free
  verification contracts.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A07.1 headless engine contract | `[+]` | Added `internal/icot/engine` with `Open`, `Snapshot`, `ApplyRound`, `Preview`, and `ApproveAndWrite`; its exported API contains no readers or writers, and snapshots are JSON-marshalable. |
| A07.2 discovery and round lifecycle | `[+]` | Engine open/refresh reuses bounded API/browser discovery, static registry state, selected-source synchronization, value-free verification attachment, readiness, frontier planning, and complete-frontier round application. Applied rounds normalize and atomically autosave resumable state. |
| A07.3 provenance resume integrity | `[+]` | Draft operation-detail refs and structured draft events now round-trip through `session.interview.evidence` with the existing annotations, assumptions, mappings, and decision evidence. Draft YAML serialization preserves shared JSON wire field names, including round-answer node IDs. |
| A07.4 shared approval writer | `[+]` | Moved source revalidation/materialization, browser capability/authentication metadata, draft promotion/cleanup, exact proposed actions, and rollback-capable atomic writes into `internal/icot/artifactwriter`; terminal iCoT and the engine use the same transaction. Engine writes require explicit human approval, with separate overwrite and incomplete-draft authority. |
| A07.5 parity and browser regression evidence | `[+]` | Tests cover empty/seeded/resumed open, preview-only behavior, malformed round rejection, provenance resume, approval refusal, byte-for-byte runtime-only-render CLI parity, browser metadata retention, and verification-report replacement rejection. |
| A07.6 post-review state and approval hardening | `[+]` | Approval refresh is transactional across retries, registry-backed selections require exact fresh rediscovery, caller slots are rebound to the authoritative frontier, returned snapshots are deep copies, and approval-capable proposed actions are derived from every file in the prepared atomic transaction. Regression tests cover verification replacement retries, vanished registries, forged slots, nested snapshot mutation, authentication/draft writes, and stale-metadata removals. |

## Acceptance Criteria

- Phase A adds no `icot ui`, HTTP server, frontend, folder browser, published
  schema, or session CLI verbs.
- Existing `cmd/icot` prompting and output behavior remain unchanged.
- A future driver can observe the full current frontier/readiness/source/file
  action state, submit one complete round, preview artifacts, and explicitly
  approve the same atomic write transaction used by the terminal path.
- Browser authoring remains non-executing in this engine; only already reviewed
  profiles, authentication metadata, registries, and value-free verification
  attachments are consumed.

## Verification

- `go test ./internal/icot/...`, `go test ./cmd/icot`, and `go test ./...`
  pass.
- `go vet ./...`, `make check`, repository boundary, doc-memory, and both
  OpenUdon/tofu `git diff --check` gates pass.
- The provider-free `examples/eval` iCoT scorecard passes all 64 fixtures with
  zero missing-detail false passes, unsafe false passes, or needs-input
  diagnostic gaps.
~~~~~~~~~~~~~~~~~~~~
