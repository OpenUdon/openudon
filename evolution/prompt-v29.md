# Unified iCoT UI, Browser Capture, And Package Handoff

Make iCoT UI the primary interactive authoring experience for API and
existing-account authenticated-browser workflows. Distribute one `icot`
executable while preserving the browser boundary by privately re-executing it
as a separate Browsertools worker using author-session v2.

Promote engine-owned journey selection, bounded private API upload and atomic
staging/removal, multi-profile browser capture staging with a safe v3 review
collection, and explicit resume after package-quality failure. Replace the
experimental UI API v2 with v3, separate authoring and asynchronous capture
revisions, keep snapshot polling available, expose only reduced typed browser
observations and approvals, and enforce one active capture plus operator-idle
and absolute ceilings.

Final authoring approval must enter an authored state. A separate deterministic
build and non-writing exact-byte assessment either returns check-specific
remediation with mandatory reapproval or freezes a handoff-ready package with
bounded allowlisted artifacts, digests, symbolic bindings, runtime approvals,
and exact approval-template argv. Do not add registration, UI-owned LLM
drafting, approval generation, credentials, trusted execution, collaboration,
profile replacement, or remote browser agents.

Coordinate Browsertools A06, OpenUdon A15/A16, and E08 real-browser
qualification. Replace the obsolete binary-wide Playwright-free claim with the
narrow invariant that the iCoT engine and HTTP server never run Playwright
in-process.
