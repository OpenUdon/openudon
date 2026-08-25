# Safety And Trusted Execution

OpenUdon follows the udon execution boundary:

```text
AI may generate workflow artifacts.
AI may not directly execute operational actions.
```

`openudon synthesize`, `openudon build`, `openudon promote`, and `openudon assess` are supervised artifact
generation and validation commands. They may compile, export, review, and assess artifacts, but they
must not perform production side effects. `openudon run` is a separate trusted-runner wrapper for an
already generated handoff package; it requires quality gates, approval JSON, package digest, and
tier checks before invoking udon.

## Rules

- Treat generated UWS, API source, browser-profile, and HCL files as untrusted until validated and reviewed.
- Keep production credentials outside agent prompts and generated artifacts.
- Keep LLM provider credentials in environment variables; do not pass tokens inline in commands that
  may be captured in shell history or issue logs.
- Use `OPENUDON_LLM_PROVIDER` and `OPENUDON_LLM_MODEL` for local LLM selection defaults; keep API
  keys in provider-native variables such as `COPILOT_API_KEY`, `OPENAI_API_KEY`,
  `ANTHROPIC_API_KEY`, or `GEMINI_API_KEY`.
- Gemini credentials are sent only in `x-goog-api-key`, never in a query
  string. Every provider success and error body is bounded to 8 MiB, caller
  deadlines are preserved, and transport errors are redacted.
- Remote source fetches validate every redirect, resolve with the caller's
  context, reject mixed or unsafe DNS answers, and dial a validated IP without
  a second lookup. Custom transports that cannot enforce this policy fail
  closed.
- Use UWS/OpenAPI validation before any runtime execution.
- For browser workflows, require an active non-expired `uws.browser.1.5`, 1.6,
  or 1.7 profile, matching `.icot/browser-sources.json`
  digest/action/origin evidence, and exact per-operation approval for every
  mutation. Login state must come from an earlier reviewed authentication step
  using the same named session or a symbolic external `browser_session` on
  each affected step. The aggregate opaque-session posture is review evidence,
  not executable session authority. Never package cookies, passwords, driver
  configuration, raw DOM/HTML, screenshots, or private cache content.
- For browser registration, require an active
  `uws.browser-registration.1.0` profile, its exact promotable
  `browsertools.registration-review.v1` bundle, and matching
  `.icot/browser-registration.json` review. Store only symbolic credential
  names. Account identifiers, verification values, cookies, browser storage,
  captures, and page content never enter the package. Duplicate prevention is
  operator attestation, duplicates fail, ambiguous post-submit outcomes stop
  without retry, and cleanup is a separately approved action.
- During authenticated authoring, bundled and expert workers use the same
  typed controller. It validates reduced event fields and disclosure paths
  before terminal, HTTP, or planner publication and requires a process-private
  attestation to bind the final trace, authentication proof, output requests,
  contexts, diagnostics, and approved-origin ledger before staging. That
  attestation and all credential/challenge values have no artifact or HTTP
  representation.
- Optional `browsertools.live-check.v1` and
  `browsertools.portability-check.v1` inputs must remain value-free, match the
  exact packaged profile/action set, and pass OpenUdon's independent shape,
  lifecycle, engine, and fixed-diagnostic checks. Only normalized summaries and
  source digests enter `.icot/browser-sources.json`; raw reports, rich evidence,
  backend errors, and local report paths are not staged. Portability is review
  confidence, not universal execution policy.
- Execute side-effectful workflows only through a trusted runner with approved credentials.
- Prefer sandbox or test endpoints for local proof runs.
- Record validation evidence in the review work item before handoff.
- Treat OpenUdon output as review state `generated`; no approval is implied by generation.
- Require `approved_for_sandbox` before a side-effectful proof run and `approved_for_production`
  before production execution.
- Treat `OPENUDON_EXECUTOR` and `OPENUDON_UDON_RUNNER` as trusted operator inputs.
  `OPENUDON_EXECUTOR` must be an absolute path to a reviewed executable or `docker://<trusted-image>`.
- Keep local verification explicit: `go test ./...`, `go vet ./...`, `make check`, and
  `git diff --check`.

Credential scanning is shared across package artifacts and LLM request
mappings. Only documented `inputs.`, `credentials.`, and prior-step reference
forms are symbolic; Google refresh/client secrets, Slack/GitHub/AWS tokens,
bearer/JWT shapes, dash-separated secrets, and high-entropy literals fail
closed. Required credential mappings also reject the reserved `none` and
`clear` sentinels as slot or binding names; rejection occurs before session or
draft mutation.

## Quality Gates

OpenUdon fails `side_effects.policy` when generated artifacts imply writes, customer communications,
command execution, SSH execution, or other side effects without approval/trusted-runtime and
sandbox proof-run policy.

OpenUdon fails `side_effects.environment` when a production-or-unknown endpoint is used without
production handoff approval language. Every external HTTP(S) endpoint outside
loopback and reserved `.test`/example domains is production-or-unknown; a host
name containing `sandbox` or `staging` is not approval evidence.

OpenUdon fails `review.approval_states`, `review.sandbox_handoff`, or `review.credential_bindings`
when review evidence lacks the review approval-state requirements, sandbox/proof-run handoff
scope, or a credential-binding inventory. The inventory must list binding names only or explicitly
state that no credential bindings are declared or required.

## Trusted Runner

`openudon run` writes a non-secret `openudon.executor-run.v2` config only after stored/current quality,
approval state, tier, credential policy, and package digest checks pass. New run configs include
`package_paths`, the sorted digest-covered handoff inventory, plus exact
approval and handoff digests. Validation reads each manifest-required input
once, verifies its declared digest, and derives the handoff, package, quality,
intent, browser profile, review, credential, session, approval, and protocol
facts from that immutable byte snapshot. Every invocation receives a unique run ID and
directory. Dry runs and real handoffs stage those files into a fresh workdir
and recompute `package_sha256` from current source files, so drift after the
snapshot is rejected before execution. Dry runs stop there and
write `openudon.run-evidence.v2` without requiring credential values. Non-dry
runs require the declared `UDON_CREDENTIAL_*` values, verify the bounded
regular executor report by path, digest, and size, then write the same
non-secret evidence shape.

For a reviewed browser workflow, `openudon run` additionally requires an
explicit absolute `--browser-driver` path. It derives a value-free browser
contract from the current plan, intent, packaged browser/authentication
profiles, and review metadata: protocol, canonical credential/session
environment mappings, and exact operation/authentication approvals. The
external runner re-derives that contract and rejects substitutions. Driver
arguments, config, evidence, and argv may not contain credential values. A
Docker executor receives the validated host driver through a read-only bind
mount at `/openudon/browser-driver`; only credentials and session bindings are
forwarded with Docker `-e`. Approved driver environment names such as `PATH`,
`HOME`, locale, temporary-directory, and `PLAYWRIGHT_BROWSERS_PATH` use the
container's own values. Host desktop, display, and socket requirements are
rejected. Unreferenced API-first fallback profiles are ignored by runtime
configuration.

Registration packages use the same immutable approval and dry-run staging
path, including canonical `UDON_CREDENTIAL_*` names and exact registration
operation approvals. Dry-run never resolves those values or launches a
browser. Non-dry registration is rejected before executor construction because
the currently pinned Udon and Browserdriver contracts do not implement
`uws.browser-registration-call.1.0`.

`OPENUDON_EXECUTOR` is the canonical final executor selector. It accepts an absolute binary path or
`docker://<image>`. `OPENUDON_UDON_RUNNER` is separate: it overrides the outer runner shim and must
be an absolute path to an executable file. Execution uses a typed
argv/directory/environment invocation. Local binaries receive only declared
credentials. Docker and outer runners additionally receive the documented
minimal platform launcher environment; the outer runner also receives explicit
`OPENUDON_UDON_BIN` and `OPENUDON_UDON_IMAGE` overrides. Cloud, proxy,
SSH-agent, and unrelated process variables are not forwarded.

On Linux, bounded subprocesses track descendant PID/start-time identities via
`/proc`, sweep detached `setpgid`/`setsid` children after normal leader exit as
well as cancellation, and verify termination without killing a reused PID.
Other Unix platforms retain process-group cleanup and Windows retains task-tree
cleanup; those fallbacks cannot provide Linux's detached-descendant identity
proof.

When `OPENUDON_UDON_RUNNER` is used, OpenUdon run evidence records `stage_kind: preflight` because
the outer wrapper can validate and stage the package before handoff but cannot observe the external
runner's final staging. It receives `--config`, `--config-sha256`, and
`--approval`, then revalidates current quality, handoff, package, approval,
tier, exact config bytes, credential bindings, staging, and digests. v1
configs cannot execute.

Run evidence may be signed with an optional PKCS#8 PEM Ed25519 private key.
The key is never placed in config, evidence, argv, or child environments.
Verification with the embedded PKIX public key proves integrity; pass
`--trusted-public-key` to additionally pin operator identity.

## Runtime Profiles

Extension-owned UWS operations, such as SMTP, SSH, SQL, command execution, or LLM calls, must name
an implementation profile with `x-uws-operation-profile`. OpenUdon project policy decides which
profiles are allowed for a given environment.

Review approval states, trusted-runner handoff package contents, and optional sibling checkout plus
secret boundaries are summarized in this guide, [Review Handoff](review-handoff.md), and the
[project authoring documentation](project-authoring.md).
