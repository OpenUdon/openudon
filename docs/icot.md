# iCoT v2

iCoT is OpenUdon's adaptive guided-authoring CLI. It turns a broad workflow
request into one reviewed active boundary, a dependency-aware decision graph,
and either a complete `workflows/intent.hcl` or an explicitly incomplete draft.
It does not execute workflows.

```bash
go run ./cmd/icot --example ./examples/<name>
```

## Inputs And Modes

```bash
# Preview only; writes no deliverables, source copies, transcript, or autosave.
go run ./cmd/icot --example ./examples/<name> --print

# Disable optional LLM extraction while retaining the adaptive interview.
go run ./cmd/icot --example ./examples/<name> --no-llm

# Load a v2 session. Add --yes for explicit noninteractive proposal approval.
go run ./cmd/icot --example ./examples/<name> \
  --answers ./session.yaml --yes

# Inspect an existing example before asking questions.
go run ./cmd/icot --example ./examples/<name> \
  --from-example ./examples/eval/weather-toronto --yes

# Declare reviewed API documents or bounded roots. Flags are repeatable.
go run ./cmd/icot --example ./examples/<name> \
  --api-source graphql:catalog=./schema.graphql \
  --openapi weather=./openapi/weather.yaml \
  --source-root ./provider-metadata

# Control the bounded remote metadata lookup.
go run ./cmd/icot --example ./examples/<name> --network never
```

`--api-source` uses `KIND:ID=PATH`; `--openapi ID=PATH` is shorthand for an
OpenAPI source. `--network` accepts `never`, `ask`, or `allow`. Interactive runs
default to `ask`. Agent mode is effectively `never` unless `allow` is explicit.

Prompt modes preserve the public v1 names with v2 behavior:

- `full` shows and asks every question in the current frontier.
- `normal` shows the entire frontier and visibly accepts safe defaults.
- `fast` silently accepts safe defaults but shows missing, low-confidence,
  conflicting, or forced decisions.

Every round is displayed in full before answers are collected. All answers in
the round are applied together, followed by one normalization and autosave.
There is no fixed question ceiling. The interview ends on completion,
cancellation, approved draft deferral, or three consecutive no-progress rounds.
Final proposal approval is forced in every prompt mode; `--yes` is the explicit
noninteractive equivalent.

## Workflow Boundary And Frontier

iCoT first confirms one active outcome with its actor, trigger, observable
success evidence, non-goals, and side-effect/approval posture. If a request
contains multiple workflows, the operator must select the active workflow even
in fast mode. Other workflows remain unnumbered candidates with a deferral
reason and promotion trigger; iCoT does not assign them sources, operations,
mappings, or steps.

The decision graph adds only applicable nodes. Its main dependency order is:

1. active boundary and outcome;
2. actor/trigger and expected result;
3. provider capabilities and workflow steps;
4. source selection;
5. operation selection;
6. mappings/data flow, credentials, and side-effect posture;
7. outputs, fallback behavior, and verification;
8. complete proposal approval.

Source, operation, mapping, and output leaves may be deferred. Each deferral
must identify an owner, impact, unblock condition, and suggested next action.
The workflow boundary and side-effect/approval posture cannot be deferred.

The durable session uses `openudon.icot-session.v2` with generic
`authoring.interview.v1` graph state and one unified evidence ledger. v1 inputs
are rejected rather than decoded compatibly. See
[iCoT v2 Session Files](icot-session-schema.md).

## Source Discovery

Before questioning, iCoT inspects existing example sources, explicit documents,
and explicit roots. Local discovery recognizes and validates:

- OpenAPI/Swagger;
- Google Discovery;
- AWS Smithy;
- AsyncAPI;
- GraphQL;
- OpenRPC;
- gRPC/protobuf;
- OData.

Directory names are hints, never proof. Discovery rejects symlinks and
non-regular paths, deduplicates identical content by SHA-256, and requires an
explicit kind for ambiguous JSON or XML. Default bounds are 10,000 visited
entries, 100 accepted candidates, and 20 MiB per file. Reaching a bound is a
visible blocker with narrowing guidance.

Ambiguity or a traversal/candidate bound makes discovery incomplete in every
mode. Interactive authoring stops before using any partial candidate set;
complete-session and agent paths expose the same blocker and narrowing
guidance.

Local and remote discovery are separate. After local evidence is exhausted,
`--network ask` requires approval. The remote lookup consults only curated
apitools catalog references and one APIs.guru list request—never a general web
crawler. It has an eight-second total deadline, returns at most three metadata
candidates, applies unsafe-host protections, and does not copy a document.
Denial, timeout, unsafe results, or an empty lookup becomes a deferrable source
blocker with provider/source hints.

## Proposal And File Lifecycle

Before writing, iCoT shows one proposal containing the active boundary,
candidate workflows, workflow steps, source origins/digests/targets, mappings,
safety policy, deferrals, and exact file actions.

During the interview, only resumable `.icot/` state may be autosaved. Catalog or
external sources and deliverable files are not copied before proposal approval.

A complete approval atomically writes:

```text
project.md
<selected source documents and reviewed security sidecars>
workflows/intent.hcl
```

An approved incomplete technical draft atomically writes:

```text
project.md
<selected confirmed source documents>
workflows/intent.draft.hcl
.icot/session.yaml
.icot/readiness.json
```

It never creates `workflows/intent.hcl`. Resuming, completing, and approving the
draft promotes it atomically and removes obsolete generated draft/readiness
files. Source targets reuse identical content, reject differing content unless
`--force` is supplied, and participate in the same backup/rollback transaction
as project and intent files.

`project.md` may include a deterministic, non-executable `Candidate Workflows`
section. Reconcile preserves it, lint validates its shape, and build ignores it.

`cancel` and `--print` leave no deliverable or copied source files. Transcripts
use `openudon.icot-transcript.v2`; see
[iCoT v2 Transcript Format](icot-transcript.md).

## Agent And Structured Reports

`--agent` never prompts and never writes deliverables. It returns the complete
frontier, candidate workflows, validated/rejected/ambiguous source evidence,
remote blocker/candidates when applicable, readiness blockers, and proposed file
actions. A complete session still returns `proposal_approval_required` so an
interactive run or explicit `--yes` can authorize the transaction.

```bash
go run ./cmd/icot --example ./examples/<name> --agent --json
go run ./cmd/icot lint --example ./examples/<name> --json
```

Current structured versions are:

- `openudon.icot-author-report.v2`;
- `openudon.icot-lint-report.v2`;
- `openudon.icot-repair-report.v2`;
- `openudon.icot-scorecard.v2`;
- `openudon.icot-authoring-eval.v2`;
- `openudon.icot-variants-validation.v2`;
- `openudon.icot-variants-coverage.v2`;
- `openudon.icot-authoring-variants.v2`;
- `openudon.icot-replay.v2`.

Support commands retain their names:

```bash
go run ./cmd/icot reconcile --example ./examples/<name>
go run ./cmd/icot lint --example ./examples/<name>
go run ./cmd/icot repair --example ./examples/<name> --dry-run --json
go run ./cmd/icot replay-eval --root examples/eval --prompt-mode fast
go run ./cmd/icot variants validate --root examples/eval
go run ./cmd/icot variants coverage --root examples/eval
go run ./cmd/icot scorecard --root examples/eval --out eval/runs/icot-scorecard-local
go run ./cmd/icot authoring-eval --root examples/eval --provider copilot-api \
  --model gpt-5.4-mini --out eval/runs/icot-authoring-eval-local
go run ./cmd/icot report verify --file eval/runs/icot-scorecard-local/scorecard.json
```

Scorecard verification checks version, counters, variant expectations,
retention/share-safety metadata, and the adjacent SHA-256 sidecar. Provider-free
scorecards remain release evidence; real-model authoring-eval output remains
local, ephemeral, and subject to credential scanning.

## Optional LLM Review

LLM extraction may propose a draft and a bounded pre-final flow review, but
operation IDs, request fields, response paths, and credential schemes must come
from inspected evidence. Inline secrets, review bypasses, unconfirmed mutations,
invented operations, and unsafe placeholder promotion remain blockers.

`--review-repair` may apply at most two narrow mapping, output-source,
dependency, or clearly local transform repairs. It cannot silently change a
source, operation, credential binding, side-effect posture, or active boundary.
All public rationale is concise evidence; hidden model chain-of-thought is never
stored or requested.
