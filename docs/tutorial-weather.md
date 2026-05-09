# Tutorial: Weather

This read-only fixture resolves Toronto coordinates and fetches current weather.

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

## Run The Artifact Loop

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
