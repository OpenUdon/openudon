# Tutorial: Weather

This read-only fixture resolves Toronto coordinates and fetches current weather. It can also be used
as the starting point for an iCoT-first workflow that fetches weather, renders a report, and sends
that report through Gmail after review.

Fixture path:

```text
examples/eval/weather-toronto/project.md
examples/eval/weather-toronto/openapi/weather.yaml
examples/eval/weather-toronto/reference/intent.hcl
examples/eval/weather-toronto/reference/plan.json
examples/eval/weather-toronto/reference/workflow.hcl
```

The project brief declares a fixed city and country, uses the local weather OpenAPI document, and
requires generated artifacts only. It is useful for checking hidden technical step expansion:
`get_coordinates` feeds latitude and longitude into `get_weather`.

## Check Provider Metadata

Before searching public API catalogs, inspect the first-class provider catalog from `apitools`:

```bash
go run ./cmd/openudon catalog inspect openweathermap
go run ./cmd/openudon catalog inspect gmail
```

The catalog records provider source metadata and security overlays. In the current catalog, Gmail's
official machine-readable source is Google Discovery and OpenWeatherMap may have a reviewed advisory
OpenAPI overlay in the sibling cache. iCoT reports those first-class sources and can migrate cached
first-class API documents or advisory overlays from `../apitools` into the current example when they
exist, but it does not treat committed eval fixture slices as available inputs for a new example.

## Start With iCoT

Create a new local example and let iCoT inspect the first-class `apitools` catalog metadata from the
brief:

```bash
go run ./cmd/icot --example ./examples/weather-toronto-gmail
```

For a brief such as "get weather in Toronto, and Gmail the report to me", iCoT reports matching
provider metadata from the sibling `../apitools/catalog-openapi-cache` when it is present. Immediately
after the first goal, it may ask the LLM to select relevant catalog artifact keys from a compact
shortlist and propose rough provider-level steps. OpenUdon validates every selected provider and
artifact key before copying anything. Unknown providers, invented paths, and non-migratable artifacts
are rejected and recorded in the local transcript.

After validated artifacts are local, iCoT lists operation IDs grouped by API document, using summaries
and descriptions to make the choices reviewable. Once operations are selected, iCoT gives the LLM a
focused chance to map required request fields from the selected operation metadata. It should fill
obvious sources such as `lat`/`lon` from a geocoding step, safe literals from the brief, runtime
inputs, prior-step outputs, or symbolic credentials. If a mapping is not defensible from local
metadata, iCoT asks the operator instead of inventing it.

For this specific weather-to-Gmail workflow, current catalog metadata can migrate Gmail's official
Google Discovery document when it is cached and can materialize an OpenWeatherMap advisory OpenAPI
overlay when present. Discovery is still authoring metadata; if a final package needs executable
OpenAPI-bound metadata for Gmail, lower or provide a local OpenAPI file before synthesis or trusted
handoff. The committed weather eval fixture remains an example, not an implicit input to the new
workflow.

Use answers like these:

```text
Project name: Weather Toronto Gmail Report
Goal: Resolve Toronto, Canada to coordinates, fetch current weather, render a concise report, and send it by Gmail.
Inputs: `recipient_email`: required string for the report recipient.
Outputs: `weather_report`: rendered report body; `gmail_result`: Gmail send response.
Data flow: Resolve Toronto to coordinates; pass latitude and longitude to the weather step; render a report from the weather response; send the report to `inputs.recipient_email` with Gmail.
Function contracts: `render_weather_report`: inputs weather response; outputs subject and body; side effects none.
Does this project need API/OpenAPI integration? yes
OpenAPI files, URLs, or service hints: let iCoT migrate cached OpenWeatherMap advisory OpenAPI and Gmail Discovery artifacts if prompted; lower or provide Gmail OpenAPI before synthesis if executable Gmail metadata is required.
Approve cmd runtime? no
Approve ssh runtime? no
Side-effect scope (read-only/sandbox-only/after-approval): sandbox-only
Credential binding names only: weather_appid, gmail_oauth_token
Safety and approval notes: Generate and validate artifacts only; Gmail send requires approved sandbox credentials and trusted-runner execution.
Fallback behavior: Stop if coordinates, weather lookup, report rendering, or Gmail send fails.
```

Then lint the authored source artifacts:

```bash
go run ./cmd/icot lint --example ./examples/weather-toronto-gmail
```

iCoT writes `project.md` and `workflows/intent.hcl`. Review those files before synthesis, especially
the selected API documents, operation IDs, request field mappings, symbolic credential names, and the
Gmail approval boundary. If iCoT reports only Discovery, Smithy, docs, or advisory overlays for a
provider and executable OpenAPI-bound metadata is required, first lower or provide a local OpenAPI
file under `openapi/`; direct committed eval slices are not assumed.

## Run The Artifact Loop

```bash
go run ./cmd/openudon synthesize --example ./examples/weather-toronto-gmail
go run ./cmd/openudon build --example ./examples/weather-toronto-gmail
go run ./cmd/openudon assess --example ./examples/weather-toronto-gmail
```

For the committed weather-only fixture, use:

```bash
go run ./cmd/openudon synthesize --example ./examples/eval/weather-toronto
go run ./cmd/openudon build --example ./examples/eval/weather-toronto
go run ./cmd/openudon assess --example ./examples/eval/weather-toronto
```

Inspect:

```text
examples/eval/weather-toronto/expected/plan.md
examples/eval/weather-toronto/expected/quality.md
examples/eval/weather-toronto/expected/review.md
examples/eval/weather-toronto/expected/symphony-handoff.json
```

## Approval Dry Run

Weather lookup is documented as generated-artifacts-only in the fixture. If you still want to test
the handoff gates, generate sandbox approval and use a dry run:

```bash
mkdir -p approvals
go run ./cmd/openudon approval-template \
  --example ./examples/eval/weather-toronto \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/weather-toronto-sandbox.json

go run ./cmd/openudon run \
  --example ./examples/eval/weather-toronto \
  --tier sandbox \
  --approval approvals/weather-toronto-sandbox.json \
  --dry-run
```

`--dry-run` validates the package, approval, digest, quality, and tier compatibility without
invoking the trusted executor.
