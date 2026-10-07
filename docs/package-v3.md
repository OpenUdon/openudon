# Explicit-byte package v3 (P09 in progress)

The public `packagev3` package defines structural records and explicit-byte
`Build`/`Assess` APIs. P09.1–.2 provide construction and review assessment;
Concrete plan/authority verification is implemented in P09.3;
compatibility qualification and publication remain P09.4–.5. A valid record or compatible assessment never grants execution.

The package uses exact approved `workflows/workflow.uws.yaml` bytes,
`expected/data.json`, `expected/operation-shapes.json` and explicit source
artifacts under `sources/<family>/`. `expected/package.json` identifies those
inputs. `expected/assessment.json` reports stable value-free findings;
`expected/review-handoff.json` identifies exact input/manifest/assessment
artifacts and remains `review_required`. Authoring `intent.hcl`, packaged
`workflow.hcl`, private `.icot` and browser package artifacts are excluded.
Derived presentation views remain consumer-owned, outside this package.

The new record discriminators are `openudon.package.v3`,
`openudon.review-handoff.v3` and `openudon.assessment.v3`. Existing handoff
v1/v2, approval v1/v2, authority v1 and report/evidence wires are unchanged.
Public schemas are additive new files under `docs/schemas/`; Go validation
also enforces canonical scope/path, role/inventory and cross-field identities.
Decoders reject unknown/duplicate/case aliases, missing/null fields and trailing
documents under the existing public strict byte/node/depth limits.

Limits are 512 files, 8 MiB each/32 MiB combined, 32 API sources plus one independently verified runtime catalog and 128 stable
findings (with explicit truncation). Source IDs are permanent public metadata;
credential values and hidden model reasoning have no record field. Handoff
credentials are symbolic names only. Library operations perform no filesystem,
network, credential, model/provider or runtime I/O; workers own isolation and
stable byte custody, and Kinet owns private policy/confirmation/publication.

`Manifest.InputDigest` delegates to the unchanged
`handoff.DigestFiles(scope, openudon.handoff-package-digest.v1, files)` envelope
over canonical reviewed YAML/data/shapes/source hashes. Reports have separate
identities, avoiding input/report digest cycles. The final package digest
includes actual manifest/assessment/handoff artifact bytes through that same
algorithm during qualified construction/inspection. Changed package contents
necessarily have new identities and require fresh displayed approval.

Structural validation does not establish that a ShapeTable corresponds to
source bytes. Verify independently reproduces API-source claims and requires
a trusted runtime-owned catalog verifier for runtime functions. Concrete
operation/input/worker constraints are derived before downstream approval. Unknown
metadata stays indeterminate; non-HTTP source families do not expand hosted
execution. Public OpenUdon imports no private Udon/runtime module. Historical
v2 inspection/approval/report reading and the separately pinned browser path
remain through their own qualified interfaces; conversion never carries grants.

`Build(ctx, BuildOptions)` accepts scope, YAML, JSON data and explicit source
ID/kind/path/bytes. It copies bounded inputs before parsing, reproduces shapes
through the exact accepted APItools M82 source, independently verifies them,
and emits deterministic manifest/assessment/handoff bytes. No intent/HCL is
generated. `Assess` rechecks exact input hashes and independently reproduces
supplied shape claims, refusing stale/forged tables instead of trusting flags.

`DecodeWorkflow` admits one guarded JSON-compatible YAML tree (JSON is its
subset), rejecting aliases, anchors, tags, duplicate keys, multiple documents,
non-JSON numeric spellings and limits. It preserves json.Number values in the
raw projection and restores open fields after legacy model decoding. The exact
17 published core UWS schemas from accepted M08 are embedded; tests compare
all bytes to the selected module. Schema loading refuses external resources.
Schema, semantics, executable/pending, entrypoint and strict-portability checks
precede source-bound binding review. Partial schemas/expressions, runtime
profiles, step-supplied values and work/effect bounds remain explicitly unproved.
Output pointers use verified response schemas; open schemas do not prove a
missing field impossible. Stable findings name the workflow artifact without
parser excerpts or private values. Flow observations remain advisory.

Default literal-secret policy is the shared public `credentialpolicy` package;
legacy internal adapters retain the same checks. Build's optional `Private`
predicate applies the consuming host's known-secret policy to each input and
emitted artifact, without loading secrets. Kinet still supplies short/encoded
known-value detection and worker isolation. Credential readiness and current
symbolic revisions are separately checked before execution authority. Historical
browser artifact masking stays private and retains its unchanged adapter.

The eight copied synthetic source fixtures retain their accepted APItools
hashes, native protocols/selectors and 9,829-byte golden shape table. The
fixture package has a pending step, so metadata coverage never becomes
execution permission. Literal mismatch, pending, unsupported expression,
unknown type/output, stale source, forged method, privacy, alias, duplicate,
size, cancellation and snapshot independence are tested offline.

`Verify(ctx, VerifyOptions)` receives trusted scope/digest plus the complete
explicit file map. It copies bounded bytes, requires canonical manifest/handoff,
checks the closed inventory and all links, independently reproduces shapes and
assessment, and retains private VerifiedPackage state. Snapshot/Assessment
accessors copy returned values. CredentialNames is independently derived from
known symbolic security alternatives of selected operations; unknown security
stays unknown and supplies no invented slot. Forged names or changed canonical
records refuse even when an attacker recomputes advertised artifact hashes.

`DeriveExecutionPlan` binds exact package/handoff/input identities, source/native
selectors, every raw operation/step constraint, data identity and selected
WorkerIdentity (binary, module closure and Udon runtime revision). Its supported
concrete profile is one bounded sequence with unique operation invocations and
already preceding dependencies. This retains Kinet's qualified managed-local
and hosted broker profile; structural authoring remains reviewable and generic
runtime support is not automatically new host permission. Unsupported controls,
overrides, browser/non-HTTP leaves, ambiguity and incompatible review outcomes
refuse. MetadataComplete remains explicit; a plan is review metadata, never a
grant. CheckExecutionPlan reproduces the plan from the private snapshot.

Runtime source kind `runtime-function` is limited to one
`udon-runtime-functions` catalog under `sources/runtime-function/`. Its exact
source bytes and table must be independently reproduced by the trusted
RuntimeVerifier supplied by the actual qualified consuming runtime. A producer
string or verification flag is insufficient. Function plans additionally bind
the catalog revision to WorkerIdentity.RuntimeRevision and require the trusted
RuntimeAdmission adapter to the selected runtime's non-effectful
Compile/CheckSupported path over independent supplied byte copies. The public
SDK imports no private runtime and performs no function invocation. Native
catalog type/domain/invocation gaps remain partial; admission is not a claim
that those metadata schemas became complete. Catalog production and independent
verification are value-free, and new hosted functions are not enabled here.

`DeriveBrokerAuthority` accepts a fresh identity/time seed and current symbolic
credential revisions, deriving package/input/executor/ordered operations through
the unchanged broker-authority v1 wire. It requires compatible review metadata,
complete fixed HTTP shapes, one server and one fixed security alternative. The
APItools native inventory independently distinguishes bearer from basic (the
coarse shared security type alone cannot). Only known unscoped bearer and
header/query API keys are admitted, with exact canonical placement and no extra
credential names. Missing/stale revisions, altered policy/constraints, unsupported
auth, OR alternatives, functions, unsafe origins and old populated seeds refuse.
The host still enforces owner/current grant custody, revocation and destinations.

`CheckExecutionApproval` requires the trusted host's separately confirmed exact
PlanSHA256, including the worker/closure, and a current exact-scope/package
approval with explicit expiry. HTTP requires broker approval v2 and independent
concrete-authority verification; pure functions retain approval v1 with native
admission and no credentials. Existing approval wires are unchanged. Approval
v1's package digest alone is insufficient for a changed worker. The SDK never
reads a host grant store or treats package text as confirmation authority.

All default checks remain offline. Tampering/rehashed provenance/security,
missing artifacts, stale sources/inputs/credential revisions, changed workers,
unknown versions, private values, cancellation and zero-value proofs are tested.
The synthetic runtime fixture is independently generated and verified by the
published Udon M48 module; see testdata/runtime/README.md. A preliminary private
adapter uses that ordinary exact runtime module with a temporary local SDK
bootstrap and synthetic worker hashes. It performs no effects and is not final
publication or worker qualification. P09.5 owns the clean ordinary published
consumer/closure proof, while Kinet M47 owns actual worker isolation and pins.
