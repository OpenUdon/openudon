# M36 - Slack Smoke And v0.1.2 Tag Gate

## Goal

Run one manual real Slack sandbox smoke through the approved OpenUdon trusted-runner handoff before
tagging `v0.1.2`. This is release confidence evidence, not a new product feature and not a
provider-free release gate.

## Status

| Item | State | Notes |
|---|---|---|
| Smoke fixture selection | `[+]` | Used an ignored `.openudon-run/m36-slack-smoke/slack-message-audit-log` package copied from `examples/eval/slack-message-audit-log`, narrowed to a single Slack `postMessage` HTTP step, with real Slack API path `/api/chat.postMessage` and bearer auth. |
| Slack sandbox policy | `[+]` | Operator provided `OPENUDON_SLACK_CHANNEL_ID` and `UDON_CREDENTIAL_SLACK_BOT_TOKEN`. Runtime data uses an env marker for the channel and non-secret smoke text. Token value stayed outside artifacts. |
| Trusted executor setup | `[+]` | Built local ignored udon binary `/home/peter/Workspace/udon/dist/udon-openudon-m36` from `../udon` head `0404b29` and used it through `OPENUDON_EXECUTOR`. |
| Package build and review | `[+]` | Smoke package build/assess passed. Sandbox approval JSON was generated under ignored `.openudon-run/m36-slack-smoke/approvals/`. |
| Dry-run handoff | `[+]` | `openudon run --dry-run` passed after rebuilding the final package. Digest: `ff0836319380416d1eba3400ad2d125cfd4d908f2c2285b49c186fbf8aac1d5f`; credential binding: `slackBearer`; data file: `expected/data.hcl`. |
| Live Slack smoke | `[+]` | Non-dry `openudon run` completed through local udon with executor stage evidence and digest `ff0836319380416d1eba3400ad2d125cfd4d908f2c2285b49c186fbf8aac1d5f`. Udon did not persist Slack response JSON, so the operator confirmed the message `OpenUdon v0.1.2 Slack smoke test` appeared in the target Slack channel. |
| Tag decision | `[+]` | Slack smoke passed with operator channel confirmation. Tagged OpenUdon `v0.1.2` on commit `3373cd4`. |

## Boundary Notes

- M35 remains the provider-free release-readiness baseline.
- M36 may call real Slack only through `openudon run` after approval and trusted-runner validation.
- Do not add real Slack execution to `make release-check`, `make release-saas-check`, or public CI.
- Do not commit Slack tokens, approval JSON, run configs, raw provider responses, or private channel
  content.
- If a reusable OpenUdon/UWS/udon defect is found, fix it in the owning repo before tagging.
