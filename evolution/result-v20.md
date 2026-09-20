# Complementary Real-Browser Scenario Evaluation Result

OpenUdon `0ea9f3eff281cffbcd699b2b62905090b51d5e28` publishes
`openudon browser-scenario-eval` with strict embedded manifests, an exact
compatibility lock, `openudon.browser-scenario-eval.v1` reports, and SHA-256
sidecars. The required local suite passed all 21 real headed-Chromium cases
from Browsertools author-session/result v2 through OpenUdon profile
reconstruction, UWS 1.8/1.9 synthesis, Udon v3, and Browserdriver v3. The
separate explicit-network suite passed all four fixed anonymous read-only
public canaries through Browsertools value-free live checks and independent
credential-free Udon/Browserdriver v2 presence replay.

Post-publication review is closed by OpenUdon
`0a14a9fad7c86318f5d23028496963b4cfe01dcf` and Browserdriver
`0efd276ad77cae23dd1c3bd915ea107890ec7204`. Both Ubuntu workflows now
provision sandbox-compatible unprivileged user namespaces, strict scenario
JSON rejects duplicate keys recursively before decoding, and only literal
`presence: true` enters Boolean match mode. UWS browser 1.7 remains
byte-unchanged: its documented and schema-accepted `presence: false` form now
retains the declared runtime type.

| Layer | Published revision/version |
|---|---|
| UWS | `dd9eb32105131bdbc2855090ae0639b22d12de2b`; `v0.0.0-20260817013720-dd9eb3210513`; UWS 1.9/browser 1.7 |
| Browserdriver | `0efd276ad77cae23dd1c3bd915ea107890ec7204`; v3 navigation/click lifecycle, v2 browser 1.5 wait compatibility, and exact true-only presence mode |
| Udon | `2d2f4979d9a66f5bcb723ab4944e17f31dd1a8b8`; authentication-call 1.0/1.1 dispatch and v2/v3 replay |
| Browsertools | `4d940eaaae16390dd79b65f1829d66e198095d7d`; `v0.0.0-20260817224213-4d940eaaae16`; author-session/result v2 navigation settlement |
| OpenUdon | `0a14a9fad7c86318f5d23028496963b4cfe01dcf`; complementary evaluator plus hosted sandbox and strict duplicate-key hardening |

The loopback profile matrix covers authentication 1.1 plus browser 1.5, 1.6,
and 1.7 with the oldest sufficient UWS 1.8/1.9 contract and Browserdriver v3.
The public canary uses credential-free authentication 1.0, browser 1.5 Boolean
presence, UWS 1.7, and Browserdriver v2. Default Go/release checks remain
credential-free, provider-free, and browser-free except that the explicit
`release-saas-check` now includes the network-free loopback. Public targets
remain informational and require explicit `--allow-network`.
