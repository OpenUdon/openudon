# Retired milestone M08 - Status M08 - Local Checks And Release Process

**Milestone.** M08
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M08.md
**Source status SHA-256.** d306693ad259851e2eb87afafaf6aaa5f6c097911336a1572e99edc7800f3638
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
# Status M08 - Local Checks And Release Process

| Item | State | Notes |
|---|---|---|
| Deterministic repository checks established | `[+]` | Tests, vet, `make check`, document-memory, boundary, schema, and diff checks form the default provider-free gate. |
| Release check established | `[+]` | Deterministic pre-tag readiness is separate from optional real-provider smoke evidence. |
| Provider drift posture documented | `[+]` | Real-provider runs remain local/manual until protected credentials, redaction, and retention automation are approved. |
| Public documentation gate established | `[+]` | Strict docs builds and checked memory boundaries participate in release readiness. |
| Verification completed | `[+]` | Historical release and repository gate acceptance was completed without production execution. |
~~~~~~~~~~~~~~~~~~~~
