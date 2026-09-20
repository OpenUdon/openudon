# A14 Browser Authoring And Synthesis Hardening

Item | State | Notes
--- | --- | ---
A14.1 Live disclosure and approval containment | `[+]` | Frame names use Browsertools' canonical reducer; live approval tuples and exact origin inventories are validated; terminal fields are quoted; compatible MFA subsets remain human-only.
A14.2 Process and executable ownership | `[+]` | iCoT privately copies and revalidates the Browsertools executable before launch, starts an interactive process group, and kills the complete tree on cancellation without an arbitrary wall clock charging human prompts.
A14.3 Fail-closed intent and lowering | `[+]` | Browser-family reads/discriminators, auth timeouts and fields, symbolic credential bindings, exact approvals, nested steps, and generic browser outputs are validated before UWS lowering.
A14.4 Shared nested session analysis | `[+]` | One conservative document-order walker serves quality and elicitation; authentication in a conditional branch is not treated as guaranteed for later protected actions.
A14.5 Hardened review, registry, and staging | `[+]` | Authentication review uses bounded strict regular-file reads, registry text is control/secret-safe, equal cross-registry IDs remain visible with collision-safe materialization, and durable create-only staging removes the check-rename overwrite race.
A14.6 Regression closure | `[+]` | Focused workflow-intent, synthesis, iCoT, elicitor, registry, process, and atomic-writer suites cover the reviewed failure classes.

This milestone adds no unattended login, secret capture, browser-state transfer,
or broader browser runtime ownership.

Implementation commit: OpenUdon `69b29e4`.
