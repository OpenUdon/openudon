# Status M72 - Structured API Security Alternatives

State: Complete

| Item | State | Notes |
|---|---|---|
| M72 security-alternative authoring | `[+]` | Commit `00a5a4e` migrates to Apitools security requirement sets and `authoring.prompt-context.v2`; prompts retain OR/AND/anonymous structure; iCoT forces a stable indexed choice, persists it and safety evidence across resume, requires only the chosen credentials, and rejects unselected request fields. Incomplete discovery, prompt-budget deferral, network-policy retention, and source target collision regressions are covered; full tests and vet pass. |

## Boundary Checks

- Selection is authoring evidence, not runtime credential resolution or API
  execution approval.
- Candidate workflows receive no source/operation/security breakdown.
- Final intent approval remains non-deferrable.
