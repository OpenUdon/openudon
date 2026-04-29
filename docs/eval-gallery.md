# Eval Sample Gallery

The eval corpus under `examples/eval` is intentionally curated. Each sample should demonstrate a
specific workflow behavior or failure class rather than merely adding volume.

| Sample | Purpose |
| --- | --- |
| `cmd-allowed-deploy` | Approved `cmd` runtime for a sandbox deployment status command. |
| `cmd-disallowed-deploy` | Negative runtime-policy coverage for disallowed command execution. |
| `crm-note-write` | Side-effectful write operation with trusted-runner and sandbox policy. |
| `cursor-pagination-report` | Cursor pagination, bearer security, response cursor extraction, and local report rendering. |
| `customer-export-two-pages` | Multi-step pagination and merge-style function handling. |
| `inventory-api-key-binding` | Credential binding names for API-key-style request parameters. |
| `order-fulfillment-chain` | Multi-service OpenAPI chain with per-service credentials, response extraction, request-body construction, and a sandbox write. |
| `paginated-list` | Simple OpenAPI list operation with bounded request parameters. |
| `profile-boundary-manifest` | Future runtime/profile boundary coverage: renders a local manifest with `fnct` instead of inventing SQL, SSH, or `x-udon-*` profile semantics. |
| `runtime-only-render` | No-OpenAPI runtime-only `fnct` rendering workflow. |
| `support-email` | API lookup plus approved side-effectful email adapter and safety boundary. |
| `support-priority-routing` | Function-backed classification/routing with explicit contracts. |
| `timeout-idempotency-controls` | Runtime-only workflow with explicit workflow timeout, step timeout, and workflow idempotency metadata. |
| `weather-toronto` | Hidden technical step expansion from city lookup to weather lookup. |

## Adding Samples

Add samples only when they strengthen coverage:

- Prefer one clear purpose per sample.
- Include `reference/intent.hcl` when reference comparison should detect drift.
- Add `reference/policy.json` when the reference is illustrative or needs per-fixture triage notes.
  Use `mode: "strict"` for golden references and `mode: "advisory"` when deterministic quality
  gates are authoritative but exact intent shape may drift.
- Treat step names, output names, request literal names, and bind field names as semantic hints by
  default. They should help diagnose drift, but they are not release-blocking by themselves.
- Treat wrong runtime type, wrong selected OpenAPI operation, and reference parse/compare failures
  as behavioral drift. These are blocking unless a fixture policy deliberately overrides them.
- Keep `max_blocking` at `0` unless the fixture is intentionally tracking a temporary known gap.
- Keep secret-shaped values fake and avoid real provider data.
- Document credential bindings by name only.
- For side-effectful workflows, include approval/trusted-runtime policy and sandbox/test proof-run
  policy.
- Keep the corpus curated; grow size only after current samples remain stable.
