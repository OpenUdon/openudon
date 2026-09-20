# Evolution Result v3

OpenUdon accepts a thin first-class provider catalog integration above
`apitools/catalog`.

The OpenUdon CLI may expose operator-facing commands for catalog listing,
provider inspection, advisory output, security reports, and package-local
OpenAPI import. These commands are authoring and review assistance only.

Catalog maintainer workflows such as source refresh, refresh reports, stats,
security audits, and catalog curation remain in `../apitools`.

OpenUdon imports only actual provider-owned OpenAPI references into
`examples/<name>/openapi/`. Native Google Discovery, AWS Smithy, Dropbox Stone,
and human-docs catalog entries remain advisory metadata until an OpenAPI
document is supplied or generated through an upstream-owned path.
