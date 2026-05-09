# OpenUdon

[![test](https://github.com/OpenUdon/openudon/actions/workflows/test.yml/badge.svg)](https://github.com/OpenUdon/openudon/actions/workflows/test.yml)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)

OpenUdon is the public UWS workflow authoring, review, package, and executor-handoff tool. It can run
directly or under optional Symphony-managed orchestration, and it hands approved packages to a
trusted executor boundary such as the `udon` runtime.

It owns project templates, optional Symphony workflow policy, example artifacts, deterministic
validation, review handoff evidence, package digests, credential policy, and trusted-runner glue.
Public workflow semantics belong in `github.com/OpenUdon/uws`; OpenAPI search/discovery/import/indexing belongs in
`github.com/OpenUdon/apitools`.

## Quick Start

Install the main CLI:

```bash
go install github.com/OpenUdon/openudon/cmd/openudon@latest
```

Optional companion tools:

```bash
go install github.com/OpenUdon/openudon/cmd/icot@latest
go install github.com/OpenUdon/openudon/cmd/udon-runner@latest
```

Useful checks:

```bash
go test ./...
go vet ./...
go run ./cmd/openudon check
go run ./cmd/openudon check-apitools-boundary
go run ./cmd/openudon validate ./examples/uws-validation
make check
make release-check
git diff --check
```

Execute through `openudon run` and the portable run-config handoff. Configure the final executor
with `OPENUDON_EXECUTOR` as either an absolute binary path or `docker://<image>`.

## Layout

- `cmd/openudon`: local CLI for checks, synthesis, assessment, eval, readiness, approval templates,
  and trusted execution.
- `cmd/icot`: guided authoring CLI for `project.md` and `workflows/intent.hcl`.
- `internal/`: reusable OpenUdon implementation.
- `examples/`: committed examples and eval corpus.
- `templates/project.md`: starter project brief.
- `docs/`: detailed architecture, safety, operator, XRD, and release notes.

## Execution Boundary

The intended lifecycle is:

```text
natural-language project brief
  -> Symphony-managed implementation issue or local authoring session
  -> generated OpenAPI/UWS artifacts
  -> deterministic validation and review
  -> approved handoff package
  -> trusted executor handoff
```

`openudon synthesize`, `openudon build`, `openudon promote`, `openudon assess`, `cmd/icot`, and eval commands
generate, compile, validate, and report on artifacts. They do not execute production workflows.

`openudon run` is separate. It validates the handoff manifest, stored and current quality, approval
JSON, package digest, and tier before writing a non-secret `openudon.executor-run.v1` run config and
invoking the Go trusted executor runner. The runner stages the reviewed UWS/OpenAPI files into the
run workdir before calling the configured executor.
The runner is also available directly as `go run ./cmd/udon-runner --config <run-config.json>`.
`OPENUDON_EXECUTOR` accepts either an absolute path to an executable file or `docker://<image>`.
The outer `OPENUDON_UDON_RUNNER` override must be an absolute path to an executable file.

## Authoring With iCoT

iCoT turns a project idea into reviewed authoring artifacts. It writes `project.md` and
`workflows/intent.hcl`; it does not synthesize compiled artifacts or execute workflows.

```bash
go run ./cmd/icot --example ./examples/<name>
```

Common modes:

```bash
# Print rendered project.md and intent.hcl without writing files.
go run ./cmd/icot --example ./examples/<name> --print

# Use the fixed manual flow without LLM extraction.
go run ./cmd/icot --example ./examples/<name> --no-llm

# Seed from an existing example.
go run ./cmd/icot --from-example ./examples/eval/runtime-only-render --example ./examples/<name>

# Use YAML or JSON session/legacy answers.
go run ./cmd/icot --answers ./answers.yaml --example ./examples/<name>

# Rebuild project.md from workflows/intent.hcl.
go run ./cmd/icot reconcile --example ./examples/<name>

# Check authoring quality, intent parseability, and advisory drift.
go run ./cmd/icot lint --example ./examples/<name>

# Replay eval references through iCoT and save ignored transcripts.
go run ./cmd/icot replay-eval --root ./examples/eval --provider copilot-api --model gpt-5.4-mini
```

iCoT autosaves incomplete local sessions under `<example>/.icot/session.yaml` and resumes by
default. Successful saves delete the autosave. Transcripts are written under
`<example>/.icot/transcript.json` unless `--no-transcript` is used. These local files are ignored by
git.

Side-effect scope in iCoT:

- `read-only`: generate and validate artifacts only.
- `sandbox-only`: sandbox proof runs require `approved_for_sandbox`, approved bindings, and a
  trusted runner.
- `after-approval`: sandbox and production execution require the full OpenUdon/Symphony approval path.

## Synthesize And Assess

Generate all reviewed artifacts for an example:

```bash
export COPILOT_API_BASE_URL=http://localhost:4141
export OPENUDON_LLM_PROVIDER=copilot-api
export OPENUDON_LLM_MODEL=gpt-5.4-mini

go run ./cmd/openudon synthesize \
  --example ./examples/support-email \
  --provider "$OPENUDON_LLM_PROVIDER" \
  --model "$OPENUDON_LLM_MODEL" \
  --max-attempts 5
```

The command reads `project.md`, discovers or imports OpenAPI documents under `openapi/`, writes
`workflows/intent.hcl` when needed, and generates equivalent public UWS HCL/YAML workflow
artifacts:

```text
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

Use narrower stages after editing artifacts:

```bash
# intent.hcl -> workflow/UWS/plan/review/quality
go run ./cmd/openudon build --example ./examples/support-email --max-attempts 5

# workflow.hcl -> UWS/review/quality
go run ./cmd/openudon promote --example ./examples/support-email

# quality reports only
go run ./cmd/openudon assess --example ./examples/support-email
```

The bounded refinement loop records retried stages, failed checks, and stop reason in
`expected/refinement.json`.

## Quality And Repair Loop

The pipeline is validation-first:

1. Run `synthesize` for a new or substantially changed `project.md`.
2. If it fails, read `expected/refinement.json` and `expected/quality.json`.
3. Repair the earliest failing stage.
4. For `openapi.*`, add a valid local OpenAPI file or explicit OpenAPI URL.
5. For `intent.*`, edit `project.md` or `workflows/intent.hcl`, then rerun `build`.
6. For `workflow.*`, prefer improving intent and rerunning `build`; use `promote` and `assess` for
   narrow workflow repairs.
7. For `uws.*`, `review.*`, `symphony_handoff.*`, or `artifacts.*`, repair the generated artifact
   or evidence, then run `promote` or `assess`.
8. Stop after the configured attempt limit and report blocking checks if quality still fails.

## Eval And Release Evidence

Use deterministic checks for routine development:

```bash
go test ./...
go vet ./...
make check
git diff --check
```

Use the eval harness when changing prompts, synthesis/refinement behavior, model defaults, or
quality gates that could affect generated artifacts:

```bash
go run ./cmd/openudon eval --root ./examples/eval --provider copilot-api --model gpt-5.4-mini
```

Eval reports are written under ignored `eval/runs/`. They include pass/fail summaries,
provider/model/mode/prompt-version breakdowns, approximate prompt-token totals, generated workspace
paths, provider drift watch data, and comparison against a previous report when available.

Use release gates only for candidate release evidence:

```bash
make release-eval
```

`make release-eval` uses `OPENUDON_LLM_PROVIDER` and `OPENUDON_LLM_MODEL`, defaulting to `copilot-api` and
`gpt-5.4-mini`, and requires the current eval corpus size as the minimum brief count.

## Readiness

Local readiness reports record optional sibling checkout presence, deterministic gate results, git
state, ignored local artifacts, provider credential environment presence as booleans only, and
current maintainer automation policy.

```bash
go run ./cmd/openudon readiness --out eval/readiness/local.json
go run ./cmd/openudon readiness --run-gates --out eval/readiness/local.json
```

GitHub Actions runs public-module vet/test gates without local sibling checkouts. Real-provider
release evidence remains local/manual.

## Trusted Execution

After artifacts pass review, generate approval JSON with the current package digest:

```bash
mkdir -p approvals
go run ./cmd/openudon approval-template \
  --example ./examples/support-email \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/support-email-sandbox.json
```

Validate approval, quality, handoff policy, package digest, and tier compatibility before trusted
executor handoff:

```bash
go run ./cmd/openudon run \
  --example ./examples/support-email \
  --tier sandbox \
  --approval approvals/support-email-sandbox.json
```

Use `--dry-run` to validate all gates and write run config without invoking the executor.

Approval JSON shape:

```json
{
  "version": "openudon.approval.v1",
  "scope": "examples/support-email",
  "state": "approved_for_sandbox",
  "reviewer": "Reviewer Name",
  "approved_at": "2026-04-29T12:00:00Z",
  "expires_at": "2026-05-06T12:00:00Z",
  "package_sha256": "<current handoff package digest>",
  "notes": "optional"
}
```

Tier rules:

- `sandbox` accepts `approved_for_sandbox` or `approved_for_production`.
- `production` accepts only `approved_for_production`.
- Expired approvals fail.
- Scope mismatch fails.
- Package digest mismatch fails.
- Stored or current quality failures fail.
- Malformed handoff manifests fail.
- Credential-value artifacts and direct production execution remain prohibited.
- Approval JSON and saved run configs from before the OpenUdon package rename should be regenerated
  so scope, version, and package digest fields match the current artifact set.

## Symphony Agent Workflow

OpenUdon issues may be run through Symphony-managed Codex sessions. Agents should follow this policy:

- Use UWS as the workflow interchange format.
- Use OpenAPI for HTTP method, path, schema, server, and security details.
- Use extension-owned UWS operations for non-HTTP runtimes such as SMTP, command execution, SSH,
  SQL, or LLM calls.
- Use `../uws` for public schema/model validation.
- Use `openudon run` to hand approved UWS/OpenAPI packages to a trusted executor such as udon.
- Use `../symphony` only as the work orchestration service.
- Do not execute production side effects directly from an agent session.
- If execution is requested, produce or update the approved artifact and document the trusted runner
  command.

Expected artifact locations:

```text
examples/<name>/project.md
examples/<name>/openapi/
examples/<name>/workflows/intent.hcl
examples/<name>/workflows/workflow.hcl
examples/<name>/workflows/workflow.uws.yaml
examples/<name>/expected/plan.json
examples/<name>/expected/plan.md
examples/<name>/expected/discovery.json
examples/<name>/expected/refinement.json
examples/<name>/expected/refinement.md
examples/<name>/expected/review.md
examples/<name>/expected/symphony-handoff.json
examples/<name>/expected/quality.json
examples/<name>/expected/quality.md
```

Before handoff:

```bash
go test ./...
go vet ./...
make check
git diff --check
go run ./cmd/openudon validate examples/uws-validation
go run ./cmd/openudon assess --example examples/<name>
```

If side-effectful execution is explicitly requested, use `openudon run` with approval JSON. Do not run
production effects from synthesis, build, promote, assess, iCoT, or eval.

## Model And Credential Guidance

Use the local `copilot-api` proxy with `gpt-5.4-mini` as the default model for synthesis. OpenUdon
reliability comes mostly from prompt preprocessing, structured output when available, deterministic
quality gates, and bounded repair attempts. Escalate to a larger model only after the default model
fails deterministic checks.

LLM credentials must come from provider environment variables such as `COPILOT_API_BASE_URL`,
`COPILOT_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, or `ANTHROPIC_API_KEY`. Do not place tokens in
prompts, commands, examples, or workflow artifacts.

Use `OPENUDON_LLM_PROVIDER` and `OPENUDON_LLM_MODEL` when you want shell-level defaults for local
LLM-assisted commands; explicit `--provider` and `--model` flags still take precedence.

## More Documentation

- [Project authoring](docs/project-authoring.md)
- [Intent contract](docs/intent.md)
- [Data flow](docs/data-flow.md)
- [Safety](docs/safety.md)
- [Eval gallery](docs/eval-gallery.md)
- [Release note template](docs/release-note-template.md)
- [Contributing](CONTRIBUTING.md)
- [License](LICENSE)
