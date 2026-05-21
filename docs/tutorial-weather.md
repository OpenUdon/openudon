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
official machine-readable source is Google Discovery and OpenWeatherMap is documented through
official human docs, so a local OpenAPI slice or user-provided OpenAPI file is still needed for
OpenUdon synthesis. iCoT reports those first-class sources and can migrate cached first-class API
documents from `../apitools` into the current example when they exist, but it does not treat
committed eval fixture slices as available inputs for a new example.

## Start With iCoT

Create a new local example and let iCoT inspect the first-class `apitools` catalog metadata from the
brief:

```bash
go run ./cmd/icot --example ./examples/weather-toronto-gmail
```

For a brief such as "get weather in Toronto, and Gmail the report to me", iCoT reports matching
provider metadata from the sibling `../apitools/catalog-openapi-cache` when it is present. It first
checks whether every matched provider has a local or migratable API document. If cached first-class
documents are available, iCoT asks whether to migrate them into the workflow. If local API documents
already exist, it asks whether to use those for operation selection. After that document step, it
lists operation IDs grouped by API document, using summaries and descriptions to make the choices
reviewable.

For this specific weather-to-Gmail workflow, current catalog metadata can migrate Gmail's official
Google Discovery document when it is cached, but OpenWeatherMap still needs a local OpenAPI slice or
lowering output because the catalog only records official docs and advisory overlay metadata for it.
The committed weather eval fixture remains an example, not an implicit input to the new workflow.

Use answers like these:

```text
Project name: Weather Toronto Gmail Report
Goal: Resolve Toronto, Canada to coordinates, fetch current weather, render a concise report, and send it by Gmail.
Inputs: `recipient_email`: required string for the report recipient.
Outputs: `weather_report`: rendered report body; `gmail_result`: Gmail send response.
Data flow: Resolve Toronto to coordinates; pass latitude and longitude to the weather step; render a report from the weather response; send the report to `inputs.recipient_email` with Gmail.
Function contracts: `render_weather_report`: inputs weather response; outputs subject and body; side effects none.
Does this project need API/OpenAPI integration? yes
OpenAPI files, URLs, or service hints: add or lower `openapi/weather.yaml`; migrate cached Gmail Discovery if prompted, then lower or provide Gmail OpenAPI before synthesis.
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
the selected API documents, operation IDs, symbolic credential names, and the Gmail approval
boundary. If iCoT reports only Discovery, Smithy, docs, or advisory overlays for a provider, first
lower or provide a local OpenAPI file under `openapi/`; direct committed eval slices are not assumed.

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
