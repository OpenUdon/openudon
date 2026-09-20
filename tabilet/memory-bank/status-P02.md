# P02 Trusted Execution And Package Integrity

Item | State | Notes
--- | --- | ---
P02.1 v2 executable wires and handoff input/self digests | `[+]` | Added `openudon.executor-run.v2`, `openudon.run-evidence.v2`, and `apitools.review-handoff.v2`; v1 execution is rejected while legacy evidence remains read-only inspectable.
P02.2 Parent-to-runner validation and TOCTOU resistance | `[+]` | The outer runner receives the exact config digest and approval path; the external runner rebuilds and revalidates current package, quality, handoff, approval, tier, and canonical config bytes.
P02.3 Typed invocation and environment isolation | `[+]` | Executor calls carry argv, directory, and an allowlisted environment. Declared credentials survive; unrelated, proxy, cloud, and SSH-agent variables do not.
P02.4 Unique evidence and report ownership | `[+]` | Every run has a random ID and separate config, stage, async, report, evidence, digest, and archive paths; executor reports are bounded workdir-relative regular files bound by size and digest.
P02.5 Optional Ed25519 evidence signatures | `[+]` | Added PKCS#8/PKIX key generation, detached embedded-key signatures, trusted-key verification, required-signature mode, and negative trust/tamper coverage.
P02.6 Trusted-runner adversarial coverage | `[+]` | Covers config/approval replacement, forged configs, v1 rejection, package drift, wrong trust keys, report substitution, traversal, symlinks, and archive collisions.

No release tag is created by P02.
