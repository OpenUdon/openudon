# n8n Pattern Bridge

The n8n pattern bridge is a review-first authoring aid. It records useful n8n
evidence as a small OpenUdon summary so an operator or agent can draft a native
`project.md` and `workflows/intent.hcl` candidate without importing n8n runtime
behavior.

It does not execute n8n workflows, translate n8n JSON into UWS, preserve n8n
item semantics, or emulate n8n credentials, triggers, scheduling, binary data,
or expression evaluation.

## Summary Contract

Bridge summaries live at `examples/eval/<name>/reference/n8n-bridge.json` and
use:

```json
{
  "version": "openudon.n8n-pattern-summary.v1",
  "fixture": "n8n-slack-message-post",
  "boundary": "authoring_assistance_only",
  "source": {
    "kind": "n8n_workflow_fixture",
    "paths": ["reference/n8n.json", "../try-n8n/NODE_MATRIX.md"]
  },
  "services": [{"name": "Slack", "operations": ["postMessage"]}],
  "nodes": [{
    "name": "Slack",
    "type": "n8n-nodes-base.slack",
    "resource": "message",
    "operation": "post",
    "openudon_step": "post_message",
    "openapi_operation_id": "postMessage",
    "mapping_status": "advisory"
  }],
  "generated_candidates": {
    "project_path": "project.md",
    "intent_path": "reference/intent.hcl",
    "promoted": false
  },
  "validation": {"status": "advisory"}
}
```

The summary records:

- source evidence paths, including fixture-local `reference/n8n.json` files or
  `../try-n8n` scanner evidence;
- services, n8n node names, n8n resource/operation pairs, OpenUdon step names,
  and OpenAPI operation candidates;
- symbolic credential binding names only;
- data-flow hints when a later step depends on a prior response;
- unsupported semantics as diagnostics or TODOs;
- validation status for any candidate `project.md` or `intent.hcl`.

## Unsupported Semantics

Unsupported n8n behavior must stay visible. Summaries should use
`unsupported_semantics` entries instead of silently dropping behavior.

| n8n behavior | OpenUdon bridge handling |
| --- | --- |
| Triggers and webhooks | Diagnostic or TODO until modeled by a public UWS/OpenAPI or approved runtime contract. |
| Schedules and wait nodes | TODO; OpenUdon does not infer timing behavior from n8n. |
| Expressions | Diagnostic; convert only reviewed request fields and response paths. |
| Item batching and pagination | TODO; require explicit loop or pagination policy. |
| Binary data | Manual contract; use explicit OpenAPI body mapping or an approved `fnct` adapter. |
| Custom code | Manual contract; rewrite as reviewed `fnct` behavior if allowed. |
| Credentials | Symbolic binding names only; never copy credential values. |

## Local Validation

Validate the checked-in bridge summaries with:

```bash
go run ./cmd/openudon n8n-bridge validate --root examples/eval
```

Validate one summary with:

```bash
go run ./cmd/openudon n8n-bridge validate --file examples/eval/n8n-slack-message-post/reference/n8n-bridge.json
```

This command checks the summary contract and prints the fixture validation
status. It does not read upstream n8n workspaces, run `../try-n8n`, generate
project files, call providers, or run a UWS executor.

## Promotion Boundary

A bridge summary can guide authoring, but promotion still requires the normal
OpenUdon path:

1. write an OpenUdon-owned `project.md`;
2. write or generate `workflows/intent.hcl`;
3. ensure local OpenAPI operation IDs expose the needed request, response, and
   credential fields;
4. run iCoT lint, build, quality, review, package, approval, and trusted-runner
   dry-run gates;
5. keep the fixture advisory until unsupported semantics are either modeled as
   explicit OpenUdon behavior or intentionally removed from scope.
