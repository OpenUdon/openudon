# Private capture rejection diagnostics

The source candidate writes `openudon.capture-diagnostic.v2` for the existing
opt-in `--capture-diagnostic` path. Its public application/capture frames are
unchanged. Exact retained older binaries continue to produce v1. A consuming
runner must select the matching strict reader before adopting new binaries.

V2 adds a private `rejection` object with closed `boundary`, `resource` and
`origin_relation` classifications from Browsertools. It is written only after
the worker and event delivery have joined. It never retains a URL, error message,
page text, account value, header, body, cookie or browser state. Resource type
does not establish a frame or worker identity, and policy rejection is not an
independent proof that no network request occurred.

The worker companion reader accepts separately validated worker v2 and historical
v1. A v1 origin rejection retains its original class and supplies explicitly
unknown detail. Missing/invalid worker evidence remains missing/invalid, with
no invented rejection. Invalid detailed metadata is reduced to an invalid
backend diagnostic, never copied into the application companion. Public event
JSON and the historical `BackendDiagnostic()` method stay unchanged; the
separate `BackendRejectionDiagnostic()` method is meaningful only after closure.

One diagnosed capture consumes the private companion. Existing files cannot be
overwritten; failed diagnostic writes remove a promotable result. Independent
process cleanup remains the runner's responsibility. This source change is not
publication, qualification, adoption or authority for another live capture.
