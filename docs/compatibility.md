# v0.2 Compatibility Contract

OpenUdon v0.2 is a security migration. Its compatibility boundary remains
deliberately smaller than the full command surface.

## Stable During v0.2.x

- Deterministic UWS validation and package generation through `validate`,
  `build`, `promote`, and `assess`.
- Digest-bound approval generation and trusted handoff through
  `approval-template` and `run`.
- Run evidence verification and archival through `run-evidence`.
- `openudon.approval.v1` and `openudon.async-evidence-bundle.v1`.
- Executable `openudon.executor-run.v2`, `openudon.run-evidence.v2`, and
  `apitools.review-handoff.v2` documents.
- The `OPENUDON_EXECUTOR` and `OPENUDON_UDON_RUNNER` selector rules and the
  portable `udon-runner` run-config boundary.
- Additive value-free browser replay fields in executor-run and run-evidence
  v2: driver path/arguments, protocol, canonical environment mappings, and
  exact reviewed operation/authentication approvals.

Additive flags and artifact fields may be introduced in v0.2.x. Existing
fields, accepted inputs, and documented core behavior will not intentionally
be removed or redefined. Readers continue to reject unknown future wire
versions rather than guessing.

Public UWS document semantics and schema compatibility remain governed by the
UWS project. OpenUdon does not promise that an external executor supports
every UWS source family or runtime selector it can review and package.

## Experimental Before v1

The iCoT authoring CLI, LLM-assisted synthesis and repair, prompt wording,
provider and model behavior, catalog advice, eval/readiness/smoke commands,
n8n bridge evidence, release helpers, and exact generated Markdown remain
experimental. These surfaces may change between pre-1.0 releases while the
deterministic package and trusted-handoff boundary stays compatible.

OpenUdon currently has no supported importable Go packages. The module's
implementation packages remain internal, and v0.2 compatibility is a CLI and
versioned-artifact contract.

## v1 Migration

Existing v1 packages must be regenerated with `openudon build` before they can
execute. `openudon run` and `udon-runner` reject v1 handoffs and run configs.
Legacy v1 run evidence may be structurally verified for inspection, but it
cannot be signed as v2 evidence or archived where bounded executor-report
ownership cannot be proven.

Optional Ed25519 signatures authenticate an operator only when verification
pins a trusted PKIX public key. Embedded-key verification proves evidence
integrity but does not by itself establish configured operator identity.
