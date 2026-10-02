# Browser system qualification

Use `make fast` for browser-free routine development. Deliberately select one
synthetic affected `make smoke`; its development v2 report never qualifies a
runtime. `make qualify` and `make browser-system-current-check` select fresh
current native qualification, not the development cache. No public target,
provider call, account operation, installation or publication is implied.

## Current owner qualification

```sh
openudon browser-system-eval --stack current --suite loopback \
  --repo-root /absolute/clean/openudon --udon-repo /absolute/prepared/udon \
  --browserdriver-node-modules /absolute/read-only/node_modules \
  --out /tmp/NEW-native.json
openudon browser-system-eval --verify /tmp/NEW-native.json
```

Native v6 runs thirteen stages three fresh times (39 total): capture protocol,
pure registration definitions, capture lifecycle, exact build inputs, Udon browser
contract and CLI, Browserdriver registration, loopback and journey scenarios,
BAP/BCP transaction, registration capture-to-runtime handoff and both actual
public registration/authenticated capture-to-package journeys. Authentication
retains TOTP; registration includes typed conditional history/previews and separate
verification refusal/approval, reviewed import and independent native selection.
BRP separately approved synthetic replay proves exactly one POST, with value-free
package/run reports. Kinet owns consumer UI conformance, not these native adapters.

Before launching, inspect cheap prerequisites, record the exact source/tool/input
selection, elapsed-cost estimate and authorized synthetic operation. Prepare clean
nineteen-source roots, including the fourteen-source M45 executor closure from
`internal/browserscenario/current-qualification-build-inputs-v5.json`. Current
compatibility is UWS1.12/M45 via `current-compatibility-lock-v5.json`; actual Udon
source is 238f2e487d50ffec057b7a109a35c9db03f59c55. Installed external Browserdriver
modules must match its package-lock and remain outside the clean checkout. Node,
Go, Playwright and Chromium versions/bytes are bound, not inferred from labels.
Native selectors recheck sources after every stage and use disposable outputs.
No missing prerequisite is downloaded, upgraded or bypassed.

A private temporary display is a separate explicitly authorized operation.
Retain Chromium sandboxing, closed child environments and joined PID/start-time
owners. Failure streams stay bounded in owner-only diagnostic sidecars; unknown
outcomes, cancelled workers or missing teardown are failures, not passing proof.
Component failures print only fixed qualification phase codes. Dynamic executor
errors, paths and private values are never printed. The synthetic private-input
fixture waits for the actual Apply checkpoint before filling runtime fields;
package startup does not substitute for readiness or extend workflow deadlines.

## Versioned readers

| Evidence | Fresh current | Retained readers |
| --- | --- | --- |
| Native system | v6 neutral thirteen-stage inventory | v1 eleven legacy stages; v2–v5 original UI thirteen-stage inventories and locks |
| Integration | v7 neutral dependency/authoring gates | v1–v6 original selectors and locks |
| Scenario/journey | v5 UWS1.12/M45 | Earlier original inventories/pins |
| Input identity | v3 actual source/tool/namespace closure | Historical v1 unchanged |
| Development | v2 current neutral adapter | No development report is runtime qualification |

Removed UI/control stages cannot freshly execute from current source.
Historical verification remains read-only and dispatches by saved report version;
it does not relabel old evidence, authorize a target or qualify new runtime bytes.
Historical M79/M86/E21/E22/M91/M92/M96 records retain the original chronology.

## Development and input-only checks

```sh
openudon browser-system-dev --mode smoke --stage registration_capture_handoff \
  --repo-root /absolute/openudon --udon-repo /absolute/prepared/udon \
  --browserdriver-node-modules /absolute/read-only/node_modules \
  --out /tmp/NEW-development.json --cache /absolute/private/cache
openudon browser-system-input --stack current \
  --repo-root /absolute/clean/openudon --udon-repo /absolute/prepared/udon \
  --browserdriver-node-modules /absolute/read-only/node_modules
```

Development permits dirty OpenUdon inputs while enforcing current sibling locks
and before/after byte identities. Only explicit `--reuse` accepts a matching
successful transaction smoke younger than24hours; execution identity/time are
preserved. Other stages use ordinary fresh test execution. Native three-repeat
qualification never consumes the development cache. Input-only inventory starts
no browser and establishes no qualification/cache authority. External W8M consumers
own complete cache binding/freshness, fresh journeys and final pin adoption.
A cache miss stops; it never automatically falls back to an expensive seed.

Registration foreground/countdown, prior-attempt attestation v1/v2/v3, input
privacy, one-attempt claims and runtime approval remain native contracts, unchanged
by transport removal. See [public capture](browser-capture-protocol.md),
[native package handoff](browser-package-handoff.md) and
[the retained capability migration](authoring-retirement.md).

Current native v6 also retains the four browser-free aggregate stages under
`--stack current --suite offline`: OpenUdon units, Browsertools units,
Browserdriver units and neutral capture lifecycle. `make qualify` runs/verifies
that current offline report before the three fresh loopback passes; old v1–v5
readers keep their original version/lock/suite contracts. No gate is dropped
because the former historical default no longer executes a removed UI.
