---
tracker:
  kind: linear
  project_slug: ""
  active_states:
    - Todo
    - In Progress
    - Rework
    - Merging
  terminal_states:
    - Done
    - Closed
    - Cancelled
    - Canceled
    - Duplicate
polling:
  interval_ms: 5000
workspace:
  root: ~/code/ramen-workspaces
hooks:
  after_create: |
    git clone git@github.com:your-org/your-ramen-project.git .
    ./scripts/check-siblings.sh || true
agent:
  max_concurrent_agents: 3
  max_turns: 20
codex:
  command: codex app-server
  approval_policy: never
  thread_sandbox: workspace-write
  turn_sandbox_policy:
    type: workspaceWrite
---

You are working on a Symphony-managed Ramen issue.

Issue:
{{ issue.identifier }} - {{ issue.title }}

Description:
{% if issue.description %}
{{ issue.description }}
{% else %}
No description provided.
{% endif %}

Ramen boundaries:

1. Use UWS as the workflow interchange format.
2. Use OpenAPI for HTTP method, path, schema, server, and security details.
3. Use extension-owned UWS operations for non-HTTP runtimes such as SMTP, command execution, SSH,
   SQL, or LLM calls.
4. Use `../uws` for public schema/model validation.
5. Use `../udon` for private runtime validation, lowering, and local sandbox execution.
6. Use `../symphony` only as the work orchestration service; do not put UWS or udon runtime
   semantics into Symphony prompts unless they are project policy.
7. Do not execute production side effects directly from the agent session.
8. If execution is requested, produce or update the approved artifact and document the trusted
   runner command.

Expected artifact locations:

- Natural-language project brief: `examples/<name>/project.md`
- OpenAPI inputs: `examples/<name>/openapi/`
- Generated intent: `examples/<name>/workflows/intent.hcl`
- Generated workflow HCL: `examples/<name>/workflows/workflow.hcl`
- Exported UWS workflow: `examples/<name>/workflows/workflow.uws.yaml`
- Expected workflow plan: `examples/<name>/expected/plan.json`
- Human-readable workflow plan: `examples/<name>/expected/plan.md`
- OpenAPI discovery report: `examples/<name>/expected/discovery.json`
- Machine-readable refinement report: `examples/<name>/expected/refinement.json`
- Human-readable refinement report: `examples/<name>/expected/refinement.md`
- Review notes: `examples/<name>/expected/review.md`
- Machine-readable quality report: `examples/<name>/expected/quality.json`
- Human-readable quality report: `examples/<name>/expected/quality.md`

Preferred generation command:

```bash
go run ./cmd/ramen synthesize --example examples/<name> --provider gemini --model gemini-2.5-flash --max-attempts 5
```

Refinement commands:

```bash
go run ./cmd/ramen build --example examples/<name> --provider gemini --model gemini-2.5-flash --max-attempts 5
go run ./cmd/ramen promote --example examples/<name>
go run ./cmd/ramen assess --example examples/<name>
```

Eval command for prompt/pipeline changes:

```bash
go run ./cmd/ramen eval --root examples/eval --provider gemini --model gemini-2.5-flash
```

Default to `gemini-2.5-flash` for Gemini-backed synthesis. The pipeline is validation-first:
project preprocessing, structured output where supported, deterministic checks, and bounded repair
attempts provide most of the reliability. Escalate to `gemini-2.5-pro` only after Flash fails
deterministic checks; preview Pro models are not preferred defaults because they can be slower or
capacity-limited.

Symphony refinement loop:

1. Run `synthesize` for a new or substantially changed `project.md`.
2. If it fails, read `expected/refinement.json` and `expected/quality.json`, then repair the earliest failing stage.
3. For `openapi.*` failures, add a valid OpenAPI file under `openapi/` or add an explicit OpenAPI
   URL to `project.md`, then rerun `synthesize`.
4. For `intent.*` failures, edit `project.md` or `workflows/intent.hcl`, then rerun `build`.
5. For `workflow.*` failures, prefer improving `workflows/intent.hcl` and rerunning `build`; only
   edit `workflow.hcl` directly for narrow structural repairs, then run `promote` and `assess`.
6. For `uws.*`, `review.*`, or `artifacts.*` failures, repair the generated artifact or evidence,
   then run `promote` or `assess` as appropriate.
7. Stop after the configured attempt limit. If quality still fails, leave the best artifacts in place
   and report the blocking checks from `expected/refinement.json` and `expected/quality.json`.

Use provider environment variables for LLM credentials, such as `GEMINI_API_KEY`. Do not paste API
keys into prompts, issue descriptions, commands, or generated artifacts.

Before handoff:

1. Run `go test ./...`.
2. Run `go vet ./...`.
3. Run `make check`.
4. Run `git diff --check`.
5. Validate any generated UWS artifact with `./scripts/validate-uws.sh`.
6. Run `go run ./cmd/ramen assess --example examples/<name>` and confirm quality status is `pass`.
7. If side-effectful execution is explicitly requested, use `ramen run` with approval JSON. Do not
   run production effects from synthesis, build, promote, or assess.
8. Report exactly what was validated and whether any side-effectful execution was skipped.
