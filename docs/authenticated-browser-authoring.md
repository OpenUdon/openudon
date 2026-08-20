# Authenticated Goal-Directed Browser Authoring

`icot browser-author live` is the one explicit iCoT mode that launches an
external Browsertools process. It keeps a single headed Chromium context alive
while a human signs in and completes MFA, then guides bounded exploration in
that same context toward a reviewed goal. Normal iCoT, `--agent`, and
`icot browser-authoring plan` remain non-executing.

This closes the earlier authenticated-dashboard gap without moving browser
ownership into OpenUdon. Browsertools owns Playwright-Go and the live
non-persistent context. iCoT owns the goal interview, disclosure and approval
gates, strict protocol client, independent result validation, and atomic import
of canonical profiles.

```text
human operator
  -> iCoT typed goal, origins, continuation, and approvals
  -> Browsertools author-session process
  -> one headed Playwright-Go Chromium context
  -> private, deterministic authenticated-authoring envelope
  -> independent OpenUdon validation and final staging approval
  -> reviewed UWS authentication and capability profiles
  -> Udon runtime approval
  -> Browserdriver trusted replay in a separate runtime session
```

No cookie, storage state, credential value, OTP value, Playwright object, or
browser handle crosses a process boundary or becomes a UWS artifact. A live
authoring session cannot be resumed or exported.

## Operator Flow

Create an existing private directory with no group or other access, and use an
absolute, non-symlink Browsertools executable:

```bash
install -d -m 0700 /private/operator/member-authoring

icot browser-author live \
  --example ./examples/member-dashboard \
  --browsertools /absolute/path/browsertools \
  --url https://members.example.com/login \
  --dashboard-url https://members.example.com/dashboard \
  --goal "reach the member dashboard and learn how to read account status" \
  --origin https://members.example.com \
  --origin https://login.example-idp.com \
  --private-root /private/operator/member-authoring \
  --profile-id member \
  --goal-role heading \
  --goal-label Dashboard \
  --after-authentication ask_after_authentication
```

Use `--driver-dir` when the installed Playwright-Go driver is not available at
its normal location. Chromium must already be installed; the command never
downloads a driver or browser.

The after-authentication choice is typed and reviewed before launch:

- `continue_current_page` observes the successful redirect as it stands.
- `navigate_absolute` opens the exact reviewed `--dashboard-url`.
- `ask_after_authentication` asks the human to choose after sign-in.

The typed completion predicate is the exact dashboard origin and clean path,
context ID, accessibility role, and optional redacted-safe label. Browsertools
must observe exactly one match, and the human must separately type `confirm`.
Neither a model nor a redirect can declare completion.

## Human And Model Checkpoints

The command requires explicit review of the continuation, completion predicate,
and full origin allowlist. If local API discovery finds a plausible operation,
the human must type `use browser` to override API preference.

The human types credentials and OTP values directly into Chromium after
Browsertools focuses the reviewed field. At an MFA checkpoint, iCoT displays
only the compatible kinds reported by Browsertools and requires the human to
choose the exact exercised kind: `totp`, `sms_otp`, `email_otp`, `voice_otp`,
`push`, `push_number_match`, `passkey`, or `security_key`. The model cannot make
this choice. Browsertools does not read entered values, and iCoT passes a
minimal child environment that excludes credential and model-provider
variables. Clicks require approval. A previously unapproved
origin requires approval. Authentication clicks carry an explicit bounded POST
budget; Browsertools fails closed if the observed requests exceed it.

By default, iCoT may use its configured provider/model as a typed action
planner. Before the first page-derived observation reaches that planner, iCoT
names the provider and model and asks once for `disclose`. Choosing `human`
keeps the run human-guided. `--no-llm` prevents disclosure entirely. The model
sees only the reviewed goal and Browsertools' reduced observation:

- exact approved origin and clean path;
- page/frame context ID;
- Browsertools-issued candidate ID;
- portable accessibility role, redacted accessible label, and match count;
- closed diagnostic identifiers.

It never sees a DOM, full accessibility snapshot, screenshot, page text, form
value, cookie, storage, header, request body, selector, coordinate, JavaScript,
or Playwright object. Its only valid proposals are focus, candidate click,
bounded GET navigation, or human fallback. iCoT rejects invented candidate IDs,
unknown fields, trailing data, selectors, and malformed actions.

Phrase screening and fixed label markers are defense in depth, not the primary
LLM containment boundary. Every planner response must still decode as one
closed typed action: an observed unique candidate ID, a bounded same-origin GET
navigation in a disclosed context, or human fallback. The planner cannot
invent a selector, script, coordinate, URL authority, input value, completion
claim, or unapproved click, and human approval remains mandatory for clicks.

When the completion predicate matches, the human may review zero through 16
outputs from that final observation with repeated commands of the form
`output CANDIDATE_ID KEY TYPE LOCATOR_MODE`, followed by `done`. Types are
`string`, `integer`, `number`, `boolean`, and `presence`; locator modes are
`exact_name` and `unique_role`. iCoT rejects stale, ambiguous, form-control,
marker-labeled, duplicate, secret-shaped, and malformed selections, prints a
value-free sorted summary, and requires a final `confirm`. Output selection is
never delegated to the planner.

`--yes` is accepted for CLI compatibility but cannot bypass typed-goal review,
API override, disclosure, origin, action, authentication, completion, or final
staging approval.

## Local Protocol And Artifacts

iCoT starts:

```bash
browsertools author-session chromium \
  --private-root /private/operator/member-authoring
```

The child speaks newline-delimited `browsertools.author-session.v2` JSON on
standard input/output. iCoT accepts only the closed `hello`, `state`,
`observation`, `approval_required`, `human_checkpoint`, `diagnostic`, and
`result` messages, with bounded lines, candidates, diagnostics, and values.
Client actions reference Browsertools-issued candidate IDs; CSS, XPath,
coordinates, raw scripts, and browser objects are not protocol fields. iCoT
does not persist the protocol transcript.

Version 2 requires the `reviewed_mfa_kind` and `reviewed_outputs`
capabilities, negotiates `maxOutputs: 16`, and uses the distinct
`human_input_complete` message for reviewed credential/MFA completion.
Protocol v1 is rejected; there is no compatibility fallback at this live
boundary.

Unknown or oversized messages, timeouts, browser failure, unexpected
navigation, origin escape, ambiguous targets, denied approvals, malformed
results, CAPTCHA, or process failure close the context and produce no imported
artifact. Browsertools writes a mode-`0600`
`browsertools.authenticated-authoring.v2` envelope under the private root only
after typed and human completion. The envelope remains private and is never
copied into the example or package.

OpenUdon reopens the result as a stable non-symlink file, checks the protocol
digest, strict-decodes and bounds every field, validates the origin/context
graph, trace, goal proof, human confirmation, profile-review digests, freshness,
secret absence, reviewed challenge kinds, credential-slot kinds, resolved
value-free output proofs, profile versions, and both embedded UWS schemas.
OpenUdon reconstructs the expected profiles from the reviewed v2 facts and
rejects a substituted profile even when its digest was also replaced. After the operator types
`stage`, one atomic new-only transaction adds:

```text
browser-authentication/<id>-auth.json
browser-profiles/<id>.json
.icot/browser-authentication.json
.icot/browser-sources.json
.icot/authenticated-browser-authoring.json
```

The `.icot/authenticated-browser-authoring.json` file contains only safe,
value-free review facts and the private envelope digest. Run normal iCoT next
to select the exact authentication flow and capability action, bind symbolic
runtime credential slots, review API preference again, and author the workflow.
Existing browser sources or conflicting targets fail closed instead of being
silently overwritten.

## Context And Version Selection

The imported profiles may use the additive UWS 1.8 and 1.9 contracts:

- `uws.browser.1.6`
- `uws.browser.1.7`
- `uws.browser-authentication.1.1`
- `uws.browser-authentication-call.1.1`

They add portable popup/frame context graphs and authentication success paths.
`main` remains implicit. Frames use an exact path or frame name and must resolve
uniquely. A popup must be opened by one approved click that names
`opensContext`. Origins must be allowlisted, graphs must be acyclic, and their
depth cannot exceed four.

Existing `uws.browser.1.5`, `uws.browser-authentication.1.0`, and call 1.0
profiles remain accepted. Synthesis emits the oldest sufficient workflow
version: old main-page profiles continue to produce UWS 1.7, while a browser
1.6 or authentication 1.1 source selects UWS 1.8 and call 1.1 where required.
Browser 1.7 selects UWS 1.9. It adds locale-free accessibility-text conversion
for integer, number, and Boolean outputs; string text remains trimmed text and
presence remains Boolean matching. Noncanonical or out-of-range values fail
closed during trusted replay.

## Authoring Is Not Runtime Replay

Playwright-Go is an authoring dependency owned by Browsertools. It is used to
learn a portable, reviewed description while a human is present. It is not the
trusted workflow executor.

Later, Udon independently loads the packaged profiles, requests exact runtime
approval, resolves credential slots only inside the trusted execution boundary,
brokers MFA challenges, and invokes Browserdriver. Browserdriver creates a new
session and replays the portable workflow; it never receives the authoring
session, its cookies, or its browser handles. Runtime approval is separate from
every authoring approval.

Approved browser packages execute through the same digest-bound OpenUdon gate
as API packages. Supply the reviewed Browserdriver executable explicitly and
place only the derived symbolic credential variables in the operator
environment:

```bash
export OPENUDON_EXECUTOR=/absolute/path/udon
export UDON_CREDENTIAL_MEMBER_PASSWORD='<runtime value>'

openudon run \
  --example ./examples/member-dashboard \
  --tier sandbox \
  --approval ./approvals/member-dashboard.json \
  --browser-driver /absolute/path/browserdriver
```

OpenUdon derives the browser protocol, canonical credential/session mappings,
and exact operation and authentication approvals from the current reviewed
intent, profiles, and review files. Those value-free fields are bound into the
v2 run config and evidence, then re-derived by an external `udon-runner` before
execution. Browser driver arguments are optional repeatable
`--browser-driver-arg` values and must not contain credentials. Only declared
`UDON_CREDENTIAL_*` values, required external `UDON_BROWSER_SESSION_*` values,
and the documented minimal Browserdriver launcher environment cross the
executor boundary. With `OPENUDON_EXECUTOR=docker://<image>`, the supplied
host driver is validated, mounted read-only at `/openudon/browser-driver`, and
translated in Udon's argv. A package may retain reviewed browser profiles as
an API-first fallback without activating this runtime contract when its plan
contains no browser step.

## Exclusions And Failure Posture

Initial live authoring is Chromium-only. CAPTCHA, enrollment, recovery,
password changes, consent, account creation, logout, downloads, uploads,
permission grants, arbitrary scripting, and unattended credential entry are
unsupported. Popup support is limited to one explicitly opened exact-origin
page per declared context. Frames require portable exact metadata. Real-site
qualification must be operator-authorized against a non-production tenant;
retain only value-free evidence and never commit credentials, session state,
private envelopes, or page captures.

The browser-free contract matrix is described in [Browser Integration
Evaluation](browser-integration-eval.md). The real deterministic loopback,
realistic local journey, and opt-in public canary suites are described in
[Browser Scenario Evaluation](browser-scenario-eval.md). For both the
offline reviewed-artifact path and live orchestration boundary, see
Browsertools' [canonical OpenUdon integration
reference](https://github.com/OpenUdon/browsertools/blob/main/docs/openudon-integration.md).
The detailed producer-side protocol and responsibility matrix are in
[Authenticated goal-directed browser
authoring](https://github.com/OpenUdon/browsertools/blob/main/docs/authenticated-goal-authoring.md).
