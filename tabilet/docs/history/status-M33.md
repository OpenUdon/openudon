# Retired milestone M33 - M33 - Runtime Data-File Handoff

**Milestone.** M33
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M33.md
**Source status SHA-256.** 25dbd8c88ea281dad5cc6406beb6d3d539451487c2b4d1619f8f4f1a2e0fb763
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
~~~~~~~~~~~~~~~~~~~~
