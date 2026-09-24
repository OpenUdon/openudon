# M85 qualification target

M85 owns a versioned current-stack browser integration matrix. Its new report
contract checks the published M84 component commits, current module pins and
named UWS 1.11, Browser 1.8/1.9 and v10 tests. The prior scenario lock and v1
matrix reports retain their meaning. A passing `make browser-integration-check`
and digest verification at clean commits are required before M85 completes.

At clean OpenUdon `c60a2ccc2ee84553db56f7fee5be4c7cfea737d1`, the complete
provider-free v2 matrix passes 16 required gates, fails none and skips three
unrequested browser opt-ins. The written report verifies at SHA-256
`97fc8f429b4fc281417c55883afd1a6d11488d62743ff2e3adeef0436fd8b25b`.
The first clean attempt exposed a stale dependency assertion: UI qualification
uses Playwright explicitly while the iCoT engine remains implementation-free.
The current gate now checks those boundaries separately. The failed attempt is
preserved as diagnostic evidence, and review iteration 1 has no open P1/P2.
