# Pending step contracts

`openudon step pending --example DIR --request FILE|-` authors one explicitly
reviewed unresolved contract in `workflows/intent.hcl`. Its additive envelope is
`openudon.step-pending.v1`; it reuses the published `stepContract`,
`intentRevision` and `scaffold` fields from step-authoring v1. The dedicated
[schema](schemas/openudon.step-pending.v1.schema.json) references the existing
schema: offline validators must load both resources explicitly.

A request has `kind: request`, `command: step.pending`, `step_id`, `contract`
(purpose, object field-set input/output schemas and read/write/unknown effect),
`intent_revision`, optional `depends_on`, and a scaffold if intent is missing.
It does not take an operation reference, guessed endpoint, credentials or
execution approval. Callers own approval of these exact authoring writes.
The command uses revision-checked atomic replacement and preserves unrelated
intent blocks/comments. Refusals make no writes. An indeterminate write must be
inspected before retrying. Legacy declarations cannot be silently upgraded.

The intent has `type = "pending"` and a `pending` block with `purpose`, `effect`,
and `inputs`/`outputs` encoded as JSON strings. This preserves public schema
constructs without a second HCL schema dialect. The generated UWS uses the
native 1.12 `pending` object, with the same step ID and contract. Pending-only
packages need no source document. Building publishes their review artifacts,
while its execution-quality gate deliberately fails with pending-step checks.
`assess` reports `workflow.pending_steps` and `uws.pending_steps` separately.

Resolve with `openudon step bind` using the exact declared contract, reviewed
source/operation, explicit mappings and current intent digest. Successful
binding removes the pending block and records the confirmed effect. Changed
contracts require an explicit pending-contract update first. Dependencies,
authentication and source classifications retain their existing bind checks.

New UWS 1.12 packages emit bound operation effects. Rebuilding a declared legacy
version retains its wire shape; it never silently adds 1.12-only semantics.
Effect labels do not authorize actions. Approval templates, package inspection,
dry runs and real runs independently inspect both captured UWS artifacts and
invoke public executable validation for pending packages, before assessment or
executor dispatch. A stored pass or pending contract in an unused workflow or
unselected branch cannot bypass that refusal. Simulation is separate work and
confers no publication or execution authority.
