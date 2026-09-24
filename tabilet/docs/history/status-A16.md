# Retired milestone A16 - A16 Unified iCoT UI And Package Handoff

**Milestone.** A16
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A16.md
**Source status SHA-256.** 0ff17d48e108928b8cbe78b8598a4a9162f75e6c27cf6fc652e1d5eea9f37cda
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
# A16 Unified iCoT UI And Package Handoff

| Item | State | Notes |
| --- | --- | --- |
| A16.1 One-binary isolated worker | `[+]` | `icot` privately stabilizes and re-executes its own hidden Browsertools author-session/doctor worker; the engine and HTTP server never initialize Playwright in-process. |
| A16.2 Shared browser-author coordinator | `[+]` | UI capture and bundled terminal live mode share typed v2 orchestration, minimal child environment, process-group cancellation, stdout draining, descendant termination, the ten-minute browser-active bound, 30-minute operator-idle cancellation, and two-hour absolute ceiling; cancellation is monotonic, rejects queued late results, remains active until the worker event stream confirms teardown, and a containment timeout requires process restart. |
| A16.3 Experimental API v3 transport | `[+]` | v1/v2 routes are absent; tokenless bootstrap/access-code exchange, scoped cookie and bearer auth, exact Host/Origin policy, strict JSON/multipart bounds, polling, shutdown, and complete-state ETags are retained for `openudon.icot-ui-api.v3`. |
| A16.4 Revisioned capture lifecycle | `[+]` | Authoring and capture revisions are independent; preflight/start/stage require both where they cross authorities, every response atomically consumes one capture revision, dirty frontier answers block acquisition changes, one capture including its canceling/teardown interval blocks authoring/package mutation while snapshots remain readable, and terminal states are explicit. |
| A16.5 Reduced accessible shell | `[+]` | Starter/source explorers, typed doctor/capture controls, exact origin/action/POST approval cards, Chromium-only credential guidance, reported MFA kinds, bounded output review, provenance, symbolic maps, workflow graph, and recovery guidance are embedded without a client build system. |
| A16.6 Authored and package-failure lifecycle | `[+]` | Final authoring approval writes reviewed authoring artifacts and enters `authored`; a separately confirmed two-minute deterministic build plus non-writing current-byte assessment provides check-specific remediation and revision-protected resume with mandatory reapproval. |
| A16.7 Frozen handoff inspection | `[+]` | Passing bytes enter `handoff_ready` only when the complete manifest-bound generation is unchanged across current-state assessment; its policy facts and handoff hash come from the same stable read, and quality, package/handoff digests, symbolic credentials/sessions, approvals, separately rendered exact `openudon approval-template` argv, and a bounded closed artifact allowlist remain drift-checked. |
| A16.8 Authority exclusions | `[+]` | No registration, approval-generation, credential, run, or execution endpoint exists; passwords, challenges, cookies, storage, raw worker output, child stderr, private result paths, signing material, and runtime credentials have no HTTP representation. |
| A16.9 Coordinated dependency publication | `[+]` | Browsertools A06 commit `7ea7e832d7f85060cf57a7a08bf9af6bb7eca896` is published; OpenUdon pins `v0.0.0-20260821154836-7ea7e832d7f8`, and the early standalone iCoT build passes without a local `replace` directive. |
| A16.10 Parent-attested staging | `[+]` | Bundled, expert external, UI, and scenario staging use one typed controller whose process-private attestation binds ordered parent actions, exact checkpoints and approvals, observations, dashboard authentication proof, output requests, additive contexts, diagnostics, and the accumulated exact-origin ledger to the final envelope before any file is staged. |
| A16.11 Disclosure and lifecycle hardening | `[+]` | Browsertools-owned path validation runs before terminal/HTTP/model/result disclosure; closed state is terminal, terminal reads are context-cancelable through one input pump, and worker binaries use a byte-verified private content-addressed mode-0500 cache with narrowly owned stale-temp cleanup. |
| A16.12 Credential and review hardening | `[+]` | Credential binding answers are validated atomically before mutation/autosave, browser-specific draft review reports missing session/mutation authority without auto-repair, verification evidence is canonically ordered with retained evidence first, and HTTPS/loopback HTTP is enforced at OpenUdon URL/origin admission. |
| A16.13 Hardened dependency publication | `[+]` | Browsertools hardening revision `e392fd080fc47789ac64edd5fe7953a8cf104ee0` is published and OpenUdon pins pseudo-version `v0.0.0-20260822182551-e392fd080fc4`; a fresh proxy-only module cache resolves it, standalone build/test/vet pass, external-worker events regain strict pre-publication reduction/vocabulary/state validation, and reserved credential-clear sentinels fail atomically. |

Product implementation, dependency publication, coordinated-workspace
verification, and hardened standalone dependency closure are complete. E09's
prior release evidence is unchanged.
~~~~~~~~~~~~~~~~~~~~
