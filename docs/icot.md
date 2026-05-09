# iCoT

iCoT is OpenUdon's guided authoring CLI. It helps an operator turn a workflow idea into
`project.md` and `workflows/intent.hcl`.

```bash
go run ./cmd/icot --example ./examples/<name>
```

The command creates the example directories when needed and writes the standard OpenUdon authoring
sections. It does not synthesize compiled artifacts and it does not execute workflows.

## Common Modes

```bash
# Print rendered project.md and intent.hcl without writing files.
go run ./cmd/icot --example ./examples/<name> --print

# Use the fixed manual flow without optional LLM extraction.
go run ./cmd/icot --example ./examples/<name> --no-llm

# Seed from an existing fixture.
go run ./cmd/icot --from-example ./examples/eval/weather-toronto --example ./examples/<name>

# Use YAML or JSON answers.
go run ./cmd/icot --answers ./answers.yaml --example ./examples/<name>

# Rebuild project.md from workflows/intent.hcl.
go run ./cmd/icot reconcile --example ./examples/<name>

# Check brief quality, intent parseability, and drift.
go run ./cmd/icot lint --example ./examples/<name>
```

## Provider Defaults

Shell-level provider selection uses OpenUdon names:

```bash
export OPENUDON_LLM_PROVIDER=copilot-api
export OPENUDON_LLM_MODEL=gpt-5.4-mini
```

Provider API keys stay in provider-native environment variables such as `COPILOT_API_KEY`,
`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or `GEMINI_API_KEY`. Do not paste credentials into prompts,
examples, generated artifacts, or approval files.

## Output

iCoT saves the source artifacts:

```text
project.md
workflows/intent.hcl
```

Then use `openudon build` or `openudon synthesize` to produce generated UWS, plan, quality, review,
and handoff artifacts.
