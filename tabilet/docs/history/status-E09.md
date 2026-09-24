# Retired milestone E09 - E09 Post-Remediation Release Evidence Closure

**Milestone.** E09
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-E09.md
**Source status SHA-256.** d39bcdc6a60a4c8e028b2cae9ca44c0ee85234e8c96fc07aef6d2625a34148cb
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# E09 Post-Remediation Release Evidence Closure

| Item | State | Notes |
| --- | --- | --- |
| E09.1 Strict typed execution failures | `[+]` | OpenUdon consumes only `udon.execution-report.v2`; failed reports require one closed code, successes contain none, and malformed/missing/v1/unknown/unrelated evidence is `unclassified`. `unclassified` cannot be an expected negative outcome. |
| E09.2 Clean-root and typed public evidence | `[+]` | Browser scenario and integration evaluators reject a dirty OpenUdon root as well as dirty siblings while explicitly ignoring generated `site/`. Public failures use typed report/live-result facts instead of stderr substrings. |
| E09.3 Server-observable MFA approval | `[+]` | Push, number-match, passkey, and security-key replay waits for Udon's exact prompt, approves exactly one observed pending loopback session, then supplies `y`; direct POST bypass is rejected. Runtime state is mutex-protected and method/challenge routing is non-recursive. |
| E09.4 Sandboxed and standalone release gates | `[+]` | Required UI browser qualification rejects the sandbox-disable override, asserts/logs sandbox-enabled execution, and has a separately named unsandboxed diagnostic target. Normal and release CI build `internal/icot` and `cmd/icot` with `GOWORK=off` before full gates. |
| E09.5 Published compatibility set | `[+]` | Browserdriver `f4d76f8ffb60f26f01485954f954616bfb7873bd`, Udon `33e088994655cc2e91119fbb591d44a07cccd47f`, and Browsertools `7ea7e832d7f85060cf57a7a08bf9af6bb7eca896` are remotely resolvable; OpenUdon pins Browsertools `v0.0.0-20260821154836-7ea7e832d7f8` and refreshes the compatibility lock. |
| E09.6 Final qualification | `[+]` | Full tests/vet, focused race tests, `make release-saas-check`, standalone cold-cache download/build/test/vet, strict docs and boundary checks, silent pinned deadcode, six CGO-disabled cross-build targets, clean-root integration plus 21-case loopback and eight-case journey matrices, and all 13 UI journeys with `chromium_sandbox_enabled=true sandbox_required=true` pass. The sandbox-disable override is rejected and was not used. |
| E09.7 Private Udon Actions checkout | `[+]` | Weekly public canaries and tag-release browser scenarios resolve the locked Udon SHA from the compatibility lock and check out private `genelet/udon` with a non-persisted, read-only `GENELET_READ_TOKEN`; public Browsertools and Browserdriver clones remain anonymous, and regression coverage forbids restoring the invalid anonymous `OpenUdon/udon` clone. |

All E09 release gates are complete. The compatibility set and final commits
use the dependency order recorded in the milestone and evolution result.
~~~~~~~~~~~~~~~~~~~~
