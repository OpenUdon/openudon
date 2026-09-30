# M90 per-step handoff implementation

M90.1 and M90.2 have delivered the explicit v5 CLI/config handoff, independent
bounded wire/inventory validation, value-free v3 observations and compatible
verification, signatures and archives. New conformance fixtures preserve
uncertainty for dry-run, missing, stale, mismatched, malformed or incomplete
inventory. Default report v3/v4 and evidence v2 remain unchanged.

M90.3 real qualification against accepted Udon M44 source
`1a5e9aa2045e3d875da2e18aab2d6db869ac5223` passes eight disposable loopback
cases, including failed reads, killed writes, filesystem checkpoint failure
and duplicate refusal. Whole-package checks and the two-iteration bounded review pass after three
P2 findings were fixed. Accepted implementation
`ed5b206a524e6e193123b2d06714b75160379560` is published; remote readiness
commit `a388b235edc733fd23962ff006d2406d4965b48f` was independently verified.
Kinet W07 is reconciled to the exact accepted source and owns adoption;
M90 is retired in its package-local history.
