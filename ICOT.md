# iCoT Authoring Design

## Purpose

iCoT is Ramen's interactive thinking layer for turning a project idea into reviewed authoring
artifacts. It is not a synthesis or execution engine. Its job is to help an operator clarify intent,
capture the structured workflow contract, and preserve the human policy context needed by later
Ramen stages.

The implemented design keeps the current Ramen artifact model:

- `project.md` is the human-readable policy and prose artifact.
- `workflows/intent.hcl` is the structured saved contract.
- `ramen build --example <dir>` is the next step after iCoT when intent is already authored.

There is no `project.spec.yaml` sidecar in this design. Earlier notes explored that option, but the
current implementation uses `workflows/intent.hcl` directly to avoid introducing a second structured
source of truth.

## Flow

The command is:

```bash
go run ./cmd/icot --example ./examples/<name>
```

The session asks bounded questions, updates a running summary, validates the draft, and then saves
both `project.md` and `workflows/intent.hcl`. It also creates the expected example directories when
needed:

- `openapi/`
- `workflows/`
- `expected/`

The high-level loop is:

1. Capture a short workflow brief.
2. Fill workflow name, goal, and optional workflow timeout/idempotency metadata.
3. Select whether OpenAPI/API steps are needed.
4. Capture runtime inputs.
5. Capture steps, operations, request fields, optional step timeouts, and step-to-step binds.
6. Capture outputs.
7. Capture runtime approvals for `cmd` or `ssh` only if those runtimes appear.
8. Capture side-effect scope.
9. Capture credential binding names only.
10. Capture safety notes and fallback behavior.
11. Render and validate `intent.hcl`, then ask for final `save`, `edit <slot>`, or `cancel`.

The timeout and idempotency questions default to blank. When left blank, iCoT emits no timeout or
idempotency metadata. When provided, the values are written to `workflows/intent.hcl`; `project.md`
remains the readable policy by-product.

## Side-Effect Scope

iCoT asks for a structured side-effect scope after steps and before credentials:

- `read-only`: generate and validate artifacts only; no workflow execution or external effects.
- `sandbox-only`: sandbox proof runs require `approved_for_sandbox`, approved bindings, and a
  trusted runner; production execution is not approved by the project contract.
- `after-approval`: the full Ramen/Symphony approval path, including `approved_for_sandbox`,
  `approved_for_production`, approved credential bindings, and trusted-runner execution.

The scope controls the boilerplate rendered in `project.md`'s Safety and Approval Boundary section.
Operator-entered safety notes are appended after that boilerplate.

## Autosave And Resume

Interactive state is local working state, not a source artifact. iCoT autosaves the incomplete
session after each accepted slot:

```text
<example>/.icot/session.yaml
```

Default behavior:

- If no `--answers` or `--from-example` is provided, iCoT loads this draft automatically.
- A successful save deletes the draft.
- `cancel` deletes the draft.
- EOF or an error preserves the draft so the operator can resume.
- `--print` may load an existing draft but does not write autosave, project, intent, or backup
  files.

`.gitignore` excludes `.icot/session.yaml` because it is local authoring state.

## Reconcile

Reconcile regenerates only `project.md` from existing intent:

```bash
go run ./cmd/icot reconcile --example ./examples/<name>
```

It parses `workflows/intent.hcl`, loads existing `project.md` when present, preserves local policy
content such as credential bindings, safety notes, fallback behavior, side-effect scope, and runtime
approvals, then rewrites `project.md`.

Useful modes:

```bash
go run ./cmd/icot reconcile --example ./examples/<name> --print
go run ./cmd/icot reconcile --example ./examples/<name> --yes
```

`--print` writes nothing. Normal reconcile backs up the old `project.md` before replacing it and
does not rewrite or back up `workflows/intent.hcl`.

## Lint And Drift

`icot lint` keeps the existing markdown authoring checks and secret-like content checks. When an
example has `workflows/intent.hcl`, lint also parses intent and reports advisory drift warnings.

```bash
go run ./cmd/icot lint --example ./examples/<name>
```

Hard failures:

- unreadable `project.md`
- existing fail-level markdown checks, such as secret-like content
- invalid or incomplete `workflows/intent.hcl`

Warnings only:

- goal drift
- OpenAPI reference drift
- input drift
- output drift
- data-flow or bind drift
- function-step drift
- credential binding drift
- `cmd` or `ssh` runtime approval drift

Drift warnings do not make lint exit nonzero unless another fail-level check exists.

## Eval Replay

`icot replay-eval` replays each eval fixture's `reference/intent.hcl` through the real iCoT
question/answer loop. It can run with Gemini/OpenAI/Anthropic extraction enabled and writes the
simulated turns, LLM calls, stdout, generated `intent.hcl`, generated `project.md`, and reference
comparison result under an ignored `eval/runs/icot-replay-*` directory.

```bash
go run ./cmd/icot replay-eval --root ./examples/eval --provider gemini --model gemini-2.5-flash
```

This command is local evidence, not a committed fixture format. The eval references remain the
source of truth for intent correctness.

## Atomic Artifact Writes

iCoT treats final artifact writes as a small transaction:

1. Validate/render artifacts before touching final files.
2. Create parent directories.
3. Write temp files for all target artifacts.
4. Back up existing final files when overwriting.
5. Rename temp files into place.
6. Best-effort restore from backups if a rename fails.

Generated backups are ignored:

```text
**/project.md.bak.*
**/workflows/intent.hcl.bak.*
```

## LLM Assistance

LLM use is optional and bounded. iCoT works offline with `--no-llm` or when no provider credential is
available.

The extractor has three roles:

- `Kickoff`: obvious prefill from the opening brief only. It must not invent OpenAPI paths, URLs,
  operation IDs, inputs, outputs, or steps.
- `Refine`: prose cleanup for goal, safety, and fallback while preserving structural fields exactly.
- `Disambiguate`: rank existing local OpenAPI documents only. It filters out invented paths and
  never imports URLs in this pass.

Prompt templates live under:

```text
internal/icot/elicitor/prompts/
```

LLM-derived session annotations record a prompt version so future prompt changes can be traced.

## Command Summary

```bash
# Guided authoring; writes project.md and workflows/intent.hcl.
go run ./cmd/icot --example ./examples/<name>

# Print rendered project.md and intent.hcl without writing files.
go run ./cmd/icot --example ./examples/<name> --print

# Seed from an existing example.
go run ./cmd/icot --from-example ./examples/eval/runtime-only-render --example ./examples/<name>

# Use YAML or JSON session/legacy answers.
go run ./cmd/icot --answers ./answers.yaml --example ./examples/<name>

# Rebuild project.md from workflows/intent.hcl.
go run ./cmd/icot reconcile --example ./examples/<name>

# Check authoring quality, intent parseability, and advisory drift.
go run ./cmd/icot lint --example ./examples/<name>

# Replay eval references through iCoT and save transcripts.
go run ./cmd/icot replay-eval --root ./examples/eval --provider gemini --model gemini-2.5-flash
```

## Boundaries

iCoT belongs in Ramen because it owns guided authoring, templates, examples, and policy capture.

- Public workflow semantics still belong in `../uws`.
- Generic OpenAPI/UWS compilation and execution still belong in `../udon`.
- Symphony workflow orchestration is configured through Ramen policy artifacts and should not be
  forked here.

iCoT must not execute side-effectful workflows. Production side effects remain gated behind approved
trusted-runtime paths.

## Relationship To Earlier Notes

The existing `icot.md` and `llm-icot.txt` were planning notes. They contain useful ideas that were
kept:

- interactive challenge/response authoring
- optional LLM kickoff and refinement
- side-effect scope
- reconcile
- drift lint
- autosave/resume
- atomic writes

The main architectural change from those notes is that the structured source of truth is now
`workflows/intent.hcl`, not `project.spec.yaml`.
