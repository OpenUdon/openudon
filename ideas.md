# Implementation tracker: improve project.md → workflow.hcl accuracy & reliability

Status markers:

- `[todo]` not started
- `[in-progress]` actively being implemented
- `[blocked]` waiting on an external dependency or decision
- `[done]` implemented and verified

## Context

The user has hardened the pipeline (shared loop, ctx cancellation, externalized prompts,
metadata-driven classifier, and expanded secret detection). They selected all three improvement
blocks from the previous plan:

- **Block A** — Prompt + preprocessing (pure Ramen, fast)
- **Block B** — Schema-guided generation (Ramen + udon, eliminates JSON parse failure class)
- **Block C** — Eval harness first (foundation for measuring everything else)

**Sequencing:** C → A → B, as three independent PRs. Block C first because every later change needs measurable before/after numbers. Block A second because it's high-leverage and fully Ramen-local. Block B last because it requires a coordinated change in `../udon` and benefits from having the eval harness in place to confirm parse-failure elimination.

The current pipeline writes intent.v1 prompt to `internal/synthesize/prompts/intent_generation.tmpl`, calls `chat.Chat` (a single text-out interface defined in `udon/pkg/rollout/llm.go:57`), parses the response with the brittle `extractJSON` (synthesize/prompt.go:189), and runs 31 deterministic checks. There is one example (`examples/support-email`) with no committed reference outputs.

---

## [done] Block C — Evaluation harness (foundation, do first)

### Goal
Make every prompt or pipeline change measurable: pass-rate, attempts-to-pass, per-check failure distribution, latency, token cost, by prompt version.

### Deliverables

**C1. Eval corpus (6 example briefs)**

Create `examples/eval/<name>/` directories, each containing `project.md`, `openapi/*.yaml` (where applicable), and a `reference/` subdirectory with hand-curated `intent.hcl`, `workflow.hcl`, `plan.json`. Briefs to add:

| Name | Pattern | Purpose |
| --- | --- | --- |
| `support-email` (existing) | fetch + classify + write | already present, lift to eval |
| `weather-toronto` | two-step API chain with `bind` (geocode → weather) | tests hidden-step expansion + bindings |
| `runtime-only-render` | `OpenAPI: none required`, fnct-only | tests runtime-only path |
| `paginated-list` | OpenAPI op with pagination params | tests required-parameter inference |
| `cmd-allowed-deploy` | `cmd` runtime explicitly allowed | tests runtime policy gate (positive) |
| `cmd-disallowed-deploy` | same brief, no `cmd` policy | tests runtime policy gate (negative — must fail with `intent.runtime_policy`) |

Six briefs is enough for signal without breaking the LLM budget. References are hand-curated once, then frozen.

**C2. `internal/eval` package**

New package with one entry point:

```go
type EvalResult struct {
    Name              string
    PromptVersion     string
    Provider          string
    Model             string
    Passed            bool
    AttemptsToPass    int    // 0 if did not pass within MaxAttempts
    FailingChecks     []string
    DurationMs        int64
    PromptTokensApprox int    // estimated, not authoritative
    Error             string
}

func RunOne(ctx context.Context, exampleDir string, opts synthesize.Options) EvalResult
func RunAll(ctx context.Context, evalRoot string, opts synthesize.Options) []EvalResult
```

`RunOne` wraps `synthesize.Synthesize`, captures the resulting `RefinementReport`, and produces an `EvalResult`. `RunAll` discovers `examples/eval/*/project.md` and runs them in parallel (worker pool sized by `--concurrency` flag, default 2 to avoid rate limits).

**C3. `cmd/ramen-eval` binary** (or `ramen eval` subcommand — pick subcommand for now to avoid binary proliferation)

```
ramen eval --root examples/eval [--name <single>] [--concurrency 2] [--out eval/runs/<ts>.json]
```

Writes a JSON file under `eval/runs/<ts>.json` with all results, plus a Markdown summary at `eval/runs/<ts>.md`. Exit non-zero if the *aggregate* pass-rate dropped below the previous run's pass-rate, OR if any specific brief that previously passed now fails.

**C4. `make eval` local release evidence**

`make eval` runs `ramen eval` with sensible defaults. Keep eval runs local/manual while the private
dependency layout is still changing and LLM cost is real. Store results under ignored `eval/runs/`
for manual release review.

**C5. Reference comparator**

`internal/eval/compare.go` — given a generated `intent.hcl` and a `reference/intent.hcl`, produce a structural diff:

- Step set match (by name)
- Step type match
- `operation` field match
- `bind`/`with` field map equivalence (order-insensitive)
- Inputs/outputs match

This is *not* used to fail the eval (the deterministic `Assess` already does that). It produces a richness signal — "passes the gates but is structurally different from reference" — useful for prompt regression detection.

### Files touched

- `examples/eval/*/project.md`, `openapi/*.yaml`, `reference/*` (new)
- `internal/eval/{eval.go, compare.go, report.go}` (new)
- `cmd/ramen/main.go` — add `eval` subcommand
- `Makefile` — add `eval` target
- `internal/eval/eval_test.go` (new) — covers the comparator and `RunOne` with a fake `ChatClient`

### Verification
- `make eval` runs locally with `GEMINI_API_KEY` set, produces `eval/runs/<ts>.json` and `.md`.
- Each of the 6 briefs reports a clear pass/fail and attempts-to-pass.
- Local release evidence includes JSON and Markdown reports.

### Estimated effort
~3 days. Most of it is curating the 5 new briefs and their reference outputs.

---

## [done] Block A — Prompt + preprocessing

### Goal
Reduce first-attempt failure rate on the eval corpus by tightening what the LLM sees and how it's instructed.

### Deliverables

**A1. Few-shot examples in the prompt template**

Bump `intentPromptVersion` from `intent.v1` → `intent.v2` (refinement.go:12). Rewrite `prompts/intent_generation.tmpl` to include a `## Examples` section with 3 worked examples covering:

1. **Single OpenAPI op** — input brief snippet → output JSON snippet
2. **Two-step chain with `bind`** — geocode then weather
3. **Runtime-only with `fnct`** — `OpenAPI: none required` plus a function contract

Examples come from the eval corpus references (Block C) — keep them as the single source of truth so prompt and eval don't drift. Embed them at build time via `//go:embed prompts/examples/*.json` and templated into the prompt.

**A2. Pre-LLM enrichment of project.md**

Extend `analyzeProject` (project_authoring.go:21) to extract:

- `Inputs:` section → `[]InputDecl{Name, Type, Required, Description}`
- `Outputs:` section → `[]OutputDecl{Name, From}`
- "Pass X to Y" sentences in `Data Flow` → `[]BindingHint{From, To, Field}`
- `Function Contracts` → `[]FunctionContract{Name, Inputs, Outputs, SideEffects}`

Add these to `projectPolicy` as new fields, then surface them in the user prompt as a new **REQUIRED bindings** section above the OpenAPI list:

```
## Required by project.md
- Step `classify_ticket` MUST receive `get_ticket.received_body` as input `ticket`.
- Step `write_draft` MUST receive `classify_ticket.received_body` as input `classification`.
- Inputs: ticket_id (string, required)
- Outputs: result from write_draft.received_body
```

This redirects LLM effort from "parse the brief" to "wire what's already extracted."

**A3. Lower temperature for intent generation**

Add a `withLLMTemperature` option in `udon/pkg/rollout/llm.go` and wire it through `runner.NewLLMClientFromEnv`. Default to 0.2 for intent generation, 0.0 for self-critique passes (Tier-2, future). Ramen-side: pass via a new `Options.IntentTemperature` field, default 0.2.

Provider mapping:
- Gemini: `generationConfig.temperature`
- OpenAI: `temperature`
- Anthropic: `temperature`

This is a small udon change; the provider request bodies in `llm.go` already exist, only need a temperature field added to each.

**A4. JSON shape cleanup**

In `intentJSONShape()` (synthesize/prompt.go:170), replace `"openapi/example.yaml"` with `"<primary openapi path provided above>"` so the model can't parrot the literal filename. Same for any other parroting risks (`step_name`, `prior_step` are fine because they describe the slot, but the openapi path is an actual artifact path and a real source of bugs).

**A5. Capture rendered prompt in refinement report on first attempt**

Add `RefinementReport.PromptSnapshot string` (refinement.go) — written only on attempt 1 to keep size bounded. Lets you reproduce a failed run without re-invoking the LLM.

**A6. Distinguish model failure from validation failure**

Add `RefinementAttempt.FailureClass string` (`"model" | "validation" | "infra" | ""`). Set in `addAttempt` based on the err type/wrap chain:
- `errors.Is(err, context.Canceled/DeadlineExceeded)` or contains `"API returned status"` → `"model"`
- `extractJSON` / decode / `RenderIntentHCL` errors → `"model"` (model produced bad output)
- Quality-gate failures → `"validation"`
- File I/O errors → `"infra"`

Useful for telling apart "the model crashed" from "the model produced parseable but wrong output" in eval reports.

### Files touched

- `internal/synthesize/prompts/intent_generation.tmpl` (rewrite, embed examples)
- `internal/synthesize/prompts/examples/*.json` (new — 3 example JSONs)
- `internal/synthesize/prompt.go` — bump version, render examples block, fix JSON shape literal, add required-bindings section
- `internal/synthesize/project_authoring.go` — extend `projectPolicy`, add extractors for Inputs/Outputs/BindingHints/FunctionContracts
- `internal/synthesize/refinement.go` — add `PromptSnapshot`, `FailureClass`, `intentPromptVersion = "intent.v2"`
- `internal/synthesize/synthesize.go` — wire `Options.IntentTemperature`, capture prompt snapshot, classify failure class in `runRefinement`
- `cmd/ramen/main.go` — add `--temperature` flag (default 0.2)
- `udon/pkg/rollout/llm.go` — add `withLLMTemperature` option and provider plumbing
- `udon/pkg/runner/llm_env.go` (or wherever `NewLLMClientFromEnv` lives) — accept temperature
- New tests: `prompt_test.go` for examples rendering; `project_authoring_test.go` for new extractors; `refinement_test.go` for failure-class classification

### Verification
- `make eval` (from Block C) shows aggregate pass-rate strictly improves over Block A baseline.
- `intent.v2` appears in `refinement.json`.
- `prompt_snapshot` is non-empty on attempt 1 of failing runs.
- `FailureClass` distinguishes "decode intent JSON" failures from "intent.openapi_refs" quality failures.
- Lower temperature setting confirmed in provider request body (test against fake transport).

### Estimated effort
~2 days, plus ~0.5 day in udon for the temperature option.

---

## [done] Block B — Schema-guided generation

### Goal
Eliminate the entire "extractJSON / decode intent JSON / RenderIntentHCL" failure class by having providers emit JSON that conforms to the intent schema natively.

### Deliverables

**B1. Promote `intentJSONShape` to a real JSON Schema**

Create `internal/synthesize/schemas/intent.schema.json` — a strict JSON Schema describing `rollout.Intent`. Include:

- Required fields: `workflow.name`, `workflow.description`, `steps`
- Step schema: `name`, `type` (enum: `http`, `openapi`, `fnct`, `cmd`, `ssh`), `do`, optional `operation`, `depends_on`, `with`, `bind`, `security`
- Conditional: if `type == "openapi"`, then `operation` is required
- `additionalProperties: false` on all object types

Validate the schema in local deterministic tests by parsing it with `santhosh-tekuri/jsonschema`
(already a dependency).

**B2. Add `StructuredChat` interface in udon**

In `udon/pkg/rollout/llm.go`:

```go
// StructuredChat extends ChatClient with provider-native structured output.
// Returns a JSON string that conforms to the supplied schema.
type StructuredChat interface {
    ChatClient
    StructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error)
}

type StructuredOpts struct {
    Temperature *float64
    MaxTokens   int
}
```

Per-provider implementation:

- **Gemini** (`GeminiClient.StructuredChat`): set `generationConfig.responseMimeType = "application/json"`, `generationConfig.responseSchema = <schema>`. Note: Gemini's schema dialect is a subset of JSON Schema — needs a small translator (or restrict the intent schema to Gemini-compatible features).
- **OpenAI** (`OpenAIClient.StructuredChat`): use `response_format = { type: "json_schema", json_schema: { name, strict: true, schema } }`. Available on `gpt-4o`, `gpt-4o-mini`, etc.
- **Anthropic** (`AnthropicClient.StructuredChat`): use tool-use with one tool `{ name: "emit_intent", input_schema: <schema> }` and `tool_choice = { type: "tool", name: "emit_intent" }`. Extract `tool_use.input`.

Each implementation falls through to `Chat()` + a clear "structured output unsupported" error if the provider/model doesn't support it; Ramen catches this and falls back.

**B3. Wire structured generation into `generateIntent`**

In `internal/synthesize/prompt.go:generateIntent`:

```go
if structured, ok := chat.(rollout.StructuredChat); ok {
    schema := embeddedIntentSchema  // //go:embed
    raw, err := structured.StructuredChat(ctx, messages, schema, rollout.StructuredOpts{Temperature: &temp})
    if err == nil {
        // skip extractJSON; raw is already valid JSON
        return decodeIntent(raw)
    }
    // fall through to legacy path on error
}
// existing chat.Chat + extractJSON path as fallback
```

Record in `RefinementAttempt` whether structured or legacy path was used (`Mode string` field). This lets eval reports break out structured-mode pass rates separately.

**B4. Drop `extractJSON` from the happy path**

Once structured mode is the default and verified, `extractJSON` becomes only a fallback for non-structured providers. Add a metric in eval reports for "fell back to extractJSON" so we can track when fallback is hit.

**B5. Update prompt for structured mode**

When structured mode is in use, the prompt no longer needs to say "Return only JSON. Do not include Markdown." — the schema enforces it. Conditionally trim those lines from the system prompt in structured mode and bump version to `intent.v3` (or `intent.v2.structured`).

### Files touched

- `internal/synthesize/schemas/intent.schema.json` (new)
- `internal/synthesize/prompt.go` — embed schema, wire `StructuredChat`, conditional prompt
- `internal/synthesize/refinement.go` — add `Mode` to `RefinementAttempt`
- `udon/pkg/rollout/llm.go` — `StructuredChat` interface, `StructuredOpts`, three provider impls
- `udon/pkg/rollout/llm_test.go` — fake transports asserting request body contains the right schema fields
- `internal/synthesize/prompt_test.go` — fake `StructuredChat` client, asserts structured path is taken when available, falls back when not

### Verification
- Eval corpus run with `intent.v3.structured`: zero `extractJSON failed` or `decode intent JSON` failures across all 6 briefs.
- For Anthropic-served runs, `tool_use.input` is decoded directly.
- Fallback to legacy path works when a `ChatClient` does not implement `StructuredChat` (covered by test using a basic fake).

### Estimated effort
~3–5 days, of which roughly half is in udon (provider implementations + tests).

---

## Cross-cutting

- **Documentation**: update `docs/architecture.md` to describe the eval harness; update `WORKFLOW.md` agent prompt to mention the `eval` subcommand.
- **Automation**: keep development and eval gates local/manual until the private dependency layout
  stabilizes.
- **Backwards compatibility**: every change must be additive. Existing `synthesize`/`build`/`promote`/`assess` commands and their on-disk artifacts stay unchanged. New fields in `RefinementReport` are appended (JSON unmarshal tolerates extras).

---

## Recommended PR shape

1. **PR 1 — eval harness** (Block C, ~3 days). Lands `internal/eval`, the corpus, and `make eval`.
   No behavior change to the pipeline.
2. **PR 2 — prompt v2 + preprocessing** (Block A, ~2 days + 0.5 day udon). Lands few-shot examples, project.md extractors, temperature, prompt snapshot, failure class. Eval shows the delta.
3. **PR 3 — schema-guided generation** (Block B, ~3–5 days + udon). Lands `StructuredChat`, intent JSON Schema, structured path in `generateIntent`. Eval confirms parse-failure-class elimination.

Each PR is independently shippable and reverts cleanly.

---

## Critical files to modify (quick index)

| File | Block | What changes |
| --- | --- | --- |
| `internal/synthesize/prompts/intent_generation.tmpl` | A | Add few-shot, conditional structured-mode lines |
| `internal/synthesize/prompts/examples/*.json` | A | New — embedded examples |
| `internal/synthesize/schemas/intent.schema.json` | B | New — strict JSON Schema |
| `internal/synthesize/prompt.go` | A, B | Required-bindings section, StructuredChat dispatch |
| `internal/synthesize/project_authoring.go` | A | Extract Inputs/Outputs/BindingHints/FunctionContracts |
| `internal/synthesize/refinement.go` | A, B | `PromptSnapshot`, `FailureClass`, `Mode`, version bump |
| `internal/synthesize/synthesize.go` | A | Wire temperature, capture snapshot, classify failure |
| `internal/eval/{eval,compare,report}.go` | C | New package |
| `cmd/ramen/main.go` | A, C | `--temperature` flag, `eval` subcommand |
| `Makefile` | C | `eval` target |
| `examples/eval/<name>/...` | C | 5 new briefs + references |
| `udon/pkg/rollout/llm.go` | A, B | `withLLMTemperature`, `StructuredChat` interface + 3 impls |
| `udon/pkg/runner/llm_env.go` | A | Accept temperature in `NewLLMClientFromEnv` |

---

## Verification (end-to-end, after all three PRs)

1. `make check` — existing tests + sibling check passes.
2. `make eval` — produces `eval/runs/<ts>.json` and `.md` with pass-rate, attempts-to-pass, and structured-mode usage per brief.
3. Compare three eval runs (post-C, post-A, post-B): aggregate pass-rate strictly increases across the three; `extract_json_failures` count goes to zero in run 3.
4. `grep "intent.v3" expected/refinement.json` confirms new prompt version is recorded after Block B.
5. Manually break one project.md (e.g., remove the `Runtime Policy` section in `weather-toronto`) and verify the eval reports it as a failing brief with a clear failing check, not as an infra error.
