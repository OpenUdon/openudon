# E07 Authentication-Real Browser Evidence

Item | State | Notes
--- | --- | ---
E07.1 Stateful authentication fixture | `[+]` | Loopback goal pages require a random HttpOnly/SameSite session cookie; exact password and SMS/email/voice/TOTP/non-input MFA challenges are verified server-side.
E07.2 Replay success assertion | `[+]` | A scenario cannot pass unless the fixture records authenticated replay, so navigation-shaped bypasses and wrong or absent challenge values fail.
E07.3 Compatibility lock v2 | `[+]` | The lock pins the corrected Browsertools revision, exact Playwright package, and exact Chromium browser version, and rejects dirty or revision-mismatched pinned sibling worktrees.
E07.4 Shared integration policy | `[+]` | Browser integration evaluation imports the same repository-state validator and Playwright contract; scenario preparation owns the actual installed Playwright/Chromium launch comparison.
E07.5 Evidence hygiene | `[+]` | Structured failure classes replace broad context substrings, expected outputs derive from fixture facts, journey responses use bounded server headers/timeouts, and loopback evaluation does not inherit proxy variables.

`not_run` remains an inspectable non-pass status when readiness is not required;
release targets continue to require actual ready execution.

Implementation commit: OpenUdon `ad66c6a`.
