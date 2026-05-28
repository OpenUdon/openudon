# OpenUdon

[![test](https://github.com/OpenUdon/openudon/actions/workflows/test.yml/badge.svg)](https://github.com/OpenUdon/openudon/actions/workflows/test.yml)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)

OpenUdon is the public UWS workflow authoring, review, package, and executor-handoff tool. It can run
directly or under optional external orchestration, and it hands approved packages to a
trusted executor boundary such as the `udon` runtime.

It owns project templates, optional workflow orchestration policy, example artifacts, deterministic
validation, review handoff evidence, package digests, credential policy, and trusted-runner glue.
Public workflow semantics belong in `github.com/OpenUdon/uws`; API/event source metadata discovery,
import, materialization, search, and indexing belong in `github.com/OpenUdon/apitools`; static
Terraform/OpenTofu parsing for `openudon convert tf` belongs in `github.com/OpenUdon/tfconfig`.
OpenUdon can stage OpenAPI, Google Discovery, AWS Smithy, AsyncAPI, GraphQL, OpenRPC,
gRPC/protobuf, and OData source documents as first-class UWS source descriptions when the trusted
executor supports them.

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
make eval-seed-build
make release-saas-check
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

Authoring details:

- [Agentic SaaS authoring](docs/agentic-saas-authoring.md) describes the M15 path for common SaaS
  workflows and the role of n8n-derived evidence.
- [SaaS operator release path](docs/saas-operator-release.md) gives the provider-free demo from
  strict SaaS fixtures to approval JSON and trusted-runner dry-run evidence.
- [Project briefs](docs/project-authoring.md) and [Data Flow](docs/data-flow.md) describe the
  reviewable artifact contracts.

## Execution Boundary

The intended lifecycle is:

```text
natural-language project brief
  -> externally orchestrated task or local authoring session
  -> generated OpenAPI/UWS artifacts
  -> deterministic validation and review
  -> approved handoff package
  -> trusted executor handoff
```

`openudon synthesize`, `openudon build`, `openudon promote`, `openudon assess`, `cmd/icot`, and eval commands
generate, compile, validate, and report on artifacts. They do not execute production workflows.

`openudon run` is separate. It validates the handoff manifest, stored and current quality, approval
JSON, package digest, and tier before writing a non-secret `openudon.executor-run.v1` run config and
`openudon.run-evidence.v1` evidence. Dry runs stage the reviewed package into a fresh workdir and
verify the staged digest without invoking the executor or requiring credential values. Non-dry runs
perform the same staging and digest check before calling the configured executor.
The runner is also available directly as `go run ./cmd/udon-runner --config <run-config.json>`.
`OPENUDON_EXECUTOR` accepts either an absolute path to an executable file or `docker://<image>`.
The outer `OPENUDON_UDON_RUNNER` override must be an absolute path to an executable file.
When that outer override is used, OpenUdon evidence marks its staged package as `stage_kind:
preflight`; the external runner owns any final executor-visible staging and must fail closed on its
own config checks.

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

# Ask every question and let you confirm defaults. This is the default mode.
go run ./cmd/icot --example ./examples/<name> --prompt-mode full

# Print defaulted questions and accept their defaults automatically.
go run ./cmd/icot --example ./examples/<name> --prompt-mode normal

# Ask only when iCoT has no default or answer.
go run ./cmd/icot --example ./examples/<name> --prompt-mode fast

# Seed from an existing example.
go run ./cmd/icot --from-example ./examples/eval/runtime-only-render --example ./examples/<name>

# Use YAML or JSON session/legacy answers.
go run ./cmd/icot --answers ./answers.yaml --example ./examples/<name>

# Rebuild project.md from workflows/intent.hcl.
go run ./cmd/icot reconcile --example ./examples/<name>

# Check authoring quality, intent parseability, and advisory drift.
go run ./cmd/icot lint --example ./examples/<name>

# Noninteractive agent/JSON report surface.
go run ./cmd/icot --example ./examples/<name> --agent --json

# Provider-free iCoT reliability scorecard.
go run ./cmd/icot scorecard --root ./examples/eval --out eval/runs/icot-scorecard-local

# Include curated natural-language authoring variants.
go run ./cmd/icot scorecard --root ./examples/eval --include-variants --out eval/runs/icot-authoring-scorecard-local

# Verify scorecard report JSON plus digest sidecar.
go run ./cmd/icot report verify --file eval/runs/icot-authoring-scorecard-local/scorecard.json

# Validate variant metadata and reference-seeded clear slots.
go run ./cmd/icot variants validate --root ./examples/eval

# Check provider-family coverage across variant classes.
go run ./cmd/icot variants coverage --root ./examples/eval

# Optional real-LLM natural-language authoring evidence.
go run ./cmd/icot authoring-eval --root ./examples/eval --include-variants --provider copilot-api --model gpt-5.4-mini --out eval/runs/icot-authoring-eval-local

# Optional/manual verification for real-LLM authoring evidence.
go run ./cmd/icot report verify --file eval/runs/icot-authoring-eval-local/authoring-eval.json

# Bounded deterministic repair for mappings, outputs, and depends_on.
go run ./cmd/icot repair --example ./examples/<name> --dry-run --json

# Replay eval references through iCoT and save ignored transcripts.
go run ./cmd/icot replay-eval --root ./examples/eval --provider copilot-api --model gpt-5.4-mini
```

iCoT autosaves incomplete local sessions under `<example>/.icot/session.yaml` and resumes by
default. Successful saves delete the autosave. Transcripts are written under
`<example>/.icot/transcript.json` unless `--no-transcript` is used. These local files are ignored by
git.

`--prompt-mode full` is the default when the flag is omitted; it prints every question and waits for
you to confirm or replace defaults. `--prompt-mode normal` prints every question and automatically
accepts defaults. `--prompt-mode fast` skips defaulted questions entirely, suppresses catalog/status
chatter plus review-only fallback and assumption text, and asks only for required values without a
safe default, such as the initial workflow goal.

When LLM extraction is enabled, iCoT also runs a bounded pre-final flow review before showing the
current draft. That review is advisory: it looks for cross-step data-flow mistakes such as a report
email step not consuming the report content, and surfaces findings as warnings without rewriting the
draft.

`--review-repair` turns selected warnings into a bounded repair loop. It can apply narrow wiring
repairs or add a local `fnct` transform/report/render step when the goal clearly asks for produced
content and one known producer step can feed it. It rejects operation, source, credential, and
side-effect-scope mutations.

For SaaS briefs, iCoT first checks the local and sibling `apitools` provider catalog. When cached
OpenAPI, Google Discovery, or reviewed advisory OpenAPI overlay artifacts are available, it can use a
bounded LLM catalog plan to select validated local artifacts and seed rough provider steps. Concrete
operation IDs and request mappings still come from local operation metadata. After operation
selection, iCoT gives the LLM a focused chance to fill required request fields from structured
operation details before asking the operator for any unresolved mappings.

When a goal explicitly asks to stop and report a missing or ambiguous provider/API/source capability,
and no usable API source or operation is available, iCoT can produce a local `render_capability_gap`
`fnct` workflow with `provider` and `action` inputs instead of inventing an API execution plan.

Side-effect scope in iCoT:

- `read-only`: generate and validate artifacts only.
- `sandbox-only`: sandbox proof runs require `approved_for_sandbox`, approved bindings, and a
  trusted runner.
- `after-approval`: sandbox and production execution require the full OpenUdon review approval path.

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

The command reads `project.md`, discovers or imports API/event source documents under `openapi/`,
`google-discovery/`, `aws-smithy/`, `asyncapi/`, `graphql/`, `openrpc/`, `grpc-protobuf/`, or
`odata/`, writes `workflows/intent.hcl` when needed, and generates equivalent public UWS HCL/YAML workflow
artifacts:

```text
expected/plan.json
expected/plan.md
expected/discovery.json
expected/data.hcl
expected/refinement.json
expected/refinement.md
expected/review.md
expected/review-handoff.json
expected/quality.json
expected/quality.md
```

`expected/data.hcl` is for reviewed runtime inputs and env references, not
plaintext secrets. Udon resolves markers such as
`client_secret = "ENVIRONMENT:GOOGLE_CLIENT_SECRET"` from the execution
environment.

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

## Provider Catalog

OpenUdon can inspect first-class provider metadata from `github.com/OpenUdon/apitools/catalog`
before falling back to public search. Catalog data is advisory: local API source files and explicit
source inputs remain authoritative for generated packages.

```bash
# List known first-class providers and auth/security status.
go run ./cmd/openudon catalog list

# Inspect a provider's official OpenAPI, Discovery, Smithy, docs, and security-overlay metadata.
go run ./cmd/openudon catalog inspect github
go run ./cmd/openudon catalog advisory gmail

# Import a provider-owned OpenAPI document directly into an example.
go run ./cmd/openudon catalog import-openapi \
  --provider stripe \
  --example ./examples/<name> \
  --name stripe
```

`import-openapi` writes only actual OpenAPI references into `examples/<name>/openapi/`. Catalog
materialization and iCoT artifact migration may stage Google Discovery under `google-discovery/`,
AWS Smithy JSON under `aws-smithy/`, AsyncAPI source documents under `asyncapi/`, GraphQL under
`graphql/`, OpenRPC under `openrpc/`, gRPC/protobuf under `grpc-protobuf/`, and OData under
`odata/`. Dropbox Stone, Postman Collection, RAML, API Blueprint, and
human-docs entries remain advisory until lowered or reviewed separately.

## Quality And Repair Loop

The pipeline is validation-first:

1. Run `synthesize` for a new or substantially changed `project.md`.
2. If it fails, read `expected/refinement.json` and `expected/quality.json`.
3. Repair the earliest failing stage.
4. For `openapi.*`, add a valid local OpenAPI file or explicit OpenAPI URL.
5. For `intent.*`, edit `project.md` or `workflows/intent.hcl`, then rerun `build`.
6. For `workflow.*`, prefer improving intent and rerunning `build`; use `promote` and `assess` for
   narrow workflow repairs.
7. For `uws.*`, `review.*`, `review_handoff.*`, or `artifacts.*`, repair the generated artifact
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
make eval-seed-build
make release-saas-check
make release-eval
```

`make release-saas-check` is the provider-free local SaaS release gate. It runs deterministic checks,
the eval seed/build matrix, `icot-variants-validate`, `icot-authoring-scorecard`, UWS validation,
doc-memory validation, n8n bridge validation, strict MkDocs, selected strict fixture lint, and
trusted-runner dry-run demos without live provider credentials or live provider execution. `icot
scorecard --include-variants` is deterministic reference/variant package evidence; use `icot
authoring-eval` separately for optional real LLM natural-language authoring evidence.

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

Use `--dry-run` to validate all gates, stage the package, verify the staged digest, and write
run evidence without invoking the executor.

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
- `run-evidence.json` records gate outcomes, package paths, staged paths, stage kind, executor
  status, and credential binding names only; it must not contain credential values.
- Approval JSON and saved run configs from before the OpenUdon package rename should be regenerated
  so scope, version, and package digest fields match the current artifact set.

## Agent Workflow

OpenUdon issues may be run through externally orchestrated Codex sessions. Agents should follow this policy:

- Use UWS as the workflow interchange format.
- Use reviewed API/event source documents for HTTP method, path, channel, message, schema, server,
  and security details.
- Use `openudon catalog inspect` or `openudon catalog import-openapi` when a first-class
  provider-owned OpenAPI source is available, and use first-class materialization for Google
  Discovery, AWS Smithy, AsyncAPI, GraphQL, OpenRPC, gRPC/protobuf, or OData sources when supported.
- Use extension-owned UWS operations for non-HTTP runtimes such as SMTP, command execution, SSH,
  SQL, or LLM calls.
- Use `../uws` for public schema/model validation.
- Use `openudon run` to hand approved UWS/API-source packages to a trusted executor such as udon.
- Do not execute production side effects directly from an agent session.
- If execution is requested, produce or update the approved artifact and document the trusted runner
  command.

Expected artifact locations:

```text
examples/<name>/project.md
examples/<name>/openapi/
examples/<name>/google-discovery/
examples/<name>/aws-smithy/
examples/<name>/discovery/
examples/<name>/asyncapi/
examples/<name>/graphql/
examples/<name>/openrpc/
examples/<name>/grpc-protobuf/
examples/<name>/odata/
examples/<name>/workflows/intent.hcl
examples/<name>/workflows/workflow.hcl
examples/<name>/workflows/workflow.uws.yaml
examples/<name>/expected/plan.json
examples/<name>/expected/plan.md
examples/<name>/expected/discovery.json
examples/<name>/expected/data.hcl
examples/<name>/expected/refinement.json
examples/<name>/expected/refinement.md
examples/<name>/expected/review.md
examples/<name>/expected/review-handoff.json
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
- [iCoT](docs/icot.md)
- [iCoT corpus and provider roadmap](docs/icot-corpus-and-provider-roadmap.md)
- [Intent contract](docs/intent.md)
- [Data flow](docs/data-flow.md)
- [Enterprise authoring/execution boundary](docs/enterprise-authoring-execution.md)
- [Safety](docs/safety.md)
- [Eval gallery](docs/eval-gallery.md)
- [SaaS operator release path](docs/saas-operator-release.md)
- [Terraform/API source conversion](docs/terraform-openapi-conversion.md)
- [Release stewardship](docs/release-stewardship.md)
- [Release note template](docs/release-note-template.md)
- [Contributing](CONTRIBUTING.md)
- [License](LICENSE)
