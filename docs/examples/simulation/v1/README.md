# Pure simulation conformance v1

`example/` was produced by `step pending` and `PackageFromIntent` through the
real owner paths. Its quality fails and its UWS 1.12 contract remains pending.
`results/pending.json` is actual deterministic pure simulation output for this
package scope; `results/blocked.json` is a fixed versioned refusal.

Inputs and results are checked against their published local schemas. Invalid
vectors reject execution authority, invented pending endpoints and live-read
provenance. Runtime/CLI tests copy the package into disposable roots and prove
its byte/digest immutability. Never execute this unresolved example.
