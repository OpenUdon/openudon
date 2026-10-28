# P16 execution report pairing

OpenUdon's v3/v4 execution-report reader accepts the optional typed
`credential_sources` array as symbolic observation only. Each entry has exactly
`binding` and `source` with nonempty strings. The member is not accepted on
legacy v2; it does not grant approval or execution authority.

The P16 pairing used Udon source `81e62149d48ce17ee153d931af27143f0f30d304`
(the subsequent `e11f007` commit changed planning files only) and OpenUdon
consumer `25b00f8` plus its P16 public-surface fixture. A Go test overlay called
Udon's actual `writeExecutionReport` with an environment resolver holding a
synthetic value. It wrote v3/v4 success and `driver_error` reports. An isolated
consumer called OpenUdon's `Decode` and `FailureCode` on each original output;
all four passed, preserving the symbolic binding/source and failure code.
No live execution or publication is established.

The private workspace evidence directory is
`/home/peter/Workspace/.p16-local-pins/pairing`. Retain its overlay source,
consumer source and original reports; their SHA-256 hashes are:

- `producer_test.go`: `8cf693d6944aec9016e60f45d17c064d463e11fc271941f9781eab628e61970a`
- `consumer/main.go`: `b3abe74cc6d83fd981d10914ee25da3caa4e45243e65c0a1e3860a2c814f9f0f`
- `v3-success.json`: `963f76e8ab224f7c3846f2d52185fce5ac61ff67d0028882651a4dbd69bd0fdb`
- `v3-error.json`: `22818e1d627a23c64619262195f173412d650c104deccd200cf993a1ae1940f3`
- `v4-success.json`: `5dfbaf2846a8798af6bdfb6e3069fc13d02bfdf716fe9f806e95dea8e36d3bfc`
- `v4-error.json`: `443d7db62268de2d756c9e426e5ffed4774fc65e1274bb21590ff7404f4b98a0`

The exact UWS and Browsertools modules were reconstructed offline from local
Git commits and matched Udon's recorded `go.sum` module and go.mod hashes. The
producer test used only local module-cache archives and workspace-contained
scratch. No repository dependency pin was changed.
