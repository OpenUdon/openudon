# A23 Status - Browsertools A10 Consumer Adoption

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Complete and published at OpenUdon
`dd7437c0149903ee7af987cd2e02380735ccbc40` after published Browsertools A10
`3107470313d447e29c5ac5912c3cc9d221d46967`
(`v0.0.0-20260829181035-3107470313d4`).

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A23.1 Pin and qualify published Browsertools A10 | `[+]` | OpenUdon updates only `go.mod`, `go.sum`, and the exact content-trust dependency expectation. The complete offline test suite, vet, `make check`, sibling/API boundary checks, and documentation-memory validation pass with the published module. The adoption is published at `dd7437c0149903ee7af987cd2e02380735ccbc40`. |

## Compatibility And Non-Goals

- The Browsertools public wire, UWS schema, Browserdriver behavior, and target
  allowlist are unchanged.
- OpenUdon gains no browser session, account, submit, execution, deployment,
  or public-target authority.
- The historical E12 tagged compatibility fixture remains locked to the exact
  M28 checkout it qualified; A23 updates the current module dependency and its
  ordinary exact-version assertion.

## Scoped Commits

- Browsertools A10 implementation and publication:
  `3107470313d447e29c5ac5912c3cc9d221d46967`.
- OpenUdon exact consumer adoption:
  `dd7437c0149903ee7af987cd2e02380735ccbc40`.

## Verification

- Pinned Go 1.26.6 `go test -count=1 ./...` and `go vet ./...` with
  `GOWORK=off`, `GOPROXY=off`, `GOSUMDB=off`, and `GOTOOLCHAIN=local`.
- Offline `make check`, including standalone iCoT build, full tests, sibling
  presence, and API-tools boundary validation.
- `openudon check`, `openudon check-doc-memory`, and `git diff --check`.
- Exact published Browsertools module checksum and dependency-lock assertion.
