# project.md Schema

`project.md` is a human-readable Markdown contract. OpenUdon parses common headings and optional
machine-readable policy, but reviewers should still treat the whole file as the business and safety
source of truth.

## Recommended Sections

Use these headings for portable authoring:

```markdown
# <Project Name>

## Goal

What the workflow should accomplish in business terms.

## Inputs

Runtime inputs, trigger payload fields, files, or operator-provided values.

## Outputs

Reports, API writes, files, notifications, or returned values.

## External Systems and API Sources

API source files, providers, service names, or `OpenAPI: none required`.

## Data Flow

How fields move between steps, especially when one API response feeds another request.

## Function Contracts

Local `fnct` steps, inputs, outputs, and side effects.

## Runtime Policy

Allowed runtimes and explicit denial of `cmd` or `ssh` when they are not needed.

## Credentials and Secrets

Symbolic credential binding names only.

## Safety and Approval Boundary

What may be generated, validated, sandbox-tested, or executed after approval.

## Fallback Behavior

When OpenUdon should stop instead of guessing.
```

## Parsed Fields

The legacy `--answers` shape maps to these fields:

| Field | Meaning |
| --- | --- |
| `project_name` | Project title. |
| `goal` | Workflow outcome. |
| `inputs` | Runtime input summary. |
| `outputs` | Output summary. |
| `data_flow` | Important field mappings and dependencies. |
| `function_contracts` | Local function contracts. |
| `uses_openapi` | Whether API source documents are expected. |
| `openapi` | API source files, URLs, or service hints. |
| `cmd_approved` | Whether `cmd` runtime is explicitly allowed. |
| `ssh_approved` | Whether `ssh` runtime is explicitly allowed. |
| `side_effect_scope` | `read-only`, `sandbox-only`, or `after-approval`. |
| `credentials` | Symbolic credential binding names. |
| `safety` | Approval and execution boundary. |
| `fallback` | Stop conditions. |

## Optional Policy Block

Add a fenced `openudon-policy` block when policy should be machine-readable:

```openudon-policy
openapi: none required
runtimes:
  cmd: false
  ssh: false
credential_bindings:
  - support_api_token
timeouts:
  workflow: 120
idempotency:
  key: inputs.request_id
  onConflict: returnPrevious
  ttl: 86400
```

Never put credential values, API tokens, passwords, refresh tokens, or production-only endpoints in
`project.md`.
