# Enhanced Phase B Reliability And Status UX

Promote A09 instead of reopening A08. Keep `icot ui` single-workspace,
IPv4-loopback-only, dependency-light, and read-only in the browser while
closing mutation lifecycle, workspace drift, transport resource, and status UX
gaps.

Make engine mutations prospective transactions. A round must build its new
state and snapshot before atomic draft persistence, and approval must return an
exact prebuilt snapshot with the write result so no fallible post-commit refresh
can turn success into a 500. Add typed engine failures and harden the shared
writer's backup cleanup, rollback classification, canonical-root containment,
and descendant-symlink checks.

Fingerprint engine-owned project, draft, final, metadata, and materialized
source paths. Detect external edits and competing engines optimistically,
preserve cached inspection, revise workspace status, and block mutation until
restart without a persistent lease.

Replace experimental API v1 with v2 snapshot, round, and approval routes. Add
conditional snapshots, exact bootstrap exchange, strict recursive duplicate
and UTF-8 JSON handling, media/body/header/time limits, retryable request-ID
errors, safe 500-only logs, and COOP/CORP. Poll the read-only shell while
visible with bounded error backoff and a prominent restart-required state.
