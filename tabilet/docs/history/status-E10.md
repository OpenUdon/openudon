# Retired milestone E10 - E10 Cross-Package Browser Transaction Qualification

**Milestone.** E10
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-E10.md
**Source status SHA-256.** e399399d054602319cf6082b0707ee82df78530a3bc37723db4838e86330c7bf
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
# E10 Cross-Package Browser Transaction Qualification

| Item | State | Notes |
| --- | --- | --- |
| E10.1 Deterministic value-free evidence report | `[+]` | OpenUdon `a8806e1a7cd579e46ce53c02c3242ee5755f9957` adds the canonical `openudon.browser-transaction-qualification.v1` report and independent `browser-transaction-eval --verify` path. The closed wire fixes 18 ordered gate IDs, typed failures, exact local-unpublished OpenUdon and published locked Browsertools/UWS identities, nine lifecycle digests for each BAP+BCP/BRP case, sandbox/loopback/GET+HEAD/zero-POST/no-account/no-registration-executor posture, derived summaries, and a SHA-256 sidecar. It has no free-form or path-bearing evidence field. Strict bounded reading rejects symlinks, unknown/missing/null/duplicate fields, noncanonical JSON, contract/dependency drift, tampering, and private-material-shaped additions. Focused/full/race/vet/strict-doc/doc-memory/diff gates pass; task review found no P1/P2 issue. |
| E10.2 Authenticated BAP+BCP loopback | `[+]` | Complete at OpenUdon `f2125591562f1c21eea85c15c61de35b0483bb9c`. A real Browsertools-produced authentication/capability pair passes OpenUdon adoption, reviewed same-session composition, prepare/qualify/atomic promotion with a rebuilt private prior generation, selected review and trusted dry run, actual packaged UWS/Browserdriver replay, server-observed password/TOTP behavior, typed outputs, and nine unique value-free lifecycle digests under sandboxed Chromium. Iteration 1 found package-quality, BAP 1.1 classification, and path-bearing diagnostic defects, fixed by `da53e9bf88e1bca0e8733dba325feab0fd8bce79`, `0b734b43450f772e2001f7507b2389cc9e62f005`, and `d3a3a95570d88f1f5ab5bd5f7e9efb56b5e6a80e`. Iteration 2 found a P2 opaque-session posture defect, fixed by `ccec7442afee398fcbd7b2a204c5cb3495716d40` and `55450907c99758a597881f9b62a5d010365079d8`. Iteration 3 found stale prior-fixture quality and incomplete lifecycle diagnostics, fixed by `190b81add6d0bcfaa559abb73e2556fb114eeaab` and `8abb75be59cc23fdd96eb636878ca8e8e05827fa`. Iteration 4 found a P2 UWS output-binding defect; `d4a02442f82b0131b5497528eca93ae676afe092` through `5a042ef7179bf34da9d2c8b0dee6ea5d179e8b3a` close replay diagnostics and `e20204e3c1976c3f9650d3c9f53ea07cc2b65ebd` binds only proven browser outputs to native `$steps` expressions. Iteration 5 found valid bare package/handoff digests at the tagged E10 evidence boundary; `f2125591562f1c21eea85c15c61de35b0483bb9c` normalizes only validated SHA-256 values. Iteration 6 passes the real `browser-transaction-bap-bcp` gate from a clean commit. Focused tests, focused race, vet, full `go test ./...`, and diff checks pass; bounded review finds no unresolved P1/P2 issue. |
| E10.3 Zero-POST BRP loopback | `[+]` | Complete at OpenUdon `ab26da298cfda0df3b43c1b45743e541ad41e3fe`. `d4ba92aefd3f18d6299955d01beaaf77b386d39c` adds a real published-Browsertools registration worker run against a deterministic loopback fixture, strict private candidate adoption, session-free virtual-source composition, prepare/qualify/atomic promotion with prior preservation, selected review, trusted dry run, and a non-dry invocation sentinel. Retained evidence contains only nine unique tagged lifecycle digests, GET/HEAD counts, zero mutation/submit/account/session/runtime facts, and a false executor-invoked flag. Iteration 1 found coarse producer diagnostics; `057203c9dd911701cb521095f65d8c32e7505993` and `dc8f1f33a372bb89de2c2c43944dc774bc611ec3` expose only fixed producer/controller stages. Iteration 2 found a P2 consumer mismatch where Browsertools-valid observation-only empty labels caused whole-observation rejection; `03974fc63d53f1c41a0696ccce6a8f990cdbdd16` admits protocol-valid non-promotable labels while explicitly blocking empty/redacted/untrusted review selection. Iteration 3 reached package assessment and found a secret-scanner-shaped long symbolic binding; `ab26da298cfda0df3b43c1b45743e541ad41e3fe` uses the established short symbolic form. Iteration 4 passes the clean real `browser-transaction-brp` gate. Focused/race/vet/full tests, both real BAP+BCP and BRP Chromium gates, and diff checks pass; bounded review finds no unresolved P1/P2 issue. No account or registration runtime support was created. |
| E10.4 Adversarial, concurrency, and rollback matrix | `[+]` | Complete at OpenUdon `73a55e56b4d216cd1d941b28fc20616252813e3c`. The deterministic `browser-transaction-adversarial` target composes exact OpenUdon and published Browsertools tests for malicious/oversized/ambiguous protocols, unsafe labels/paths/origins, private-result and review drift, stale virtual sources and generations, digest/dependency drift, symlink/executable replacement, worker failure/cancellation, optimistic revision races, concurrent promotion, every durable promotion fault boundary, partial filesystem rollback, indeterminate inspection/reconciliation, secret/PII scanning, registration fail-before-executor, and UI/terminal conflicts. It verifies fixed diagnostics, cleanup, exact selected/prior/target preservation, and no blind retry. The complete target passes, and race tests pass for the stateful OpenUdon engine/package/writer/terminal/UI/browserauthor packages and Browsertools registration session/worker/result/candidate/capture packages. Review iteration 1 finds no P1/P2 issue; the initial Browsertools race invocation used the OpenUdon module root and failed setup without running tests, then the exact same command passed from the required Browsertools root. |
| E10.5 Full release gates and bounded review | `[+]` | Complete at local, unpublished OpenUdon `bb69c5a530eac303646e49a37455ce1bf19b3f57`. OpenUdon `b14bd5060666519402e23d13ff79be187ba2cc23` adds the consolidated `browser-transaction-eval --out` / `make browser-transaction-qualification` path: the adversarial target, real BAP+BCP replay, real zero-submit BRP lifecycle, closed report construction, exact five-worktree pre/post checks, read-only published Browsertools/UWS resolution, bounded discarded child output, atomic report/sidecar write, and independent verification. The clean `xvfb-run -a make browser-transaction-qualification` gate passes all 18 closed gates and independently verifies its report. The complete SaaS release gate passes all 23 loopbacks, all 8 journeys, sandbox-required UI Chromium, integration, docs, scorecard, validation, and dry-run checks; full race, cold-cache standalone build/test/vet, six-platform cross-build, pinned vulnerability scan, and pinned dead-code scan also pass. Review iteration 1 found a P2 exact-identity gap: the initial runner revalidated only the three report repositories and accepted a published-head line by prefix; the fix validates clean exact Udon/Browserdriver locks before and after execution and requires the complete Browsertools/UWS `ls-remote` line. Review iteration 2 found two P2 launch-identity gaps: explicit `--repo-root` left default sibling paths relative to the caller working directory, and local-unpublished OpenUdon was only checked for inequality with a potentially stale tracking ref rather than proving `main` strictly ahead of the independently resolved origin; the fixes resolve siblings relative to the explicit root and require local `main` strictly ahead of the independently resolved matching origin. Review iteration 3 found the required pinned dead-code gate was not silent; OpenUdon `f7adf828ec8b348505eef6324df05164165a8b19` removes three unused E10 report utilities and five older unreachable internal iCoT/registration wrappers, after which focused/full/race/vet/standalone tests and the pinned analyzer pass silently and the clean 18-gate qualification passes again. Review iteration 4 found one P2 operational-diagnostic defect; OpenUdon `5d60db72bac545d06ac69b25147cf32bc80f229f` preserves only the typed fixed sandbox-prerequisite cause through the consolidated runner while keeping all other backend detail closed, and focused/full/race/vet/dead-code/standalone plus the clean 18-gate qualification pass. Review iteration 5 found one P2 retained-provenance gap; OpenUdon `168cbfa40d14a67aa4c773a162908552b58b84db` requires all nine lifecycle digests to be distinct within each retained case and adds the recomputed-sidecar tamper regression, after which focused/full/race/vet/dead-code and the clean 18-gate qualification pass. Review iteration 6 found one P2 local command-boundary defect; OpenUdon `1f9517860456848b001b01bca2d617960f3ac304` passes the configurable Browsertools path only through a quoted child environment field and adds source/environment regressions, after which focused/full/race/vet/dead-code/adversarial plus the clean 18-gate qualification pass. Review iteration 7 found a broader P2 environment-authority gap: inherited Make control files/flags or Go command/flags could dry-run, replace, or filter adversarial tests despite the safe path; OpenUdon `2770f868db508ad82cca042fec18163d8e15aa1e` invokes resolved tools, closes Make/Go control variables, fixes `GOENV=off`, and shields Git identity commands from replace/config/worktree environment overrides. Review iteration 8 found one final P2 duplicate-environment ambiguity because the intended resolved-tool `PATH` was appended without first removing the inherited entry; OpenUdon `bb69c5a530eac303646e49a37455ce1bf19b3f57` emits exactly one controlled path and adds its regression. The affected focused/full/vet/dead-code and clean sandboxed 18-gate qualification pass. Review iteration 9 covers the complete E10 delta, failure semantics, evidence retention, exact dependencies, filesystem lifecycle, browser/network posture, and CLI boundary with no unresolved P1/P2 or higher-severity issue. Browsertools A08 remains independently resolvable at its exact published commit, OpenUdon remains local and unpushed, and evolution result v31 is recorded after offline W01 passed. |

Dependencies: OpenUdon M77, A19, P05, A20; published Browsertools A08
`39e32c1d6f601561cc5c13ec85201815ce85ab9b`; published UWS `895aa45`.
Public-target access and live registration,
sign-in, or campaign actions remain outside this qualification.

Post-completion publication reconciliation: after W01 closed and the user
explicitly authorized ordered pushes, OpenUdon E10 commit
`bb69c5a530eac303646e49a37455ce1bf19b3f57` was published. Follow-up
`42767a160dc88ac18ac5d624ad9e17151ba50d77` preserves v1 report compatibility
for both local-unpublished and published OpenUdon classifications while the
runner independently proves the exact state against `origin/main`. Complete Go
tests, vet, and diff checks passed before that follow-up was published. This
publication grants no browser, target, credential, account, registration,
workflow-execution, or deployment authority.

M77 reconciliation: qualification must use the public schema plus OpenUdon's
strict semantic decoder and transition validator, cover both published JSON
examples, and prove immutable identity/provenance across every lifecycle edge.
The report may retain transaction/package/workflow digests but never private
result locations or contents. UWS profile discriminators remain the published
BAP 1.0/1.1, BCP 1.5/1.6/1.7, and BRP 1.0 set; adversarial decoding includes
the v1 byte/depth/UTF-8/duplicate-name bounds. No UWS change is expected.

Browsertools A08 completion reconciliation: the BRP qualification matrix must
strict-decode registration session/result v1, prove accepted-observation time
and current-generation submit binding, exercise cancel/blocked-input and
context-bounded teardown, verify anchored owner-only no-replace persistence,
and confirm only fixed diagnostics and GET/HEAD accounting cross the boundary.
Use `v0.0.0-20260825225202-39e32c1d6f60` as the exact independently resolved
producer. The installed BRP loopback must preserve the validated sandbox-helper
boundary when required by the host, contact only the deterministic local
fixture, create no account, and retain no helper/private/result path.

A19 completion reconciliation: the adversarial matrix must reopen a selected
virtual source with the exact candidate and prove in-memory byte rehydration,
then reject missing, stale, or identity-changed rediscovery. It must prove
source removal/replacement invalidates the affected BAP, BCP, and BRP authoring
approvals, cancellation produces no candidate, partial/ambiguous private writes
remain unusable, workspace drift blocks mutation, and Browsertools
`39e32c1d6f601561cc5c13ec85201815ce85ab9b` plus UWS
`895aa4546067e25f9dd525b1356abf1945d223b4` resolve without replacement.

P05 completion reconciliation: qualification must exercise the exact
`packagepipeline` prepare/qualify/promote and selected approval/dry-run
adapters at OpenUdon `2121f06d6eab173012ab9d2a0f797ea45cece617`. Evidence
must bind preparation, qualification, generation, selection, package,
handoff, selected, and prior digests while excluding store/scratch/work paths.
The adversarial matrix must cover every durable intent, generation rename,
directory sync, selector rename/sync, intent cleanup, and lock cleanup
boundary; prove typed rollback/indeterminate outcomes and no blind retry; and
show digest-confirmed reconciliation preserves target, selected, and prior
generations without changing `current.json`. Legacy artifact promotion and
existing approval/handoff schemas must remain byte-compatible.

A20 completion reconciliation: qualify the one driver-free engine and its
API v4, accessible UI, and terminal adapters at local unpublished OpenUdon
`2b3687de2e9d1b38499e70137336a45d705e60a4`. Evidence must show identical
kind-specific BAP+BCP and BRP review facts, separate digest/revision-bound
review, prepare, promote, recover, cancel, and inspect operations, and closure
of every run, execute, sign-in, submit, credential, and account route. Recheck
the shared controller's canonical title, post-observation assessment time,
reviewed popup/frame context, exact typed Browsertools context encoding, and
fail-closed expected-negative causes. The release matrix must keep Browsertools
at `v0.0.0-20260825225202-39e32c1d6f60`, UWS at
`895aa4546067e25f9dd525b1356abf1945d223b4`, require sandboxed Chromium, and
retain no private result or package-working path.
~~~~~~~~~~~~~~~~~~~~
