# Tutorial: Weather

This read-only fixture resolves Toronto coordinates and fetches current weather. It can also be used
as the starting point for an iCoT-first workflow that fetches weather, renders a report, and sends
that report through Gmail after review.

## Start With iCoT

Create a new local example and let iCoT inspect the first-class `apitools` catalog metadata from the
brief. Fast prompt mode skips defaulted questions and asks only when no safe answer is available:

```bash
go run ./cmd/icot --example ./examples/weather-toronto-gmail --prompt-mode fast
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

Before final confirmation, LLM-assisted iCoT runs a single advisory flow review. For this workflow,
that review is meant to catch cross-step mistakes such as a Gmail send step that does not consume the
weather report body or an output that returns only the Gmail API response when the goal asks for the
report content. Advisory `llm_flow_review_*` findings are also written as comments in the generated
`workflows/intent.hcl` so reviewers see the issue near the relevant step or output.

For this specific weather-to-Gmail workflow, current catalog metadata can migrate Gmail's official
Google Discovery document when it is cached and can materialize an OpenWeatherMap advisory OpenAPI
overlay when present. Gmail can remain a first-class `google-discovery` source in the final UWS 1.2
package when the trusted executor supports typed sources. The committed weather eval fixture remains
an example, not an implicit input to the new workflow.

Use answers like these when a prompt has no accepted default:

```text
Workflow goal: Resolve Toronto, Canada to coordinates, fetch current weather, render a concise report, and send it by Gmail.
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
the selected API documents, operation IDs, request field mappings, symbolic credential names, Gmail
approval boundary, and any generated flow-review comments. If iCoT reports only Stone, human-docs, or
other non-first-class metadata for a provider, first lower or provide a reviewed local source file;
direct committed eval slices are not assumed.

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
examples/eval/weather-toronto/expected/review-handoff.json
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
