# Related Projects

OpenUdon is the integration layer above several sibling projects. Keep changes in the repo that owns
the relevant behavior.

| Project | Ownership Boundary |
| --- | --- |
| [UWS](https://github.com/OpenUdon/uws) | Public workflow semantics, UWS versions, schema, parsing, validation, and Go model. |
| [apitools](https://github.com/OpenUdon/apitools) | API source metadata discovery, import/materialization, search, indexing, summaries, auth/security summaries, catalog metadata, protocol-to-UWS-source-type mapping, and operation ranking for OpenAPI, Google Discovery, and AWS Smithy sources. |
| [tfconfig](https://github.com/OpenUdon/tfconfig) | Static Terraform/OpenTofu configuration parsing used by `openudon convert tf`. |
| `udon` | Private UWS/OpenAPI compiler and runtime executor. OpenUdon invokes it only through the trusted run-config handoff. |
| `symphony` | Optional work orchestration, isolated workspaces, reviewer routing, managed state transitions, identity, and audit persistence. |
| n8n / `../try-n8n` | Service-priority and workflow-pattern evidence for SaaS authoring. OpenUdon does not import or execute n8n workflows. |
| [OpenW8M](https://github.com/OpenUdon/openw8m) | Public OpenAPI-backed infrastructure authoring and planning. It is not an OpenUdon compatibility gate while the API source metadata boundary is active. |

## Rule Of Thumb

- Public workflow semantics belong in UWS.
- Generic execution or compilation behavior belongs in executor implementations such as `udon`.
- API source metadata search, discovery, import/materialization, first-class provider catalog
  metadata, and operation metadata belong in apitools.
- OpenUdon may expose thin `openudon catalog ...` wrappers for authoring and package-local API
  source import or materialization, but catalog data stays advisory and does not change workflow
  semantics.
- Static Terraform/OpenTofu parsing belongs in `github.com/OpenUdon/tfconfig`.
- Terraform/OpenTofu provider execution, state, plan/apply, refresh, imports, and cloud SDK calls
  stay outside OpenUdon.
- n8n workflows are evidence for authoring, not an execution or compatibility target.
- Symphony-owned workflow management stays in Symphony.
- Live SaaS provider calls are outside build, assess, iCoT, eval, and dry-run release demos.
- OpenUdon owns project templates, examples, review state, approval templates, package contents,
  package digests, and local trusted-runner enforcement.

Concrete infrastructure authoring and `.tf` generation are outside OpenUdon's scope.
