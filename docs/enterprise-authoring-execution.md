# Enterprise Authoring And Execution Boundary

OpenUdon uses an LLM to help author a workflow specification. It does not rely
on an LLM to execute the workflow.

The intended enterprise flow is:

```text
natural-language request
  -> LLM-assisted iCoT authoring
  -> reviewable intent.hcl / UWS artifacts
  -> deterministic build, quality, review, and approval gates
  -> trusted executor handoff
```

The specification is the contract between the user request and real execution.
It declares which APIs are called, what data moves between steps, what
credentials are required, what systems are read or written, and which side
effects need approval.

This separation is important because enterprise automation needs:

- predictability: the same approved workflow runs the same way;
- auditability: reviewers can inspect touched systems before execution;
- security: credentials and external writes are explicit bindings, not model
  improvisation;
- compliance: the workflow artifact becomes review and change-control evidence;
- low token use: the LLM is used mainly during authoring, not every runtime
  step;
- lower operational risk: production side effects are handled by deterministic
  runtime logic after approval.

iCoT therefore produces a candidate `project.md` and `workflows/intent.hcl`.
Execution readiness is established only after deterministic lint, build,
quality, review, approval, package digest, and trusted-runner checks pass.

Provider credentials, channel IDs, OAuth tokens, and live provider responses
must not be pasted into prompts or committed artifacts. Real provider execution
remains an operator-owned, side-effectful handoff after review.
