# Retired milestone M38 - M38 - iCoT Scorecard And Agent Mode

**Milestone.** M38
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M38.md
**Source status SHA-256.** c89029b26f449c3754efeba572edaeee2b0fa5fdf24b960c23c2dd52246e6894
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
# M38 - iCoT Scorecard And Agent Mode

## Goal

Make iCoT reliability measurable and easier to call from local agents without adding MCP or changing
the review-first product boundary.

## Task State

| Item | State | Notes |
|---|---|---|
| Structured author report | `[+]` | `icot --agent --json` emits `openudon.icot-author-report.v1`. |
| Noninteractive agent mode | `[+]` | Complete sessions write final artifacts; incomplete sessions return `needs_input` with readiness issues instead of prompting. |
| Structured lint report | `[+]` | `icot lint --json` emits `openudon.icot-lint-report.v1` with project checks, intent parse, drift warnings, and failure family. |
| Provider-free scorecard | `[+]` | `icot scorecard` emits `openudon.icot-scorecard.v1` over eval seed/build policies without LLM or provider network calls. |
| Documentation | `[+]` | iCoT, session, eval seed/build, and corpus roadmap docs describe agent/JSON/scorecard behavior. |

## Acceptance Evidence

- Focused tests cover agent `needs_input`, complete agent artifact writes, lint JSON, and single-fixture scorecard.
- Post-review hardening adds focused coverage for renderable sessions blocked by low-confidence
  decision evidence, agent loading of `.icot/session.yaml`, and `--report` write failures.
- The scorecard reuses the existing no-LLM seed/build behavior and records expected vs observed outcome, fixture class, failure family, and failure codes.
- Agent mode does not introduce execution or provider calls; it only writes source authoring artifacts when the session is complete.
- Full provider-free scorecard passed on 2026-05-26:
  `go run ./cmd/icot scorecard --root examples/eval --out eval/runs/icot-scorecard-m38-m39`.
  Summary: 56 total, 56 passed, 0 failed; 40 strict-positive, 10 advisory, 6 expected-negative.

## Notes

- `needs_input` is a valid structured authoring result, not a workflow execution failure.
- Existing interactive text output remains the default for human CLI use.
~~~~~~~~~~~~~~~~~~~~
