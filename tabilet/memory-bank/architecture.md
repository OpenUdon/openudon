# Architecture

## Approved Stage 11 architecture target — not implemented

[Stage 11](../../../kinet/docs/stage11.md) and [local milestones](milestone.md#stage-11-cross-package-refactoring) define the approved target. OpenUdon:M98 will expose format-neutral public handoff/digest/approval/Authority and bounded trust/evidence verification contracts. Synthesis-coupled v2 construction, assessment and simulation orchestration remain private legacy adapters without a new public compatibility promise. OpenUdon:P09 owns supported public v3 construction and source/shape verification; isolation belongs to the consuming worker. Kinet becomes the primary non-browser authoring product in Phase B. Public OpenUdon never imports private Udon modules; browser-dependent compatibility code remains until the Stage 12 closure gate.
Both phases belong to one stage. Current facts below remain the observed implementation; no new acceptance, publication or installed behavior is claimed. The installed Kinet M44 service remains unchanged, and Stage 12 owns the browser-dependent removal gates.

## Stage 9 broker handoff — accepted 2026-10-05

[M97](../docs/history/status-M97.md) accepted concrete authority v1, approval v2,
executor config v3 and evidence v4 at qualified source
`f4127c159e18fa66619659bc3c4b8757b7022267`; source-record publication
`55e1be3eb4eb728005d3f51f582f9fceda547b56` was independently verified.
Closing review 2 passed. [Contract](../../docs/broker-execution-handoff.md) and
[exact qualification](../../docs/m97-qualification.md) bind package/input/executor,
ordered operations, symbolic credential revisions and complete artifact hashes.
Fresh native39, offline4, seven actual M46 broker cases, smoke and focused races
passed. Existing schemas/default bytes, sandbox protection and frozen
`c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0` authoring/capture pin remain unchanged.
No private executor import or Kinet network/grant policy is owned here. Kinet
M35/W14/M37 consume this separate qualified profile after exact reconciliation;
no deployment, live service or W8M adoption is established by producer acceptance.

## Versioned browser qualification locks

Scenario, integration and native qualification readers dispatch from each
report's version. Historical v1 and M86 v2 retain their original meanings;
E21 current v3 readers use frozen Udon `6d32d49` compatibility and 14-source
build-input snapshots. E22 advanced the explicit current selector to Browser
1.10 report v4, with the published UWS M05, Browsertools M32, Browserdriver
M15 and Udon M43 pins plus their separate 14-source build closure. Its three
count scenarios stay outside the v3 manifest inventory. All selected sibling
worktrees must match exact commits and be clean before browser work. Historical
qualification reports remain verifiable; their removed UI gates cannot execute
from this source. Use the explicit current selector for fresh loopback evidence. The retained E22/M91 context
used v4 locks and closure; M92/M96 native v5 reports remain frozen. New `--stack current` evidence uses M95 native v6,
scenario/journey v5 and the exact UWS 1.12/M45 v5 input locks; integration
emits v7. Its declared Browsertools UWS edge remains 1.11, with effective UWS
1.12 selected separately. Earlier versioned readers retain their original pins. The v4 integration selector adds named count-profile, producer,
schema, Udon v11 consumer, and Browserdriver extraction markers while the v2
and v3 integration readers keep their frozen gate inventory. Full E22
qualification and bounded review passed on clean OpenUdon
`9be9ff3f195ecaa8bdac88cc8616c5fc345dfeb3`; the reports and review are in the
[E22 history record](../docs/history/status-E22.md).

The v2 and v3 snapshots prevent later current-stack updates from changing the
meaning of retained reports. The E21 repair lineage is in
[status-E21.md](../docs/history/status-E21.md).
BRP's temporary repository-local example parent is removed on every exit when
the qualification created it; pre-existing paths are preserved and symlink
parents are rejected so the per-stage clean-source check remains meaningful.

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
travel only through private runtime environment values. Consumer
authority is enforced by the public capture protocol; issued decisions,
expiry and worker teardown remain native-owned. No private Udon import
or target-specific browser implementation is added.

OpenUdon has a Go module, a thin `cmd/openudon` CLI, closed neutral authoring commands, deterministic
synthesis/build/promote/assess commands, an eval harness, local readiness reporting, and a trusted
runner wrapper. It emits reviewed package artifacts under each example directory and validates those
artifacts before any approved udon execution path.

The v0.2 public boundary is CLI- and artifact-first. Deterministic package,
approval, handoff, and run-evidence commands are supported through v0.2.x;
implementation packages remain internal and are not a supported Go API.
Release archives co-version `openudon` and `udon-runner`, while
`openudon version --json` is the archive's build-metadata authority.

Generated packages now include project briefs, structured intent, workflow HCL, UWS YAML, expected
plans, OpenAPI discovery reports, refinement reports, review notes, quality reports, and
`expected/review-handoff.json` manifests. Executable packages use
`apitools.review-handoff.v2`; each input has a SHA-256 digest and the handoff
self-digest clears its own field before canonical JSON hashing.

## System Boundary

- `internal/registrationdiscovery` retains neutral private inventory and bounded
  revision-history records. OpenUdon exposes no UI/control transport; Kinet
  owns interactive presentation and Browsertools supplies native observations.
  UWS owns reviewed portable flows, fields, steps and success predicates; its
  optional 1.1 discovery metadata remains readable. New registration recipes
  omit inventory metadata. Selection grants no runtime, browser or retry authority.
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
- OpenUdon's non-interactive step-authoring CLI consumes APItools' published operation-candidate
  contract, now adopted at published APItools M81/M80 revision
  `fb132631c9827eae5f2ec4503d03f21eabfb4113`. `step candidates` scans bounded local family directories and returns path-free exact
  source/digest references, consumer summaries, match evidence, auth alternatives, effects, and
  capability gaps. `step check` revalidates that exact operation and effect against current local
  bytes, including source request locations, colliding unqualified names,
  self-reference, and dependency-cycle checks; `step bind` applies the same
  mapping gate before writing one selected step through the shared atomic
  artifact writer and rejects a cyclic prerequisite graph;
  direct mapped workflow inputs are checked against contract and source type
  and requiredness, while unproven expressions stay indeterminate and cannot
  be bound;
  an unrecognized leading operation action stays unknown, and nullability is
  reported for selected response outputs and their schema ancestors without
  downgrading unrelated sibling outputs;
  `step source add` accepts exact local file paths and approved SHA-256 values,
  validates all selected documents through APItools without network fetching,
  and atomically writes create-only source files plus
  `expected/api-source-manifest.json` under the workflow package. The manifest
  is optimistic-revision-bound, content-digest-checked, and included in the
  required package handoff inventory. Results contain only package-relative
  paths and both the caller-selected manifest ID and path-derived candidate ID.
  Kinet owns the user confirmation that precedes this package write;
  inline credential references share explicit binding symbol validation,
  including rejection of reserved `none` and `clear` values;
  `flow-review` is advisory and read-only. These commands neither fetch source URLs nor resolve
  credentials, invoke API operations, approve packages, or execute workflows. Kinet owns external
  orchestration and its user-confirmation ledger.
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
families in new UWS 1.12.0 documents for reviewed local artifacts backed by source-aware apitools metadata, while downstream
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
| Content trust | Allowed only through an explicit operator-authored registry. It requires UWS 1.9.1 or later; newly generated workflows declare UWS 1.12.0 and existing packages retain their declared versions. Assessment explicitly invokes UWS analysis, using Browsertools for contained browser-profile contracts, and emits warning-only quality/review evidence without entering ordinary validation or execution. |

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

## Neutral authoring boundary (M95)

OpenUdon provides external CLI and artifact contracts. Kinet owns interactive
interviews, chat, browser UI, approvals and user-ledger publication. OpenUdon
contains no iCoT entry point, application HTTP/control transport or embedded UI.
Removal is accepted in M95 at application
`c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0`, review3 passed. Kinet M20 and
W8M W29 still own exact final consumer adoption; native/delta evidence retains
its actual sources in M95's record.

`openudon authoring draft` is a closed seeded/local authoring adapter over the
single `authoringengine`, `elicitor` and `artifactwriter` implementation.
It never reads terminal interview answers. Partial inputs return the structured
frontier; `--print` is read-only and artifact publication requires `--yes`.
`--from-example --prompt-mode fast` retains deterministic corpus defaults through
the existing neutral elicitor without network/model calls or an autosaved draft.
Source validation, symbolic credential policy, atomic writes and rollback rules
remain authoritative. Generic frontier mechanics belong to Authoring's public
`engine`; no copied interview engine is introduced.

`openudon authoring browser-plan` emits an inert bounded-capture plan.
`openudon authoring registration-draft` constructs reviewed typed/conditional
field definitions through the one pure `internal/registrationdraft` builder.
It accepts no credential values, launches no browser, writes no package and
makes unobserved success proof explicitly deferred. See
[registration draft](../../docs/registration-draft.md).

Interactive acquisition uses the public supervised `browser-capture` protocol.
Browsertools owns its isolated Chromium worker, parent-attested origin/action
policy and reduced observations. Exact issued decisions and separate review,
finish, import, package preparation and promotion remain distinct gates.
`browser-author` and `package` commands validate the original reviewed capture
receipt; a caller cannot relabel a reviewed transaction as promoted evidence.
Credentials and runtime sessions never enter authoring state. Trusted execution
remains an external Udon handoff with its separate approval.

Retained expert lint/reconcile/repair/report/variants/scorecard/replay-eval and
explicit authoring-eval commands use the same neutral implementation. Historical
`.icot` artifact names and report schema labels remain byte compatible; they do
not expose a retired transport. Older qualification readers and locks preserve
their recorded meaning. Authoring and udon-ui package retirement is deferred.

### Retained core policy

Source discovery remains bounded to explicit example/API roots, rejects symlinks
and ambiguous or truncated inventories before publication, deduplicates by digest,
and defaults to10,000 entries/100 documents/20MiB per file. API sources remain
preferred; browser profile/action selection precedes mappings and separately
reviewed mutating actions. Remote acquisition retains explicit authority,
DNS-pinned safe destinations and independent digest validation.

Security alternatives remain OR-of-AND sets with one stable fingerprinted choice;
credential bindings are symbolic and alternatives are never silently unioned.
Generic Authoring frontier nodes retain settled/open/deferred/inapplicable states,
atomic complete rounds and explicit blockers. Boundary/side-effect posture cannot
be deferred. Neutral authoringengine keeps deep-cloned snapshots, exact-byte
workspace fingerprints, prepared atomic write plans and rollback/indeterminate
results. Same-size or restored-mtime changes are still drift; cached inspection
never authorizes mutation after conflict. Artifact writer revalidates sources and
credential scanning before publication. Current draft adapters do not expose an
interactive reader, HTTP server or persistent workspace lease.

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

- `cmd/openudon/`: public authoring, capture, build, review, package and run CLI.
- `cmd/udon-runner/`: external executor handoff wrapper.
- `internal/authoringcli/`, `authoringengine/`, `elicitor/`, `artifactwriter/`:
  closed expert adapters, neutral authoring lifecycle and transactional writer.
- `internal/authoring/`: adapters over Authoring's public engine.
- `internal/browserauthor/`, `browserauthoring/`, `browsercapture/`:
  native capture controllers, worker dispatch and issued-decision protocol.
- `internal/registrationdraft/`: single pure reviewed registration definition builder.
- `internal/capturequalification/`: synthetic fixture adapters over actual public
  capture/browser-author/package commands; no product UI or production authority.
- `internal/browserpackage/`, `packagepipeline/`, `browsertransaction/`:
  reviewed receipt validation, immutable package lifecycle and digests.
- `internal/synthesize/`, `trustedrunner/`: deterministic generation and separate
  approval/credential/executor boundary.
- `internal/eval/`, `browserintegrationeval/`, `browsersystem/`,
  `browserscenario/`, `browsertransactioneval/`: neutral evaluation, versioned
  historical verifiers and explicit fresh current qualification.
- `examples/`, `templates/`: retained package corpus and starter briefs.
- `tabilet/memory-bank/`, `tabilet/evolution/`: current truth and approved direction.

## Security Boundary

Generated UWS, source documents, HCL, review and approval artifacts remain
untrusted until validated. Credential bindings are symbolic names; values are
resolved only behind the trusted executor's separate approval. Authoring,
synthesis, simulation, assessment and package promotion grant no runtime or
live target authority. Public capture accepts only exact issued decisions,
validated origins and bounded actions under parent-attested worker containment.
No OpenUdon HTTP listener, UI capability cookie or generic control server remains.
Kinet owns its browser access and user authorization. Native registration success
is explicitly deferred until separately approved execution proves it.

## Harness Layout

Private planning uses permanent `status-<LANE><NN>.md` ledgers. `B` preserves
the pre-numbered bootstrap, `M` preserves legacy/cross-cutting work, `A` owns
iCoT/intent authoring, `P` owns package/review/quality/handoff, and `E` owns
eval/scorecard/release evidence. The unattended runner reads the second column
of `Item | State | Notes` tables. Candidates remain unnumbered.

## Native aggregate evidence and retained attestation

The native evaluator composes scenario/transaction owners with exact source,
component and build/tool identities. Each current loopback qualification runs
three fresh thirteen-stage passes, with no reuse or skipped prerequisite.
M95 v6 replaces removed UI/control stages with actual public capture/package
journeys and pure registration definitions; older native v1–v5 readers keep
original inventories. Historical UI execution is unavailable from this source.
Process owners join their workers and treat teardown/prerequisite failure as failure.

`openudon run --interactive-browser` forwards explicitly selected private stdin
to Udon's existing human verification boundary. Default runs have no input stream.
Exact declared symbolic credential positions remain exempt from scanning;
provider tokens never do. External operating consumers own persistent claims and
prior-attempt proof. Registration attestation v1/v2/v3 contracts, exact profile
validation and executor handoff are unchanged. W8M qualifies final consumer pins
separately; producer evidence alone grants no real operating authority.

## E13 development evidence and cache

`internal/browsersystem` owns the closed development-stage runner and input
fingerprint. `internal/browsercheck` owns private immutable artifact copies and
JSONL timing. Development evidence has a distinct schema, no runtime authority,
and at most 24-hour explicit reuse. Cache keys bind dirty source/fixture bytes,
Go dependency files, JS modules, checker/toolchain/browser/sandbox bytes and
environment. Cached build artifacts contain only explicit compiler outputs;
browser profiles and private runtime state are excluded. Native qualification
uses M95 v6 while retaining old readers and fresh three-repeat semantics. W8M owns its aggregate v2
composition and consumer-specific smoke; generic code imports no private Udon
packages or target-specific policy.

Native input inventories belong to OpenUdon. The explicit current-stack v2 input mode binds the supplied external Browserdriver bundle and current nineteen-source closure; external consumers own cache publication, reuse policy and fresh consumer acceptance.

E24 additionally binds actual process user, mount, network, PID, UTS, IPC, cgroup and available time namespace identities before/after current input inventory. Values are hashed only. Legacy input v1 and completed E23 history remain unchanged.

## Explicit per-step execution evidence (M90)

Report-v5 observations bind exact attempt, staged workflow bytes and ordered
flat HTTP inventory. Rejected/missing reports produce fixed-class unknown
observations, never unstarted proof. Run-evidence v3 is explicit opt-in; private
Udon remains an external CLI.

M90.2 implements this in `internal/udonreport` (independent wire/shape
validation), `internal/udonrunner` (pre-dispatch staged inventory and explicit
v5 flags) and `internal/trustedrunner` (v3 binding, conservative uncertainty,
signature/archive verification). Canonical external run-config revalidation
preserves explicit v5 only for HTTP-only packages. Report validation never
imports a private executor package or decides a downstream retry.

## Shared authoring implementation (M91/M95)

M91 extracted the shared implementations; M95 removes only the legacy
transports. The [neutral authoring boundary](#neutral-authoring-boundary-m95)
defines the single draft, registration definitions, capture and package owners.
No second interview, writer, browser engine or UI implementation is introduced.

## M92.1 version-preserving authoring

Synthesis reads declared versions from bounded regular existing HCL/export
artifacts before refinement/discovery writes. Conflicting declarations refuse
without rewriting the package. New packages default to public UWS 1.12.0;
existing declared versions retain their generation and approval path. Browser
qualification explicitly carries the immutable manifest version into the same
synthesis implementation; it cannot overwrite a different existing declaration.
External executor compatibility uses M45's verified frozen binary and build
closure, including report-v5 evidence for both retained 1.11 and new 1.12
packages. M92 records passed fresh three-repeat browser qualification at its exact runtime checkpoint and separately verified final simulation producer revision.

M92.2 shares declared-version reading in `uwsexec` across synthesis and pending
commands. Pending intent schema field sets are JSON strings in HCL, decoded
into the public `uws1.PendingStep`; generation uses its native UWS shape and no
executable operation. Pending-only authoring skips API discovery without
changing the project brief or later source/quality checks. A new additive
`openudon.step-pending.v1` envelope uses revision-checked atomic intent writes
and existing scaffolds/dependency checks. Resolution uses the existing bind
path, requires the exact pending contract and removes that pending block.

Assess distinguishes pending contracts in HCL and exported UWS. Trusted runner
admission independently decodes both captured artifacts before stored quality
or an injected assessor can authorize anything, and uses public executable
validation when pending contracts exist. This includes unused workflows and
unselected branches. Refusal produces no approval, staging, credential lookup
or executor dispatch. The unchanged legacy version retains its wire shape;
confirmed 1.12 effects are additive and descriptive.

M92.3's `internal/simulation` captures bounded regular package files and uses
shared review-handoff digest and public UWS decoders. Captured HCL/YAML must
agree; empty operation inventories normalize the public decoder nil/empty
difference. Pending operations exist only in memory. Legacy request expression
wrappers and received_body outputs adapt to public expressions; the public
orchestrator/mock runtime owns all scheduling, branching, loops and evaluation.
No network, browser worker, executor or credential resolver is connected.
Explicit fixtures/examples/schemas supply responses; original package files
and digest are checked again after computation. Fixed diagnostics and bounded
redacted shapes are exported as `openudon.simulate.v1`. This is preview evidence,
not approval or real-run evidence; M92 records final producer conformance and the distinct native-runtime qualification context.

## M92.4 conformance and qualification context

New current browser evidence selects scenario/journey v5, integration v6 and
native system v5. Exact UWS 1.12/M45 locks preserve old report readers and their
frozen contexts. The new lock separately records Browsertools' retained older
declared UWS edge and the effective UWS 1.12 module; it never rewrites that
sibling's go.mod. Fresh-package qualification advances only declared UWS versions
while retaining all fixture journey semantics. Published pending/simulation
schemas and examples are checked offline against actual owner output, including
package digest and revision-bound pending intent writes. M92 records observed
producer and browser qualification with exact source identities. The final
pure-simulation namespace fix does not change browser/authoring/executor code
or pins; native evidence retains its original source identity.

## M93.1 supervising capture envelope foundation

`internal/browsercapture` implements `openudon.browser-capture.v1` as a
process-local single-owner gate above the existing neutral browser controllers.
A new session has random IDs, one mode, a fixed deadline, monotonic event
revisions and bounded transient reduced views. A typed proposal executes
nothing: it produces an exact server-held review card; approval/refusal names
that issued event/action/digest and consumes it once. Changed or unpublishable
worker state invalidates earlier authority. Terminal results, cancellation and
expiry cannot restore or replay a command. The existing controllers retain
semantic validation, origin/action/verification gates and private browser input.

The closed wire imports Browsertools reduced record types, keeps full
registration history controller-local, and exposes no worker-result paths,
attestations, raw page/browser state or credential/code values. Model-disclosure
proposals bind the current observation and supply no model invocation themselves.
Kinet owns transient UI/Ask handling and its A10/W09 persistence projection.
The exact published schema is embedded and enforced before typed records are
interpreted. This foundation alone exposes no capture CLI or completed journey;
M93.2–M93.5 own adapters, profiles/worker handoff and acceptance.

M93.2 adds the authenticated/TOTP adapter over `browserauthor.Session` and a
single-owner closeable-stream driver reusable by both modes. Controller-owned
pure decision checks reuse its existing observation/checkpoint conversions;
actual dispatch retains native worker and parent-attestation checks. No worker,
profile validator or model client is duplicated. A proposal never responds to
the controller; only the exact one-use approved command does. Worker-issued
origin/action approvals remain independent checkpoints. Disclosure consent is
an exact observation event, not a model call or persistent session grant.

The adapter closes and joins its reader and drains controller events through
joined worker closure on EOF, malformed/stale input, cancellation, absolute
expiry or output failure. Late teardown failure overrides a nominal cancel.
A joined capture emits terminal state `captured` with no profile metadata;
its private result/attestation has no JSON representation and remains input
to M93.4's independent review/package lifecycle. M93.3 adds registration;
M93.4 adds command/worker embedding/import, with M93.5 owning fresh
browser and human-visible acceptance of both journeys.

The registration adapter consumes an already reviewed fixed initial authority
and uses the selected existing no-submit controller; v4 includes native verification,
public preview and canonical profile/history validation. Current-state proposal
checks use an immutable controller-owned snapshot; no consumer reconstructs
the registration state machine or resends full history. Only current observation
and latest preview cross the stream. GET/HEAD navigation stays within approved
origins; exact verification refusal sends no command. Native reduced terminal
diagnostics are retained, with containment failures taking precedence.

The shared driver reads the native registration controller's retained terminal
outcome after joined closure; a dropped terminal event cannot hide a candidate
or late containment failure. Registration now joins its protocol reader and
private cleanup before closing its event stream. A native subprocess check
holds private cleanup and proves the stream cannot close early. Private
candidates stay in-process with no wire representation until independent
M93.4 package admission. M95 removes the old iCoT transports; the public
capture and package commands retain the same native authority.


M93.4 exposes a digest-approved reviewed start file and embeds the shared hidden
worker in the main CLI. Both capture modes retain the same stream, one-use gate,
reader and absolute deadline through a separate post-join import phase. Native
attested reconstruction/adoption, virtual-source validation, workspace fingerprint
and atomic authoring writer own source identity, drift/expiry, targets and rollback.
A result binds the held reviewed transaction; approval imports profiles/reviews
and a metadata-only native transaction/file receipt together. Conservative write
effects include login/submission recipes. This is local authoring admission,
not full-package promotion or execution authority. Only a committed import emits
terminal imported metadata; refusal/cancel/EOF/expiry writes no profiles. A
failed write or lost terminal delivery requires inspection, never automatic replay.
M93 native/visible qualification, explicit human acceptance and publication
are complete; see [its permanent record](../docs/history/status-M93.md).
Kinet and W8M retain their independent adoption qualifications.

## M94 catalog discovery adapter

`step discover` returns the native APItools catalog-discovery/v1 report from
its indexed discovery/ranking implementation. Root/registry/index, optional
installation catalog metadata and remote capability are selected only through
trusted CLI configuration; requests retain the native bounded decoder plus
OpenUdon's shared duplicate-key check. Read-only registration access uses
APItools sqlitecache; no implicit roots, index writes or copied ranking.
All five outcomes/coverage/reference/license facts remain unchanged. Only
scoped no-qualifying-api can support automatic browser fallback; the adapter
never performs routing, confirmation or execution. Optional remote retrieval
requires both installation and request opt-in under native bounds.

`step source add --catalog` consumes the closed additive
`openudon.step-source-catalog.v1` request. Native APItools artifact-scoped export
prepares only confirmed references in disposable private staging. OpenUdon
independently checks selectors against the exact exported raw source and
disjoint final package/catalog roots, preserves every selected provider link
and applicable advisory overlays, then shares `step source add`'s one atomic
writer for raw sources, optimistic manifest and digest-bound
`openudon.catalog-source-provenance.v1`. No separate writer/ranking/parser or
implicit registration/index mutation was added. Likely concrete credentials
are refused without rewriting the raw bytes. Catalog mode results add only a
provenance path to the source-add result shape; local v1 remains unchanged.

## Reviewed capture package authoring (M96)

The public `browser-author plan/apply` adapter in internal/browserpackage consumes
an exact original approved capture receipt. Closed256KiB requests carry base64
of exact native start bytes, native transaction/receipt identities, symbolic-only
inputs and operation choices. Bounded read-only plans bind full owned package
inventory, exact request/source/review, file actions and generated artifact bytes.
Native candidate/discovery validation and pure neutral elicitor lowering retain
both authenticated/TOTP and inert registration recipes. No UI/controller or
second semantic writer is imported. The existing native writer reports its exact
own transient paths to the pre-replacement inventory/freshness guard; its legacy
callbacks keep their behavior. Separate exact authoring confirmation precedes
native deterministic build. A partial build or lost output preserves authored
state and requires inspection, never blind replay; cleanup uncertainty is explicit.
Original start hashes bind registration capture metadata not retained in its
recipe; native origins/transaction validation remains authoritative. Authentication
also matches native login, dashboard proof and goal review. Receipts are unsigned
local content-addressed evidence, not arbitrary replacement attestations.

Application source eed683f27d448ca96af90e7bc5987967a6cd0335 passed three fresh
native repeats, integration and both-mode public capture/package journeys;
review1 passed. Tests-only partial-build coverage has its own full/race/vet
context. M96 acceptance/source publication is verified through `d77f6d51262d0f311910070bc4a43f662260cd7e`; [retired evidence](../docs/history/status-M96.md) preserves original application/build identities. Consumer delivery remains separate.
Kinet M19 owns external single-use delivery/recovery and its own checks; M95
must retain these commands and legacy public review artifacts after UI removal.
See [the CLI/wire contract](../../docs/browser-package-handoff.md).

## M97 private broker wiring

The additive broker path takes concrete approval v2, publishes value-free executor
config v3, and passes a separate owner-only private transport reference to the
external Udon CLI. It pins and snapshots executor/transport bytes, excludes host
credential/proxy environment values, and records authority/step observation in
evidence v4 with the private argument redacted. Create-only config/evidence and a
durable executor claim preserve uncertain attempts. Read-only broker-inspect
metadata comes through the existing APItools adapter; the host still owns current
grants, credentials and concrete network/request policy. Legacy serialization,
report readers, authoring/capture pins and destination classification are unchanged.
M97's separate broker profile is accepted as above; host consumer adoption remains owner-local.
