# Browser-Profile Authoring Transactions

OpenUdon uses `openudon.browser-profile-transaction.v1` as the value-free
coordination record for adopting reviewed browser-profile candidates, preparing
a package, and atomically making that package current. It coordinates existing
UWS browser profile families; it is not a UWS document, does not define a new
UWS discriminator, and grants no browser or runtime authority.

The public wire is defined by the
[`openudon.browser-profile-transaction.v1` JSON Schema](schemas/openudon.browser-profile-transaction.v1.schema.json).
OpenUdon keeps its implementation internal, so this JSON wire and schema—not a
Go package—are the compatibility surface.

## Compositions

Each transaction has exactly one of these compositions:

| Transaction kind | Ordered candidates | Session | Authoring network posture |
| --- | --- | --- | --- |
| `authentication_capability` | one BAP, then one BCP | one symbolic name | Human-approved authentication may submit; post-authentication capability observation stays bounded by the authenticated-authoring contract. |
| `registration` | one BRP | forbidden | Browsertools authoring is observation only: GET/HEAD, no submit, no account creation. |

BAP means an existing `uws.browser-authentication.1.0` or 1.1 profile. BCP
means an existing `uws.browser.1.5`, 1.6, or 1.7 capability profile. BRP means
the existing `uws.browser-registration.1.0` profile. `BxP` is only shorthand
for those families.

The BAP and BCP are reviewed as a pair because the authentication flow
establishes the transaction's symbolic session and the protected capability
consumes that same runtime session. The transaction never carries the session
object, cookies, storage, or credentials. A BRP neither establishes nor
consumes a session. Its credential bindings name runtime-owned values but do
not contain them.

Published, value-free examples are available for an
[authentication-capability review](examples/browser-profile-transaction-authentication-capability.json)
and a [registration candidate](examples/browser-profile-transaction-registration.json).
Their `.test` origins and repeated digests are illustrative only.

## Immutable identity and provenance

`id`, `kind`, ordered `candidates`, `provenance`, `credential_bindings`, and
`session` form the immutable transaction identity. Each candidate records its
published profile discriminator plus two digests:

- `source_sha256` binds the exact candidate bytes.
- `review_sha256` binds the independently reviewed, value-free evidence.

Provenance names Browsertools as the producer and binds either the existing
`browsertools.authenticated-authoring.v2` result or the separate no-submit
`browsertools.registration-authoring.v1` result, its digest, observation and
expiry times, and sorted origin set. It does not name the private result path.
A consumer must independently obtain the private result, verify its exact
digest, validate the embedded profiles, and reject stale evidence before
changing `candidate` to `reviewed`.

For a Browsertools registration result, OpenUdon anchors the canonical
mode-`0700` per-run private root before launching the child. The worker emits
only the strict `browsertools.registration-author-session.v1` NDJSON stream;
it never emits a result name, path, digest, browser handle, or session state.
After the protocol reaches `closed`, OpenUdon drains stdout, requires a clean
process-tree exit, and admits exactly one new mode-`0600` digest-named result.
The anchored reader rejects root or file replacement, symlinks, non-regular
files, permissive modes, partial/oversized reads, and any result above the
transaction's 256 KiB adoption ceiling.

OpenUdon then strict-decodes the exact result bytes at the current assessment
time, rebuilds canonical BRP and Browsertools review bytes independently, and
checks the complete result/source/review digests, producer/session versions,
origins, reviewed candidate generation, flow, cleanup policy, and explicit
human review. Symbolic bindings must exactly cover the reviewed credential
slots. The resulting `candidate` transaction retains only those bindings,
digests, times, origins, and published discriminators; the private envelope
and locator stay inside the process boundary.

Origins are sorted, unique serialized origins: HTTPS, or HTTP only for
`localhost` and canonical loopback IPs. They have no user information, path,
query, fragment, empty/default port, Unicode host form, or noncanonical IP
spelling, and each is at most 1,024 characters.

All digests are lowercase `sha256:` strings. Arrays that are sets use canonical
order. OpenUdon limits one encoded transaction to 256 KiB and rejects invalid
UTF-8, duplicate object names, unknown fields, excessive nesting, and trailing
JSON. Fields for a credential value, account identifier, verification response,
request or page content, cookie, storage state, raw worker output, or private
path therefore cannot be added to this record.

## Lifecycle

The transaction is a sequence of immutable snapshots:

```text
candidate -> reviewed -> prepared -> promoted
    |           |           |
    +-----------+-----------+-> cancelled | failed
                            |
                            +-> indeterminate
                                  |-> prepared
                                  |-> promoted
                                  |-> cancelled | failed
```

`candidate` and `reviewed` have no preparation, promotion, or failure object.
The review edge means an operator accepted the exact profile and review
digests; it does not mean that the profiles were packaged or run.

`prepared` adds `preparation.package_sha256` and
`preparation.qualification_sha256`. Preparation writes and validates a complete
candidate package in a restrictive scratch location. It must not modify the
currently selected package. A failed preparation leaves the current generation
unchanged.

`promoted` retains the preparation and adds `promotion.generation_sha256`.
Promotion is one atomic selection operation over the already qualified
generation. It is not browser execution, account activity, approval, release,
or deployment.

`cancelled` is an intentional terminal stop. `failed` carries only a closed
failure class and code:

| Class | Codes |
| --- | --- |
| `rejected` | `transaction_invalid`, `candidate_invalid`, `candidate_stale`, `digest_mismatch`, `review_rejected` |
| `conflict` | `workspace_conflict` |
| `operational` | `preparation_failed`, `qualification_failed`, `promotion_failed` |
| `indeterminate` | `promotion_indeterminate` |

The indeterminate class/code pair is valid only in the `indeterminate` state;
the other classes are valid only in `failed`. Human-readable errors, subprocess
output, and paths belong in private operator diagnostics, not the transaction.

`indeterminate` is reserved for an interrupted or ambiguous promotion. It
requires the prepared digests and the exact
`indeterminate/promotion_indeterminate` outcome. Do not retry promotion or
prepare over it blindly. Reconcile the selected generation and its digest:

- if the prepared generation is current, record `promoted` with that exact
  generation digest;
- if the old generation is current and the scratch package is intact, return
  to `prepared` before a newly approved promotion attempt;
- otherwise stop as `failed` or `cancelled` after preserving the last known
  good generation.

Every transition revalidates the complete snapshot and proves that immutable
identity, provenance, review digests, and any already-recorded preparation or
promotion facts did not change.

## Operator sequence

1. Obtain the Browsertools result through its private, owner-readable channel.
   Do not paste its path or contents into a prompt, log, package, report, or
   transaction.
2. Verify the separately retained result digest, freshness, origins, producer
   version, embedded UWS schemas, candidate digests, and independent reviews.
3. Inspect the exact BAP+BCP pair or BRP and explicitly accept or cancel it.
4. Build in prepare-only mode. Qualify the scratch tree with restrictive
   permissions and record only its package and qualification digests.
5. Recheck workspace/current-generation preconditions, then atomically promote
   the complete prepared generation. Never copy individual files into the
   current generation.
6. If promotion returns without a provable result, record `indeterminate` and
   reconcile before any retry.

For a registration candidate, steps 1–3 never authorize Browsertools to submit
a form. The producer must remain GET/HEAD-only and no account may be created
during authoring. The reviewed profile may describe a later submit action and
its exact approval policy, but that is inert source material.

The bundled registration producer uses the same content-addressed mode-`0500`
re-execution cache, minimal environment, bounded deadlines, stdout drain, and
complete process-tree teardown as authenticated authoring. On Linux,
`CHROME_DEVEL_SANDBOX` is merely forwarded as a selector; Browsertools accepts
it only after validating the administrator-owned sandbox helper. OpenUdon
never substitutes `--no-sandbox` or relaxes host policy.

## Trusted-runtime boundary

Transaction review, preparation, and promotion are artifact operations.
Runtime values are resolved only after a separately approved OpenUdon handoff
reaches the trusted Udon/Browserdriver boundary. That boundary revalidates the
package, approval, symbolic credential mappings, and session contract before
invocation.

Registration runtime execution is currently unsupported. OpenUdon may build,
assess, prepare, promote, generate an approval template, and perform a dry-run
for a reviewed registration package, but every non-dry registration attempt
must fail before executor invocation until compatible Udon and Browserdriver
registration contracts are independently published and pinned.

See [Authenticated Browser Authoring](authenticated-browser-authoring.md) for
the live BAP+BCP producer boundary and [Safety And Trusted Execution](safety.md)
for package and executor requirements.
