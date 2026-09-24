# Product

## Current Browser Authoring

Reviewed structural-query navigation is retained in generated browser profiles;
observed query values remain undisclosed. Registration foreground and private
checkpoint countdowns do not grant live registration authority. Authoring and
package review likewise do not grant live browser or target authority.
[E20](../docs/history/status-E20.md) records the authenticated-authoring qualification lineage,
while [M86](../docs/history/status-M86.md) records the current UWS 1.11 real-browser
qualification. Historical publication and failed-attempt details are preserved
in the [history index](../docs/history/index.md).

## Memory Bank Index

- This file owns product purpose, audience, workflows, scope, and non-goals.
- Use [architecture.md](architecture.md) for system boundaries, data flow, and planned structure.
- Use [tech-stack.md](tech-stack.md) for implementation technologies and dependency constraints.
- Use [milestone.md](milestone.md) for milestones, work sequencing, acceptance criteria, current
  completion state, and the status-file index.

OpenUdon is the public-facing UWS workflow authoring, review, package, and executor-handoff tool. It
can be used directly by operators or under optional external orchestration, and it hands
approved packages to a trusted executor boundary such as the private `udon` runtime.

OpenUdon turns reviewed project briefs into deterministic workflow artifacts: `project.md`,
`workflows/intent.hcl`, `workflows/workflow.hcl`, exported UWS, expected plans, quality reports,
review evidence, refinement reports, package digests, credential policy, and machine-readable
review handoff manifests. It does not own public workflow semantics or generic execution. Those
remain in `../uws` and executor implementations such as `../udon`.

## Product Goal

iCoT can author generic registration 1.1 profiles and UWS calls through guided
field definitions, required/optional/conditional rules, named input checkpoints
and reviewed public wizard previews. Suggestions remain editable and require
explicit confirmation. Runtime values and credentials are entered in Udon's
separate private form after package review, never in iCoT authoring state.
Registration 1.0 remains supported. See [A29](../docs/history/status-A29.md).

Registration discovery is private authoring state in the shared application and
iCoT. Operators maintain possible routes and registration types, record reduced
Browsertools observations, revise limitations and review the inventory. Coverage
and owner review are separate and never promise exhaustive discovery. Selection
prepares editable wizard fields; canonical recipes contain reviewed workflow
definitions without inventory metadata. Private values remain outside packages.
See [A28](../docs/history/status-A28.md); this adds no automatic crawler or live authority.

Make UWS workflow projects authorable, reviewable, packageable, and executable only through a
validated trusted handoff path, with clear evidence for every generated artifact and side-effect
boundary.

## v0.2 Security Migration

OpenUdon v0.2 is a security migration whose supported core is
deterministic UWS validation and package generation, digest-bound approval,
trusted executor handoff, and run-evidence verification/archive/signatures
through the documented `openudon` commands and v2 executable artifacts.
Existing v1 evidence remains read-only inspectable; v1 handoffs and run configs
cannot execute. OpenUdon does not yet expose a supported Go-library API.

The bundled `icot` authoring surfaces and experimental loopback API, LLM/provider behavior, prompt wording,
catalog/eval/readiness/smoke helpers, and exact generated prose remain
experimental before v1.

## Primary Users

- Operators authoring or reviewing UWS API/event-source, browser-profile,
  browser-authentication, and browser-registration workflow projects.
- Optional externally orchestrated agents generating artifacts inside isolated workspaces.
- Reviewers checking side effects, credential bindings, generated plans, quality reports, and
  approval states before execution.
- Runtime operators using `openudon run` to validate approval and package digests before invoking a
  trusted executor.
- OpenUdon maintainers extending prompts, iCoT, eval fixtures, quality gates, and cross-repo glue.

## Core Workflows

1. Author or refine an example brief under `examples/<name>/project.md`.
2. Use iCoT's primary single-workspace loopback UI, or the terminal expert
   fallback, to select a journey, acquire reviewed API/browser sources, select
   one active workflow, preserve later candidates, and approve `project.md`
   plus final or explicitly incomplete workflow intent.
3. Separately build and assess deterministic package bytes; a failed UI build
   may return to authoring only through explicit revision-protected resume and
   repeated authoring approval.
4. Validate API or browser-profile source availability, intent shape, workflow compilation, UWS
   export, expected-plan matching, review evidence, credential policy, and secret scanning.
5. Run eval fixtures to compare prompt/model/pipeline behavior across curated briefs.
6. Generate local readiness evidence for optional sibling checkout state and deterministic gates.
7. Produce approval JSON from the current handoff package digest.
8. Use `openudon run` to validate handoff, stored and current quality, approval state, package digest,
   tier compatibility, and trusted executor invocation.
9. Optionally surface catalog-derived provider, spec, and auth/security advice in review evidence
   while preserving explicit local API source inputs as authoritative project context.
10. Run the consolidated local release-evidence flow to build sibling udon,
    execute provider-free smoke, archive and verify run evidence, draft
    release-note evidence, and write compact summaries when preparing release
    or handoff evidence.

## Core Concepts

- **Project brief** is the human policy and workflow source in `project.md`.
- **Intent** is OpenUdon's structured `workflows/intent.hcl` contract for workflow metadata, inputs,
  steps, outputs, data-flow hints, credentials, runtime approvals, side-effect scope, timeouts, and
  idempotency metadata.
- **Content-trust intent** is the optional operator-authored provenance portion
  of intent. It names reviewed source paths, leaf-operation outputs, triggers,
  and external `main` workflow inputs using UWS levels `unknown`, `trusted`, or
  `untrusted`. It requires UWS 1.9.1 or later; new workflows declare UWS 1.11.0.
  It does not authorize execution, clear
  attacker control, or replace package approval and runtime policy.
- **Content-trust analysis** is an explicit assessment-only UWS pass for
  packages with that registry. Browser operations use the contained
  Browsertools profile contract; stable findings become non-failing quality
  warnings and value-free review evidence. Analyzer severity never grants or
  denies trusted-runner authority.
- **Workflow HCL** is one full UWS Document serialization generated from intent.
- **UWS YAML** is an equivalent full UWS Document serialization of the same workflow. OpenUdon treats
  `workflow.hcl` and `workflow.uws.yaml` as public UWS documents, not separate semantic layers.
- **Runtime data file** is reviewed non-secret execution input evidence, emitted as
  `expected/data.hcl` when declared workflow inputs need concrete run values.
  It may also contain env-reference markers such as `ENVIRONMENT:NAME` for
  values that must stay in the operator environment.
- **Expected plan** records inferred steps, runtimes, API source operations, dependencies, request
  inputs, bindings, credentials, structural results, side effects, and action policies.
- **API security alternative** is one operation-level authentication choice
  preserved from reviewed source metadata. Alternatives are OR, bindings
  inside one alternative are AND, and an empty alternative explicitly permits
  anonymous access. iCoT selects one alternative before request mapping and
  stores only symbolic binding names, never credential values.
- **Lifecycle operation ranking** is prompt-safe sibling-role inference over
  Apitools operation summaries. Apitools owns the generic ranking algorithm;
  OpenUdon owns which workflow operations seed it and how ranked hints affect
  questions, prompt detail, and reviewed workflow intent.
- **Browser profile** is a reviewed `uws.browser.1.5` through `uws.browser.1.9`
  contract produced by Browsertools. Versions 1.8/1.9 opt into component-safe
  parameter templates; 1.9 also supports literal-brace escapes.
  OpenUdon may author against its declared actions only when no adequate API operation is available;
  it packages the profile and safe digest/lifecycle evidence without browser sessions or raw captures.
- **Browser authentication profile** is a reviewed, secret-free
  `uws.browser-authentication.1.0` or additive `uws.browser-authentication.1.1` sign-in recipe. OpenUdon selects an explicit
  flow, named execution-local session, symbolic credential bindings, bounded
  timeout, and authoring approval; Udon resolves values, brokers MFA, and owns
  live session state behind a separate runtime approval.
- **Browser registration profile** is a reviewed, secret- and account-free
  `uws.browser-registration.1.0` account-creation recipe. OpenUdon may package
  a Browsertools digest-bound review, select one explicit flow with complete
  symbolic credential bindings, and lower fixed duplicate, ambiguity, cleanup,
  and exact submit-approval policy to
  `uws.browser-registration-call.1.0`. Registration creates no browser session.
  Guided no-submit authoring binds macro controls only to current reduced
  observations, permits one symbolic password slot to be reused for
  confirmation, and treats contact names as identifier-class symbolic input.
  Because authoring cannot observe a post-submit page, its success locator is
  separately operator-reviewed, explicitly unobserved, and deferred until a
  trusted runtime proves it. One process-local Launch attempt is consumed
  immediately before worker construction; later starts remain locked even
  after failure or cancellation, and final state exposes only a closed,
  value-free failure class rather than worker prose.
  OpenUdon permits only inert authoring, build, assessment, approval-template,
  and dry-run qualification until an independently compatible Udon and
  Browserdriver runtime contract is published and pinned.
- **Browser-profile transaction** is OpenUdon's public, value-free
  `openudon.browser-profile-transaction.v1` coordination record over those
  existing profile families. It binds exact candidate/review/provenance
  digests, a BAP+BCP symbolic session or session-free BRP, prepare-only package
  and qualification digests, atomic promotion identity, and closed failure or
  recovery states. It is not a UWS document, browser session, approval,
  execution record, private-result locator, or new UWS semantic contract. One
  internal driver-free engine supplies matching value-free API v4, accessible
  local UI, and exact stdin/NDJSON terminal views; every review, preparation,
  promotion, and recovery decision remains separate and none grants runtime
  authority.
- **Browser authoring handoff** is an inert
  `openudon.browser-authoring-handoff.v1` plan that tells an operator how to
  run bounded Browsertools authoring outside iCoT and how to return a reviewed
  result. It carries typed argv templates and review gates, not browser
  authority, credentials, session state, or captured content. Its existing,
  restrictive private root contains every persisted handoff artifact.
- **Authenticated browser authoring** is the primary iCoT UI capture flow with
  `icot browser-author live` as an expert fallback. Browsertools owns one headed,
  non-persistent Playwright-Go context while the human enters credentials/MFA;
  iCoT owns typed-goal review, reduced-observation disclosure, API/origin/action/
  completion/staging gates, strict local protocol consumption, and independent
  validation of the private digest-bound result. No live context or secret is
  transferred or packaged. Reduced accessibility labels are useful heuristics,
  not DLP: ordinary names, identifiers, and order numbers may remain, so the
  operator reviews them before planner disclosure or retained trace use.
- **Reviewed live MFA and outputs** are author-session v2 human-only choices.
  Browsertools advertises compatible MFA kinds; the human chooses the exact
  exercised kind and may declare at most 16 final-observation scalar/presence
  outputs. The planner cannot choose either, and runtime values never enter the
  result or package review.
- **Browser verification summary** is optional, value-free review evidence
  derived from an explicit `browsertools.live-check.v1` or
  `browsertools.portability-check.v1` report. OpenUdon independently binds its
  declared paths/counts/types/fixed diagnostics to one exact profile/action set
  and retains only normalized facts plus the report digest. It is neither raw
  browser evidence nor runtime authority; portability remains optional.
- **Browser integration evaluation** is a provider-free release-evidence
  matrix across OpenUdon, Browsertools, UWS, Udon, and Browserdriver. Its
  digest-bound report records fixed gate outcomes and repository revisions for
  anonymous handoff, explicit authenticated authoring, UWS 1.8 contexts, UWS
  1.9/browser 1.7 scalar conversion, and
  trusted v2/v3 replay,
  never child-process output or browser evidence. Installed-engine and headed
  authentication checks remain separate loopback-only opt-ins.
- **Exact authenticated-authoring qualification** requires a real Browsertools
  envelope to cross OpenUdon's strict import boundary and its exact authored
  profile pair to cross Udon/Browserdriver replay. Exact bounds, disclosed
  context topology, and page-derived labels are checked before model access;
  compatible-looking hand-built fixtures are insufficient.
- **Browser scenario evaluation** is the real-browser complement to the fast
  browser-free integration matrix. A required 23-case loopback suite exercises
  Browsertools author-session v2, OpenUdon staging, UWS synthesis, Udon v3, and
  Browserdriver v3 without external network access. A required eight-case
  headless journey suite adds realistic reviewed forms, multi-action reads,
  structured extraction, approved and rejected writes, parameter failures,
  and cross-execution isolation from Browsertools guided bundles through UWS
  1.8 replay. A separate opt-in four-site public suite performs anonymous
  read-only presence canaries with explicit network authority. All retain only
  closed value-free reports.
- **Quality report** is the deterministic release gate for current generated artifacts.
- **Review evidence** is the human-readable side-effect, risk, credential, and trusted-runner
  package summary.
- **Review handoff** is an `apitools.review-handoff.v2` manifest with digest-bound OpenUdon package inputs,
  approval states, owner split, execution policy, credential binding names, and trusted runner
  metadata. The wire version remains stable during the migration.
- **Trusted runner** is the local `openudon run` gate that invokes a private executor path only after
  approval and package validation pass. Every run has a unique ID and directory,
  and v2 evidence binds approval, config, handoff, package, and executor-report
  digests. Validation and runtime derivation use one immutable manifest-bound
  byte snapshot, while staging rehashes current files and rejects later drift.
  Strict Udon execution-report v2 remains the legacy/BAP contract, while the
  exact registration-only Browserdriver-v4 handoff requires execution-report
  v3 and its closed registration failures. Unrelated crashes cannot satisfy
  expected-failure evidence. Optional Ed25519 signatures can pin operator
  identity.
- **Fnct helper selector** is a public helper function name, such as `gmail.render_raw`, that
  OpenUdon may author and review in `x-uws-runtime`. OpenUdon does not execute the helper; trusted
  runtimes import and register the implementation.
- **Local iCoT UI server** is the primary interactive authoring surface and an
  experimental `icot ui` transport over one engine. It binds only `127.0.0.1`, opens a tokenless page, and uses a
  terminal-only five-minute single-use access code to install the scoped
  cookie beneath a separate unguessable path; a used or expired code can be
  rotated through a throttled root-page action that prints the replacement
  only in the terminal. It retains separate exact authoring/capture revisions,
  a complete-state ETag, and optimistic fingerprints over engine-owned files.
  Journey selection, bounded API upload/staging, isolated existing-account
  Chromium capture, frontier rounds, approval, package build/assessment,
  failure resume, and closed-allowlist handoff inspection form one lifecycle.
  It freezes only after a passing package build. External changes retain cached
  inspection but require restart before mutation. Its JSON v4 wire remains an
  experimental local coordination contract; v1/v2 are not served. The UI does
  not invoke an LLM extractor, create runtime approvals, accept credentials, or
  execute workflows.

## Scope

- OpenUdon-owned project templates, prompts, guided iCoT authoring, examples, eval fixtures, and review
  policy.
- UWS-facing artifacts authored or generated outside OpenUdon as review inputs
  when users bring them to OpenUdon. Desired-state conversion and
  provider/resource operation mapping are owned by `../ramen`, not OpenUdon.
- Deterministic artifact generation from project briefs and intent into workflow HCL, UWS, plan,
  quality, refinement, review, and handoff artifacts.
- Quality gates and local validation wrappers for OpenUdon package correctness.
- Optional workflow orchestration prompt/config policy for OpenUdon-managed work items.
- Local trusted execution wrapper, approval template generation, package digest checks, and tier
  enforcement.
- Cross-repo compatibility evidence for UWS semantics, udon lowering/runtime behavior, provider
  drift, release gates, and optional sibling checkout readiness.
- OpenUdon-owned iCoT workflow graph, v2 session/transcript/report adapters, reviewed source staging,
  proposal/draft approval lifecycle, review evidence, package digest, credential policy, and trusted
  executor handoff helpers. Generic graph/frontier mechanics remain in `../authoring` and API-source
  discovery remains in `../apitools`.
- Structured API security-alternative authoring: OpenUdon retains Apitools'
  OR-of-AND sets through `authoring.prompt-context.v2`, forces one stable
  SHA-256-fingerprinted resume-safe selection before mappings, and refuses to union credentials or
  promote unresolved alternatives into runnable intent.
- Shared fail-closed credential detection across artifacts and LLM request
  mappings, bounded provider responses, header-only Gemini authentication, and
  DNS-pinned remote-source acquisition.
- A single-workspace loopback-only iCoT UI process with an experimental
  authenticated JSON v3 transport, separate authoring/capture revisions,
  optimistic workspace-drift enforcement, bounded API and existing-account
  browser acquisition, explicit authoring/package phases, and polling embedded
  accessible authoring/review/handoff assets.
- API/event source metadata discovery/search/import/indexing reuse plus optional provider/spec/security
  catalog advice through `github.com/OpenUdon/apitools`. OpenAPI/Swagger sources remain the default,
  while Google Discovery, AWS Smithy JSON, AsyncAPI, GraphQL, OpenRPC, gRPC/protobuf, and OData can
  be staged as first-class UWS source descriptions when the trusted executor supports them. GraphQL,
  OpenRPC, gRPC/protobuf, and OData use the UWS 1.4 source-description contract and the reviewed
  local source metadata exposed by `apitools`. AsyncAPI is treated as a
  source document family for event/message operations, not as an OpenUdon-owned execution runtime.
- API-first browser fallback through verified local Browsertools profiles or its static read-only
  registry catalog. OpenUdon owns selection, interview evidence, materialization, review, and handoff;
  Browsertools owns capture/profile/catalog mechanics and Udon owns runtime execution.
- Package-local browser sign-in authoring from verified Browsertools
  authentication profiles. OpenUdon owns the intent fields, profile/flow
  selection, safe review metadata, UWS supplement lowering, quality gates, and
  handoff inventory; it owns no credential value, challenge response, or live
  browser state.
- Digest-bound trusted browser replay through `openudon run`. OpenUdon derives
  a value-free Browserdriver protocol, symbolic credential/session environment
  mapping, and exact operation/authentication approval contract from the
  reviewed package; Udon and Browserdriver retain runtime implementation,
  credential resolution, MFA, and browser-session ownership.
- Explicit Browsertools authoring handoff planning for UI-only capability gaps.
  OpenUdon can replay and reduce an explicitly supplied
  `browsertools.guided-authoring.v1` result to its verified embedded profile,
  but it never stages the result's evidence envelope or launches the external
  authoring commands.
- Explicit authenticated goal-directed authoring orchestration through a
  separately re-executed Browsertools worker, bundled in `icot` by default
  with an absolute external CLI override only for experts. OpenUdon may guide only Browsertools-issued
  candidate actions, retain no protocol transcript, and atomically stage only
  canonical profiles plus safe review metadata after final human approval.
- Strict author-session/result v2 MFA and output review. OpenUdon negotiates a
  16-output bound, keeps exact challenge/output choices human-only, validates
  value-free proofs, and reconstructs both profiles before staging.
- Optional value-free current-page and cross-engine Browsertools verification
  attachment. OpenUdon owns strict downstream validation, package summaries,
  quality/review evidence, and digest binding; Browsertools owns live
  acquisition. Raw/rich/private/authentication evidence and local report paths
  do not enter the final package.
- Provider-free integrated qualification across the exact OpenUdon,
  Browsertools, UWS, Udon, and Browserdriver revisions. Default evidence uses
  fake/synthetic sessions and launches no browser; installed engines and
  headed authentication/authoring remain explicit loopback-only opt-ins.
- Complementary real-browser scenario qualification with an embedded,
  deterministic loopback corpus and a realistic local read/write journey
  corpus as release gates, plus a fixed anonymous public inventory as
  informational drift evidence. Public network authority is never implicit,
  and no suite archives page or subprocess content.

## Non-Goals

- Public UWS schema or semantics; those belong in `../uws`.
- Generic OpenAPI/UWS compilation, lowering, or runtime execution behavior; those belong in
  executor implementations such as `../udon`.
- Reviewer identity storage, managed state transitions, or audit persistence; those belong outside
  OpenUdon.
- Concrete IaC intent models, `.tf` generation, graph/profile/planning/state/drift behavior, or
  `w8m` executor contracts; those belong outside OpenUdon.
- Desired-state parsing, conversion, provider/resource operation mapping,
  state, drift, plan/apply, import, refresh, or infrastructure audit behavior;
  those belong in `../ramen`.
- Secret storage, credential resolution, account selection, endpoint selection, production
  execution policy, or provider SDK ownership.
- Treating catalog provider/spec/security metadata as mandatory workflow behavior or a release gate.
- Direct production side effects from synthesis, build, promote, assess, iCoT, or eval commands.
- Browser capture implementation, drivers, credential/session storage,
  authentication or registration execution, profile execution, authenticated
  authoring-context transfer, membership accounts, or a
  mutable registry service. A Browsertools catalog is static HTTPS/object storage with repository
  review as its contribution path.
- Remote/LAN iCoT serving, TLS or account management, live registration
  discovery/capture, consent/enrollment/CAPTCHA/billing discovery, persistent UI tokens,
  multi-workspace or multi-user hosting, folder browsing, React/Node frontend
  dependencies, UI-owned LLM drafting, remote browser agents, trusted
  execution, profile replacement/relearning, or browser-session transfer.
- Accounts, collaboration, a history service, push transport, or any widening
  beyond the deliberate single-workspace, single-operator process.

## Consolidated browser system engineering

M79 adds one application-owned registration lifecycle shared by the UI and a
supervised command interface, plus an unattended local qualification entry
point. Browsertools remains an observer; approved actions and registration
remain with Udon/Browserdriver. Synthetic qualification grants no real-target
authority. Local qualification and review are complete; publication and
operational adoption retain their separate boundaries.

The opt-in application control flow lets operators review and promote BRP
and BAP/BCP authoring in one application while trusted execution remains
separate. See [architecture.md](architecture.md#icot-architecture) and
[M80](../docs/history/status-M80.md) for its boundary and evidence.

M81/W09 completes the local integration through the W8M-owned operating adapter
and concrete synthetic target. OpenUdon supplies opt-in private human response
input to the trusted runtime and native qualification of supervised BRP and
BAP/BCP packages. Three complete W8M units pass independent source, runtime and
receipt verification. W8M owns its target tests and separately authorized real
registration/login/campaign work; generic authoring and qualification stay here.

A private v2 registration attestation can represent one independently verified pre-submission failed attempt. Its owning operating application must separately approve and consume a persistent one-use recovery claim; OpenUdon neither infers no submission from failure nor issues recovery authority.

## Development feedback

E13 separates `fast`, one affected synthetic `smoke`, and full `qualify` checks.
Feature iteration can reuse unchanged compiled outputs and explicitly selected
successful development results. Reuse is visible and retains the original run;
it never represents independent browser execution or operational qualification.
