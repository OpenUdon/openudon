# Retired milestone A08 - Status A08 - Local iCoT UI Server

**Milestone.** A08
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A08.md
**Source status SHA-256.** 2bbb90097a9a831bc13bb05a78c3a168dca3d70df838367f63cfe2ed8f9fe4c1
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
# Status A08 - Local iCoT UI Server

## Goal

Add a single-workspace, loopback-only local HTTP transport over the A07 engine
with an embedded read-only shell and no workflow or live-browser execution
authority.

## State

Complete.

OpenUdon implementation commit:
`01b169ed8a022b48c8ac530a0be32dc9783c513c`.

## Dependencies

- A07 headless authoring engine and shared atomic writer.
- M70 adaptive iCoT v2 state/frontier/source lifecycle.
- A01-A06 and P01 reviewed browser source and verification contracts.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A08.1 CLI and startup lifecycle | `[+]` | Added `icot ui --example` with explicit seed/source/network/port/no-open flags, deterministic seed precedence, 127.0.0.1 binding, platform opener behavior, and bounded signal shutdown. |
| A08.2 authenticated local transport | `[+]` | Added a 256-bit per-process token bootstrap, an HttpOnly SameSite=Strict cookie scoped beneath an unguessable instance path, bearer auth, exact active Host/Origin checks, no CORS, restrictive security headers, and strict 1 MiB one-document JSON decoding. |
| A08.3 revision and freeze state | `[+]` | One server lock protects cached snapshot/revision/completion/write state and engine mutation; exact revisions reject stale/concurrent requests, detached bounded refresh prevents cancellation from advertising stale state, caller slots/sources are impossible, and approved final/incomplete writes freeze mutation. |
| A08.4 embedded read-only shell | `[+]` | Embedded separate HTML, JavaScript, and CSS assets display workspace paths, readiness/top issue, frontier/preview/completion state, and formatted snapshot JSON without authoring controls. |
| A08.5 regression and parity evidence | `[+]` | Focused tests cover auth, Host/Origin, headers/no-CORS, methods, JSON/limits, revisions, explicit approval flags, freeze, concurrent one-winner behavior, real verification/registry refresh failures, CLI startup/open/shutdown, and runtime-only-render HTTP/direct-engine byte parity. |
| A08.6 docs and boundary record | `[+]` | Updated README, CLI help, MkDocs navigation/operator page, product/architecture/tech-stack/milestone memory, and evolution v23 while retaining the experimental non-public API boundary. |
| A08.7 review hardening | `[+]` | Isolated browser cookies from sibling loopback services and concurrent UI processes, added fail-closed post-error revision synchronization, classified operational engine failures as 500, and added focused regressions for each review finding. |

## Acceptance Criteria

- One process owns one explicit example and one trusted local operator.
- The API and shell add no remote/LAN bind, TLS/account service, persistent
  token, folder browser, React/Node dependency, LLM drafting, workflow
  execution, or Browsertools live-authoring authority.
- A07 remains authoritative for frontier slots, source refresh, preview,
  incomplete approval, collision handling, rollback, and artifact bytes.

## Verification

- Focused UI/CLI tests and race tests pass, including concurrent same-revision
  mutation, per-instance cookie isolation, cancellation resynchronization,
  operational/domain error classification, real source-refresh failures, and
  HTTP/direct-engine parity.
- `go test ./...`, `go vet ./...`, `GOWORK=off go test ./... -count=1`,
  `GOWORK=off go vet ./...`, and `make check` pass.
- A standalone `CGO_ENABLED=0 GOWORK=off` iCoT build embeds the shell and runs
  `icot ui --help`; Windows amd64 and macOS arm64 cross-builds also pass.
- The provider-free iCoT authoring scorecard passes 103/103 cases with zero
  missing-detail false passes, unsafe false passes, or needs-input diagnostic
  gaps, and report verification passes.
- `mkdocs build --strict`, doc-memory, repository boundary, and both
  OpenUdon/tofu diff checks pass.
~~~~~~~~~~~~~~~~~~~~
