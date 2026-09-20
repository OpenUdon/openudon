# OpenUdon v0.2 Deep-Review Remediation

Treat the deep review as a security migration rather than a compatible v1
extension. Deliver four ordered milestones: P02 trusted execution and package
integrity, A12 authoring/provider/UI safety, E06 honest reproducible evidence,
and M74 consolidation/release closure.

Executable packages must use review-handoff, executor-run, and run-evidence v2
contracts with exact input, self, approval, config, package, executor-report,
and archive binding. The trusted external runner must independently rebuild
current state, match canonical config bytes, isolate child environments, and
reject v1 execution. Add optional Ed25519 signatures without placing private
keys at the executor boundary.

Harden credential recognition, provider body/key handling, browser bootstrap,
DNS-aware remote-source fetches, canonical seed sources, clone isolation,
security-alternative persistence, and HCL diagnostics. Evidence must report
`not_run` honestly, terminate bounded process trees, retain portable paths,
self-clean eval workspaces, sort map-derived output, and use minimal fixtures.

Consolidate atomic/evidence/validation/CLI policy, remove dead code and unused
schema snapshots, split large files below 1,000 lines, update durable memory,
and run the complete verification matrix. Do not tag or publish v0.2.0. Keep
A11.5 pending until a hosted sandboxed Chromium run succeeds.
