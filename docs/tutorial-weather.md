# Tutorial: Weather

This read-only fixture resolves Toronto coordinates and fetches current weather. It can also be used
as the starting point for a Kinet-authored workflow that fetches weather, renders a report, and sends
that report through Gmail after review.

## Author the workflow

Use Kinet's workflow route for the interactive weather/report/Gmail request.
It calls OpenUdon's native step commands and requires exact proposal and workflow
approval. A message delivery is a write, never authorized by an ambiguous brief.
This tutorial's retained fixture can be copied deterministically without a model:

```bash
go run ./cmd/openudon authoring draft \
  --from-example ./examples/eval/runtime-only-render \
  --example .openudon-run/tutorial-neutral --prompt-mode fast --no-llm --yes
```

That local function-only example performs no weather request or email. For this
weather fixture, build/assessment below inspect reviewed artifacts; real execution
requires the separately approved native package and declared runtime credentials.


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

Real weather-to-Gmail execution is manual evidence only. Credentials and credential environment are
operator-owned, `expected/data.hcl` is reviewed package input rather than a secret store, and running
without `--dry-run` can send email or call live provider APIs. Do not include real provider
execution in the current public release gates.
