# Ramen Safety

Ramen follows the udon execution boundary:

```text
AI may generate workflow artifacts.
AI may not directly execute operational actions.
```

## Rules

- Treat generated UWS, OpenAPI, and HCL files as untrusted until validated and reviewed.
- Keep production credentials outside agent prompts and generated artifacts.
- Keep LLM provider credentials in environment variables; do not pass tokens inline in commands that
  may be captured in shell history or issue logs.
- Use UWS/OpenAPI validation before any runtime execution.
- Execute side-effectful workflows only through a trusted runner with approved credentials.
- Prefer sandbox or test endpoints for local proof runs.
- Record validation evidence in the Symphony-managed work item before handoff.
- Treat Ramen output as Symphony state `generated`; no approval is implied by generation.
- Require `approved_for_sandbox` before a side-effectful proof run and `approved_for_production`
  before production execution.

## Quality Gates

Ramen fails `side_effects.policy` when generated artifacts imply writes, customer communications,
command execution, SSH execution, or other side effects without approval/trusted-runtime and
sandbox proof-run policy.

Ramen fails `side_effects.environment` when an explicit production endpoint is used without
production handoff approval language. Use sandbox, staging, localhost, `.test`, or documented
example endpoints for proof runs.

Ramen fails `review.approval_states`, `review.sandbox_handoff`, or `review.credential_bindings`
when review evidence lacks the Symphony approval-state requirements, sandbox/proof-run handoff
scope, or a credential-binding inventory. The inventory must list binding names only or explicitly
state that no credential bindings are declared or required.

## Runtime Profiles

Extension-owned UWS operations, such as SMTP, SSH, SQL, command execution, or LLM calls, must name
an implementation profile with `x-uws-operation-profile`. Ramen project policy decides which
profiles are allowed for a given environment.

Symphony approval states, trusted-runner handoff package contents, and future CI/secrets
automation prerequisites are defined in `docs/cross-repo-contracts.md`.
