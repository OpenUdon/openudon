# Per-step run evidence v3

Opt in with `openudon run --executor-report-version v5` (or the exact
`udon.execution-report.v5` name). Omitting it preserves report v3/v4 selection
and `openudon.run-evidence.v2`. Executable run-config v2 already has
`executor_report_version`; explicit v5 forwards `--execution-report-version v5`
and the config's unique `--execution-run-id ID` to Udon. The initial OpenUdon admission requires reviewed OpenAPI-backed HTTP sources;
other source families await independent qualification. V5 excludes browser
execution and admits only M44's one flat HTTP sequence (1–256 unique operations).
Unsupported shape is refused before invocation. No UWS/private runtime code
moves into OpenUdon.

Selecting v5 emits `openudon.run-evidence.v3`, retaining v2's gate/digest/async
contract plus one required `step_execution` observation. Signatures and archives
support v3; older evidence remains readable under its original contract.
The observation contains expected `run_id`, `workflow_id`, `workflow_digest`
(`sha256:` plus exact staged-workflow bytes), and complete ordered expected
step/operation/invocation identities. `report_version` is v5. Invocation ID equals
step ID. No bodies, headers, values or arbitrary diagnostic text is retained.

`state` is `dry_run`, `validated`, `missing`, `invalid`, or `mismatched`.
Only `validated` has `inventory_complete: true`, report status, timestamps,
closed error codes and the exact validated reported outcomes. In every other
state inventory completeness is false and every expected step is `unknown`
without timestamps or error codes. A dry run does not claim `not_started` from
executor observation. A missing or rejected report cannot prove a write absent.
Malformed/duplicate/unknown/oversized records are rejected, with only the fixed
observation class preserved; untrusted raw report bytes are not archived.

A validated report can be `incomplete`: a crash may preserve a complete durable
inventory with some `not_started` steps. That differs from incomplete inventory.
A process-success gate additionally requires a validated `success` report.
An executor failure still produces conservative evidence, including missing or
rejected reports. Every read verifies exact attempt/workflow/inventory and
causal timestamp/result constraints; schema validation alone never authorizes
retry. Retry policy belongs to downstream consumers and requires a fresh approval.

Fixtures in `fixtures/per-step-run-evidence-v3/` publish this observation member
and the corresponding input report where available. They cover dry-run,
success, failed-read/write, interrupted, missing, stale, workflow mismatch,
incomplete inventory and malformed input. Expected identities are in
`expected.json`. Source report schema/fixtures are frozen from accepted Udon
M44 `1a5e9aa2045e3d875da2e18aab2d6db869ac5223`.
