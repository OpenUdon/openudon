# Retired milestone P05 - P05 Prepare-Only Build And Atomic Promotion

**Milestone.** P05
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-P05.md
**Source status SHA-256.** b0a6adc3ed46763dfc04b33c8f95d2dc6561a74ddc3173467bdcb0e2a17fa248
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
# P05 Prepare-Only Build And Atomic Promotion

| Item | State | Notes |
| --- | --- | --- |
| P05.1 Pure package preparation | `[+]` | Complete at OpenUdon `21a32edb982da399cc6e05fd97d6a1f350aca4d6`. `internal/packagepipeline.PrepareCurrent` reads one bounded, strict, manifest-complete package generation twice, rejects path/inventory/byte drift and an unexpected optimistic input digest, and returns defensive in-memory bytes plus a value-free portable manifest containing compatible package/input/manifest, handoff, and quality digests, passing quality status, approval states, execution policy, and symbolic credential names. It requires an explicit portable scope, caps file count and aggregate bytes, and writes no project, package, pointer, approval, report, or runtime state. Determinism, immutable-read drift, cancellation, no-write tree identity, defensive copies, local-path omission, trusted package-digest compatibility, focused, race, vet, and diff checks pass. Task review iteration 1 found a local-root accessor, path-derived default scope, and missing aggregate bounds; all three are removed/fixed, and iteration 2 is clean. |
| P05.2 Restrictive scratch qualification | `[+]` | Complete at OpenUdon `dcb5686ffe175b1dfe62f1e9735af288eaed1828`. `packagepipeline.Qualify` materializes only the immutable prepared bytes beneath a fresh same-filesystem mode-0700 root, enforces mode-0700 package directories plus mode-0600 single-link regular files, rejects aliases and unsupported members, re-prepares the exact scratch generation, reruns current quality/secret and package/handoff inspection, and completes a trusted dry-run without executor invocation. It removes the anchored scratch tree on success and every failure, returns closed typed failures and deterministic value-free evidence, and keeps source bytes unchanged. Focused, repeatability, cancellation, unsafe-mode, symlink, hard-link, alias, unsupported-member, injected-dry-run/cleanup, stale-quality, race, vet, strict-doc, and diff checks pass. Task review iteration 1 clarified the scratch evidence posture and added repeatability coverage; iteration 2 is clean. |
| P05.3 Atomic package promotion | `[+]` | Complete at OpenUdon `319babd8033a871143f665c3c79358bf4ef23129`. `packagepipeline.Promote` validates the exact qualification, publishes a mode-restricted immutable generation under a record-derived SHA-256 identity, synchronizes files and directories where supported, and performs the only ownership transition by atomically replacing a strict value-free `current.json`. The selector binds exact selected/prior generation, package, scope, qualification, and self digests; repeated selection is idempotent, no generation is deleted, current readers ignore staging, and a create-only store lock makes competing builders fail closed. Focused/full, 20-repeat concurrency, old/new reader, generation-collision, cancellation, restrictive-mode, race, vet, strict-doc, boundary-path, and diff checks pass. Task review iteration 1 required exact persisted qualification-posture validation and single-link selection/generation metadata; both are fixed, and iteration 2 is clean. |
| P05.4 Rollback and indeterminate outcomes | `[+]` | Complete at OpenUdon `e3c7497409d1b2d2c9c06e344120b99e6a46d2e4`. Promotion now persists a strict value-free target/baseline intent behind an atomic create-only digest-bound lock. Failures before selector replacement return typed `rolled_back`; failures after replacement or cleanup ambiguity return `indeterminate`, expose only the recoverable target generation digest, retain exact intent/lock evidence, and block blind retry. `InspectRecovery` validates current, prior, target, intent, lock, and a 128-entry transient bound; `Reconcile` requires and rechecks its exact report digest, rejects drift, and removes only anchored staging/selector-temp/intent/lock artifacts without rewriting `current.json` or deleting any generation. Every intent/generation/selector/sync/cleanup boundary, both cancellation sides, target-absent and target-complete restarts, deterministic reports, stale selector drift, wrong digest, orphan cleanup, transient bounds, and simultaneous selected/prior/target preservation pass focused/full/race/vet/strict-doc/diff gates. Task review iteration 1 added the rolled-back target identity and explicit three-generation preservation test; iteration 2 is clean. |
| P05.5 Existing build compatibility and bounded review | `[+]` | Complete at OpenUdon `2121f06d6eab173012ab9d2a0f797ea45cece617`. `PrepareAndQualifyCurrent`/`PromoteCurrent` and exact-selector inspection, approval-template, and trusted-run adapters form the package-engine compatibility boundary. The CLI adds explicit `package prepare|promote|inspect|recover` without changing legacy artifact `openudon promote`; prepare retains no filesystem state, selection requires explicit promotion confirmation, recovery requires the exact observed report digest, and selected approval/run replace `--example` with exact `--package-store`/`--selection`. Selected run requires approval/work directories outside the immutable store, resolves symlinked ancestors, reruns real assessment, preserves identical package/approval/handoff/dry-run bytes and schemas, and leaves executor authority plus BRP pre-executor rejection in trustedrunner. Existing iCoT engine/UI/package-review tests remain compatible; A20 is reconciled to consume these adapters rather than duplicate package logic. Focused CLI/package/trustedrunner/engine/UI, full, full-race, vet, standalone, release-check, docs, memory, boundary, stale-selection, symlink-containment, no-write, and diff gates pass. The all-package race initially exhausted `/tmp` quota for three unrelated link steps; all three passed independently with `GOTMPDIR` on `/var/tmp`. Task review iteration 1 corrected transient-qualification help wording and added canonical outside-store path resolution; iteration 2 is clean with no P1/P2 finding. |

Dependencies: OpenUdon M77 and A19. Retention deletion, runtime execution,
deployment, and dependency upgrades remain separately authorized operations.

M77 reconciliation: `prepared` adds only exact package and qualification
digests to the immutable transaction; `promoted` retains them and adds the
complete generation digest. `indeterminate` is reserved for ambiguous
promotion and requires `promotion_indeterminate`; reconciliation may return to
the same prepared generation or prove it promoted. No prepare/promote state is
runtime authority, and selected/prior generations remain preserved.

A19 completion reconciliation: preparation may consume virtual BAP/BCP/BRP
bytes only after exact candidate rediscovery rehydrates the selected in-memory
plan. It must include BRP profile and adjacent independent review together,
bind the freshness-validated source/review identities, and fail if a candidate
is missing, stale, replaced, or deselected. Source changes invalidate their
authoring approvals before preparation; private producer envelopes, paths, and
byte bodies remain absent from durable transaction state.
~~~~~~~~~~~~~~~~~~~~
