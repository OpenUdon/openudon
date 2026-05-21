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

## Guided SaaS Authoring

For common SaaS workflows, iCoT now keeps the guided loop focused on the
reviewable OpenUdon contract:

- choose a local OpenAPI document and listed `operationId` instead of inventing
  provider calls;
- inspect first-class provider metadata in sibling `../apitools` and retrieve
  available cached API documents or reviewed advisory OpenAPI overlays into the
  workflow before asking for operation choices;
- confirm existing local API documents before using them for operation
  selection;
- map required path, query, header, and body fields to `inputs.<name>`, safe
  literals, prior-step outputs, or `credentials.<binding>`;
- name symbolic credential bindings only, never token values;
- choose outputs from known response paths or declared function outputs;
- classify execution posture as `read-only`, `sandbox-only`, or
  `after-approval`.

iCoT lists operation IDs grouped by API document with summaries or descriptions.
If a provider operation, request field, response path, or credential scheme is
not visible in local metadata, leave it unresolved and repair or lower the
OpenAPI slice before trusted handoff.

If a required provider is missing local OpenAPI, iCoT tries the first-class
apitools catalog/cache automatically. It only asks for user-provided API
artifacts after apitools reports that no first-class or advisory
OpenAPI/lowering-ready artifact is available.

When an original provider OpenAPI document and a reviewed advisory OpenAPI
overlay are both available, iCoT defaults to the advisory overlay for operation
selection.

## Draft Pipeline

iCoT is optimized to produce a useful starting `intent.hcl`, not a perfect final
workflow. The guided SaaS path is:

1. Resolve API artifacts from the brief. If required OpenAPI, Discovery, or
   advisory overlay artifacts are missing locally, try `../apitools` first and
   materialize available artifacts into the workflow.
2. For each local API artifact or provider-backed step, ask which listed
   `operationId` to use. iCoT should offer a ranked default; when multiple
   candidates remain plausible, the operator chooses one.
3. Build compact per-operation API context from the selected operation IDs,
   including the single operation, relevant schemas, and security requirements.
4. Send the original goal, selected operation contexts, and `intent.hcl`
   guardrails to the LLM to draft the structured intent.
5. Show the resulting draft, assumptions, and warnings for confirmation. If the
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
