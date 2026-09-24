# Architecture

## Current Browser Authoring Boundary

The shared Browsertools authorurl validator binds reviewed command URLs,
parent and worker action traces, and generated navigation. Origin and path
observations remain separate from native redirect containment. Closed worker
and controller diagnostics stay private and value-free; native input identity
does not itself prove a browser run. W8M selects exact retained passing runtime
bytes, while OpenUdon's authoring boundary does not select a live target.
Registration checkpoint timing passes from Browserdriver through Udon to its
separate private form, outside OpenUdon authoring state. [E20](../docs/history/status-E20.md),
[E19](../docs/history/status-E19.md), [E16](../docs/history/status-E16.md), and [E14](../docs/history/status-E14.md) retain the
qualification lineage; the [history index](../docs/history/index.md) retains
superseded wording.

## Memory Bank Index

- This file owns system boundaries, data flow, planned structure, and security boundaries.
- Use [product.md](product.md) for product scope and non-goals.
- Use [tech-stack.md](tech-stack.md) for implementation technologies.
- Use [milestone.md](milestone.md) for milestones, acceptance criteria, current completion state,
  and the status-file index.

OpenUdon is the public UWS authoring, review, package, and executor-handoff layer above UWS,
API source metadata tooling, optional external orchestration, and private executor
implementations. It owns the workflow package lifecycle from brief authoring through review handoff
and trusted local execution.

## Current State

Registration v3 observations and ordered preview evidence produce review-v2
BRP 1.1 and transaction v3. The guided editor records only reviewed public
definitions; `inputBinding` follows the selected flow through intent, HCL, UWS,
quality and package review. Trusted execution invokes Udon through its external
CLI and protocol v5. Prepared form capability and expected snapshot identity
travel only through private runtime environment values. Optional consumer
authority restricts the shared application used by both HTTP and control;
expiry and worker teardown apply to both transports. No private Udon import
or target-specific browser implementation is added.

OpenUdon has a Go module, a thin `cmd/openudon` CLI, a guided `cmd/icot` authoring CLI, an
experimental single-workspace loopback iCoT UI/API, deterministic
synthesis/build/promote/assess commands, an eval harness, local readiness reporting, and a trusted
runner wrapper. It emits reviewed package artifacts under each example directory and validates those
artifacts before any approved udon execution path.

The v0.2 public boundary is CLI- and artifact-first. Deterministic package,
approval, handoff, and run-evidence commands are supported through v0.2.x;
implementation packages remain internal and are not a supported Go API.
Release archives co-version `openudon`, `icot`, and `udon-runner`, while
`openudon version --json` is the archive's build-metadata authority.

Generated packages now include project briefs, structured intent, workflow HCL, UWS YAML, expected
plans, OpenAPI discovery reports, refinement reports, review notes, quality reports, and
`expected/review-handoff.json` manifests. Executable packages use
`apitools.review-handoff.v2`; each input has a SHA-256 digest and the handoff
self-digest clears its own field before canonical JSON hashing.

## System Boundary

- Registration discovery inventory, coverage limitations and owner review are
  OpenUdon application concerns. `internal/registrationdiscovery` owns private
  bounded revision history; shared UI/control application operations expose it
  only on explicit request. Browsertools supplies native page observations.
  UWS owns reviewed portable flows, fields, steps and success predicates; its
  published optional 1.1 discovery metadata stays compatible, while new iCoT
  recipes omit inventory metadata. Selection does not alter drafts, runtime
  authority, browser lifecycle, containment or consumed-attempt state.
- `../uws` owns public workflow semantics, UWS versions, schema lookup, document parsing, JSON
  Schema validation, artifact discovery, Go model, and the explicit advisory
  content-trust analyzer contract.
- `../apitools` owns API source metadata search, discovery, import/materialization, download, local
  file scanning, operation indexing, operation summaries, auth/security summaries, catalog metadata,
  protocol-to-UWS-source-type mapping, operation ranking, and generic create/read/update/delete
  lifecycle-role ranking over operation summaries. OpenAPI/Swagger sources map to UWS
  `openapi`, Google Discovery maps to `google-discovery`, AWS Smithy JSON maps to `aws-smithy`,
  AsyncAPI maps to `asyncapi`, GraphQL maps to `graphql`, OpenRPC maps to `openrpc`,
  gRPC/protobuf maps to `grpc-protobuf`, and OData maps to `odata`.
- `../browsertools` owns browser capability/authentication/registration-profile
  validation and offline registration draft/review tooling, the headed
  author-session state machine and Playwright-Go context, private raw/normalized
  cache, guided-authoring/review envelopes, reviewed capability bundles, local discovery, and service-free static
  registry publication/search/pull. OpenUdon consumes only verified profiles
  and prompt-safe metadata. Its one explicit live client consumes only reduced
  protocol observations and the reviewed result; OpenUdon does not own
  accounts, membership, raw captures, drivers, or browser sessions. Its M28
  resolver supplies browser-profile channel and output capability semantics to
  explicit downstream content-trust analysis.
- `../ramen` owns desired-state conversion, reconciliation, state, graphing,
  planning, drift, import, apply/delete, audit artifacts, and infrastructure
  conversion source dependencies. OpenUdon may review/package UWS-facing
  artifacts generated elsewhere but must not import Ramen or own conversion
  mappings.
- `../evidence` owns product-neutral trust/evidence primitives used where the record shape is shared
  across products: digest records, artifact manifests, diagnostic records, redaction helpers,
  approval evidence primitives, and neutral async execution request/response/status/read
  observation records. OpenUdon keeps product-specific review handoff, approval JSON, package
  digest policy, tier rules, run evidence, async sidecar references, and trusted-runner semantics.
- `../authoring` owns product-neutral authoring primitives used where behavior is shared across
  products: the interview graph, deterministic frontier, round engine, prompt sessions/defaults,
  structured JSON fallback, lifecycle atomic writes, prompt-safe context records, agent result
  contracts, report metadata, scorecard summaries, and the clone-based binding that projects and
  settles one complete interview frontier atomically. OpenUdon keeps workflow-specific graph
  construction, prompts, workflow intent, v2 adapters, source staging, repair/proposal lifecycle,
  package artifacts, authoring-eval categories, and trusted-runner handoff.
- The shared prompt-safe source contract is `authoring.prompt-context.v2`.
  Apitools supplies ordered security requirement sets: the outer sequence is
  OR, requirements inside each set are AND, and an empty set is anonymous.
  OpenUdon owns the canonical SHA-256-fingerprinted operation-specific selection and does not flatten
  those alternatives into a union of credentials.
- `../openudon` owns UWS authoring, iCoT/progressive loops, prompt transcripts/replay, artifact sets,
  review evidence, approval state policy, package digests, credential-value scanning, symbolic
  binding contracts, review handoff validation, package gates, and trusted executor handoff.
- `../udon` owns private UWS/API/browser-profile compilation, lowering,
  persistent browser-driver lifecycle, credential resolution, MFA challenge
  brokering, named and legacy opaque session binding, exact runtime browser
  authentication/mutation approval,
  execution, runtime profiles, and
  runtime-plan behavior. The target public OpenUdon boundary invokes udon through CLI/Docker-compatible
  executor handoff rather than broad Go library coupling.
- `../openw8m` owns public IaC authoring/planning, concrete IaC intent, `.tf` generation, graph,
  profile, state, drift, and `w8m`-facing public artifacts.
Transitional debt:

- OpenUdon no longer imports `github.com/genelet/udon`; approved packages are handed to external executors through a portable run-config shim.
- OpenUdon imports `github.com/OpenUdon/apitools` only for API source metadata tooling.
  Product lifecycle helpers such as iCoT, transcript, review, handoff, credential scan, package
  digest, and approval policy are OpenUdon-owned.
- OpenUdon may consume `github.com/OpenUdon/apitools/catalog` for optional provider/spec/security
  advice in intent review evidence. Catalog matches do not change workflow semantics, release gates,
  credential binding, account selection, signing, or execution policy.

## Cross-Repo Contract Summary

OpenUdon must not teach prompts to emit workflow semantics that lack a public UWS contract. UWS 1.4
adds GraphQL, OpenRPC, gRPC/protobuf, and OData source description types on top of UWS 1.3 AsyncAPI
and the UWS 1.2 first-class API source description types. OpenUdon emits those source
families in new UWS 1.11.0 documents for reviewed local artifacts backed by source-aware apitools metadata, while downstream
trusted executors still own protocol execution compatibility. UWS 1.1 defines portable timeout fields and workflow-level
idempotency metadata; OpenUdon may preserve those only when project policy or intent explicitly
requests them. Switches, loops, structural results, failure branches, retries, and runtime profiles
are either public UWS constructs or extension-owned profiles, but OpenUdon still needs udon
compatibility proof and project policy before making them generation defaults.

- Private Browsertools results cross into iCoT only as immutable candidate
  transactions plus defensive canonical source/review bytes. The engine turns
  these into a generation-bound in-memory virtual catalog. Public snapshots
  expose digests, schemas, deterministic targets, symbolic bindings, and
  dependency/session metadata but omit source/review bytes and private paths.
  BCP selection closes over its exact BAP dependency and shared symbolic
  session; BRP is session-free. Catalog replacement and selection use an
  optimistic generation, exact selected identities are revalidated on every
  refresh, physical target collisions fail closed, and API sources stay ahead
  of virtual browser fallbacks. Only ordinary final authoring approval may
  materialize the retained bytes into the workspace.
- Authenticated-authoring v2 adoption now composes the independently validated
  BAP and BCP into one immutable candidate. The candidate rechecks canonical
  source/review bytes, exact review digests and assessment time, one selected
  authentication flow, complete symbolic slot bindings, login-state posture,
  exact origin union, and matching definitions for shared contexts. Its one
  symbolic session is provided by BAP and required by BCP; explicit operator
  acceptance produces only the validated `candidate -> reviewed` snapshot.
  Dependency-closed lowering and package metadata derive the two-source
  inventory, session bindings, and step-scoped authentication approvals while
  retaining no runtime session object.

Current generation policy:

| Capability | OpenUdon generation policy |
| --- | --- |
| Switch branches | Allowed; OpenUdon has prompt, plan, review, and quality support. |
| Loops | Allowed; OpenUdon proves loop lowering, plan/review evidence, UWS export, and quality coverage. |
| Structural results | Allowed for generated structural step outputs and validated against expected plans. |
| Failure branches | Allowed only when brief or intent explicitly asks for failure routing. |
| Retries | Allowed only when explicitly requested; side-effectful retries need retry/idempotency policy. |
| Timeouts | Allowed only when explicit `openudon-policy` or intent metadata requests them. |
| Idempotency | Allowed for explicit workflow-level UWS 1.1 metadata; OpenUdon does not inject API keys. |
| Runtime profiles | Allowed only for existing validated UWS runtime supplement shapes and project/environment policy. |
| Content trust | Allowed only through an explicit operator-authored registry. It requires UWS 1.9.1 or later; newly generated workflows declare UWS 1.11.0 and existing packages retain their declared versions. Assessment explicitly invokes UWS analysis, using Browsertools for contained browser-profile contracts, and emits warning-only quality/review evidence without entering ordinary validation or execution. |

The public UWS runtime supplement is a slim non-HTTP invocation selector for extension-owned
execution only. Public `x-uws-runtime` carries only `type`, `command`, `workingDir`, `function`,
`workflow`, and `arguments`. HTTP/API-source operations must use core UWS source binding fields plus
referenced source documents; `type: http` in public `x-uws-runtime` is rejected rather than treated
as a runtime profile. Provider selection, credentials, client/security configuration, and
request/response schemas belong in runtime-private configuration or product-owned profiles, not
public `x-uws-*`. This is intentional because runtime auth/security shapes for `ssh`, `cmd`,
`fnct`, `fileio`, `sql`, `s3`, `smtp`, `dns`, `ldaps`, `scp`, `sftp`, and `llm` are
implementation-specific and usually appear as runtime-owned arguments or private runtime
configuration rather than a portable public config object. Udon's legacy private `x-udon-runtime`
remains a separate compatibility concern until udon migrates its public DTO/export surface.

Closed cross-repo dependencies remain regression responsibilities:

- Structured output and UWS artifact preservation regressions are watched in OpenUdon and udon tests.
- Rich OpenAPI behavior is covered by OpenUdon eval fixtures first; reusable gaps move to `../udon`
  only after concrete failures.
- Review approval handoff is closed in OpenUdon through emitted artifacts and the trusted wrapper;
  managed reviewer routing remains optional external orchestration work.
- Provider drift is reported in eval JSON/Markdown and release notes.
- Optional sibling checkout readiness and secret-backed real-provider automation remain local/manual
  through readiness reports; public CI runs only provider-free module gates.
- Runtime/profile coverage stays as OpenUdon policy/eval evidence unless a reusable UWS/udon semantic
  gap is proven.

## System Flow

1. A trusted user or externally orchestrated task starts from a natural-language project brief.
2. iCoT may guide the user through goal, API source, operation, inputs, outputs, credential bindings,
   side-effect scope, safety, and fallback questions through the terminal or the experimental
   single-workspace loopback JSON transport.
3. OpenUdon saves `project.md` and `workflows/intent.hcl`.
4. An operator may add source/output/trigger/workflow-entry content-trust
   declarations after reviewing the intent; LLM generation does not author
   them.
5. `openudon synthesize` discovers/imports local API source inputs, generates intent when needed,
   optionally attaches catalog-derived provider/spec/security advice to review evidence, builds
   public UWS HCL/YAML artifacts directly from intent, and writes plan, refinement, review, handoff,
   and quality artifacts.
6. `openudon build`, `openudon promote`, and `openudon assess` rerun narrower stages after edits.
7. Quality gates validate project policy, API source availability, intent validity, data flow, workflow
   compilation, expected-plan matching, UWS profile/schema checks, review evidence, credential
   policy, side-effect policy, handoff contract, and secret scanning.
8. Reviewers inspect the minimum review package and, when appropriate, create approval JSON for
   sandbox or production tier.
9. `openudon run` revalidates the package, current quality, approval JSON, digest, tier/state rules,
   credential-value policy, and direct-production policy before writing run config and invoking the Go trusted executor runner by argv.
10. `openudon release-evidence` can verify and archive run evidence bundles,
   draft release-note evidence from the current commit/gates/verifier output,
   and run a provider-free non-dry-run smoke against a sibling-built udon
   executable without adding udon as a Go dependency.

## Desired-State Conversion Boundary

Desired-state conversion belongs in `../ramen`. OpenUdon no longer exposes a
conversion CLI, imports parser/conversion packages, or owns provider/resource
operation mappings. Ramen owns conversion source facts, API source binding,
review artifacts for converted projects, and native UWS/Ramen output.

OpenUdon may review or package UWS-facing artifacts after they are authored or
generated elsewhere, but conversion, provider plugins, state, plan/apply,
refresh, imports, credential resolution, cloud SDKs, and live infrastructure
APIs are outside the OpenUdon module boundary.

## Artifact Flow

- Inputs: `project.md`, local API/event source files under `openapi/`, `google-discovery/`,
  `aws-smithy/`, `asyncapi/`, `graphql/`, `openrpc/`, `grpc-protobuf/`, `odata/`, and legacy-readable `discovery/`, optional advisory security sidecars next to
  those API source files, optional existing `workflows/intent.hcl`, provider credentials in
  environment variables, optional reviewed non-secret runtime input data at `expected/data.hcl`
  including env-reference markers for environment-owned values, and sibling schemas.
- Authoring outputs: `workflows/intent.hcl` and regenerated `project.md` from iCoT reconcile.
- Local authoring coordination output: optional
  `openudon.browser-authoring-handoff.v1` JSON under an existing non-symlink
  private root with no group/other access, outside the example, new-only mode
  `0600`, local-ephemeral, and redaction-required before sharing. It is never a
  package input or trusted-handoff artifact.
- Generated outputs: `workflows/workflow.hcl`, `workflows/workflow.uws.yaml`,
  `expected/plan.json`, `expected/plan.md`, `expected/discovery.json`,
  optional `expected/data.hcl`, `expected/refinement.json`, `expected/refinement.md`,
  `expected/review.md`,
  `expected/review-handoff.json`, `expected/quality.json`, and `expected/quality.md`.
- Eval outputs: ignored JSON/Markdown reports and optional archived workspaces under `eval/runs/`
  and `eval/artifacts/`.
- Readiness outputs: ignored local readiness JSON under `eval/readiness/`.
- Trusted-runner outputs: local approval JSON and workdir artifacts under ignored operator paths.
- Release evidence outputs: archived `run-evidence.json`, `async-evidence.json`,
  optional `executor-report.json`, local release-note drafts, local udon smoke
  summaries, and `openudon.release-evidence-summary.v1` JSON/Markdown summaries
  under ignored operator paths.

## iCoT Architecture

iCoT is OpenUdon's adaptive interview and workflow-authoring layer, not a synthesis or execution
engine. It maps a broad request into one confirmed active boundary—outcome, actor/trigger,
observable success evidence, non-goals, and side-effect posture—and keeps later workflows as
unnumbered candidates with deferral reasons and promotion triggers. Candidate workflows receive no
sources, operations, mappings, or implementation breakdown.

The generic interview graph is `authoring.interview.v1`. Every round contains all open nodes whose
dependencies are settled; the complete frontier is displayed before answers are collected, all
answers are applied together, and normalization/autosave runs once. Node states are `open`,
`settled`, `deferred`, and `inapplicable`. Source, operation, mapping, and output leaves may be
deferred only with owner, impact, unblock condition, and next action. Boundary and side-effect
posture cannot be deferred. There is no breadth ceiling; cancellation, approved draft deferral,
completion, or three consecutive no-progress rounds end the loop.

Before questioning, OpenUdon calls apitools' bounded local discovery over existing example sources,
explicit `--api-source`/`--openapi` documents, and explicit `--source-root` paths. Discovery
validates OpenAPI/Swagger, Google Discovery, AWS Smithy, AsyncAPI, GraphQL, OpenRPC,
gRPC/protobuf, and OData; rejects symlinks and non-regular paths; deduplicates by SHA-256; requires
an explicit kind for ambiguous JSON/XML; and exposes limit diagnostics. Directory conventions are
hints only. The default bounds are 10,000 visited entries, 100 accepted documents, and 20 MiB per
file. Ambiguity or truncation blocks interactive, complete-session, and agent paths before any
partial candidate set can be approved or materialized.

Browsertools runs alongside apitools over explicit `--browser-profile` files,
existing `browser-profiles/`/`capability-bundles/`, and the same explicit source
roots. API sources remain preferred when an operation covers the active
capability. A verified browser profile is eligible only for an uncovered
UI-only capability or an explicit reviewed browser route. Profile precedes
action; action precedes mappings, opaque session posture, mutation approval,
outputs, fallback, and verification. Mutating actions require exact step-level
authoring approval and retain Udon's separate runtime approval gate.

Remote discovery is separate and approval-gated. After local evidence is exhausted, iCoT may consult
only curated apitools catalog references plus one APIs.guru list request. The total deadline is eight
seconds, output is capped at three metadata candidates, unsafe hosts remain rejected, and no remote
document is copied. Denial, timeout, unsafe results, or no matches becomes a deferrable source
blocker.

Configured Browsertools registries remain service-free static directories or
HTTPS object-storage catalogs. Local registries are offline. HTTPS registry
lookup has a separate forced approval decision from API lookup, the same
eight-second/three-result/20 MiB bounds, and Browsertools unsafe-host and
lifecycle verification. No result creates a placeholder. Selected bundle bytes
remain external/in-memory until proposal approval, when only the verified
materialized profile is staged.

All OpenUdon-owned remote source HTTP uses a DNS-aware transport. Initial URLs
and redirects are checked, resolution observes the caller context, any unsafe
or mixed answer rejects the request, and the connection dials a validated IP
without a second lookup. A supplied custom transport is rejected unless this
policy can be enforced. One canonical source-directory inventory covers every
API family plus browser profiles, authentication profiles, and capability
bundles for CLI, engine/UI, discovery, and seed copying.

The active graph orders source before operation and operation before mappings, security, and output;
all execution-critical leaves precede proposal approval. Existing metadata-bound ranking,
operation-detail expansion, deterministic prework, request-mapping assistance, Gmail report helper,
and bounded flow-review repair remain available only after their dependencies are settled. LLM-added
operations must still be listed in inspected metadata. `--review-repair` remains limited to two
narrow mapping/output/dependency or proven local transform/prework changes and cannot silently
change source, operation, credential, active boundary, or side-effect posture.

For an operation with multiple security alternatives, the graph adds a forced
selection node before any credential or request-mapping node. The answer is
stored in interview metadata as a canonical SHA-256 fingerprint independent of
source ordering; display labels are descriptive only and ambiguous labels are
rejected. A legacy one-based index is accepted only when unique decision
evidence confirms the same alternative. Only the
selected alternative's full AND binding set is eligible for mappings. An
explicit anonymous alternative requires no credential mapping. Selection and
evidence attributes survive draft save/resume, while an unresolved alternative
can produce only an incomplete, non-runnable draft.

Prompt sanitization is fail-closed for semantic loss. If per-operation or
aggregate prompt budgets would omit security alternatives, bindings, fields,
or selected source context, the diagnostic becomes a visible deferable
technical readiness blocker; it is never treated as permission to approve the
partial interpretation.

The durable OpenUdon session is `openudon.icot-session.v2`; the transcript is
`openudon.icot-transcript.v2`; author/lint/repair/scorecard/authoring-eval/variant/replay wires are
v2. v1 inputs are rejected without a compatibility decoder. One unified interview evidence ledger
replaces the overlapping durable annotation, assumption, mapping-classification, and
decision-evidence collections and records concise public rationale only, never hidden model
chain-of-thought. Machine-readable evidence attributes preserve confidence and explicit-confirmation
qualifiers so resume reconstructs the same safety/readiness decisions as the pre-save session.

Only ignored resumable `.icot/` state may be autosaved during the interview. Before any deliverable
write, iCoT shows the active boundary, candidates, steps, source origins/digests/targets, mappings,
safety policy, deferrals, and exact file actions. Complete approval atomically writes `project.md`,
selected sources/security sidecars, and `workflows/intent.hcl`. Approved incomplete technical work
writes `project.md`, `workflows/intent.draft.hcl`, confirmed sources, `.icot/session.yaml`, and
`.icot/readiness.json`, never final intent. Promotion atomically removes obsolete draft/readiness
files. Source collisions reuse identical content, reject differing content without `--force`, and
share the same backup/rollback transaction. Multiple selected sources targeting the same path reuse
one identical digest or fail before staging when their digests conflict.

Browser routes additionally write `.icot/browser-sources.json` in the same
transaction. It binds the package profile digest to actions, origins,
lifecycle/expiry, provenance, registry coordinate, login-state requirement,
session posture, and exact authoring approvals without carrying driver,
credential, session value, or raw-capture data. Build/assess revalidate this
metadata and profile, inventory both in the handoff digest, and reject stale,
revoked, invented, sensitive-shaped, or unconfirmed browser behavior.

`internal/browsertransaction` owns strict semantic validation, deterministic
encoding/digests, and immutable lifecycle transitions for the public
`openudon.browser-profile-transaction.v1` JSON wire. Its published schema is
structural; OpenUdon's validator additionally enforces canonical array order,
published UWS family/version composition, canonical origins and timestamps,
symbolic bindings, failure class/code pairing, and allowed state edges. Strict
input is bounded to 256 KiB and rejects invalid UTF-8, duplicate object names,
excessive nesting, unknown fields, and trailing JSON. The wire composes one
authentication then capability candidate with a symbolic session, or one
session-free registration candidate. It retains only digests and value-free
provenance from a private Browsertools result; result paths, worker output,
page/request data, credential/account values, and runtime session material
never enter the transaction. Preparation and promotion are artifact lifecycle
facts and grant neither runtime nor target authority.

`internal/browsertransaction/engine` now coordinates start/observe/review,
prepare/qualify, atomic promotion, cancellation, selected inspection, and
digest-confirmed recovery through optimistic value-free snapshots. It calls
the packagepipeline adapters directly and contains no browser, credential,
prompt, or runtime operation. `internal/browsertransaction/presentation`
derives one kind-specific review shared by experimental API v4, the accessible
embedded shell, and `icot browser-transaction`: BAP+BCP retains only symbolic
session/bindings, while BRP adds the canonical heuristic/not-DLP,
GET/HEAD-only, no-submit/no-account/no-session/no-runtime disclosure. UI launch
and terminal launch accept only a bounded public transaction artifact plus
process-private package configuration; private Browsertools result paths and
bodies have no frontend representation.

`internal/browsertransactioneval` owns the separately versioned,
independently verifiable `openudon.browser-transaction-qualification.v2`
release-evidence wire. Its canonical JSON binds exact clean OpenUdon,
Browsertools, Browserdriver, Udon, and unchanged UWS commits plus their
independently resolved publication classifications; nine BAP+BCP lifecycle
digests; eleven BRP authoring/package/attestation/execution digests; 18 fixed
gate outcomes with closed failure codes; and explicit sandbox, loopback,
GET/HEAD-only registration authoring followed by one separately approved POST,
fixture-only account creation, executor invocation, a fixed result, and no
registration session. The schema intentionally has no free-form
diagnostic, path, subprocess-output, browser-content, account, credential,
cookie, storage, or session-material field. A bounded stable reader, strict
decoder, canonical-byte comparison, and exact digest sidecar make retained
reports independently tamper-verifiable; the report grants no publication or
non-loopback target authority. The `browser-transaction-eval --out` runner checks all five
OpenUdon, Browsertools, UWS, Udon, and Browserdriver worktrees against their
exact local/published posture before and after execution, independently
resolves every origin `main` read-only, requires UWS to remain at its published
lock, runs the bounded adversarial target, executes the real sandboxed
loopback BAP+BCP and complete iCoT-to-Browserdriver BRP
qualifications, then atomically writes and independently re-verifies the sole
report and sidecar. Child output is discarded rather than entering evidence.

`internal/browsercandidate` anchors a canonical mode-`0700` per-run private
root before the Browsertools registration worker starts, snapshots existing
digest-named results, and admits exactly one new mode-`0600` regular result
only after clean process exit. Its 256 KiB stable reader detects root, entry,
identity, size, mode, and modification drift; rejects symlinks and replacement;
strict-decodes the registration-authoring v1 result at the assessment instant;
and independently rebuilds canonical BRP and registration-review bytes. The
result must exactly match the human-confirmed profile, flow, current-generation
candidates, origins, cleanup disposition, symbolic slot bindings, GET/HEAD
accounting, and false submit/account/session/runtime claims before an M77
candidate transaction exists. The returned candidate owns defensive copies of
only canonical source/review bytes and the value-free transaction; it retains
no private result name, path, or envelope.

The same package also owns the path-free authenticated BAP+BCP composition.
It accepts only exact canonical source and independent review bytes already
recovered from the private envelope, revalidates compatibility and earliest
expiry, and returns defensive copies plus one candidate transaction. The
engine adapter can produce either that candidate view or its immutable
reviewed transition and then feeds the existing generic virtual catalog.

The shared `internal/icot/browserauthor` boundary also re-executes the
importable Browsertools registration worker from the stabilized mode-`0500`
cache. It uses the existing minimal environment, process-group containment,
fixed idle/absolute ceilings, strict type-specific NDJSON decoder, and complete
stdout drain. Candidate adoption begins only after the worker reports `closed`
and the contained process tree exits successfully. The Linux sandbox-helper
selector is forwarded without credentials or model environment and remains
subject to Browsertools' administrator-owned helper validation.

A27 retains the final value-free registration outcome independently of the
bounded event stream and reconciles it after stream closure. A joined process
termination timeout is always `worker_teardown`, never ordinary `worker_exit`.
That containment failure becomes process-global: registration, capture and its
preflight/staging, package construction/resume, browser-transaction changes,
and ordinary authoring mutations remain closed until iCoT restarts. The Linux
process-tree tracker never replaces a recorded PID/start-time identity, treats
procfs health loss as failed containment, takes a verified live-leader group
kill before cancellation can reap that leader, avoids post-reap numeric
process-group signals, and preserves teardown timeout through a caller deadline.

The A24 guided-draft adapter keeps observed and declared authority distinct.
Current-generation reduced candidates may supply macro locators and the one
submit control, while credential steps select declared symbols and may reuse
one `password` symbol for confirmation. `contact_name` is a value-free
identifier-class default. The post-submit success origin/path/role/name is an
explicit operator-reviewed declaration, never a pre-submit candidate; API
review state marks it unobserved and runtime-proof-required, and profile
evidence records mixed no-submit observation plus operator review. Invalid,
undeclared-origin, noncanonical, sensitive, or unacknowledged success input
fails before Browsertools review.

A25 keeps registration binding validation name-aware without treating an
environment symbol as a credential value. Ordinary low-entropy portable names
remain valid. When the value-oriented entropy heuristic fires, the draft
requires positive descriptive snake-case structure from a closed purpose-word
vocabulary and at most one alphanumeric product namespace of up to 12
characters containing one digit run. At least two purpose words are required,
and the name does not need to repeat the declared slot. Known credential
formats, short opaque-prefix suffix bypasses, and digit-bearing or letters-only
opaque multi-token names fail before draft construction. Binding-specific
rejection remains value-free. Bindings remain transaction-only metadata and
never enter the canonical BRP.

A26 makes one-session authority process-authoritative at the API boundary.
After authenticated start preconditions pass, the server consumes one
registration-authoring attempt immediately before worker construction and
records `attempt_consumed` on every subsequent public authoring state. Any
later start, including a rapid request carrying the original revision, returns
the fixed `registration_authorization_consumed` conflict before workspace
inspection or worker construction. Terminal worker codes pass through a
closed OpenUdon allowlist into `failure_code`; unknown input collapses to
`worker_failed`, and raw worker text, paths, or target observations have no
representation. The browser UI uses the server state to disable Launch and
explain that a fresh preflight, authorization, and process are required.

Explicit repeatable `--browser-verification` inputs add a downstream-only
adapter for Browsertools' value-free `live-check.v1` and
`portability-check.v1` wires without pinning OpenUdon to Browsertools'
unpublished capture package. The adapter bounds regular-file reads, rejects
duplicate/unknown/missing/trailing JSON, reconstructs the closed profile probe
plan, validates lifecycle/origin/action/engine/fixed-diagnostic consistency,
and deduplicates cross-report facts. iCoT retains the external path only in
resumable local state, reopens the report at approval, and writes only the
normalized summary plus source digest into `.icot/browser-sources.json`.
Build/assess independently revalidate the summary and selected-action coverage.
The report itself, rich evidence, backend errors, and private session material
are not packaged. Absence is valid; portability never becomes an execution
requirement or a locator rewrite.

When no reviewed source exists, `icot browser-authoring plan` emits an inert
`openudon.browser-authoring-handoff.v1` local-ephemeral plan. The plan contains
typed argv templates and explicit human gates for a separate Browsertools run;
iCoT does
not execute the argv, inspect credential environment variables, contact the
site, install a browser, or place private capture material under the example.
An explicit `browsertools.guided-authoring.v1` result can return through
`--browser-profile`: OpenUdon strictly decodes it, replays Browsertools' draft
construction, verifies exact decisions/profile/review evidence at the current
time, rejects secret/session/private-browser-shaped content and literal guided
text/select values, and stages only canonical `uws.browser.1.5` profile bytes
after proposal approval. Broad source-root scans do not promote guided
envelopes.

`icot browser-author live` is a separate explicit execution boundary; normal
iCoT and agent mode remain non-executing. It validates an absolute Browsertools
binary, launches `author-session chromium` with a minimal credential-free child
environment, and strict-decodes the bounded
`browsertools.author-session.v2` NDJSON union. Browsertools owns one
non-persistent context across human credential/MFA entry and goal exploration.
iCoT receives only exact origin/path/context plus candidate role, redacted
label, match count, complete portable context inventory, and fixed diagnostics.
Candidate authority is observation-generation scoped. A once-per-run provider/
model disclosure gate precedes model access, denial falls back to human
guidance, and no protocol transcript is persisted. Typed continuation,
API-first override, new-origin, click/POST, human-input, typed-plus-human
completion, and final import remain separate gates; `--yes` bypasses none.

Bundled, expert external, HTTP/UI, and loopback qualification launches now
share `internal/icot/browserauthor`. The parent keeps a non-serializable
attestation of the exact ordered actions, human checkpoints, approval cards,
observations, additive context inventories, requested outputs, dashboard proof,
and approved-origin ledger. Final staging requires that attestation to match
the child envelope; bounded execution counts remain child-owned. The
attestation has no HTTP, JSON, transcript, or workspace representation.

Every configured or page-derived URL path is admitted through Browsertools'
shared disclosure validator before terminal, HTTP state, planner input, or
result import. Worker executables are fully copied and hashed into a private
content-addressed mode-0500 cache; hard termination can leave at most one
bounded owned temporary, and later startup sweeps only stale regular files with
the exact owned prefix.

Browsertools writes a deterministic mode-`0600`
`browsertools.authenticated-authoring.v2` envelope only after teardown. The
envelope stays under a disjoint private root. OpenUdon reopens it as a stable
regular file, verifies its digest, time, bounds, origins, context graph, trace,
goal proof, human confirmation, profile review digests, freshness, schemas, and
secret absence, then atomically stages only canonical authentication/capability
profiles plus safe `.icot` review metadata after explicit approval. Existing
targets fail closed; normal iCoT performs the later flow/action/session interview.

E03 requires the initial state and final result to echo the exact finite bounds
iCoT granted, validates every disclosed context as an additive exact-origin
graph, and rejects unknown planner contexts or unsafe raw labels before model
disclosure. Review decisions use the producer's bounded discriminator alphabet
(including dots and hyphens) independently of closed runtime diagnostic codes.
The release matrix names a real Browsertools envelope consumption test and a
separate Browsertools-to-Udon/Browserdriver replay test; component-local
fixtures cannot stand in for either seam.

A05 imports Browsertools E06's canonical accessibility-label reducer instead
of maintaining a second phrase/redaction policy. The protocol reader uses 512
only as its pre-negotiation absolute ceiling, switches to the requested 128
immediately after `start`, and validates observation length plus every match
count before display or model disclosure. Candidate checks are ordered by ID
syntax, duplication, role, match count, and canonical label. A rejected valid
ID may appear on local stderr only with a closed reason; malformed IDs, labels,
role text, page content, and child-process prose do not. Phrase screening stays
defense in depth: the primary model boundary remains closed typed action
validation over observed unique IDs, same-origin GET navigation, and
human-approved clicks.

A06 requires the v2 reviewed-MFA/output capabilities and sends
`MaxOutputs: 16`. Credential/MFA attention completes through a distinct
`human_input_complete`; only the human may return one exact advertised
challenge kind. At completion the human may declare a bounded list of
current-observation outputs, receives a value-free sorted summary, and must
confirm before `human_complete` sends the explicit list. OpenUdon validates
returned challenge kinds, symbolic credential slots, output keys/types/
locators, context and match proofs, profile discriminators and reviews, then
deterministically reconstructs both profiles to reject substitution even when
the attacker also updates a digest.

`openudon browser-integration-eval` is a release-evidence adapter outside the
iCoT runtime path. It runs fixed named tests and boundary checks in OpenUdon,
Browsertools, UWS, Udon, and Browserdriver, observes all three pinned browser
component inventories without installing or launching them, and emits only a
strict `openudon.browser-integration-eval.v2` report plus digest sidecar. The
verifier retains the historical v1 report and gate inventory. The
report binds each sibling commit and dirty-state bit, fixed argv/assertions,
closed result details, and the no-browser/no-target/no-credential-value/no-write
authoring claims. Child stdout/stderr, repository paths, page values, raw/rich
evidence, credentials, cookies, storage state, and sessions are never retained.
Requested installed-engine or headed-authentication loopback checks are
skipped when their doctor prerequisites are unavailable; they never authorize
a real site or account.

E02 extends that same v1 report without adding execution authority: required
gates now prove OpenUdon's strict live adapters and UWS 1.7/1.8/1.9 selection,
Browsertools' author-session and deterministic profile synthesis, UWS context
schema dispatch/compatibility, Udon v2/v3 selection, and Browserdriver
popup/frame enforcement. `--headed-auth` activates separate authentication and
same-context authoring loopback fixtures; default evaluation remains
browser-free, network-free, credential-free, and target-free.

E03 strengthens the unchanged report contract with exact artifact seams and
freshness assertions. Browsertools gates cover generation-scoped candidates,
action-time semantic revalidation, actual response sizes, closed phases, and
ordered exploration synthesis. UWS covers fresh decoding into reused Go
values. Udon accepts authentication 1.1 with browser 1.5, 1.6, or 1.7 under
v3, and Browserdriver revalidates cached contexts before every use and at flow
completion. A06 extends the required names to exact MFA/output v2 authoring,
UWS 1.9 lowering, scalar validation, and replay failure non-disclosure.

E04 adds `internal/browserscenario` outside normal iCoT authoring. Its embedded
strict manifests and v2 compatibility lock are validated before browser or
network authority. The loopback executor drives the production Browsertools v2
wire through OpenUdon's normal result reconstruction and staging, synthesizes a
real UWS document, and replays it through external Udon/Browserdriver v3. Its
fixture gates goal pages behind a random session cookie and verifies passwords
and every MFA challenge server-side, so navigation alone cannot prove
authentication. The
public executor requires explicit network authority, runs fixed anonymous
Browsertools presence probes, and independently replays credential-free
browser 1.5/UWS 1.7 presence outputs through Udon/Browserdriver v2. Every case
gets a fresh server or work root and browser lifecycle. Reports contain only
exact revisions, closed phases/assertions/failure classes, counters, safety
claims, and a digest sidecar; they contain no values, page content, or child
output.

M85 keeps that scenario lock for historical real-browser qualification. The
provider-free integration evaluator embeds a separate exact UWS 1.11 stack
lock and emits a v2 report with named Browser 1.8/1.9 and v10 evidence. Its
verifier dispatches v1 reports to the original lock and gate inventory, so
later source qualification does not redefine prior report meaning.

M86 adds a distinct current-stack scenario lock and v2 local report dispatch.
Historical v1 scenario reports and the 23+8 corpus retain their meaning. The
current loopback suite reuses the 23 cases; the current journey suite adds
three reviewed local Browser 1.8/1.9 template cases, including one mixed
legacy/modern named session through v10. The current verifier requires its
exact sibling revisions, module pins, closed assertions and complete local
inventory for passing release evidence. Make and hosted release checks select
the current stack explicitly; public canaries keep the historical opt-in.
Real-browser qualification and exact report digests remain M86 acceptance.

The scenario JSON boundary pre-scans tokens recursively and rejects duplicate
decoded keys in every object before unknown-field decoding, so no consumer can
select a different repeated value. Hosted Ubuntu release/public jobs retain
Chromium's sandbox and explicitly provision the unprivileged-user-namespace
kernel settings required by Playwright; `xvfb-run` remains display-only.
The lock rejects dirty or revision-mismatched Browsertools, Udon, UWS, and
Browserdriver siblings and pins the actual Playwright package and Chromium
browser versions. Browser integration evaluation consumes the same repository
validator and Playwright contract; scenario preparation independently launches
the installed pair once to compare both actual versions.

E05 adds the local headless `journey` branch to the same evaluator. Each case
constructs normalized Browsertools evidence, authors a deterministic
`browsertools.guided-authoring.v1` bundle with the public guide API, and sends
that private bundle back through OpenUdon's strict source discovery and
materialization seam. Only the canonical browser 1.5 profile crosses into the
example; evidence, decisions, review, and draft spec remain private. A
credential-free authentication 1.1 fixture selects UWS 1.8, ordered
parameterized browser operations share one execution-local named session, and
external Udon plus Browserdriver v3 drive fresh headless Chromium. Positive
cases compare typed outputs and local server state exactly; negative approval,
ambiguity, input-schema, additional-parameter, type, and origin cases prove no
mutation. The dedicated `openudon.browser-journey-eval.v1` report remains
value-free and uses the existing compatibility lock.

Login-required ordinary authoring plans still fail closed, but the explicit
live command supports same-context post-login dashboard learning and portable
popup/frame SSO. The context never transfers: only UWS 1.8 profile contracts
may describe reviewed popup/frame relationships, while CAPTCHA, enrollment,
recovery, password change, consent, account creation, and logout remain outside
scope.

Reviewed `uws.browser-authentication.1.0` and `1.1` profiles are a separate local-only
Browsertools source family. They are staged under `browser-authentication/`
with `.icot/browser-authentication.json` digest, flow, origin, expiry,
credential-slot, named-session, and exact authoring-approval evidence. iCoT
orders a selected flow before symbolic credential mappings, bounded timeout,
authentication approval, and a protected `uws.browser.1.5` action using the
same session. OpenUdon lowers these steps to the public
`uws.browser-authentication-call.1.0` and named-session supplements in UWS 1.7
for old main-page sources. Authentication 1.1 requires authentication-call 1.1;
old profile meanings remain unchanged. Newly generated workflows declare UWS
1.11.0. Browser 1.7 retains its scalar conversion under the legacy inner
action protocol. Browser 1.8/1.9 profiles pass local validation and review
with their exact discriminator and select trusted browser-driver v10 for
action execution, including mixed sessions with older profile actions.
All scalar outputs remain subject to Udon's post-conversion schema and secret
checks.
Credential-less passkey/security-key flows lower an explicit empty binding
object. Browsertools `*.review.json` sidecars are not inventoried as profiles,
and session review metadata walks nested structural steps exactly as quality
validation does.

Reviewed `uws.browser-registration.1.0` profiles form a third, separate
browser source family under `browser-registration/`. Each source requires its
exact Browsertools `*.review.json` bundle plus OpenUdon review evidence that
binds the profile bytes, selected flow, exact origins, complete symbolic
credential map, registration approval name, fixed duplicate and ambiguity
policy, and preselected cleanup disposition. OpenUdon lowers the intent to the
extension-owned `uws.browser-registration-call.1.0` operation and treats it as
an account-creation side effect. It never stores an account identifier or
claims that an attempt occurred. Registration has no browser-session field.
Approval-template and trusted-runner dry-run validate the immutable package.
A non-dry path exists only when the exact Udon execution-report-v3 and
Browserdriver-v4 contracts are configured; it requires a separate private
digest-bound dedicated-test attestation plus exact submit approval. Legacy,
mixed, or incomplete configurations still fail before process invocation.
OpenUdon never resolves a credential, handles an MFA response, launches a
driver itself, or stores cookies/storage state/live sessions; those remain
behind Udon's private runtime and separate exact approvals.

Private BRP adoption now follows the same immutable candidate-to-reviewed edge
as BAP+BCP while remaining session-free. The reviewed transaction exposes one
exact virtual registration operation carrying its symbolic bindings, bounded
timeout, authoring-time no-runtime marker, and fixed duplicate, ambiguity, and
cleanup policy. That marker prevents transaction review or the iCoT shell from
becoming an execution route; the separately configured trusted handoff
revalidates the promoted package and private attestation before Udon can open
the runtime operation.
iCoT lowering requires a fresh step-scoped authoring confirmation, preserves
the profile's submit and human-checkpoint sequence only as inert source, and
rejects any invented session, operation, binding, or policy. Ordinary approval
revalidates the in-memory canonical profile and review bundle before one atomic
artifact transaction proposes the profile, adjacent review, and strict
`.icot/browser-registration.json` inventory. Drafts and public snapshots omit
both byte bodies, so resumed authoring must rediscover the exact transaction.
Exact rediscovery rehydrates the selected plan only in memory and retains its
source-scoped authoring approvals. Missing, stale, or identity-changed
rediscovery blocks resume; deselection or replacement clears only affected
BAP, BCP, and BRP approvals before repair and reapproval. Registration
cancellation and partial private results cannot produce a candidate.

Package promotion begins with a pure byte-generation boundary in
`internal/packagepipeline`. Preparation requires an explicit portable scope,
strictly reads and then rechecks the complete handoff-bound inventory, enforces
file and aggregate byte bounds, and rejects optimistic generation drift. Its
defensively copied manifest contains only portable paths, digests, passing
quality state, approval-state names, execution policy, and symbolic credential
names; the canonical source root stays private and preparation performs no
write or approval operation.

Qualification materializes only that prepared byte set beneath a fresh
same-filesystem mode-0700 root. Package directories must remain mode 0700 and
files must be mode 0600, single-link regular members with no case aliases or
unsupported entries. The scratch generation is independently re-prepared,
quality- and secret-checked, package/handoff-inspected, and trusted-dry-run
without executor invocation. Anchored cleanup runs on every exit; the returned
report is deterministic and value-free and carries no scratch or source path.

Promotion stores each complete restrictive package beneath an immutable
generation directory named by a digest over its preparation and qualification
records. Files and descendant directories are synchronized before publishing
that directory; the generation parent is synchronized before one atomically
replaced `current.json` can select it. The strict selector records the exact
selected and immediately prior generation plus scope/package/qualification
digests. Same-generation promotion is idempotent, a create-only lock rejects
competing builders, readers resolve only published generations, and promotion
does not perform retention deletion.

A durable digest-bound intent records the exact baseline selector and target
selector until selection plus cleanup are proven. Pre-selector failures remove
that intent and return a typed rollback while leaving any published target
generation unselected. Post-selector or cleanup ambiguity retains recovery
evidence and returns an indeterminate state that blocks retry. Read-only
recovery inspection validates current, prior, target, intent, lock, and a
bounded transient inventory; reconciliation requires the exact observation
digest and a second unchanged read before anchored transient cleanup. It never
rewrites the selector or removes target, selected, prior, or any other
generation, and any identity drift remains recovery-required.

Compatibility adapters reapply the exact lifecycle to an existing package and
route selected-generation inspection, approval templates, and trusted runs
through a caller-observed selector digest. Selected runs canonicalize approval
and work paths outside the immutable store and delegate all dry-run/non-dry
authority to the existing trusted runner. The `openudon package` namespace
exposes prepare, explicitly confirmed promotion, read-only inspection, and
digest-confirmed reconciliation; the pre-existing artifact `openudon promote`
command is unchanged. The browser transaction engine, API/UI, and terminal
consume these adapters rather than duplicating filesystem behavior.

Prompt modes retain `full`, `normal`, and `fast`; final proposal approval is forced in every mode,
with `--yes` as explicit noninteractive approval. Agent mode never prompts or writes. It returns the
whole frontier, candidate workflows, source evidence, blockers, and proposed file actions, including
`proposal_approval_required` for otherwise complete state.

The internal `internal/icot/engine` package is the Phase A driver boundary for
the terminal and local UI authoring interfaces. It has no reader/writer handles: `Open`
loads empty, seeded, explicit-session, or resumable state and performs the same
bounded API/browser discovery; `Snapshot` returns JSON-marshalable frontier,
readiness, evidence, source, proposed-action, and preview state; `ApplyRound`
accepts exactly one complete dependency-ready frontier and autosaves only
resumable state; `ReopenDecision` transactionally clears one advertised
settled human decision and returns its replacement frontier; `Preview` renders without final writes; and
`ApproveAndWrite` requires explicit human approval. Terminal iCoT and this
engine share one OpenUdon-owned artifact transaction for source
revalidation/materialization, browser capability/authentication review
metadata, draft promotion/cleanup, and rollback-capable atomic writes. The
engine returns deep-cloned snapshots, derives approval-capable file actions
from that prepared transaction, and refreshes approval state transactionally.
One source-refresh routine now serves interactive, complete-session, agent,
progressive, restart, and engine/UI paths. It owns local discovery,
inactive/ambiguous/truncated assessment, registry trigger evaluation,
selected-registry coordinate/target/digest revalidation, plan synchronization,
and verification attachment. Retained registry profiles must be rediscovered
with the same coordinate and content digests before approval, and failed verification/registry refreshes
leave the previously reviewed state intact for every retry. Frontier question
slots, rather than caller-provided routing fields, remain authoritative. The
engine neither starts Browsertools live authoring nor owns an HTTP server,
frontend, folder browser, or public schema.

A09 makes both engine mutations prospective transactions. `ApplyRound` builds
the refreshed state and exact snapshot before its atomic draft save, then
installs both without consulting request cancellation again. Approval returns
one `ApprovalResult`: its exact approved snapshot and prepared write plan exist
before commit, while `WriteResult` is constructed afterward directly from the
commit outcome. There is no fallible post-commit refresh or redundant draft deletion.
Engine failures have closed rejected, conflict, operational, and indeterminate
classes.

The engine maintains a sorted SHA-256 fingerprint over fixed project, draft,
final-intent, metadata, selected materialized-source, and proposed-action
paths. It checks that baseline before mutation and again at the writer
replacement boundary. Pre-refresh observation is bounded to current watched
paths and the current local/registry materialization targets; unrelated
workspace files are never enumerated. Hashing streams through context checks
and verifies file identity, type, size, mode, and modification time before and
afterward. A newly produced, previously unobserved target is treated as
missing, so an existing path becomes drift rather than an adopted baseline. A regular-file change from an editor
or second process latches `externally_modified`, changes the HTTP revision,
preserves cached inspection, and rejects mutations until restart. Unsafe or
unreadable watched paths fail closed operationally. This optimistic design
intentionally adds no persistent workspace lease. A13 retains exact-byte
hashing on every visible poll: the reproducible 1 MiB benchmark measured about
3.2 ms on the reference Haswell VM, while a regression proves that same-inode,
same-size, restored-mtime byte changes are still detected. A metadata-only fast
path would violate that ownership guarantee.

`internal/icot/ui` is the Phase B local transport and Phase C browser shell over exactly one engine and
one explicitly selected example. `icot ui` opens the engine using explicit
answers/from-example, resumable session, existing final state, then empty-state
precedence; generates a 256-bit internal process token; binds only `127.0.0.1`;
and opens a tokenless bootstrap page. A random 12-character Crockford Base32
code is printed only in the terminal. It expires after five minutes, is
single-use, and permits five failed attempts per minute. An exact-origin POST
exchanges it for an HttpOnly SameSite=Strict cookie scoped beneath an
unguessable per-process path and redirects to the clean instance URL. After a
used or expired code, the root page can perform a separately rate-limited
rotation that writes the replacement only to the terminal. A lost scoped
shell cookie redirects to that recovery page. Only the exact instance root can
perform these exchanges. Browser routes
remain under that path so sibling loopback ports do not
receive or replace the capability cookie; canonical API paths accept bearer
authentication for local clients. Exact active Host and optional Origin
validation, no CORS, strict bounded JSON, security headers, and a single server
lock protect the shell and versioned internal API.

The A13 server exposes only `/api/v2/snapshot`, `/api/v2/round`,
`/api/v2/reopen`, and `/api/v2/approve`; v1 routes are retired. It caches each engine snapshot and
computes a `sha256:<hex>` revision over the snapshot, completion state,
workspace status, and optional write result. A mutation
must present that exact revision; concurrent same-revision requests admit one
winner. The server constructs human answers from only question ID and value so
the engine remains authoritative for frontier slots and evidence source. A
successful final or incomplete atomic write freezes mutation while retaining
inspection. Transactional engine results remove post-error cache recovery as a
transport concern: rejection leaves the cache usable, while workspace status
is re-inspected on reads and mutations. Conditional snapshots return `304` for
an unchanged revision. Strict request decoding rejects invalid UTF-8,
recursive duplicate names, unknown fields, multiple documents, unsupported
charsets, and over-limit bodies with distinct statuses. Error envelopes carry
retryability, request ID, and the current revision for authenticated state
errors. Only sanitized 500-class causes are logged. The embedded
HTML/JavaScript/CSS shell polls while visible, backs off after errors, preserves
cached JSON, and displays revision, refresh time, sources and discovery
blockers, candidate workflows, prompt-safe review evidence, readiness, top
issue, structured current-frontier controls, revisable settled answers,
preview, proposed actions, write conflicts, completion, and workspace drift.
It has no extractor dependency or in-browser LLM invocation; CLI-created
drafts can resume through the same engine and unconfirmed classifications.

A10 renders every current-frontier question as a required accessible form
control and submits the whole round with the displayed revision. Recommendations
are explicit fill actions, not implicit answers. The review surface shows the
exact previews and read-only overwrite-conflict preflight before enabling
separate final and incomplete approval controls. Review and overwrite
acknowledgements are independent. The shared writer validates the complete
prepared plan before any filesystem mutation, rejects ambiguous duplicate,
case-folded, ancestor/descendant, or remove/write paths, and reserves `.icot/**`,
`project.md`, and both intent paths from source materialization. The client
announces mutation status politely and restores successful-mutation focus to
the next question, proposal-review heading, or completion banner. It never automatically retries a
mutation: domain rejection remains editable, retryable transport/operational
failure reconciles before offering an explicit retry, stale snapshots preserve
unsent input until explicit adoption, drift requires restart, indeterminate
failure locks mutation, and completion remains frozen and inspectable.

A11's implementation qualifies those behaviors through a build-tagged Playwright-Go suite that
launches real Chromium against the actual loopback listener. The suite covers
accessible naming and keyboard order, full rounds, preview/conflict approval,
both completion modes, stale/drift/retry/freeze state, polling/visibility/304,
and narrow plus 200-percent layout. That suite imports Playwright directly;
A16 also links it transitively into the release executable for the hidden
worker, while keeping it out of the engine and server processes. The release
runner requires and logs Chromium sandboxing, rejects the disable override,
and passed all 13 journeys under sandbox-compatible user namespaces during
E09. The separately named unsandboxed target remains diagnostic only. The API
remains experimental local coordination, not a published schema or supported
remote service.

A15 extends the engine with revision-protected acquisition mutations. A
journey starter and goal become human decision evidence; API-family uploads
enter a mode-`0700` private-root inbox through a 20 MiB bound, secret scan, and
exactly-one Apitools candidate check before explicit atomic staging; the UI may
remove only unchanged files recorded in its own staged-source registry.
Browser capture staging accepts only an independently validated profile pair
and safe review collection, creates collision-free targets, and refreshes
source discovery inside the same engine mutation. The v3 review collection
migrates one fully safe v2 singleton only when its ID is exactly
`legacy-<12 digest hex>`, its legacy targets are empty, and all timestamp,
profile/envelope digest, evidence, and decision invariants pass. Source removal
and browser staging observe the workspace before semantic reads and repeat
target, digest, append, and absence checks inside the replacement callback
after fingerprint comparison. Browser staging always requires the
engine-configured private root, independent of transport behavior. Capture staging deliberately
does not write final browser source/authentication selection metadata; ordinary
workflow approval owns that evidence.

A16 replaces the retired API v2 transport with experimental
`openudon.icot-ui-api.v3`. It keeps one authoring revision for journey/source/
interview/write/resume/package mutations, a separate capture revision for
asynchronous Browsertools events, and an ETag over the complete displayed
state. The UI stabilizes its own executable beneath the private root and
re-executes a hidden Browsertools worker in a separate process group. A shared
`internal/icot/browserauthor` coordinator serves both UI and terminal paths;
the engine and HTTP server do not initialize Playwright in-process. Only one
capture may run, snapshots remain available throughout readiness and capture,
and authoring/package mutations are blocked until it is terminal.

The v3 shell exposes only reduced candidate, exact approval, credential/MFA
checkpoint, completion, and bounded output structures. Credential/challenge
values, cookies, storage, raw protocol output, child stderr, private result
paths, signing material, and runtime credentials have no HTTP representation.
Final authoring approval enters `authored`. A separate two-minute deterministic
build followed by non-writing assessment either enters `package_failed`, from
which explicit resume requires complete reapproval, or `handoff_ready`, where
only bounded closed-allowlist artifacts, quality, digests, symbolic bindings,
approval requirements, and exact approval-template argv are inspectable.
There is no UI route for registration authoring or execution, approval generation, credentials,
trusted run, or workflow execution.

The stabilized iCoT worker copy is content-bound across source-before/source-
after/destination hashing and mode `0500`. Browsertools' full doctor report is
retained only for CLI diagnostics; UI state, ETags, and HTTP serialization use
the separate path-free doctor shape on both success and failure. A failed
initial revision calculation rolls back the preflight transition, preventing a
nil-session wedge. Strict JSON accepts at most 64 nested containers.

Process containment sweeps after normal leader exit as well as cancellation.
Linux tracks descendant PID/start-time identities through `/proc`, terminates
detached `setpgid`/`setsid` children, verifies their exit, uses a process-group
kill only while the immutable group-leader identity remains live, and avoids
PID-reuse kills after reap. Other Unix systems retain process-group cleanup and Windows retains
task-tree cleanup as platform-qualified fallbacks.

OpenUdon owns workflow-specific graph construction, prompts, intent schema, v2 wires, source staging,
repair rules, proposal lifecycle, reports, and trusted handoff. `../authoring` owns the generic graph,
frontier-round engine, interview transaction binding, prompt/lifecycle mechanics, and shared public
authoring contracts. `../apitools` owns source discovery, validation, metadata, operation and
lifecycle-role ranking, catalog references, and remote
search primitives. `../browsertools` owns browser source validation, cache,
bundles, discovery, and static registry mechanics. `tfconfig` remains unchanged
because Terraform facts are outside this workflow
authoring boundary.

## Review Handoff And Trusted Execution

The OpenUdon-owned handoff package gives reviewers or external orchestration enough evidence to
route review, but OpenUdon does not implement managed reviewer identity or audit history.

The local trusted runner is intentionally separate from synthesis. It validates
`expected/review-handoff.json`, `expected/quality.json`, current in-memory quality, approval JSON,
canonical package digest, and tier compatibility. The package digest uses OpenUdon-local handoff digest
helpers over OpenUdon's required input set, including every regular file under `openapi/`,
`google-discovery/`, `aws-smithy/`, `asyncapi/`, `graphql/`, `openrpc/`, `grpc-protobuf/`, `odata/`,
legacy `discovery/`, and reviewed runtime data files when
present.
`internal/packageartifacts` owns the required package inventory, safe relative path validation,
manifest-required path normalization, regular-file checks, digest input construction, and staging
input construction. Symlinked, directory, special-file, unsafe relative, and unstated required
handoff inputs are rejected before approval can authorize execution. It rejects credential values
in artifacts and direct production execution. One immutable manifest-bound
snapshot reads every required input once; declared input digests, stored
quality, package/handoff digests, plan/intent/review/profile parsing,
credential/session mappings, protocol choice, and run config all derive from
those same bytes. It then writes a non-secret `openudon.executor-run.v2`
config with sorted `package_paths`, a unique run ID, and exact package,
handoff, and approval digests. Dry runs and real handoffs stage every declared
package path into a unique run workdir, recompute `package_sha256` from current
source files in the staged copy, and fail before executor invocation if those
bytes drifted from the validated snapshot or the staged digest differs from
the approval digest. Dry runs stop after staging and write non-secret `openudon.run-evidence.v2` with gate
outcomes, staged paths, stage kind, package paths, credential binding names,
config/handoff/approval/package digests, verified executor-report path/digest/size, and workdir-relative
digest references to `async-evidence.json`. `openudon run` also prints the resolved sidecar path for
operator convenience. The async sidecar is an
`openudon.async-evidence-bundle.v1` wrapper over neutral Evidence async request/response records for
OpenUdon package/run forwarding only. It does not carry credential values, raw executor
stdout/stderr, Ramen resource addresses, desired hashes, convergence outcomes, or state semantics.
When an outer `OPENUDON_UDON_RUNNER` override is used, evidence marks the staged path as
`stage_kind: preflight` because the external runner owns final executor-visible staging. It receives
the config path, config digest, and approval path, rebuilds current validated
state, and requires byte-identical canonical config before execution. Direct
`cmd/udon-runner` rejects v1 and also fails closed when `package_sha256` or
`package_paths` are missing, or when `direct_production_run` is true. Real handoffs then invoke udon through `OPENUDON_EXECUTOR`, either
as an absolute binary path or `docker://<image>`. Typed invocations pass local
executors only declared `UDON_CREDENTIAL_*` values. Docker `-e` receives only
declared credentials and session bindings; approved browser-driver environment
names resolve from container defaults and host desktop/socket requirements are
rejected. Outer runners additionally receive explicit `OPENUDON_UDON_BIN` and
`OPENUDON_UDON_IMAGE` overrides. Cloud, proxy, SSH-agent, and unrelated
variables do not cross the boundary.
`openudon run-evidence verify --file run-evidence.json` verifies the archived run evidence shape,
relative sidecar paths, sidecar SHA-256 digests, record counts, and neutral async request/response
records. Udon M35 defines the current non-secret output contract as strict
`udon.execution-report.v2`: failed reports require one closed code and
successful reports contain none. OpenUdon translates only valid reports into
neutral async status and confirmation-read observations; missing, malformed,
v1, unknown, or unrelated failures are `unclassified` and cannot satisfy an
expected-failure scenario. Raw executor stdout/stderr,
credential values, provider state, and Ramen convergence semantics remain outside OpenUdon evidence.
An optional detached Ed25519 signature binds the exact v2 evidence bytes. Its
embedded public key proves integrity; a separately configured trusted PKIX key
is required to claim operator identity. Private signing keys never enter run
configs, evidence, argv, or child environments.

Browser workflows use the same v2 execution boundary. OpenUdon parses the
snapshot's plan, nested intent, packaged browser and authentication profiles,
and strict review files to derive the oldest required Browserdriver protocol,
canonical `UDON_CREDENTIAL_*` and external `UDON_BROWSER_SESSION_*` mappings,
and exact operation/authentication approvals. The operator supplies only an
absolute reviewed driver path and optional non-secret arguments. That
value-free browser contract is embedded in config and evidence, rendered into
Udon's public CLI flags for local or Docker execution, and independently
re-derived by an external runner before invocation. Browser credentials are
part of the package handoff inventory and only their declared values cross the
executor environment allowlist. Docker execution validates and bind-mounts the
host driver read-only at `/openudon/browser-driver`, then passes that container
path to Udon. Browser profiles retained only as API-first fallback evidence do
not activate runtime browser configuration.

Required handoff inputs are `project.md`, `workflows/intent.hcl`, `workflows/workflow.hcl`,
`workflows/workflow.uws.yaml`, `expected/plan.json`, `expected/quality.json`,
`expected/refinement.json`, `expected/review.md`, `expected/review-handoff.json`, optional
`expected/data.hcl`, and any staged API source file under `openapi/...`, `google-discovery/...`,
`aws-smithy/...`, `asyncapi/...`, `graphql/...`, `openrpc/...`, `grpc-protobuf/...`, `odata/...`, or legacy `discovery/...`. Approval states are `generated`, `validated`,
`review_required`, `approved_for_sandbox`,
`approved_for_production`, and `rejected`.

Automation tiers:

| Tier | Gate | Location |
| --- | --- | --- |
| Public module CI | reject local replaces, `GOWORK=off go mod download`, vet, test, boundary and whitespace checks, plus six-target cross-builds | GitHub Actions without local siblings or provider credentials. |
| Public tag release | repeat standalone gates, package three CLIs for six targets, run the credential-free archive smoke, write checksums, publish release | GitHub Actions on annotated `v*` tags. |
| Local deterministic | `go test ./...`, `go vet ./...`, `make check`, `git diff --check` | Trusted workstation with public OpenUdon siblings. |
| Local readiness report | `openudon readiness --run-gates --out eval/readiness/local.json` | Trusted workstation with public OpenUdon siblings. |
| Local/manual real LLM | `openudon eval --release-gate` or `make release-eval` | Trusted workstation with provider env vars. |
| Future protected real-provider automation | New design required | Protected runner only after checkout and redaction controls stabilize. |

## Planned File And Folder Structure

- `cmd/openudon/`: thin CLI for check, synthesize, build, promote, assess, eval, readiness, approval
  template, and trusted run commands.
- `cmd/icot/`: guided OpenUdon authoring CLI.
- `internal/icot/engine/`: driver-agnostic iCoT session, snapshot, frontier
  round, preview, autosave, and explicit-approval lifecycle.
- `internal/icot/ui/`: stdlib-only production loopback HTTP transport,
  capability-token bootstrap, revision/freeze state, embedded plain
  HTML/CSS/JavaScript authoring assets, and build-tagged real-browser tests.
- `internal/icot/artifactwriter/`: shared terminal/engine source
  revalidation, browser review metadata, draft cleanup, and atomic artifact
  transaction.
- `internal/synthesize/`: artifact generation, expected plans, quality gates, review evidence,
  refinement loop, and review handoff manifests.
- `internal/icot/`: interactive authoring session, reconcile, lint, replay, extraction, and prompt
  handling.
- `internal/eval/`: fixture eval, reference comparison, run comparison, reporting, release gates,
  and provider drift watch.
- `internal/browserintegrationeval/`: fixed cross-repository browser boundary
  matrix plus strict value-free report writing and verification.
- `internal/browsertransactioneval/`: canonical cross-package BAP+BCP/BRP
  lifecycle qualification evidence and independent verification.
- `internal/trustedrunner/`: approval schema, package digest, handoff validation, tier checks, and
  udon invocation wrapper.
- `internal/readiness/`: local optional sibling checkout readiness reports and deterministic gate execution.
- `internal/workflowintent/`: OpenUdon compatibility adapter over local authoring concepts and
  optional apitools catalog advisory metadata.
- `examples/`: committed examples and eval corpus.
- `templates/`: project brief starter templates.
- `tabilet/memory-bank/`: living project memory.
- `tabilet/evolution/`: versioned prompt/result snapshots for milestone-level direction changes.

## Security Boundary

Generated UWS, OpenAPI, HCL, review, and approval artifacts are untrusted until validated. OpenUdon
must not put secrets in prompts, examples, eval fixtures, committed artifacts, or logs. Credential
bindings are symbolic names only until a trusted runtime resolves them. Production side effects are
never allowed from agent sessions, synthesis, build, promote, assess, iCoT, or eval.
The local iCoT UI adds no execution authority: it is IPv4-loopback-only, has
one ephemeral capability token and one human operator, emits no CORS headers,
and cannot bind a LAN address, host multiple sessions, or execute workflows.
Existing-account authoring capture is its one browser-launch authority: it
starts only the isolated bundled Browsertools worker under the typed,
parent-attested disclosure boundary described above and never initializes
Playwright in the engine or HTTP server. Its HTTP server caps headers at 32
KiB, uses five-second header, 15-second read, and 30-second idle timeouts, and
intentionally has no global write timeout for potentially long reviewed-source
refreshes.
Registration support does not widen that browser-launch authority: A18 accepts
only already-reviewed local, credential-free artifacts and performs no browser
or network operation.
## Harness Layout

Private planning uses permanent `status-<LANE><NN>.md` ledgers. `B` preserves
the pre-numbered bootstrap, `M` preserves legacy/cross-cutting work, `A` owns
iCoT/intent authoring, `P` owns package/review/quality/handoff, and `E` owns
eval/scorecard/release evidence. The unattended runner reads the second column
of `Item | State | Notes` tables. Candidates remain unnumbered.

## M79 application control and aggregate evidence

Registration application methods own the existing one-attempt state, separate
revisions, draft decisions, worker consumption and global containment latch.
HTTP adapters and supervised NDJSON control call those methods. The control
transport owns bounded private streams and cancellation; it is not another
browser executor. Actual UI qualification now drives DOM controls and shipped
JavaScript through the real worker and package handoff.

The browser-system evaluator composes the existing scenario and transaction
owners, with source-tree digests for local engineering deltas, exact component
evidence, fixed stages and mandatory three-pass loopback repetition. Its
process supervisor joins children and treats missing prerequisites as failure.
Application request allowlists do not claim network-wide containment.

M79 local acceptance includes exact observed toolchains and all auxiliary Udon
source trees around each loopback stage. Three full eleven-stage passes and
independent digest verification pass; source publication remains separate.

M80/W08 adds opt-in `openudon.application-control.v1` over owned private
pipes. UI and command adapters share capture, authoring, transaction adoption,
package and revision checks. The supervisor can complete BRP and BAP/BCP
authoring through promotion in one application; trusted execution remains
separate. The registration-only protocol stays compatible. See
`docs/application-control.md` and `tabilet/docs/history/status-M80.md`.

M81/W09 forwards explicitly selected private stdin through existing trusted
runners to Udon's human verification boundary. The executor runs from its
reviewed stage, so native file outputs remain at their reported private paths.
Credential scanning exempts only exact declared symbolic names in structured
binding positions; HCL exemptions require literal string syntax and never
exempt provider tokens. Native qualification v2 adds supervised BRP and BAP/BCP
packages while retaining verification of the original v1 inventory.

W8M's external overlay reuses generic application test helpers for its concrete
fixture without importing target code here. Its complete aggregate binds the
native component reports and actual runtime receipts. Three fresh units and
independent verification pass; publication and real operating authority remain
W8M's separate downstream boundary.

Registration attestation v2 adds value-free links to prior native evidence, executor report, prior attestation, reviewed authority and consumed claim. External operating applications own historical proof and persistent claim serialization. OpenUdon retains exact package/profile/operation validation and unchanged executor handoff.

M83 adds private attestation v3 for exactly two prior attempts. Its links bind
the immediate predecessor; the operating consumer verifies the entire earlier
chain and separately consumes the second claim. Existing v1/v2 meanings and
the credential, UWS, run-config and execution-receipt boundaries remain intact.

## E13 development evidence and cache

`internal/browsersystem` owns the closed development-stage runner and input
fingerprint. `internal/browsercheck` owns private immutable artifact copies and
JSONL timing. Development evidence has a distinct schema, no runtime authority,
and at most 24-hour explicit reuse. Cache keys bind dirty source/fixture bytes,
Go dependency files, JS modules, checker/toolchain/browser/sandbox bytes and
environment. Cached build artifacts contain only explicit compiler outputs;
browser profiles and private runtime state are excluded. Native qualification
retains its version and fresh three-repeat semantics. W8M owns its aggregate v2
composition and consumer-specific smoke; generic code imports no private Udon
packages or target-specific policy.
