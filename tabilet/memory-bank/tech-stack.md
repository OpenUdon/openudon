# Tech Stack

## Approved Stage 11 tooling target — not implemented

[Stage 11](../../../kinet/docs/stage11.md) and [local milestones](milestone.md#stage-11-cross-package-refactoring) define the approved target. Exact published module revisions, frozen build closures and standalone verification are acceptance gates. go test ./...; go vet ./...; make check; API/import-boundary and trust-wire fixtures; affected exact-pin consumer checks; git diff --check. Use owner-required offline browser smoke/qualification only for affected retained browser paths.
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

## Retained Browser 1.10 qualification stack (E22/M91)

This retained context pinned published UWS M05
`80ee9bfb24a688b5e875dadf9ecacdc65398f1ff` and Browsertools M32
`3abe70efc03d9ccb97b8b30e5e86328f60a70c64`. The exact Browserdriver M15 and
Udon M43 source pins, module versions, and separate 14-repository Udon build
closure are recorded in
`internal/browserscenario/current-compatibility-lock-v4.json` (SHA-256
`58363021e44961527468bc686df114ce69770709345eb39702fbf38e84da6d2a`) and
`current-qualification-build-inputs-v4.json` (SHA-256
`10fa8b2570f0a72688a1c8d282fe84cf7ea4af6e82ad5ffa85aa6fc994ff371a`). The
E22 scenario, integration and native selectors emitted v4 reports; its
journey suite has 14 cases, including the three Browser 1.10 count
journeys through Udon v11. Its v4 integration matrix requires named count
profile, producer, schema, Udon v11 consumer, and Browserdriver extraction
tests; v2/v3 reports retain their original gate inventory. Fresh full E22
qualification and bounded review passed at clean OpenUdon
`9be9ff3f195ecaa8bdac88cc8616c5fc345dfeb3`; report digests, exact source/runtime
bindings and review evidence are in the
[E22 history record](../docs/history/status-E22.md).
UWS maintains the profile-version checklist at
[`future-source-profiles.md`](../../../uws/docs/future-source-profiles.md#adding-a-browser-profile-version).

## M86 v2 lock snapshots

The M86 current scenario v2 verifier reads
`internal/browserscenario/current-compatibility-lock-v2.json` (SHA-256
`57ebe6c70bc0b1e810ed4fb490f36ecb227c47f362bb738b56680ce7abcd6a77`). The
integration v2 verifier reads its own byte-preserved snapshot at
`internal/browserintegrationeval/current-compatibility-lock-v2.json` (SHA-256
`9eec17f1489e1c805e2d2bfb8b89a439ee7153d5c49ec09a3ee903ce6761d393`). The
frozen E21 current v3 scenario and integration locks are retained at
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
digests; E21 evidence is recorded in [status-E21.md](../docs/history/status-E21.md).

Native `openudon browser-system-eval --stack current --suite loopback` selects
M92 v5 compatibility and 14-source build-input locks, requires all primary
and auxiliary worktrees to be clean, and emits
`openudon.browser-system-qualification.v6`. Scenario/journey use v5 and
integration uses v7. The retained v4 reader uses E22/M91 snapshots; v3 uses E21.
The effective UWS pin is published 1.12 `a7688f54c68f5a75c7cc95aa2b31cea98b31af41`;
Browsertools retains its declared 1.11 dependency edge, separately recorded
in the new lock. Accepted Udon M45 supplies the exact executor closure. Historical native v1–v5 reports remain verifiable; removed UI gates cannot execute from this source.
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
The Node readiness probe, native Browserdriver registration test stage, and
Browserdriver execution subprocess use `--preserve-symlinks` so Playwright
resolves its lock-matched `playwright-core` sibling through the staged
`node_modules` link.
The integration npm-test gate copies the already-validated module tree into the
disposable Browserdriver clone as a read-only `node_modules` directory. This
keeps Node and TypeScript dependency lookup beside the package when tests launch
child processes with a minimal environment; no package installation runs.
Udon Go test gates clone its exact current source and fourteen locked sibling
inputs to a temporary workspace; the native runner removes it before passing
the stage and rechecks the original closure.
Native public registration and authenticated package fixtures
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

## Non-interactive step authoring (M87/M89)

The `step source add`, `step candidates`, `step check`, `step bind`, and
`flow-review` commands use strict versioned JSON request/result contracts and
bounded safe reads.
`step source add` accepts up to 16 explicitly selected local files (8 MiB each,
32 MiB per request), validates them with APItools' local source inventory, and
atomically writes create-only source files plus the digest-bound
`expected/api-source-manifest.json`. Appending requires the exact current
manifest SHA-256. The manifest is included in package handoff inputs; results
omit absolute source paths and content. Source-add results provide both the
manifest ID and path-derived candidate ID. Kinet owns the preceding
user-confirmation gate.
Candidates scans the eight supported local source families, including the
legacy `discovery/` directory as a Google Discovery alias, and calls published
APItools operation-candidate metadata at adopted M81/M80
`v0.0.0-20260930205753-fb132631c982` (commit
`fb132631c9827eae5f2ec4503d03f21eabfb4113`); the legacy candidates path still
does not fetch URLs. It preserves
consumer summaries, compatibility evidence, source-backed effect classes,
authentication alternatives, and source capabilities while omitting paths
from results. Check and bind match the exact APItools source kind, ID, digest,
native selector, and operation key from the pinned candidate API and verify digest-bound
intent/source revisions, mappings, outputs, dependencies, and OR-of-AND
authentication alternatives, including self-reference and prerequisite-cycle
rejection. Check also uses APItools' exact-operation effect
evidence: a known conflicting class fails, and unknown remains indeterminate.
Compound read/mutation wording and unrecognized leading actions stay unknown.
Response nullability is field- and ancestor-scoped, so a nullable selected
output remains indeterminate without compatibility points while an unrelated
nullable sibling does not downgrade other outputs.
Explicit nested and renamed request mappings are checked from the exact
source field metadata against the corresponding contract path, including type,
format, and requiredness; workflow input root declarations are checked
separately. Explicit output mappings map contract paths to actual response
paths and receive the same type, format, requiredness, and nullability checks.
Omitted check output mappings retain same-name behavior for v1 compatibility.
Kinet retains confirmed output mappings outside UWS intent and resends them to
`step check`. `step bind` requires supported source input/output capabilities
and refuses unresolved or incompatible mappings; APItools' name-based
candidate score is advisory and no longer blocks an explicit, proven alias.
Check is read-only; bind replaces or adds one step in `workflows/intent.hcl` through
`internal/artifactwriter`, using optimistic SHA-256 checks and create-only
installation for an initially absent intent. Bind locates path-free source IDs
by hashing contained regular source files under the matching source-family
directory; the Google Discovery alias is included and supported security
sidecars are excluded. Symlink traversal is rejected. Credential bindings are symbolic
and lower to `credentials.<name>` request mappings; output paths are checked
against the selected operation response. HTTP verbs alone do not imply a
read/write classification.

`flow-review` reuses iCoT's deterministic local review and optionally its
existing chat extractor. Model review must name an explicit provider and model;
provider credentials remain process-environment-only. Credential-shaped
context is not sent, result messages and diagnostics are bounded and filtered,
and skipped/unavailable/failed/cancelled review states are not conflated with a
passed model review. The command is read-only and preserves existing iCoT
callers' behavior. The v1 schema and synthetic consumer fixtures are published
under `docs/step-authoring-contract-v1.md` and
`docs/examples/step-authoring/v1/`.

## Language And Runtime

- Registration 1.1 pins published UWS `9ff877ebce55`, Browsertools
  `ec0b9e9d6ca1`, Udon `5ef6af99430c` and Browserdriver `9d13e8b35394`.
  Actual browser qualification uses Node 24 and the existing Playwright runtime.
  Run `go test -tags browser_system_qualification ./internal/browserscenario
  -run TestTypedRegistrationUIToTrustedRuntime` for the complete synthetic typed
  journey. Values and private snapshot digests are scanned out of retained
  package/run artifacts. Native spinbutton locators remain outside immutable
  BRP 1.1; numeric scalar fixtures use supported textboxes.

- Primary language: Go, with module directive `go 1.26.6`.

- Module path: `github.com/OpenUdon/openudon`.

- Headless authoring lives under `internal/authoringengine`. Its internal methods
  use `context.Context` plus typed configuration, snapshots, acquisition,
  round, preview, approval, resume, and capture-stage records. The bounded API
  upload mutation accepts an `io.Reader`; no terminal reader/writer or browser
  object enters the engine. Snapshots are JSON-marshalable but are not a
  published schema or supported Go API.

- `internal/artifactwriter` is the common neutral authoring transaction for
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
  `uws1.ContentTrust` (supported since 1.9.1). New workflows declare UWS 1.12.0;
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

- OpenUdon also scans and stages verified `uws.browser.1.5` through `1.10` profiles under
  `browser-profiles/`, emits UWS `browser-profile` source descriptions, and
  records prompt-safe source review metadata in `.icot/browser-sources.json`.
  Browsertools owns validation, private cache, bundles, discovery, and the
  service-free static registry; Udon owns drivers, sessions, and execution.

- Repeatable `openudon authoring draft --browser-verification PATH` accepts only strict, bounded
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
  declare UWS 1.12.0. Browsertools owns local validation; Udon and its persistent
  Browserdriver own credential resolution, MFA challenge interaction, session
  state, and execution. Active Browser 1.8/1.9 actions select private
  browser-driver v10, which carries older actions as inner v2. Browser 1.10
  count actions select v11, matching Udon's typed count consumer. An active
  older/1.10 mix fails before execution. Older-only workflows keep their prior
  protocol selection.

- OpenUdon scans reviewed `uws.browser-registration.1.0` profiles and their
  digest-bound `browsertools.registration-review.v1` bundles under
  `browser-registration/`. Explicit `browser_registration` intent lowers to
  `uws.browser-registration-call.1.0` with exact symbolic bindings and fixed
  duplicate/ambiguity/cleanup policy. Package, quality, approval-template, and
  trusted-runner dry-run are supported; executor argv construction rejects the
  workflow until compatible Udon and Browserdriver contracts are pinned.

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

- `internal/elicitor` validates those path-free transaction candidates as
  deterministic `virtual-browser://` sources. It binds candidate and review
  digests, schema, freshness, origins, exact reviewed-flow identity and
  symbolic slot coverage, canonical target, and BAP-provides/BCP-requires
  session dependencies; BRP remains
  dependency- and session-free. `internal/authoringengine` exposes only a
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
  `internal/artifactwriter` revalidates the canonical BRP and independent
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

- `openudon authoring browser-plan` emits exact JSON argv templates, typed dynamic
  argument declarations, and review stages for an external Browsertools
  authoring run. Its private root must already be a restrictive non-symlink
  directory disjoint from the example, and any file output remains inside that
  root. The plan is non-executing and accepts no credential or session input.
  Explicit `browsertools.guided-authoring.v1`
  files supplied through `--browser-profile` are strict-decoded and replayed
  with Browsertools' existing draft/profile/review APIs; only the canonical
  embedded profile can enter an approved package.

- `openudon browser-integration-eval` and `make browser-integration-check`
  run the provider-free A03/P01/A04/A06/E02/E03 release matrix across sibling OpenUdon,
  Browsertools, UWS, Udon, and Browserdriver checkouts. The strict current
  `openudon.browser-integration-eval.v6` JSON report and `.sha256` sidecar live
  under ignored `eval/runs/`, record all five commit/dirty states, fixed named
  evidence and closed diagnostics, and retain no subprocess output. Browsertools
  doctor checks Chromium, Firefox, and WebKit without installation or browser
  launch. Required named gates cover a real Browsertools envelope through
  OpenUdon, Browsertools author-session/result freshness and synthesis,
  UWS 1.8 context, UWS 1.9 scalar and UWS 1.11 typed contracts, Browser
  1.8/1.9 templates, Udon/Browserdriver v10 handoff, and OpenUdon UWS 1.12
  output, plus pending/simulation producer conformance. V1 reports continue to use the unchanged historical scenario lock
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

- Run evidence maintenance commands include `openudon run-evidence verify`,
  `openudon run-evidence archive`, `openudon release-notes draft`, and
  `openudon local-udon-smoke`. `openudon release-evidence` and
  `make release-evidence` wrap these into one local release-evidence workflow
  that writes compact JSON/Markdown summaries under ignored `.openudon-run`
  paths.

- Public CI cross-builds `openudon` and `udon-runner` for Linux, macOS and
  Windows on amd64 and arm64. Tag automation packages those two commands,
  README and Apache-2.0 license into six archives with `SHA256SUMS`; earlier
  published archives retain their original contents. Release browser setup
  reads the actual v5 compatibility and M45 build-input snapshots, staging all
  sixteen distinct browser/executor repositories with fourteen local replacements.
  Private GitHub read headers stay in the clone environment, never Git config.

- Public docs publishing runs `mkdocs build --strict` before GitHub Pages deploy.
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
  build-tagged native capture qualification and enters OpenUdon transitively through
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

- Explicit real-LLM synthesis and neutral expert evaluation default to the `copilot-api` OpenAI-compatible proxy
  at `http://localhost:4141` with model `gpt-5.4-mini`.

- Escalate to a larger model only after the default proxy model fails deterministic checks.

- Shell-level provider/model defaults use `OPENUDON_LLM_PROVIDER` and `OPENUDON_LLM_MODEL`, matching
  the optional expert-evaluation pattern used by sibling OpenW8M. Explicit CLI `--provider` and `--model` flags
  take precedence.

- Provider credentials and proxy endpoints come from environment variables such as
  `COPILOT_API_BASE_URL`, `COPILOT_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, or
  `ANTHROPIC_API_KEY`.

- Provider-specific API keys do not make OpenUdon auto-select that provider; set `OPENUDON_LLM_PROVIDER`
  or pass `--provider` for explicit non-default providers such as Gemini.

- Gemini sends credentials through `x-goog-api-key`, not the URL. All provider
  success/error bodies are limited to 8 MiB, preserve caller deadlines, and
  redact transport errors.

- `openudon authoring draft --api-source KIND:ID=PATH`, `--openapi ID=PATH`,
  `--browser-profile ID=PATH`, `--browser-verification PATH`,
  `--browser-registry PATH|HTTPS_URL`, and
  `--source-root PATH` are repeatable reviewed source inputs. API-capable
  operations take precedence over browser fallback. `--browser-profile`
  accepts capability profiles/bundles, explicit guided-authoring results, and
  local secret-free authentication profiles. Guided results are explicit-file
  only and are reduced to profiles; authentication profiles are never pulled from or published to the
  static capability registry. `--network
  never|ask|allow` defaults to `never` in the closed draft; `ask` is refused and
  `allow` must be explicit. Local discovery is bounded to 10,000
  visited entries, 100 candidates, and 20 MiB per file by default; remote lookup is one approved
  curated-catalog plus APIs.guru pass with an eight-second deadline and at most
  three candidates. Configured HTTPS browser registries use a separate approval
  decision and the same deadline/result/file-size bounds; local static
  registries remain offline.

- `openudon authoring draft --agent --json` is the noninteractive local-agent surface. It never prompts or writes
  deliverables; it reports the full frontier, workflow candidates, source evidence/blockers, and
  proposed file actions, including proposal approval for complete state. `openudon authoring lint --json`,
  `openudon authoring scorecard`, and `openudon authoring repair --json` emit
  versioned JSON reports for reliability scoring and bounded remediation. `openudon authoring variants
  validate` checks authoring variant metadata and reference-seeded clear slots without running the
  scorecard. `openudon authoring variants coverage` requires every provider family represented in
  authoring variants to have positive, missing-detail, and unsafe-negative coverage. `openudon authoring scorecard
  --include-variants` also reads `reference/authoring-variants.json` and summarizes provider-free
  reference/variant package coverage by provider family, variant class, failure family, and top
  readiness issue; `needs_input` variants must declare expected top issue code and slot, and the
  scorecard enforces exact matches. Scorecard reports also carry run ID, commit, command,
  prompt/readiness provenance, explicit missing-detail or unsafe false-pass counters, a
  `needs_input` diagnostic-gap counter, SHA-256 digest sidecar, and retention/share-safety metadata
  marking scorecards as provider-free `release_evidence` that is safe to archive. `openudon authoring
  authoring-eval` is the optional real-LLM authoring evidence lane; it records provider/model, run
  ID, command, commit, prompt/readiness versions, LLM call count, generated paths, drift counts,
  per-variant pass/fail, and structured failure categories, scans generated
  project/intent/transcript/report output for credential-like literal values, writes a SHA-256
  digest sidecar, and marks reports as local ephemeral provider-output evidence requiring redaction
  review before sharing. It is intentionally not part of provider-free release gates. `openudon authoring
  report verify --file ...` revalidates scorecard or authoring-eval JSON, summary counters, variant
  top-issue expectations, failure categories, retention/share-safety metadata, and adjacent SHA-256
  digest sidecars after generation or archival; `make authoring-scorecard` and `make
  release-saas-check` run it for the provider-free scorecard, while authoring-eval verification
  remains optional/manual real-LLM evidence. `make release-saas-check` also runs the variant
  coverage gate before producing scorecard evidence.

- `openudon authoring replay-eval` defaults its own `--prompt-mode` to `fast` and reports actual transcript
  prompt counts, auto-accepted defaults, LLM calls, repair attempts, rejected repairs, and unresolved
  final review warnings.

- Optional expert model evaluation runs an advisory pre-final flow review before showing the current
  draft. The review is structured JSON and only reports cross-step data-flow warnings that
  deterministic request/schema checks may miss. By default it does not mutate the draft; advisory
  review warnings are also written as non-executable comments in the generated `intent.hcl`.

- Neutral authoring retains `openudon.icot-session.v2`, `openudon.icot-transcript.v2`, and v2 OpenUdon
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
  and an empty set is anonymous. Neutral authoring persists one canonical SHA-256 fingerprint
  before request mappings, exposes only the selected set's fields, and treats
  prompt-budget loss of this structure as a deferable readiness blocker.

- Explicit expert evaluation review repair is an experimental opt-in mode. It can make at most two bounded
  pre-final review repairs, and only to request mappings, output sources, and `depends_on`; source,
  operation, credential, and side-effect-scope mutations are rejected and recorded in the transcript.
  M65 adds one narrowly proven exception for deterministic API prework: a read-only local GET/HEAD
  operation may be inserted when it is the only metadata-listed producer for a missing request field
  and requires no inputs. Response-field repair can use local OpenAPI `$ref`, nested object, array,
  and Swagger `schema` response metadata.

- Historical-format neutral sessions use one unified interview evidence ledger for observed facts, user decisions,
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

- Retained iCoT format documents describe historical v2 session/transcript and
  interview behavior. Current seeded draft and expert command migration is in
  docs/authoring-retirement.md; Kinet owns interactive conversation persistence.
  The historical documents grant no removed transport or execution authority.

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

- Keep `cmd/openudon` thin.
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

## Native current and historical qualification

`make browser-system-current-check` and `make qualify` explicitly select the
current UWS1.12/M45 closure. Current native v6 retains thirteen stages and three
fresh passes, replacing retired UI/control checks with actual public capture,
pure registration definitions and both public package journeys. Default checks
remain browser-free. Chromium sandboxing and exact external installed modules
are required; nothing installs or upgrades dependencies. Historical native
v1–v5 reports remain read-only verifiable under their frozen inventories/locks.
Current integration v7 retains expert/authoring gates and tests the neutral
capture/draft dependency boundary; integration v1–v6 selectors stay frozen.
Failure streams are private bounded diagnostic sidecars. Missing prerequisites,
process teardown failure and unknown native outcomes cannot become passing proof.

Attestation v1/v2/v3, per-attempt claims and executor authority are unchanged.
The retained M79/M80/M81 histories document former application-control evidence;
they are not callable current transports. See the [neutral authoring boundary](architecture.md#neutral-authoring-boundary-m95).

## E13 development commands

`make fast` uses Go test caching and documentation checks, without browsers or a
private executor checkout. `make smoke` defaults to `registration_capture_handoff`;
it invokes the existing native adapter directly, allows dirty OpenUdon sources
and binds before/after source, tool, runtime and current sibling identities.
`browser-system-dev --mode smoke --stage registration_capture_handoff
--udon-repo ABS --browserdriver-node-modules ABS --out /tmp/NEW.json
--cache ABS` emits `openudon.browser-development.v2`, never runtime qualification.
Only explicit `--reuse` admits a matching success younger than24hours, retaining
its original execution identity/time. No automatic native qualification fallback.
`make qualify` selects a fresh complete current gate; three repetitions never
consume development cache. Private timing remains `openudon.browser-check-timing.v1`.

`browser-system-input --stack current --repo-root ABS --udon-repo ABS
--browserdriver-node-modules ABS` emits value-free input v3. It binds current
nineteen-source closure, installed dependency bytes/modes, actual namespace/user,
source roots, tool/browser and effective environment identities, but executes
no browser and grants no cache reuse or runtime authority. Historical input v1
remains unchanged. External consumers own their cache policy and fresh acceptance.

## Explicit per-step execution evidence (M90)

M90 selects `openudon.run-evidence.v3` with `--executor-report-version v5`.
The report-v5 schema/fixtures are frozen from accepted Udon M44 source
`1a5e9aa2045e3d875da2e18aab2d6db869ac5223`; no private Go dependency is added.

`make report-v5-qualification` is an explicit real-executor loopback gate;
set `OPENUDON_M44_EXECUTOR` and `OPENUDON_M44_CLOSURE` to the accepted frozen
M44 artifact pair. Default checks skip that real executor. Qualification binds
source, exact closure and binary digests and executes a private binary copy;
see [per-step run evidence](../../docs/per-step-run-evidence.md).

## Shared authoring commands and qualification (M91/M95)

Authoring's published neutral `engine` remains pinned at
`18056cb6b0c1007dd567a4a825a6b4311a357185`. No sibling retirement occurs here.
Single implementations live in `artifactwriter`, `elicitor`, `stepauthoring`,
`browserauthoring`, `browserauthor`, `authoringengine`, `authoringcli` and
`registrationdraft`; synthetic native callers live in `capturequalification`.
OpenUdon has no `cmd/icot`, `internal/icot`, `authoringui` or embedded UI assets.

Use `openudon authoring draft|browser-plan|registration-draft` for closed neutral
adapters and `lint|reconcile|repair|report|variants|scorecard|replay-eval|authoring-eval`
for retained expert/evaluation commands. Existing report schema labels and
`.icot` artifact layouts stay compatible. Explicit model evaluation remains
outside offline checks. M91's published/qualified evidence is preserved in its
retired record; it does not qualify M95's changed runtime. M95 owner qualification/removal passed review3 at application
`c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0`; exact final M20/W29 adoption
is still required. Native/delta reports retain their actual sources.

## M92 applied UWS dependency and executor compatibility

M92.1 pins published UWS source `a7688f54c68f5a75c7cc95aa2b31cea98b31af41`
as `v0.0.0-20260927134327-a7688f54c68f`; Browsertools stays at its
existing `3abe70efc03d9ccb97b8b30e5e86328f60a70c64` module. New authoring
packages default to UWS 1.12.0. Rebuilding reads existing HCL/export versions
before artifact writes, preserves the declared version, and refuses conflicting,
unsafe or unsupported declarations. Historical scenario qualification selects
its manifest version explicitly and cannot override an existing document.

`make report-v5-m45-qualification` is an opt-in loopback-only gate selecting the
accepted M45 executor/closure via `OPENUDON_M45_EXECUTOR` and
`OPENUDON_M45_CLOSURE`. Its test verifies both digests, source identity, the
clean fourteen-source closure and all four passed qualification commands before
executing a private binary copy. It covers eight report-v5 cases at each of
UWS 1.11 and 1.12. The M44 gate retains its own frozen binding. Udon remains an
external CLI, never an imported dependency.

M91's browser integration v5/current scenario v4 locks remain immutable rollback
contexts. They do not qualify M92's newer pin. M92 publishes separate v5
locks/build inputs and current report selectors, with its own source-bound
qualification evidence.

M92.2 adds `openudon step pending --example DIR --request FILE|-`, with
`openudon.step-pending.v1` request/result envelopes. See `docs/step-pending.md`
and its schema, which references unchanged step-authoring v1 definitions;
offline validators load both resources. Confirmed bind effects are stored in
intent HCL and exported as public operation effects for 1.12 packages. Old v1
fixture bytes remain frozen; current bind tests independently verify the new
intent digest and effect, plus the exact older digest with just that additive
annotation absent. `step bind` resolves a pending contract using the current
intent revision, exact contract and its existing source/auth/mapping checks.

M92.3 adds `openudon simulate --example DIR [--input FILE] [--fixtures FILE]
[--allow-generated-fallback]`, with `openudon.simulate-input.v1` input and
`openudon.simulate.v1` results (see `docs/simulation.md`). It uses published UWS
1.12 `mockruntime`, public fixture/JCS matching and deterministic schema synthesis;
the exact transitive dependency is `github.com/gowebpki/jcs v1.0.2`. No private
executor import or schema/expression engine is introduced. Numeric user inputs
retain JSON number identity for fixture matching. M92 records passed producer
conformance and source-bound runtime qualification; consumer adoption is separate.

## M92.4 producer and runtime contracts

Published additive schemas: `openudon.step-pending.v1`,
`openudon.simulate-input.v1` and `openudon.simulate.v1`, with versioned
examples under `docs/examples/step-pending/v1` and `docs/examples/simulation/v1`.
Conformance loads local schema resources only. Current browser compatibility
and build-input files are versioned `*-v5.json`; native v5 and integration v6
qualify the UWS 1.12/M45 context and preserve all earlier report contexts.
The temporary M92 Xvfb permission is synthetic loopback qualification only.

## M93 private development desktop

The approved M93.0 operation prepares the existing development host with
Xvfb 2:21.1.22-1ubuntu1, xauth 1:1.1.2-1.1build1, Openbox 3.6.1-12ubuntu3
and x11vnc 0.9.17-2. Temporary X authentication/private password files are
owner-only; X listens on no TCP port and VNC listens on 127.0.0.1:5901 only.
Use an SSH tunnel and noVNC in a local browser; clipboard exchange and VNC
remote command/control are disabled. A bounded user process, not a permanent
system service, owns teardown. Exact session bindings/expiry and the required
human connection checkpoint live in status-M93.md. Desktop setup grants no
capture target, model disclosure, credentials, registration or execution
permission and proves no M93.5 journey. Restoring an expired session requires
its currently selected authorized operation; never replay closed task rows.

The browser transport extension was approved on 2026-10-01: use the full
Ubuntu noVNC application and websockify on the same named host, not Kinet's
user-installed root npm core library. Installation/verification is owned by
the still-selected M93.0 operation. The old session expired without a human
connection. Installation completed: noVNC 1:1.6.0-2, websockify and
python3-websockify 0.13.0+dfsg1-2ubuntu1; `/usr/share/novnc/vnc.html` is
served by a temporary loopback relay at 127.0.0.1:6080. Exact fresh session
bindings/expiry and automated readiness/teardown evidence live in status-M93.md. Both listeners must remain loopback-only and SSH-forwarded, with
password-required VNC, clipboard disabled and bounded automatic teardown.

## M93.1 capture protocol foundation

`docs/schemas/openudon.browser-capture.v1.schema.json` is the exact embedded
resource in `docs/schemas.BrowserCaptureResources`; runtime decoders use the
already-pinned JSON Schema v6 dependency and shared strict evidencefile reader.
No new dependency, worker implementation or schema engine was added. The
NDJSON message ceiling is 256 KiB, nesting 64 levels, session count 4096
events and absolute deadline at most two hours. Native Browsertools reduced
records keep their own field semantics; a registration event transmits the
current observation/latest preview rather than the entire growing history.
Published examples and `go test ./internal/browsercapture` cover both mode
envelopes, TOTP checkpoint metadata, reviewed proposals and exact single-use
approval/refusal. See `docs/browser-capture-protocol.md`; later M93 tasks
expose the command and run browser/transaction qualification.

## M93.2 authenticated supervising transport

`internal/browsercapture.RunAuthenticated` wraps the existing controller;
`browserauthor.NormalizeConfig` and `ValidateResponse` are neutral pure entry
points over its existing configuration/decision checks. An internal generic
stream driver serializes both mode adapters without a second browser engine.
Input/output must be closeable pipes or local sockets whose closure interrupts
I/O. Absolute context expiry closes them; worker and reader teardown join before
return. Invalid input is fail-closed and never echoed. Private completion data
has no JSON representation. No dependency, public CLI or profile import is added
by this row. Race/pipe tests include TOTP and credential acknowledgments,
separate worker origin denial, exact observation disclosure, replay, changed
state, EOF, expiry, blocked output and late teardown failure.

## M93.3 registration supervising transport

`RunRegistration` uses the explicitly reviewed existing native protocol (v1–v4);
v4 is the start default and retains native verification support.
its reviewed GET/HEAD/no-submit start and existing verification/preview/profile
validators. Legacy registration transports keep their original version defaults.
`RegistrationSession.ValidateDecision` is a pure check against its own current
snapshot; `NormalizeRegistrationConfig` and `NormalizeRegistrationStart` reuse
existing authority/bounds/URL helpers before allocating a worker. No dependency
or public command is added. Native terminal outcomes remain separately readable
after joined worker/reader/private cleanup, including a dropped late failure.
Authenticated blocked-script/diagnostic-file options stay mode-specific;
registration uses its existing closed diagnostics and no-submit traffic policy.
Tests cover verification refusal, preview definitions, symbolic canonical
profile validation, exact current generations, retained outcomes, private
cleanup ordering and value-free bounded wire. Public CLI/worker embedding,
independent package import and native/visible qualification were accepted in
M93; its complete evidence is in the package-local history record. M95 keeps
these native contracts while separately qualifying its transport removal.


## M93.4 command, reviewed start and profile admission

`openudon browser-capture --start FILE --approve-start-sha256 SHA --example DIR
--private-root DIR [--driver-dir DIR]` uses the exact embedded
`openudon.browser-capture-start.v1` schema (closed union, same 256 KiB limit).
Start examples and command/import details live in docs/browser-capture-protocol.md.
The main CLI's hidden worker reuses browserauthoring.RunWorker; no dependency
or browser engine was added. Credentials/model settings have no start fields.
Native configuration validates disjoint prospective package/private roots,
exact goal/role/context/origin policy and declared finite timeouts.

Import approval uses the existing event/command schema, after joined completion.
Native candidate conversion/virtual discovery and the shared authoring workspace
fingerprint stay owner implementations. The retained atomic artifact writer
commits profiles/reviews plus expected/browser-capture/<transaction-id>.json;
receipt metadata binds the reviewed start/transaction and exact file digests.
It grants no package promotion/executor authority. New profile/receipt targets
are create-only; the retained auth review append uses its exact prior digest.
Native expiry/drift are rechecked immediately before replace. Uncertain writes
or terminal delivery never authorize retries. M93.5 qualification is accepted
and preserved in [M93 history](../docs/history/status-M93.md); changed M95
runtime bytes require their own native qualification.

The confirmed initial noVNC session expired cleanly at 07:58:47 UTC on
2026-10-01, with owned processes gone and private auth removed. Human desktop
confirmation is recorded independently. Restore a bounded display only for the
currently selected, authorized later-reuse row, never by replaying closed M93.0.

## M94.1 published APItools and explicit discovery

Current APItools pin: `v0.0.0-20260930205753-fb132631c982`, exact source
`fb132631c9827eae5f2ec4503d03f21eabfb4113`; module sum
`h1:ELxWOW2xW+3ahSrArRi78JD8kVKDFI7TduZrpBBWgwE=`. No replacement or sibling
checkout is used. M79 remains a rollback/historical reference, not the current
pin. `openudon step discover --request FILE|- [--catalog-root DIR]
[--catalog-registry REL] [--catalog-index REL] [--catalog-metadata FILE]
[--enable-remote]` uses the native APItools request/report; stdout is one
bounded JSON report. Explicit operator indexing remains `apitools catalog
index`, not an implicit discovery side effect. See docs/catalog-discovery.md
for outcomes, bounds, exit codes and dual remote opt-in. No new module was
added; APItools sqlitecache remains the source owner's read-only adapter.

Catalog provisioning uses `step source add --catalog --example DIR --request
FILE|-` with the same explicit catalog configuration. Its additive closed
request schema is `docs/schemas/openudon.step-source-catalog.v1.schema.json`,
embedded/validated with the existing JSON-schema dependency and disabled remote
schema loading. APItools ExportCatalogArtifacts, BuildOperationCandidates and
the existing OpenUdon source writer own materialization, selector checks and
atomic publication respectively. Bounds remain 256KiB request/result, 8MiB
source and 32MiB combined selected source/overlay bytes; additional provenance
is at most1MiB, overlays at most64 ×2MiB. Source/manifest paths and legacy
source-add v1 do not change. See docs/catalog-discovery.md for the separate
confirmation, conflict, indeterminate-write and no-automatic-replay contract.

## Received consumer TOTP/qualification guidance

The [consumer handoff](../../docs/consumer-totp-qualification-handoff.md)
records credential-kind/binding separation, exact inventory preparation,
current native input identity and production-parser evidence. E23/E24 code
is already integrated and qualified by the consumer; optional generic UX
and reduced diagnostic improvements remain unpromoted. Receiving these
lessons changes no source/dependency pin or runtime-adoption authority.

## Reviewed capture package CLI (M96)

```
openudon browser-author plan --example DIR --request FILE|-
openudon browser-author apply --example DIR --request FILE|- --expected-plan sha256:HEX --confirmed
```

No new dependency or provider invocation. Version openudon.browser-author.v1:
request256KiB UTF-8, report2MiB; exact native start bytes are base64 strings.
Original receipt byte SHA uses lowercase64hex; request/plan/input/transaction
use tagged SHA256. Package inventory512 files,8MiB each,32MiB total excludes
.git, refuses symlinks/special files/hardlinks/foreign or writable ownership.
Existing native semantics, elicitor/artifactwriter and deterministic build own
materialization. CLI stdout can contain personal previews and stays transient;
fixed errors/metadata must not disclose payloads. Partial-write outcomes and
cleanup flags require inspection. Package promotion remains separately confirmed.

Full make check, affected race, Go vet/format/diff and real main conformance
passed. Frozen Go1.26.6 application source eed683f27d448ca96af90e7bc5987967a6cd0335
at /var/tmp/openudon-m96-qualified-yvoho85o passed native39/3fresh repeats,
integration17/0failed/3unrequested optional plus independent verifiers and all
three capture/package journeys. Later tests-only failure regression is separately
qualified; evidence source/time is never relabeled. Review1 passed; accepted
source publication is independently verified at `d77f6d51262d0f311910070bc4a43f662260cd7e`. See [retired M96](../docs/history/status-M96.md). Public fixtures: docs/fixtures/browser-author-v1.

## Neutral seeded drafts and registration definitions (M95)

See [architecture neutral boundary](architecture.md#neutral-authoring-boundary-m95)
and the [registration-draft contract](../../docs/registration-draft.md). Closed
seeded drafts never read terminal answers; print makes no state writes, explicit
yes gates publication, and missing mandatory input remains a structured frontier.
Historical artifact labels and report readers are preserved; transport removal
never grants capture, package or executor authority.

Current native v6 also retains the four browser-free aggregate stages under
`--stack current --suite offline`: OpenUdon units, Browsertools units,
Browserdriver units and neutral capture lifecycle. `make qualify` runs/verifies
that current offline report before the three fresh loopback passes; old v1–v5
readers keep their original version/lock/suite contracts. No gate is dropped
because the former historical default no longer executes a removed UI.

## M97 broker commands

- openudon broker-inspect --example <package>: exact value-free package and request review metadata.
- openudon approval-template --example <package> --state approved_for_production --reviewer <host> --broker-authority <authority.json>: explicitly bound approval v2.
- openudon run --example <package> --tier production --approval <approval.json> --http-broker-config <absolute-private.json>: config v3, evidence v4 and unchanged report v5.
- udon-runner takes the same private flag across the revalidated external boundary.

Broker execution requires the explicit absolute executor path and exact approved
binary digest. The new path does not use legacy Docker or sibling fallback.
No module pin or private executor import changes. Private transport/config and
executor snapshots stay in the private run staging; no credentials enter portable
artifacts. Exact qualification/source publication and closing review are recorded
in [M97 history](../docs/history/status-M97.md).
