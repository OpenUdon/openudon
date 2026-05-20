# Related Projects

OpenUdon is the integration layer above several sibling projects. Keep changes in the repo that owns
the relevant behavior.

| Project | Ownership Boundary |
| --- | --- |
| [UWS](https://github.com/OpenUdon/uws) | Public workflow semantics, UWS versions, schema, parsing, validation, and Go model. |
| [apitools](https://github.com/OpenUdon/apitools) | OpenAPI-first API metadata discovery, import/lowering, search, indexing, summaries, auth/security summaries, catalog metadata, and operation ranking. Discovery and Smithy are upstream source families lowered before OpenUdon consumes operation metadata. |
| [tfconfig](https://github.com/OpenUdon/tfconfig) | Static Terraform/OpenTofu configuration parsing used by `openudon convert tf`. |
| `udon` | Private UWS/OpenAPI compiler and runtime executor. OpenUdon invokes it only through the trusted run-config handoff. |
| `symphony` | Optional work orchestration, isolated workspaces, reviewer routing, managed state transitions, identity, and audit persistence. |
| n8n / `../try-n8n` | Service-priority and workflow-pattern evidence for SaaS authoring. OpenUdon does not import or execute n8n workflows. |
| [OpenW8M](https://github.com/OpenUdon/openw8m) | Public OpenAPI-backed infrastructure authoring and planning. It is not an OpenUdon compatibility gate while the OpenAPI-first apitools boundary is active. |

## Rule Of Thumb

- Public workflow semantics belong in UWS.
- Generic execution or compilation behavior belongs in executor implementations such as `udon`.
- OpenAPI-first API metadata search, discovery, import/lowering, and operation metadata belong in apitools.
- Static Terraform/OpenTofu parsing belongs in `github.com/OpenUdon/tfconfig`.
- Terraform/OpenTofu provider execution, state, plan/apply, refresh, imports, and cloud SDK calls
  stay outside OpenUdon.
- n8n workflows are evidence for authoring, not an execution or compatibility target.
- Symphony-owned workflow management stays in Symphony.
- Live SaaS provider calls are outside build, assess, iCoT, eval, and dry-run release demos.
- OpenUdon owns project templates, examples, review state, approval templates, package contents,
  package digests, and local trusted-runner enforcement.

Concrete infrastructure authoring and `.tf` generation are outside OpenUdon's scope.
