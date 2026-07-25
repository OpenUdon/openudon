# Security Policy

## Reporting

Report suspected vulnerabilities through GitHub's private security-advisory
feature for the OpenUdon/openudon repository. Do not include credentials,
access tokens, live provider responses, customer identifiers, workflow input
values, or exploitable details in a public issue.

If private reporting is unavailable, open a public issue containing only a
request for a private contact channel.

## Supported Versions

The latest v0.1.x release receives security fixes on a best-effort basis.
Unreleased commits and older pre-1.0 lines are not maintained as separate
security branches.

## Security Boundary

OpenUdon treats project briefs, API source documents, UWS files, generated
artifacts, approvals, run configs, and executor evidence as untrusted until
validated. Authoring, build, assess, eval, and dry-run commands do not perform
live provider operations. Credential values remain in the trusted operator or
executor environment and must not be placed in projects, prompts, approvals,
release evidence, examples, logs, or issue reports.

OpenUdon is pre-1.0 software and is provided without an operational
response-time or support SLA.
