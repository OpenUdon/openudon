# P03 Trusted Browser Execution

Item | State | Notes
--- | --- | ---
P03.1 Package-derived browser run contract | `[+]` | The trusted runner derives Browserdriver protocol, driver launcher fields, canonical credential/session mappings, and exact operation/authentication approvals from current plan, intent, profiles, and strict review files.
P03.2 Browser credential inventory | `[+]` | Browser authentication bindings are merged into authoring credentials and expected-plan handoff inventory, so only their declared `UDON_CREDENTIAL_*` values can cross the executor boundary.
P03.3 Local and Docker invocation | `[+]` | Udon receives the complete browser CLI surface for local executables and Docker images; driver/session state uses a fixed launcher allowlist and proxy, cloud, SSH-agent, and unrelated variables remain excluded.
P03.4 External revalidation and evidence | `[+]` | Browser config is embedded value-free in v2 evidence, strict-validated during verification, and independently re-derived by an external runner so forged mappings, approvals, protocols, or arguments fail before execution.
P03.5 Adversarial qualification | `[+]` | Focused tests cover dry/real driver requirements, local/Docker argv and environments, declared credentials, external sessions, direct-runner config forgery, and unsafe persisted arguments.

OpenUdon invokes Udon and Browserdriver as external trusted components; it does
not import their runtime implementations or store credential/session values.

Implementation commit: OpenUdon `aabc408`.
