# Status M93 — Supervised authenticated and registration browser capture

**State:** M93.0 private desktop confirmed; M93.1 protocol implementation selected. M92 is accepted/published; capture implementation/qualification remain incomplete.

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
| M93.0 — Prepare the private remote desktop (operation) | `[+]` | Operation row: run only while the launch request's named EXTERNAL_MUTATIONS authorization for the development desktop is in force. Check installation privilege first and stop if missing. Install and start Xvfb, a minimal window manager, x11vnc, the full noVNC viewer and websockify on the existing host; x11vnc and websockify listen on loopback only. The user connects once with noVNC in a local browser through an SSH tunnel to confirm the session. No public listener, firewall change or permanent service; record versions, display bindings, relay checks and teardown. Reused by M93.5, Kinet W09/U07 and W8M W28/W29. |
| M93.1 — Freeze capture event/decision protocol | `[~]` | Bound fields and event sizes; publish conformance fixtures and issued-reference/revision validation for both modes. |
| M93.2 — Authenticated and TOTP capture | `[ ]` | Preserve goal/dashboard/origin policy, MFA-kind selection, human credential entry, disclosure consent and exact action approval. |
| M93.3 — Registration and verification capture | `[ ]` | Retain registration authority, preview/navigation, verification approval, diagnostics and blocked-script rules; no production-registration authority is implied. |
| M93.4 — Embed worker and import reviewed profiles | `[ ]` | Reuse Browsertools worker entry and package lifecycle; classify submissions as write; preserve iCoT on the same implementation during 5A. |
| M93.5 — Qualify headless and visible sessions, review and publish | `[ ]` | Requires M93.0's prepared desktop; run both synthetic modes and human-visible evidence, owner qualification, review and publication. Do not defer this environment prerequisite to Kinet W09. |

## Acceptance and verification

Versioned conformance and headless loopback checks cover both capture modes, TOTP, verification refusal, stale decisions, expiry and teardown. Before visible qualification, operation row M93.0 prepares Xvfb, a minimal window manager, x11vnc, the full noVNC viewer and websockify on the development host under the launch reference's named authorization; x11vnc and websockify listen on loopback only and the user connects with noVNC in a local browser through an SSH tunnel. Record versions/display bindings and preserve Chromium sandboxing. Use disposable fixtures and bounded sessions; no public listener, service deployment or real target login. One explicit human-visible qualification covers both retained journeys. Qualify under owner policy; review and publish. A proven upstream protocol gap requires its owner's own approved plan, not copied code.

Default checks use fake providers, disposable roots and loopback fixtures. No live target operation or deployment is authorized. Preserve package instructions, one execution owner and exact upstream reconciliation before advancing. Task commits/publication follow only the separately launched goal's explicit policy; this planning approval performs neither.

## Provenance and lineage

Source: “Stage 5 draft reconciliation” (Kinet `stage5/REVIEW.md` findings F01–F10 and its accompanying drafts; refinement findings SR01–SR06; see the findings table in Kinet `docs/kinet-order.md` §6). Source priorities: not supplied. Draft baseline: e12a6488b86cafddb9298c7917de84fbc1cc85ff. Revalidated at this repository's full HEAD `e12a6488b86cafddb9298c7917de84fbc1cc85ff`. The draft source is read-only. Relevant uncommitted evidence: Kinet's `stage5/` drafts and APItools' approved M81/M80 planning changes; no implementation changes were used or made. User approved the complete dispositions and planning-file actions on 2026-09-30.

Supports F05 (primary migration owner W8M W28) and F06 (existing Browsertools worker, no new owner work). Evidence: internal/icot/browserauthor/, internal/icot/ui/, docs/authenticated-browser-authoring.md; Browsertools authorworker/worker.go and authorsession/session.go expose reduced observations and reviewed MFA kinds. Registration retention and remote desktop were explicitly selected during reconciliation.

Lineage: Preserve accepted browser-authoring/transaction gates and M91 inventory; cancelled W8M production registration remains cancelled.

## Closing review

Persisted iteration count: 0/10. Not started; this reconciliation is intake, not a closing-review iteration. Resume any interrupted future review at its persisted number. Acceptance, exact source/build revisions, publication and downstream reconciliation remain pending and must be recorded from observed evidence before normal package retirement.

## M91 exact producer reconciliation — 2026-10-01

Accepted extraction application source: `3fd40d3f874bdcf668a018550112a02cd0d02409`;
qualified review/source publication: `c8f2de71d983bea93dc1c045568c397a10a4eb56`.
Resolve the producer through OpenUdon's history index and its permanent M91
record; no producer ledger is merged here. Review 1 passed, 405 fixture bytes
and three UI assets stayed identical, integration v5 passed 16 required gates
with three unrequested optional gates, and native current-stack qualification
passed three fresh complete repeats (39 stages). Summary SHA-256:
`9a2524deddccc400457ae76d2e432a205898a181bcfe8bdc84f62348b4c28075`.

The single implementations now live in `internal/artifactwriter`, `elicitor`,
`browserauthor`, `browserauthoring`, `authoringengine`, `authoringui` and
`authoringcli`; Authoring is pinned to its published neutral `engine` source
`18056cb6b0c1007dd567a4a825a6b4311a357185`. `internal/icot` is a temporary
legacy forwarding adapter, and the old UI/control/terminal still works during
5A. All current public approval, credentials, cancellation, recovery and
v10/v11 browser dispatch boundaries remain unchanged. Historical report
selectors/locks and `.icot` package data stay frozen. These source facts satisfy
the extraction prerequisite only; every task in this consumer remains pending.

Capture adapters reuse neutral browserauthor/browserauthoring controllers and
existing Browsertools workers. The temporary M91 Xvfb session was automatically
torn down and grants no M93.0 completion or visible demonstration evidence.
Perform M93.0 under its own named operation authority and retain both journey
checkpoints. The runtime sandbox and private human credential path stay closed.

## M92 exact producer reconciliation — 2026-10-01

Final qualified application source: `96c16acacc7f442858dac8a0fcb36c84991ebddf`;
source/qualification publication independently verified at `bbb03effbe4215cf15473c4dfec561b64b80a122`.
Review 4/10 passed with no open findings. Frozen final producer bundle:
`/var/tmp/openudon-m92-corrected-producer-z89k3b36`, summary SHA-256 `a87277a65341e17b3f2e40daf275197cc02ff4377d160fdd0ba383fb7684ec30`;
CLI SHA-256 `dd109478d24321733fc63b20c163340e4786727fe18f39146c69be714d8edbef`. Published UWS source is
`a7688f54c68f5a75c7cc95aa2b31cea98b31af41` (1.12); exact M45 executor source,
binary and fourteen-source closure remain accepted: source
`238f2e487d50ffec057b7a109a35c9db03f59c55`, executor SHA-256
`cb4b94c968aa3f3de4106a440fdcd02e6c210941eb25e6666b84cfbc7f63868b`,
closure SHA-256
`10d4c613c4882365f2799789d456e8a3b15484b1995a052e334616cad9d0fd59`.
Final frozen full checks and integration v6 passed seventeen required gates
(fourteen named producer tests), zero failed, three optional unrequested.
Fresh native v5 passed39/39 at `7efb58678954a037a54e5d5874020258ce98cdca`.
Its evidence retains that actual source; the final delta changes only pure
simulation inventory, its test/marker, current documents and a new example's
formatting/digest, with browser/runtime/authoring/pin scope proven identical.
Both reports independently verify; temporary display teardown is verified.
Never relabel native evidence or treat a same-version binary as adoption.

Additive contracts and local conformance fixtures are
`openudon.step-pending.v1`, `openudon.simulate-input.v1`, and
`openudon.simulate.v1`. The legacy step-authoring v1 remains unchanged. New
packages default to UWS1.12, existing declarations remain unchanged. Pending
contracts refuse every approval/dry/real path across both artifacts and all
branches/workflows before credential/executor dispatch. Simulation is pure
public mock orchestration and in-memory projection, with no network, browser
worker, credentials or executor. Browser results are mocked contracts, not
page verification; previews grant no action authority. Every task in this
consumer remains pending; resolve producer closure through OpenUdon's history
index after normal retirement, preserving each package's own ledger.

Reuse neutral browserauthor/browserauthoring/authoringui controllers and current
Browsertools worker interfaces; preserve both capture modes and exact-origin,
issued-decision/revision/deadline/private-input rules. Native selectors are now
v5 and integration v6 with separate retained v4/v5 readers. M93.0 must prepare
its own private desktop and obtain the user's Remote Desktop Manager
connection confirmation; M92's disposable Xvfb is already torn down and proves
neither M93.0 nor M93.5 human acceptance. Capture submissions retain write
effects; any pending resolution binds the exact contract/current revision.

## M93.0 selected — 2026-10-01

M92 normal closure is independently verified published at
`f33d41f7202c00b986b640dee17861a39c79c027`; resolve its permanent history
record, not a missing active status. Qualified application is96c16acacc7f442858dac8a0fcb36c84991ebddf.
This is the sole general in-progress row across the goal's package ledgers.
The approved Kinet launcher explicitly authorizes installing/running Xvfb,
a minimal window manager and x11vnc on vps-f7dfc687.vps.ovh.us, with VNC
loopback-only and SSH transport, no public listener/firewall/permanent service.
Check actual installation privilege before installation and stop if missing.
Preserve private X authentication, private VNC credentials and disposable
settings. Record exact installed versions and display/session bindings. The
user must connect once with Remote Desktop Manager and confirm the display
before this operation completes or M93.1 begins. Synthetic M92 Xvfb evidence
is already torn down and is not this human checkpoint's acceptance.

## M93.0 first startup attempt — retained diagnostic 2026-10-01

Installed the authorized missing Openbox/x11vnc packages after successful
noninteractive installation privilege and repository metadata checks; existing
Xvfb/xauth remain installed. The first private desktop attempt created its
listener, but the helper's short RFB-banner socket probe timed out during the
VNC server's connection sniffing. That is not a human connection result.
Attempt `/var/tmp/openudon-m93-desktop-xqnd4u4y` records failure and teardown:
all owned children exited and its private authentication/password directory
was removed. Never reuse that failed attempt as acceptance. Correct the helper
readiness check to verify the exact x11vnc PID owns the loopback TCP listener;
start a new disposable session. The required human RDM check remains pending.

## M93.0 desktop ready; human checkpoint pending — 2026-10-01

Privilege check `sudo -n -v` passed. Authorized apt metadata refresh and
`apt-get install --no-install-recommends openbox x11vnc` completed with exit 0;
logs `/var/tmp/openudon-m93-desktop-apt-update.log` and
`/var/tmp/openudon-m93-desktop-apt-install.log`. Installed versions:
Openbox 3.6.1-12ubuntu3, x11vnc 0.9.17-2, Xvfb 2:21.1.22-1ubuntu1,
xauth 1:1.1.2-1.1build1. No permanent service, firewall or public listener
was configured. New private session `/var/tmp/openudon-m93-desktop-uyyn13g5` uses display `:98`,
1280x800x24, TCP disabled for X and exact VNC listener 127.0.0.1:5901.
Owned listener PID and absence of X TCP listener were independently checked.
RFB 3.8 probe offered only password authentication (type 2), no unauthenticated
access; this is a local readiness check, not the human confirmation.

Root/private directories are 0700, Xauthority/password files 0600 and owned by
peter. Password values are absent from commands, environment, repo and audit;
retrieve the private password only in the user's SSH terminal. Clipboard
exchange, x11vnc remote-control and external-command hooks are disabled;
existing user x11vnc config is bypassed. No provider secrets enter child env.
A standalone user supervisor expires this disposable session at
`2026-10-01T04:00:24.673997+00:00` and removes private credentials after stopping its owned
children. No system service was installed. Session metadata/expiry and local
RFB evidence remain in that private root. The prior failed attempt is retained
separately with verified teardown; it was not retried as historical evidence.

Human connection instructions: on the user's workstation, forward local
15901 to this host's 127.0.0.1:5901 with SSH. In Remote Desktop Manager choose
VNC, host 127.0.0.1, port 15901 and the private VNC password. The display contains
an OpenUdon M93.0 connection-check message. The user must confirm that message
is visible before M93.0 can complete. Until then this row remains in progress,
M93.1–M93.5 remain pending, and the ordered goal waits at its explicit checkpoint.

## M93.0 approved browser transport update — 2026-10-01

The user approved the complete browser-viewer proposal and explicitly resumed
the existing Stage 5 goal. M93.0 remains the sole in-progress operation; no new
ID, reordered dependency, reset review count or human acceptance is implied.
The operator uses noVNC in a normal browser through an SSH-forwarded loopback
websockify relay. Install the full distro viewer/relay on the named development
host, preserve private X authentication and password-required VNC, disable
clipboard exchange and automatically stop owned processes/remove private auth
when the bounded session expires. No public listener, firewall change, permanent
service, real target login or model disclosure is authorized by this update.

The earlier desktop at `/var/tmp/openudon-m93-desktop-uyyn13g5` expired at
2026-10-01T04:00:24.673997+00:00 without human confirmation. Its metadata records
no remaining children and removal of private auth. Preserve both earlier attempt
records; their Remote Desktop Manager instructions are historical, not current
connection instructions. Restore only this currently selected authorized row
with a new disposable session. Check viewer serving, WebSocket/RFB transport,
password-only authentication, mapped desktop, exact loopback listeners, secret
handling and teardown; then obtain the user's visible-message confirmation
before marking M93.0 complete or starting M93.1. M93.5's two visible journeys
and later consumer checkpoints remain required.

## M93.0 full browser viewer ready; human confirmation pending — 2026-10-01

Approved distro installation completed successfully under the verified
`sudo -n` privilege: noVNC `1:1.6.0-2`, websockify and python3-websockify
`0.13.0+dfsg1-2ubuntu1`. Installation log:
`/var/tmp/openudon-m93-browser-apt-install.log`. Kinet's untracked root npm
files and node_modules were preserved and are not product dependencies.

Fresh session `/var/tmp/openudon-m93-browser-n5sop97f` uses X display `:98` with X TCP disabled,
VNC `127.0.0.1:5901` and browser relay `127.0.0.1:6080`. Created at
`2026-10-01T06:58:47.310535+00:00`; expires at `2026-10-01T07:58:47.310535+00:00`. Supervisor
`/var/tmp/openudon-m93-browser-supervisor.py` SHA-256
`b83e0558cf6c8d460ba1b8bb542fe894c1d75fb9f946d2dca8adc2a14c8f2253` owns the bounded lifecycle. It uses no permanent
service, disables directory listing/clipboard/remote commands and receives
no provider credentials. Private directories are 0700 and auth files 0600;
password values never enter command arguments, environment, logs, audit or
repository. The exact owned listener PIDs, X TCP refusal and mapped message
were checked independently. Earlier expired attempts remain untouched.

A separate short-lived test at `/var/tmp/openudon-m93-browser-zoe4eri5`
expired automatically, removed private auth and left no running owned children
or desktop/relay listeners. Its session JSON SHA-256 is
`9f1f5bff911f8cf549054fa36a8ec87a62a67b411f265d0a12ae779c2639e2cf`.
Fresh sandboxed Chromium/Playwright readiness took 6.230 seconds: full viewer
loaded, password prompt required, wrong password refused, correct password
rendered a 1280x800 desktop canvas, and the test client disconnected. It used
only loopback with external requests blocked, no capture target or model.
Browser readiness SHA-256 `86706f33de4c17c8cb4d5464c33a91169f6f850aa7eb7976898767ef3058dc6b`;
security readiness SHA-256 `e8804b0e03734b8aa82761b90c7510302d4d6b819eb7bf738bd406c046c64ac7`.
These are automated connection checks, not the human checkpoint or M93.5's
journey qualification. The captured image caught the UI fade after connection;
it proves rendered pixels, not a separate user-visible acknowledgement.

Planning verification: Kinet `make check` and OpenUdon's tabilet-cwd
`check-doc-memory` passed; the evolution warning is expected because an
operator transport change does not meet the direction-change trigger. W8M's
browser-free structural checker, Go test and Go vet passed. Go operating
unit tests took 103.670 seconds; no native browser qualification was run.
Its `make fast` stopped at
the pre-existing absent exact Node 24.13.0 path; the separate Go test passed (`/tmp/w8m-m93-browser-plan-go-test.log`). No runtime qualification is inferred
from that partial gate, and no W8M scripts/locks/old records were altered.

Connection instructions: on the user's workstation run
`ssh -N -o ExitOnForwardFailure=yes -L 127.0.0.1:16080:127.0.0.1:6080 peter@vps-f7dfc687.vps.ovh.us`,
then open `http://127.0.0.1:16080/vnc.html?autoconnect=1&resize=scale`.
Privately retrieve the password with SSH from
`/var/tmp/openudon-m93-browser-n5sop97f/private/vnc-password` and enter it in noVNC; never put it in a URL
or chat. The user must confirm seeing "OpenUdon M93.0 connection check".
M93.0 remains in progress and M93.1–M93.5 remain pending until that evidence
arrives. Review count remains 0/10. The goal is open, not completed.

## M93.0 human connection accepted; M93.1 selected — 2026-10-01

The user reported that the browser view is good after receiving the SSH/noVNC
connection instructions for `/var/tmp/openudon-m93-browser-n5sop97f`. This
satisfies M93.0's required human desktop connection checkpoint alongside its
recorded readiness/security/automatic-teardown checks. The separate observation
record is `human-confirmation.json` in that disposable root. Earlier attempts
remain retained failures/expiry records. The active session keeps its original
bounded expiry; no public listener or permanent service is authorized.

M93.0 is complete; its operation is not replayed. M93.1 is the sole selected
general row. M93.2–M93.5 remain pending. This human connection does not qualify
authenticated/TOTP or registration capture, accept M93, or satisfy later M93.5
or Kinet U07 visible journeys. Review count remains 0/10.
