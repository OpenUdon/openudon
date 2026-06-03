# Related Projects

OpenUdon is the integration layer above several sibling projects. Keep changes in the repo that owns
the relevant behavior.

| Project | Ownership Boundary |
| --- | --- |
| [UWS](https://github.com/OpenUdon/uws) | Public workflow semantics, UWS versions, schema, parsing, validation, and Go model. |
| [apitools](https://github.com/OpenUdon/apitools) | API source metadata discovery, import/materialization, search, indexing, summaries, auth/security summaries, catalog metadata, protocol-to-UWS-source-type mapping, and operation ranking for OpenAPI, Google Discovery, AWS Smithy, AsyncAPI, GraphQL, OpenRPC, gRPC/protobuf, and OData sources. |
| [Ramen](https://github.com/OpenUdon/ramen) | Public API-source desired-state conversion, reconciliation, state, graphing, planning, drift, import, apply/delete, and audit artifacts. |
| `udon` | Private UWS/OpenAPI compiler and runtime executor. OpenUdon invokes it only through the trusted run-config handoff. |
| n8n / `../try-n8n` | Service-priority and workflow-pattern evidence for SaaS authoring. OpenUdon does not import or execute n8n workflows. |
| [OpenW8M](https://github.com/OpenUdon/openw8m) | Public OpenAPI-backed infrastructure authoring and planning. It is not an OpenUdon compatibility gate while the API source metadata boundary is active. |

## Rule Of Thumb

- Public workflow semantics belong in UWS.
- Generic execution or compilation behavior belongs in executor implementations such as `udon`.
- API/event source metadata search, discovery, import/materialization, first-class provider catalog
  metadata, and operation metadata belong in apitools.
- OpenUdon may expose thin `openudon catalog ...` wrappers for authoring and package-local API
  source import or materialization, but catalog data stays advisory and does not change workflow
  semantics.
- Ramen owns desired-state reconciliation and Ramen-specific run/audit behavior. OpenUdon may review
  or package UWS-facing artifacts generated elsewhere, but it must not import Ramen.
- n8n workflows are evidence for authoring, not an execution or compatibility target.
- External workflow orchestration stays outside OpenUdon.
- Live SaaS provider calls are outside build, assess, iCoT, eval, and dry-run release demos.
- OpenUdon owns project templates, examples, review state, approval templates, package contents,
  package digests, and local trusted-runner enforcement.
- Command code remains in the product repo that owns the command. Shared helpers should be
  extracted only after stable real overlap is proven.

Concrete infrastructure authoring and desired-state conversion are outside OpenUdon's scope.
