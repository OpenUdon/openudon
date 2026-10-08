# Public trust APIs

Accepted Stage 11 M98 exposes format-neutral trust metadata and explicit-byte
verification from the existing CLI. [Retired M98](../tabilet/docs/history/status-M98.md)
records closing review 3. Qualified published source
`08a3839f357ec40c7e50c8668e4bd7c8d86bb55a` resolves as
`v0.1.1-0.20261007040813-08a3839f357e`.
[Qualification](m98-qualification.md) and [publication](m98-publication.md)
bind exact source, artifacts and ordinary no-replacement consumer proof.

The public packages are:

| Package | Contract |
|---|---|
| `handoff` | Existing review-handoff v1/v2 manifest types, lifecycle metadata, diagnostics, canonical self digest and value validation. |
| `digest` | Exact-byte SHA-256 and the existing SHA-256/Git object spelling checks. |
| `authority` | Broker authority v1, symbolic bindings, canonical policy digest and identity/deadline validation against caller-supplied time. |
| `approval` | Approval v1/v2 and existing scope, package, tier, state and broker-authority checks. |
| `trust` | Bounded inventory/digest verification of caller-supplied required artifact bytes. |
| `wire` | Bounded strict JSON decoding, with lossless-number option and duplicate-key rejection. |
| `udonreport` | Existing execution-report v2–v5 types, validation and conservative v5 observations. |
| `runevidence` | Existing run-evidence v1–v4/async/signature wires and bounded non-browser byte verification. |

These APIs accept values and perform no filesystem, credential, provider,
network or runtime operation. They preserve existing JSON field order,
discriminators, omission rules and canonicalization. An authority validator
checks the supplied record; the trusted host still owns grant custody,
revocation, destination policy and dispatch. Metadata never grants authority.
Approval v1 retains its existing timestamp behavior; broker approval v2 uses
the concrete authority's bounded validity interval.

The `handoff` constructor builds neutral review metadata. It does not build,
assess or simulate an OpenUdon v2 workflow package. Synthesis-coupled v2
construction and orchestration remain private CLI adapters; supported v3
construction belongs to P09. Workers must bound and strictly decode untrusted
inputs, and redact private values in legacy diagnostics before display or
storage. M98 is accepted and published; Phase B public v3 construction remains P09.

`trust.Inspect` requires a context, canonical package scope and manifest path,
and one immutable byte map containing precisely the required manifest inputs.
The host supplies any additional required paths under its format-specific
inventory policy; OpenUdon performs no filesystem discovery or source parsing.
It verifies the v2 manifest self digest, each artifact hash and neutral safety
policy, then derives the existing package-digest-v1 identity. Its result proves
these bytes and metadata only. It does not claim a passing assessment or
executable support. V1 manifests remain read-only metadata through `handoff`.

Inspection allows at most 1,024 files, 8 MiB per file and 64 MiB total. Strict
JSON allows 8 MiB, 100,000 value nodes and 64 nesting levels; it rejects unknown
fields, multiple documents and exact duplicate keys (including escaped-key
aliases). Ordinary struct destinations additionally reject keys that resolve
to the same Go field under exact-name-first Unicode simple folding. Free-form
map/interface data preserves case-distinct keys such as id/ID. Inspection failures
use a fixed value-free error. The generic `wire`
decoder can return detailed errors; callers redact those before display/storage.
The caller owns stable input bytes during inspection, isolation, safe regular
file reads, format-specific inventory, source/shape validation and assessment.
`handoff.DigestFiles` separately hashes bounded sorted artifact identities with
the retained digest-v1 envelope; it does not verify the artifacts themselves.

`runevidence.Verify` takes exact evidence bytes, exactly the referenced report
and async artifact bytes, optional signature bytes and optional trusted public
key PEM. It performs no path reads. Limits are 8 MiB for evidence/each artifact,
1,024 artifacts and 64 MiB total, 1 MiB for a signature and 64 KiB for the trusted
key. Report v5 keeps its tighter 256 KiB bound and 256-step maximum.
The public verifier supports non-browser profiles; a browser record returns
`ErrUnsupportedBrowser` and uses the retained exact CLI verification path.
Browser wire types are shared metadata; no public browser execution API exists.
Broker v4 byte verification additionally applies the exact existing embedded
broker/run-evidence schemas through a closed resource loader. Unresolved
references fail without external reads; no schema or dependency version changes.
Authority digests and symbolic revisions require exact lowercase 64-hex fields;
the general legacy `digest` spelling helpers retain their trim-permissive API.

Hosts can supply independent `Expected` identity and `ExpectedInventory` to
reject a stale or unrelated attempt. Without them, verification checks the
supplied record's intrinsic consistency and integrity. Gates describe evidence,
not current grants, revocation state, source assessment or execution authority.
Historical broker validity checks the recorded interval's shape, not whether
an old grant is currently usable. `SignatureVerified` proves integrity under
the embedded key; `SignerTrusted` additionally binds it to supplied trusted
key bytes. Neither flag establishes authority. Legacy v1 is read-only and
cannot bind an executor report/signature to an exact attempt.

Missing, invalid or mismatched v5 reports retain conservative unknown outcomes;
they never imply successful dispatch or a safe retry. Validated incomplete
reports preserve observed terminal steps and unknown/unstarted remainder.
Public report parsing shares the retained CLI's wire validators. Workflow-based
inventory derivation and browser runtime enforcement stay private. `Verify`
returns fixed errors for invalid evidence; typed validators retain legacy
diagnostic text and require caller redaction. Returned wire records contain
legacy local paths and metadata; consumers choose appropriate reduced views.

Golden tests consume the retained published simulation, broker-handoff and
broker-execution fixtures, including their embedded policy/self digests.
`internal/publicapi` verifies the transitive public dependency closure excludes
OpenUdon internals and private `genelet/*` modules. Existing internal aliases
keep CLI callers on the same wire types and validation implementation.
