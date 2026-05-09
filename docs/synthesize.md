# Synthesize And Build

The synthesis path starts from a reviewed project brief and produces the package artifacts that
reviewers inspect before any trusted-runner handoff.

## Generate A Package

```bash
export OPENUDON_LLM_PROVIDER=copilot-api
export OPENUDON_LLM_MODEL=gpt-5.4-mini

go run ./cmd/openudon synthesize \
  --example ./examples/<name> \
  --provider "$OPENUDON_LLM_PROVIDER" \
  --model "$OPENUDON_LLM_MODEL" \
  --max-attempts 5
```

`synthesize` reads `project.md`, OpenAPI files under `openapi/`, and optional existing intent. It
writes:

```text
workflows/intent.hcl
workflows/workflow.hcl
workflows/workflow.uws.yaml
expected/plan.json
expected/plan.md
expected/discovery.json
expected/refinement.json
expected/refinement.md
expected/review.md
expected/symphony-handoff.json
expected/quality.json
expected/quality.md
```

## Narrower Stages

Use `build` after editing `workflows/intent.hcl`:

```bash
go run ./cmd/openudon build --example ./examples/<name> --max-attempts 5
```

Use `promote` after editing `workflows/workflow.hcl`:

```bash
go run ./cmd/openudon promote --example ./examples/<name>
```

Use `assess` to rerun deterministic quality gates only:

```bash
go run ./cmd/openudon assess --example ./examples/<name>
```

## Review Next

Generated artifacts are evidence, not approval. Reviewers should inspect `expected/quality.md`,
`expected/review.md`, `expected/plan.md`, and `expected/symphony-handoff.json`, then generate
approval JSON only when the package is acceptable for the intended tier.
