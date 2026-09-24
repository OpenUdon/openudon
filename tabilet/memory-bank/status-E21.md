# Status E21 — Repair current-stack Udon build and preserve M86 report meaning

**State:** Active; OpenUdon publication is complete and downstream W8M W21 qualification is in progress. Synthetic local qualification only.

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
| E21.4 Qualify clean current-stack evidence | `[+]` | Final qualified code commit is `1007cdedf0acebf649bd3ddd065a0e42bac4f542`. A fresh source workspace passed the v3 integration matrix 19/19, loopback suite 23/23, journey suite 11/11, and native qualification 3 passes × 13 stages with zero failures, skips, or quarantines. Exact report hashes and bindings are in the E21.4 attempt record below. Earlier loopback worker EOF and native BRP cleanup failures remain preserved as failed evidence; they are not rewritten by this success. The direct focused BRP component also passed and left all source checkouts clean. Retained M86 v2 reports still verify at 19/19, 23/23, and 11/11 with unchanged SHA-256 digests. Focused malformed/cross-version tests, full `make fast`, `go vet ./...`, documentation-memory checks, independent report verifiers, source cleanliness, and temporary workspace teardown all pass. |
| E21.5 Review and publish OpenUdon | `[+]` | Bounded review iteration 1 found no P1/P2 issues. Fast-forwarded and pushed OpenUdon `main` to `origin` at `679f0bca630862a4ff46ea5a7edc0fc871abe938`; the exact qualified code commit `1007cdedf0acebf649bd3ddd065a0e42bac4f542` is published in its history and remains the W8M dependency. Only OpenUdon was pushed. W8M remains local and unadopted. |

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
  the empty ignored `eval/runs/` parent in OpenUdon. The registration UI,
  supervised control, and authenticated package fixtures now use one cleanup
  helper. It removes only empty `eval` / `eval/runs` directories created by the
  test after fixture teardown, preserving any pre-existing path.
- Registration UI cleanup check attempt 1 failed during setup because this
  fresh checkout had no `eval` ancestor. Preserve the diagnostic
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r5/diagnostics/registration-ui-cleanup-attempt-1.txt`
  (SHA-256 `50f257abc28bb3bb844f7cf0da04854d40c0e8eeb2f1a854a69262e8ab0a522b`).
  Attempt 2 passed the real registration UI test and removed both newly created
  directories; preserve
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r5/diagnostics/registration-ui-cleanup-attempt-2.txt`
  (SHA-256 `338ece31449c54d99735cd2a402b0671597c2dacdf52309a83acd9d0e60d52cb`).
- Native attempt 2 on `1c2a916b3a99879a698e5ff4cc200cfd080dfcc3` passed
  `ui_browser` and `registration_ui`, then failed at `supervised_control` with
  diagnostic class `source_or_runtime_binding`. Preserve the failed report
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r7/evidence/native-v3-attempt-2-current-loopback.json`
  (SHA-256 `e6dc734d777e624f1c8e8c85ec54749ff334036e041f3a0685cfaed022c30816`)
  and diagnostic
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r7/evidence/native-v3-attempt-2-current-loopback.json.diagnostic.json`
  (SHA-256 `35554b45a6f846581f4abf76bf8fe82d7f7bf9dbec5bdcca53ffab5df1f9d217`).
  That fixture still used `MkdirAll(eval/runs)` without removing its empty
  ignored parent; the shared helper now covers it and the authenticated package
  test.
- Focused fixture set attempt 1 passes the real registration UI, supervised
  control, and authenticated package tests with uncached local browser state.
  It preserved the pre-existing `eval/runs` directory from the failed native
  attempt. Preserve
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r5/diagnostics/registration-fixtures-cleanup-attempt-1.txt`
  (SHA-256 `7b400c5d2ecbb108b3d6a2a57d0c03be265c8a86bc56989b4927750b41ac10b4`).

## Accepted E21.4 Qualification

All accepted current-stack reports were produced from clean isolated source
checkouts at OpenUdon `1007cdedf0acebf649bd3ddd065a0e42bac4f542`, Browsertools
`9333a9f25dbb17551998a429e123e7a9ba976648`, UWS
`e9b6181be0abb7f683fdb624d4dba282a59991d1`, Udon
`6d32d4967469c579d35adcf47eaddb76a225dbae`, and Browserdriver
`8c13b70d30a500e65e90a95a203493301b8b21a5`. The report-bound 14-source Udon
closure is the exact current build lock above; all source checkouts were clean
before and after the runs. Runtime inputs were Go 1.26.6, Node 24.13.0,
Playwright-Go v0.6201.0, and lock-matched Browserdriver dependencies
`@types/node` 24.5.2, Playwright 1.62.1, and TypeScript 5.9.2.

- Integration report
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r13/evidence/integration-v3-attempt-1.json`:
  19/19, zero skipped; SHA-256
  `042a5c46d5265b5802a76068c3742d7ebf20786fbfd294e16405a1893de9b2f4`;
  sidecar SHA-256
  `77c2349aca3ceefa602d033005f6ad53e72663ce707263c364db0b649a86bfc7`.
- Loopback report
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r13/evidence/scenario-loopback-v3-attempt-1.json`:
  23/23, zero skipped or quarantined; SHA-256
  `97373fc50901d192b7d770c33906a72bd0593ba0759dc9c1e4ff4a6a4bdbd1c9`;
  sidecar SHA-256
  `22394f309610257d703e7e17d8df10cfb19be9a61fac58f3f4959c41009dc19b`.
- Journey report
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r13/evidence/scenario-journey-v3-attempt-1.json`:
  11/11, zero skipped or quarantined; SHA-256
  `e6414861599495a28a13775467887720d4c8c812759f4fa2ef17c1cea518ca54`;
  sidecar SHA-256
  `d866f13a1d73ff71d6a75b828881799841dcca7366d4c3a936134dadf16624e8`.
- Native report
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r13/evidence/native-v3-attempt-1-current-loopback.json`:
  3 passes × 13 stages, all stages pass; SHA-256
  `9d325601c9499171cfb440c2735c789b8ae3b7db583b64314dadb29c9bf71d18`.
  Its independent v3 verifier passes. All 19 report source records bind exact
  locked commits and source hashes; every loopback and journey subreport has
  zero skipped or quarantined cases.
- Focused BRP component evidence
  `/home/peter/.local/state/openudon/e21-current-qualification-20260924-r13/diagnostics/brp-component-attempt-1.json`:
  SHA-256 `b7e879d4a431b6bf50545d264f55b14d5bd960a028486b2a4d220497c3a7ef32`.
  It confirms only fixture GET/HEAD authoring requests and the approved
  synthetic runtime POST; its temporary example and all source checkouts were
  clean after teardown.
- Frozen M86 report hashes remain integration
  `69feb11ba4742b43e12961b98a1b79d64fe304ce917880c480e62a91d4d7b469`,
  loopback
  `b3e80bba78c1b7c9ea612784b66d8a0c391f182a0bc5efb16a0cb22480c8ca40`, and
  journey
  `ab56fb0ae10118be02bb7758d098c2dbfe3c0df0ebec24493d9490917ddfb6ce`.
  Each verifies against its preserved v2 contract.

## Review and verification record

Iteration 1 reviewed the full E21 diff from origin/main through the accepted
code commit after qualification. No P1/P2 findings remain. Review checked the
v1/M86-v2/current-v3 verifier dispatch, the pinned current Udon closure,
read-only dependency staging, source-clean rechecks, native stage inventories,
and registration fixture cleanup. Focused cross-version/malformed-evidence
tests pass; `make fast`, `go vet ./...`, `check-doc-memory`, independent
verification of all four current reports and all three retained M86 reports,
all-source cleanliness, and disposable-workspace teardown checks pass. OpenUdon
was pushed alone to `origin/main` at
`679f0bca630862a4ff46ea5a7edc0fc871abe938`; the accepted runtime code commit
is `1007cdedf0acebf649bd3ddd065a0e42bac4f542`. W8M remains local and
unadopted; no live operation is included. E21 remains active for downstream
W8M W21 reconciliation.
