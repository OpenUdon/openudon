# Writing project.md for Ramen

`project.md` is the user contract for Ramen synthesis. It should describe the business goal and the
integration policy that tells Ramen when to use OpenAPI, when to use a non-HTTP udon runtime, and
when to stop.

## What To Include

Use these sections for new projects:

- Goal: the workflow outcome in business terms.
- Inputs: trigger payloads, user-provided values, files, or environment-provided bindings.
- Outputs: generated artifacts, API writes, files, notifications, or reports.
- External Systems and OpenAPI: APIs/services involved, OpenAPI files or URLs, or `OpenAPI: none required`.
- Data Flow: important field mappings between steps, especially when one API call feeds another.
- Function Contracts: `fnct` input/output contracts and side effects.
- Runtime Policy: allowed runtimes such as `openapi`, `http`, `fnct`, `cmd`, or `ssh`.
- Credentials and Secrets: credential binding names only; never secret values.
- Safety and Approval Boundary: what may be generated, validated, or executed.
- Fallback Behavior: when Ramen should stop instead of guessing.

For side-effectful workflows, the Safety and Approval Boundary must name both the approval or
trusted-runtime path and the sandbox/test proof-run policy. Ramen synthesis should not directly
execute production workflows.

## Runtime Selection Rules

Ramen should use OpenAPI for API operations when a matching OpenAPI document and operation are
available. OpenAPI should provide method, path, schemas, server, and security metadata.

Ramen should use non-OpenAPI runtimes only when the project explicitly allows them:

- `fnct`: trusted local functions, transforms, renderers, adapters, or private glue.
- `cmd`: approved local commands. Use only with an explicit allow policy.
- `ssh`: approved remote host operations. Use only with an explicit allow policy.
- `http`: direct HTTP behavior or OpenAPI-backed HTTP behavior, depending on the available metadata.

Do not ask Ramen to invent native `smtp`, `sql`, or `llm` semantics unless the project maps that
behavior to an approved `fnct` or a runtime profile implemented by `udon`.

For policy that should be machine-readable, add an optional fenced `ramen-policy` block:

```ramen-policy
openapi: none required
runtimes:
  cmd: false
  ssh: false
credential_bindings:
  - support_api_token
```

This complements the prose sections. Do not put credential values in the block.

## Data Flow

Ramen may expand one business request into multiple technical steps. For example, "search weather
in Toronto, Canada" may require one API call to resolve coordinates and another API call to fetch
weather from `lat` and `lon`.

When you know a mapping, write it explicitly:

```md
Pass `get_coordinates.body[0].lat` to `get_weather.lat`.
Pass `get_coordinates.body[0].lon` to `get_weather.lon`.
```

When you do not know the hidden API steps, describe the business goal and let Ramen infer them from
OpenAPI metadata. Ramen should expose inferred substeps and bindings in generated artifacts. See
`docs/data-flow.md` for examples.

Use structural steps when the project needs explicit branching or iteration. A loop project should
name the item source, any batch-size policy, nested work, and the output that should become the
named structural result.

## OpenAPI Policy

If the project needs API calls, provide one of these:

- OpenAPI files under `openapi/`.
- OpenAPI document URLs in `project.md`.
- Search/discovery hints precise enough for Ramen to find the relevant API document.

If the project does not need API calls, write this exact policy:

```md
OpenAPI: none required
```

When that phrase is present, Ramen should not fail only because `openapi/` is empty. It should also
reject generated artifacts that still reference OpenAPI.

## Example

```md
# Support Ticket Draft

## Goal

When a ticket is created, fetch the ticket details, classify the request, and write a draft reply.

## Inputs

- `ticket_id`: required string from the incoming event.

## Outputs

- A stored draft reply record.
- A validation report for the generated workflow.

## External Systems and OpenAPI

- Support API: use `openapi/support.yaml`.
- OpenAPI is required for ticket lookup.

## Runtime Policy

- `openapi`/`http` allowed for the Support API.
- `fnct` allowed for `classify_ticket` and `write_draft`.
- `cmd` and `ssh` are not allowed.

## Data Flow

- Pass `get_ticket.received_body` to `classify_ticket.ticket`.
- Pass `classify_ticket.received_body` to `write_draft.classification`.

## Function Contracts

- `classify_ticket`
  - Inputs: ticket body from `get_ticket`.
  - Outputs: classification label and rationale.
  - Side effects: none.
- `write_draft`
  - Inputs: ticket body and classification.
  - Outputs: draft record.
  - Side effects: writes a draft only.

Each generated `fnct` step must have a matching Function Contracts entry. Declared function inputs
must be wired in intent through `with`, `bind`, or prior-step references so review can audit where
adapter inputs came from.

## Credentials and Secrets

- Use credential binding `support_api_token`.
- Do not include credential values in generated artifacts.
- OpenAPI `securitySchemes` and operation security requirements must map to named credential
  bindings. If a secured operation requires `api_key`, declare a binding such as
  `support_api_key` and wire the request field by binding name, never by secret value.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not send any outbound customer message.
- Use sandbox endpoints for proof runs before any production handoff.
- Production execution requires human approval and trusted-runner handoff.

## Fallback Behavior

- Stop if the Support API OpenAPI document is missing.
- Stop if no approved runtime exists for classification or draft storage.
```
