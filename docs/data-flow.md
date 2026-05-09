# Ramen Data Flow

Ramen workflows pass data through explicit step outputs and request inputs. A user can describe one
business action, but generated artifacts must expose the technical steps and field mappings needed
to execute it.

`docs/intent.md` is the internal `intent.hcl` contract. This page focuses on data-flow examples and
quality expectations.

## How Data Moves

- OpenAPI steps receive request fields from literals, workflow inputs, credential bindings, or prior
  step outputs.
- During UWS generation, unqualified OpenAPI request fields are placed using the selected
  operation's public metadata. Ramen preserves explicit `path.`, `query.`, `header.`, `cookie.`, and
  `body.` prefixes, and rejects unknown unqualified fields instead of guessing a query parameter.
- Prior step outputs are referenced as `step_name.received_body...` in workflow HCL.
- `intent.hcl` should use `bind` blocks when one step feeds another.
- `fnct` steps are trusted adapters or transformations. Their input and output contract should be
  documented in `project.md`.

## Explicit Bindings

Use explicit bindings when a later step needs fields from an earlier response:

```hcl
step "get_weather" {
  type      = "http"
  do        = "Fetch current weather from coordinates"
  operation = "getWeatherData"

  bind {
    from = "get_coordinates"
    fields = {
      "lat" = "body[0].lat"
      "lon" = "body[0].lon"
    }
  }
}
```

This means:

- `get_weather.lat` comes from `get_coordinates.received_body[0].lat`
- `get_weather.lon` comes from `get_coordinates.received_body[0].lon`
- `get_weather` depends on `get_coordinates`

## Hidden Technical Steps

Users do not need to know every API endpoint. For example:

```md
Search weather in Toronto, Canada.
```

If the available OpenAPI documents expose both geocoding and weather operations, Ramen should
expand that into:

1. `get_coordinates`: call geocoding with city and country.
2. `get_weather`: call weather with `lat` and `lon` from `get_coordinates`.

Those hidden technical steps must appear in `intent.hcl`, `workflow.hcl`, review evidence, and
quality reports. They should not remain implicit.

Ramen also writes `expected/plan.json` and `expected/plan.md`. The plan records each inferred
technical step, the chosen runtime or OpenAPI operation, required parameters, dependencies, and
bindings. During assessment, Ramen parses the final public UWS `workflow.hcl` and verifies the
workflow still preserves those mappings.

Quality assessment also validates intent provenance before execution. `depends_on`, `with`, `bind`,
conditions, loop selectors, and outputs must reference declared inputs or known step names. A
reference such as `missing_step.received_body.id` fails `intent.data_flow.sources`.

When OpenAPI response schemas expose concrete object properties, response paths must match those
properties. A path such as `get_ticket.received_body.requesterEmail` fails
`intent.data_flow.response_paths` if the selected operation only documents `id` and `severity`.
Opaque or missing response schemas produce a warning instead of a failure.

## Structural Results

When an intent output references a structural `switch`, `merge`, or `loop` step, Ramen exports a
matching UWS `results[]` entry. The expected plan records the result name, kind, source, and value,
and quality assessment fails if `workflow.uws.yaml` drops or changes that structural result.

## Missing Operations

If the selected OpenAPI document only has a weather endpoint that requires `lat` and `lon`, Ramen
must not invent coordinates for a city. It should try approved OpenAPI discovery/import. If no
geocoding operation is available, it should stop with a clear missing-capability report.

## Function Contracts

For `fnct` steps, document the function contract in `project.md`:

```md
## Function Contracts

- `normalize_ticket`
  - Inputs: `get_ticket.received_body`
  - Outputs: normalized ticket fields: `requester_email`, `subject`, `summary`
  - Side effects: none
```

Ramen can then wire the function output into later API or runtime steps.

Quality checks require every generated `fnct` step to have a matching Function Contracts entry.
When the contract declares inputs, the intent must show visible input evidence through `with`, a
`bind` block, or a prior-step reference. If a project expects no function steps, say so explicitly;
any generated `fnct` step will fail review.
