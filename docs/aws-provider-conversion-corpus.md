# AWS Provider Conversion Corpus Migration

The AWS Terraform/OpenTofu conversion corpus has moved to Ramen.

OpenUdon no longer runs Terraform/OpenTofu conversion fixtures as active gates.
Ramen owns provider/resource operation mapping tests, including AWS Smithy
preference, OpenAPI fallback, S3, IAM, STS, Lambda, request-binding hints, and
credential-binding diagnostics.

Run the migrated checks from the Ramen checkout:

```bash
go test ./...
```
