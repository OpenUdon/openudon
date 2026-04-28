# Eval Sample Gallery

The eval corpus under `examples/eval` is intentionally curated. Each sample should demonstrate a
specific workflow behavior or failure class rather than merely adding volume.

| Sample | Purpose |
| --- | --- |
| `cmd-allowed-deploy` | Approved `cmd` runtime for a sandbox deployment status command. |
| `cmd-disallowed-deploy` | Negative runtime-policy coverage for disallowed command execution. |
| `crm-note-write` | Side-effectful write operation with trusted-runner and sandbox policy. |
| `customer-export-two-pages` | Multi-step pagination and merge-style function handling. |
| `inventory-api-key-binding` | Credential binding names for API-key-style request parameters. |
| `paginated-list` | Simple OpenAPI list operation with bounded request parameters. |
| `runtime-only-render` | No-OpenAPI runtime-only `fnct` rendering workflow. |
| `support-email` | API lookup plus approved side-effectful email adapter and safety boundary. |
| `support-priority-routing` | Function-backed classification/routing with explicit contracts. |
| `weather-toronto` | Hidden technical step expansion from city lookup to weather lookup. |

## Adding Samples

Add samples only when they strengthen coverage:

- Prefer one clear purpose per sample.
- Include `reference/intent.hcl` when reference comparison should detect drift.
- Keep secret-shaped values fake and avoid real provider data.
- Document credential bindings by name only.
- For side-effectful workflows, include approval/trusted-runtime policy and sandbox/test proof-run
  policy.
- Keep the corpus curated; grow size only after current samples remain stable.
