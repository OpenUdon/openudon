# Explicit-byte package v3 (P09 in progress)

The public `packagev3` package defines structural records and explicit-byte
`Build`/`Assess` APIs. P09.1–.2 provide construction and review assessment;
concrete authority verification, compatibility qualification and publication
remain P09.3–.5. A valid record or compatible assessment never grants execution.

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

Limits are 512 files, 8 MiB each/32 MiB combined, 32 API sources and 128 stable
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
source bytes. P09.3 must independently reproduce source claims and derive
concrete operation/input/worker authority before downstream approval. Unknown
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
