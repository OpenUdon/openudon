# OpenUdon Shared Intent Boundary

This note supersedes `migrate.md` conceptually for OpenUdon intent integration direction. It does not replace or edit that file.

Ramen now imports the public OpenUdon `intent` core through a compatibility adapter:

- `github.com/genelet/openudon/intent`

The first adoption slice keeps Ramen's existing `rollout.Intent` workflow contract and wraps it with OpenUdon's shared parse, render, validate, missing-slot, transcript, artifact, and chat abstractions. This proves the shared authoring boundary without replacing Ramen's workflow intent model.

The shared boundary is authoring-oriented. OpenUdon owns reusable generic intent sessions, slots, transcripts, symbolic binding refs, diagnostics, and artifacts that Ramen can reuse for its own private workflow/iCoT domain.

OpenUdon's concrete IaC path is OpenUdon-owned: `cmd/openudon`, `intent/v1alpha1`, and `intent/tfgen` draft OpenUdon infrastructure intent and render reviewed intent to `.tf`. Ramen should not import those concrete IaC packages as part of its private workflow/iCoT path, and Ramen should not own `.tf` generation.

Ramen remains responsible for Ramen-specific `intent.hcl -> workflow.hcl` behavior, Symphony policy, UWS workflow semantics, guided iCoT orchestration policy, approvals, trusted runner integration, and private runtime decisions.

The invariant is that binding happens at execution time. OpenUdon intent files declare binding names and references only. Ramen, Udon, `w8m`, or another private execution layer supplies concrete credentials, endpoints, account IDs, approval state, private profile packs, and runtime policy later.

The compatibility-adapter slice is intentionally narrow. Ramen Go code depends on the generic OpenUdon `intent` package only; it does not depend on OpenUdon's concrete IaC intent model or `.tf` generator.
