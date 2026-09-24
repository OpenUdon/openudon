# Retired milestone A12 - A12 Authoring, Provider, And UI Safety

**Milestone.** A12
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A12.md
**Source status SHA-256.** c8f7c086a9ada0d2de681d6de28c5b8bfab244f10df4fa22866fe34d3ae9e117
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# A12 Authoring, Provider, And UI Safety

| Item | State | Notes |
| --- | --- | --- |
| A12.1 Shared credential-value policy | `[+]` | Artifact and LLM mapping scans share symbolic-reference grammar and Google, Slack, GitHub, AWS, bearer/JWT, dash-separated, and entropy-backed secret detection. |
| A12.2 Provider transport hardening | `[+]` | Gemini uses `x-goog-api-key`; provider bodies are bounded to 8 MiB, caller deadlines survive, and transport failures redact credentials. |
| A12.3 Tokenless browser bootstrap | `[+]` | The opened URL contains no secret. A terminal-only 12-character Crockford code is five-minute, single-use, and limited to five failures per minute before installing the scoped cookie. |
| A12.4 DNS-aware remote-source policy | `[+]` | Redirects and all DNS answers are checked, the selected IP is dialed directly, and unenforceable custom transports fail closed. |
| A12.5 Canonical source directories and clone safety | `[+]` | CLI, engine/UI, discovery, and seed copying consume one API/browser/authentication/capability directory inventory; draft operations and nested draft events clone independently. |
| A12.6 Stable security alternatives and strict HCL | `[+]` | Selections persist by canonical SHA-256 fingerprint; ambiguous legacy indexes require reselection. Unknown attributes/blocks fail and compatibility diagnostics map to original lines. |

OpenUdon remains single-workspace and single-operator.
~~~~~~~~~~~~~~~~~~~~
