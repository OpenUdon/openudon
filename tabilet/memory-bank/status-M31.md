# M31 - Trusted Execution Hardening

## Goal

Make `openudon run` a stronger, auditable trusted-handoff gate for approved
packages. M31 is execution-handoff hardening only: it does not add public UWS
semantics, provider execution, orchestration service behavior, or udon runtime
implementation.

## Status

| Item | State | Notes |
|---|---|---|
| Milestone framed | `[+]` | M31 follows M30 and focuses on `openudon run`, approval/digest enforcement, executor handoff config, dry-run behavior, and sandbox/production boundary evidence. |
| Strict run-config parsing implemented | `[+]` | `openudon.executor-run.v1` loading now rejects unknown fields, trailing JSON, invalid package digests, missing package paths, direct production flags, unsafe paths, and credential env-name collisions. |
| Dry-run staging implemented | `[+]` | `openudon run --dry-run` now validates runner config, stages the digest-covered package, and verifies the staged digest without requiring credential values or invoking the executor. |
| Run evidence implemented | `[+]` | `openudon.run-evidence.v1` is written to `<workdir>/run-evidence.json` with non-secret gate, package, stage kind, digest, credential binding, and executor invocation evidence, including failed invocation status. |
| Approval parsing hardened | `[+]` | Approval JSON parsing now rejects trailing JSON in addition to unknown fields and existing scope/state/tier/digest/expiry checks. |
| Operator docs updated | `[+]` | README, review handoff, safety, SaaS review handoff, SaaS operator release, architecture, tech-stack, and milestone memory now describe staged dry-runs and run evidence. |
| Deterministic gates checked | `[+]` | Ran focused trusted-runner/udon-runner/package-artifact/CLI tests, `go test ./...`, `go vet ./...`, `go run ./cmd/openudon check-doc-memory`, UWS validation, `make check`, and diff checks for OpenUdon and the tracked memory bank. |

## Design Notes

- Dry-run is still provider-free and credential-value-free, but it now proves
  the same package staging and staged digest invariant used by real handoff.
- Run evidence is OpenUdon-owned operational evidence, not a public UWS schema.
  It must never include credential values, raw environment values, or approval
  notes.
- Evidence uses `stage_kind: preflight` for `OPENUDON_UDON_RUNNER` handoff
  because the external runner owns final executor-visible staging.
- `OPENUDON_EXECUTOR` remains the canonical final executor selector. The
  optional `OPENUDON_UDON_RUNNER` outer shim override remains a trusted operator
  input and is validated as an absolute executable path.
- Production execution remains outside OpenUdon and still requires
  `approved_for_production`, trusted credentials, and a configured executor.
