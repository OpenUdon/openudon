# Tech Stack

## Current Browser 1.10 qualification stack

OpenUdon's Go module pins published UWS M05
`80ee9bfb24a688b5e875dadf9ecacdc65398f1ff` and Browsertools M32
`3abe70efc03d9ccb97b8b30e5e86328f60a70c64`. The exact Browserdriver M15 and
Udon M43 source pins, module versions, and separate 14-repository Udon build
closure are recorded in
`internal/browserscenario/current-compatibility-lock-v4.json` (SHA-256
`58363021e44961527468bc686df114ce69770709345eb39702fbf38e84da6d2a`) and
`current-qualification-build-inputs-v4.json` (SHA-256
`10fa8b2570f0a72688a1c8d282fe84cf7ea4af6e82ad5ffa85aa6fc994ff371a`). The
scenario, integration and native current selectors now emit v4 reports and
include the three Browser 1.10 count journeys. Full E22 qualification and
bounded review are still pending.
UWS maintains the profile-version checklist at
[`future-source-profiles.md`](../../../uws/docs/future-source-profiles.md#adding-a-browser-profile-version).

## M86 v2 lock snapshots

The M86 current scenario v2 verifier reads
`internal/browserscenario/current-compatibility-lock-v2.json` (SHA-256
`57ebe6c70bc0b1e810ed4fb490f36ecb227c47f362bb738b56680ce7abcd6a77`). The
integration v2 verifier reads its own byte-preserved snapshot at
`internal/browserintegrationeval/current-compatibility-lock-v2.json` (SHA-256
`9eec17f1489e1c805e2d2bfb8b89a439ee7153d5c49ec09a3ee903ce6761d393`). The
The frozen E21 current v3 scenario and integration locks are retained at
`internal/browserscenario/current-compatibility-lock-v3.json` (SHA-256
`90f96fa2d02809f641c7391f1245487926a48c64cef9a4ace609ebad99667246`) and
`internal/browserintegrationeval/current-compatibility-lock-v3.json` (SHA-256
`4958e20014cb008a3d8128f0328c7f86bd07cd674a463aa7578ae4348a2e9a9f`). The
v3 14-source local replacement closure is frozen at
`internal/browserscenario/current-qualification-build-inputs-v3.json` (SHA-256
`4993304edf46953c33b6112c4f00e3fcf526811ac91400989b775a19066977b9`).
It fixes Browsertools `9333a9f25dbb17551998a429e123e7a9ba976648` and UWS
`e9b6181be0abb7f683fdb624d4dba282a59991d1` among the exact clean source inputs.
Integration runs verify the same Udon closure before executing the fixed
matrix. Historical v1 and frozen M86 v2 verification retain their original
locks and meaning. All three retained M86 reports pass with their original
digests; E21 evidence is recorded in [status-E21.md](status-E21.md).

Native `openudon browser-system-eval --stack current --suite loopback` selects
the v4 compatibility and 14-source build-input locks, requires all primary
and auxiliary worktrees to be clean, and emits
`openudon.browser-system-qualification.v4`. Its v3 reader retains the E21
snapshots. The default remains historical native v2.
`make browser-system-current-check` runs the explicit current path
and verifies its report; the verifier dispatches from the saved report version.
Current evaluation can take `--browserdriver-node-modules` for a separate
read-only module tree outside the clean Browserdriver checkout. The evaluator
checks `@types/node`, `playwright`, `playwright-core`, and `typescript` against
that checkout's `package-lock.json`. For the current native Make target, set
`OPENUDON_BROWSER_SYSTEM_BROWSERDRIVER_NODE_MODULES`. Browserdriver source
builds and npm tests use disposable output/checkouts and never install or change
the supplied source or modules. Scenario builds link the separate module tree
inside an exact disposable Browserdriver checkout so TypeScript resolves its
imports; runtime readiness uses that staged package and the same linked modules.
The integration npm-test gate copies the already-validated module tree into the
disposable Browserdriver clone as a read-only `node_modules` directory. This
keeps Node and TypeScript dependency lookup beside the package when tests launch
child processes with a minimal environment; no package installation runs.
Udon Go test gates clone its exact current source and fourteen locked sibling
inputs to a temporary workspace; the native runner removes it before passing
the stage and rechecks the original closure.
Native registration UI, supervised control, and authenticated package fixtures
and the BRP transaction fixture remove any empty `eval/runs` ancestry they
create, preserving pre-existing paths and rejecting symlink parents so source
rechecks stay clean.

## Current Browser Authoring Tooling

Reviewed query admission, generated-profile, parent-attestation, and synthetic
application/package checks use the existing dependencies. Real Chromium gates
retain the required sandbox; an administrator-owned `CHROME_DEVEL_SANDBOX` helper
may be forwarded through reviewed local runner configuration when the host
requires one. `openudon browser-system-input --repo-root ABS --udon-repo ABS`
is a browser-free input inventory, not execution evidence. Current browser
qualification commands and pinned toolchain assumptions are maintained below;
exact authenticated-authoring evidence is in [E20](../docs/history/status-E20.md), with
superseded wording in the [history index](../docs/history/index.md).

## Memory Bank Index

- This file owns implementation technologies, dependency defaults, and tooling constraints.
- Use [product.md](product.md) for product scope and non-goals.
- Use [architecture.md](architecture.md) for system boundaries and planned structure.
- Use [milestone.md](milestone.md) for milestones, current completion state, and the status-file
  index.

OpenUdon is a Go package and CLI that composes sibling modules for public UWS modeling,
API source metadata discovery/indexing, and portable trusted executor handoff.

## Language And Runtime

- Registration 1.1 pins published UWS `9ff877ebce55`, Browsertools
  `ec0b9e9d6ca1`, Udon `5ef6af99430c` and Browserdriver `9d13e8b35394`.
  Actual browser qualification uses Node 24 and the existing Playwright runtime.
  Run `go test -tags browser_system_qualification ./internal/browserscenario
  -run TestTypedRegistrationUIToTrustedRuntime` for the complete synthetic typed
  journey. Values and private snapshot digests are scanned out of retained
  package/run artifacts. Native spinbutton locators remain outside immutable
  BRP 1.1; numeric scalar fixtures use supported textboxes.

- Private registration discovery uses Go `os.Root`, strict JSON, SHA-256
  revisions, cross-process exclusive write locks and synced atomic revision
  files. Storage is outside Git/examples: 0700 directories, 0600 files, 32
  candidates and 512 revisions maximum. The separate authenticated HTTP
  resource and explicit application-control discovery replies share one service;
  ordinary API snapshots and package exports exclude inventory. Embedded plain
  JavaScript prepares existing wizard fields without launching a browser. No
  UWS, Browsertools or runtime dependency changes are required by A28.
- Primary language: Go, with module directive `go 1.26.6`.
- Module path: `github.com/OpenUdon/openudon`.
- CLI entrypoints: `cmd/openudon`, `cmd/icot`, and `cmd/udon-runner`.
- Headless authoring lives under `internal/icot/engine`. Its internal methods
  use `context.Context` plus typed configuration, snapshots, acquisition,
  round, preview, approval, resume, and capture-stage records. The bounded API
  upload mutation accepts an `io.Reader`; no terminal reader/writer or browser
  object enters the engine. Snapshots are JSON-marshalable but are not a
  published schema or supported Go API.
- Phase B local UI transport lives under `internal/icot/ui` and is standard
  library only: `net/http`, IPv4 loopback listeners, `crypto/rand` capability
  tokens, terminal-only 12-character Crockford access codes with throttled
  post-use/expiry terminal-only rotation, scoped
  per-process cookie paths, SHA-256 revisions, optimistic
  workspace inspection, strict bounded duplicate-safe UTF-8 JSON with a
  maximum nesting depth of 32, conditional
  snapshots, `embed` assets, platform opener commands, and bounded
  signal-driven shutdown. The `openudon.icot-ui-api.v4` response wire is
  experimental and has no published schema or compatibility guarantee; v1-v3
  routes are not served. Authoring and capture revisions are separate, browser
  transactions retain their own engine revision, and the response ETag binds
  the complete displayed state.
- The production shell remains separately embedded plain HTML, CSS, and
  JavaScript with no React, Node, client build step, workflow execution, or
  folder browser. It renders journey/source acquisition, isolated Browsertools
  capture, and the full
  frontier as engine-described accessible controls, posts complete
  revision-bound rounds, reopens eligible settled human answers, exposes exact
  previews/actions/read-only write conflicts plus prompt-safe review evidence, and requires
  explicit reviewed final or incomplete approval. A polite mutation live
  status and deterministic post-success focus cover the next frontier,
  proposal review, authored/package-failure recovery, and handoff-ready completion.
- `internal/icot/artifactwriter` is the common terminal/engine transaction for
  source revalidation/materialization, safe browser capability/authentication
  metadata, incomplete-draft promotion/cleanup, and rollback-capable atomic
  file writes. Outputs stay beneath the canonical example root; descendant
  symlinks and pre-commit swaps are rejected; successful and successfully
  rolled-back transactions clean temporary backups; rollback failure is
  indeterminate. A cleanup failure after every replacement succeeds is a
  non-fatal write-result warning rather than a failed transaction. One shared
  read-only plan validator runs from prepare, conflict inspection, and commit
  before directory/temp/backup creation. It rejects duplicate,
  case-insensitive-equivalent, ancestor/descendant, and remove/write output
  collisions, while source staging reserves `.icot/**`, `project.md`, and both
  intent paths.
- `openudon catalog ...` exposes thin authoring-time wrappers over
  `github.com/OpenUdon/apitools/catalog` for first-class provider inspection,
  advisory output, security reports, and package-local OpenAPI import.
- Intent may use `source = "..."` as the preferred API source document reference. `openapi = "..."`
  remains a backward-compatible alias. OpenUdon infers source type from catalog metadata,
  directory convention, or parser behavior rather than adding a separate intent `source_type`.
- The optional operator-authored `content_trust` intent block lowers to UWS
  `uws1.ContentTrust` (supported since 1.9.1). New workflows declare UWS 1.11.0;
  existing packages retain their declared versions. Source labels remain package-relative paths until
  synthesis resolves generated source-description IDs; operation labels use
  the same stable lowering as leaf steps. Empty/no-op or unresolved objects
  fail closed. The structured LLM intent schema deliberately omits this field.
- Package assessment invokes `contenttrust.Analyze` only when the generated
  document declares content trust. A package-local Browsertools M28 resolver
  stable-reads contained profiles; invalid or unavailable contracts become the
  core fixed-message resolver-failure finding. Analyzer findings map to
  quality status `warn` with code, analyzer severity, document path, and fixed
  message, and the same value-free fields enter `expected/review.md`. This pass
  is not called from execution validation, plan construction, or trusted-runner
  authorization.
- `make content-trust-qualification` runs the deterministic OpenUdon E12 matrix
  and an opt-in `udon_contenttrust_qualification` test. The tagged test requires
  clean exact UWS `9e676eaa469e`, Browsertools `75fd5c3ab81f`, and Udon
  `207e7f1` sibling checkouts before invoking M37's public analyzer tests with
  `GOWORK=off`. Udon is deliberately absent from OpenUdon's `go.mod`.
- OpenUdon A23 updates the current production Browsertools dependency to
  published A10 `v0.0.0-20260829181035-3107470313d4`; the historical E12
  tagged compatibility fixture above remains locked to its qualified M28
  checkout. The A23 module pin and exact dependency expectation pass full
  offline consumer verification.
- OpenUdon scans and stages API/event source files under `openapi/`, `google-discovery/`,
  `aws-smithy/`, `asyncapi/`, `graphql/`, `openrpc/`, `grpc-protobuf/`, `odata/`, and
  legacy-readable `discovery/`. It emits typed source descriptions for OpenAPI, Google Discovery,
  AWS Smithy, UWS 1.3 AsyncAPI, and UWS 1.4 GraphQL/OpenRPC/gRPC-protobuf/OData sources.
- OpenUdon also scans and stages verified `uws.browser.1.5` through `1.9` profiles under
  `browser-profiles/`, emits UWS `browser-profile` source descriptions, and
  records prompt-safe source review metadata in `.icot/browser-sources.json`.
  Browsertools owns validation, private cache, bundles, discovery, and the
  service-free static registry; Udon owns drivers, sessions, and execution.
- Repeatable `icot --browser-verification PATH` accepts only strict, bounded
  JSON `browsertools.live-check.v1` and
  `browsertools.portability-check.v1` files. `internal/browserverify` mirrors
  only those value-free wires and Browsertools' fixed probe/diagnostic
  derivation over the already-pinned public `profile` package; it does not
  import the unpublished live-capture implementation or Playwright. Final
  packages retain normalized summaries in `.icot/browser-sources.json`, not
  the input report or local path.
- OpenUdon scans reviewed `uws.browser-authentication.1.0`/`1.1` profiles under
  `browser-authentication/`, records safe review metadata in
  `.icot/browser-authentication.json`, and lowers explicit authentication and
  named-session intent fields to the matching public supplements. New workflows
  declare UWS 1.11.0. Browsertools owns local validation; Udon and its persistent
  Browserdriver own credential resolution, MFA challenge interaction, session
  state, and execution. Active Browser 1.8/1.9 actions select private
  browser-driver v10, which carries older actions as inner v2; older-only
  workflows keep their prior protocol selection.
- OpenUdon scans reviewed `uws.browser-registration.1.0` profiles and their
  digest-bound `browsertools.registration-review.v1` bundles under
  `browser-registration/`. Explicit `browser_registration` intent lowers to
  `uws.browser-registration-call.1.0` with exact symbolic bindings and fixed
  duplicate/ambiguity/cleanup policy. Package, quality, approval-template, and
  trusted-runner dry-run are supported; executor argv construction rejects the
  workflow until compatible Udon and Browserdriver contracts are pinned.
- The API-v4 guided registration form defaults symbolic `identifier`,
  `password`, and identifier-class `contact_name` slots. Credential macro
  controls select only current declared symbols and may reuse a symbol across
  multiple steps. Success uses an operator-reviewed role/name rather than a
  current observation candidate; the strict request fixes proof kind to
  `operator_reviewed_deferred`, requires a declared origin and canonical
  disclosure-safe accessibility name, and discloses that runtime proof remains
  required. No public UWS or Browsertools wire changes.
- Guided registration binding validation uses the portable lowercase symbol
  grammar as name authority. An entropy-like name must additionally have
  descriptive snake-case structure made from at least two closed-vocabulary
  purpose words and at most one alphanumeric namespace component of up to 12
  characters with one digit run; it need not repeat the declared slot. Known
  credential formats, short opaque prefixes disguised with a slot suffix, and
  digit-bearing or letters-only opaque multi-token names remain rejected with a
  binding-specific value-free API diagnostic. This rule is local to draft
  construction and does not weaken artifact credential scanning.
- `internal/browsertransaction` implements the unsupported-as-a-library but
  public-as-JSON `openudon.browser-profile-transaction.v1` artifact contract
  for unchanged BAP and legacy BRP results plus the registration-v2-only
  `openudon.browser-profile-transaction.v2` contract. The matching Draft
  2020-12 schemas live under `docs/schemas/`; strict Go
  decoding with a 256 KiB/32-level bound, duplicate-name and invalid-UTF-8
  rejection, semantic validation, canonical SHA-256 encoding, and immutable
  transition checks remain internal. The wire admits only the already
  published BAP 1.0/1.1, BCP 1.5/1.6/1.7, and BRP 1.0 discriminators, plus the
  exact authenticated-authoring v2 or registration-authoring v1/v2
  Browsertools provenance versions, and does not change UWS.
- `internal/browsertransaction/engine` owns the driver-free optimistic
  lifecycle and invokes `internal/packagepipeline` for preparation,
  qualification, promotion, selected inspection, and recovery. Its defensive
  snapshots expose only the public transaction, stable failure codes, allowed
  operations, and package/recovery digests. The sibling `presentation` package
  derives one Browsertools-backed BAP+BCP/BRP disclosure for API v4 and
  terminal NDJSON. `icot ui` starts the engine only from an all-or-none public
  transaction/package flag group; `icot browser-transaction` uses bounded
  exact stdin phrases and has no implicit or flag-only approval, browser,
  submit, or runtime path.
- `internal/browsertransactioneval` defines the canonical
  `openudon.browser-transaction-qualification.v2` evidence wire. It accepts
  only the unchanged published UWS lock and exact clean locked OpenUdon,
  Browsertools, Browserdriver, and Udon `main` revisions whose
  local-unpublished or published classifications are independently checked
  against their origins. Its closed gate/failure vocabularies, nine BAP+BCP
  digests, eleven BRP/runtime digests, and fixed sandbox/network posture prove
  GET/HEAD-only authoring followed by one separately approved loopback POST,
  executor invocation, a fixed result, and no registration session. Reports use
  compact canonical JSON plus a `.sha256` sidecar; `openudon
  browser-transaction-eval --out REPORT` runs the adversarial and real
  sandboxed loopback qualification only from clean exact sibling revisions,
  with read-only remote-head resolution for all five repositories plus
  post-run revision revalidation. `openudon
  browser-transaction-eval --verify REPORT` rejects noncanonical, missing,
  duplicate, unknown, oversized, symlinked, tampered, or dependency-drifted
  evidence without launching a browser or contacting a target.
- `internal/browsercandidate` consumes Browsertools registration-authoring
  v1/v2 results through an anchored mode-`0700` private inbox and a 256 KiB stable
  read. It independently reconstructs canonical profile/review bytes, checks
  exact result/source/review digests and freshness, and produces only a
  candidate-state M77 transaction plus defensive canonical source/review
  copies. Private result names, paths, and envelopes are not returned.
- `internal/browsercandidate` also composes authenticated-authoring v2 BAP+BCP
  pairs. It rechecks canonical producer encoding, both independent reviews,
  exact flow/slot/origin/context compatibility, login-state requirement, and
  earliest expiry, then owns the defensive candidate and its validated
  `candidate -> reviewed` transition. The transaction carries only a symbolic
  execution-local session; no browser handle, cookie, storage, or runtime
  session state is retained.
- `internal/icot/elicitor` validates those path-free transaction candidates as
  deterministic `virtual-browser://` sources. It binds candidate and review
  digests, schema, freshness, origins, exact reviewed-flow identity and
  symbolic slot coverage, canonical target, and BAP-provides/BCP-requires
  session dependencies; BRP remains
  dependency- and session-free. `internal/icot/engine` exposes only a
  value-free catalog summary, requires an exact optimistic catalog generation
  for dependency-closed selection/replacement, and retains canonical source
  bytes only in memory. Physical target collisions and stale selected
  identities fail closed, while existing API-first source ordering is
  preserved. Workspace materialization still occurs only through explicit
  authoring approval.
- Reviewed BRP virtual operations carry one exact flow, transaction-derived
  symbolic bindings, the producer-reviewed cleanup disposition, fixed
  duplicate/ambiguity policy, a bounded timeout, and an explicit
  `runtime_supported=false` marker. The authoring step has no browser session
  or generic operation field and requires a fresh step-scoped confirmation.
  `internal/icot/artifactwriter` revalidates the canonical BRP and independent
  Browsertools review from in-memory bytes, stages the conventional adjacent
  review, and derives `.icot/browser-registration.json`; synthesis then lowers
  the package to `uws.browser-registration-call.1.0`, while trusted non-dry
  execution continues to reject before executor invocation.
- `internal/packagepipeline.PrepareCurrent` creates the P05 value-free
  preparation manifest and immutable-by-copy byte set from one strict
  handoff-complete generation. It uses package-artifact path validation,
  bounded stable reads, existing handoff self-digest semantics, and the same
  package digest version as trusted-run inspection; it requires a caller-owned
  portable scope and never writes or exposes the canonical local root.
- `internal/packagepipeline.Qualify` is the P05 pre-promotion gate. It uses a
  fresh same-filesystem mode-0700 scratch root, anchored `os.Root` materialize
  and cleanup operations, mode/link/inventory validation, current quality and
  secret assessment, package/handoff inspection, and trusted dry-run with an
  empty environment. Its deterministic evidence contains digests and fixed
  posture/gate names only; scratch and source paths remain private.
- P05 promotion uses only Go filesystem primitives: a same-store restrictive
  staging directory, file and supported-directory `Sync`, immutable
  record-derived generation names, `Rename` publication, and one sibling-temp
  `Rename` for `current.json`. `os.OpenRoot` anchors transient cleanup, a
  create-only mode-0600 lock serializes cooperative builders, and strict
  readback re-prepares every selected generation before exposing defensive
  bytes. No generation-retention operation exists in this boundary.
- Promotion recovery uses strict JSON lock/intent/report wires with self
  digests, atomic create-only hard-link publication, typed failure state plus
  target-generation identity, and `InspectRecovery`/`Reconcile` read-check-
  confirm semantics. Reconciliation is capped at 128 recognized transients
  and uses anchored removal only for staging, temporary selector, intent, and
  lock names; it cannot mutate `current.json` or any `generations/` member.
- `packagepipeline` compatibility entry points bind every selected package
  review, approval template, and trusted run to an exact selector digest. They
  canonicalize prospective approval/work paths and reject containment in the
  immutable store, force real current assessment, and preserve trustedrunner's
  approval, run-evidence, dry-run, and executor contracts. The CLI emits only
  versioned value-free preparation/qualification/selection/recovery JSON.
- `icot browser-authoring plan` emits exact JSON argv templates, typed dynamic
  argument declarations, and review stages for an external Browsertools
  authoring run. Its private root must already be a restrictive non-symlink
  directory disjoint from the example, and any file output remains inside that
  root. The plan is non-executing and accepts no credential or session input.
  Explicit `browsertools.guided-authoring.v1`
  files supplied through `--browser-profile` are strict-decoded and replayed
  with Browsertools' existing draft/profile/review APIs; only the canonical
  embedded profile can enter an approved package.
- UI capture and `icot browser-author live` start a separate worker process.
  `internal/icot/browserauthor` owns the frontend-neutral asynchronous client;
  the hidden `__browsertools-worker` uses Browsertools' importable runner. The
  parent publishes a fully copied and hashed executable into a private
  content-addressed mode-`0500` cache, applies a minimal environment and
  process-group cancellation, re-verifies cached bytes before reuse, drains stdout, and speaks the closed
  `browsertools.author-session.v2`,
  `browsertools.authenticated-authoring.v2`, and separate no-submit
  `browsertools.registration-author-session.v1` /
  `browsertools.registration-authoring.v1` wires. A process-private parent
  attestation binds the ordered actions, checkpoints, approvals, observations,
  outputs, contexts, authentication proof, and exact-origin ledger before
  staging and is never serialized or exposed over HTTP. The UI does not accept an
  arbitrary executable; terminal `--browsertools` remains an expert
  compatibility override. The release binary links Browsertools' Playwright
  implementation for the worker, but the engine and HTTP server never run it
  in-process. Chromium remains a separately installed prerequisite; module
  maintenance does not install a driver or browser. Experimental UI API v4
  registration state includes the additive, value-free `attempt_consumed` and
  `failure_code` fields. The Go server, not JavaScript, owns the one-attempt
  latch and closed failure-code allowlist; the client only renders and locks
  from that authoritative state.
- `openudon browser-integration-eval` and `make browser-integration-check`
  run the provider-free A03/P01/A04/A06/E02/E03 release matrix across sibling OpenUdon,
  Browsertools, UWS, Udon, and Browserdriver checkouts. The strict current
  `openudon.browser-integration-eval.v2` JSON report and `.sha256` sidecar live
  under ignored `eval/runs/`, record all five commit/dirty states, fixed named
  evidence and closed diagnostics, and retain no subprocess output. Browsertools
  doctor checks Chromium, Firefox, and WebKit without installation or browser
  launch. Required named gates cover a real Browsertools envelope through
  OpenUdon, Browsertools author-session/result freshness and synthesis,
  UWS 1.8 context, UWS 1.9 scalar and UWS 1.11 typed contracts, Browser
  1.8/1.9 templates, Udon/Browserdriver v10 handoff, and OpenUdon UWS 1.11
  output. V1 reports continue to use the unchanged historical scenario lock
  and gate inventory. `--installed-engines` and
  `--headed-auth` enable only the existing engine, authentication, and
  same-context authoring loopback fixtures and remain skipped when pinned
  components are unavailable.
- `openudon browser-scenario-eval` strict-decodes 23 embedded loopback, eight
  realistic journey, and four public manifests plus
  `openudon.browser-scenario-lock.v2`. The required
  loopback suite launches real headed Chromium through Browsertools v2 and
  external Udon/Browserdriver v3 without external network access. The required
  journey suite uses Browsertools guided-authoring v1, strict canonical-profile
  import, UWS 1.8, and external Udon/Browserdriver v3 against local headless
  read/write fixtures. The public
  suite is opt-in through `--allow-network`, uses only fixed anonymous HTTPS
  origins and Boolean presence outputs, and is informational. Loopback/public
  emit `openudon.browser-scenario-eval.v1`; journey emits
  `openudon.browser-journey-eval.v1`. All add a SHA-256 sidecar with no
  credential value, page content, or subprocess output.
  `make browser-scenario-loopback` and `make browser-scenario-journey` are
  included in `make release-saas-check`; `make browser-scenario-public` and
  its weekly workflow remain explicit-network checks. Scenario manifests,
  locks, and reports reject duplicate decoded JSON keys at every object depth.
  The v2 lock rejects dirty OpenUdon and pinned sibling worktrees (while
  explicitly ignoring generated `site/`) and binds the installed
  Playwright 1.62.1 package plus Chromium 151.0.7922.34; loopback success also
  requires server-observed authenticated replay. Push, number-match, passkey,
  and security-key evidence waits for Udon's exact prompt, latches one pending
  server session, and only then supplies approval. Runtime failures are
  classified from strict `udon.execution-report.v2` codes; malformed or
  unrelated failures remain `unclassified` and cannot satisfy negative cases.
  Hosted Ubuntu jobs explicitly enable sandbox-compatible user namespaces and
  never launch Chromium with `--no-sandbox`.
- M86 adds `--stack current|historical` to the scenario CLI. Historical remains
  the CLI/default local-qualification selection and keeps its original lock,
  corpus and v1 report readers. Make and hosted release checks explicitly use
  current for local loopback/journey: the current lock matches M85's published
  UWS 1.11 stack, and v2 reports select it by version. The current journey
  corpus has eleven cases, including Browser 1.8/1.9 templates and a mixed
  1.5/1.9 named v10 session; current passing verification requires the full
  23-case or eleven-case inventory. The Node readiness launch now requires
  `chromiumSandbox: true`. Browser execution acceptance is tracked by M86.
- A browser suite with no executed scenario reports `not_run`, which is valid
  for structural inspection but never a passing release result. Probe, build,
  and scenario subprocesses have fixed 30-second, two-minute, and three-minute
  deadlines and terminate complete process trees on Unix and Windows.
- A build-tagged `internal/icot/ui` qualification suite directly pins
  `github.com/mxschmitt/playwright-go` v0.6201.0 for test-only Chromium control.
  `make icot-ui-browser-check` exercises the real listener and is required by
  `make release-saas-check` and tag release automation. The engine and HTTP
  server have no Playwright import or in-process runtime; the default gate
  sets a sandbox-required control, rejects the disable override, and logs an
  enabled assertion. `make icot-ui-browser-check-unsandboxed` is the only
  diagnostic escape hatch for hosts whose kernel policy prevents sandbox
  startup and never counts as release evidence.
- Advisory API-source security sidecars named `*.security.{json,yaml}` or
  `*.security-overlay.{json,yaml}` may live next to package-local API source files. They are
  packaged as review/build evidence and included in handoff digests when present, but are not
  treated as executable API source contracts.
- Internal implementation: reusable behavior under `internal/`.
- `internal/authoring/atomicfile` provides file sync, atomic rename,
  parent-directory durability where supported, and cleanup on failures.
  `internal/evidencefile` owns bounded regular-file reads, duplicate/unknown/
  trailing-safe JSON decoding with depth 64 accepted and depth 65 rejected,
  canonical digest sidecars, and complete
  40/64-character Git object validation.
- Artifact formats: Markdown, HCL, JSON, YAML, UWS YAML, and review handoff JSON.
- Normal verification: early `GOWORK=off go build ./internal/icot ./cmd/icot`,
  `go test ./...`, `go vet ./...`, `make check`, and `git diff --check`.
  Local repository guard checks, UWS artifact validation, and trusted executor staging are Go CLI
  commands/packages under `cmd/` and `internal/`.
- Run evidence maintenance commands include `openudon run-evidence verify`,
  `openudon run-evidence archive`, `openudon release-notes draft`, and
  `openudon local-udon-smoke`. `openudon release-evidence` and
  `make release-evidence` wrap these into one local release-evidence workflow
  that writes compact JSON/Markdown summaries under ignored `.openudon-run`
  paths.
- Public CI verification: reject committed module replacements, run
  early `GOWORK=off go build ./internal/icot ./cmd/icot`,
  `GOWORK=off go mod download`, `GOWORK=off go vet ./...`,
  `GOWORK=off go test ./... -count=1 -timeout=10m`,
  `GOWORK=off go run ./cmd/openudon check-apitools-boundary`, and
  `git diff --check` through GitHub Actions without sibling checkouts.
- Public CI cross-builds `openudon`, `icot`, and `udon-runner` for Linux,
  macOS, and Windows on amd64 and arm64. Tag automation packages those three
  commands, README, and the Apache-2.0 license into six archives and publishes
  `SHA256SUMS`.
- Public docs publishing runs `mkdocs build --strict` before GitHub Pages deploy.
- Release verification: `make release-check` for deterministic gates,
  `make icot-ui-browser-check` for the embedded Phase C shell,
  `make browser-integration-check` for the provider-free browser boundary,
  `make browser-scenario-loopback` for the required real-browser local matrix,
  `make browser-scenario-journey` for realistic headless local workflows,
  `make browser-scenario-public` for explicit-network informational canaries,
  `make release-saas-check` for the consolidated local release gate, and
  `make release-eval` for opt-in real-provider eval gates. A11's mandatory
  release qualification has passed with sandbox-required Chromium; the
  separately named sandbox-disable target remains diagnostic-only evidence.

## Module Dependencies

OpenUdon is published as a normal Go module. Its committed `go.mod` must use public module versions
for:

- `github.com/OpenUdon/uws` for the public UWS model, schema lookup, document loading, schema
  validation, and artifact discovery helpers.
- `github.com/OpenUdon/apitools` for API source metadata discovery, materialization, search,
  indexing, auth/security summaries, operation ranking, lifecycle-role ranking over operation
  summaries, optional provider/spec/security catalog advice, and catalog
  protocol-to-UWS-source-type mapping.
- `github.com/OpenUdon/browsertools` for engine-neutral browser
  capability/authentication/registration-profile, registration-review, and
  capability-bundle validation, bounded
  local discovery, local/HTTPS static registry search/pull, protocol/result
  types, and the `authorworker` process entry. Only the hidden worker invokes
  Browsertools' capture/Playwright implementation; OpenUdon does not import
  private cache payloads or expose browser objects to the engine/server.
- `github.com/OpenUdon/evidence` for product-neutral digest records, artifact manifests,
  diagnostic records, redaction helpers, approval evidence primitives, and neutral async execution
  records used behind OpenUdon-owned package, approval, and run-evidence contracts.
- `github.com/OpenUdon/authoring` for the product-neutral interview graph, deterministic frontier,
  round engine, clone-based interview transaction binding, prompt/session defaults, structured JSON
  fallback, lifecycle atomic writes,
  `authoring.prompt-context.v2` OR-of-AND security alternatives, prompt-safe
  lifecycle records, agent result contracts, report metadata, and scorecard
  summaries. OpenUdon keeps workflow-specific graph construction, prompts, intent, v2 adapters,
  reviewed source staging, repair rules, proposal/draft lifecycle, authoring-eval categories, and
  trusted-runner handoff.
- `github.com/OpenUdon/asyncapi` for native AsyncAPI source metadata parsing and selector
  compatibility checks.
- `github.com/mxschmitt/playwright-go` v0.6201.0 is a direct requirement for
  build-tagged UI qualification and also enters `icot` transitively through
  Browsertools' bundled worker. Only the worker process initializes it.

Local sibling development belongs in the parent `go.work`, not in committed `replace ../...`
directives. Verify public-module behavior with `GOWORK=off go test ./...` and
`GOWORK=off go vet ./...`.

The public module records reviewed UWS commit
`895aa4546067e25f9dd525b1356abf1945d223b4` and the published no-submit
Browsertools producer commit `39e32c1d6f601561cc5c13ec85201815ce85ab9b`
as exact pseudo-versions. Their versions are
`v0.0.0-20260825191727-895aa4546067` and
`v0.0.0-20260825225202-39e32c1d6f60`. The browser-scenario compatibility lock
uses those revisions together with the reviewed Udon and Browserdriver
checkouts. Local sibling development remains in the parent `go.work`;
committed local `replace` directives are prohibited. The Go pins supply the
UWS 1.9/browser 1.7 schemas, authenticated-authoring v2 producer, hardened
validation/conversion behavior, and canonical
`authorsession.ReduceAccessibilityLabel` policy and the isolated A16
`authorworker` plus the shared disclosure-path validator. A fresh proxy-only
module cache resolves the hardened Browsertools revision, and standalone
OpenUdon build, test, and vet pass without a local replacement.
An A03 handoff executed separately still requires a compatible standalone
Browsertools binary; iCoT never installs a driver or Chromium.

The authenticated-authoring A06 qualification binds all five exact revisions
in its generated report. The UWS and Browsertools revisions above are
published on their ordinary module origins. Normal `GOWORK=off go list -m`
resolved both pseudo-versions without URL mappings or `replace` directives.

Hosted public-canary and tag-release browser-scenario jobs clone the public
Browsertools and Browserdriver lock revisions anonymously. Their locked Udon
revision comes from the private `genelet/udon` repository through the
repository Actions secret `GENELET_READ_TOKEN`, which must have read-only
Contents access to that repository and no broader authority; checkout does not
persist the credential. The current-repository `GITHUB_TOKEN` is insufficient
for this cross-repository private checkout.

Udon is not a OpenUdon Go dependency. When used, it is an external trusted executor selected at runtime
by canonical `OPENUDON_EXECUTOR`, which accepts an absolute executable path or `docker://<image>`.
`OPENUDON_UDON_RUNNER` is separate and overrides the outer runner shim; it must be an absolute
executable path. Docker browser runs validate the operator-supplied host
Browserdriver executable, mount it read-only at `/openudon/browser-driver`,
and translate the Udon argv to that container path.

## Preferred Dependency Direction

- Use `github.com/OpenUdon/uws/versions` and `github.com/OpenUdon/uws/validation` for public UWS
  schema lookup, document loading, schema validation, and artifact discovery. `UWS_SCHEMA_DIR`
  overrides schema lookup; `OPENUDON_UWS_SCHEMA_DIR` remains a compatibility alias.
- Use `github.com/OpenUdon/apitools` APIs only for API source metadata
  discovery/materialization/search, indexing, summaries, ranking, optional catalog advisory
  metadata, generic lifecycle-role ranking, and deterministic protocol-to-UWS-source-type mapping.
  Runtime parsing and execution of
  Google Discovery and AWS Smithy stays in the trusted executor boundary.
- Use `github.com/OpenUdon/asyncapi` only for metadata parsing and local source selector
  validation. Runtime AsyncAPI protocol behavior stays in the trusted executor boundary.
- UWS 1.4 GraphQL, OpenRPC, gRPC/protobuf, and OData generation is enabled for reviewed local
  source artifacts and apitools operation summaries. Protocol execution and credential resolution
  remain trusted-executor responsibilities.
- Do not import parser/conversion packages, infrastructure engine internals,
  provider plugins, provider SDKs, or desired-state conversion packages.
  Desired-state conversion belongs in `../ramen`.
- Do not import private executor packages such as `github.com/OpenUdon/udon`,
  `github.com/genelet/udon`, or private `genelet/*` runtime modules from OpenUdon source. Use the
  trusted executor handoff instead.
- Keep OpenUdon-owned authoring, iCoT, review evidence, approval, credential scanning, package digest,
  and handoff helpers under `internal/`.
- Use `github.com/OpenUdon/evidence/...` only for neutral shared evidence primitives. Keep
  `openudon.approval.v1`, `openudon.run-evidence.v2`, `openudon.async-evidence-bundle.v1` sidecar
  references, review handoff manifests, package digest policy, tier rules, trusted-runner checks,
  release evidence archive/draft helpers, local udon smoke orchestration, and executor handoff
  behavior in OpenUdon-owned packages.
- Use `github.com/OpenUdon/authoring/...` only for neutral shared authoring mechanics, including the
  interview graph/frontier/round engine and clone-based frontier settlement binding. Keep OpenUdon
  graph construction, iCoT prompts, workflow
  intent, reviewed source staging, package artifacts, v2 wire adapters, authoring-eval categories,
  lifecycle prompt/detail planning, proposal/draft lifecycle, and repair behavior in OpenUdon-owned
  packages. Lifecycle expansion receives a
  prompt-safe full-catalog context, but OpenUdon keeps detailed prompt context
  bounded to selected, inferred sibling, and explicitly requested operation
  documents.
- Generic prompt-safe lifecycle-role ranking belongs in
  `apitools/operationlifecycle`; keep workflow selection, prompt policy, and
  artifact lifecycle in OpenUdon and do not restore an Authoring compatibility
  shim.
- OpenUdon generates public UWS documents directly and invokes executors only through UWS Document +
  staged API source files + non-secret run config + runtime credential resolver.
- Keep udon runtime-plan helpers and private HCL body representations such as `hcllight/light.Body`
  behind udon APIs. OpenUdon may consume public maps and UWS documents, but must not inspect udon's
  private runtime-plan or HCL AST types directly.
- Keep public workflow semantics out of OpenUdon and put them in `../uws`.
- Keep generic execution/compiler behavior out of OpenUdon and put it in `../udon`.
- Keep concrete IaC models and `.tf` rendering out of OpenUdon; use `apitools` only for
  API source metadata tooling.

## Artifact Contracts

- OpenUdon examples use `project.md`, API/event source directories (`openapi/`, `google-discovery/`,
  `aws-smithy/`, `asyncapi/`, `graphql/`, `openrpc/`, `grpc-protobuf/`, `odata/`, with legacy read
  support for `discovery/`), `workflows/`, and `expected/`.
- `workflows/intent.hcl` is OpenUdon's structured authoring contract.
- `workflows/workflow.hcl` is public UWS HCL.
- `workflows/workflow.uws.yaml` is equivalent public UWS YAML.
- `expected/plan.json` and `expected/plan.md` describe the expected workflow behavior.
- `expected/data.hcl` optionally records reviewed non-secret runtime input values under an
  `inputs { ... }` block; when present it is included in handoff manifests, package digests, and
  trusted-runner staging. Existing non-`inputs` blocks are preserved across build regeneration so
  operators can keep reviewed env-reference markers, for example
  `client_secret = "ENVIRONMENT:GOOGLE_CLIENT_SECRET"`, without storing secret values in package
  artifacts. When a package declares the `googleOAuth2` credential binding, build emits a
  non-secret `credentials { googleOAuth2 { ... } }` placeholder that points to `GOOGLE_CLIENT_ID`,
  `GOOGLE_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URL`, and `GOOGLE_REFRESH_TOKEN`.
- `expected/refinement.json` and `expected/refinement.md` record bounded repair attempts.
- `expected/review.md` records human review evidence and trusted-runner command text.
- `expected/quality.json` and `expected/quality.md` record deterministic gate results.
- Legacy packages may contain `apitools.review-handoff.v1` for migration inspection.
- Executable v0.2 packages use `apitools.review-handoff.v2`, with SHA-256 on
  every input and a canonical cleared-field self-digest. v1 handoffs are
  read-only migration inputs and cannot execute.

## LLM And Provider Policy

- Local real-LLM synthesis and iCoT authoring default to the `copilot-api` OpenAI-compatible proxy
  at `http://localhost:4141` with model `gpt-5.4-mini`.
- Escalate to a larger model only after the default proxy model fails deterministic checks.
- Shell-level provider/model defaults use `OPENUDON_LLM_PROVIDER` and `OPENUDON_LLM_MODEL`, matching
  the optional iCoT pattern used by sibling OpenW8M. Explicit CLI `--provider` and `--model` flags
  take precedence.
- Provider credentials and proxy endpoints come from environment variables such as
  `COPILOT_API_BASE_URL`, `COPILOT_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, or
  `ANTHROPIC_API_KEY`.
- Provider-specific API keys do not make iCoT auto-select that provider; set `OPENUDON_LLM_PROVIDER`
  or pass `--provider` for explicit non-default providers such as Gemini.
- Gemini sends credentials through `x-goog-api-key`, not the URL. All provider
  success/error bodies are limited to 8 MiB, preserve caller deadlines, and
  redact transport errors.
- `cmd/icot --prompt-mode full|normal|fast` controls frontier answer collection. `full` shows and asks
  every ready question, `normal` shows the full frontier and visibly accepts safe defaults, and
  `fast` silently accepts safe defaults while still showing missing, low-confidence, conflicting, or
  forced decisions. The final proposal approval is forced in all modes; `--yes` is explicit
  noninteractive approval.
- `cmd/icot --api-source KIND:ID=PATH`, `--openapi ID=PATH`,
  `--browser-profile ID=PATH`, `--browser-verification PATH`,
  `--browser-registry PATH|HTTPS_URL`, and
  `--source-root PATH` are repeatable reviewed source inputs. API-capable
  operations take precedence over browser fallback. `--browser-profile`
  accepts capability profiles/bundles, explicit guided-authoring results, and
  local secret-free authentication profiles. Guided results are explicit-file
  only and are reduced to profiles; authentication profiles are never pulled from or published to the
  static capability registry. `--network
  never|ask|allow` defaults to `ask` interactively and is
  effectively `never` in agent mode unless `allow` is explicit. Local discovery is bounded to 10,000
  visited entries, 100 candidates, and 20 MiB per file by default; remote lookup is one approved
  curated-catalog plus APIs.guru pass with an eight-second deadline and at most
  three candidates. Configured HTTPS browser registries use a separate approval
  decision and the same deadline/result/file-size bounds; local static
  registries remain offline.
- `cmd/icot --agent --json` is the noninteractive local-agent surface. It never prompts or writes
  deliverables; it reports the full frontier, workflow candidates, source evidence/blockers, and
  proposed file actions, including proposal approval for complete state. `cmd/icot lint --json`,
  `cmd/icot scorecard`, and `cmd/icot repair --json` emit
  versioned JSON reports for reliability scoring and bounded remediation. `cmd/icot variants
  validate` checks authoring variant metadata and reference-seeded clear slots without running the
  scorecard. `cmd/icot variants coverage` requires every provider family represented in
  authoring variants to have positive, missing-detail, and unsafe-negative coverage. `cmd/icot scorecard
  --include-variants` also reads `reference/authoring-variants.json` and summarizes provider-free
  reference/variant package coverage by provider family, variant class, failure family, and top
  readiness issue; `needs_input` variants must declare expected top issue code and slot, and the
  scorecard enforces exact matches. Scorecard reports also carry run ID, commit, command,
  prompt/readiness provenance, explicit missing-detail or unsafe false-pass counters, a
  `needs_input` diagnostic-gap counter, SHA-256 digest sidecar, and retention/share-safety metadata
  marking scorecards as provider-free `release_evidence` that is safe to archive. `cmd/icot
  authoring-eval` is the optional real-LLM authoring evidence lane; it records provider/model, run
  ID, command, commit, prompt/readiness versions, LLM call count, generated paths, drift counts,
  per-variant pass/fail, and structured failure categories, scans generated
  project/intent/transcript/report output for credential-like literal values, writes a SHA-256
  digest sidecar, and marks reports as local ephemeral provider-output evidence requiring redaction
  review before sharing. It is intentionally not part of provider-free release gates. `cmd/icot
  report verify --file ...` revalidates scorecard or authoring-eval JSON, summary counters, variant
  top-issue expectations, failure categories, retention/share-safety metadata, and adjacent SHA-256
  digest sidecars after generation or archival; `make icot-authoring-scorecard` and `make
  release-saas-check` run it for the provider-free scorecard, while authoring-eval verification
  remains optional/manual real-LLM evidence. `make release-saas-check` also runs the variant
  coverage gate before producing scorecard evidence.
- `cmd/icot replay-eval` defaults its own `--prompt-mode` to `fast` and reports actual transcript
  prompt counts, auto-accepted defaults, LLM calls, repair attempts, rejected repairs, and unresolved
  final review warnings.
- With LLM extraction enabled, iCoT runs an advisory pre-final flow review before showing the current
  draft. The review is structured JSON and only reports cross-step data-flow warnings that
  deterministic request/schema checks may miss. By default it does not mutate the draft; advisory
  review warnings are also written as non-executable comments in the generated `intent.hcl`.
- iCoT uses `openudon.icot-session.v2`, `openudon.icot-transcript.v2`, and v2 OpenUdon
  author/lint/repair/scorecard/authoring-eval/variant/replay wires. The generic graph remains
  `authoring.interview.v1`; v1 OpenUdon inputs are rejected without compatibility decoding.
- Weather/report/Gmail drafts get a deterministic `render_weather_report` `fnct` transform when
  exactly one weather producer and a Gmail delivery sink are present. It selects `gmail.render_raw`,
  passes a request-body object, and relies on the trusted executor to register the public helper.
- API-source security scheme names are the executable credential-binding source of truth. When a
  generated or edited intent maps a security field to a stale document-derived binding, build
  normalizes that request mapping to `credentials.<securitySchemeName>` before emitting workflow
  artifacts.
- Operation security uses Apitools `SecurityRequirementSets` and Authoring
  `CredentialBindingSets`: outer sets are OR, bindings within a set are AND,
  and an empty set is anonymous. iCoT persists one canonical SHA-256 fingerprint
  before request mappings, exposes only the selected set's fields, and treats
  prompt-budget loss of this structure as a deferable readiness blocker.
- `cmd/icot --review-repair` is an experimental opt-in mode. It can make at most two bounded
  pre-final review repairs, and only to request mappings, output sources, and `depends_on`; source,
  operation, credential, and side-effect-scope mutations are rejected and recorded in the transcript.
  M65 adds one narrowly proven exception for deterministic API prework: a read-only local GET/HEAD
  operation may be inserted when it is the only metadata-listed producer for a missing request field
  and requires no inputs. Response-field repair can use local OpenAPI `$ref`, nested object, array,
  and Swagger `schema` response metadata.
- iCoT sessions use one unified interview evidence ledger for observed facts, user decisions,
  recommendations, assumptions, open decisions, deferrals, and inapplicable branches. It contains
  concise public rationale, never hidden chain-of-thought. OpenUdon's ledger
  adapters also retain draft operation-detail refs and structured draft events
  so headless round autosave/resume reconstructs all authoring provenance.
- The headless engine accepts exactly one complete current frontier per
  `ApplyRound`, previews entirely in memory, and requires a non-default explicit
  human approval value before the shared writer can create deliverables. It
  canonicalizes each answer against the current question slots, returns
  deep-cloned snapshots, transactionally refreshes reviewed sources, requires
  registry selections to be freshly rediscovered at approval, and derives
  approval-capable proposed actions from the prepared writer file plan. It
  returns typed rejected/conflict/operational/indeterminate failures and
  fingerprints engine-owned workspace files across mutations. Pre-refresh
  observation covers only current watched paths and current local/registry
  candidate targets; SHA-256 is streamed with context checks and stable-file
  identity/type/size/mode/modification verification. Unobserved new targets use
  a conservative missing baseline. Round state and snapshots are built before
  draft persistence; approval builds its exact snapshot and prepared plan
  before commit, then constructs the write result directly from the commit
  outcome without a fallible post-commit refresh.
  It does not run live Browsertools orchestration; reviewed profiles,
  authentication profiles, static registries, and value-free verification
  reports remain its browser inputs.
- `icot ui --example DIR` owns one engine and uses explicit `--answers` or
  `--from-example`, `.icot/session.yaml`, existing final project/intent, then
  empty state precedence. It accepts the same reviewed API/browser source
  flags plus `--port` and `--no-open`, always binds `127.0.0.1`, and exposes an
  authenticated experimental v3 journey/acquisition/capture/authoring/package/
  handoff API plus a polling accessible embedded authoring and review shell;
  v1/v2 routes are not served.
  Callers submit only question IDs and values; the engine supplies frontier
  slots. Exact revisions serialize mutations, `If-None-Match` (including `*`) avoids unchanged
  snapshot bodies, external workspace drift blocks mutation until restart, and
  any approved final or incomplete write freezes the process. The server uses
  32 KiB headers, five-second header, 15-second read, and 30-second idle limits
  without a global write timeout. Snapshot `write_conflicts` are a read-only
  exact-path preflight used to require explicit overwrite review; authorization
  and replacement remain inside the transactional writer.
- Public iCoT format docs cover v2-only `--answers` sessions, ignored v2 transcript JSON, adaptive
  rounds, approved draft/final lifecycle, local/remote source discovery, and the recommended
  `project.md` section schema.
- Never paste credentials into prompts, commands, examples, review evidence, or eval artifacts.
- Real-provider evals are local/manual because they spend quota and may produce artifacts that need
  redaction review.

Provider drift release evidence should record provider, model, commit, comparison baseline,
`provider_drift_watch.status`, structured fallback count, maximum attempts, release-gate result,
and any provider error text. Do not loosen release criteria from a single transient run; rerun once
from a trusted workstation if the error looks external.

## Trusted Runner Contract

`openudon approval-template` prints approval JSON with:

- `version`: `openudon.approval.v1`
- `scope`: example path relative to repo root
- `state`: `approved_for_sandbox` or `approved_for_production`
- `reviewer`, `approved_at`, optional `expires_at`, optional `notes`
- `package_sha256`: canonical digest of required handoff inputs

The neutral `github.com/OpenUdon/evidence/approval` package is available for shared approval
evidence primitives, but this `openudon.approval.v1` shape and its tier/state semantics remain the
OpenUdon trusted-runner contract.

`openudon run` validates approval, package digest, stored and current quality, manifest policy,
credential-value prohibition, direct-production prohibition, and tier/state compatibility before
writing `openudon.executor-run.v2`. The run config includes a unique run ID,
approval/handoff/package digests, and sorted `package_paths` for every
digest-covered required handoff input and `data_files` for reviewed runtime data files such as
`expected/data.hcl`. It carries first-class `api_source_paths` for staged API source documents and
continues to populate legacy `openapi_paths` with the same values for executor compatibility during
the API-source transition. The digest covers the reviewed UWS artifacts, every regular API source
file staged for execution, advisory security sidecars, and reviewed runtime data files. Dry runs and real handoffs
stage the reviewed package paths into a fresh executor-visible directory under the configured run
workdir and recompute the approved package digest from the staged copy. Dry runs stop before
credential-value checks and executor invocation. Non-dry runs fail if a declared credential binding
is missing from the local `UDON_CREDENTIAL_*` environment, reject direct runner configs that omit
`package_sha256` or `package_paths` or set `direct_production_run: true`, then invoke either a trusted
absolute-path udon-compatible binary by typed invocation or a Docker image via
`OPENUDON_EXECUTOR=docker://<image>`. Local executors receive only declared
credential values. Docker and outer runners add only the documented minimal
platform launcher environment; cloud, proxy, SSH-agent, and unrelated process
variables are excluded. API-first plans may retain reviewed browser fallback
profiles without acquiring a browser runtime config. The package digest
covers reviewed workflow artifacts, staged API source files, and any associated advisory security
sidecars used as build credential evidence.

`openudon run` also writes `openudon.run-evidence.v2` inside a unique run directory. Evidence is
non-secret and records scope, tier, approval state, package digest, config path, staged package path,
package/API paths, credential binding names, derived credential env names, gate outcomes, stage kind,
executor status, whether the final executor was invoked, and optional `async_evidence_files`
references. Each run writes one `<workdir>/async-evidence.json` sidecar with version
`openudon.async-evidence-bundle.v1`, containing neutral `evidence/async` execution request and
response records for the package handoff. Successful dry-run preparation or runner invocation
records `accepted`; invocation failure records `fatal_failure`. `async_evidence_files[].path` is
workdir-relative for archive use, while CLI output prints the resolved local sidecar path. The
sidecar forwards OpenUdon run lifecycle evidence only. It does not store credential values, raw
stdout/stderr, Ramen resource addresses, desired hashes, convergence outcomes, or state semantics.
When `OPENUDON_UDON_RUNNER` overrides the outer runner shim, evidence uses
`stage_kind: preflight`. The external runner requires `--config`,
`--config-sha256`, and `--approval`, then repeats current quality, handoff,
package, approval, tier, canonical-byte, credential, staging, and digest
validation. v1 configs are rejected.
`openudon run-evidence verify --file run-evidence.json` validates archived run evidence, sidecar
relative paths, SHA-256 digests, record counts, and strict async request/response shapes. The
sidecar schema is documented at `docs/schemas/openudon.async-evidence-bundle.v1.schema.json`.
Udon M35 replaces the legacy report with strict `udon.execution-report.v2`.
Failures require one closed typed `error_code`, successes contain none, and
missing/malformed/v1/unknown/unrelated failures classify as `unclassified`.
OpenUdon ingests valid reports into neutral async status and confirmation-read
observations. Udon remains an external runtime selected by argv/Docker;
OpenUdon does not import private udon packages or parse raw executor stdout/stderr.
Executor reports must be bounded regular workdir-relative files and are bound
by digest and size. Optional PKCS#8/PKIX Ed25519 signatures are detached from
evidence; trusted-public-key verification is the operator-identity claim.

## Tooling Constraints

- Keep `cmd/openudon` and `cmd/icot` thin.
- Keep scripts small wrappers around Go behavior where practical. Prefer `cmd/openudon` subcommands
  for local repository checks.
- Do not add product-specific behavior to `../uws` or core `../udon`.
- Prefer deterministic checks and synthetic fixtures over live-provider tests during development.
- Keep generated eval outputs, readiness reports, approvals, transcripts, autosaves, and run
  workdirs ignored unless explicitly converted into reviewed fixtures.
- Keep release note evidence in Markdown templates and ignored report paths; do not commit
  real-provider JSON/Markdown eval outputs.
## Harness Runner

```bash
../skills/harness/tackle-memory-bank-api-loop --model lane-audit .
```

## M79 local browser system gate

`make browser-system-check` runs separate offline and explicit loopback suites
and independently verifies their aggregate JSON reports. Loopback requires
three fresh complete passes with no skipped browser prerequisite. Default
unit tests remain browser-free. `icot control --no-open` exposes the same
registration application decisions over bounded private NDJSON. The upstream
Chromium/Linux, Playwright-Go v0.6201.0 and Playwright 1.62.1 baseline remains
unchanged. Exact auxiliary Udon source inputs and the administrator-owned
sandbox helper are prerequisites; no gate installs or upgrades them.

The qualified M79 gate also requires 88 Udon browser-contract and 19 browser
CLI race cases, actual supervised iCoT processes, 23 loopback scenarios, eight
journeys and both existing package qualification components on each pass.
Local acceptance and review iteration 5 pass with exact Go 1.26.6/Node 24.13.0
recorded and the supplied Grand/Hcllight/Golet closure.

The opt-in `openudon.application-control.v1` transport uses owned private
pipes. See [architecture.md](architecture.md#icot-architecture) for the shared
lifecycle and [M80](../docs/history/status-M80.md) for qualification evidence.

M81/W09 adds opt-in `openudon run --interactive-browser` and forwards the private
response stream through either trusted runner to Udon's existing interaction.
The run-config schema stays unchanged and default execution has no input stream.

Native `openudon.browser-system-qualification.v2` retains three loopback passes
and adds `supervised_registration_package` and `supervised_authenticated_package`
for 13 stages per pass. Original v1 reports keep their eleven-stage inventory.
The W8M-owned complete gate runs three native-plus-target units and verifies
all embedded native reports in canonical two-space JSON with a terminal newline.
M81/W09 local acceptance passes 117 native stages and nine target workflow
receipts with the unchanged Chromium/Playwright baseline. Default unit tests
remain browser-free; no package installation or dependency upgrade is implied.

The W10 publication followup retains native failure diagnostics in an
owner-only `<report>.diagnostic.json` sidecar outside source trees. Each captured
child stream contributes at most its final 1 MiB, with explicit truncation. Fixed
failure reasons distinguish subprocess, teardown, deadline/cancellation and
output-limit failures. Reduced report schemas and inventories are unchanged;
the consumer must preserve the private sidecar before temporary cleanup.
The existing Linux supervisor starts a separate direct-child monitor before
background whole-host scans, preserving PID/start-time identities and joining
both monitors during teardown. This prevents a slow full scan from suspending
the fast discovery path; it adds no general executor or browser capability.

The additive openudon.browser-registration-attestation.v2 schema and internal decoder require one prior attempt, delete_separately, a whole-second UTC expiry within twenty minutes, five SHA-256 links and the literal submission_not_started outcome. Existing v1 rejects the recovery field, including null.

The additive v3 schema and decoder require exactly two prior attempts with the
same closed digest links and expiry limit. V2 remains restricted to one prior
attempt. Native artifact and trusted-runner tests cover all three versions;
fresh complete consumer qualification remains mandatory before live adoption.

## E13 development commands

`make fast` uses normal Go test-result caching and document checks.
`browser-system-dev --mode smoke --stage registration_ui_handoff --udon-repo
/absolute/prepared/udon --out /tmp/unique.json --cache /absolute/private/cache`
runs one fresh synthetic flow. `--reuse` explicitly permits matching successful
results younger than 24 hours, preserving execution ID/time. Empty `--cache`
bypasses custom caching. `make smoke` exposes the corresponding
`OPENUDON_SMOKE_*` variables; `make qualify` aliases the existing fresh complete
gate. The native report schema is unchanged; new development/timing schemas are
`openudon.browser-development.v1` and `openudon.browser-check-timing.v1`.
Private JSONL sidecars measure source hashing, subprocesses, selected builds,
stages and transaction teardown. No new dependencies or downloads are required.
