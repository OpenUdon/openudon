# Retired milestone M32 - M32 - Fnct Helper Contract Authoring

**Milestone.** M32
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M32.md
**Source status SHA-256.** 8a8a4143e11b71a0c787c22b71cbb49a9d172e876c57067ca266f15316f9cc2a
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
# M32 - Fnct Helper Contract Authoring

## Goal

Teach OpenUdon to author public pure `fnct` helper selectors without taking
ownership of helper execution. Gmail raw-message rendering is the first helper
case.

## Status

| Item | State | Notes |
|---|---|---|
| Helper descriptor consumption | `[+]` | Synthesis imports `apitools/helper` metadata to identify request-body-object helpers. |
| Gmail render authoring | `[+]` | iCoT weather/Gmail finalization keeps `render_weather_report` as the local step and selects `gmail.render_raw`. |
| Request-body invocation | `[+]` | Known helper functions use operation request body fields and omit `x-uws-runtime.arguments`. |
| Runtime expression commitment | `[+]` | Request values that reference inputs or upstream step outputs are emitted as explicit `$expr` wrappers instead of ambiguous literal strings. |
| Recipient input policy | `[+]` | Weather/Gmail authoring declares `recipient_email` rather than hardcoding a personal address. |
| Example regeneration | `[+]` | `examples/weather-toronto-gmail` now uses `gmail.render_raw`, maps Gmail `raw` from `received_body`, prefers OpenWeatherMap Current Weather for simple current-weather goals, and passes `openudon build`. |
| Regression coverage | `[+]` | Added focused tests for helper request-body UWS generation, `$expr` quality evidence, fnct operation validation, runtime execution, and iCoT weather/Gmail artifact rendering. |

## Boundary Notes

- OpenUdon does not execute helper functions or import `../udon`.
- `github.com/OpenUdon/apitools/helper/...` owns public pure helper code and
  descriptors.
- `../udon` owns trusted helper registration and runtime execution.
- Gmail API send remains a side-effectful API-source step subject to approval
  and credential binding policy.
~~~~~~~~~~~~~~~~~~~~
