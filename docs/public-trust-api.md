# Public trust APIs

Stage 11 M98 extracts format-neutral trust metadata from the existing CLI.
Its acceptance and source publication remain tracked in
[M98](../tabilet/memory-bank/status-M98.md). A source checkout is not yet an
accepted or published consumer pin.

The public packages are:

| Package | Contract |
|---|---|
| `handoff` | Existing review-handoff v1/v2 manifest types, lifecycle metadata, diagnostics, canonical self digest and value validation. |
| `digest` | Exact-byte SHA-256 and the existing SHA-256/Git object spelling checks. |
| `authority` | Broker authority v1, symbolic bindings, canonical policy digest and identity/deadline validation against caller-supplied time. |
| `approval` | Approval v1/v2 and existing scope, package, tier, state and broker-authority checks. |

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
storage. M98's remaining rows provide bounded trust and evidence verification.

Golden tests consume the retained published simulation, broker-handoff and
broker-execution fixtures, including their embedded policy/self digests.
`internal/publicapi` verifies the transitive public dependency closure excludes
OpenUdon internals and private `genelet/*` modules. Existing internal aliases
keep CLI callers on the same wire types and validation implementation.
