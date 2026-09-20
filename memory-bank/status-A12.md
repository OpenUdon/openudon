# A12 Authoring, Provider, And UI Safety

Item | State | Notes
--- | --- | ---
A12.1 Shared credential-value policy | `[+]` | Artifact and LLM mapping scans share symbolic-reference grammar and Google, Slack, GitHub, AWS, bearer/JWT, dash-separated, and entropy-backed secret detection.
A12.2 Provider transport hardening | `[+]` | Gemini uses `x-goog-api-key`; provider bodies are bounded to 8 MiB, caller deadlines survive, and transport failures redact credentials.
A12.3 Tokenless browser bootstrap | `[+]` | The opened URL contains no secret. A terminal-only 12-character Crockford code is five-minute, single-use, and limited to five failures per minute before installing the scoped cookie.
A12.4 DNS-aware remote-source policy | `[+]` | Redirects and all DNS answers are checked, the selected IP is dialed directly, and unenforceable custom transports fail closed.
A12.5 Canonical source directories and clone safety | `[+]` | CLI, engine/UI, discovery, and seed copying consume one API/browser/authentication/capability directory inventory; draft operations and nested draft events clone independently.
A12.6 Stable security alternatives and strict HCL | `[+]` | Selections persist by canonical SHA-256 fingerprint; ambiguous legacy indexes require reselection. Unknown attributes/blocks fail and compatibility diagnostics map to original lines.

OpenUdon remains single-workspace and single-operator.
