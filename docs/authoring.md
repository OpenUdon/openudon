# Authoring

OpenUdon has two supported authoring paths. Both produce the same reviewable package shape: a
human-readable `project.md`, a structured `workflows/intent.hcl`, public UWS artifacts, expected
plans, review evidence, quality reports, and a handoff manifest.

## Path 1: Guided iCoT

Use iCoT when you want an operator-guided session that starts from a goal and writes the initial
brief plus intent.

```bash
go run ./cmd/icot --example ./examples/<name>
```

iCoT writes `project.md` and `workflows/intent.hcl`; it does not execute workflows. It can run with
LLM assistance, with the fixed manual flow, from an existing example, or from YAML/JSON answers.

After iCoT saves artifacts, continue with:

```bash
go run ./cmd/openudon build --example ./examples/<name>
go run ./cmd/openudon assess --example ./examples/<name>
```

Use [iCoT](icot.md) for command details and [Project Briefs](project-authoring.md) for the
sections a good `project.md` should contain.

## Path 2: Brief And Synthesis

Use synthesis when you already have a project brief or are updating a fixture.

```bash
go run ./cmd/openudon synthesize --example ./examples/<name>
go run ./cmd/openudon build --example ./examples/<name>
go run ./cmd/openudon assess --example ./examples/<name>
```

`synthesize` reads `project.md`, discovers or imports local OpenAPI documents, creates or updates
intent, and writes the generated package artifacts. `build` regenerates from existing intent.
`assess` reruns deterministic quality checks without synthesizing new intent.

Use [Synthesize](synthesize.md), [intent.hcl](intent.md), and [Data Flow](data-flow.md) for the
artifact contracts.

## Safety Rules

- Put credential binding names in artifacts, never credential values.
- Keep side-effectful workflows in generated/review state until approval.
- Use sandbox proof-run language for examples that send email, write records, call commands, or
  otherwise produce effects.
- Use `openudon run --dry-run` to validate the handoff package without invoking the executor.
