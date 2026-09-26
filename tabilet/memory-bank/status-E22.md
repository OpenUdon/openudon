# Status E22 — Browser 1.10 campaign-row count workflow

**State:** Active. E22.1–E22.3 are complete. A fresh v4 qualification passed
on `724da1e`, but bounded review iteration 2 found a report-coverage gap;
corrected implementation, qualification, and review are pending.

**Goal.** Add a synthetic OpenUdon workflow that returns a bounded count of
rendered campaign rows from the default first action=topics page using Browser
1.10 selector match-count output.

**Dependencies.** Published UWS M05, Browsertools M32, Browserdriver M15 and Udon
M43 exact reviewed revisions.

**Compatibility.** Preserve E21 current-stack locks and report readers v1–v3.
Add a current-stack report v4 reader. No live target operation, runtime
adoption, registration, deployment, or campaign mutation is included.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| E22.1 Pin published Browser 1.10 dependency chain | [+] | Pinned exact published UWS M05, Browsertools M32, Browserdriver M15, and Udon M43 in the separate v4 compatibility and 14-source build-input locks; refreshed OpenUdon's Go module versions and sums. All four sources are clean and the locked commits are reachable from `origin/main`. Both lock SHA-256 values and focused-check evidence are recorded below. E22.1 left E21 selected at that time; E22.3 now promotes v4 for current-stack runs while retaining E21 snapshots. |

**E22.1 pins and verification.** The v4 compatibility lock SHA-256 is
`58363021e44961527468bc686df114ce69770709345eb39702fbf38e84da6d2a`; its
14-source build-input lock SHA-256 is
`10fa8b2570f0a72688a1c8d282fe84cf7ea4af6e82ad5ffa85aa6fc994ff371a`.
Pinned producer commits are UWS
`80ee9bfb24a688b5e875dadf9ecacdc65398f1ff`, Browsertools
`3abe70efc03d9ccb97b8b30e5e86328f60a70c64`, Browserdriver
`1f0e0d8c3bf16861f72938fa456035f1226da547`, and Udon
`4266ac99610a6fe39e363c75068e8256bdd821f5`. Exact Browsertools/UWS module
versions and verified sums are in `go.mod`/`go.sum`. Uncached focused
`go test ./internal/browserscenario ./internal/synthesize`, strict JSON parsing,
and `git diff --check` passed with module fetching disabled. E22.3 later
promoted these pins for current-stack report v4 and preserved the former E21
selector as an explicit v3 snapshot.
| E22.2 Add isolated count workflow and current-stack scenarios | [+] | Added version-isolated Browser 1.10 journey manifests for zero, one and multiple rendered rows, and exercise only `campaign_count`; retained prior current-stack manifests for v3 verification. Focused `go test ./internal/browserscenario ./internal/synthesize -count=1` passed, and one fresh `make browser110-smoke` passed against exact locked Browserdriver/Udon source clones. The first smoke diagnosis found the hidden fixture row's inline style was blocked by the fixture CSP; changed it to the HTML `hidden` attribute and verified the browser count excludes it. The smoke also exposed Node/TypeScript symlink resolution for the read-only external node_modules closure; `--preserve-symlinks` now preserves staged package and runtime module resolution. No supplier source was mutated. |
| E22.3 Preserve old reports and add strict versioned reader | [+] | Froze E21's scenario, integration and 14-source build closure as v3 snapshots; selected the v4 Browser 1.10 locks and manifests for `--stack current`. Scenario, integration and native qualification now emit v4. Focused cross-version tests, full `make fast`, `go vet ./...`, strict JSON parsing and `git diff --check` pass. The retained M86 v2 and E21 v3 reports all verify with unchanged digests; exact hashes follow. The initial `make fast` exposed an expired fixed-date registration fixture; changed it to a dynamic whole-second clock and confirmed the isolated test and full gate pass. |
| E22.4 Verify, qualify and review | [~] | Review iteration 2 found that named Browser 1.10 count tests were absent from v4 integration selectors despite guide claims; the pre-review 724da1e reports remain preserved but do not close E22. The v4 selector now adds named OpenUdon count fixtures, Browsertools producer, UWS schema, Udon v11 consumer, and Browserdriver extraction tests. M86 v2 and E21 v3 use the separately frozen 19-gate inventory (SHA-256 `65554d243f11c3514c0bff0fef619d04a46230ea0bbd8bb0a11b34392f602f5b`); all seven retained M86/E21 reports pass verification. `make fast`, `go vet ./...`, tracked JSON parsing, documentation-memory validation, focused integration tests, and `git diff --check` pass. Fresh full qualification and review iteration 3 remain. Earlier pass/failure reports remain preserved. |

**E22.3 retained-report verification.** The three M86 v2 reports independently
verify with 19/19 integration, 23/23 loopback and 11/11 journey passes. Their
report SHA-256 values remain integration
`69feb11ba4742b43e12961b98a1b79d64fe304ce917880c480e62a91d4d7b469`, loopback
`b3e80bba78c1b7c9ea612784b66d8a0c391f182a0bc5efb16a0cb22480c8ca40`, and
journey `ab56fb0ae10118be02bb7758d098c2dbfe3c0df0ebec24493d9490917ddfb6ce`.
E21's v3 integration, loopback, journey, and native qualification reports also
independently verify against their frozen v3 locks and inventories, with SHA-256
values `042a5c46d5265b5802a76068c3742d7ebf20786fbfd294e16405a1893de9b2f4`,
`97373fc50901d192b7d770c33906a72bd0593ba0759dc9c1e4ff4a6a4bdbd1c9`,
`e6414861599495a28a13775467887720d4c8c812759f4fa2ef17c1cea518ca54`, and
`9d325601c9499171cfb440c2735c789b8ae3b7db583b64314dadb29c9bf71d18`.

**E22.4 failed integration attempt and diagnosis.** The first fresh v4 matrix
attempt is retained at
`/home/peter/.local/state/openudon/e22-browser110-qualification-20260925-znGWHA/evidence/integration-v4.json`
with SHA-256
`4398239ae5168cd3b6db880ab81caff105288928476aee3217a59ee670ac3841`.
All five primary locked sources were clean and exact, all 19 gates ran with no
skips, and 18 passed. The only failure was `browserdriver-runtime`: the npm
gate's read-only module-root symlink resolved outside the disposable package,
so two synthetic child-process tests could not find `playwright-core`. The
gate now copies the already lock-validated module tree into a read-only standard
`node_modules` directory in its disposable source clone. The focused evaluator
test passes, and a fresh exact M15 Browserdriver npm run passes 157 tests with
15 existing skips; the E22.4 matrix itself must be rerun in a new output root.

**E22.4 v4 integration rerun.** On clean OpenUdon commit
`5b07bf7f0267291108c0e17327c2bd41edd87a08` and the exact v4 source pins, the
fresh matrix passed all 19 gates with zero skips. Its report is retained at
`/home/peter/.local/state/openudon/e22-browser110-qualification-20260925-MVqzdy/evidence/integration-v4.json`
with SHA-256
`97c8280f25c297a7d01c19ac860797cbe3a3faa29690c98e8e0f1fb3e70223f9`.
Independent v4 verification and exact five-repository binding passed.

The following fresh loopback run on that same commit failed all 23 cases as
`dependency_unavailable`, with zero skips; its report digest is
`ffe87a4e83bce1e77c9189e668e727742c2b9cfa63ec679a0c662c7056500bff`. A fresh
single Browser 1.10 multiple-count diagnostic also failed during readiness,
with report digest
`52225b3069628b90d09e3dc39f25329cbaf190ccbce456c734ecf53472e1226b`.
The Node readiness probe omitted `--preserve-symlinks`, so the external
read-only `node_modules` symlink hid the sibling `playwright-core` package.
The same locked Node 24.13.0, Playwright 1.62.1 and Chromium 151.0.7922.34
probe passes with that option, which is now added to the readiness command.
Both failure reports remain intact; full qualification must use a new clean
commit and output root after the focused count journey passes.

**E22.4 Browser 1.10 report/doc review.** On clean OpenUdon
`533039a7531477a2fd9710ef26bf1a5d5c1546e4` and the exact v4 source closure, the
fresh 19-gate integration matrix passes 19/19 with no skips; the full loopback
suite passes 23/23 and the full journey suite passes 14/14, including the
zero-, one-, and multiple-row Browser 1.10 cases. The journey report independently
verifies and binds all five primary sources clean at that commit; SHA-256 is
`48856e3116810ade3983e278d4ab45664d814322d2c61b30cfe7f1842eb06dbb`.
All 17 clean staged repositories match their exact locked revisions. A bounded
documentation review then found that the operator guides still described v3
as current. Their scenario, integration and native report descriptions now
identify v4 as current while preserving E21 v3 and M86 v2 readers, and the
product record states the synthetic 0–100 first-page boundary. Memory link,
JSON and whitespace checks pass. The documentation-only source change means
the final full qualification must bind a new clean OpenUdon commit; the earlier
reports remain immutable evidence for `533039a` and are not substituted for
that run.

**E22.4 first native v4 failure and diagnosis.** The fresh qualification at
clean OpenUdon `ce2be144fcfa477ee0b2c0f3631e30aa99073027` is preserved at
`/home/peter/.local/state/openudon/e22-browser110-qualification-20260925-final-5EF73f/evidence/native-v4.json`
with SHA-256
`3c235b9d67234b6dfe7a9b3f3b764fc55ad2a52e38df9846a41e06a97ea0952d` and
owner-only diagnostic digest
`858b63518ca9f643951857dc11918543d38948398f2997c6d30bf5c7a787ca05`. Its
first six stages pass; `registration_driver` fails before test bodies execute.
The bounded diagnostic reports five `ERR_MODULE_NOT_FOUND` failures for
`playwright-core` from the Node tests launched through the readonly external
`node_modules` symlink. A fresh minimal import reproduces failure without Node's
symlink preservation and passes with `--preserve-symlinks`. OpenUdon's native
Browserdriver test launcher now applies the same flag and adds an opt-in
regression test for that exact five-file synthetic stage. All staged sources
remain clean. The failed aggregate remains immutable; the affected stage must
pass before another full qualification attempt.

**E22.4 clean-source qualification on 6cfee9a (pre-review guidance fixes).** A fresh, clean Browserdriver registration
driver stage passes all five tests with no skips on OpenUdon
`6cfee9ade21694f081eb72e07f2da71062142ab5`. One subsequent integration command
ran all 19 gates successfully but could not write a report because an empty
ignored `.openudon-run/` directory left by the earlier `make fast` run made the
final source-clean check fail. The owned empty directory was removed, cleanliness
was rechecked, and a new integration attempt passed all 19 gates with no skips.

The final integration report is
`/home/peter/.local/state/openudon/e22-browser110-qualification-20260925-retry-Lq8QVS/evidence/integration-v4-clean.json`
with SHA-256
`24e79d1d9370b8577e095f60d43cdf106d30df83925bb6ec90bb4fe0da9c9ff8`.
The fresh loopback report passes 23/23 with zero skips and has SHA-256
`376d9371a19c4d691091920f4029b0139df969537703f5eaf2a5844aeb534b91`. The
fresh journey report passes 14/14 with zero skips, including all three count
cases, and has SHA-256
`a8b9505ed1774ee661c7c9c6c17cb6fdb09ca894d2e20bf68de9e45527f4376f`. The
uncached native v4 report passes three repetitions of all 13 required stages
(39/39, zero skips) and has SHA-256
`f254e6d648d0eea6f8537e5e3ecf463c0fadb33aa26e81e009b47724ef09310d`.
All reports independently pass their v4 readers. The native report's 19
commit-and-source-digest bindings were independently recomputed and match the
clean worktrees; Go is 1.26.6, Node is 24.13.0, and the lock binds Playwright
1.62.1 and Chromium 151.0.7922.34. All 23 loopback and 14 journey teardown
phases pass in each native repetition. The native runner left no owned scenario,
component, Udon-build, or Browserdriver-test temporary directories.

Focused uncached `go test` passes for `internal/browserscenario`,
`internal/browserintegrationeval`, and `internal/browsersystem`, including
unknown/malformed report rejection and v1/v2/v3/v4 cross-version lock and
inventory checks. The exact Browserdriver registration-driver stage passed
5/5 with zero skips. The complete `make fast`, `go vet ./...`, strict JSON,
memory-link and whitespace checks passed after the final source fix. Retained
M86 v2 and E21 v3 reports continue to verify with their recorded digests.

**E22.4 bounded review.** Iteration 1 reviewed the full E22 range
`92c6839f8bbf54fc20f32efb73fbb4d6d375f16a..6cfee9ade21694f081eb72e07f2da71062142ab5`.
Iteration 1 found one blocking P2: current operator/help and stack descriptions
still presented E21 v3 and Udon v10 as the active Browser 1.10 path. The scenario
CLI help omitted the three Udon v11 count journeys; the integration guide
described v3 runs instead of the retained v3 verifier and omitted v11 from the
count handoff; the current journey inventory still said 11; and the active
build-input selector comment said v3. The finding was corrected in clean
commit `724da1e377bc6248e1b0bc9167d5892d915f30e7`. Focused package tests, vet,
documentation-memory validation, strict JSON parsing, CLI help inspection,
whitespace checks, and the full fresh v4 qualification pass on that commit.
Review iteration 2 then found the report-coverage gap recorded below.

**E22.4 pre-review clean v4 qualification on `724da1e`.** The fresh run is retained
at `/home/peter/.local/state/openudon/e22-browser110-qualification-20260925-reviewfix-IAZFrL/evidence/`.
Integration passed 19/19 gates with no skips; its independent v4 verifier passed
and its SHA-256 is
`fdfd57e8960aa63a2950c04f5234446eed37a45f2df54650d18ff2289461cdca`.
Loopback passed 23/23 with no skips or quarantines; independent verification
passed and SHA-256 is
`fe0324c7762008974092ade4b4aae8cc11b8dea26d0e089ad804bf259f39007a`.
Journey passed 14/14, including zero-, one-, and multiple-row count cases, with
no skips or quarantines; independent verification passed and SHA-256 is
`41a400b986028fdcc06cd3a25bbe82b2d2f556e97ed9c5ce4401001006d61d9c`.
Native v4 passed three fresh repetitions of all 13 required stages (39/39,
no skips); its independent verifier passed and SHA-256 is
`ecdc8fe260fabd1e03829e6b643cc5045b16883f9e91104610b6848be1d4ebf6`.

The native report's 19 commit/source-digest bindings were independently
recomputed and all match. Its 17 unique staged repositories are clean and at
the locked commits. Toolchains are Go 1.26.6 and Node 24.13.0; Playwright is
1.62.1 and the executed Chromium is 151.0.7922.34. Across the three native
repetitions, all 69 loopback and 42 journey cases pass; all 111 embedded
teardown phases pass. The qualification and observed temporary browser roots
left no active evaluator process or run-owned scenario root. The reports bind
the clean implementation commit above; this ledger entry records their
evidence and does not change that implementation commit.

**E22.4 bounded review iteration 2.** Started after the pre-review qualification.
The review rechecks the full E22 range
`92c6839f8bbf54fc20f32efb73fbb4d6d375f16a..724da1e377bc6248e1b0bc9167d5892d915f30e7`,
including v1–v4 report dispatch, frozen v2/v3 locks, the v4 current stack,
synthetic count cases, build-input containment, native BAP/BRP stages, operator
guidance, and downstream E21/W8M bindings. Finding `E22-R2-1` is P2: the guide
claims Browser 1.10 count coverage in the OpenUdon handoff, Browsertools
producer, UWS contract, Udon consumer, and Browserdriver runtime, but the
19-gate v4 report currently requires only Browser 1.8/1.9 and v10 markers for
those paths. The count journey and native reports do cover the feature; the
integration report does not attest those named upstream tests. The correction
preserves the old gate contract behind a frozen selector and adds explicit
count markers to v4. M86 v2 and E21 v3 integration, loopback, journey, and E21
native reports all verify with their recorded digests after the change. The
pre-review v4 reports remain valid for their bound source commit and are not
rewritten. Corrected-source qualification and review iteration 3 are pending.

**E22.4 count-marker integration attempts on `9be9ff3`.** The fresh matrix at
`/home/peter/.local/state/openudon/e22-browser110-qualification-20260925-countcoverage-Wu4p/evidence/integration-v4-count-markers.json`
passed all 16 mandatory gates and independently verifies; three optional
installed-browser/auth gates were skipped. SHA-256 is
`215c6ca2bfd0b80015a52e5f9725064dec8e9f2bb446c1d7372be0b29ad665c7`. The
OpenUdon count-profile/current-report gate passed 27 named tests; Browsertools
count producer passed 10, UWS count schema passed 8, Udon v11 consumer passed
12, and Browserdriver passed its full 157-test suite.

A separate 19-gate run requested every opt-in and is preserved at
`/home/peter/.local/state/openudon/e22-browser110-qualification-20260925-countcoverage-Wu4p/evidence/integration-v4-full.json`;
it failed 16/19 with all three browser/auth gates failing, SHA-256
`62e72e3af20e4101b881a25212e83d452244b70574c6b430b6e7191451a7ccf6`. A fresh
Browsertools clone at the exact published commit reproduced the installed
Chromium launch failure: Chromium was unavailable, so Firefox/WebKit portability
could not establish its Chromium baseline. The host currently reports
`kernel.apparmor_restrict_unprivileged_userns=1`, and an unprivileged user
namespace probe is denied. No source checkout was changed. Full loopback
qualification requires resolving this host sandbox prerequisite; a request to
temporarily set it to `0` for local synthetic tests and restore `1` is pending
the user's response. Do not replace either integration report with a later
attempt.
