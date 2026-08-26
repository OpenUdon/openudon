# v0.2 Compatibility Contract

OpenUdon v0.2 is a security migration. Its compatibility boundary remains
deliberately smaller than the full command surface.

## Stable During v0.2.x

- Deterministic UWS validation and package generation through `validate`,
  `build`, `promote`, and `assess`.
- Digest-bound approval generation and trusted handoff through
  `approval-template` and `run`.
- Explicit `package prepare|promote|inspect|recover` lifecycle commands and
  exact `--package-store`/`--selection` approval or run adapters preserve the
  existing approval, package-byte, and trusted-runner contracts. The legacy
  artifact `openudon promote` command retains its original meaning.
- Run evidence verification and archival through `run-evidence`.
- `openudon.approval.v1` and `openudon.async-evidence-bundle.v1`.
- Executable `openudon.executor-run.v2`, `openudon.run-evidence.v2`, and
  `apitools.review-handoff.v2` documents.
- The `OPENUDON_EXECUTOR` and `OPENUDON_UDON_RUNNER` selector rules and the
  portable `udon-runner` run-config boundary.
- Additive value-free browser replay fields in executor-run and run-evidence
  v2: driver path/arguments, protocol, canonical environment mappings, and
  exact reviewed operation/authentication approvals.
- Additive value-free browser-registration package and dry-run fields:
  canonical symbolic credential mappings and exact reviewed registration
  operation approvals. This is not a live-execution compatibility claim;
  executor construction fails closed until compatible downstream contracts
  are pinned.
- The public `openudon.browser-profile-transaction.v1` JSON artifact and its
  schema. It coordinates the existing BAP/BCP and BRP families through closed,
  value-free review, preparation, promotion, and recovery states; it neither
  changes UWS semantics nor grants browser, executor, or target authority.

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
versioned-artifact contract. The browser-profile transaction therefore has a
public JSON/schema boundary but no supported Go library API. The additive
`icot browser-transaction` command and experimental iCoT API v4 expose the
same internal lifecycle, but neither frontend is part of the stable v0.2 CLI
or HTTP compatibility boundary. The replaced API v3 namespace is closed.

## v1 Migration

Existing v1 packages must be regenerated with `openudon build` before they can
execute. `openudon run` and `udon-runner` reject v1 handoffs and run configs.
Legacy v1 run evidence may be structurally verified for inspection, but it
cannot be signed as v2 evidence or archived where bounded executor-report
ownership cannot be proven.

Optional Ed25519 signatures authenticate an operator only when verification
pins a trusted PKIX public key. Embedded-key verification proves evidence
integrity but does not by itself establish configured operator identity.
