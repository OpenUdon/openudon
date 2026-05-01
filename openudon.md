# OpenUdon Shared Intent Boundary

This note supersedes `migrate.md` conceptually for OpenUdon intent integration direction. It does not replace or edit that file.

Ramen now imports the public OpenUdon `intent` core through a compatibility adapter:

- `github.com/genelet/openudon/intent`

The first adoption slice keeps Ramen's existing `rollout.Intent` workflow contract and wraps it with OpenUdon's shared parse, render, validate, missing-slot, transcript, artifact, and chat abstractions. This proves the shared authoring boundary without replacing Ramen's workflow intent model.

Ramen can later import these additional OpenUdon IaC packages for an infrastructure-shaped authoring path:

- `github.com/genelet/openudon/intent/v1alpha1`
- `github.com/genelet/openudon/intent/tfgen`

The shared boundary is authoring-oriented. OpenUdon owns reusable intent sessions, slots, transcripts, symbolic binding refs, diagnostics, artifacts, OpenUdon v1alpha1 IaC intent loading/validation, and deterministic `.tf` generation.

Ramen remains responsible for Ramen-specific `intent.hcl -> workflow.hcl` behavior, Symphony policy, UWS workflow semantics, guided iCoT orchestration policy, approvals, trusted runner integration, and private runtime decisions.

The invariant is that binding happens at execution time. OpenUdon intent files declare binding names and references only. Ramen, Udon, `w8m`, or another private execution layer supplies concrete credentials, endpoints, account IDs, approval state, private profile packs, and runtime policy later.

The compatibility-adapter slice is intentionally narrow. Ramen Go code depends on the generic OpenUdon `intent` package only; it does not yet depend on OpenUdon's concrete IaC intent model or `.tf` generator.
