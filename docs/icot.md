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

# Ask every question and let you confirm defaults. This is the default mode.
go run ./cmd/icot --example ./examples/<name> --prompt-mode full

# Print defaulted questions and accept their defaults automatically.
go run ./cmd/icot --example ./examples/<name> --prompt-mode normal

# Ask only when iCoT has no default or answer.
go run ./cmd/icot --example ./examples/<name> --prompt-mode fast

# Seed from an existing fixture.
go run ./cmd/icot --from-example ./examples/eval/weather-toronto --example ./examples/<name>

# Use YAML or JSON answers.
go run ./cmd/icot --answers ./answers.yaml --example ./examples/<name>

# Rebuild project.md from workflows/intent.hcl.
go run ./cmd/icot reconcile --example ./examples/<name>

# Check brief quality, intent parseability, and drift.
go run ./cmd/icot lint --example ./examples/<name>
```

`--prompt-mode full` is the default when the flag is omitted; it prints every question and waits for
you to confirm or replace defaults. `--prompt-mode normal` prints every question and automatically
accepts defaults. `--prompt-mode fast` skips defaulted questions entirely, suppresses catalog/status
chatter plus review-only fallback and assumption text, and asks only for required values without a
safe default, such as the initial workflow goal. Automatically accepted defaults and assumptions are
still recorded in the transcript.

## Guided SaaS Authoring

For common SaaS workflows, iCoT now keeps the guided loop focused on the
reviewable OpenUdon contract:

- choose a local API source document and listed operation ID instead of
  inventing provider calls;
- inspect first-class provider metadata in sibling `../apitools`, use a bounded
  LLM catalog plan to choose only validated local artifacts when available, and
  retrieve cached OpenAPI, Google Discovery, AWS Smithy, or reviewed advisory
  OpenAPI overlay artifacts into the workflow before asking for operation
  choices;
- confirm existing local API documents before using them for operation
  selection;
- draft required path, query, header, and body field mappings from selected
  operation details, then ask the operator only for mappings that remain
  unresolved;
- name symbolic credential bindings only, never token values;
- choose outputs from known response paths or declared function outputs;
- classify execution posture as `read-only`, `sandbox-only`, or
  `after-approval`.

iCoT lists operation IDs grouped by API document with summaries or descriptions.
If a provider operation, request field, response path, or credential scheme is
not visible in local metadata, leave it unresolved and repair or provide the
API source before trusted handoff.

If a required provider is missing a local API source, iCoT tries the first-class
apitools catalog/cache automatically. It only asks for user-provided API
artifacts after apitools reports that no first-class or advisory
source artifact is available.

When an original provider OpenAPI document and a reviewed advisory OpenAPI
overlay are both available, iCoT defaults to the advisory overlay for operation
selection.

## Draft Pipeline

iCoT is optimized to produce a useful starting `intent.hcl`, not a perfect final
workflow. The guided SaaS path is:

1. Resolve API artifacts from the brief. Immediately after `Workflow goal`, iCoT
   builds a compact catalog shortlist and may ask the LLM to choose relevant
   artifact keys and rough provider/capability steps. Every returned
   provider/artifact tuple is validated against the deterministic shortlist
   before any file is copied.
2. If required OpenAPI, Discovery, or advisory overlay artifacts are missing
   locally, try `../apitools` first and materialize available validated
   artifacts into the workflow. Unknown catalog providers, invented paths, and
   non-migratable artifacts are rejected and recorded in the transcript.
3. For each local API artifact or provider-backed step, ask which listed
   `operationId` to use. iCoT should offer a ranked default; when multiple
   candidates remain plausible, the operator chooses one.
4. Build compact per-operation API context from the selected operation IDs,
   including the single operation, relevant schemas, and security requirements.
5. Send the original goal, selected operation contexts, readiness feedback, and
   `intent.hcl` guardrails to the LLM to draft the structured intent. If
   deterministic readiness later finds missing required request values, iCoT
   gives the LLM one focused mapping pass with the selected operation details
   before asking the operator for field sources.
6. Show the resulting draft, assumptions, and warnings for confirmation. If the
   operator confirms, iCoT writes `project.md` and `workflows/intent.hcl`; the
   operator can continue editing manually before build or review. If the draft
   is wrong, reject or edit it instead of treating iCoT as the final authority.

## Provider Defaults

iCoT defaults to the local `copilot-api` gateway and `gpt-5.4-mini`, matching synthesis. If
`~/.config/systemd/user/copilot-api.service` owns the gateway, keep it running and point OpenUdon at
that local endpoint:

```bash
systemctl --user status copilot-api.service
export COPILOT_API_BASE_URL=http://localhost:4141
export OPENUDON_LLM_PROVIDER=copilot-api
export OPENUDON_LLM_MODEL=gpt-5.4-mini
```

Set `OPENUDON_LLM_PROVIDER=gemini` or pass `--provider gemini` only when you explicitly want Gemini.
Provider-specific API keys no longer make iCoT choose that provider implicitly.

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
