# OpenUdon v0.1 Support Policy

OpenUdon v0.1 is a CLI-first public beta for operators and platform teams that
author, review, package, and hand approved UWS workflows to a trusted executor.

## Supported Core

The supported v0.1 command surface is:

- `openudon version`
- `openudon validate`
- `openudon build`
- `openudon promote`
- `openudon assess`
- `openudon approval-template`
- `openudon run`
- `openudon run-evidence verify`
- `openudon run-evidence archive`

OpenUdon will not intentionally make breaking changes to these commands, the
trusted executor selector contract, or existing versioned approval, handoff,
run-config, run-evidence, and async-evidence artifacts within the v0.1.x line.
A necessary breaking change will be released as v0.2.0 and called out in
release notes.

OpenUdon does not yet expose a supported Go-library API. Its implementation
packages remain under `internal/`.

## Experimental Before v1

The `icot` companion CLI, LLM-assisted synthesis and repair behavior, prompt
wording, provider/model integrations, catalog advice, eval and readiness
commands, n8n bridge evidence, smoke matrices, release-evidence helpers, and
exact generated prose may evolve between pre-1.0 releases.

The `udon-runner` companion is distributed with OpenUdon so deployments can
use the documented portable runner shim. Its supported boundary is the
existing versioned run-config contract and the `OPENUDON_EXECUTOR` /
`OPENUDON_UDON_RUNNER` selection policy, not a promise about any particular
executor implementation or provider.

## Platforms and Help

Release archives target Linux, macOS, and Windows on amd64 and arm64. Each
archive contains `openudon`, `icot`, and `udon-runner`.

Questions and reproducible bugs may be filed in GitHub Issues. Support is
community best effort; no uptime, response-time, provider-availability, or
operational SLA is provided.
