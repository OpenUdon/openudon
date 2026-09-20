# Result V26 - OpenUdon v0.2 Deep-Review Remediation

P02 replaces executable v1 artifacts with `apitools.review-handoff.v2`,
`openudon.executor-run.v2`, and `openudon.run-evidence.v2`. Every handoff input
and run artifact is digest-bound; runs are unique; executor reports have
bounded relative ownership; external runners repeat the full current-state and
canonical-byte validation path; child environments are allowlisted; v1 cannot
execute or archive as v2. Optional detached Ed25519 signatures distinguish
embedded integrity from trusted-key operator identity.

A12 centralizes secret recognition, sends Gemini keys only by header, bounds
provider bodies, installs the iCoT cookie through a terminal-only expiring
single-use throttled code, pins DNS-validated remote connections, unifies seed
directories, deep-clones draft state, fingerprints security alternatives, and
maps strict HCL diagnostics to original lines.

E06 introduces `not_run`, fixed process-tree deadlines, package-relative
quality/refinement evidence, self-cleaning eval workspaces, deterministic map
output, and minimal Slack fixtures. M74 consolidates durable atomic and strict
evidence helpers, extracts reusable CLI policy, removes all pinned dead code
and unused schema snapshots, and splits former grab-bag sources below 1,000
lines with dedicated regression coverage.

The v0.2-compatible tree is prepared without a release tag. A11.5 remains
pending because only hosted sandboxed Chromium evidence can close it. The
2026-08-20 local sandbox-enabled attempt found Chromium but failed with `No
usable sandbox`; it was not replaced by a sandbox-disabled run.
