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
