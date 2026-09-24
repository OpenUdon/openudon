# Retired milestone M39 - M39 - iCoT Request Mapping Repair And Decision Evidence

**Milestone.** M39
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M39.md
**Source status SHA-256.** 8b32c03aea2a9d0219aa6cb6c89b67a89c525be72ffd652bf58f36d648b7d3bc
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
# M39 - iCoT Request Mapping Repair And Decision Evidence

## Goal

Improve iCoT at the most common intent-generation weak points: request mapping, bounded repair, and
confidence/evidence-driven blocking.

## Task State

| Item | State | Notes |
|---|---|---|
| Qualified request field prompt context | `[+]` | Request-mapping prompts expose known fields such as `path.ticketId` while preserving validated persisted field keys. |
| Mapping validation hardening | `[+]` | LLM-suggested unknown fields are rejected before entering intent state; qualified aliases normalize only when they map to declared fields. |
| Bounded repair command | `[+]` | `icot repair` supports JSON output, `--dry-run`, `--max-attempts`, and writes only through existing atomic artifact paths. |
| Repair scope enforcement | `[+]` | Repair is limited to request mappings, output sources, and dependency wiring; source, operation, credential, side-effect, and runtime/profile changes are rejected. |
| Decision evidence compatibility | `[+]` | Existing low/conflict decision evidence remains readiness-blocking; agent mode reports those blockers instead of prompting. |
| Documentation | `[+]` | Data-flow and iCoT docs describe qualified mapping, repair scope, and JSON inspection. |

## Acceptance Evidence

- Focused tests cover qualified request mapping aliases and repair dry-run JSON.
- Post-review hardening adds focused coverage for deterministic `depends_on` repair from step-output
  references and rejection of invented request fields in `icot repair`.
- Existing iCoT prompt-mode, replay, seed/build, and lint behavior remains compatible.
- Repair remains provider-free and does not call live providers or trusted executors.
- Full provider-free scorecard remained green after request-mapping and repair changes:
  56 total, 56 passed, 0 failed.

## Notes

- M39 keeps `--review-repair` as the interactive pre-final review repair path and adds `icot repair`
  as the deterministic command surface for existing examples.
- Further natural-language corpus expansion remains deferred to the corpus/provider roadmap, not this milestone.
~~~~~~~~~~~~~~~~~~~~
