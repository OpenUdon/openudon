# E09 Post-Remediation Release Evidence Closure

Item | State | Notes
--- | --- | ---
E09.1 Strict typed execution failures | `[+]` | OpenUdon consumes only `udon.execution-report.v2`; failed reports require one closed code, successes contain none, and malformed/missing/v1/unknown/unrelated evidence is `unclassified`. `unclassified` cannot be an expected negative outcome.
E09.2 Clean-root and typed public evidence | `[+]` | Browser scenario and integration evaluators reject a dirty OpenUdon root as well as dirty siblings while explicitly ignoring generated `site/`. Public failures use typed report/live-result facts instead of stderr substrings.
E09.3 Server-observable MFA approval | `[+]` | Push, number-match, passkey, and security-key replay waits for Udon's exact prompt, approves exactly one observed pending loopback session, then supplies `y`; direct POST bypass is rejected. Runtime state is mutex-protected and method/challenge routing is non-recursive.
E09.4 Sandboxed and standalone release gates | `[+]` | Required UI browser qualification rejects the sandbox-disable override, asserts/logs sandbox-enabled execution, and has a separately named unsandboxed diagnostic target. Normal and release CI build `internal/icot` and `cmd/icot` with `GOWORK=off` before full gates.
E09.5 Published compatibility set | `[+]` | Browserdriver `f4d76f8ffb60f26f01485954f954616bfb7873bd`, Udon `33e088994655cc2e91119fbb591d44a07cccd47f`, and Browsertools `7ea7e832d7f85060cf57a7a08bf9af6bb7eca896` are remotely resolvable; OpenUdon pins Browsertools `v0.0.0-20260821154836-7ea7e832d7f8` and refreshes the compatibility lock.
E09.6 Final qualification | `[+]` | Full tests/vet, focused race tests, `make release-saas-check`, standalone cold-cache download/build/test/vet, strict docs and boundary checks, silent pinned deadcode, six CGO-disabled cross-build targets, clean-root integration plus 21-case loopback and eight-case journey matrices, and all 13 UI journeys with `chromium_sandbox_enabled=true sandbox_required=true` pass. The sandbox-disable override is rejected and was not used.
E09.7 Private Udon Actions checkout | `[+]` | Weekly public canaries and tag-release browser scenarios resolve the locked Udon SHA from the compatibility lock and check out private `genelet/udon` with a non-persisted, read-only `GENELET_READ_TOKEN`; public Browsertools and Browserdriver clones remain anonymous, and regression coverage forbids restoring the invalid anonymous `OpenUdon/udon` clone.

All E09 release gates are complete. The compatibility set and final commits
use the dependency order recorded in the milestone and evolution result.
