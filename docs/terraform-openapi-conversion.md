# Terraform/API Source Conversion Design

This document is the durable long-form design reference for converting
Terraform/OpenTofu configuration plus API source documents into reviewable
OpenUdon UWS package artifacts. The file path is intentionally unchanged from
the earlier Terraform/OpenAPI design so existing links remain stable.

This page is the public conversion contract. Maintainer-only planning files may
exist in local checkouts, but the published behavior and release expectations
are captured here and in [Release Stewardship](release-stewardship.md).

## Goal

M7 specified the original adapter contract, M8 implemented the initial
`openudon convert tf` path, M9 integrated package-local source staging and
handoff artifacts, and M10 keeps the path stewarded through release checks.
M13 and M14 added AWS-shaped regression coverage against OpenAPI-derived
fixtures.

The reopened direction is no longer OpenAPI-first. Terraform/OpenTofu
conversion should consume static Terraform facts plus provider-appropriate API
source documents:

- OpenAPI/Swagger remains a supported source family and keeps the current
  `--openapi ID=PATH` compatibility flag.
- AWS uses official Smithy JSON as the primary source family.
- GCP/Google APIs use official Google Discovery as the primary source family.
- Azure remains OpenAPI/Swagger-oriented unless a later milestone selects a
  more authoritative public source family.

## Current Status: Hardened Native-Source Slice

Terraform/OpenTofu conversion is an implemented review-first track. The current
slice supports generic API source inputs, package-local source staging, normal
OpenUdon package artifacts, selected AWS OpenAPI regression coverage, one
native AWS Smithy acceptance fixture, and one native Google Discovery acceptance
fixture.

The track is intentionally narrow. Broad AWS, GCP, Azure, or other provider
coverage should still be added one provider-resource family at a time with
diagnostics/TODOs for unsupported behavior.

The recorded `../tfconfig` prework is complete:

- `.tf` suppression caused by `.tofu` alternatives is reported as a structured
  diagnostic instead of library stderr logging.
- Child-module parse diagnostics remain module-local.
- `ProviderRequirement` preserves `configuration_aliases`.

`tfconfig` must remain provider-neutral. AWS, GCP, Azure, or other
provider-family semantics belong in OpenUdon provider adapters above
`tfconfig`, not in the parser.

## Provider Adapter Shape

Future provider support should use this shape:

```text
Terraform/OpenTofu config directory
  -> tfconfig.LoadDir static facts
  -> provider-neutral OpenUdon conversion core
  -> provider adapter: AWS, GCP, Azure, or another provider family
  -> API source operation metadata from apitools
  -> request bindings, credential binding names, diagnostics, and TODOs
  -> normal OpenUdon review package artifacts
```

OpenUdon owns the conversion CLI, target/action policy, provider-adapter
selection, source operation mapping, package-local API source staging, review
evidence, quality gates, and handoff state. It does not own OpenTofu source
sync, parser internals, provider plugins, state, plan/apply, credential
resolution, or live cloud APIs.

Provider adapters may own:

- provider-family resource and data-source classification;
- mapping from Terraform resource/data-source facts to operation candidates;
- request bindings and symbolic value preservation;
- credential binding naming for provider configs and aliases;
- provider-local metadata classification;
- deterministic diagnostics and review TODOs for unsupported behavior.

Provider adapters must not execute Terraform/OpenTofu, provider plugins, cloud
SDKs, state, plan, apply, refresh, imports, credential resolution, or live
provider APIs.

## First Reopened Acceptance Paths

The first reopened native-source paths are implemented without claiming broad
provider coverage:

- **AWS:** `aws_iam_role` maps against native AWS Smithy metadata.
- **GCP:** `google_storage_bucket` maps against native Google Discovery
  metadata.

Both paths must remain review-first. Unsupported or ambiguous provider behavior
must produce deterministic diagnostics and TODOs, not hidden assumptions or
provider execution.

Azure remains in scope as an OpenAPI/Swagger-oriented provider family, but it is
not a first reopened acceptance path unless M47 or a later milestone explicitly
adds it.

## CLI Direction

The implemented command currently accepts repeatable OpenAPI inputs:

```bash
openudon convert tf \
  --config-dir ./tf \
  --openapi app=./openapi/app.yaml \
  --action create \
  --out ./.openudon/convert
```

Keep this compatibility surface. `--openapi ID=PATH` remains a
backward-compatible alias for an OpenAPI API source document.

The additive source flag is:

```bash
openudon convert tf \
  --config-dir ./tf \
  --api-source aws-smithy:s3=./aws-smithy/s3.json \
  --api-source google-discovery:compute=./google-discovery/compute.v1.json \
  --api-source openapi:arm=./openapi/arm.yaml \
  --action create \
  --out ./.openudon/convert
```

For the first reopened milestone, `KIND` is one of:

| Kind | Source family | Primary use |
|---|---|---|
| `openapi` | OpenAPI or Swagger | Existing compatibility, Azure-oriented inputs, generic APIs. |
| `aws-smithy` | Official AWS Smithy JSON | AWS provider adapters. |
| `google-discovery` | Official Google Discovery | GCP/Google provider adapters. |

Flag behavior for `--api-source KIND:ID=PATH`:

- `KIND` selects the source parser and operation metadata family.
- `ID` is required, non-empty, unique within the conversion run, stable, and
  used in diagnostics and review output.
- `PATH` must be a local source document. Missing, unreadable, malformed, or
  duplicate-ID inputs produce diagnostics.
- `--openapi ID=PATH` is equivalent to `--api-source openapi:ID=PATH`.
- Generated operation references should include both source kind and source ID
  so identical operation names in different documents remain unambiguous.
- Trusted-runner configs should carry staged source documents through
  `api_source_paths`; `openapi_paths` remains a populated compatibility alias
  until all supported executors consume the source-family-neutral field.

The rest of the CLI contract remains:

```bash
openudon convert tf \
  [--config-dir DIR] \
  [--openapi ID=PATH ...] \
  [--api-source KIND:ID=PATH ...] \
  [--action create|update|delete|replace] \
  [--target ADDRESS ...] \
  [--out DIR] \
  [--strict]
```

- `--config-dir` defaults to `.` and identifies the root Terraform/OpenTofu
  configuration directory passed to `tfconfig.LoadDir`.
- `--action` accepts `create`, `update`, `delete`, or `replace`. It is required
  whenever the selected Terraform facts include at least one managed resource.
  It is not required when the selected facts contain only data sources.
- `--target` is repeatable. Without targets, all loaded managed resources and
  data sources are selected. With targets, only exact address matches are
  selected; unmatched targets are deterministic diagnostics.
- `--out` defaults to `./.openudon/convert`. API source staging directories
  under `--out` are pruned only when they carry the `openudon convert tf`
  ownership marker; use a dedicated output directory or remove/relocate
  pre-existing `openapi/`, `aws-smithy/`, or `google-discovery/` directories.
- `--strict` exits non-zero when any strict-failure diagnostic remains.

## Static Parser Boundary

The parser belongs in `github.com/OpenUdon/tfconfig`.

`tfconfig` should:

- load Terraform/OpenTofu configuration directories;
- support `.tf`, `.tofu`, `.tf.json`, `.tofu.json`, and override files;
- support multiple files per module;
- preserve variables, locals, outputs, providers, provider aliases, required
  providers, configuration aliases, resources, data sources, lifecycle,
  dependencies, moved/import/removed/check/test blocks;
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
an export/debug boundary unless a later milestone explicitly promotes it.

## API Source Metadata Boundary

The OpenUdon adapter must use public `github.com/OpenUdon/apitools` metadata
APIs for source loading, operation summaries, auth/security summaries, and
operation ranking. It must not reintroduce old `apitools` lifecycle APIs, review
package builders, LLM clients, binding contracts, or trusted-runner concepts.

Source-family behavior:

- OpenAPI/Swagger inputs use the existing OpenAPI inventory, operation index,
  summary, ranking, and security metadata APIs.
- AWS Smithy inputs use native Smithy metadata exposed through public apitools
  parser/source APIs; adapters should not require Smithy-to-OpenAPI lowering as
  the primary path.
- Google Discovery inputs use native Discovery metadata exposed through public
  apitools parser/source APIs; adapters should not require
  Discovery-to-OpenAPI lowering as the primary path.

OpenUdon must keep passing `openudon check-apitools-boundary`, and `make check`
should continue to include the apitools boundary target.

API source documents are sorted by source kind and source ID before loading and
before output. The provided ID becomes the stable document key in generated
review notes, diagnostics, and operation references.

Duplicate operation IDs are fatal only within the same source document when the
source parser requires unique bare operation IDs. The same operation ID may
appear in different source documents. Generated operation references therefore
use a namespaced shape such as `{source_kind, source_id, operation_id}`.

## Multi-File, Provider, And Module Support

Real Terraform/OpenTofu projects commonly split configuration across files such
as `versions.tf`, `providers.tf`, `variables.tf`, `main.tf`, and `outputs.tf`.
The converter must treat all eligible files in one directory as one module.

Multiple providers and aliases must be preserved as symbolic review facts:

- `terraform.required_providers`;
- provider local names;
- source addresses and version constraints;
- `configuration_aliases`;
- provider aliases;
- resource/data-source provider references;
- module `providers = { ... }` mappings.

Local module support is v1 scope. Remote or unavailable modules are diagnostics.

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
plus API source operation metadata from `apitools`. CLI JSON remains an
export/debug format unless a later milestone promotes it as an integration
boundary.

### Managed Resources

Managed resources represent potential side effects and require a global
`--action` when any managed resource is selected. The adapter must not infer
side effects from resource type names, API source methods, Terraform lifecycle
blocks, provider defaults, or previous state.

Action mapping:

| Action | Draft scaffold purpose |
|---|---|
| `create` | Create a remote object from symbolic Terraform attributes. |
| `update` | Update an existing remote object using symbolic identity and attributes. |
| `delete` | Delete an existing remote object using symbolic identity. |
| `replace` | Composite review scaffold with separate delete and create operation slots. |

If no confident source operation exists, the resource still appears in draft
review scaffolding with a deterministic TODO operation ID. Missing or ambiguous
matches are diagnostics, not hidden assumptions.

`replace` is not passed to operation ranking as a single operation purpose
unless a provider adapter explicitly supports replacement. For `--action
replace`, the default adapter behavior builds one delete-purpose candidate slot
and one create-purpose candidate slot for the same Terraform resource. Each
slot may resolve to a namespaced source operation reference or produce its own
deterministic TODO.

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
- `provider.google` -> `google`;
- `provider.google.europe` -> `google_europe`;
- other provider local names and aliases use the same lower-case,
  identifier-safe normalization.

Aliased providers remain distinct. Provider requirements, source addresses,
version constraints, configuration aliases, resource/data-source provider
references, and module provider mappings remain review facts. The adapter must
not resolve provider credentials, endpoints, regions, accounts, tenants, or
auth flows from the environment.

### Symbolic Values

The adapter preserves variables, locals, outputs, references, expressions,
`count`, `for_each`, `depends_on`, lifecycle facts, module inputs, and module
provider mappings as symbolic review text. Literal values may be carried only
when they are not sensitive and do not look secret-like.

The adapter must not invent runtime values. Unknowns, expressions, unresolved
references, dynamic counts, and dynamic `for_each` values remain symbolic and
may produce TODO diagnostics when they block confident request mapping.

### Operation Matching

For each selected data source or managed-resource action, the adapter builds
operation-selection hints from prompt-safe facts:

- provider local name or normalized provider binding;
- provider family and selected adapter;
- action purpose: `read`, `list`, `create`, `update`, or `delete`;
- Terraform address, type, name, and attribute names;
- API source kind and ID;
- source-family metadata such as AWS service/operation traits, Google
  resource/method names, or OpenAPI operation IDs where available.

Candidates are sorted deterministically by source kind, source ID, and source
operation order. A single confident match becomes a draft operation reference
shaped as `{source_kind, source_id, operation_id}`. No match produces an
unresolved operation TODO. An ambiguous match produces an ambiguity TODO
listing deterministic namespaced candidate operation references.

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
| `source_range` | Source path/range from `tfconfig` or API source input when available. |
| `api_source_kind` | API source kind when applicable. |
| `api_source_id` | API source ID when applicable. |
| `todo_id` | Deterministic TODO identifier when the diagnostic creates reviewer work. |
| `strict_failure` | Boolean indicating whether `--strict` turns this diagnostic into command failure. |

Strict mode fails on:

- parser error diagnostics from `tfconfig`;
- missing, unreadable, malformed, unsupported-kind, or duplicate-ID API source
  documents;
- unmatched `--target` values;
- selected managed resources without a valid required `--action`;
- unresolved operation TODOs;
- ambiguous operation matches;
- unreviewed sensitive redaction TODOs.

Non-strict mode still emits the same diagnostics and review TODOs but may exit
successfully if artifact writing succeeds.

Diagnostics and TODOs are sorted by `code`, `address`, `module_address`,
`api_source_kind`, `api_source_id`, and `todo_id`, with empty fields sorted
before non-empty fields. Markdown mirrors the JSON order.

## Determinism

Generated output must be stable for review:

- sort API source documents by kind and ID;
- sort targets by exact address;
- preserve modules, resources, data sources, provider configs, variables,
  locals, outputs, diagnostics, and source files in `tfconfig` canonical order;
- sort provider bindings by normalized binding name;
- sort operation candidates by source kind, source ID, then operation ID;
- sort diagnostics and TODOs by stable fields;
- use relative, normalized paths under the output directory where possible;
- avoid timestamps, random values, absolute temp paths, hostnames, usernames,
  environment-derived credentials, and network fetches.

The converter must be static and local. It must not run provider plugins,
`tofu init`, module downloads, backend initialization, state loading, refresh,
plan, apply, or API operations.

## Draft Output

The existing draft layout defaults to `./.openudon/convert`:

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
  bindings, API source IDs, action policy, and unresolved review work.
- `workflows/intent.hcl` contains symbolic draft intent suitable for later
  OpenUdon package generation, clearly marked as unapproved review scaffolding.
- `expected/diagnostics.json` contains the stable diagnostic records.
- `expected/diagnostics.md` contains reviewer-friendly diagnostics in the same
  deterministic order.
- `expected/review.md` explains operation mappings, redactions, provider
  bindings, symbolic values, and TODOs.

Current implementation copies API source inputs under their package-local source
directories (`openapi/`, `aws-smithy/`, or `google-discovery/`) when conversion
loads them. The converter writes a root ownership marker so later reruns can
prune stale staged source directories without deleting unrelated pre-existing
content in user-selected output directories.

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
expected/review-handoff.json
```

Generated artifacts remain unapproved until normal OpenUdon review, quality,
approval, digest, and trusted-runner handoff checks pass.

## Implementation Notes

Keep `cmd/openudon` thin and reusable conversion behavior in
`internal/tfconvert`.

The next implementation slices should grow provider adapters and fixtures one
family at a time from the existing `--api-source` and native AWS/GCP baseline.

The implementation should keep OpenUdon free of direct OpenTofu internals and
should rely on `openudon check-apitools-boundary` to prevent regressions to old
`apitools` lifecycle APIs.
