# Release Stewardship

OpenUdon release checks are split between public, provider-free gates and local
maintainer evidence. Public automation must not depend on ignored memory-bank,
evolution, readiness, eval, approval, or run-workdir files.

## Public Gates

GitHub Actions runs the public Go module with workspace mode disabled:

```bash
test -z "$(grep -E '^[[:space:]]*replace[[:space:]]' go.mod)"
GOWORK=off go mod download
GOWORK=off go vet ./...
GOWORK=off go test ./... -count=1 -timeout=10m
GOWORK=off go run ./cmd/openudon check-apitools-boundary
git diff --check
```

Pull-request and main-branch CI also cross-builds `openudon`, `icot`, and
`udon-runner` for Linux, macOS, and Windows on amd64 and arm64.

Documentation publishing builds the MkDocs site in strict mode before deploy:

```bash
mkdocs build --strict
```

The repository boundary check rejects direct OpenUdon imports of old lifecycle
`apitools` APIs, private udon executor packages, private `genelet/*` executor
modules, infrastructure engine internals, and parser/conversion packages.

## Local Maintainer Gates

`make release-check` is the fast deterministic local pre-tag gate:

```bash
make release-check
```

`check-doc-memory` is intentionally local. It verifies ignored memory-bank and
evolution harness files in maintainer checkouts and warns when milestone changes
may need a new evolution record. It is not a public CI gate.

For the SaaS release story, run the comprehensive provider-free local gate:

```bash
make release-saas-check
```

`release-saas-check` runs `release-check`, `browser-integration-check`, the
required network-free `browser-scenario-loopback`, `eval-seed-build`,
`icot-variants-validate`, `icot-variants-coverage`, `icot-authoring-scorecard`, UWS validation,
doc-memory, n8n bridge validation, strict MkDocs build, selected strict SaaS fixture lint, and the
provider-free dry-run demo in
[SaaS Operator Release Path](saas-operator-release.md). `icot-authoring-scorecard` generates the
provider-free scorecard and then runs `icot report verify` against `scorecard.json`, including the
digest sidecar and retention/share-safety metadata. The selected demo
examples are:

- `gmail-send-audit-receipt` for a single-service side-effectful send-and-audit
  workflow;
- `order-fulfillment-chain` for a multi-service lookup-and-create workflow.

`browser-integration-check` runs the named, provider-free browser
authoring-to-handoff matrix across OpenUdon, Browsertools, UWS, Udon, and
Browserdriver, writes a value-free digest-sidecar report, and verifies it. It
does not retain child-process output. See
[Browser Integration Evaluation](browser-integration-eval.md) for the exact
gates and separate loopback-only installed-browser opt-ins.

`browser-scenario-loopback` is the real-browser release complement to that
browser-free matrix. It runs the fixed 21-case author-session v2 and trusted
v3 replay corpus, then verifies the value-free report sidecar. The weekly
`browser-scenario-public` workflow runs four anonymous read-only canaries with
explicit network authority and is informational. See [Browser Scenario
Evaluation](browser-scenario-eval.md). Both hosted Ubuntu jobs explicitly
enable sandbox-compatible unprivileged user namespaces on their ephemeral
runner and retain Chromium's sandbox; `xvfb-run` supplies only the display.

The demo must use ignored `.openudon-run/...` output, sandbox approval JSON, and
`openudon run --dry-run`. Do not commit approval JSON, run configs, transcripts,
or real-provider outputs.

For trusted executor proof runs with a compatible udon binary, archive the local
`run-evidence.json`, `async-evidence.json`, and the udon `executor-report.json`
from the staged executor workdir. Verify the archived bundle with:

```bash
go run ./cmd/openudon run-evidence verify --file <archive>/run-evidence.json
```

The executor report is non-secret status metadata. Do not archive credential
values or raw executor stdout/stderr as release evidence.

To run the consolidated provider-free evidence flow, use:

```bash
go run ./cmd/openudon release-evidence
# or
make release-evidence
```

This builds the sibling udon CLI, runs the local runtime-only smoke, archives
and verifies the emitted evidence bundle, drafts release-note evidence, and
writes JSON/Markdown summaries under ignored `.openudon-run/release-evidence/`.
It does not tag, publish, commit artifacts, or run live providers. Add
repeatable gate notes with `--gate "go test ./...=pass"` when capturing a
specific release candidate.

Run the eval seed/build matrix directly when changing authoring fixtures or
reference intents:

```bash
make eval-seed-build
make icot-variants-validate
```

The v0.1.0 tag gate requires the provider-free release gates plus the local
trusted-executor evidence flow:

```bash
make release-saas-check
make release-evidence
```

`make release-evidence` may build and invoke a reviewed sibling udon binary
against the runtime-only fixture. It does not call a live SaaS provider and
does not make udon a public Go dependency.

The product smoke matrix remains optional historical/provider evidence:

```bash
make product-smoke-check
OPENUDON_EXECUTOR=/absolute/path/to/udon make product-smoke-live
```

`product-smoke-check` is provider-free and builds ignored scratch packages from
the reviewed eval fixtures. `product-smoke-live` is local maintainer evidence:
local synthetic APIs run against a stub server, and optional Slack or
OpenWeatherMap proof runs require explicit operator-owned credentials. Gmail has credential-backed examples
and manual proof-run support, but the product smoke matrix records dry-run
evidence for Gmail unless an operator separately runs and records a reviewed
Gmail proof. Jira currently has fixture/dry-run coverage but no recorded
real-key proof. See
[Product Smoke Matrix](product-smoke-matrix.md).

## Publishing v0.1.0

Only tag a clean commit after standalone CI and the selected local release
evidence pass. Push an annotated `v0.1.0` tag; the release workflow reruns
standalone gates, builds six platform archives, verifies the Linux amd64
credential-free author/build/assess/approval/dry-run path, writes
`SHA256SUMS`, and publishes the GitHub release.

Each archive contains `openudon`, `icot`, `udon-runner`, `README.md`, and
`LICENSE`. Release binaries inject the tag into `openudon version --json`;
`go install ...@v0.1.0` derives the same version from Go build information.
After publication, download every asset, verify all checksums, and repeat the
installed-binary golden path before recording the release complete.

Real-provider evals remain opt-in local evidence:

```bash
make release-eval
go run ./cmd/icot authoring-eval --root examples/eval --include-variants --provider copilot-api --model gpt-5.4-mini --out eval/runs/icot-authoring-eval-local
go run ./cmd/icot report verify --file eval/runs/icot-authoring-eval-local/authoring-eval.json
```

Record provider, model, corpus size, comparison baseline, provider drift status,
optional authoring-eval report path, authoring-eval pass summary, retention/share-safety metadata,
and known gaps in the release notes.

Provider/model drift is release evidence, not a deterministic gate by itself.
Record transient provider failures and rerun once from a trusted workstation
when availability or rate limits look external.

## Boundary Recap

OpenUdon's release evidence must keep these ownership boundaries clear:

- n8n and `../try-n8n` provide service-priority and pattern evidence only; they
  are not runtime dependencies or import targets.
- Live SaaS providers are not contacted by build, assess, iCoT, eval, or
  trusted-runner dry-run demo commands.
- External review orchestration may route review from OpenUdon evidence, but
  identity, state transitions, and audit persistence stay outside OpenUdon.
- Udon or another trusted executor receives a package only through
  `openudon run` after approval and digest validation.
