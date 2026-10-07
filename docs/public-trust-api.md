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
| `trust` | Bounded inventory/digest verification of caller-supplied required artifact bytes. |
| `wire` | Bounded strict JSON decoding, with lossless-number option and duplicate-key rejection. |

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
storage. Evidence verification and whole qualification remain pending.

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
fields, multiple documents and duplicate keys (including casing/escaped-key
aliases). Inspection failures use a fixed value-free error. The generic `wire`
decoder can return detailed errors; callers redact those before display/storage.
The caller owns stable input bytes during inspection, isolation, safe regular
file reads, format-specific inventory, source/shape validation and assessment.
`handoff.DigestFiles` separately hashes bounded sorted artifact identities with
the retained digest-v1 envelope; it does not verify the artifacts themselves.

Golden tests consume the retained published simulation, broker-handoff and
broker-execution fixtures, including their embedded policy/self digests.
`internal/publicapi` verifies the transitive public dependency closure excludes
OpenUdon internals and private `genelet/*` modules. Existing internal aliases
keep CLI callers on the same wire types and validation implementation.
