# Retired milestone A20 - A20 Unified Browser Transaction Engine, UI, And Terminal UX

**Milestone.** A20
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A20.md
**Source status SHA-256.** 2c934448b5e4ff12c35bedb6ffd1944c2a67a99315dabb9d9bd76676800badec
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
# A20 Unified Browser Transaction Engine, UI, And Terminal UX

| Item | State | Notes |
| --- | --- | --- |
| A20.1 Shared transaction engine APIs | `[+]` | OpenUdon `d886bf671ede7337e22c8cba92d3c1fbfe0bc950` adds the driver-free `internal/browsertransaction/engine` lifecycle shared by future terminal and UI adapters. Its typed start/observe/review/prepare/promote/cancel/inspect/recover operations use digest revisions, exact transaction/evidence authority, separate human review/prepare/promotion/recovery approval, candidate expiry, defensive value-free snapshots, and closed frontend errors. The production adapter calls P05 prepare/qualify/promote/current/selected-inspection/recovery/reconcile entrypoints; snapshots retain preparation, qualification, generation, selection, baseline, selected, prior, safe target, and exact recovery digests while exposing no path, candidate body, worker output, credential value, browser, or runtime operation. BRP and BAP+BCP tests cover explicit empty binding serialization, denial, stale revision, concurrent one-winner mutation, cancellation, expiry, exact selected inspection, indeterminate promotion, blind-recovery rejection, defensive copies, and deterministic JSON. Focused/full dependent tests, focused race, and vet pass. Review iteration 1 corrected untagged package-digest adaptation and nil-versus-empty BAP+BCP binding cloning; iteration 2 restricted starts to candidate/reviewed transactions, distinguished invalid wire from digest drift, tightened recovery operation advertisement, and added nil-engine failures; iteration 3 found no P1/P2 issue. |
| A20.2 Experimental iCoT UI API v4 | `[+]` | OpenUdon `e171c2bc2f33a2f90838e99f6aa353b45989de0e` retires the complete `/api/v3` namespace, moves compatible authoring/capture/package resources to `openudon.icot-ui-api.v4`, and adds authenticated `browser-transactions` current/start/review/prepare/promote/cancel/recovery/selected-inspection resources backed only by the A20.1 typed interface. Transaction responses use their own revision/ETag and expose the exact engine snapshot plus a deterministic kind-specific review: BAP+BCP keeps only symbolic session/bindings, while BRP shows Browsertools' canonical heuristic/not-DLP disclosure, accepted-observation timestamps and expiry-recheck rule, GET/HEAD-only zero-mutation/no-submit/no-account/no-session/no-runtime posture, and explicitly inert approval symbol. Loopback Host, origin, access-token/cookie scope, content type, duplicate-name, UTF-8, 1 MiB body, 32-level depth, operation timeout, method, and route closure gates apply; no run/execute/submit route exists. Tests cover all typed operations, absent configuration, v3 and runtime-route closure, hostile origin, authorization, malformed/duplicate/deep/oversized input, value/path omission, symbolic serialization, stable disclosures, and typed conflict errors. Full Go tests, focused race, vet, and diff checks pass. Review iteration 1 replaced a time-varying freshness boolean with immutable times plus the engine recheck rule and mapped operational errors away from 422; iteration 2 found no P1/P2 issue. |
| A20.3 Accessible transaction review journey | `[+]` | OpenUdon `f0ca59750de782c453a0f99c65b5cd7767275648` adds a keyboard-first embedded transaction review that distinguishes BAP+BCP from BRP; presents value-free origins, times, digests, symbolic bindings/session, candidate outputs and registration cleanup, Browsertools' exact heuristic/not-DLP and GET/HEAD-only no-submit disclosure, preparation/qualification/promotion/recovery evidence, and downstream side-effect posture; and provides separate revision/digest-bound consent for review, scratch preparation, promotion, recovery, and cancellation without adding runtime controls. Registration virtual-candidate snapshots now expose only the validated cleanup disposition needed for exact review. Stateful local Chromium tests cover accessible labels/status/focus, disabled-without-consent denial, stale-revision refresh and re-consent, expiry, indeterminate promotion and exact recovery, selected-package inspection, BRP versus BAP+BCP rendering, 360 px reflow, reduced motion, and absence of execute/register/sign-in/submit actions. Static asset, full tagged Chromium, full Go, focused race, vet, syntax, and diff gates pass. Review iteration 1 corrected the preparation consent's inaccurate write-free wording and enumerated allowed actions/checkpoints/cleanup in exact review consent; iteration 2 found no P1/P2 issue. |
| A20.4 Terminal compatibility | `[+]` | OpenUdon `389fc7c29a5019a82c0cf8429a703db641b05bbd` adds `icot browser-transaction`, a no-runtime shared-engine adapter that reads one bounded public transaction artifact, emits the same value-free BRP/BAP+BCP resource as API v4 as NDJSON on stdout, and keeps exact digest-bound review, scratch preparation, promotion, cancellation, indeterminate recovery, and selected inspection as separate stdin authorizations with closed typed failures on stderr. Empty/wrong stdin fails without mutation, signals cancel blocked prompts and engine work with exit 130, and neither paths nor input lines are echoed. `icot ui` now accepts an all-or-none public transaction/package flag group and starts that same engine for the implemented browser journey; old terminal authoring and `openudon package` behavior remain unchanged. Tests cover shared presentation parity, scripted streams, existing-package prepare/promote/inspection, cancellation, expiry, recovery, noninteractive denial, path/value omission, successful help, UI launch wiring, and real SIGINT/SIGTERM subprocess containment. Full Go, focused race, vet, full local Chromium, and explicit trusted-runner BRP dry-run/live fail-before-executor gates pass. Review iteration 1 fixed the previously test-only UI engine wiring and failing subcommand help; iteration 2 found no P1/P2 issue. |
| A20.5 Documentation, qualification, and bounded review | `[+]` | OpenUdon `1d3bf9e5f7be45b483d43df77d8b40f5d840c549` reconciles public UI/terminal/browser-authoring/package documentation and the credential-free browser-scenario lock. Review iteration 1 fixed stale Browsertools/UWS release-lock revisions. Iteration 2 recorded the exact AppArmor user-namespace prerequisite after refusing an unsandboxed substitute. Iteration 3, after the sandbox became available, found P2 shared-controller defects in canonical title authority, post-observation assessment time, reviewed frame selection, Browsertools' typed context encoding, and expected-negative cause masking; OpenUdon `2b3687de2e9d1b38499e70137336a45d705e60a4` fixes them with focused regression coverage. Focused/full/race/standalone/vet/boundary, strict docs, doc-memory, diff, immutable cross-package integration, all 23 authenticated browser loopbacks, all 8 package journeys, sandbox-required UI Chromium, scorecard, validation, and dry-run release gates pass with no scenario skip or quarantine. Iteration 4 reviewed the complete A20 implementation and fixes and found no remaining P1/P2 issue. E10 and W01 are reconciled to the final local commit; OpenUdon remains unpushed and evolution result files remain deferred until W01 closes. |

Dependencies: OpenUdon M77, A19, P05, and published Browsertools
`39e32c1d6f601561cc5c13ec85201815ce85ab9b`. The
UI remains local and exposes no registration runtime or production target
authority.

M77 reconciliation: every adapter presents the exact candidate, reviewed,
prepared, promoted, cancelled, failed, and indeterminate states and the closed
failure class/code pairs from `openudon.browser-profile-transaction.v1`.
Snapshots may expose immutable digests, canonical origins/times, and symbolic
bindings/session only; the internal Go implementation remains unsupported as a
public library and API v4 must not broaden the JSON transaction wire.

Browsertools A08 completion reconciliation: UI and terminal adapters must show
the exported heuristic/not-DLP label disclosure wherever registration
candidates are displayed or retained, present accepted-observation freshness
and fixed no-submit/network facts from registration-authoring v1, and never
turn its inert `approvalSymbol` or reviewed submit description into execution
authority. The exact producer dependency is
`v0.0.0-20260825225202-39e32c1d6f60`; no UI/API surface may expose its private
root/result path, worker detail, or Linux sandbox-helper path.

A19 completion reconciliation: shared adapters must distinguish an exact
candidate resume, which rehydrates bytes in memory and preserves approvals,
from missing, stale, or identity-changed rediscovery, which fails closed.
Deselect/replacement must visibly invalidate only the affected BAP, BCP, or BRP
authoring approvals and require repair/reapproval. Registration cancellation
must close without a candidate or retained result, and snapshots must continue
to omit source/review bodies while showing only safe conflict facts.

P05 completion reconciliation: the shared engine must call the committed
`packagepipeline` prepare/qualify/promote/inspect/reconcile and exact-selector
approval/handoff adapters rather than reproduce their filesystem logic. It
must preserve preparation, qualification, generation, selection, baseline,
selected, and prior digests in value-free snapshots; require separate prepare
and promote authority; surface rolled-back versus indeterminate state plus the
safe target digest; block blind retry; and require exact recovery-report
acceptance. Existing artifact `openudon promote` semantics remain unchanged,
and UI/terminal workdirs, approvals, or runtime evidence may not enter the
immutable generation store. P05 is complete at OpenUdon
`2121f06d6eab173012ab9d2a0f797ea45cece617`.
~~~~~~~~~~~~~~~~~~~~
