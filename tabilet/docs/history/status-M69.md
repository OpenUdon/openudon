# Retired milestone M69 - Status M69 - CLI-First Public Beta

**Milestone.** M69
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M69.md
**Source status SHA-256.** 91f778827d62b817f4b1c9ae3848184224cca524d8027ba68b3d534fccb62ea5
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# Status M69 - CLI-First Public Beta

State: Complete

State of the standalone module, supported CLI/artifact boundary, release
archives, and first public OpenUdon release.

## Goal

Make OpenUdon installable and evaluable as a CLI-first public beta for workflow
authors, reviewers, and platform operators while preserving the trusted
executor boundary and keeping LLM/provider behavior experimental.

## Scope

- Support the deterministic package, approval, trusted-handoff, and
  run-evidence command surface through the v0.1.x line.
- Keep iCoT/LLM behavior, prompts, catalog/eval/readiness/smoke helpers, and
  provider behavior experimental before v1.
- Publish only public module dependencies and prove the module without the
  parent workspace or sibling checkouts.
- Bundle `openudon`, `icot`, and `udon-runner` for six OS/architecture targets.
- Require provider-free local udon smoke evidence before tagging.

## Task Status

| Item | State | Notes |
|---|---:|---|
| Actions failure diagnosis | `[+]` | Run `27101284157` failed during standalone vet because the pinned Authoring revision lacked `operationlifecycle`; the failure occurred before the eval matrix. |
| Lifecycle/eval fixes | `[+]` | Existing commits `1cb5a54` and `249e7b4` preserve hermetic eval discovery and align lifecycle hints/details. |
| Public dependencies | `[+]` | Authoring is pinned to published lifecycle revision `3aa69a0`; UWS is aligned with published revision `d428f54`; no local replaces are committed. |
| Public beta contract | `[+]` | README, support, security, compatibility, contribution, release, and documentation pages define the supported core and experimental surfaces. |
| Version metadata | `[+]` | `openudon version --json` reports local build metadata without network access; release archives inject the tag and module installs derive it from Go build information. |
| Hermetic release evidence | `[+]` | Release-note and consolidated evidence commands accept an explicit validated commit revision when `.git` is unavailable. |
| Automation | `[+]` | Public CI downloads dependencies, rejects replaces, tests/vets standalone, checks boundaries, and cross-builds all three CLIs for six targets. Tag CI verifies, packages, checksums, and publishes releases. |
| Credential-free golden path | `[+]` | The release archive workflow uses the runtime-only fixture for offline iCoT authoring, build, assess, approval, dry-run staging, and runner-shim help. |
| Local release gates | `[+]` | Clean-archive and full local test/vet passed; `make release-saas-check` passed; `make release-evidence` completed a provider-free non-dry-run sibling udon handoff and reverified the archived async sidecar. |
| Public CI | `[+]` | Main run `30166478994` passed the standalone public-module job and all six three-command cross-build jobs. |
| v0.1.0 release | `[+]` | Annotated tag `v0.1.0` points to `86b02af`; release run `30166536155` published six archives plus `SHA256SUMS`. All downloads and checksums passed, archive contents were verified, and isolated installs completed the credential-free golden path. |

## Scoped Commits

- Eval discovery: `1cb5a54`
- Lifecycle planning: `249e7b4`
- Standalone dependency and CI repair: `29e2db2`
- Public beta contract and version metadata: `80485cd`
- Release automation: `86b02af`
- Tracked memory/evolution update: this tofu commit

## Verification

- `GOWORK=off go mod download`
- `GOWORK=off go test ./... -count=1 -timeout=10m`
- `GOWORK=off go vet ./...`
- clean `git archive` standalone test/vet
- `go test ./...`, `go vet ./...`, `make release-saas-check`
- `make release-evidence`
- `mkdocs build --strict`
- `actionlint`
- Linux/macOS/Windows amd64/arm64 builds for all three commands
- installed `v0.1.0` version and credential-free package lifecycle
- downloaded release archive verification against `SHA256SUMS`
- `git diff --check`
- `git -C ../tofu diff --check -- openudon`

Final evidence:

- clean `git archive 86b02af`: standalone download/test/vet passed without
  `.git`, parent workspace, or sibling checkouts
- local `make release-saas-check`: passed
- local `make release-evidence`: pass at commit `86b02af`, with one smoke
  scenario and one verified async sidecar
- public test run `30166478994`: standalone module plus all six cross-builds
  passed
- release run `30166536155`: verify, six builds, packaged Linux amd64 golden
  path, checksums, and publication passed
- release URL: `https://github.com/OpenUdon/openudon/releases/tag/v0.1.0`
- downloaded all seven assets; every archive passed `sha256sum -c
  SHA256SUMS` and contained `openudon`, `icot`, `udon-runner`, README, and
  license
- Linux amd64 archive reported version `0.1.0`, revision `86b02af`, and
  `vcs.modified=false`
- isolated `go install` of all three commands at `v0.1.0` succeeded; the
  installed `openudon` reported `0.1.0` and the installed commands completed
  offline authoring, build, assess, approval, dry-run staging, and runner help
~~~~~~~~~~~~~~~~~~~~
