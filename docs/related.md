# Related Projects

OpenUdon is the integration layer above several sibling projects. Keep changes in the repo that owns
the relevant behavior.

| Project | Ownership Boundary |
| --- | --- |
| [UWS](https://github.com/OpenUdon/uws) | Public workflow semantics, UWS versions, schema, parsing, validation, and Go model. |
| [apitools](https://github.com/OpenUdon/apitools) | OpenAPI discovery, import, search, indexing, summaries, auth/security summaries, and operation ranking. |
| `udon` | Private UWS/OpenAPI compiler and runtime executor. OpenUdon invokes it only through the trusted run-config handoff. |
| `symphony` | Optional work orchestration, isolated workspaces, reviewer routing, managed state transitions, identity, and audit persistence. |
| [OpenW8M](https://github.com/OpenUdon/openw8m) | Public OpenAPI-backed infrastructure authoring and planning. It is not an OpenUdon compatibility gate while the OpenAPI-only apitools boundary is active. |

## Rule Of Thumb

- Public workflow semantics belong in UWS.
- Generic execution or compilation behavior belongs in executor implementations such as `udon`.
- OpenAPI search, discovery, import, and operation metadata belong in apitools.
- Symphony-owned workflow management stays in Symphony.
- OpenUdon owns project templates, examples, review state, approval templates, package contents,
  package digests, and local trusted-runner enforcement.

Concrete infrastructure authoring and `.tf` generation are outside OpenUdon's scope.
