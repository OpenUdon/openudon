# M61 Status - Release-Note Generation

State markers and commit rules are defined in [milestone.md](milestone.md).

| Item | State | Notes |
|---|---|---|
| Draft helper | `[+]` | Added release-note draft generation from verified run evidence, current commit, gates, verifier output, and evidence paths. |
| CLI command | `[+]` | Added `openudon release-notes draft --run-evidence ... --out ...`. |
| Verification | `[+]` | CLI smoke confirms draft output includes gate and async sidecar evidence paths. |
