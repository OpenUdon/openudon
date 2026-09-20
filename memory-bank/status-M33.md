# M33 - Runtime Data-File Handoff

## Goal

Make declared workflow inputs executable through reviewed, non-secret package data instead of host
environment variables. The first concrete case is `expected/data.hcl` for the weather-to-Gmail
recipient address.

## Status

| Item | State | Notes |
|---|---|---|
| Data-file artifact generation | `[+]` | `openudon build` emits `expected/data.hcl` with an `inputs { ... }` block for declared runtime inputs and preserves existing declared values. |
| Env-marker preservation | `[+]` | `openudon build` now rewrites the generated `inputs` block while preserving other reviewed data-file blocks such as credential env references. |
| Google OAuth2 placeholders | `[+]` | Packages declaring `googleOAuth2` now get a generated non-secret `credentials { googleOAuth2 { ... } }` block pointing to `GOOGLE_*` env markers so Udon can mint an access token from reviewed data-file configuration. |
| Source security binding names | `[+]` | Build normalizes credential request mappings for API-source security fields to the exact source security scheme name, avoiding stale document-derived bindings that do not match executor credential resolution. |
| Executable input expressions | `[+]` | Generated UWS request/control expressions lower `inputs.<name>` to `variables.inputs.<name>` while intent, plan, and review evidence keep authoring-level `inputs.<name>` semantics. |
| Handoff packaging | `[+]` | Runtime data files are included in review handoff inputs, package digests, run config `data_files`, staging, and executor argv when present. |
| Udon runtime support | `[+]` | Udon execute mode accepts repeatable `--datafile` flags and exposes merged document variables as `variables` in runtime expression evaluation. |
| Regression coverage | `[+]` | Added focused tests for data-file generation/preservation, package inventory, run config data files, Udon CLI datafile forwarding, and `variables.inputs.*` expression evaluation. |

## Boundary Notes

- Runtime data files are non-secret reviewed inputs. Credentials still use `UDON_CREDENTIAL_*` or
  env-reference markers resolved by the trusted runtime; plaintext credential values should not be
  committed into package artifacts. The generated `googleOAuth2` block contains only env marker
  names, not OAuth client secrets or refresh tokens.
- Changing `expected/data.hcl` changes the approved package digest and requires a fresh approval.
- `../apitools` is not involved in this feature.
