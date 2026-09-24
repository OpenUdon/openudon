# Retired milestone A17 - A17 Acquisition, Synthesis, And UI Remediation Closure

**Milestone.** A17
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A17.md
**Source status SHA-256.** f133af2fa2c7bb6e3fe31f30dae43d8892503f6bb8f548b134c1503f29191551
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
# A17 Acquisition, Synthesis, And UI Remediation Closure

| Item | State | Notes |
| --- | --- | --- |
| A17.1 Observation-before-read acquisition | `[+]` | Source removal and browser staging observe the workspace before semantic reads, then repeat target, digest, review-append, and absence checks inside the replacement callback after fingerprint comparison. Browser staging requires the engine private root. |
| A17.2 Strict legacy review migration | `[+]` | A migrated v2 singleton requires the digest-derived `legacy-<12 hex>` ID, empty legacy targets, valid timestamps and digests, and the complete safe-evidence invariants used by current records. |
| A17.3 Stable executable and profile bytes | `[+]` | iCoT re-execution hashes source before/after copying and the mode-`0500` destination. Packaged browser profiles are stable non-symlink regular-file reads whose digest and parse share one byte generation. |
| A17.4 Shared effective-source traversal | `[+]` | One traversal covers nested steps, cases, and defaults; quality, side-effect analysis, and lowering select the same effective profile and mutation approvals. Conservative session analysis includes `loop`. |
| A17.5 Safe UI state and strict JSON | `[+]` | Both successful and failed doctor state use the path-free UI shape before storage, ETag, and HTTP serialization. Failed initial revision generation rolls preflight back. JSON depth 64 is accepted and depth 65 rejected. |
| A17.6 Regression proof | `[+]` | Tests cover three acquisition drift points, private-root enforcement, strict migration, executable-copy mutation, nested duplicate action names, stable profile reads, doctor redaction, preflight rollback, and JSON depth. |

A17 preserves the existing API-first and isolated-worker boundaries.
~~~~~~~~~~~~~~~~~~~~
