# Consumer TOTP and qualification handoff

Received 2026-10-01 from a private consumer's completed W27 integration. This
record contains generalized findings, synthetic reproductions, regression
references and dispositions. Target pages, profiles, identifiers, credentials,
traffic and raw logs are excluded. The receiving OpenUdon baseline is
`e12a648`; qualified integration used `dc2c1a970d8e5f47e284d0949f9928c3c35fa1f8`.
Receiving documentation does not replace that frozen qualified selection.

## Keep credential kind separate from its binding name

A TOTP setup seed is the persistent authenticator key; the runtime derives a
six-digit current code from it. Recovery codes are single-use alternatives to
TOTP, not seeds. Captured human entry records the challenge, not secret bytes.
The profile's `totp_seed` slot/kind and the package's operator-selected binding
name are separate concepts. Show the actual credential kind in the dialog
title/instruction and check that the selected entry method works on the
chosen display. A generic Password label can obscure a distinct MFA input.
Never log secret values while diagnosing a prompt or MFA failure.

The Browsertools receiving record (`../browsertools/docs/consumer-totp-capture-handoff.md`;
its receipt revision is recorded in W27 history and is unavailable in this checkout)
documents the existing producer and focused TOTP slot regression. OpenUdon's
current `mfa-totp-scalars` scenario already exercises the generic integration;
[scenario guidance](browser-scenario-eval.md) gives its bounded selector.
The consumer's fresh synthetic cases covered two/three bindings, a fresh TOTP
step and count values 0/1/3; an authorized live adapter count then passed with
MFA independently confirmed and no recovery-code use.

**Disposition:** existing profile/runtime integration is sufficient; no new
schema, OpenUdon credential store or target-specific prompt policy is added.
The consumer owns its reviewed private credential dialogs. More explicit
generic UI terminology is an optional OpenUdon UX follow-up, requiring normal
planning approval before implementation; it is not promoted by this handoff.

## Inspect the actual declared credential inventory before execution

Credential-section prose can accidentally declare a kind token as a binding.
For a synthetic reproduction, analyze a brief declaring `sample_identifier`,
`sample_password` and `sample_seed`, then append ``(kind `totp_seed`)`` to the
last binding line. `credentialBindingNames` in
[plan.go](https://github.com/OpenUdon/openudon/blob/68118634de49ef1ec48dbcd8824dd9467ea2650a/internal/synthesize/plan.go) also declares `totp_seed`. Supplying
only the three intended variables therefore leaves an extra required variable.
Use plain explanatory wording `(TOTP setup seed)` for the kind and compare
the resulting declared/expected inventory with the exact workflow/profile.
Native preparation must pass before consuming an execution attempt.

The consumer reproduced the original missing-variable rejection offline with
synthetic placeholders, then verified the corrected exact three-binding
preparation without invoking an executor. The original consumed failure stays
failed. `TestCredentialInventoryUsesBindingNamesNotRequestTargets` and
`TestAssessReviewRequiresCredentialBindingInventory` retain the neighboring
inventory/approval regressions. The received synthetic before/after probe
exercises the production parser through a temporary test overlay, without
changing supplied source bytes.

**Disposition:** the consumer brief is corrected and the declared inventory is
explicitly checked. OpenUdon owns any future structured kind/name declarations
or reduced preparation diagnostics; these are optional unpromoted improvements.
Changing existing prose extraction is not selected because it could change
historical briefs. A consumer that discards child stderr owns that loss of
detail; do not describe an offline reproduced error as a recovered original log.

## Reuse native evidence only on identical admitted inputs

[E23](https://github.com/OpenUdon/openudon/blob/68118634de49ef1ec48dbcd8824dd9467ea2650a/tabilet/docs/history/status-E23.md) supplies the current-stack input v2
endpoint; E24 additionally binds process namespace membership. Inputs include
the actual external dependency bundle, executable modes, tools, browsers,
display/auth/runtime paths, environment and workstation state. A source-only
consumer change does not protect the cache from a workstation package update.
Record safe input hashes before long stages and recheck at the required
boundaries. An empty ignored runtime directory can also violate native source
cleanliness: use the native source guard, not a weaker Git-status approximation.

Synthetic regressions in [current_inputs_test.go](https://github.com/OpenUdon/openudon/blob/68118634de49ef1ec48dbcd8824dd9467ea2650a/internal/browsersystem/current_inputs_test.go)
cover missing/unsafe/cancelled inputs, changed bytes/modes and namespace
membership. The consumer also checked exact production admission before its
long seed, instead of relying only on fabricated passing reports.

**Disposition:** required E23/E24 source changes are integrated and were
qualified at the exact selected pin. The consumer owns its versioned native
cache, freshness limit, explicit selection and fresh consumer composition.
A miss stops without automatic full-suite fallback. No upstream cache or
workstation update manager is added. Optional reduced changed-input diagnostics
remain with OpenUdon; they must not expose raw environment values.

## Verify output integrity, run identity and cleanup independently

A reduced count report can have the same digest on different successful runs.
Its output digest is not the full producer execution-report digest. Use the
documented schema role for each comparison and unique packet/claim/run evidence
for freshness. A consumer qualification must pass the real native output
through the production parser, rather than checking a fabricated flat scalar
or a lenient regular expression. Cover extra fields, wrong count/digest and
legacy shape rejection. Keep the private output handoff exclusive, owner-only
and deleted after verification.

The consumer implements this handoff and strict parser; its 0/1/3 qualification
passed on the real native envelope. It also joins journey and teardown errors,
retains independently verified process identities and preserves old failures.
A zero count is valid but alone does not prove non-empty production selectors.
Classify a preparation failure, MFA rejection and count extraction failure by
their actual stage; `invalid_response` alone cannot identify the cause.

**Disposition:** consumer parser, fixture, launch supervision and target policy
stay consumer-owned; no target code is copied into OpenUdon. Existing native
report verification remains the upstream authority. More detailed generic
error presentation is an optional OpenUdon integration follow-up, not a new
accepted report contract or an inferred cause for old failures.

## Verification scope

This is a documentation handoff on unchanged source. Focused browser-free
inventory/parser tests, repository checks and documentation checks are enough
for receipt of the lessons. Existing qualified browser evidence retains its
original execution identity; no fresh browser repetition is claimed. Any
future behavior change must follow the owning repository's planning and
qualification rules. This record grants no real-service retry or publication.

Receipt checks on 2026-10-01: the production-parser inventory overlay passed
in 0.159 s; default browser-free tests passed in 111.625 s, vet in 0.900 s,
`check` in 0.615 s, `check-apitools-boundary` in 0.746 s and `check-doc-memory`
in 0.968 s. Diff checks passed. No opt-in browser stage or source change ran.
