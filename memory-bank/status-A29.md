# A29 — Generic registration 1.1 through iCoT

| Item | State | Notes |
| --- | --- | --- |
| A29.1 Adopt v3 producer evidence and transaction lineage | `[+]` | Published UWS 1.1 and Browsertools v3/result v3/review v2; independent evidence and legacy nonmix tests pass. |
| A29.2 Add deterministic guided form authoring | `[+]` | Typed definitions, conditional fields, reviewed public previews, ordered checkpoints and shared HTTP/control authority; actual UI qualification passes. |
| A29.3 Carry symbolic input bindings through packaging and handoff | `[+]` | UWS call 1.1 inputBinding and exact external private-form handoff; actual Start/Apply/submit replay and retained-artifact privacy scan pass. |
| A29.4 Qualify and publish | `[+]` | Published source passes full owner checks, race, staged native v4/v5 and the complete three-unit W15 consumer gate. Independent verification and exact runtime adoption pass; final review 9 has no P1/P2 findings. |

Approved successor to A28, dependent on Browsertools A12, Udon M39 and
Browserdriver M11. The Udon private form remains a separate runtime UI, invoked
through the external CLI. W8M supplies only a consumer test case and operating
policy. A26/A27 consumed attempts and containment remain enforceable in the
shared application service, including supervised and HTTP adapters.

## Implementation and review history

Review iteration 0 of maximum ten. No P1/P2 at closure; no live target or
qualified-runtime adoption is claimed by this implementation status.

Review iteration 1: native typed wizard qualification exposes a P2 editor
integration gap: newly added fill-input steps must include ordinary field
symbols in their slot selector. Correct the selector, then repeat the native
UI to canonical BRP and package journey. The legacy offline suites pass except
for a child CLI build still using the old dependency pin; final published-pin
qualification remains pending.

The same iteration also corrects the client completeness check for input
checkpoints, which carry slot lists rather than browser locators. Inspection of
the producer adapter finds the analogous populated-slice copy issue; decode
observations into a fresh destination and test nested isolation.

Native qualification identifies an existing public-contract limit: BRP 1.1's
closed locator-role enum excludes spinbutton. Keep that omission diagnostic and
test numeric scalars through supported textboxes; do not mutate the published
UWS schema or claim ordinary spinbutton support. This limitation remains a
future UWS-contract candidate, outside A29's approved 1.1 scope.

The actual typed iCoT browser journey now passes through preview, BRP review,
UWS build and promotion. Review iteration 2 examines consumer/UI authority and
finds a P2 shutdown gap: the HTTP entry point must explicitly join the same
application close gate used by supervised control. Add that join before the
W8M UI adapter can claim successful teardown. Prepared private inputs also need
an expected initial snapshot identity transferred only through private runtime
environment, so a newly armed packet cannot switch to another form revision.

The complete actual typed iCoT-to-BRP/UWS-to-private-form-to-v5 journey passes,
including Start, Apply, exact submit approval, one POST, final form clearing and
scanning every retained package/run file for synthetic private values and the
initial snapshot digest. HTTP/control consumer authority and version/schema
refusal tests pass. Full tests, vet, make check, documentation-memory and example
validation pass against the published dependency pins.

Review iteration 3 finds a pre-existing qualification boundary problem: the
native driver check rebuilt dist in its supplied source worktree. Stage builds
in a disposable directory against already installed modules instead. Extend the
same native driver stage to both v4 and v5. Final race, staged qualification and
consumer acceptance remain pending before closure.

Review iteration 4: staged native v4/v5 tests pass with supplied source bytes
and modification times unchanged. The complete actual UI-to-private-runtime
journey passes from staged output. Parallel verification exposed a fixture wait
budget covering all DOM editing as one response; give that full sequence a
bounded 120 seconds while retaining 15-second individual control waits. Remove
the temporary response-message diagnostic. The prompt-process race regression
passes independently; its parallel failure was a scheduling threshold. Final
make check passes after these fixes. No remaining P1/P2 in the implementation
review; exact consumer qualification remains the W15 gate.

Scoped source commits: A29.1 `645a49d`, A29.2 `401e463`, A29.3 `a9ea48b`,
A29.4 qualification/docs `9494afe11e58172d2a337c8c38304053f3cfba8b`.
The complete tree passed owner checks before row commits; publication exposes
the dependency-closed final head. Final W15 qualification remains outstanding.

Review iteration 5: the immutable native component passes three full repeats,
but the W8M consumer rejects the generated workflow before launch. Version
selection recognizes only BRP 1.0 when requesting UWS 1.9.0; add BRP 1.1 to
that rule and assert the exported UWS version in a typed generation regression.
The runtime accepted the older document version, so native replay alone did not
expose this consumer compatibility defect. Source publication and complete
consumer qualification must use the corrected OpenUdon head.

The typed export regression now asserts UWS 1.9.0, registration call 1.1 and
the exact inputBinding. The actual generic typed replay also includes the
post-submit human checkpoint through the private form; Start, Apply, submit,
Continue, one POST and artifact privacy all pass. Full make check and vet pass.
Review iteration 6 finds no remaining P1/P2 in the corrected implementation;
the new head still requires complete consumer qualification before closure.

Review iteration 7: W15's third aggregate passes two native repeats, then fails
repeat three at loopback_scenarios. All twenty source trees independently match
their original bindings. The native diagnostic says component_validation but
drops the scenario report returned with the error, leaving no failed-case or
phase detail. Diagnose the unchanged loopback suite independently, then retain
its bounded private failure report through the existing diagnostic channel.
This reporting gap is P2; do not turn failed or missing evidence into a pass.

The unchanged isolated loopback suite passes all 23 cases with no skip or
quarantine, so the original failure has not reproduced. The diagnostic
correction preserves the returned report and cause in the existing bounded
private fields. Its regression proves failed-case/phase preservation, closed
ordinary errors and absence from progress or successful report evidence.
Full tests, vet and make check pass. Review iteration 8 finds no remaining
P1/P2 in this correction. Fresh focused repetitions and complete consumer
qualification remain required; no failed aggregate is reclassified.

The reporting correction is published in OpenUdon commit `1f8b08f`; the UWS,
Browsertools, Udon and Browserdriver revisions remain unchanged.

Final consumer evidence: W8M W15's fresh fourth aggregate passes three complete
units, including 117 native stages and nine synthetic workflow receipts. The
original native reports and aggregate pass owner verification. Independent
recomputation verifies twenty source bindings and the same eight actual
runtime/module hashes across all three units; W15 adopts exact tested bytes.
Aggregate SHA-256: `49dc9e251c86eb4859eb12b088965b5bfe620419c68ee72f1b58aefae75c3b47`.
Qualified source heads remain Browsertools `ec0b9e9d6ca1`, OpenUdon
`1f8b08f150f9`, Udon `ad257817e5fd`, Browserdriver `22eb8f1b5e3d` and unchanged
UWS `9ff877ebce55`. Later coordination commits do not repin that qualified
closure. Failed predecessor aggregates remain failures; no live W8M operation
or account outcome is claimed. The existing evolution direction is unchanged.

Review iteration 9 confirms the full evidence and generic UI/private-runtime
boundaries with no P1/P2 findings. The earlier unreproduced scenario failure
remains recorded separately from the diagnostic correction and fresh passing
qualification. A29 is complete.
The current documentation-memory check and final diff check pass.
