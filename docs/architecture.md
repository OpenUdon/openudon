# Ramen Architecture

Ramen is an integration layer above Symphony, UWS, and udon.

## Responsibilities

Ramen owns:

- Symphony workflow templates for managing UWS/OpenAPI projects.
- Example project briefs and expected artifact layouts.
- Local validation and execution wrappers.
- Product-specific policy for when generated artifacts are safe to execute.

Ramen does not own:

- Public workflow semantics; those belong in `../uws`.
- Generic workflow compilation or execution; those belong in `../udon`.
- Agent scheduling, workspace isolation, or issue polling; those belong in `../symphony`.

## Data Flow

```text
natural-language project brief
  -> Symphony issue
  -> Codex session in isolated workspace
  -> Ramen synthesize command
  -> OpenAPI import/selection
  -> intent.hcl
  -> udon-generated workflow.hcl
  -> exported UWS artifact
  -> refinement report, quality report, and review evidence
  -> approved UWS artifact
  -> udon execution by trusted runner
```

## Dependency Direction

```text
ramen -> uws
ramen -> udon
ramen -> symphony operationally
```

Ramen may import Go packages from `uws` and `udon`. It should not import or fork Symphony's Elixir
implementation; Symphony is invoked and configured operationally through `WORKFLOW.md`.

Generic OpenAPI parsing, workflow lowering, and execution behavior should remain in `udon`. Ramen
may wrap those capabilities to enforce project layout, review evidence, and trusted handoff policy.

Ramen performs a bounded deterministic refinement loop for generation stages and records each
attempt in `expected/refinement.json`. Symphony owns the broader work loop around these commands:
Symphony/Codex uses refinement and quality reports to decide whether to improve OpenAPI inputs,
intent HCL, workflow HCL, project policy, or review evidence.

Ramen also includes an opt-in eval harness for prompt and pipeline changes. Eval briefs live under
`examples/eval`, are copied to temporary workspaces before synthesis, and produce JSON/Markdown run
reports under `eval/runs` so prompt versions, pass rates, attempts-to-pass, and reference intent
drift can be compared without dirtying committed examples.
