# Status M06 — iCoT Authoring

State of M06 items. See [milestone.md](milestone.md) for milestone scope and
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
| Guided authoring CLI implemented | `[+]` | `cmd/icot` guides operators from a broad idea to `project.md` and `workflows/intent.hcl`. |
| Optional LLM path implemented | `[+]` | LLM kickoff/refine/disambiguate roles are available while offline manual authoring remains supported. |
| Autosave implemented | `[+]` | Incomplete sessions are saved under ignored `.icot/session.yaml`. |
| Transcript support implemented | `[+]` | Optional transcripts are saved under ignored `.icot/transcript.json`. |
| Reconcile/lint/replay implemented | `[+]` | iCoT supports deterministic reconcile, lint, and replay workflows. |
| Atomic artifact writes implemented | `[+]` | Final artifact writes are treated as an atomic small transaction. |
| OpenAPI operation ranking improved | `[+]` | Operation context and ranking use narrowed `apitools` APIs. |
| Readiness and follow-up loop implemented | `[+]` | Readiness checks, grouped questions, confidence/evidence classification, and final edit/explain confirmation are implemented. |
