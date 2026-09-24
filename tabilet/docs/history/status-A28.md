# Retired milestone A28 - A28 Status - Private Registration Discovery Inventory

**Milestone.** A28
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A28.md
**Source status SHA-256.** 4a7c0fd52cfe00f57cd4a6784f648c9067f5f58e4d92e1d4826fe70e6d862ee3
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
# A28 Status - Private Registration Discovery Inventory

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Complete. Registration discovery inventory, coverage limitations and owner
review are private shared-application state exposed by iCoT and supervised
control. UWS keeps portable recipes and published 1.1 compatibility. Both source
commits are published and independently verified at their existing origins.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A28.1 Implement and verify private discovery authoring | `[+]` | Implemented private inventory/history, independent coverage/review, operator-added and native-observed candidates, explicit wizard selection and HTTP/control parity. Browser authority, consumed attempts and canonical recipe bytes are preserved. Full tests/vet in both repositories, make check, focused race, all 18 sandboxed UI tests and documentation checks pass. |
| A28.2 Publish and reconcile the boundary | `[+]` | Published UWS `9ff877ebce55fba345b7d59ebb838db521822c67` then OpenUdon `539eb8ff99777673fe2dabfa9c080249ff47c0f9`; both exact origin/main revisions independently verified. Product, architecture, tech stack, dashboard/index and evolution v36 reconciled in this canonical harness unit. Live adoption, typed-input runtime adoption and W8M registration remain separate. |

## Dependencies and acceptance

- Published UWS registration 1.1 remains immutable; its optional discovery field
  remains accepted. New authoring exports omit discovery metadata.
- Browsertools remains the observation producer; this milestone adds no crawler
  or claim of exhaustive discovery and no dependency/runtime upgrade.
- Inventory and history must stay outside examples, packages and Git. Ordinary
  application snapshots must not expose unreviewed inventory URLs.
- Inventory mutations cannot start a browser, grant authority, reset an attempt,
  or silently rewrite a reviewed profile. Selection only prepares editable UI
  fields; the existing observation and draft review remain required.
- Verify stale writes, independent review/coverage, private persistence,
  malformed inputs, snapshot/export isolation, both adapters and loopback UI.

## Bounded review-fix gate

Maximum 10 iterations. No P1/P2 or higher-severity finding may remain at closure.
Review iteration 1 corrected inventory polling that could overwrite unsaved
coverage controls, rejected gaps and unexpected files in private history, and
enforced the encoded revision-size limit before writing. Regression tests cover
oversize rejection without damaging the prior revision and competing writers.

Review iteration 2 passes: bounded review of source ownership, HTTP/control parity,
snapshot and canonical-draft isolation, storage permissions, publication scope,
and browser authority preservation. No P1/P2 or higher-severity finding remains.
Final documentation-memory, UWS example validation, formatting and diff checks
pass. Both focused source commits are published through normal fast-forward
pushes; unchanged origins were verified independently.

## Verification

- UWS full Go tests and vet; optional discovery absence and legacy round trips.
- OpenUdon full Go tests, vet and `make check`.
- Focused registration inventory/UI tests with the race detector.
- All 18 sandbox-required loopback UI tests pass, including selection without
  target launch, inventory reload and the existing guided registration wizard.
- JavaScript syntax and working-tree diff checks pass.
- Documentation-memory and UWS example validation pass. UWS versioned schema,
  spec and archive bytes and OpenUdon module pins are unchanged.
- No live target, account credential, qualified runtime adoption or W8M mutation
  is involved. The existing W12 operating blocker and earlier evidence remain.
~~~~~~~~~~~~~~~~~~~~
