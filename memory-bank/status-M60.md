# M60 Status - Executor-Report Archive Command

State markers and commit rules are defined in [milestone.md](milestone.md).

| Item | State | Notes |
|---|---|---|
| Archive helper | `[+]` | Added archive copying for run evidence, async sidecars, and executor report files when present. |
| CLI command | `[+]` | Added `openudon run-evidence archive --file ... --out ...`. |
| Verification | `[+]` | Archive helper verifies source and archived evidence; CLI smoke covers archive output. |
