# OpenUdon

[![test](https://github.com/OpenUdon/openudon/actions/workflows/test.yml/badge.svg)](https://github.com/OpenUdon/openudon/actions/workflows/test.yml)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)

OpenUdon is the public UWS workflow authoring, review, package, and executor-handoff tool. It can run
directly or under optional external orchestration, and it hands approved packages to a
trusted executor boundary such as the `udon` runtime.

OpenUdon is a CLI and artifact tool. Kinet owns interactive authoring, chat and
browser UI. Use `openudon authoring draft` for explicit seeded/local drafts,
`authoring browser-plan` for inert capture planning, and
[`authoring registration-draft`](docs/registration-draft.md) for pure typed
registration definitions. Expert evaluation commands remain available.
`browser-capture` supervises an isolated Browsertools worker through exact
issued decisions; `browser-author` and `package` handle reviewed receipts and
native preparation/promotion. Browsertools owns acquisition and safety;
Udon and Browserdriver own separately approved runtime replay. The iCoT terminal,
UI/control and release binary have been removed; historical package artifacts
and versioned report readers remain supported. M95 final qualification and
consumer adoption are recorded separately in each owner's ledger.
See Browsertools' [canonical OpenUdon integration
reference](https://github.com/OpenUdon/browsertools/blob/main/docs/openudon-integration.md).

It owns project templates, optional workflow orchestration policy, example artifacts, deterministic
validation, review handoff evidence, package digests, credential policy, and trusted-runner glue.
Public workflow semantics belong in `github.com/OpenUdon/uws`; API/event source metadata discovery,
import, materialization, search, and indexing belong in `github.com/OpenUdon/apitools`; desired-state
conversion, planning, reconciliation, and audit behavior belong in `github.com/OpenUdon/ramen`.
OpenUdon uses shared `github.com/OpenUdon/evidence/...` primitives for neutral digest, artifact,
diagnostic, redaction, and approval evidence where the records are product-independent. Current
shared use routes review/package hashing through `evidence/digest` and package artifact path-safety
through `evidence/artifact` with OpenUdon labels for stable CLI wording.
OpenUdon-specific approval JSON, review handoff, package digest policy, run evidence, tier rules,
and trusted-runner behavior remain OpenUdon-owned; the tier-plus-digest approval model does not map
onto `evidence/approval`.
OpenUdon can stage OpenAPI, Google Discovery, AWS Smithy, AsyncAPI, GraphQL, OpenRPC,
gRPC/protobuf, and OData source documents as first-class UWS source descriptions when the trusted
executor supports them.

An operator may also add reviewable content-provenance declarations to
`workflows/intent.hcl`. Newly generated workflows declare UWS 1.11.0; existing
packages retain their declared UWS version. The optional `contentTrust` registry
requires UWS 1.9.1 or later. These declarations
are metadata for advisory analysis and human/AI review, not execution approval
or runtime policy. During assessment, declared packages are analyzed with the
UWS analyzer and contained browser profiles use the Browsertools resolver.
Findings appear as non-failing warnings in quality and review evidence; they do
not alter ordinary validation, trusted-runner authorization, or execution. See
[the intent content-trust contract](docs/intent.md#content-trust).

## v0.2 Security Migration

The source tree implements the unreleased v0.2 compatibility boundary for the deterministic package lifecycle
(`validate`, `build`, `promote`, and `assess`), digest-bound approval and
trusted handoff (`approval-template` and `run`), run-evidence
verification/archive/signatures, and the v2 handoff artifacts. Existing v1
run configs and handoffs must be rebuilt before execution; v1 run evidence is
read-only inspectable and cannot be archived as v2 evidence. Neutral expert authoring,
LLM/provider behavior, eval/catalog/smoke helpers, and exact generated prose
remain experimental before v1. OpenUdon does not yet expose a supported
Go-library API.

See [SUPPORT.md](SUPPORT.md) and the
[v0.2 compatibility contract](docs/compatibility.md) for the exact boundary.

The most recent published binaries remain v0.1.0 until v0.2 release evidence
is complete; this change does not publish or tag v0.2.0.

## Quick Start

Install the main CLI:

```bash
go install github.com/OpenUdon/openudon/cmd/openudon@v0.1.0
openudon version --json
```

For current source, build the two supported executables:

```bash
GOWORK=off go build -o /tmp/openudon ./cmd/openudon
GOWORK=off go build -o /tmp/udon-runner ./cmd/udon-runner
```

The published v0.1.0 archives are historical and contain `openudon`, `icot`
and `udon-runner`; verify their `SHA256SUMS`. Current release builds contain
only `openudon` and `udon-runner`. Source retirement does not rewrite or tag a
published release. Use binaries built from this source for the commands below.

From a source checkout, the credential-free release path authors a local
function-only project, builds and assesses it, then stages an approved sandbox
dry run without invoking an executor:

```bash
DEMO_ROOT=.openudon-run/v0.1.0-quick-start
openudon authoring draft \
  --from-example ./examples/eval/runtime-only-render \
  --example "$DEMO_ROOT/package" \
  --no-llm \
  --yes
openudon build --example "$DEMO_ROOT/package"
openudon assess --example "$DEMO_ROOT/package"
openudon approval-template \
  --example "$DEMO_ROOT/package" \
  --state approved_for_sandbox \
  --reviewer "Local Reviewer" \
  > "$DEMO_ROOT/approval.json"
openudon run \
  --example "$DEMO_ROOT/package" \
  --tier sandbox \
  --approval "$DEMO_ROOT/approval.json" \
  --workdir "$DEMO_ROOT/run" \
  --dry-run
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
make browser-integration-check
make browser-scenario-loopback
make browser-scenario-journey
make eval-seed-build
make release-saas-check
git diff --check
```

Execute through `openudon run` and the portable run-config handoff. Configure the final executor
with `OPENUDON_EXECUTOR` as either an absolute binary path or `docker://<image>`.
Reviewed browser workflows additionally use
`--browser-driver /absolute/path/browserdriver`; OpenUdon derives the protocol,
symbolic credential/session environment names, and exact browser approvals
from one immutable, manifest-digest-validated package byte snapshot and binds
them into the v2 config and evidence. Staging rehashes the current files and
rejects later drift. For a Docker
executor, that host executable is mounted read-only at
`/openudon/browser-driver` and Udon receives the container path; the image does
not need to contain the driver at the host path. Docker forwards only declared
credentials and sessions with `-e`; approved driver environment names resolve
from container-owned defaults and host desktop/socket requirements are
rejected.

## Layout

- `cmd/openudon`: local CLI for checks, synthesis, assessment, eval, readiness, approval templates,
  and trusted execution.
- `cmd/udon-runner`: portable external executor handoff wrapper.
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

`openudon synthesize`, `openudon build`, `openudon promote`, `openudon assess`, `openudon package prepare|promote|inspect|recover`, neutral authoring, and eval commands
generate, compile, validate, and report on artifacts. They do not execute production workflows.

`openudon run` is separate. It validates the handoff manifest, stored and current quality, approval
JSON, package digest, and tier before writing a non-secret `openudon.executor-run.v2` run config and
`openudon.run-evidence.v2` evidence in a unique per-run directory. Dry runs stage the reviewed package into a fresh workdir and
verify the staged digest without invoking the executor or requiring credential values. Non-dry runs
perform the same staging and digest check before calling the configured executor.
The runner is also available directly with the digest and approval pinned by
the parent process:

```bash
go run ./cmd/udon-runner \
  --config <run-config.json> \
  --config-sha256 <sha256> \
  --approval <approval.json>
```

It revalidates current quality, handoff, package, approval, tier, and exact
canonical config bytes before execution. v1 configs are rejected.
`OPENUDON_EXECUTOR` accepts either an absolute path to an executable file or `docker://<image>`.
The outer `OPENUDON_UDON_RUNNER` override must be an absolute path to an executable file.
When that outer override is used, OpenUdon evidence marks its staged package as `stage_kind:
preflight`; the external runner owns any final executor-visible staging and must fail closed on its
own config checks.

## Neutral authoring

Kinet owns the interactive authoring loop. OpenUdon retains explicit seeded/local
commands and their native source validation, policy and atomic writer. No terminal
interview, application HTTP/control service or embedded UI remains.

```bash
openudon authoring draft --from-example ./examples/eval/runtime-only-render \
  --example .openudon-run/draft-example --prompt-mode fast --no-llm --print
openudon authoring draft --from-example ./examples/eval/runtime-only-render \
  --example .openudon-run/draft-example --prompt-mode fast --no-llm --yes
openudon authoring lint --example .openudon-run/draft-example
openudon authoring repair --example .openudon-run/draft-example --dry-run --json
openudon authoring variants validate --root examples/eval
openudon authoring variants coverage --root examples/eval
openudon authoring scorecard --root examples/eval --include-variants --out /tmp/NEW-scorecard
openudon authoring report verify --file /tmp/NEW-scorecard/scorecard.json
```

Print and partial-frontier responses make no state writes; complete publication
requires explicit `--yes`. Existing answers/session/transcript bytes and report
schemas remain readable; no hidden defaults grant runtime approval. Optional real
model evaluation/replay still requires explicit invocation and provider authority.
Use [the retirement/migration guide](docs/authoring-retirement.md) for all retained
entry mappings, pure registration definitions and versioned qualification.

## Non-interactive step authoring

External orchestrators can use the versioned, non-prompting step-authoring CLI. Candidate discovery
is local and offline; it consumes APItools metadata but never fetches source URLs or calls API
operations. `step source add` accepts exact local paths and content digests, validates documents
through APItools, and atomically writes the source files plus package provenance manifest. Kinet
calls it only after the user confirms the exact source proposal. Bind requires an explicit selected
operation, mappings, and symbolic credential names; it does not approve or execute a workflow.

```bash
openudon step candidates --example docs/examples/step-authoring/v1/example \
  --request docs/examples/step-authoring/v1/requests/step-candidates.json
openudon step check --example docs/examples/step-authoring/v1/example \
  --request docs/examples/step-authoring/v1/requests/step-check-runnable.json
openudon step bind --example docs/examples/step-authoring/v1/bind-example \
  --request docs/examples/step-authoring/v1/requests/step-bind.json
openudon flow-review --example docs/examples/step-authoring/v1/example \
  --request docs/examples/step-authoring/v1/requests/flow-review.json
```

Unresolved contracts use the additive [pending-step command](docs/step-pending.md).
Approval and all execution paths refuse them; bind resolves an exact reviewed
contract against a source and explicit mappings.

The [v1 contract](docs/step-authoring-contract-v1.md), [JSON schema](docs/schemas/openudon.step-authoring.v1.schema.json),
and [request/result fixtures](docs/examples/step-authoring/v1/) define the wire format and limits.
Kinet can validate the fixtures independently without importing OpenUdon Go packages.

iCoT maps broad requests into one active workflow boundary plus unnumbered candidate workflows. It
shows every dependency-ready decision as one frontier round before collecting answers, and has no
fixed question ceiling. Candidate workflows receive a deferral reason and promotion trigger but no
sources, operations, mappings, or implementation steps.

Historical `.icot/session.yaml`, `.icot/transcript.json` and review metadata
remain unchanged and readable. Current draft publication does not run an
interactive autosave loop. For browser acquisition use Kinet over
[public capture](docs/browser-capture-protocol.md) and
[reviewed package authoring](docs/browser-package-handoff.md); credentials and
sessions remain private to Browsertools/Udon. Inert plans and pure draft commands
never start a browser. Native package preparation, promotion, registration
attestation and trusted execution require their own exact decisions.

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
make browser-transaction-qualification
make release-evidence
make release-eval
```

`make release-saas-check` is the provider-free local SaaS release gate. It runs deterministic checks,
the required sandboxed `browser-capture-check`, the browser-free
[browser integration evaluation](docs/browser-integration-eval.md), the real
network-free [browser scenario loopback and journey suites](docs/browser-scenario-eval.md), the eval seed/build matrix,
`authoring-variants-validate`, `authoring-scorecard`, UWS validation,
doc-memory validation, n8n bridge validation, strict MkDocs, selected strict fixture lint, and
trusted-runner dry-run demos without live provider credentials or live provider execution. `openudon authoring
scorecard --include-variants` is deterministic reference/variant package evidence; use `openudon authoring
authoring-eval` separately for optional real LLM natural-language authoring evidence.

`make browser-capture-check` exercises both actual public capture-to-package
journeys on synthetic loopback services with sandboxed Chromium: login/TOTP and
typed registration, including separate verification refusal/approval. Kinet's
UI conformance belongs to Kinet; no removed UI selector silently stands in for
this native boundary. Full current native qualification remains three fresh passes.

`make release-eval` uses `OPENUDON_LLM_PROVIDER` and `OPENUDON_LLM_MODEL`, defaulting to `copilot-api` and
`gpt-5.4-mini`, and requires the current eval corpus size as the minimum brief count.

`make release-evidence` runs the local udon smoke, archives and verifies
`run-evidence.json` plus async/executor report sidecars, drafts local release
notes, and writes compact summaries under ignored `.openudon-run/release-evidence/`.
It does not tag, publish, commit artifacts, or contact live providers.

`make browser-integration-check` runs and then verifies the value-free,
digest-bound authoring-to-handoff matrix across OpenUdon, Browsertools, UWS,
Udon, and Browserdriver. Its default path is offline/provider-free and does not
launch a browser; installed-engine and headed-authentication checks require
separate loopback-only CLI opt-ins.
The browser integration matrix selects the published UWS 1.11 and Browser
1.8/1.9 stack and retains verification of historical v1 reports. The separate
real-browser scenario suites keep their earlier exact compatibility lock and
qualification history.

`make browser-scenario-loopback` runs and verifies the required 23-case real
Browsertools v2 to Udon/Browserdriver v3 release matrix. It requires installed
pinned Chromium dependencies and a display (use `xvfb-run -a` on headless
Linux). `make browser-scenario-journey` runs the required eight-case headless
local read/write matrix through guided authoring, UWS 1.8, Udon v3, and
Browserdriver v3. `make browser-scenario-public` is an explicit-network,
informational four-site canary; it is never part of default tests.

`xvfb-run -a make browser-transaction-qualification` runs the complete
cross-package BAP+BCP replay and BRP authoring-to-runtime qualification against
only embedded loopback fixtures, after its adversarial matrix, and writes a
canonical value-free v2 report under ignored `eval/runs/`. The BRP case keeps
authoring GET/HEAD-only, then uses the exact private attestation and submit
approval to prove one Browserdriver-v4 POST through Udon report v3 with no
named session. It requires clean exact sibling revisions and sandboxed
Chromium. Xvfb supplies only the display;
see [Browser-Profile Authoring Transactions](docs/browser-profile-transactions.md#qualification-evidence)
for the host sandbox prerequisite and independent report verification.

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

The shared `github.com/OpenUdon/evidence/approval` package supplies neutral approval evidence
primitives for cross-product reuse. The `openudon.approval.v1` JSON shape above remains the
OpenUdon trusted-runner contract.

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
prompts, commands, examples, or workflow artifacts. Gemini sends its key only
through `x-goog-api-key`; provider response bodies are bounded to 8 MiB.

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
- [Release stewardship](docs/release-stewardship.md)
- [Release note template](docs/release-note-template.md)
- [v0.2 compatibility contract](docs/compatibility.md)
- [Support policy](SUPPORT.md)
- [Security policy](SECURITY.md)
- [Contributing](CONTRIBUTING.md)
- [License](LICENSE)

Use `make fast` for routine browser-free iterations and one deliberately selected
`make smoke` for an affected synthetic capture-to-runtime flow. `make qualify`
(and `make browser-system-current-check`) selects the fresh complete current
native gate. See [browser qualification](docs/browser-system-eval.md) for exact
source/runtime inputs, cost selection, failure retention and verification.
Consumer adoption and real target authority remain separate.

### Retained expert authoring commands

`openudon authoring --help` lists closed draft/browser-plan/registration-draft,
lint, reconcile, repair, report verification, variants, scorecard and optional
model evaluation. Historical artifact/report labels are retained; they do not
make a removed transport callable. Pure previews, including unresolved contracts:
[workflow simulation](docs/simulation.md).
