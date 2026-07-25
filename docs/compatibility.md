# v0.1 Compatibility Contract

OpenUdon's v0.1 compatibility boundary is deliberately smaller than its full
command surface.

## Stable During v0.1.x

- Deterministic UWS validation and package generation through `validate`,
  `build`, `promote`, and `assess`.
- Digest-bound approval generation and trusted handoff through
  `approval-template` and `run`.
- Run evidence verification and archival through `run-evidence`.
- Existing OpenUdon-owned versioned approval, run-config, run-evidence, and
  async-evidence documents, plus the current review-handoff package contract.
- The `OPENUDON_EXECUTOR` and `OPENUDON_UDON_RUNNER` selector rules and the
  portable `udon-runner` run-config boundary.

Additive flags and artifact fields may be introduced in v0.1.x. Existing
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
implementation packages remain internal, and v0.1 compatibility is a CLI and
versioned-artifact contract.
