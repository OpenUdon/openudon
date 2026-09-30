# Status M93 — Supervised authenticated and registration browser capture

**State:** Approved planning, 2026-09-30; every task pending. No implementation or publication is established by this record.

**Goal.** Expose both existing browser-capture journeys to Kinet through a bounded non-interactive protocol.

**Dependencies.** M92 accepted/published; M91 retained-journey inventory. Existing Browsertools authorworker/authorsession and registration protocols; no new Browsertools work is presumed.

**Downstream.** M94; Kinet W09/M19/U07; W8M W28/W29.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled. (OpenUdon's convention; W8M's differs.)

## Scope and contract

Publish openudon.browser-capture.v1 events and decisions for state, reduced observation, issued action approvals, human sign-in/MFA-kind checkpoints, preview, diagnostic and result. Bind decisions to issued IDs and revisions. Cover authenticated goal/dashboard/origin capture including TOTP, and registration-authority binding, verification approvals, preview/navigation, diagnostic and blocked-script policies. Preserve exact origins, action approvals, deadlines, POST limits, cancellation/teardown and explicit model-disclosure consent; human-guided is the default. Embed the existing Browsertools worker under openudon and import only reviewed profiles using package transactions. Credentials/codes stay in the private browser input path, never application protocol payloads or ordinary logs; this does not prohibit the human's protected desktop input transport. Keep iCoT on the shared implementation until M95.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| M93.0 — Prepare the private remote desktop (operation) | `[ ]` | Operation row: run only while the launch request's named EXTERNAL_MUTATIONS authorization for the development desktop is in force. Check installation privilege first and stop if missing. Install and start Xvfb, a minimal window manager and x11vnc on the existing host; x11vnc listens on loopback only. The user connects once with Remote Desktop Manager through an SSH tunnel to confirm the session. No public listener, firewall change or permanent service; record versions and display bindings. Reused by M93.5, Kinet W09/U07 and W8M W28/W29. |
| M93.1 — Freeze capture event/decision protocol | `[ ]` | Bound fields and event sizes; publish conformance fixtures and issued-reference/revision validation for both modes. |
| M93.2 — Authenticated and TOTP capture | `[ ]` | Preserve goal/dashboard/origin policy, MFA-kind selection, human credential entry, disclosure consent and exact action approval. |
| M93.3 — Registration and verification capture | `[ ]` | Retain registration authority, preview/navigation, verification approval, diagnostics and blocked-script rules; no production-registration authority is implied. |
| M93.4 — Embed worker and import reviewed profiles | `[ ]` | Reuse Browsertools worker entry and package lifecycle; classify submissions as write; preserve iCoT on the same implementation during 5A. |
| M93.5 — Qualify headless and visible sessions, review and publish | `[ ]` | Requires M93.0's prepared desktop; run both synthetic modes and human-visible evidence, owner qualification, review and publication. Do not defer this environment prerequisite to Kinet W09. |

## Acceptance and verification

Versioned conformance and headless loopback checks cover both capture modes, TOTP, verification refusal, stale decisions, expiry and teardown. Before visible qualification, operation row M93.0 prepares Xvfb, a minimal window manager and x11vnc on the development host under the launch reference's named authorization; x11vnc listens on loopback only and the user connects with Remote Desktop Manager through an SSH tunnel. Record versions/display bindings and preserve Chromium sandboxing. Use disposable fixtures and bounded sessions; no public listener, service deployment or real target login. One explicit human-visible qualification covers both retained journeys. Qualify under owner policy; review and publish. A proven upstream protocol gap requires its owner's own approved plan, not copied code.

Default checks use fake providers, disposable roots and loopback fixtures. No live target operation or deployment is authorized. Preserve package instructions, one execution owner and exact upstream reconciliation before advancing. Task commits/publication follow only the separately launched goal's explicit policy; this planning approval performs neither.

## Provenance and lineage

Source: “Stage 5 draft reconciliation” (Kinet `stage5/REVIEW.md` findings F01–F10 and its accompanying drafts; refinement findings SR01–SR06; see the findings table in Kinet `docs/kinet-order.md` §6). Source priorities: not supplied. Draft baseline: e12a6488b86cafddb9298c7917de84fbc1cc85ff. Revalidated at this repository's full HEAD `e12a6488b86cafddb9298c7917de84fbc1cc85ff`. The draft source is read-only. Relevant uncommitted evidence: Kinet's `stage5/` drafts and APItools' approved M81/M80 planning changes; no implementation changes were used or made. User approved the complete dispositions and planning-file actions on 2026-09-30.

Supports F05 (primary migration owner W8M W28) and F06 (existing Browsertools worker, no new owner work). Evidence: internal/icot/browserauthor/, internal/icot/ui/, docs/authenticated-browser-authoring.md; Browsertools authorworker/worker.go and authorsession/session.go expose reduced observations and reviewed MFA kinds. Registration retention and remote desktop were explicitly selected during reconciliation.

Lineage: Preserve accepted browser-authoring/transaction gates and M91 inventory; cancelled W8M production registration remains cancelled.

## Closing review

Persisted iteration count: 0/10. Not started; this reconciliation is intake, not a closing-review iteration. Resume any interrupted future review at its persisted number. Acceptance, exact source/build revisions, publication and downstream reconciliation remain pending and must be recorded from observed evidence before normal package retirement.
