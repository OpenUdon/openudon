# Brokered executor handoff

M97 adds an explicit contract for one approved HTTP execution over the published
Udon M46 broker transport. OpenUdon validates reviewed packages and passes the
external CLI handoff; the trusted host owns accounts, grants, credentials,
network destination policy, durable dispatch claims and product quotas. No
private executor module is imported.

## Version selection

| Artifact | Broker contract | Preserved legacy contract |
|---|---|---|
| Concrete authority | `openudon.broker-authority.v1` | None |
| Approval | `openudon.approval.v2` | `openudon.approval.v1` |
| Executor configuration | `openudon.executor-run.v3` | v2 execution, v1 read-only |
| Run evidence | `openudon.run-evidence.v4` | v1–v3 readers |
| External transport | `udon.http-broker.v1` | Legacy direct runtime modes |
| Step report | Existing `udon.execution-report.v5` | Earlier reports unchanged |

These definitions are additive. Every broker envelope requires its concrete
`broker` object and version; attaching that object to an old version refuses.
The old default serialization omits the new field. M97.1 defines and tests
the contracts; M97.2 wires execution. Definition or package import alone
does not establish runtime adoption or side-effect authority.

## Concrete approval

The authority binds an owner, agent, grant revision and occurrence to one
lowercase-hex run ID. It binds the exact package, handoff, concrete input digest,
executor bytes, approval lifetime, ordered step/operation/invocation inventory,
reviewed methods/origins and request-constraint digests. Each credential binding
names only its scheme, location and current revision digest. API-key header/query
and bearer header bindings match the published Udon contract; OAuth, cookies
and signing are outside this profile.

`policy_sha256` hashes the standard Go `encoding/json` bytes of the declared
Authority struct with that field empty. Field order and binding order are part
of v1; the hashed [fixture manifest](fixtures/broker-handoff-v1/manifest.json)
and semantic tests detect drift. Changing a package, input, executor,
occurrence, operation or credential revision changes the policy digest. Hashes
describe approval; the host must still check current authority before each I/O.
An approved recurring grant supplies no reusable concrete-run approval: its
trusted host creates and consumes a new occurrence-bound approval every time.

The authority requires an ordered, bounded, unique straight-line OpenAPI HTTP
inventory of at most 100 operations. Unsupported versions and shapes refuse.
An origin identifies scheme and authority only, with no userinfo, path, query
or fragment; the host independently enforces its stronger destination policy.
Sandbox still uses existing non-production address checks. A production state
does not relax any legacy approval or package gate.

### Exact package review

`openudon broker-inspect --example <package>` returns
`openudon.broker-package.v1`: existing package/handoff/credential review metadata
plus `plan.inputs_sha256`, ordered operations and ordered request constraints.
It stages approved bytes privately for inspection, removes that staging, and
changes neither the package nor any approval. No executor or service is called.
`inputs_sha256` hashes the exact compiled workflow bytes; the first broker
profile accepts no additional runtime data file. Static inputs and dynamic
expressions therefore remain part of the reviewed package. Each operation's
`constraints_sha256` hashes standard Go JSON for
`{"operation": <UWS operation>, "step": <UWS step>}`. Inspection returns those
canonical bytes alongside the source operation ID, API server and path template
so the host can apply explicit static/dynamic request policy without copying an
OpenAPI parser or private executor compiler.

Review uses the existing APItools metadata adapter. This first profile requires
one fixed OpenAPI server, operation IDs and a fixed API-key/bearer security set;
server variables/overrides, OAuth/signing, ambiguous security alternatives and
custom credential-binding overrides are unsupported. Inspection's symbolic
bindings have no credential revision yet; the host supplies the exact current
revision when constructing authority. Empty bindings are omitted.

`approval-template --broker-authority <authority.json>` emits approval v2 only
after checking current authority and exact package/input/operation constraints.
The ordinary template remains approval v1. Neither template resolves an account
credential or validates current host grants; those checks belong to the host.

## Private transport input

The published Udon executor receives a separate owner-only, bounded private
broker configuration, containing a run-scoped Unix socket and ephemeral RPC
capability. The capability is not an API credential or a portable artifact.
Only the declared symbolic binding inventory bypasses legacy environment-value
requirements. API credential values remain at the host, outside executor
environment, approved artifacts, normal diagnostics and run evidence.

`openudon run --http-broker-config /absolute/private.json` requires approval v2;
the external `udon-runner` accepts the same explicit private reference with
executor config v3. Broker mode selects report v5 automatically and requires an
explicit absolute `OPENUDON_EXECUTOR` or `OPENUDON_UDON_BIN` whose bytes match the
approved executor digest. It has no sibling-checkout or legacy Docker fallback.
Checked executor and transport bytes are copied privately into staging before
Udon starts. Credential values and proxy variables are excluded from both
process environments. The capability and socket/config values never enter the
portable configuration; evidence redacts the private config argument.

Broker configs/evidence use create-only publication. The executor path also
persists a create-only run claim before invocation; interruption never removes
it. Reusing that run/workdir refuses before another invocation and preserves
earlier evidence. Dry-run invokes nothing and claims no I/O; use a separate
output root from real execution because its immutable evidence already occupies
its run directory. The host must independently enforce durable claims across
workers/workdirs, current credentials/grants and all concrete destinations.

The v4 evidence records the value-free concrete authority and existing per-step
observation. Missing, malformed, stale or lost completion cannot prove a failed
write or successful execution; the unchanged report-v5 reader preserves unknown
outcomes. No broker retry, redirect or direct-network fallback is permitted.

## Schemas and upstream provenance

The published schemas are [authority v1](schemas/openudon.broker-authority.v1.schema.json),
[approval v2](schemas/openudon.approval.v2.schema.json),
[configuration v3](schemas/openudon.executor-run.v3.schema.json) and
[evidence v4](schemas/openudon.run-evidence.v4.schema.json). They are embedded
once in `BrokerHandoffResources`; Go validators additionally check digests,
inventory uniqueness, temporal posture and package/runtime correspondence.
Positive Go envelopes validate against the schemas; downgrade fixtures refuse.

Udon M46 accepted source is `95c5850fd446e06ac6f79d943774db67e417c989`, with
completed record published at `71071537890599e98541abe8ca564490660dfd16`.
Executor SHA-256: `53bb9e8976f67c6a5880f07248a99cd195e6285ce7a5a68239793dd5dc6eb429`.
Fourteen-source closure SHA-256: `a286f21a5dcd4180b7ec19bf0630188ddacec26e0532198db14decd0c86b2069`.
Udon fixture manifest SHA-256: `7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f`.
Use those exact clean exports and executor bytes for M97 qualification.
Existing authoring/browser runtime acceptance remains frozen at
`c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0`; this new profile needs its own
fresh owner qualification and Kinet adoption.

M97.2's focused tests cover private handoff, legacy downgrade refusal, stale
package/handoff/input/method/origin/constraints/executor/bindings, expiry, public
socket/config refusal and conservative interruption/replay/archive behavior.
Full owner qualification, exact-source artifact provenance and publication
remain M97.3–M97.4; no new runtime acceptance is claimed yet.
