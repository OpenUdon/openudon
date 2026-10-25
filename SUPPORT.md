# OpenUdon support policy

Current OpenUdon provides supported public Go libraries for reviewed UWS package
construction, trust metadata, exact approvals, handoff and evidence verification.
The [public API contract](docs/public-trust-api.md),
[package v3 contract](docs/package-v3.md) and
[browser supplement](docs/browser-package-v3.md) define their scope.

A32 retires the legacy `openudon` and `udon-runner` executables, private authoring
and browser harnesses, and CLI archive publication. Existing published releases,
consumer pins and historical artifacts remain unchanged. See the
[retirement guide](docs/legacy-cli-retirement.md) for command ownership.
Versioned public API, wire and history-reader contracts remain supported.

Kinet owns the authoring product, isolated workers and current execution
confirmation. Private Udon owns execution. Producer byte verification supplies
no permission or runtime qualification.

Questions and reproducible bugs may be filed in GitHub Issues. Support is
community best effort; no uptime, response-time, provider-availability, or
operational SLA is provided. Report vulnerabilities through the
[security policy](SECURITY.md), keeping private data out of public reports.
