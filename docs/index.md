# OpenUdon public libraries

OpenUdon supplies public Go contracts for reviewed UWS packages, approval,
executor handoff and run evidence. Kinet owns the authoring product and isolated
worker integration; private Udon owns runtime execution.

- [Public trust APIs](public-trust-api.md)
- [Package v3](package-v3.md)
- [Public browser supplement](browser-package-v3.md)
- [Broker execution handoff](broker-execution-handoff.md)
- [Per-step run evidence](per-step-run-evidence.md)
- [Legacy CLI retirement](legacy-cli-retirement.md)
- [Compatibility](compatibility.md)

The legacy CLI, browser authoring/capture harnesses and image/runner publication
are retired. Earlier guides and frozen examples describe pinned historical
versions. Supported public API/wire/history contracts remain unchanged; actual
execution always requires the consuming host's current authority and admission.
