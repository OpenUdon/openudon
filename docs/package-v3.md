# Explicit-byte package v3 (P09 in progress)

The public `packagev3` package defines structural v3 records. P09.1 supplies
record/schema validation only; construction, assessment, source/shape proof,
authority, compatibility qualification and publication remain P09.2–.5. A
valid record never supplies approval or execution authority.

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
