# XRD-008 Runtime/Profile Eval Coverage

This plan keeps XRD-008 in Ramen's integration layer. Ramen owns project policy, curated eval
fixtures, and compatibility evidence. Runtime/profile semantics and generic execution remain
upstream in `../udon` or `../uws`.

## Coverage Matrix

| Category | Fixtures | Boundary proven |
| --- | --- | --- |
| Approved function runtime | `runtime-only-render`, `support-priority-routing`, `profile-boundary-manifest` | Trusted local adapters and renderers can be generated when declared in project policy and Function Contracts. |
| Approved command runtime | `cmd-allowed-deploy` | `cmd` is allowed only when project policy explicitly permits a sandbox command. |
| Denied command runtime | `cmd-disallowed-deploy` | A generated `cmd` step remains a policy failure when the project denies command execution. |
| Denied SSH/runtime profiles | `cmd-disallowed-deploy`, `profile-boundary-manifest` | Ramen policy keeps `ssh` and future profile runtimes out of generated intent unless an upstream public/runtime contract exists and project policy approves it. |
| Future profile boundary | `profile-boundary-manifest` | SQL-style profile work is represented as a trusted `fnct` manifest request instead of inventing `sql`, `ssh`, or `x-udon-*` runtime semantics in Ramen. |

## Acceptance Criteria

- Eval fixtures cover approved `fnct`, approved `cmd`, denied `cmd`, denied `ssh`, and future
  profile-boundary behavior.
- Ramen reference intents do not invent unsupported runtime types such as `sql`, `smtp`, `llm`, or
  profile-specific `x-udon-*` payloads.
- Any future fixture that needs real profile semantics opens upstream work in `../udon` or `../uws`
  before Ramen prompt defaults emit those fields.

## Boundary

Ramen may validate policy language, reference intent shape, and review evidence. It must not define
new public UWS fields, profile payloads, or runtime execution behavior in this pass.
