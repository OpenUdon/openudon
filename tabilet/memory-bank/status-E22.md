# Status E22 — Browser 1.10 campaign-row count workflow

**State:** Active. E22.1–E22.2 are complete; the v4 verifier/current-stack
transition and qualification are underway.

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
| E22.4 Verify, qualify and review | [~] | Both Browserdriver npm staging and Node readiness symlink failures are repaired. A fresh exact-source run passes the 19-gate integration and 23-case loopback suites; the 14-case Browser 1.10 journey independently verifies. Review found operator documents that still called v3 current; the version and lock references are corrected before the final clean qualification. |

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
