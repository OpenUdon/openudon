# v35 — Truthful single pre-submission recovery attestation

An operating application may have an independently proven failed launch that
never reached submission approval. Preserve the zero-attempt v1 contract and
add a private v2 attestation for exactly one prior attempt. Require exact
package/profile/operation, delete_separately, a twenty-minute expiry, and links
to prior evidence, separate authority and a consumed single-use claim.

Historical verification, identity binding and persistent claim serialization
remain with the operating application. OpenUdon validates the operator artifact;
it must not infer submission absence from a missing receipt, fabricate history,
issue recovery authority, or change portable UWS/runtime/receipt formats.
