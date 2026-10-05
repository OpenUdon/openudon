# Prompt v49 — Stage 9 approved direction

Approved 2026-10-05 by the user's “Implement the plan” response to the complete memory-bank-propose proposal. [Stage 9 contract](../../../kinet/docs/stage9.md) records decisions and boundaries. This repository owns M97; each other owner keeps its own ledger. Planning baseline `fbda7e9231b8b306fd1ae3ac623e9d70331b3e08`.

The new direction is hosted user-approved HTTP reads/writes, Kinet-metered AI summaries, verified-owner notification email and explicit bounded schedules. Generic broker transport belongs to Udon, reviewed handoff to OpenUdon, and user authorization/networking/credentials/model/mail/scheduling to Kinet. Workers remain network=none. New broker/configuration/evidence versions preserve old contracts. Kinet targets app/audit v9 and first clears all 40 existing staticcheck diagnostics without a waiver.

Approved serial order: `Kinet:M34 -> Udon:M46 -> OpenUdon:M97 -> Kinet:M35 -> Kinet:A14 -> Kinet:W14 -> Kinet:M36 -> Kinet:U12 -> Kinet:M37`. One execution owner reconciles exact accepted and published upstream revisions before advancing. Udon/OpenUdon publication requires separate confirmed normal-push authority. The endpoint is a qualified bundle; existing installed M28, M17 history, public signup/disclosure gates and prior runtime acceptance are not altered.

Default recurring permission: hourly/daily/weekly, 30 days, at most 1000 runs, USD 0.05 model/run and USD 1 model/grant, with existing monthly limits. No automatic renewal, no catch-up, and uncertain writes/email pause for owner review without resend. OAuth/Gmail, browser replay, P-to-W, authoring tiers 2–3 and live activation stay deferred.

This material contract/authority direction justifies a new evolution version. No code, test, dependency, installed service or live effect is changed by this snapshot. Later task-commit execution is a separate request using the Kinet launch reference and each owner's rules.
