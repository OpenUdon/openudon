# Enhanced Phase B Reliability And Status UX Result

OpenUdon A09 makes the Phase B engine/write boundary transactional. Round
state and snapshot construction complete before atomic draft persistence, and
request cancellation is no longer consulted after persistence begins.
Approval returns one `ApprovalResult` with its exact approved snapshot and
write result, removes the redundant post-write draft deletion, and performs no
fallible refresh after commit. Typed engine errors now distinguish rejected,
conflict, operational, and indeterminate outcomes.

The shared writer confines every prepared output to the canonical example
root, rejects descendant symlinks and pre-commit swaps, removes temporary
backups after successful commits and completed rollbacks, and makes rollback
failure explicitly indeterminate. A failed backup cleanup after every
replacement succeeds is returned as a non-fatal write-result warning.
Best-effort directory pruning cannot convert a successful write into failure.

Each engine fingerprints fixed project/draft/metadata paths, selected
materialized sources, and every snapshot action. External changes—including a
second engine's write—latch `externally_modified`, change the response revision,
keep cached inspection available, and reject later mutation with
`workspace_changed` until restart. A pre-refresh observation binds targets that
first become selected during a mutation, so their contents cannot be silently
adopted. Forced overwrite cannot bypass this drift boundary, and
unsafe/unreadable paths fail closed operationally.

The loopback server now serves only experimental
`openudon.icot-ui-api.v2` snapshot, round, and approval routes. It supports
ETag/`If-None-Match`, exact instance-root bootstrap, strict recursive
duplicate-name and UTF-8 decoding, closed 400/413/415/409/422/500 mappings,
request IDs and retryability, workspace-aware revisions, 32 KiB headers,
bounded header/read/idle timeouts, COOP/CORP, and sanitized 500-only logs.

The separate embedded shell remains read-only. It polls every two seconds only
while visible, refreshes immediately when visible again, backs off to 30
seconds after failures, retains cached JSON, and displays revision, refresh
time, source/action counts, readiness, top issue, frontier, preview,
completion, and restart-required workspace state. Phase B still has no LAN
bind, accounts, persistent lease/token, folder browser, React/Node, mutation
controls, LLM drafting, workflow execution, or live browser orchestration.
