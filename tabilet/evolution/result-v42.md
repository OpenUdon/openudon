# M85 qualification target

M85 owns a versioned current-stack browser integration matrix. Its new report
contract checks the published M84 component commits, current module pins and
named UWS 1.11, Browser 1.8/1.9 and v10 tests. The prior scenario lock and v1
matrix reports retain their meaning. A passing `make browser-integration-check`
and digest verification at clean commits are required before M85 completes.
