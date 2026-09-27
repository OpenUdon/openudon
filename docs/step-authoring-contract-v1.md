# OpenUdon step-authoring command contract v1

**M87 release contract.** This document and its fixtures define the versioned
OpenUdon CLI surface delivered by M87. UWS owns pending-step semantics; the
recorded local C07.2 mapping check does not publish or modify UWS.

This additive CLI contract is owned by OpenUdon. It gives Kinet a stable,
versioned subprocess boundary for `step candidates`, `step bind`, `step check`,
and `flow-review`. It does not add UWS execution behavior. Kinet remains the
workflow planner and owns user confirmation, its ledger, receipts, and repair
orchestration. UWS owns workflow semantics; APItools owns generic operation
summaries, effect evidence, and ranking.

## Invocation and transport

```text
openudon step candidates --example DIR --request FILE|-
openudon step bind       --example DIR --request FILE|-
openudon step check      --example DIR --request FILE|-
openudon flow-review     --example DIR --request FILE|-
```

`--request -` reads one request from standard input. The request is strict
UTF-8 JSON, at most 256 KiB, and rejects unknown fields and trailing JSON
values. Commands never prompt. Each valid invocation writes exactly one JSON
result followed by a newline to stdout; progress, diagnostics, and errors do
not add text to stdout. Once a command route is known, malformed JSON,
unsupported versions, and schema-invalid requests still produce a v1 `failed`
result with a stable diagnostic and the command's normal nonzero exit code.
Invalid CLI usage before a command route is known writes usage to stderr and no
JSON result. Result messages omit absolute filesystem paths and do not echo
request text. Diagnostics use stable codes and bounded, sanitized messages.
`--help` remains ordinary command help.

Every request and result has `version: "openudon.step-authoring.v1"`, a
`kind` (`request` or `result`), and the exact `command` name. The published
schema validates both directions. A result has `status`, `diagnostics`, and a
command-specific `result` only when one is available. Status describes command
completion, not whether a checked step is compatible:

| Result status | Meaning |
|---|---|
| `completed` | The requested operation ran; inspect its result assessment and diagnostics. |
| `needs_input` | A human decision or missing declaration is required. No write occurred. |
| `blocked` | Required source/evidence/capability is unavailable. No write occurred. |
| `conflict` | An expected intent or source revision is stale. No write occurred. |
| `failed` | The command could not produce a trustworthy result. |

Exit codes are stable: `0` completed; `1` operational or explicitly requested
model-review failure; `2` invalid CLI usage, JSON, schema, or version; `3`
stale revision or write conflict; `4` needs-input or blocked. A completed
`step check` may report an incompatible or indeterminate assessment and still
exit `0`; consumers use the structured assessment rather than interpreting
exit success as approval.

The diagnostic code and severity are machine-readable. Diagnostic prose is
advisory, bounded, and safe to display. It must not contain credential values,
raw source text, prompt text, or absolute local paths. Input, output, and
fixture budgets are part of the schema; implementations may impose lower
operational limits only when they report a stable blocking diagnostic.

Diagnostic codes are stable identifiers, not prose-derived keys. The initial
registry is:

| Code | Meaning |
|---|---|
| `request.cancelled` | Candidate discovery, step check, or step bind was cancelled before completion. |
| `request.invalid_json` | Request bytes are not one strict UTF-8 JSON value. |
| `request.unsupported_version` | Request version is not supported by this binary. |
| `request.invalid` | Request schema or command-specific invariants are invalid. |
| `example.invalid` | The selected example is missing or cannot be safely read. |
| `intent.invalid` | Existing workflow intent is malformed or cannot be checked. |
| `intent.stale` | Expected intent digest does not match the current bytes. |
| `source.unavailable` | A requested source cannot be read or verified. |
| `source.unsupported` | A requested source family or metadata dimension is unavailable. |
| `source.digest_mismatch` | The current source bytes differ from the requested digest. |
| `operation.not_found` | Exact source operation identity does not resolve. |
| `operation.ambiguous` | Operation identity does not select exactly one candidate. |
| `mapping.incomplete` | Required request/output or dependency mappings are missing. |
| `mapping.workflow_inputs` | A selected step mapping references an input not declared by the workflow. |
| `mapping.contract_mismatch` | Source input/output types conflict with the declared step contract. |
| `dependency.cycle` | The selected step references its own output or participates in a dependency cycle. |
| `authentication.unknown` | Authentication evidence is unavailable or incomplete. |
| `effect.unknown` | Effect evidence is insufficient or conflicting. |
| `effect.conflict` | The source-backed operation effect conflicts with the step contract. |
| `write.conflict` | An optimistic or concurrent write check failed before replacement. |
| `write.indeterminate` | The filesystem outcome cannot be reported as definitively committed or rejected. |
| `review.model_unavailable` | Requested model review could not be started. |
| `review.model_failed` | Requested model review failed or was cancelled. |
| `review.cancelled` | Local or model-assisted flow review was cancelled before completion. |

New codes may be added within v1. Consumers branch on status and command, not
on diagnostic prose, and must tolerate unknown codes.

## Shared step contract

A command contract carries an identifier and the shared UWS declaration fields:
`purpose`, `inputs`, `outputs`, and `effect`. Each of `inputs` and
`outputs` is one recursive UWS `ParamSchema` object whose root type is
`object`; named values are its `properties`, and its `required` list names
the required values. A declared property absent from `required` is optional.
Nested properties, array items, and schema-composition members retain the UWS
shape. These four fields are the same declaration shape used by UWS C07.2's
pending-step contract. Input/output entries are declarations, not runtime
values, request mappings, or output expressions. OpenUdon adds
`account_constraints` and `destination_constraints`; these are
operator-reviewed symbolic constraints, not account selection or execution
authorization. The schemas are declarations, not runtime values; UWS
`x-*` extensions must not be used to carry examples, defaults, or credential
material.

The fixture `uws-c07-2-pending-fields-draft.json` checks the shared fields
against the public UWS `ParamSchema` type in OpenUdon's pinned UWS module
`v0.0.0-20260925154821-80ee9bfb24a6`. It verifies recursive JSON round-trip
preservation for standalone consumers. The tagged workspace test
`go test -tags uws_pending_step_c07_2 ./internal/stepauthoringcontract`
decodes the same fixture using `uws1.PendingStep` at local UWS revision
`7f843af78e508fee140b3b43f28a6b73b37a66a8` and compares all four shared
fields losslessly. The owner selected one
`ParamSchema` object per field set on 2026-09-27, matching the cross-package
design sync recorded as Kinet ordering X2. OpenUdon's standalone module pin
intentionally remains unchanged; no UWS publication is part of this release.

The allowed effect values are `read`, `write`, and `unknown`. The contract's
effect is the operator's expected effect class. Candidate metadata reports an
independently evidenced operation effect. Unknown evidence is conservative;
an HTTP method alone never proves `read`, and unknown does not satisfy a
read-only constraint. A match or rank is advisory and does not grant authority.

The `operation_ref` binds one OpenUdon source ID, source kind, source content
digest, source-native selector, and APItools operation key; the protocol's
operation ID is included when the source has one. For OpenAPI, the native
selector is the source JSON pointer (for example
`#/paths/~1projects/get`); other source kinds retain their native selector.
The source ID is a
workspace-local symbolic identity; filesystem paths and source URLs never
appear in the OpenUdon result. The APItools source SHA-256 is emitted with the
`sha256:` prefix used by this contract. Authentication alternatives preserve
OR-of-AND semantics: requirements inside an alternative are conjunctive, and
alternatives are disjunctive. An explicit empty alternative means anonymous
access is allowed; missing or unsupported authentication evidence is reported
as unknown, not anonymous.

## Command behavior

### `step candidates`

The request supplies one shared step contract and optional exact source filters
and a bounded candidate limit. OpenUdon passes the shared purpose/input/output/
effect declaration to APItools' `apitools.operation-candidates/v1` contract,
mapping each root schema's named properties and required list to APItools'
named input/output values. Types, formats, nested properties, items, and
requiredness are preserved. ParamSchema references or composition constructs
or extensions that APItools cannot represent are reported as
indeterminate/unsupported evidence; they are never silently discarded or
fetched. Each candidate returns exact source identity, a
structured consumer summary (description, inputs, outputs, evidence, and gaps),
separate purpose/input/output/effect match evidence and scores, OR-of-AND
authentication alternatives, and the effect class with its source evidence
and reasons. `rank` is the 1-based position in APItools' deterministic result
order; it is advisory, not an independent score or authority. The result also
retains path-free source capability reports so partial or unsupported source
families are visible. No candidate is still a valid completed result with an
empty list and a diagnostic. A requested source or metadata dimension that
cannot be represented is reported explicitly rather than dropped or invented.

Candidate discovery is local and offline: it scans only the recognized
source-family directories under `--example` (`openapi`, `google-discovery`,
the legacy Google Discovery alias `discovery`, `aws-smithy`, `asyncapi`,
`graphql`, `openrpc`, `grpc-protobuf`, and `odata`). Both Google Discovery
directories produce the `google-discovery` source kind. It does not follow
symlinks, accept special files, fetch URLs, or search outside the selected
example. Conventional directory names classify candidate families but APItools
still validates document contents. Supported package security sidecars are
excluded from candidate discovery and source resolution. Without `source_filters`,
discovery accepts at most 32 files and 4096 visited entries; each file is
bounded to 8 MiB and combined source bytes to 32 MiB. APItools is bounded to
100 candidates and 1000 operations. Reaching a bound or finding an unsafe or
ambiguous source blocks the result rather than returning a partial rank as
complete. An exact source filter binds the path-derived source ID to the
requested SHA-256 digest; stale bytes return `conflict`.

The runnable request/result pair under `examples/step-authoring/v1/` exercises
the candidate command. A credential-free end-to-end CLI test also selects a
local operation, binds and checks it, runs deterministic flow review, and then
passes the existing build and assessment gates using only a symbolic
credential binding.

### `step check`

This read-only request names one contract, exact step ID, expected intent
digest, and the exact `operation_ref` selected from `step candidates`. The
result echoes that reference and checks its source identity/digest and native
selector against the step and current local source, then checks required
request/output mappings, input/output types and requiredness against the step
contract, an available authentication alternative, declared dependencies, and
effect against the contract. Unsupported schema constructs or incomplete
source metadata remain indeterminate; known type/requiredness conflicts fail.
Request keys must select the location declared by the source (`body`, `query`,
`path`, `header`, or `cookie`). An unqualified name shared by multiple
locations is ambiguous. For direct `inputs.<name>` mappings, the declared
workflow input's type and requiredness must satisfy both the step contract
and selected source field. A nested or otherwise unproven expression is
indeterminate; `step bind` refuses it until the mapping is made provable.
Inline credential references use `credentials.<symbol>` with a lowercase
symbol of letters, digits, `_`, or `-`; `none` and `clear` are reserved and
cannot name credentials. Bind rejects malformed inline references, and check
reports them as incompatible in existing intent files.

A selected step that consumes its own output or participates in a cycle through
its transitive prerequisites fails the dependency check.
A source or intent digest mismatch is a `conflict`, not a fresh successful check. Its assessment is
`compatible`, `incompatible`, or `indeterminate`; deterministic checks and
unresolved semantic questions are separate. A structural match does not prove
that a natural-language outcome has been met.

Effect verification reuses APItools' local operation-candidate metadata for
the exact source selector and digest. Matching source-backed `read`/`write`
classification passes; a known conflicting class fails; missing or unknown
classification remains indeterminate. The check never infers effect from an
HTTP method.

`contract.id` and `step_id` are separate symbolic identities and need not be
equal: one contract may be checked against a specifically named intent step.

The implementation reads only `workflows/intent.hcl` and the selected local
source beneath `--example`. The final file and every intermediate component
must be regular/non-symlink paths contained in that example. The
path-independent `operation_ref.source_id` is `src-` plus the first 24
lowercase hexadecimal characters of SHA-256 over the cleaned, slash-normalized,
example-relative source path. This keeps source IDs stable within the example
without disclosing filenames; `step candidates` uses the same mapping.
OpenAPI operation selectors are JSON pointers such as
`#/paths/~1projects/get`; non-OpenAPI adapters retain their source-native
selector. Inventory compaction/truncation blocks a definitive operation check.
Candidates expose APItools' source-backed effect class, evidence, and reason.
An unknown effect remains indeterminate in `step check`; an HTTP method alone
never proves a read effect.

The runnable synthetic example under `examples/step-authoring/v1/example/`
can be checked with:

```sh
openudon step check --example docs/examples/step-authoring/v1/example \
  --request docs/examples/step-authoring/v1/requests/step-check-runnable.json
```

It completes with a compatible structural/effect assessment while retaining
the unresolved natural-language purpose question. The assessment does not
prove that the workflow outcome is fulfilled. `results/step-check-runnable.json`
is the expected value-free result fixture; digest conflicts and structural
incompatibility have separate result fixtures.

### `step bind`

The request names one step, contract, exact operation reference, explicit
request/output mappings, authentication alternative and symbolic bindings, and
the expected intent revision. Existing intent is updated atomically only when
that digest matches. A stale request writes nothing. If intent is absent, the
request must include an explicit workflow scaffold with workflow metadata and
top-level input/output declarations; the command never invents workflow-wide
policy. The source ID and digest must resolve to exactly one regular source file
under the matching source-family directory, including the legacy Google
Discovery alias; supported security sidecars and symlinks are excluded. The command
rechecks that digest and exact native operation selector/key before writing.
Required request fields, contract outputs, the selected OR-of-AND authentication
alternative, credential symbols, declared input references, and dependencies
must be validated before mutation; self-output references and prerequisite
cycles are rejected. The selected source operation's complete
input/output match must also be compatible with the declared contract; known
type/requiredness conflicts and unsupported schema dimensions return
`needs_input` without a write. Credential symbols lower to existing
`credentials.<symbol>` request mappings; reserved `none`/`clear` sentinels and
credential-looking values are rejected. Output mappings must resolve within the
selected response summary; OpenUdon's native step result remains
`received_body`, so these mappings do not add an unsupported HCL extension.
The selected operation must also have source-backed effect evidence compatible
with the declared `read` or `write` contract. Missing or unknown effect evidence,
an `unknown` contract effect, and a known effect conflict return `needs_input`
without writing. Authentication must have a complete source alternative;
missing/partial evidence returns `needs_input`. An explicitly empty alternative
is known anonymous access and remains distinct from absent security evidence.

The transaction replaces the selected step in place, or adds one top-level step
when no target exists, while retaining unrelated HCL blocks and comments. A
composite target that cannot safely be reduced to one API operation is rejected.
The shared transactional artifact writer enforces the expected intent digest
again at commit. Success reports the resulting intent digest and `written` or
`unchanged`; a failure after the filesystem commit point is represented as an
indeterminate write outcome, not as an ordinary rejection that invites blind
retry. Rejected validation and stale-revision requests preserve the original
bytes.

`step bind` writes only `workflows/intent.hcl`. It does not select credentials,
approve a package, call an API operation, or perform an external side effect.
Kinet must obtain its user's confirmation before asking OpenUdon to bind a
confirmed step.

### `flow-review`

This command exposes the existing advisory review over one assembled flow and
does not modify files. Deterministic local review always runs. Model review is
optional and requires explicit provider and model names in the request; its
credential comes only from the configured process environment and is never
included in the request or result. The result distinguishes `completed`,
`skipped`, `unavailable`, and `failed` model review from local findings.
Supported providers are `openai`, `anthropic`, `gemini`, and `copilot-api`;
model names use a bounded provider-safe identifier. An unavailable model
configuration leaves a completed local result with an explicit
`model_review.status: unavailable`; a failed or cancelled model call is never
reported as a passed model review. Unsafe credential-shaped review context is
not sent to a provider. Findings contain only advisory codes, warning severity,
safe messages, and validated step IDs; model evidence, repair suggestions,
provider errors, source paths, and prompt payloads are not returned.

Review findings are advisory and cannot replace build, assessment, approval,
or digest-bound package gates. For example, request and result fixtures can be
run against the synthetic example with:

```sh
openudon flow-review --example docs/examples/step-authoring/v1/example \
  --request docs/examples/step-authoring/v1/requests/flow-review.json
```

`model_review.enabled: false` runs deterministic review only. Enabling model
review sends the bounded intent-derived draft to the explicitly named provider;
the provider credential is read from its normal environment variable.

## Revisions and compatibility

Requests that read an existing intent carry its expected SHA-256 digest.
Candidates, check, and bind carry the same exact operation reference,
including the source digest. A source or intent change between candidate
selection, check, and bind returns `conflict`; the consumer must fetch current
evidence and make a new decision. Digests use
`sha256:` followed by 64 lowercase hexadecimal characters. Paths are never
used as the sole operation identity.

Unknown fields and unsupported versions are rejected. Additive compatible
changes require a new minor-compatible contract document and fixtures without
changing v1 interpretation. Changes to status names, digest meaning, command
identity, source identity, shared UWS fields, write guarantees, or effect/auth
semantics require a new major wire version. Consumers select behavior by
`version`, `command`, `kind`, and status; they do not infer success from
diagnostic text.

## Conformance corpus

The adjacent `examples/step-authoring/v1/` corpus is the public consumer
fixture set. Every valid request/result must pass
`schemas/openudon.step-authoring.v1.schema.json`; malformed, unsupported
version, unknown-field, and impossible status/result combinations must fail.
The corpus uses synthetic identifiers and no credentials, customer data,
provider calls, or environment-specific paths. Kinet validates these fixtures
without importing OpenUdon Go packages.

M87.1 recorded the APItools and UWS revisions inspected after reconciling their
contract rows, plus the Kinet W03 consumer requirements. APItools M77 is now
published and M87 consumes the exact pseudo-version pinned by OpenUdon. The
UWS field-shape agreement is complete; the tagged mapping check uses the
recorded local UWS declaration without publishing that sibling package.
