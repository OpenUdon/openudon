# Browser-Profile Authoring Transactions

OpenUdon uses `openudon.browser-profile-transaction.v1` as the immutable
value-free coordination record for unchanged BAP and legacy BRP authoring.
The additive `openudon.browser-profile-transaction.v2` is restricted to a BRP
whose provenance is exactly `browsertools.registration-authoring.v2`; it exists
so an explicitly reviewed structural query can remain in the canonical BRP.
Both versions coordinate existing UWS browser profile families; neither is a
UWS document, defines a new UWS discriminator, or grants browser/runtime
authority.

The public wires are defined by the
[`openudon.browser-profile-transaction.v1` JSON Schema](schemas/openudon.browser-profile-transaction.v1.schema.json)
and the registration-only
[`openudon.browser-profile-transaction.v2` JSON Schema](schemas/openudon.browser-profile-transaction.v2.schema.json).
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

## Qualification evidence

Cross-package qualification produces one canonical
`openudon.browser-transaction-qualification.v2` JSON report plus an exact
SHA-256 sidecar. The report binds exact OpenUdon, Browsertools, Browserdriver,
Udon, and UWS commits and their publication classifications. UWS must remain at
the unchanged published lock; each implementation repository may be either at
its published lock or at a clean local `main` descendant while publication is
pending. The report carries nine BAP+BCP lifecycle digests and eleven BRP
authoring, package, attestation, workflow, and execution-report digests. Its
posture proves GET/HEAD-only registration authoring followed by one separately
attested and approved loopback POST, a fixed registration result, account
creation only in the disposable fixture, executor invocation, and no named
registration session. Its closed schema has no path, free-form diagnostic,
subprocess-output, page/request-content, account identity, credential, cookie,
storage, or session-material field.

Run the complete qualification from a clean OpenUdon checkout with the exact
locked sibling repositories:

```bash
xvfb-run -a make browser-transaction-qualification
```

The target first runs the adversarial contract, lifecycle, rollback,
concurrency, frontend-conflict, and sensitive-artifact matrix. It then runs
one real Browsertools BAP+BCP transaction through package promotion and
Udon/Browserdriver replay. The BRP case drives the authenticated iCoT wizard,
Browsertools result v2, transaction v2, package prepare/qualify/promote,
selected-package inspection, the private digest-bound attestation, Udon report
v3, and Browserdriver protocol v4 through the fixed loopback result. Only
embedded loopback fixtures are reachable. All five repositories must be clean
on `main`, match their compatibility lock where applicable, and either equal
or descend from an independently resolved `origin/main`; the report records
which exact revisions are already published.

Verify a retained report independently with:

```bash
openudon browser-transaction-eval --verify eval/runs/browser-transaction-qualification-local/report.json
```

Verification rejects missing or extra fields, duplicate names, noncanonical
JSON, unsupported failure codes, dependency-lock drift, digest tampering,
symlinks, and oversized input. A passing report authorizes neither publication
nor registration against a non-loopback target.

`xvfb-run` supplies a display on headless Linux; it does not disable or supply
the Chromium sandbox. Chromium must be able to use the host's unprivileged
user-namespace sandbox. On a host where policy disables that mechanism, an
administrator may install Chromium's setuid helper and export its absolute
path as `CHROME_DEVEL_SANDBOX`. Browsertools accepts the helper only when it is
a root-owned, single-link, mode-`4755` regular file on a setuid filesystem,
its resolved path is unchanged, and every ancestor is root-owned and not
writable except for a root-owned sticky directory. Browsertools never passes a
sandbox-disable flag. A qualification error naming the sandbox prerequisite is
distinct from a scenario-level `authoring_failed` result.

## Immutable identity and provenance

`id`, `kind`, ordered `candidates`, `provenance`, `credential_bindings`, and
`session` form the immutable transaction identity. Each candidate records its
published profile discriminator plus two digests:

- `source_sha256` binds the exact candidate bytes.
- `review_sha256` binds the independently reviewed, value-free evidence.

Provenance names Browsertools as the producer and binds either the existing
`browsertools.authenticated-authoring.v2` result, the legacy no-submit
`browsertools.registration-authoring.v1` result, or—only in transaction v2—the
query-capable `browsertools.registration-authoring.v2` result, its digest,
observation and expiry times, and sorted origin set. It does not name the
private result path or any retained query.
A consumer must independently obtain the private result, verify its exact
digest, validate the embedded profiles, and reject stale evidence before
changing `candidate` to `reviewed`.

For a Browsertools registration result, OpenUdon anchors the canonical
mode-`0700` per-run private root before launching the child. The worker emits
only the deliberately selected strict
`browsertools.registration-author-session.v1` or v2 NDJSON stream;
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

## Virtual candidate discovery

After private adoption, the iCoT engine represents each exact source as an
in-memory virtual candidate. A public candidate summary contains only the
transaction/source/review digests, published schema, canonical future target,
symbolic bindings, and kind-specific dependency metadata. It contains no
private path, result envelope, review body, or profile bytes. The engine uses a
monotonic catalog generation, and selection must name that exact generation;
replacement or a stale selection fails closed.

The BAP candidate provides the transaction's symbolic session. Its BCP
candidate requires that same session and depends on the exact BAP candidate,
so selecting the BCP traverses and selects the dependency pair. A BRP has no
session provider, session requirement, or dependency. In every composition,
source and review digests, schemas, origins, expiry, and complete symbolic
credential slots for the exact reviewed BAP or BRP flow are rechecked against
the transaction before discovery. The flow identity is digest-bound into the
virtual plan provenance so changing it invalidates an existing selection.

Virtual candidates use deterministic `virtual-browser://` identities and
canonical targets below `browser-authentication/`, `browser-profiles/`, or
`browser-registration/`. The URI is an engine identity, not a private
filesystem location. API-family documents retain selection priority over all
browser candidates, and virtual targets may not shadow an ordinary local or
registry source. Selection persists only value-free resumable metadata; exact
canonical profile bytes remain process-local and are copied to a package only
by the ordinary explicit approval path.

For an authenticated-authoring v2 result, OpenUdon independently parses both
canonical profiles and both canonical reviews before composing the transaction.
The capability must require login state, the authentication profile must expose
one exact selected flow, shared popup/frame identifiers must have identical
definitions, and the union of profile and context origins must exactly equal
the transaction provenance. Symbolic bindings exactly cover the selected
flow's credential and challenge slots. The transaction session is a portable
symbol derived from the profile identity; it names the execution-local session
provided by the BAP and consumed by the BCP, but is never a browser handle.

Explicit acceptance changes only `candidate` to `reviewed` through the normal
immutable transition validator. Virtual lowering remains deterministic: BCP
selection closes over BAP, package inventory contains both canonical target
files and both safe source-review metadata files, and authentication approval
evidence is derived from the authored step rather than from transaction review.
Drafts and public snapshots omit source/review bytes and never retain cookies,
storage, credentials, or a runtime session object.

A reviewed BRP exposes only its exact selected flow for authoring. Selecting it
creates a standalone `browser_registration` step with the transaction's exact
symbolic bindings, no browser session, a bounded timeout, and the fixed
`operator_attestation` / `fail` / `stop_without_retry` policy. The reviewed
cleanup disposition is either `delete_separately` or
`retain_dedicated_test_identity`; cleanup is never part of the call. The
profile's ordered submit and human-checkpoint descriptions remain inert source
material. A separate step-scoped authoring confirmation is required and does
not claim that submission or account creation occurred.

## Private registration attestation

Non-dry registration additionally requires one
`openudon.browser-registration-attestation.v1` JSON artifact. Its path must be
absolute, canonical, outside the OpenUdon repository, and name an owner-readable
owner-only regular file without symlinks. The artifact is consumed locally and
is never copied into the package, executor stage, run evidence, or argv.

The closed artifact binds the exact handoff package and canonical BRP digests,
lowered operation, selected flow, zero prior attempts, dedicated-test posture,
reviewed cleanup disposition, symbolic reviewer, and a whole-second UTC expiry
no more than 24 hours away. Unknown fields—including account, credential, and
verification values—are rejected. This attestation is distinct from both the
ordinary package approval and the submit approval requested immediately before
the registration runtime's sole submit.

On ordinary authoring approval, OpenUdon revalidates both in-memory inputs and
atomically proposes the canonical profile, its adjacent `*.review.json`
Browsertools bundle, and `.icot/browser-registration.json`. The latter binds
the exact source/review digests, flow, bindings, approval, timeout, and fixed
failure policy consumed by package quality and trusted dry-run. Resumable
drafts retain only those value-free identities; after restart the exact private
candidate must be rediscovered before its bytes can be materialized. Exact
rediscovery rehydrates both byte bodies only in memory. Missing, stale, or
identity-changed candidates block resume; deselection or replacement clears
the affected registration, authentication, and mutating-operation authoring
approvals so the repaired proposal must be reviewed again. Cancellation and
partial private results produce no candidate, and workspace drift still blocks
the ordinary atomic write boundary.

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
`preparation.qualification_sha256`. The internal preparation boundary first
reads and rechecks one bounded, manifest-complete generation into defensive
memory and emits only portable paths, digests, passing stored-quality status,
approval-state names, execution policy, and symbolic credential names. It
requires an explicit portable scope and performs no write. Qualification then
materializes those exact bytes beneath a fresh same-filesystem mode-0700
scratch root, requires mode-0700 package directories and mode-0600
single-link files, rejects aliases and unsupported members, and reruns current
quality/secret, package/handoff, and trusted dry-run gates. The scratch tree is
removed on success or failure, and no executor is invoked. Neither phase may
modify the currently selected package; a failed phase leaves it unchanged.

`promoted` retains the preparation and adds `promotion.generation_sha256`.
Promotion is one atomic selection operation over the already qualified
generation. The internal store publishes the complete restrictive generation
under a collision-resistant content identity, synchronizes its files and
directories where supported, and only then atomically replaces a value-free
`current.json` selector. That selector binds the selected generation and its
immediately prior generation; repeated selection of the same generation is
idempotent, concurrent builders fail closed on a create-only store lock, and
readers resolve only complete immutable generations. Promotion never removes
a generation. It is not browser execution, account activity, approval,
release, retention, or deployment.

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
prepare over it blindly. The internal promoter keeps a digest-bound,
value-free intent until selection and cleanup are proven. Recovery first uses
`InspectRecovery` to revalidate current, prior, target, lock, intent, and a
bounded transient inventory; `Reconcile` requires that exact report digest
and rechecks it before removing only staging, temporary selector, intent, and
lock artifacts. It never rewrites `current.json` or deletes a generation.
Typed promoter failures distinguish `rolled_back`, `indeterminate`, and
`recovery_required` outcomes and include only the recoverable target
generation digest. A rolled-back target may remain as an immutable unselected
generation; its presence does not make it current or grant retry authority.
Reconcile the selected generation and its digest:

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

The compatible terminal package boundary keeps the long-standing artifact
`openudon promote` command unchanged. Use the explicit package namespace:

```bash
openudon package prepare \
  --example examples/<name> --scope examples/<name> \
  --scratch /absolute/restrictive-scratch-parent

openudon package promote \
  --example examples/<name> --scope examples/<name> \
  --scratch /absolute/restrictive-scratch-parent \
  --store /absolute/generation-store --confirmed

openudon package inspect --store /absolute/generation-store
openudon package recover --store /absolute/generation-store
openudon package recover --store /absolute/generation-store \
  --accept sha256:EXACT_RECOVERY_REPORT_DIGEST
```

Prepare emits only preparation and qualification evidence and accepts neither
a store nor promotion confirmation. Promote requires both. `inspect` emits the
exact selection digest needed by selected-package approval and handoff. A
recovery call without `--accept` is read-only; reconciliation requires the
exact just-observed digest.

For the unified transaction journey, use the experimental iCoT adapter:

```bash
icot browser-transaction \
  --transaction ./transaction.json \
  --example examples/<name> --scope examples/<name> \
  --scratch /absolute/restrictive-scratch-parent \
  --store /absolute/generation-store --prepare
```

With no lifecycle flag it only emits the started value-free snapshot.
`--review`, `--prepare`, and `--promote` request progressively later
checkpoints; promotion implies the earlier two and a candidate transaction
still asks separately at every applicable checkpoint. `--recover` is valid only with promotion and
accepts an indeterminate recovery report by its just-inspected digest.
`--inspect-selected` is a post-promotion, non-writing package inspection. State
events use value-free NDJSON on stdout; prompts and closed failure
class/code/operation records use stderr. Empty or incorrect input fails without
mutation, and SIGINT/SIGTERM cancel a blocked prompt or engine operation.

The same public transaction can be supplied to `icot ui` with
`--browser-transaction`, `--package-scope`, `--package-scratch`, and
`--package-store`. The UI and terminal derive one identical kind-specific
review resource from the shared engine. Neither command accepts a private
Browsertools result, credential value, browser handle, or runtime action.

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

Registration runtime execution is restricted to one exact registration-only
workflow through Browserdriver protocol v4 and Udon execution report v3.
OpenUdon requires the private digest-bound attestation described above plus a
separate exact `--approve-browser-registration OP_ID`; it then hands Udon both
`--attest-browser-registration OP_ID` and the submit approval. Missing,
expired, drifted, mixed-browser, v2/v3-driver, or report-v2 configurations fail
before executor invocation. BAP execution retains its v2/v3 behavior.

Selected generations use the existing approval and runner contracts by
replacing `--example` with `--package-store /absolute/store --selection
sha256:...`. Selected `run` also requires an explicit approval and work
directory outside the immutable store. Approval bytes, package digests, dry-
run evidence, and executor authority remain unchanged; stale selection
digests fail before handoff.

See [Authenticated Browser Authoring](authenticated-browser-authoring.md) for
the live BAP+BCP producer boundary and [Safety And Trusted Execution](safety.md)
for package and executor requirements.
