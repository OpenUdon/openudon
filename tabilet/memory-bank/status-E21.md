# Status E21 — Repair current-stack Udon build and preserve M86 report meaning

**State:** Active; E21.4 is in progress. Synthetic local qualification only.

**Scope boundary:** Do not contact providers or target accounts, run a public
canary, adopt a runtime, or deploy. Publish the reviewed OpenUdon source only
after current-stack qualification and bounded review pass. W8M adoption and
push remain outside E21.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| E21.1 Freeze M86 current-lock meaning | `[+]` | Copied the exact pre-E21 scenario lock bytes (SHA-256 `57ebe6c70bc0b1e810ed4fb490f36ecb227c47f362bb738b56680ce7abcd6a77`) and integration lock bytes (`9eec17f1489e1c805e2d2bfb8b89a439ee7153d5c49ec09a3ee903ce6761d393`) into versioned v2 snapshots and routed the M86 readers to them. Focused scenario/integration tests pass. M86 integration/loopback/journey reports independently verify at 19/19, 23/23 and 11/11 with their recorded digests unchanged. E21.2 advanced the mutable current locks while retaining these v2 snapshots. |
| E21.2 Pin and emit the repaired current stack | `[+]` | Current scenario/integration locks select Udon `6d32d4967469c579d35adcf47eaddb76a225dbae`; the separate 14-repository closure is exact and checks clean commits before execution. Current scenario/journey and integration contracts emit v3; M86 v2 uses frozen lock snapshots. Focused tests and vet pass; build-input lock SHA-256 is `4993304edf46953c33b6112c4f00e3fcf526811ac91400989b775a19066977b9`. |
| E21.3 Extend native current-stack qualification | `[+]` | Added `browser-system-eval --stack current --suite loopback` and `make browser-system-current-check`; historical remains default v2 and v1/v2 verifiers keep their inventories. Current native v3 routes the 14-source clean closure through the build, scenario, BAP and BRP stages, with clean-source preflight and per-stage rechecks. Focused tests and vet pass. |
| E21.4 Qualify clean current-stack evidence | `[~]` | OpenUdon `f44170d` passes all 19 integration gates, all 23 loopback cases, and all 11 journey cases. Native qualification passes `ui_browser` then fails `registration_ui`; diagnosis found the test leaves an empty ignored `eval/runs` directory, which trips the native clean-source recheck. A test cleanup correction is in progress; repeat all required evidence at its committed source revision in fresh workspaces. |
| E21.5 Review and publish OpenUdon | `[ ]` | Complete all required checks and the bounded review-fix gate with no open P1/P2, record report/source digests and teardown, then push OpenUdon only. Do not push W8M or adopt its candidate. |

## Acceptance

The original M86 integration and two current scenario v2 reports verify with
their recorded digests after the E21 current lock changes. Malformed and
cross-version evidence is rejected. `--stack current` resolves Udon
`6d32d4967469c579d35adcf47eaddb76a225dbae` plus exactly 14 clean locked local
replacements before browser launch. Current scenario, journey, integration
and native reports use v3; v1 historical default and M86 v2 verifiers keep
their original semantics. The 19 integration gates, all 23 loopback cases,
all 11 journey cases, current native scenario/BAP/BRP stages, independent
verification, source cleanliness, and bounded review pass. Only OpenUdon is
published.

## Dependencies and downstream impact

M86 is retired history and supplies the v2 lock/report compatibility contract;
it is verification input, not reopened work. The W8M W20.6 affected candidate
smoke and W21 uncached qualification depend on this published OpenUdon commit
and both current lock digests. No other repository source is changed or
published by E21.

## E21.4 Attempt Record

- Attempt 1 used clean OpenUdon `31765b65a3b36e4daf78d5eddb45801b5eb728b8`.
  Its v3 integration report failed 18/19 because Browserdriver `npm test`
  could not find TypeScript in the clean source checkout. Preserve
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924/evidence/integration-v3.json`
  (SHA-256 `b06a0d19a5d4921c0cf34995e260a683fbce4e39cd6c58617b2484d9e1860d30`).
  Commit `b9d80b8` added read-only lock-matched module validation and disposable
  Browserdriver builds/tests.
- Attempt 2 used `b9d80b8189d84324f05472ff4f36267ca88d7333` but stopped in Udon
  closure preflight before any gate or browser launched: its prior clone
  contained ignored `spider/tmp` test output. No report was written. Preserve
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r2/evidence/attempt-2-preflight.txt`
  (SHA-256 `67ecf14874dbc737bc79df95e8023fe95fe7aa10ba95f1b5179d9942f2fa7109`).
- Attempt 3 used the same OpenUdon commit and a newly clean Udon closure. Its
  integration report passed 19/19 with zero skips and independently verified,
  at `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r2/evidence/integration-v3-attempt-3.json`
  (SHA-256 `907ce9e8a458213836c684f436ba41b3065c2e4718936e04d6287148c91095a6`).
  The post-run clean-source check found ignored `spider/tmp` output in the
  supplied Udon clone, so this report is preserved but is not accepted evidence.
  The pending correction runs Udon test gates in temporary clones of Udon and
  all fourteen exact sibling sources, verifies cleanup, and rechecks every
  supplied current input after each integration gate.
- Integration attempt 4 uses OpenUdon `4efd39c92355c4c5864a0d6b979f865a90531c2e`
  and a fresh exact Udon closure. It passes all 19 gates with zero failures or
  skips and independently verifies at
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r3/evidence/integration-v3-attempt-4.json`
  (report SHA-256 `70298e6fe6453696276beb604ff171e61ded3dbea78615e54f67235df6c90519`;
  sidecar SHA-256 `4a85f9e4c05f20be67773a8e8cf0633687e8f4d1b30e343e1b935c0f6f5d5cc9`).
- Loopback attempt 1 used that same commit and clean source closure. Its
  structurally valid v3 report records 23/23 failures at `fixture_ready` before
  scenario assertions; verification correctly returns failure for this failed
  result. Preserve
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r3/evidence/scenario-loopback-v3-attempt-1.json`
  (SHA-256 `73aa7e4327c17bd5bb7dfe0febfcc100d31330f15d63711c0143291ca4c249d6`).
  Diagnosis showed the compiler could find `tsc` on `PATH` but TypeScript
  resolved modules relative to the original source checkout, which has no
  `node_modules`; the readiness probe also used that checkout as its working
  directory. The correction makes both use an exact disposable Browserdriver
  clone with the supplied read-only modules linked into it. A focused scenario
  must pass before another full loopback attempt.
- Integration attempt 1 on OpenUdon `f44170d2f59a005fce096aef9bc3e05f1d694a63`
  passed the default 16 gates and skipped three explicit opt-in browser gates;
  preserve its report at
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r4/evidence/integration-v3-attempt-1.json`
  (SHA-256 `942706f0194ef993d5f9e7a277f100408cf48d6b3b63a3c4d9b0b818c0b959c5`).
- Integration attempt 2 explicitly ran both opt-in flags and passed 19/19 with
  zero skips; its v3 report independently verifies at
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r4/evidence/integration-v3-attempt-2.json`
  (SHA-256 `f737c5e993d28ff253cf4e7d10ff9cb3b9f1f04a6fc048ce2b215ff0024a1a4d`).
  Loopback attempt 1 passes 23/23 and independently verifies at
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r4/evidence/scenario-loopback-v3-attempt-1.json`
  (SHA-256 `cd69b9cd563ca9e4584e7ce4b34fe5e0a4f9cc05e58ef95bf059a274a0d0e21a`);
  journey attempt 1 passes 11/11 and independently verifies at
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r4/evidence/scenario-journey-v3-attempt-1.json`
  (SHA-256 `459c81966ed9c50187827d4a6d5f12c2fd7e1fa7efca42713d8dab4da0f9f78b`).
- Native attempt 1 bound the same exact source revisions, passed `ui_browser`,
  then failed `registration_ui`. Preserve its failed v3 report
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r4/evidence/native-v3-attempt-1-current-loopback.json`
  (SHA-256 `7b1ee9a945347a0b06b513046e64ee356f4f8e59c957b55a02d21d0b660a0889`)
  and diagnostic
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r4/evidence/native-v3-attempt-1-current-loopback.json.diagnostic.json`
  (SHA-256 `ebf1ec6b27cb90c562c7f769b75cdbd771d2ba6d435fa4e0096f8f33ea5324e2`).
  The diagnostic class is `source_or_runtime_binding`. After the failed stage,
  all source checkouts were clean except that the registration UI test had left
  the empty ignored `eval/runs/` parent in OpenUdon. The correction tracks the
  empty `eval` / `eval/runs` directories it creates and removes only those
  directories after the fixture is removed, preserving any pre-existing path.
- Registration UI cleanup check attempt 1 failed during setup because this
  fresh checkout had no `eval` ancestor. Preserve the diagnostic
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r5/diagnostics/registration-ui-cleanup-attempt-1.txt`
  (SHA-256 `50f257abc28bb3bb844f7cf0da04854d40c0e8eeb2f1a854a69262e8ab0a522b`).
  Attempt 2 passed the real registration UI test and removed both newly created
  directories; preserve
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r5/diagnostics/registration-ui-cleanup-attempt-2.txt`
  (SHA-256 `338ece31449c54d99735cd2a402b0671597c2dacdf52309a83acd9d0e60d52cb`).

## Review and verification record

Review starts at iteration 1 after automated checks and current qualification.
Record each whole-scope review, findings, fixes and verification here. Preserve
the iteration number across continuation and stop after iteration 10 if any
P1/P2 remains.
