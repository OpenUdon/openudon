# Status M12 — Package Artifact And Local OpenAPI Safety Hardening

State of M12 items. See [milestone.md](milestone.md) for milestone scope and
acceptance criteria.

Status markers:

| Symbol | Suggested Status | Interpretation |
|---|---|---|
| `[ ]` | Pending | Item not started or pending action. |
| `[+]` | Completed | Item finished or done. |
| `[~]` | In Progress | Item is being worked on. |
| `[!]` | Blocked | Item requires attention or is on hold. |
| `[X]` | Cancelled | Item is no longer needed. |

| Item | State | Notes |
|---|---|---|
| Package root validation hardened | `[+]` | Required handoff inputs share safe relative path validation, package-root validation, and regular-file checks. |
| Required handoff inventory hardened | `[+]` | Manifest inventory checks reject unsafe, missing, symlinked, directory, special-file, and unstated required inputs. |
| Package digest inputs hardened | `[+]` | Digest input validation covers reviewed UWS artifacts and staged OpenAPI files. |
| Trusted-runner staging guards hardened | `[+]` | Staging rejects symlinks and verifies staged package digest before executor invocation. |
| apitools local OpenAPI reads hardened | `[+]` | Local OpenAPI reads reject symlinked roots/paths/parents, directories, special files, and oversized path-backed documents. |
| Shell/Python runner replaced | `[+]` | Go run-config parsing and staging live in `internal/udonrunner` and `cmd/udon-runner`. |
| Executor selector simplified | `[+]` | `OPENUDON_EXECUTOR` is canonical and accepts absolute binary paths or `docker://<image>`. |
| Direct runner fail-closed checks added | `[+]` | Direct `cmd/udon-runner` rejects missing package digest, missing package paths, and direct production run configs. |
| workflowintent package split | `[+]` | Intent/HCL, authoring adapter, chat adapter, provider client, OpenAPI, and helper files replace the monolith without changing the package API. |
