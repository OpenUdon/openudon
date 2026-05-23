# Writing project.md for OpenUdon

`project.md` is the user contract for OpenUdon synthesis. It should describe the business goal and the
integration policy that tells OpenUdon when to use OpenAPI, when to use a non-HTTP udon runtime, and
when to stop.

`go run ./cmd/icot --example examples/<name>` is an optional guided authoring tool. With LLM
assistance available, it starts from one plain-language goal, drafts `workflows/intent.hcl` from
that answer plus local API source metadata, and asks only the next blocking question needed to reach a
valid intent. With `--no-llm`, it uses the fixed manual prompt flow. `project.md` remains the OpenUdon
policy/prose artifact, while `workflows/intent.hcl` is the structured saved contract that `openudon
build` consumes next.

`icot` is deterministic. It can print without writing (`--print`), seed prompts from another
example (`--from-example`), render from YAML or JSON answers (`--answers`), resume interrupted
interactive sessions from `.icot/session.yaml`, reconcile `project.md` from existing intent
(`icot reconcile --example examples/<name>`), and lint an existing brief plus intent drift (`icot
lint --example examples/<name>`). Drift findings are warnings unless a parse or existing fail check
also fails.

When provider credentials are available, `icot` uses AI assistance to draft operation choices,
request mappings, outputs, credentials, and policy prose from the brief plus local API source metadata.
After each answer, deterministic readiness checks decide whether to ask about the goal, API
document, operation, required request values, credential bindings, runtime inputs, outputs, or
safety policy. The first valid intent jumps to final review; remaining warnings and inferred values
are shown as assumptions, and saving confirms them. When LLM extraction is enabled, iCoT also runs a
single advisory pre-final flow review that looks for cross-step data-flow mistakes such as a report
email step not consuming report content. Flow warnings are classified into remediation actions and
kept as visible `intent.hcl` comments when they are not automatically repaired. Experimental
`--review-repair` can apply bounded wiring repairs or add a local `fnct` transform/report step when
the existing draft has one defensible producer; it does not change API sources, operations,
credentials, or side-effect scope. The saved `intent.hcl` is a useful starting draft for
build/review, not a promise that iCoT found the perfect workflow; operators should reject bad drafts
or confirm and continue editing manually.

Prompt volume is controlled by `--prompt-mode full|normal|fast`. Omitted mode is `full`, which asks
every question and waits for confirmation. `normal` prints high-confidence and review-level defaults
and accepts them automatically, but still asks for missing, low-confidence, or conflicting answers.
`fast` silently accepts safe defaults and suppresses catalog/status chatter plus review-only
assumption text while preserving transcript and decision evidence.

For catalog-backed SaaS briefs, iCoT first checks local `openapi/`, `google-discovery/`,
`aws-smithy/`, and legacy `discovery/` documents plus the sibling `../apitools` first-class provider
cache. If a local API artifact is missing, iCoT tries to retrieve or materialize first-class
apitools artifacts or reviewed advisory OpenAPI overlays into the workflow before asking for an API
path. It only asks for a user-provided artifact after apitools reports that no first-class or
advisory source artifact is available. Discovery and Smithy documents can drive operation review,
synthesis, packaging, and trusted handoff directly. When both an original provider OpenAPI document
and a reviewed advisory OpenAPI overlay are available, iCoT defaults to the advisory overlay for
operation selection because it carries OpenUdon-reviewed endpoint/security scope.

iCoT defaults to the local `copilot-api` gateway, using `COPILOT_API_BASE_URL` when set and
`http://localhost:4141` otherwise. Use `OPENUDON_LLM_PROVIDER` and `OPENUDON_LLM_MODEL` for
shell-level overrides, or pass `--provider` and `--model` when you want an explicit provider
selection. Provider credentials stay in provider-native environment variables such as
`COPILOT_API_KEY`, `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or `GEMINI_API_KEY`; those keys do not
change the iCoT provider unless `OPENUDON_LLM_PROVIDER` or `--provider` selects them.

## What To Include

Use these sections for new projects:

- Goal: the workflow outcome in business terms.
- Inputs: trigger payloads, user-provided values, files, or environment-provided bindings.
- Outputs: generated artifacts, API writes, files, notifications, or reports.
- External Systems and API Sources: APIs/services involved, API source files or URLs, or `OpenAPI: none required`.
- Data Flow: important field mappings between steps, especially when one API call feeds another.
- Function Contracts: `fnct` input/output contracts and side effects.
- Runtime Policy: allowed runtimes such as `openapi`, `http`, `fnct`, `cmd`, or `ssh`.
- Credentials and Secrets: credential binding names only; never secret values.
- Safety and Approval Boundary: what may be generated, validated, or executed.
- Fallback Behavior: when OpenUdon should stop instead of guessing.

For a concise field-level reference, see [project.md Schema](project-authoring-schema.md).

For guided authoring, choose one side-effect scope:

- `read-only`: artifact generation and validation only; no workflow execution or external effects.
- `sandbox-only`: sandbox proof runs only after `approved_for_sandbox` through a trusted runner.
- `after-approval`: sandbox and production execution require the existing approval/trusted-runner
  states and approved credential bindings.

Guided authoring also accepts optional workflow timeout, workflow idempotency, and per-step timeout
answers. Leave those prompts blank unless the project contract requires portable UWS 1.1 metadata.

For side-effectful workflows, the Safety and Approval Boundary must name both the approval or
trusted-runtime path and the sandbox/test proof-run policy. OpenUdon synthesis should not directly
execute production workflows. Review evidence treats generated artifacts as Symphony state
`generated`; side-effectful proof runs require `approved_for_sandbox`, and production execution
requires `approved_for_production`.

## Runtime Selection Rules

OpenUdon should use API source documents for API operations when a matching document and operation are
available. The source should provide method, path, schemas, server, and security metadata.

OpenUdon should use non-OpenAPI runtimes only when the project explicitly allows them:

- `fnct`: trusted local functions, transforms, renderers, adapters, or private glue.
- `cmd`: approved local commands. Use only with an explicit allow policy.
- `ssh`: approved remote host operations. Use only with an explicit allow policy.
- `http`: API-source-bound HTTP behavior when a reviewed OpenAPI, Google Discovery, or AWS Smithy
  source is available.

Do not ask OpenUdon to invent native `smtp`, `sql`, or `llm` semantics unless the project maps that
behavior to an approved `fnct` or a runtime profile implemented by `udon`.

For policy that should be machine-readable, add an optional fenced `openudon-policy` block:

```openudon-policy
openapi: none required
runtimes:
  cmd: false
  ssh: false
credential_bindings:
  - support_api_token
timeouts:
  workflow: 120
  steps:
    call_api: 10
idempotency:
  key: inputs.request_id
  onConflict: returnPrevious
  ttl: 86400
```

This complements the prose sections. Do not put credential values in the block.

## Data Flow

OpenUdon may expand one business request into multiple technical steps. For example, "search weather
in Toronto, Canada" may require one API call to resolve coordinates and another API call to fetch
weather from `lat` and `lon`.

When you know a mapping, write it explicitly:

```md
Pass `get_coordinates.body[0].lat` to `get_weather.lat`.
Pass `get_coordinates.body[0].lon` to `get_weather.lon`.
```

When you do not know the hidden API steps, describe the business goal and let OpenUdon infer them from
OpenAPI metadata. OpenUdon should expose inferred substeps and bindings in generated artifacts. See
`docs/data-flow.md` for examples.

Use structural steps when the project needs explicit branching or iteration. A loop project should
name the item source, any batch-size policy, nested work, and the output that should become the
named structural result.

## OpenAPI Policy

If the project needs API calls, provide one of these:

- OpenAPI files under `openapi/`.
- OpenAPI document URLs in `project.md`.
- Search/discovery hints precise enough for OpenUdon to find the relevant API document.

If the project does not need API calls, write this exact policy:

```md
OpenAPI: none required
```

When that phrase is present, OpenUdon should not fail only because `openapi/` is empty. It should also
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
- Sandbox proof runs require `approved_for_sandbox`.
- Production execution requires `approved_for_production`, human approval, trusted-runner handoff,
  and approved credential bindings.

## Fallback Behavior

- Stop if the Support API OpenAPI document is missing.
- Stop if no approved runtime exists for classification or draft storage.
```
