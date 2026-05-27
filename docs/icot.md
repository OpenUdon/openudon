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

# Ask only when iCoT has no safe default/answer, or confidence requires review.
go run ./cmd/icot --example ./examples/<name> --prompt-mode fast

# Experimental: let pre-final flow review apply bounded safe repairs.
go run ./cmd/icot --example ./examples/<name> --review-repair

# Seed from an existing fixture.
go run ./cmd/icot --from-example ./examples/eval/weather-toronto --example ./examples/<name>

# Use YAML or JSON answers.
go run ./cmd/icot --answers ./answers.yaml --example ./examples/<name>

# Rebuild project.md from workflows/intent.hcl.
go run ./cmd/icot reconcile --example ./examples/<name>

# Check brief quality, intent parseability, and drift.
go run ./cmd/icot lint --example ./examples/<name>

# Noninteractive agent mode: write final artifacts when complete, or return needs_input.
go run ./cmd/icot --example ./examples/<name> --agent --json

# Structured lint report.
go run ./cmd/icot lint --example ./examples/<name> --json

# Provider-free reliability scorecard over the eval corpus.
go run ./cmd/icot scorecard --root examples/eval --out eval/runs/icot-scorecard-local

# Include curated natural-language authoring variants.
go run ./cmd/icot scorecard --root examples/eval --include-variants --out eval/runs/icot-authoring-scorecard-local

# Verify scorecard report JSON plus digest sidecar. `make icot-authoring-scorecard`
# and `make release-saas-check` run this automatically for the provider-free scorecard.
go run ./cmd/icot report verify --file eval/runs/icot-authoring-scorecard-local/scorecard.json

# Validate variant metadata and reference-seeded clear slots without running scorecard.
go run ./cmd/icot variants validate --root examples/eval

# Check provider-family coverage across positive, missing-detail, and unsafe-negative variants.
go run ./cmd/icot variants coverage --root examples/eval

# Optional real-LLM natural-language authoring evidence.
go run ./cmd/icot authoring-eval --root examples/eval --include-variants --provider copilot-api --model gpt-5.4-mini --out eval/runs/icot-authoring-eval-local

# Optional/manual verification for real-LLM authoring evidence.
go run ./cmd/icot report verify --file eval/runs/icot-authoring-eval-local/authoring-eval.json

# Bounded deterministic repair for mappings, outputs, and depends_on only.
go run ./cmd/icot repair --example ./examples/<name> --dry-run --json

# Replay eval fixtures with prompt-mode and repair metrics.
go run ./cmd/icot replay-eval --root examples/eval --prompt-mode fast --review-repair
```

See [iCoT Session Files](icot-session-schema.md) for the accepted `--answers` shapes and
[iCoT Transcripts](icot-transcript.md) for the ignored local transcript format.

`--prompt-mode full` is the default when the flag is omitted; it prints every question and waits for
you to confirm or replace defaults. `--prompt-mode normal` prints high-confidence and review-level
defaulted questions and automatically accepts them, but still asks when the default is missing,
low-confidence, conflicting, or tied to a blocking review decision. `--prompt-mode fast` silently
accepts high-confidence and review-level defaults, suppresses catalog/status chatter plus
review-only fallback and assumption text, and asks only for required values without a safe default
or with low/conflicting confidence. Automatically accepted defaults, assumptions, and decision
evidence are still recorded in the transcript.

For `icot replay-eval`, the omitted `--prompt-mode` default is `fast` so replay metrics measure the
progressive loop's defaulted path instead of requiring a manual answer script for every prompt.

## Agent And JSON Modes

`--agent` is the noninteractive iCoT mode for local agents or scripts. It does not read blocking
prompts. If the provided session, answers, draft, or seed fixture is complete, it writes
`project.md` and `workflows/intent.hcl` using the normal atomic write path. If required authoring
state is missing, it returns a structured `needs_input` report with the top readiness issue,
suggested answer, failure family, and all readiness issues.

`--json` writes an `openudon.icot-author-report.v1` report to stdout. `--report <path>` writes the
same report to a file. `icot lint --json` writes `openudon.icot-lint-report.v1` with project checks,
intent parse status, drift warnings, and the first failure family.

`icot scorecard` runs the provider-free seed/build reliability path over eval fixtures. It writes
`openudon.icot-scorecard.v1` under the requested output directory and records expected outcome,
observed outcome, fixture class, first failure family, failure codes, prompt/readiness provenance,
run ID, commit, generation time, and the command used to produce the report. The command validates
report consistency before write and emits a `scorecard.json.sha256` digest sidecar. Scorecards are
marked `retention_class: release_evidence`, `contains_provider_output: false`, `safe_to_archive:
true`, and `redaction_required_before_share: false`. With
`--include-variants`, it also runs checked-in natural-language authoring variants from
`reference/authoring-variants.json` and groups results by provider family, variant class, and
failure family. It also counts missing-detail or unsafe-negative variants that unexpectedly observe
`pass` as explicit false-pass counters, and fails variants that return `needs_input` without top
issue diagnostics. This variant lane mutates reviewed reference packages and verifies deterministic
package behavior; it is not proof that a live LLM generated the workflow from the alternate brief.
It does not call an LLM, retrieve remote provider metadata, or execute workflows.

`icot variants validate` checks `reference/authoring-variants.json` metadata without generating
workspaces. It validates expected failure-family names, duplicate IDs, missing-detail expectations,
required `expected_top_issue_code` and `expected_top_issue_slot` values for `needs_input`
variants, and `seed_from_reference` `clear_fields` or `clear_slots` against the fixture reference
intent. Use it before scorecard runs when editing variant metadata. The scorecard compares observed
top issue code and slot to those expectations so a variant cannot pass by asking the wrong
follow-up question.

`icot variants coverage` aggregates checked-in authoring variants by provider family and fails if
any provider family lacks at least one `positive`, `missing-detail`, or `unsafe-negative` variant.
This keeps corpus breadth explicit before the scorecard runs.

`icot authoring-eval` is the optional real-LLM authoring lane. It runs selected fixture briefs or
`--include-variants` entries through the iCoT progressive draft path with LLM extraction enabled,
then runs lint/build-equivalent checks and compares the generated `intent.hcl` against the reviewed
reference. The report is `openudon.icot-authoring-eval.v1` and records provider/model, prompt
version, readiness classifier version, run ID, commit, command, LLM call count, generated paths,
failure family, drift counts, and per-variant pass/fail. The report is consistency-checked before
write and emits an `authoring-eval.json.sha256` digest sidecar. Authoring-eval reports are marked
`retention_class: local_ephemeral`, `contains_provider_output: true`, `safe_to_archive: false`,
and `redaction_required_before_share: true`.
Generated project files, intents, transcripts, and the final report JSON are scanned for
credential-like literal values before the report is accepted. Failures include a structured
`failure_category` such as `provider_unavailable`, `provider_timeout`, `malformed_model_json`,
`structured_output_unsupported`, `model_refusal`, `incomplete_draft`, `lint_fail`,
`credential_scan_fail`, `build_fail`, or `reference_drift`. Keep this evidence local/manual; it can
spend model quota and is not part of `release-check` or `release-saas-check`.

`icot report verify --file <report.json>` verifies archived `openudon.icot-scorecard.v1` and
`openudon.icot-authoring-eval.v1` reports after generation. It checks the report version, summary
counters, variant top-issue expectations, authoring-eval failure categories, pass/fail consistency,
retention/share-safety metadata, and the adjacent `.sha256` digest sidecar.

`icot repair` is a bounded deterministic repair command. It may edit request mappings, output
sources, and `depends_on` only. It rejects source document, operation ID, credential binding,
side-effect policy, and runtime/profile mutations. Use `--dry-run --json` to inspect proposed
repairs before writing.

With LLM extraction enabled, iCoT runs a bounded pre-final flow review before printing the current
draft. The review is advisory and focuses only on cross-step data-flow mistakes that deterministic
checks may miss, such as an email/report step not consuming the data it should send.

The review classifies each warning with a flow-gap kind and remediation action. Gap kinds include
missing local transform/report steps, missing API prework, disconnected notifications, ambiguous
outputs, operation mismatches, unavailable sources, unclear intent, and narrow repairable wiring.
Invalid or absent model classifications are reclassified locally before use.

The pre-final review does not mutate the draft by default. Unresolved issues are preserved as
non-executable `intent.hcl` comments with the gap kind, remediation action, slot, evidence, and any
suggested review. `--review-repair` is an experimental opt-in mode that can make at most two bounded
repair attempts from flow-review suggestions. It first applies narrow request-mapping,
output-source, and `depends_on` repairs, then may add a local `fnct` transform/report/render step
only when the goal clearly asks for produced content and exactly one existing producer step can feed
it. It rejects source, operation, credential, side-effect-scope, and ambiguous structural mutations
and records the repair attempt in the transcript.

`fast` mode still stops for true ambiguity. If the flow review marks an issue as requiring user
intent, iCoT asks one forced high-priority question even when defaulted prompts would normally be
accepted silently. The answer is recorded as decision evidence; iCoT does not hide a workflow rewrite
behind that answer.

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

If the goal clearly asks to stop, render, or report a missing or ambiguous
provider/API/source capability and no usable API source or operation exists,
iCoT can emit a deterministic local gap-report draft instead of an API workflow.
That fallback creates required `provider` and `action` inputs, a
`render_capability_gap` `fnct` step wired from those inputs, and a `gap_report`
output. Transcript decision evidence marks it as a no-source safety fallback,
not an execution plan.

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
6. Run a bounded advisory flow review that looks only for cross-step data-flow
   mistakes, optionally apply bounded `--review-repair` fixes, then show the resulting draft,
   assumptions, decision evidence, and warnings for confirmation. If the
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
