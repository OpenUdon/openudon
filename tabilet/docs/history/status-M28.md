# Retired milestone M28 - M28 — iCoT Decision Evidence And Repair-Loop Evaluation

**Milestone.** M28
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M28.md
**Source status SHA-256.** 0399cbc112a4887a395aeac548fb42731c3fc5a58ae3106afeb6492d4ac83792
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
# M28 — iCoT Decision Evidence And Repair-Loop Evaluation

## Goal

Make iCoT a stronger interactive plan-and-execute authoring loop by preserving
compact decision evidence, using confidence to drive prompting, and evaluating a
bounded review-repair loop against a small representative automation corpus.

## Status

| Item | State | Notes |
|---|---|---|
| Milestone framed | `[+]` | M28 is planned after M27 and focuses on decision evidence, confidence-driven prompting, optional bounded repair, and sample coverage. |
| Sample corpus selected | `[+]` | Added six M28 OpenUdon-native eval fixtures covering Airtable-style normalization, Gmail audit receipt, Slack/Jira support status, Jira/Slack/Drive incident archive, order fulfillment, and missing-source negative behavior. |
| Decision evidence schema designed | `[+]` | Added compact `decision_evidence` session/transcript state for stage, slot, value, source, confidence, reason, evidence, alternatives, and confirmation requirement. |
| Confidence policy designed | `[+]` | `normal` and `fast` may auto-accept high/review defaults; low/conflict mapping or decision evidence forces an operator question. Ambiguous side-effect commitments now force an operator question even when a likely operation exists. |
| Review repair experiment scoped | `[+]` | Added opt-in `--review-repair` with two bounded repair attempts limited to request mappings, output sources, and `depends_on`; forbidden mutations are rejected. |
| Replay/eval measurement added | `[+]` | `icot replay-eval` accepts `--prompt-mode` and `--review-repair`, defaults replay measurement to `fast`, reports actual transcript prompt counts, LLM-call, repair, rejection, and final flow-review metrics, and lets fixtures set strict warning/review limits. |

## Candidate Samples

Keep the corpus small and workflow-shaped:

- `examples/eval/m28-airtable-normalize`
- `examples/eval/m28-gmail-audit-receipt`
- `examples/eval/m28-support-ticket-slack-status`
- `examples/eval/m28-incident-drive-slack`
- `examples/eval/m28-order-fulfillment-chain`
- `examples/eval/m28-ambiguous-source-negative`

These samples may use n8n, Zapier, Make, Pipedream, and Tray.ai examples as
service/action vocabulary and prioritization evidence, but all committed
fixtures must remain OpenUdon-native and must not depend on those products'
runtime semantics.
~~~~~~~~~~~~~~~~~~~~
