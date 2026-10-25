# Public compatibility boundary

Current OpenUdon supports the [public trust APIs](public-trust-api.md),
[package v3 byte API](package-v3.md) and
[browser supplement](browser-package-v3.md). A32 retires the private legacy
CLI/authoring implementation and runner publication; the
[retirement guide](legacy-cli-retirement.md) identifies affected entrypoints.
Existing published revisions and frozen consumer pins remain available under
their original contracts.

Public API declarations, JSON wires, schema bytes and historical fixtures are
preserved. V2/v3 package history inspection remains read-only and establishes no
source proof or execution authority. Historical approvals never authorize a
rebuilt successor. Readers reject unsupported future wire versions.

P11 preserves named-output ownership, deterministic assessment and refusal of
stale proof. P12 retains authored bounded authentication and readable untimed
history. P13 distinguishes saved metadata from actual session eligibility and
requires observed fallback to narrow reuse/save rights. Host/native admission,
current grants, revocation, credentials and runtime custody remain independent.

Public UWS semantics remain governed by UWS. OpenUdon's ability to review supplied
source bytes does not establish executable support in any consuming runtime.
Optional Ed25519 signatures establish configured operator identity only when a
trusted key is independently pinned; embedded-key verification establishes
integrity.

Additional Kinet physical qualification deferred at M53 remains unqualified.
Retirement adds no deployment, account, registration, private-input or runtime
permission and changes no existing consumer pin.
