# M74 Consolidation And v0.2 Closure

Item | State | Notes
--- | --- | ---
M74.1 Durable atomic evidence writes | `[+]` | Shared writes sync file data, rename atomically, sync parent directories where supported, clean failures, and cover replacement/rename faults.
M74.2 Reusable CLI policy extraction | `[+]` | Build-info parsing, public UWS validation, quality remediation, and catalog selection moved from `cmd/openudon` into focused internal packages.
M74.3 Shared evidence helpers | `[+]` | Bounded regular-file reads, duplicate/unknown/trailing-safe JSON, digest sidecars, and full 40/64-character Git IDs share one implementation.
M74.4 Dead-code and schema cleanup | `[+]` | Pinned `deadcode v0.47.0 -test ./...` reports no functions; unused UWS schema snapshots and obsolete non-atomic APIs were deleted.
M74.5 File responsibility split and regression tests | `[+]` | iCoT, progressive elicitation, and OpenAPI quality files are split below 1,000 lines; iCot reports, atomic files, endpoints, verbs, and evidence helpers have dedicated tests.
M74.6 Documentation and release-ready tree | `[+]` | README, compatibility, safety, handoff, memory, status, and evolution describe v0.2 without tagging or publishing it.
M74.7 Hosted sandboxed Phase C proof | `[~]` | Remains owned by A11.5. The 2026-08-20 local sandbox-enabled attempt failed because the host has no usable Chromium sandbox; no sandbox-disable override was used, and hosted release-runner proof remains required.
