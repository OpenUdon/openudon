# Status M03 — Structured Output And Provider Drift

State of M03 items. See [milestone.md](milestone.md) for milestone scope and
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
| Provider-native structured output path implemented | `[+]` | Structured generation is used where supported. |
| Legacy extraction fallback preserved | `[+]` | Fallback extraction remains available and is counted rather than hidden. |
| Structured fallback reporting implemented | `[+]` | Eval evidence records fallback count and attempts-to-pass. |
| Provider failure reporting implemented | `[+]` | Reports include provider errors, model availability, and provider drift watch data. |
| Release comparison reporting implemented | `[+]` | Reports include comparison deltas and release-gate failures. |
| Real-provider automation policy documented | `[+]` | Real-provider evals remain local/manual until protected secret and redaction automation exists. |
