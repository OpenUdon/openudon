# v35 Result — Explicit pre-submission recovery facts

The additive private attestation v2 records one prior attempt, delete_separately,
a twenty-minute expiry, and exact authority/claim/prior-attestation/native/
executor-report links. Only submission_not_started is representable. Existing
v1 remains closed and requires zero prior attempts. The current package,
profile, operation and flow are still independently bound at trusted handoff.

The operating application, not this artifact decoder, must verify historical
submission absence and durably consume a persistent one-use claim. The native
artifact carries those reviewed operator facts without credentials, identifiers,
private paths or browser state. No automatic recovery is inferred from failure,
and UWS, run-config and receipt formats remain unchanged.

Full offline tests, both native handoff variants, make check, doc-memory, UWS
validation and vet pass. Focused review iteration 1 passes. Complete consumer
runtime qualification remains separately pending in M82.2 before live adoption.
