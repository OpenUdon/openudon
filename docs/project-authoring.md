# Writing project.md for OpenUdon

`project.md` is the user contract for OpenUdon synthesis. It should describe the business goal and the
integration policy that tells OpenUdon when to use OpenAPI, when to use a non-HTTP udon runtime, and
when to stop.

Kinet owns the adaptive interview and user-confirmation ledger. OpenUdon's
`authoring draft` accepts explicitly reviewed local seeds or existing session
artifacts without terminal input. Use print for read-only review, yes for complete
publication, or the returned structured frontier to supply missing decisions.
`openudon authoring reconcile` reconstructs a brief from intent, and
`openudon authoring lint` checks parseability and advisory drift. Generic Authoring
frontier/transaction mechanics remain shared, with native source and writer rules.
See [current migration](authoring-retirement.md) for precise command/approval
boundaries. Expert evaluation/replay can explicitly invoke a model; provider
credentials alone authorize no model call, publication or external action.

For SaaS briefs, neutral discovery checks existing sources, explicit `--api-source`/`--openapi` documents,
explicit `--browser-profile ID=PATH` inputs, and explicit `--source-root` paths before rendering or publication.
Bounded apitools discovery validates
OpenAPI/Swagger, Google Discovery, AWS Smithy, AsyncAPI, GraphQL, OpenRPC, gRPC/protobuf, and OData;
rejects symlinks; deduplicates by digest; and treats ambiguous JSON/XML as a blocker until its kind is
declared. It never copies a source before proposal approval. If local evidence is exhausted, an
approved remote lookup is limited to curated apitools references plus one APIs.guru request and
returns metadata candidates rather than materializing files. Protocol execution remains in the
trusted executor boundary.

When no adequate API operation covers the active outcome, iCoT can select a verified
`uws.browser.1.5` profile from Browsertools. `--browser-registry PATH|HTTPS_URL` searches a static,
read-only catalog; local catalogs work with `--network never`, while HTTPS catalogs require their own
network approval, separate from API discovery. The selected action, origin, digest, lifecycle,
runtime session posture, and any operation-specific mutation approval are shown before writing.
OpenUdon never stores browser sessions, drivers, credentials, or raw captures.

The same `--browser-profile ID=PATH` input and bounded roots also accept
secret-free `uws.browser-authentication.1.0` profiles. When a reviewed browser
action requires login state, iCoT can author one explicit sign-in/MFA flow that
establishes an execution-local named session, then bind the protected action to
that session. It records only symbolic credential bindings and safe review
metadata. Udon resolves credential values, brokers challenges, owns the live
session, and requires separate runtime approval.

The product keeps browser registration authoring offline and no-submit rather
than exposing a registration capture or runtime flow. Place an
already-reviewed `uws.browser-registration.1.0` profile
and its conventional `*.review.json` Browsertools bundle under
`browser-registration/`, author an explicit `browser_registration` intent,
and provide matching `.icot/browser-registration.json` review evidence.
The shared iCoT transaction engine can adopt an explicitly reviewed no-submit
Browsertools candidate into the same session-free intent and package artifacts.
Its UI and terminal review surfaces show only the public transaction,
candidate/output identity, cleanup disposition, producer disclosure, and
package/recovery digests; they never accept a private result path or grant
submit/account/session/runtime authority.
OpenUdon may build, assess, generate approval, and dry-run that package. Non-dry
execution additionally requires the exact Udon report-v3/Browserdriver-v4
contract, an owner-only digest-bound dedicated-test attestation outside the
repository, and a separate exact submit approval.

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
- Runtime Policy: allowed runtimes such as `openapi`, `http`, `browser`, `fnct`, `cmd`, or `ssh`.
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
execute production workflows. Review evidence treats generated artifacts as review state
`generated`; side-effectful proof runs require `approved_for_sandbox`, and production execution
requires `approved_for_production`.

## Runtime Selection Rules

OpenUdon should use API source documents for API operations when a matching document and operation are
available. The source should provide method, path, schemas, server, and security metadata.

OpenUdon should use the `browser` runtime only as an explicit fallback when no adequate reviewed API
operation covers the active capability. A browser step must name an action declared by a packaged,
active Browsertools profile, map all required parameters, and expose its session and side-effect
approval posture. Udon resolves the operator-owned session binding and executes the action.

OpenUdon should use non-OpenAPI runtimes only when the project explicitly allows them:

- `fnct`: trusted local functions, transforms, renderers, adapters, or private glue.
- `cmd`: approved local commands. Use only with an explicit allow policy.
- `ssh`: approved remote host operations. Use only with an explicit allow policy.
- `http`: API-source-bound HTTP behavior when a reviewed OpenAPI, Google Discovery, or AWS Smithy
  source is available.
- `browser`: a Browsertools-profile-bound UI action used only after the API-first capability check.

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

## API Source Policy

If the project needs API or event-source-bound calls, provide one of these:

- OpenAPI files under `openapi/`.
- Google Discovery files under `google-discovery/` or legacy `discovery/`.
- AWS Smithy JSON files under `aws-smithy/`.
- AsyncAPI files under `asyncapi/` when the workflow binds to event or message source operations.
- GraphQL, OpenRPC, gRPC/protobuf, and OData files under `graphql/`, `openrpc/`,
  `grpc-protobuf/`, and `odata/` when the workflow binds to UWS 1.4 source operations.
- OpenAPI document URLs in `project.md`.
- Search/discovery hints precise enough for OpenUdon to find the relevant API document.

If the project does not need API or event-source-bound calls, write this legacy-compatible exact
policy:

```md
OpenAPI: none required
```

When that phrase is present, OpenUdon should not fail only because API source directories are empty.
It should also reject generated artifacts that still reference API or event source operations.

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
