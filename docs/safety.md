# OpenUdon Safety

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

- Treat generated UWS, OpenAPI, and HCL files as untrusted until validated and reviewed.
- Keep production credentials outside agent prompts and generated artifacts.
- Keep LLM provider credentials in environment variables; do not pass tokens inline in commands that
  may be captured in shell history or issue logs.
- Use UWS/OpenAPI validation before any runtime execution.
- Execute side-effectful workflows only through a trusted runner with approved credentials.
- Prefer sandbox or test endpoints for local proof runs.
- Record validation evidence in the Symphony-managed work item before handoff.
- Treat OpenUdon output as Symphony state `generated`; no approval is implied by generation.
- Require `approved_for_sandbox` before a side-effectful proof run and `approved_for_production`
  before production execution.
- Treat `OPENUDON_EXECUTOR`, `OPENUDON_UDON_BIN`, `OPENUDON_UDON_RUNNER`, and `OPENUDON_UDON_IMAGE`
  as trusted operator inputs. Binary selectors must be absolute paths to reviewed executables;
  Docker image selectors must name an image the operator intentionally trusts.
- Keep local verification explicit: `go test ./...`, `go vet ./...`, `make check`, and
  `git diff --check`.

## Quality Gates

OpenUdon fails `side_effects.policy` when generated artifacts imply writes, customer communications,
command execution, SSH execution, or other side effects without approval/trusted-runtime and
sandbox proof-run policy.

OpenUdon fails `side_effects.environment` when an explicit production endpoint is used without
production handoff approval language. Use sandbox, staging, localhost, `.test`, or documented
example endpoints for proof runs.

OpenUdon fails `review.approval_states`, `review.sandbox_handoff`, or `review.credential_bindings`
when review evidence lacks the Symphony approval-state requirements, sandbox/proof-run handoff
scope, or a credential-binding inventory. The inventory must list binding names only or explicitly
state that no credential bindings are declared or required.

## Trusted Runner

`openudon run` writes a non-secret `openudon.executor-run.v1` config only after stored/current quality,
approval state, tier, credential policy, and package digest checks pass. New run configs include
`package_paths`, the sorted digest-covered handoff inventory. The executor shim stages those files
into a fresh workdir and recomputes `package_sha256` from the staged copy before invoking a binary
or Docker executor.

`OPENUDON_EXECUTOR`, `OPENUDON_UDON_BIN`, and `OPENUDON_UDON_RUNNER` are rejected unless they are
absolute paths to executable files. `OPENUDON_UDON_IMAGE` remains a Docker image reference and is not
interpreted as a filesystem path. Docker mode passes only declared `UDON_CREDENTIAL_*` environment
variable names into argv; non-Docker binary execution inherits the operator environment by design.

## Runtime Profiles

Extension-owned UWS operations, such as SMTP, SSH, SQL, command execution, or LLM calls, must name
an implementation profile with `x-uws-operation-profile`. OpenUdon project policy decides which
profiles are allowed for a given environment.

Symphony approval states, trusted-runner handoff package contents, and private checkout/secret
boundaries are summarized in `../memory-bank/architecture.md` and
`../memory-bank/tech-stack.md`.
