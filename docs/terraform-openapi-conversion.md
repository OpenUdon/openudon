# Terraform/OpenAPI Conversion Design

This document is the durable long-form design reference for converting
Terraform/OpenTofu configuration plus OpenAPI documents into reviewable OpenUdon
UWS package artifacts.

This page is the public conversion contract. Maintainer-only planning files may
exist in local checkouts, but the published behavior and release expectations
are captured here and in [Release Stewardship](release-stewardship.md).

## Goal

M7 specified the adapter contract, M8 implemented the initial `openudon convert
tf` path, M9 integrated package-local OpenAPI staging and handoff artifacts, and
M10 keeps the path stewarded through release checks.

## Current Status: Parked

Terraform/OpenTofu conversion is parked as an implemented but non-primary
product track. The existing `openudon convert tf` command, static parser
boundary, diagnostics, package artifact generation, and selected AWS regression
coverage should remain maintained by normal tests and release checks, but broad
Terraform provider coverage is deferred.

The near-term OpenUdon product focus is agentic UWS authoring for common SaaS
workflows backed by local OpenAPI documents. n8n and `../try-n8n` are useful
service-priority and workflow-pattern evidence for that path, but n8n workflow
import is deferred. Terraform work should resume only when the project is ready
to invest in a stable provider adapter architecture and a provider-family
corpus.

When resumed, the next Terraform milestone should not make `tfconfig` an
AWS/GCP/Azure dialect engine. `tfconfig` should remain provider-neutral and
continue to emit static Terraform/OpenTofu facts: resources, data sources,
providers, aliases, modules, expressions, source ranges, diagnostics, and
sensitive candidates. Provider semantics belong in an OpenUdon conversion
adapter layer above `tfconfig`.

Future provider support should use a provider-adapter shape:

```text
tfconfig static facts
  -> provider-neutral OpenUdon conversion core
  -> provider adapter: AWS, GCP, Azure, or another provider family
  -> OpenAPI operation mapping, request bindings, credential binding names
  -> normal OpenUdon review package artifacts
```

Planned provider-family direction:

- **AWS**: continue the existing S3/IAM/Lambda/STS corpus; grow mappings from
  selected `../terraform-provider-aws` acceptance-test snippets; keep SigV4,
  query-protocol `Action`/`Version`, provider alias credential names, and
  provider-local metadata in the adapter layer.
- **GCP**: add a future adapter for Google provider resources and data sources
  after identifying stable Google API/OpenAPI inputs, project/location/zone
  conventions, service account credential bindings, and provider alias behavior.
- **Azure**: add a future adapter for AzureRM-style resources and data sources
  after identifying ARM OpenAPI operation IDs, API-version request bindings,
  subscription/resource-group/location conventions, and tenant/subscription
  credential binding names.

Any future AWS/GCP/Azure adapter must stay review-first. It must not execute
Terraform/OpenTofu, provider plugins, cloud SDKs, live cloud APIs, state, plan,
apply, refresh, imports, or credential resolution. Unsupported or ambiguous
provider behavior should remain deterministic diagnostics and review TODOs.

Support a command shaped like:

```bash
openudon convert tf \
  --config-dir ./tf \
  --openapi app=./openapi/app.yaml \
  --action create \
  --out ./.openudon/convert
```

The command should produce review scaffolding, not Terraform-compatible
execution. Ambiguous or unsupported behavior must become diagnostics and TODOs.
Release stewardship, unsupported behavior review, and regression coverage are
tracked in [release-stewardship.md](release-stewardship.md).

## Command Integration

M8 should add an `openudon convert tf` command group in
`../openudon/cmd/openudon/main.go`. The CLI entrypoint should only parse flags,
format user-facing errors, and route to reusable adapter logic in a new
`../openudon/internal/tfconvert` package.

`internal/tfconvert` should own the conversion contract:

- load Terraform/OpenTofu static facts through `tfconfig`;
- load OpenAPI operation indexes through `apitools`;
- apply target, action, mapping, diagnostic, redaction, and deterministic output
  policy;
- write draft review artifacts.

The package boundary matters because later OpenUdon package generation, tests,
and quality gates should reuse the adapter without shelling out to the CLI.

## Static Parser Boundary

The parser belongs in `github.com/OpenUdon/tfconfig`.

`tfconfig` should:

- load Terraform/OpenTofu configuration directories;
- support `.tf`, `.tofu`, `.tf.json`, `.tofu.json`, and override files;
- support multiple files per module;
- preserve variables, locals, outputs, providers, provider aliases, required
  providers, resources, data sources, lifecycle, dependencies, moved/import/
  removed/check/test blocks;
- preserve `count`, `for_each`, expressions, references, source ranges, and
  diagnostics;
- load local modules that are already present on disk;
- report missing or remote modules as diagnostics;
- mark likely secret literals as sensitive candidates and keep public JSON from
  emitting raw likely-secret values;
- emit deterministic `tfconfig.static.v1` output.

`tfconfig` must not:

- run provider plugins;
- run `tofu init`;
- download modules;
- initialize backends;
- load state;
- refresh, plan, or apply;
- resolve credentials.

OpenUdon must consume the `tfconfig.LoadDir` Go API from
`github.com/OpenUdon/tfconfig`. The deterministic CLI JSON projection remains
an export/debug boundary and must not be the M8 integration contract unless a
later milestone explicitly promotes it.

## Multi-File, Provider, And Module Support

Real Terraform/OpenTofu projects commonly split configuration across files such
as `versions.tf`, `providers.tf`, `variables.tf`, `main.tf`, and `outputs.tf`.
The converter must treat all eligible files in one directory as one module.

Multiple providers and aliases must be preserved as symbolic review facts:

- `terraform.required_providers`;
- provider local names;
- source addresses and version constraints;
- provider aliases;
- resource/data-source provider references;
- module `providers = { ... }` mappings.

Local module support is v1 scope. Remote or unavailable modules are diagnostics.

## CLI Contract

The M8 CLI contract is:

```bash
openudon convert tf \
  [--config-dir DIR] \
  --openapi ID=PATH \
  [--openapi ID=PATH ...] \
  [--action create|update|delete|replace] \
  [--target ADDRESS ...] \
  [--out DIR] \
  [--strict]
```

Flag behavior:

- `--config-dir` defaults to `.` and identifies the root Terraform/OpenTofu
  configuration directory passed to `tfconfig.LoadDir`.
- `--openapi` is repeatable. Each value must be `id=PATH`; IDs are required,
  non-empty, unique, stable, and used in diagnostics and review output. Paths
  are local OpenAPI or Swagger documents. Missing, unreadable, malformed, or
  duplicate-ID inputs produce diagnostics.
- `--action` accepts `create`, `update`, `delete`, or `replace`. It is required
  whenever the selected Terraform facts include at least one managed resource.
  It is not required when the selected facts contain only data sources. M8 may
  reject a provided action outside the allowed set before conversion begins.
- `--target` is repeatable. Without targets, all loaded managed resources and
  data sources are selected. With targets, only exact address matches are
  selected; unmatched targets are deterministic diagnostics.
- `--out` defaults to `./.openudon/convert`.
- `--strict` exits non-zero when any strict-failure diagnostic remains.

## OpenAPI Input Boundary

The OpenUdon adapter must use only the OpenAPI-first API metadata `apitools` APIs:

- `BuildOperationInventory` to load local OpenAPI/Swagger inputs;
- `NewOperationIndex` to validate and index operation IDs from an inventory;
- `SortedOperationSummaries` to produce deterministic candidate lists;
- `SelectOperationByHints` for prompt-safe operation selection;
- `AuthRequirementsForOperation` and `AuthRequirementsForOperations` for
  symbolic credential and configuration summaries.

The adapter must not reintroduce old `apitools` lifecycle APIs, review package
builders, LLM clients, binding contracts, or trusted-runner concepts. OpenUdon
must keep passing `openudon check-apitools-boundary`, and `make check` should
continue to include the apitools boundary target.

OpenAPI documents are sorted by CLI ID before loading and before output. The
provided ID becomes the stable document key in generated review notes,
diagnostics, and operation references.

The adapter loads and indexes each OpenAPI ID independently:

- call `BuildOperationInventory` with one `InventoryDocument` whose `Name` is
  the CLI ID and whose `Path` is the local document path;
- call `NewOperationIndex` for that one-document inventory;
- keep indexes in a map keyed by OpenAPI ID.

Duplicate operation IDs are fatal only within the same OpenAPI ID because
`NewOperationIndex` indexes bare operation IDs. The same operation ID may appear
in different OpenAPI documents. Generated operation references therefore use
the namespaced shape `{openapi_id, operation_id}`. Candidate lists are built by
iterating OpenAPI IDs in sorted order, then each document's
`SortedOperationSummaries(index.OperationIDs)` output.

If implementation needs package-local OpenAPI inputs, M8 may copy or reference
them under `openapi/`; M7 does not require that artifact.

## Target Filtering

Targets match exact adapter canonical addresses derived from `tfconfig` module
and object addresses. `tfconfig` stores `Module.Address` separately from each
resource or data source address, so the adapter constructs a full target address
before matching:

- root managed resource: `resource.Address`;
- root data source: `dataSource.Address`;
- child module managed resource: `module.Address + "." + resource.Address`;
- child module data source: `module.Address + "." + dataSource.Address`.

Examples:

- root managed resource: `aws_instance.web`;
- root data source: `data.aws_ami.base`;
- child module managed resource: `module.child.example_child.main`;
- nested child module data source:
  `module.child.module.grandchild.data.example_item.selected`.

Target matching is exact string equality after trimming surrounding whitespace.
The adapter must not perform fuzzy matching, globbing, implied module expansion,
or provider-type inference for targets. Targets select only managed resources
and data sources, not module calls, provider configs, outputs, moved blocks, or
imports.

Unmatched targets produce diagnostics with the original target text. Matching
targets, selected modules, resources, and data sources are emitted in
`tfconfig` canonical order. Diagnostics for unmatched targets are sorted by
target address.

## Mapping Contract

The OpenUdon adapter consumes `tfconfig.static.v1` through the `tfconfig` Go API
plus OpenAPI operation indexes from `apitools`. CLI JSON remains an export/debug
format unless a later milestone promotes it as an integration boundary.

### Managed Resources

Managed resources represent potential side effects and require a global
`--action` when any managed resource is selected. The adapter must not infer
side effects from resource type names, OpenAPI methods, Terraform lifecycle
blocks, provider defaults, or previous state.

Action mapping:

| Action | Draft scaffold purpose |
|---|---|
| `create` | Create a remote object from symbolic Terraform attributes. |
| `update` | Update an existing remote object using symbolic identity and attributes. |
| `delete` | Delete an existing remote object using symbolic identity. |
| `replace` | Composite review scaffold with separate delete and create operation slots. |

If no confident OpenAPI operation exists, the resource still appears in draft
review scaffolding with a deterministic TODO operation ID. Missing or ambiguous
matches are diagnostics, not hidden assumptions.

`replace` is not passed to `SelectOperationByHints` as an operation purpose
because current `apitools` classifiers do not classify operations as `replace`.
For `--action replace`, the adapter builds one delete-purpose candidate slot and
one create-purpose candidate slot for the same Terraform resource. Each slot may
resolve to a namespaced OpenAPI operation reference or produce its own
deterministic TODO. Provider-specific single-operation replacement remains a
review TODO unless a later milestone adds an explicit mapping input.

### Data Sources

Data sources map only to read/list candidate operations. They never require
`--action` and must not produce create, update, delete, or replace scaffolds.

If a data source cannot be mapped confidently to a read/list operation, it
appears as review scaffolding with an unresolved operation TODO. The adapter
must preserve the data source address, provider reference, symbolic arguments,
and source range so a reviewer can decide whether to keep, rewrite, or remove
the scaffold.

### Provider Bindings

Provider config addresses become symbolic OpenUdon binding names. The binding
name is normalized from the `tfconfig` provider address:

- `provider.aws` -> `aws`;
- `provider.aws.west` -> `aws_west`;
- other provider local names and aliases use the same lower-case,
  identifier-safe normalization.

Aliased providers remain distinct. Provider requirements, source addresses,
version constraints, resource/data-source provider references, and module
provider mappings remain review facts. The adapter must not resolve provider
credentials, endpoints, regions, accounts, tenants, or auth flows from the
environment.

### Symbolic Values

The adapter preserves variables, locals, outputs, references, expressions,
`count`, `for_each`, `depends_on`, lifecycle facts, module inputs, and module
provider mappings as symbolic review text. Literal values may be carried only
when they are not sensitive and do not look secret-like.

The adapter must not invent runtime values. Unknowns, expressions, unresolved
references, dynamic counts, and dynamic `for_each` values remain symbolic and
may produce TODO diagnostics when they block confident OpenAPI request mapping.

### OpenAPI Operation Matching

For each selected data source or managed-resource action, the adapter builds
operation-selection hints from prompt-safe facts:

- provider local name or normalized provider binding;
- action purpose: `read`, `list`, `create`, `update`, or `delete`;
- Terraform address, type, name, and attribute names;
- OpenAPI ID.

Candidates are sorted deterministically by OpenAPI ID and
`SortedOperationSummaries`, then selected with `SelectOperationByHints`. A
single confident match becomes a draft operation reference shaped as
`{openapi_id, operation_id}`. No match produces an unresolved operation TODO. An
ambiguous match produces an ambiguity TODO listing deterministic namespaced
candidate operation references.

TODO operation IDs are deterministic and derived from address, purpose, and
action. They must not include timestamps, random suffixes, process IDs, or
machine-specific paths. A conventional shape is:

```text
todo.<normalized-address>.<purpose>.<action>
```

For data sources, the action segment should be `read` or `list`. For
`--action replace`, the adapter emits separate delete and create TODO IDs using
the same target address.

## Redaction And Sensitive Inputs

`tfconfig` sensitive values and sensitive candidates become sensitive OpenUdon
inputs or review variables. Generated artifacts must never include raw
secret-like literal values.

Redaction behavior:

- preserve the source address, attribute path, and redaction reason;
- replace the literal with a symbolic sensitive variable name in generated
  review scaffolding;
- emit a review TODO when a reviewer must confirm the redaction or binding;
- avoid writing secret-like literal defaults, examples, comments, or Markdown
  prose.

## Diagnostics And Strict Mode

Diagnostics are emitted to JSON and Markdown. JSON records use stable fields:

| Field | Meaning |
|---|---|
| `code` | Stable machine-readable code. |
| `severity` | `info`, `warning`, or `error`. |
| `message` | Human-readable diagnostic text. |
| `address` | Terraform resource, data-source, provider, module-call, or target address when applicable. |
| `module_address` | `tfconfig` module address when applicable. |
| `source_range` | Source path/range from `tfconfig` or OpenAPI input when available. |
| `todo_id` | Deterministic TODO identifier when the diagnostic creates reviewer work. |
| `strict_failure` | Boolean indicating whether `--strict` turns this diagnostic into command failure. |

Strict mode fails on:

- parser error diagnostics from `tfconfig`;
- missing, unreadable, malformed, or duplicate-ID OpenAPI documents;
- unmatched `--target` values;
- selected managed resources without a valid required `--action`;
- unresolved operation TODOs;
- ambiguous operation matches;
- unreviewed sensitive redaction TODOs.

Non-strict mode still emits the same diagnostics and review TODOs but may exit
successfully if artifact writing succeeds.

Diagnostics and TODOs are sorted by `code`, `address`, `module_address`, and
`todo_id`, with empty fields sorted before non-empty fields. Markdown mirrors
the JSON order.

## Determinism

Generated output must be stable for review:

- sort OpenAPI documents by ID;
- sort targets by exact address;
- preserve modules, resources, data sources, provider configs, variables,
  locals, outputs, diagnostics, and source files in `tfconfig` canonical order;
- sort provider bindings by normalized binding name;
- sort operation candidates by OpenAPI ID then operation ID;
- sort diagnostics and TODOs by stable fields;
- use relative, normalized paths under the output directory where possible;
- avoid timestamps, random values, absolute temp paths, hostnames, usernames,
  environment-derived credentials, and network fetches.

The converter must be static and local. It must not run provider plugins,
`tofu init`, module downloads, backend initialization, state loading, refresh,
plan, apply, or OpenAPI operations.

## Draft Output

M8 should write the draft layout below by default under `./.openudon/convert`:

```text
.openudon/convert/
  project.md
  workflows/
    intent.hcl
  expected/
    diagnostics.json
    diagnostics.md
    review.md
```

Artifact roles:

- `project.md` summarizes selected modules, resources, data sources, provider
  bindings, OpenAPI IDs, action policy, and unresolved review work.
- `workflows/intent.hcl` contains symbolic draft intent suitable for later
  OpenUdon package generation, clearly marked as unapproved review scaffolding.
- `expected/diagnostics.json` contains the stable diagnostic records.
- `expected/diagnostics.md` contains reviewer-friendly diagnostics in the same
  deterministic order.
- `expected/review.md` explains operation mappings, redactions, provider
  bindings, symbolic values, and TODOs.

M8 may copy or reference OpenAPI inputs under `openapi/` only if the
implementation needs package-local inputs. The initial contract does not require
embedding OpenAPI documents into the draft output.

The target output converges on normal OpenUdon package artifacts:

```text
workflows/workflow.hcl
workflows/workflow.uws.yaml
expected/plan.json
expected/plan.md
expected/discovery.json
expected/review.md
expected/quality.json
expected/quality.md
expected/symphony-handoff.json
```

Generated artifacts remain unapproved until normal OpenUdon review, quality,
approval, digest, and trusted-runner handoff checks pass.

## M8 Implementation Notes

Implement M8 in `../openudon` as a thin CLI route plus reusable
`internal/tfconvert` package. Keep package tests focused on deterministic
conversion from in-memory `tfconfig` facts and `apitools` inventories, then add
CLI tests for flag parsing, target errors, strict failures, and output paths.

The implementation should keep OpenUdon free of direct OpenTofu internals and
should rely on `openudon check-apitools-boundary` to prevent regressions to old
`apitools` lifecycle APIs.
