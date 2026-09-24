# Retired milestone M29 - M29 - iCoT Flow Remediation And Draft Expansion

**Milestone.** M29
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M29.md
**Source status SHA-256.** 28d897f618cc83d54dc75bb3e12e1089da335290cc0aaba50716086648cfe99d
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
# M29 - iCoT Flow Remediation And Draft Expansion

## Goal

Make iCoT produce the best defensible draft from incomplete natural-language
goals, then stop at the exact point where the operator must clarify intent.

## Status

| Item | State | Notes |
|---|---|---|
| Milestone framed | `[+]` | M29 follows M28 and focuses on general flow remediation, not a weather/Gmail-specific fix. |
| Gap taxonomy implemented | `[+]` | Draft-review issues now carry locally validated `gap_kind` values for missing transform/report step, missing API prework, disconnected notification/message, ambiguous output, operation mismatch, unavailable source/artifact, unclear intent, and narrow repair. |
| Remediation action model implemented | `[+]` | Draft-review issues now carry `remediation_action` values for narrow repair, proposed local `fnct` step, proposed API prework, forced user question, and comment-only warning; invalid model values fall back to deterministic classification. |
| Structural draft expansion implemented | `[+]` | `--review-repair` may add a local `fnct` transform/report/render step only when the goal clearly asks for produced content and exactly one existing producer step can feed it. |
| Ambiguity stop rule implemented | `[+]` | Flow-review issues marked `ask_user` and ambiguous provider-as-verb side-effect goals force one operator question even in `fast` mode; unresolved issues remain visible as non-executable `intent.hcl` comments. |
| Focused replay follow-up implemented | `[+]` | Catalog rough steps are advisory-only, operation defaults require exactly one filtered local operation, fnct remediation renames raw default result outputs, and no-source gap-report fallback is deterministic. |
| Focused real-model replay closure completed | `[+]` | Real replay with `--review-repair` passed `m28-gmail-audit-receipt` and `m28-ambiguous-source-negative`; `m28-airtable-normalize` and `itops-workflow-backup-github` now stop at explicit operation-selection questions with no rough-step leakage and no silent first-operation default. |
| No-source fallback docs polished | `[+]` | Added operator-facing guidance that missing/ambiguous provider source evidence can produce a local `render_capability_gap` report instead of an invented API workflow. |
| Response-field metadata expansion follow-up | `[+]` | M65 expands review-repair response metadata for local OpenAPI `$ref`, nested object, array, and Swagger `schema` response fields. Discovery/Smithy response-field claims remain source-family evidence work. |
| Deterministic API prework generalization follow-up | `[+]` | M65 adds a bounded deterministic API prework path: exactly one listed local GET/HEAD operation, no required inputs, and a schema-visible response field matching the missing request field. Unsafe or ambiguous prework still rejects. |
| Replay quality gate polish follow-up | `[+]` | M65 adds the focused `make icot-replay-repair-check` gate for known `cmd/icot replay-eval --prompt-mode fast --review-repair` repair fixtures. |
| Eval coverage expanded | `[+]` | Added unit coverage for taxonomy fallback, invalid model values, safe `fnct` expansion, ambiguous producer rejection, comment-only safety classes, forced fast-mode ambiguity prompts, no-source fallback review skip, no-default operator stops, and local `fnct` output review false positives. |

## Design Notes

- iCoT should maximize locally defensible structure from the brief, catalog,
  selected operations, schemas, deterministic checks, and decision evidence.
- It must not silently invent provider operations, API artifacts, credentials,
  side-effect policy, or user intent.
- Unresolved flow issues remain useful output when they precisely explain what
  the user must refine.
- Structural remediation remains behind `--review-repair`; default iCoT keeps
  flow review advisory except for true forced ambiguity questions.
~~~~~~~~~~~~~~~~~~~~
