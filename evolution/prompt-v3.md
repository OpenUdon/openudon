# Evolution Prompt v3

OpenUdon should consume the now-expanded `../apitools` first-class provider
catalog directly during authoring.

`apitools` has provider-source-first catalog coverage across official OpenAPI,
Google Discovery, AWS Smithy, Dropbox Stone, official docs-derived overlays,
and security-overlay metadata. OpenUdon should expose this as operator-facing
authoring assistance so users can inspect provider metadata and import
provider-owned OpenAPI documents without searching APIs.guru first.

The integration must keep boundaries intact:

- catalog metadata is advisory;
- local `openapi/` files and explicit OpenAPI inputs remain authoritative;
- Discovery, Smithy, Stone, and human-docs records are not treated as OpenAPI;
- no provider operations are executed;
- no credentials, account selection, token acquisition, or request signing
  belongs in OpenUdon catalog commands.
