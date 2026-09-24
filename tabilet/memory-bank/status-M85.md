# M85 — Current-stack browser integration qualification

**Goal.** Make `make browser-integration-check` qualify the published UWS
1.11 stack adopted in M84 and produce a verifiable provider-free matrix report.

**Provenance.** The user requested full OpenUdon integration matrix
qualification after the M84 handoff. Baseline OpenUdon `81cf0cdabf45db9204460ecb7cb3eeac2c8ed72a`
and the exact published siblings in M84 were clean. The existing v1 evaluator
uses `internal/browserscenario/compatibility-lock.json`, which pins older
revisions and stops before its gates run. The historical lock and reports must
remain verifiable.

**Scope.** The default provider-free matrix and its report verifier. Keep
installed-browser/headed opt-ins, W13/W16 scenario locks, live targets,
credentials and runtime adoption outside this milestone. No source or target
checkout may be silently rewritten to satisfy a lock.

| Item | State | Notes |
|---|---|---|
| M85.1 | `[+]` | Added an embedded current component lock, v2 report contract, named UWS 1.11/Browser 1.8/1.9/v10 gates and original v1 verifier dispatch. Focused owner commands pass in all five repositories; OpenUdon full tests/vet, standalone evaluator test/build, doc-memory and diff checks pass. The complete matrix run remains M85.2. |
| M85.2 | `[+]` | Clean OpenUdon `c60a2ccc2ee84553db56f7fee5be4c7cfea737d1` with the four exact published siblings passed `make browser-integration-check`: 16 required gates passed, 0 failed, 3 explicit browser opt-ins skipped. The v2 report and sidecar verify at SHA-256 `97fc8f429b4fc281417c55883afd1a6d11488d62743ff2e3adeef0436fd8b25b`. Full OpenUdon tests/vet, standalone evaluator test/build, CLI/boundary/example checks and doc-memory pass. Bounded review iteration 1 found no open P1/P2. |

**Dependencies.** M84 complete and published Browsertools `9333a9f25dbb17551998a429e123e7a9ba976648`,
Browserdriver `8c13b70d30a500e65e90a95a203493301b8b21a5`, Udon
`080b8282e2b8f7ca7a9994b6d9f0e3d2891d853f`, and UWS
`e9b6181be0abb7f683fdb624d4dba282a59991d1`.

**Review gate.** Complete at iteration 1 with no open P1/P2. The v2 report
selects the published M84 stack and checks named UWS 1.11, Browser 1.8/1.9 and
v10 evidence, exact module pins, clean revisions and value-free report details.
The v1 verifier still selects the original lock and gate inventory. The
passing candidate report is preserved under ignored
`eval/runs/browser-integration-m85-candidate/`; its digest sidecar matches the
recorded SHA-256. The first failed attempt remains diagnostic only.

**First clean matrix attempt.** OpenUdon `e6fcc1b87c5f461f30100d9fd033be4e413b1065`
ran all 18 v2 gates. Fourteen passed, three opt-ins were skipped and the
dependency gate failed because `internal/icot/ui` already includes a
registration-only Playwright qualification adapter. The engine graph remains
free of Browsertools capture and Playwright. The failed report and sidecar are
preserved under ignored `eval/runs/browser-integration-m85-attempt1/`; they
are not qualification.
M85.2 now makes the boundary precise: the engine remains implementation-free,
and the UI has no Browsertools capture implementation dependency. The adapter
still requires explicit registration qualification to launch a browser.
