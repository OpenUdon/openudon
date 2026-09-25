# Status E22 — Browser 1.10 campaign-row count workflow

**State:** Active. Approved planning; implementation is pending.

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
| E22.1 Pin published Browser 1.10 dependency chain | [+] | Pinned exact published UWS M05, Browsertools M32, Browserdriver M15, and Udon M43 in the separate v4 compatibility and 14-source build-input locks; refreshed OpenUdon's Go module versions and sums. All four sources are clean and the locked commits are reachable from `origin/main`. Both lock SHA-256 values and focused-check evidence are recorded below. |

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
and `git diff --check` passed with module fetching disabled. The E21 current
lock remains selected until versioned v4 report routing is implemented.
| E22.2 Add isolated count workflow and current-stack scenarios | [+] | Added version-isolated Browser 1.10 journey manifests for zero, one and multiple rendered rows, and exercise only `campaign_count`; retained prior current-stack manifests for v3 verification. Focused `go test ./internal/browserscenario ./internal/synthesize -count=1` passed, and one fresh `make browser110-smoke` passed against exact locked Browserdriver/Udon source clones. The first smoke diagnosis found the hidden fixture row's inline style was blocked by the fixture CSP; changed it to the HTML `hidden` attribute and verified the browser count excludes it. The smoke also exposed Node/TypeScript symlink resolution for the read-only external node_modules closure; `--preserve-symlinks` now preserves staged package and runtime module resolution. No supplier source was mutated. |
| E22.3 Preserve old reports and add strict versioned reader | [ ] | Keep v1–v3 readers and their lock semantics unchanged; independently reject malformed or cross-version evidence. |
| E22.4 Verify, qualify and review | [ ] | Run focused checks and current-stack synthetic qualification; verify report bindings and source cleanliness; complete bounded review before publication. |
