# Intent HCL

`workflows/intent.hcl` is Ramen's internal structured authoring contract between
`project.md` and `workflows/workflow.hcl`. It records what Ramen intends to build before udon
lowers that intent into executable workflow source and exported UWS.

This is a Ramen-owned document. It is not the public UWS specification and it is not a generic
udon runtime specification. Public workflow semantics remain in `../uws`; generic compiler and
runtime behavior remains in `../udon`.

## Purpose

`intent.hcl` exists to make workflow generation auditable and repeatable:

```text
project.md -> workflows/intent.hcl -> workflows/workflow.hcl -> workflows/workflow.uws.yaml
```

Ramen accepts intent through the `github.com/genelet/udon/pkg/rollout.Intent` model and the
structured JSON schema embedded at `internal/synthesize/schemas/intent.schema.json`. `ramen build`
parses an existing `workflows/intent.hcl`; `ramen synthesize` generates one from `project.md`,
OpenAPI discovery, and project policy.

Intent HCL parsing is delegated to udon's `rollout.ParseIntent` API. The canonical renderer is
`runner.RenderIntentHCL`; generated workflow lowering uses `Intent.NormalizedForGeneration` before
udon produces `workflow.hcl`. Ramen does not import udon's private HCL parser packages directly.

## Accuracy Profile

Ramen accepts some incomplete or descriptive intent so the refinement loop can repair generated
artifacts. That looser mode is not the accuracy target.

For near-100% conversion from `intent.hcl` to `workflow.hcl`, author the high-fidelity profile:

- Give every step an explicit `type`.
- For API steps, set `operation` and either top-level `openapi` or step-local `openapi`.
- Put every required request field in `with` or `bind`.
- Use `depends_on` and `bind` for every step-to-step data dependency.
- Use credential binding names only; never put credential values in intent.
- Treat `do`, `using`, and `set` as review or repair hints, not as the source of required wiring.

An intent that follows this profile should lower to semantically equivalent `workflow.hcl`. Missing
OpenAPI files, unknown operations, unresolved references, disallowed runtimes, absent credential
bindings, or unverifiable response paths should fail validation instead of being guessed.

## File Shape

An intent file is HCL with these top-level fields and blocks:

```text
openapi    = "openapi/example.yaml"   # optional default OpenAPI document
server_url = "https://sandbox.example.com" # optional default server URL
locals     = { region = "us-east-1" } # optional string map

workflow { ... }                      # required by Ramen-generated intent
input "<name>" { ... }                 # zero or more
trigger "<name>" { ... }               # zero or more
security "<name>" { ... }              # zero or more
step "<name>" { ... }                  # one or more for normal generation
output "<name>" { ... }                # zero or more
```

`rollout.ParseIntent` requires at least one `step` or `trigger`. Ramen-generated intent should
always include a `workflow` block with non-empty `name` and `description`, and at least one `step`.
Block order is not significant to parsing. The renderer emits a stable order, and hand-authored
intent should follow it for reviewability: top-level attributes, `workflow`, inputs, triggers,
security, steps, then outputs.

Top-level `openapi` is the default OpenAPI document path for API steps that do not declare
step-local `openapi`. `Intent.MissingSlots()` reports `"OpenAPI specification URL or content"` when
a step names an `operation` but neither step-local nor top-level `openapi` is available.

Top-level `server_url` is an optional server override for the selected OpenAPI documents, typically
used to steer proof runs toward sandbox endpoints. Top-level `locals` is an optional string map for
workflow constants referenced as `locals.<name>`.

## Workflow

The `workflow` block names and describes the workflow.

```hcl
workflow {
  name        = "runtime_only_render"
  description = "Render a local summary report."
}

input "summary" {
  type     = "string"
  required = true
}

step "render_report" {
  type = "fnct"
  do   = "Render the summary report."
  with = {
    summary = "inputs.summary"
  }
}

output "report" {
  from = "render_report.received_body"
}
```

`workflow.name` becomes the workflow identity in generated artifacts. Use stable `snake_case`.
`workflow.description` is human review context and should summarize the workflow goal.

## Inputs

`input` blocks declare values supplied by the caller at runtime. They are referenced as
`inputs.<name>`.

Supported attributes:

| Attribute | Type | Meaning |
| --- | --- | --- |
| label | string | Input name. |
| `type` | string | Optional type hint such as `string`, `integer`, `boolean`, or `object`. |
| `description` | string | Human-readable input purpose. |
| `required` | boolean | Whether callers must provide the value. |
| `sensitive` | boolean | Marks the value as sensitive review metadata. Prefer credential bindings for secrets. |
| `default` | string | Optional literal default. |

High-fidelity API steps should reference runtime inputs explicitly in `with`:

```text
with = {
  customerId = "inputs.customerId"
}
```

## Steps

`step` blocks are the core units that lower into `workflow.hcl`.

Supported step attributes:

| Attribute | Meaning |
| --- | --- |
| label | Stable step name used by dependencies, binds, outputs, and review evidence. |
| `type` | Runtime or structural type. |
| `do` | Human action description for review and generation. |
| `using`, `set` | Legacy/descriptive hints. Prefer explicit `with` and `bind`. |
| `depends_on` | Earlier step names that must complete first. |
| `with` | Request fields, function inputs, literals, credential bindings, or references. |
| `openapi` | Step-local OpenAPI document path. |
| `operation` | OpenAPI `operationId`. |
| `provider` | Optional provider label for multi-provider workflows. |
| `bind` | Structured source-step-to-target-field wiring. |
| `when`, `for_each`, `items`, `mode`, `batch_size` | Structural control-flow hints. |
| `successCriteria`, `onFailure`, `onSuccess` | Explicit UWS operation actions for leaf steps. |
| nested `step`, `case`, `default` | Structural child steps. |

Supported leaf step types are `http`, `openapi`, `fnct`, `cmd`, and `ssh`.
Supported structural step types are `sequence`, `parallel`, `switch`, `merge`, `loop`, and `await`.

Ramen policy allows `openapi`, `http`, and `fnct` by default. `cmd` and `ssh` require explicit
approval in `project.md`; otherwise quality assessment fails the intent.

Unsupported runtime names such as `sql`, `smtp`, `llm`, and profile-specific `x-udon-*` names must
not appear in Ramen intent unless a later Ramen change adds explicit schema, policy, and fixture
coverage.

## API Steps

API steps should be explicit enough that lowering does not need to infer the operation:

```hcl
openapi = "openapi/weather.yaml"

workflow {
  name        = "weather_toronto"
  description = "Resolve Toronto coordinates and fetch current weather."
}

step "get_coordinates" {
  type      = "http"
  do        = "Resolve Toronto, Canada to coordinates."
  operation = "direct_get"
  with = {
    q = "Toronto,CA"
  }
}

step "get_weather" {
  type       = "http"
  do         = "Fetch weather for the resolved coordinates."
  operation  = "getWeatherData"
  depends_on = ["get_coordinates"]
  with = {
    appid = "weather_appid"
  }
  bind {
    from = "get_coordinates"
    fields = {
      lat = "received_body[0].lat"
      lon = "received_body[0].lon"
    }
  }
}

output "weather" {
  from = "get_weather.received_body"
}
```

Lowering requirements:

- Top-level `openapi` is the default OpenAPI document for API steps.
- Step-local `openapi` overrides the default for that step.
- `operation` must match an operationId in the selected OpenAPI document.
- `with` keys become request fields or function inputs.
- Literal strings stay literal unless they are recognized references such as `inputs.name`,
  `step_name.received_body.path`, or other generated HCL expressions.
- Credential binding names in `with` remain names only; secret values must not be emitted.

For multi-API workflows, prefer step-local `openapi` on every API step:

```hcl
workflow {
  name        = "order_lookup"
  description = "Fetch a customer and inventory record from separate APIs."
}

input "customerId" {
  type     = "string"
  required = true
}

input "sku" {
  type     = "string"
  required = true
}

step "get_customer" {
  type      = "http"
  do        = "Fetch the customer profile."
  openapi   = "openapi/customers.yaml"
  operation = "getCustomer"
  with = {
    customerId    = "inputs.customerId"
    Authorization = "customers_bearer_token"
  }
}

step "check_inventory" {
  type      = "http"
  do        = "Check inventory availability."
  openapi   = "openapi/inventory.yaml"
  operation = "getInventory"
  with = {
    sku               = "inputs.sku"
    inventory_api_key = "inventory_api_key"
  }
}

output "customer" {
  from = "get_customer.received_body"
}

output "availability" {
  from = "check_inventory.received_body"
}
```

## Data Flow

Use `depends_on` to express ordering and `bind` to express field-level data flow. A `bind` block is
authoritative:

```text
bind {
  from = "source_step"
  fields = {
    targetField = "received_body.path"
  }
}
```

This means `targetField` on the current step comes from
`source_step.received_body.path`, and the current step depends on `source_step`. Ramen quality gates
validate that referenced source steps exist and, when response schemas are available, that response
paths are plausible.

Use `with` for runtime inputs, literals, and credential bindings:

```text
with = {
  page          = "1"
  ticketId      = "inputs.ticketId"
  Authorization = "support_bearer_token"
}
```

Do not rely on prose such as "use the prior step's ID" when an exact `bind` can be written.

## Function Steps

`fnct` steps represent approved local adapters, renderers, classifiers, or transformations. They are
not OpenAPI calls and must not carry `operation`, `openapi`, HTTP method, or path semantics.

For high-fidelity lowering:

- Declare the function contract in `project.md`.
- Use `with` and `bind` to show every input.
- Use `output` references to expose the result needed by later steps or reviewers.

If a generated draft omits function implementation detail, udon may provide a deterministic
function default during lowering, but authors should not depend on that default as business logic.

## Actions And Control Flow

Leaf steps may carry explicit `successCriteria`, `onFailure`, and `onSuccess` blocks. Ramen should
preserve them only when the project or intent explicitly asks for success checks, retry, failure
routing, or success routing. Retry is never a default behavior, and side-effectful retry requires
project-level retry or idempotency policy.

Structural steps may use nested `step`, `case`, and `default` blocks. Structural output references
may be exported as UWS structural `results[]` entries when they reference `switch`, `merge`, or
`loop` steps.

`onFailure` retry actions must include `retryLimit`. `retryAfter` is optional. `goto` actions may
name `workflowId` or `stepId`; criteria use `condition`, optional `type`, and optional `context`.

## Credentials And Security

Intent may name credential bindings, but must never contain credential values.

Allowed places for binding names:

- Request fields in `with`, such as `Authorization = "orders_bearer_token"`.
- `security "<name>" { token_from = "binding_name" }`.
- Project-documented credential fields inferred into generated request bindings.

Ramen quality gates compare credential-like OpenAPI parameters and security schemes against
declared project credential policy. Missing bindings fail validation.

Security blocks use this shape:

```text
security "<scheme_name>" {
  description = "Human review text."
  token_from  = "credential_binding_name"
}
```

`token_from` is a binding name only. When it maps to an OpenAPI security scheme, quality checks
verify that the project declares an appropriate credential binding.

Trigger blocks use this shape:

```text
trigger "<name>" {
  path           = "/hooks/example"
  authentication = "scheme_name"
  methods        = ["POST"]

  route "<output>" {
    to = ["step_name"]
  }
}
```

Triggers are accepted by the intent model, but most Ramen examples are synchronously invoked
workflows without triggers.

## Reference Grammar

Values inside `with`, `bind.fields`, `output.from`, and related string fields are parsed as strings
first and interpreted during lowering. Common forms are:

| Form | Meaning |
| --- | --- |
| `inputs.<name>` | Runtime input. |
| `inputs.<name>.<path>` | Nested field of an object input. |
| `<step>.received_body` | Full response body of a prior step. |
| `<step>.received_body.<path>` | Field of a prior response body. |
| `<step>.received_body[<n>]` | Array element of a prior response body. |
| `<step>.received_headers.<header>` | Response header. |
| `<step>.received_raw` | Raw response text. |
| `<step>.status_code` | HTTP status code. |
| `locals.<name>` | Workflow local value. |
| `${...}` | Full HCL expression escape hatch; use sparingly. |
| bare credential binding name | Runtime credential binding, never a literal secret. |
| any other string | Literal request or function input value. |

Inside `bind.fields`, source paths may use short forms. With `from = "get_coordinates"`,
`received_body[0].lat`, `[0].lat`, and `lat` lower to references rooted at
`get_coordinates.received_body`. Prefer the explicit `received_body...` form for clarity.

`bind.fields` target names may include request-location prefixes: `query.`, `path.`, `header.`,
`cookie.`, `body.`, or `payload.`. Bare target names let the OpenAPI resolver choose the request
location from metadata. If both `bind` and `with` provide the same target, the explicit `with` value
wins.

The parser also accepts label-bind syntax:

```text
bind "get_coordinates" {
  fields = {
    lat = "received_body[0].lat"
  }
}
```

The canonical renderer emits the attribute form:

```text
bind {
  from = "get_coordinates"
  fields = {
    lat = "received_body[0].lat"
  }
}
```

## Lowering Contract

Lowering from `intent.hcl` to `workflow.hcl` must preserve these semantics:

| Intent construct | `workflow.hcl` expectation |
| --- | --- |
| `workflow.name` and `workflow.description` | Workflow identity and review metadata. |
| Top-level or step-local `openapi` | Provider/OpenAPI binding using only listed documents. |
| API `operation` | Selected OpenAPI operationId. |
| Step `type` | Canonical workflow step kind, preserving `fnct`, `cmd`, `ssh`, and structural kinds. |
| `depends_on` | Step dependency ordering. |
| `with` | Request fields, function inputs, literals, inputs, and credential binding names. |
| `bind` | Source-step dependencies and target request or function input fields. |
| `input` | Runtime input references available as `inputs.<name>`. |
| `output` | Workflow outputs or UWS structural results. |
| `successCriteria`, `onFailure`, `onSuccess` | Leaf operation actions when explicitly authored. |
| `trigger` and `security` | Trigger and security metadata when supported by the lowering path. |

The lowering path must not invent OpenAPI filenames, operation IDs, runtime types, credential
values, production execution authority, or hidden side-effect policy. If required information is not
present in the intent, project policy, or discovered OpenAPI metadata, validation should report the
gap.

For a fully formed intent, `bind` blocks are normalized before generation: each `bind.from` adds a
deduplicated dependency, each target field is normalized, and each source path is rewritten into a
canonical prior-step reference. API request location is then resolved from OpenAPI metadata where
possible.

## Validation

Current validation is split across parser checks, structured generation schema, and Ramen quality
gates:

- `rollout.ParseIntent` requires valid HCL, at least one `step` or `trigger`, labels for labeled
  blocks, `do` on leaf steps, and non-empty `from` plus fields on leaf-step `bind` blocks.
- `Intent.MissingSlots()` reports missing default OpenAPI context, missing steps, and missing leaf
  descriptions.
- `internal/synthesize/schemas/intent.schema.json` rejects unknown generated JSON fields, restricts
  generated step types to the supported enum, and requires `operation` when generated
  `type = "openapi"`.
- Ramen quality gates validate runtime policy, OpenAPI references and operations, required
  parameters, credential bindings, function contracts, data-flow references, response paths, and
  side-effect policy.

## Authoring Style

- Use stable `snake_case` workflow and step names.
- Keep `do` text concise, present-tense, and review-oriented.
- Set explicit `type` on every step.
- Prefer explicit `depends_on` even when `bind` would imply it.
- Prefer `bind` for cross-step data flow; reserve `with` for literals, runtime inputs, and
  credential binding names.
- Keep credential binding names symbolic and declare them in `project.md`.
- Do not add unknown top-level attributes such as `version`; `intent.hcl` currently has no explicit
  version field.

## More Patterns

API to function adapter:

```hcl
openapi = "openapi/support.yaml"

workflow {
  name        = "support_priority_routing"
  description = "Fetch a support ticket, classify priority, and prepare one internal routing result."
}

input "ticketId" {
  type     = "string"
  required = true
}

step "get_ticket" {
  type      = "http"
  do        = "Fetch support ticket details."
  operation = "getTicket"
  with = {
    ticketId = "inputs.ticketId"
  }
}

step "classify_priority" {
  type       = "fnct"
  do         = "Classify the support ticket priority."
  depends_on = ["get_ticket"]
  bind {
    from = "get_ticket"
    fields = {
      ticket = "received_body"
    }
  }
}

output "routing_result" {
  from = "classify_priority.received_body"
}
```

Approved command runtime:

```hcl
workflow {
  name        = "cmd_allowed_deploy"
  description = "Run the approved deployment status command."
}

step "check_deploy_status" {
  type = "cmd"
  do   = "Run the sandbox deployment status command."
}

output "status" {
  from = "check_deploy_status.received_body"
}
```

`cmd` and `ssh` are valid intent runtimes only when `project.md` explicitly approves them.

Switch routing:

```hcl
workflow {
  name        = "priority_switch"
  description = "Route a classified ticket by priority."
}

input "ticketId" {
  type     = "string"
  required = true
}

step "route_by_priority" {
  type = "switch"

  case "high" {
    when = "inputs.priority == 'high'"

    step "page_oncall" {
      type = "fnct"
      do   = "Prepare an on-call paging request."
      with = {
        ticketId = "inputs.ticketId"
      }
    }
  }

  default {
    step "queue_for_triage" {
      type = "fnct"
      do   = "Queue the ticket for normal triage."
      with = {
        ticketId = "inputs.ticketId"
      }
    }
  }
}

output "route" {
  from = "route_by_priority"
}
```

## Non-Goals

`intent.hcl` does not replace `project.md`; project policy remains the source of runtime approval,
credential binding policy, safety boundary, and fallback behavior.

`intent.hcl` does not replace `workflow.hcl` or UWS. The generated `workflow.hcl` and exported UWS
remain the artifacts compiled and reviewed before trusted execution.

`intent.hcl` does not authorize execution. Production side effects require the approved trusted-runner
path documented in Ramen safety and handoff artifacts.

## Reference Implementation

- Go model and parser: `github.com/genelet/udon/pkg/rollout.Intent` and `rollout.ParseIntent`.
- Canonical renderer: `github.com/genelet/udon/pkg/runner.RenderIntentHCL`.
- Generation normalizer: `Intent.NormalizedForGeneration`.
- Structured generation schema: `internal/synthesize/schemas/intent.schema.json`.
- Ramen validation: `ramen assess` quality checks under `internal/synthesize`.
- Reference fixtures: `examples/eval/*/reference/intent.hcl`.
