# OpenUdon

OpenUdon provides public Go libraries for constructing, reviewing and verifying
UWS packages, exact approvals, executor handoffs and run evidence. Kinet owns
the authoring product and isolated workers; the private Udon runtime owns
execution. OpenUdon's libraries supply verified metadata and byte contracts.
The consuming host owns permission, credentials, session custody, runtime
admission and dispatch.

## Public packages

| Package | Purpose |
|---|---|
| `authoringcard` | Typed authoring drafts, explicit incomplete values, strict validation and deterministic review text; data alone grants no approval or execution authority. |
| `packagev3` | Construct and independently verify packages from reviewed UWS, source and shape bytes; inspect v2/v3 history read-only. |
| `browsercontract` | Versioned public browser plan, source, worker and session metadata. |
| `browsertransaction` | Canonical value-free browser-profile transactions and lifecycle validation. |
| `browserverify` | Independently verify supplied browser review and capture receipt evidence. |
| `approval`, `authority` | Exact package/operation approval and bounded authority validation. |
| `credentialpolicy` | Symbolic credential metadata policy. |
| `handoff`, `digest`, `trust`, `wire` | Review handoffs, exact digests, bounded inventories and strict JSON decoding. |
| `udonreport`, `runevidence` | Versioned execution reports and conservative evidence verification. |

The [public trust API](docs/public-trust-api.md),
[authoring card guide](docs/authoring-card-v1.md),
[package v3 contract](docs/package-v3.md) and
[browser supplement](docs/browser-package-v3.md) define the supported surface.
The browser supplement preserves named-output ownership and historical-proof
refusal, bounded authored authentication timeouts, exact authority binding and
observed fallback rights. Old package, approval, evidence and report wires remain
readable at their documented versions. Historical inspection grants no execution
authority and never upgrades an old approval.

## Legacy command retirement

A32 removes the legacy `openudon` and `udon-runner` executables, private
synthesis/intent authoring, browser capture/authoring harnesses, and CLI/runner
archive publication. Current source is a library module. Use Kinet for the
supported authoring and worker integration; use its separately reviewed runtime
handoff for execution. The [retirement guide](docs/legacy-cli-retirement.md)
records the affected commands and replacement owners.

Existing consumers retain their exact published source/module pins. Frozen W8M
and Ramen histories are preserved; this source change does not update their
workflows or qualify adoption. Additional Kinet physical/observer methods remain
unqualified at the scope recorded in its accepted handoff. Registration is a
human action outside Stage 12 functional qualification.

## Development

Go 1.26.6 or the version selected by `go.mod` is required. Run independently of
an operator workspace:

```sh
GOWORK=off go test ./...
GOWORK=off go vet ./...
make check
```

`make public-boundary` runs the frozen public API manifests and import-boundary
checks. `tools/packagev3-fixtures` is an explicit developer fixture generator;
ordinary tests never rewrite fixtures. The offline consumer inventory recipe is
in [the A32 handoff](docs/a32-transition-cleanup.md).

Public production bytes, embedded schemas and historical fixture identities are
preserved through retirement. Legacy examples and guides remain available for
interpreting pinned versions. Source publication, downstream adoption and
execution qualification have their separate owner gates.

See [contributing](CONTRIBUTING.md), [security](SECURITY.md),
[support](SUPPORT.md) and the [Apache 2.0 license](LICENSE).
