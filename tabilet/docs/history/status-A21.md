# Retired milestone A21 - A21 Status - Guided iCoT Browser Registration Authoring

**Milestone.** A21
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A21.md
**Source status SHA-256.** 3174175bf297e6c4fa7581657e497a0a063de316c102f9a507eb774c0d7d1a2d
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
# A21 Status - Guided iCoT Browser Registration Authoring

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Complete locally after M78 at OpenUdon commit `fd86051`. Publication is not
authorized; E11 is the next strict status.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A21.1 Add registration-authoring v2 coordinator | `[+]` | Commit `f27d588` preserves empty-protocol v1 behavior while deliberately launching Browsertools with `--protocol v2`, pins every NDJSON message and response to the selected protocol, revalidates v2 retained navigation, and accepts only the corresponding transaction/result version after private-inbox adoption. The child-process test proves a reviewed structural `action=startnew` query reaches the worker and canonical profile while reduced observations disclose no query; focused coordinator tests pass. |
| A21.2 Add authenticated revision-bound API lifecycle | `[+]` | Commit `1f190a7` adds authenticated API-v4 start, typed command, and cancel routes with an independent registration-authoring revision, one contained session across capture and BRP authoring, bundled worker protocol-v2 selection, monotonic cancellation, and result availability only after the event stream closes. Closed request structs admit no raw profile, credential, verification, or value fields; public state cannot represent URLs, queries, private paths, raw output, or the adopted candidate. Focused API, coordinator, iCoT, race, and vet tests pass, including stale revision, single-session, containment-failure, and retained-query non-disclosure cases. |
| A21.3 Build the accessible BRP wizard | `[+]` | Commit `6173f76` adds a keyboard-accessible, no-submit wizard for metadata, confidence/expiry, symbolic credential slots, ordered declarative macro steps, observed accessibility locators, effects, confirmation, success proof, fixed call controls, and cleanup. The server—not JavaScript—constructs and revalidates canonical UWS registration 1.0 bytes, rejects unsafe/repeated/secret-shaped queries and incomplete submit authority, and discloses exact retained structural query pairs plus canonical JSON for a separate confirmation before worker review and finish. Focused API, builder, race, vet, JavaScript syntax, static accessibility, and real loopback Chromium interaction tests pass. |
| A21.4 Adopt through the existing package lifecycle | `[+]` | Commit `cf41e5e` requires an inactive package-configured transaction engine before launch, starts the exact v2 candidate only after clean worker teardown, independently binds reviewed transaction bytes before virtual-source selection, and keeps package preparation closed until the ordinary authoring write and passing package-build boundary. One real integration test proves candidate review, selected credential-free BRP/review materialization, package writing, qualification, preparation, and atomic promotion with runtime authority still false; focused UI/iCoT/synthesis tests, race tests, vet, JavaScript syntax, and diff checks pass. The same task also made registration-only sources count as first-class package sources, rejected secret-shaped symbolic binding names, and preserved the initial failing build report rather than trying to inspect a deliberately unbound failed handoff. |
| A21.5 Qualify UI, terminal, and legacy compatibility | `[+]` | Commit `fd86051` closes A21 after `browser-transaction-adversarial`, all 17 sandbox-required real-Chromium UI journeys, full `go test ./...`, full race, vet, workspace build, UWS/docs/boundary checks, strict MkDocs build, formatting/diff checks, and the A21-range secret scan passed. The first bounded review found and corrected stale operator guidance and a synthetic secret-shaped history fixture; the second review passed the cumulative coordinator/API/wizard/package/legacy diff without another finding. The whole-tree secret scan still reports the 10 pre-existing repository-baseline fixtures, while the exact A21 history range is clean. The standalone/release gate remains intentionally unresolved until the unpushed Browsertools M27/A09 revision is published, and the old browser compatibility lock fails as expected because E11.1 owns its refresh. No publication occurred. |

Evolution result v32 remains absent until E11 passes.
~~~~~~~~~~~~~~~~~~~~
