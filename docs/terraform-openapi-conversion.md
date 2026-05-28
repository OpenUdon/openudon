# Terraform/API Source Conversion Migration

Terraform/OpenTofu conversion has moved out of OpenUdon and into
`github.com/OpenUdon/ramen`.

OpenUdon no longer exposes `openudon convert tf`, imports
`github.com/OpenUdon/tfconfig`, or carries Terraform/OpenTofu conversion
fixtures as active gates. Ramen owns static Terraform/OpenTofu facts,
provider/resource-to-operation mappings, API source binding, diagnostics, and
UWS-facing conversion artifacts.

Use:

```bash
go run ../ramen/cmd/ramen convert tf \
  --config-dir path/to/tf \
  --api-source openapi:example=path/to/openapi.yaml \
  --action create \
  --out ./.ramen/convert
```

OpenUdon remains responsible for UWS authoring, review evidence, package
approval, package digests, and trusted-runner handoff.
