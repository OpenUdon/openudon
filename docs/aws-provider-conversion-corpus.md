# AWS Provider Conversion Corpus Discovery

This note records the first pass over the sibling
`../terraform-provider-aws` acceptance-test corpus for OpenUdon Terraform /
OpenTofu conversion hardening.

The pass stayed static:

- no Terraform/OpenTofu commands were run;
- no provider plugins were installed or loaded;
- no AWS credentials, state, plan, apply, refresh, or live API calls were used;
- Terraform snippets were copied from provider acceptance-test config helpers
  into scratch directories under `/tmp/openudon-aws-provider-corpus`.

## Source Corpus

The sibling checkout contains 3,888 Go test files under
`../terraform-provider-aws/internal/service`.

Good initial tests were selected because they are small, provider-shaped, and
exercise distinct converter behavior:

| Case | Provider source | Terraform shape | OpenAPI shape |
|---|---|---|---|
| `s3-bucket-accelerate` | `internal/service/s3/bucket_accelerate_configuration_test.go`, `TestAccS3BucketAccelerateConfiguration_basic`, `testAccBucketAccelerateConfigurationConfig_basic` | `aws_s3_bucket` plus `aws_s3_bucket_accelerate_configuration` | single S3 OpenAPI document |
| `s3-bucket-data-source` | `internal/service/s3/bucket_data_source_test.go`, `TestAccS3BucketDataSource_basic`, `testAccBucketDataSourceConfig_basic` | `aws_s3_bucket` plus `data.aws_s3_bucket` | single S3 OpenAPI document |
| `lambda-function-url` | `internal/service/lambda/function_url_test.go`, `TestAccLambdaFunctionURL_basic`, `testAccFunctionURLConfig_basic`, `testAccFunctionURLConfig_base` | `data.aws_partition`, IAM role, IAM inline policy, Lambda function, Lambda function URL | multiple IAM, Lambda, and STS OpenAPI documents |

OpenAPI inputs came from APIs.guru:

- `amazonaws.com/s3/2006-03-01/openapi.yaml`
- `amazonaws.com/iam/2010-05-08/openapi.yaml`
- `amazonaws.com/lambda/2015-03-31/openapi.yaml`
- `amazonaws.com/sts/2011-06-15/openapi.yaml`

`../apitools` imported the S3 document successfully. IAM, Lambda, and STS were
downloaded directly after `apitools import` rejected them as not looking like
OpenAPI/Swagger even though the raw files start with `openapi: 3.0.0`.

## Commands

Single-document S3 resource case:

```bash
go run ../openudon/cmd/openudon convert tf \
  --config-dir /tmp/openudon-aws-provider-corpus/cases/s3-bucket-accelerate \
  --openapi s3=/tmp/openudon-aws-provider-corpus/openapi/s3.yaml \
  --action create \
  --out /tmp/openudon-aws-provider-corpus/out/s3-bucket-accelerate
```

Single-document S3 data-source case:

```bash
go run ../openudon/cmd/openudon convert tf \
  --config-dir /tmp/openudon-aws-provider-corpus/cases/s3-bucket-data-source \
  --openapi s3=/tmp/openudon-aws-provider-corpus/openapi/s3.yaml \
  --action create \
  --out /tmp/openudon-aws-provider-corpus/out/s3-bucket-data-source
```

Multi-document Lambda/IAM/STS case:

```bash
go run ../openudon/cmd/openudon convert tf \
  --config-dir /tmp/openudon-aws-provider-corpus/cases/lambda-function-url \
  --openapi iam=/tmp/openudon-aws-provider-corpus/openapi/iam.yaml \
  --openapi lambda=/tmp/openudon-aws-provider-corpus/openapi/lambda.yaml \
  --openapi sts=/tmp/openudon-aws-provider-corpus/openapi/sts.yaml \
  --action create \
  --out /tmp/openudon-aws-provider-corpus/out/lambda-function-url
```

Each non-strict conversion produced `project.md`, `workflows/intent.hcl`,
`workflows/workflow.hcl`, `workflows/workflow.uws.yaml`, plan, diagnostics,
review, and quality artifacts.

Strict mode failed as expected:

| Case | Strict result | First diagnostics |
|---|---:|---|
| `s3-bucket-accelerate` | failed with 2 diagnostics | ambiguous matches for `aws_s3_bucket.test` and `aws_s3_bucket_accelerate_configuration.test` |
| `s3-bucket-data-source` | failed with 3 diagnostics | ambiguous `aws_s3_bucket.test`, ambiguous read for `data.aws_s3_bucket.test`, unresolved list for `data.aws_s3_bucket.test` |
| `lambda-function-url` | failed with 5 diagnostics | ambiguous IAM role, IAM role policy, Lambda function; unresolved partition data-source read/list |

## First Failures

### M13: single OpenAPI AWS operation ranking

The S3 OpenAPI document contains real operations that are likely intended for
the selected provider resources:

- `CreateBucket`
- `PutBucketAccelerateConfiguration`
- `GetBucketAccelerateConfiguration`
- `GetBucketLocation`
- `ListBuckets`

The converter still emitted deterministic TODO operation IDs for the S3
resource and data-source cases. The high-value gap is AWS-aware operation
ranking that can map Terraform AWS resource names and data source names to
operation IDs without crossing into provider execution.

M13 follow-up added committed OpenUdon regression coverage for the two S3 cases
and mapped:

- `aws_s3_bucket` create to `CreateBucket`
- `aws_s3_bucket_accelerate_configuration` create to
  `PutBucketAccelerateConfiguration`
- `data.aws_s3_bucket` read to `GetBucketLocation`

The regression keeps quality expectations review-first: these cases must not
fail because of unavailable TODO operation IDs or conversion diagnostics, while
request-field and credential binding gaps remain review evidence for later AWS
protocol work.

### M14: multiple OpenAPI documents and AWS protocol details

The Lambda case partially matched `aws_lambda_function_url.test` to
`CreateFunctionUrlConfig`, but the generated plan still had unresolved review
gaps:

- `FunctionName` is a required path parameter and was not bound from Terraform
  `function_name`.
- the operation requires `hmac` security, but the generated package has no
  auditable SigV4 credential binding.
- `aws_lambda_function.test` remained a TODO instead of `CreateFunction`.
- IAM resources remained TODOs even though the IAM document contains
  `GET_CreateRole`, `POST_CreateRole`, and related AWS query-protocol
  operations.
- `data.aws_partition.current` is provider-local metadata and should not be
  forced into IAM OpenAPI read/list TODOs.
- `apitools import` rejected valid-looking AWS query-protocol OpenAPI documents
  for IAM, Lambda, and STS.

These failures are registered as M14 because they require multi-document service
selection plus AWS protocol-specific request and credential handling.

M14 follow-up added committed OpenUdon regression coverage for the Lambda
Function URL case using local IAM, Lambda, and STS OpenAPI fixtures. The
converter now:

- maps `aws_iam_role` create to `POST_CreateRole`;
- maps `aws_iam_role_policy` create to `POST_PutRolePolicy`;
- maps `aws_lambda_function` create to `CreateFunction`;
- maps `aws_lambda_function_url` create to `CreateFunctionUrlConfig`;
- preserves `data.aws_partition.current` as provider-local metadata without an
  OpenAPI read/list operation;
- binds Terraform `function_name` to the Lambda URL `FunctionName` path
  parameter while retaining `body.terraform.*` review facts;
- emits a symbolic `aws_hmac` credential binding for AWS SigV4/hmac security.

The same milestone also narrowed the `../apitools` import fix to downloaded and
cached AWS APIs.guru-style OpenAPI 3 documents whose primary typed validation
fails, while keeping external-reference rejection in place.

M14 review follow-up closed the remaining real-document gaps:

- AWS query-protocol `Action` and `Version` constants are now bound for IAM and
  STS operations selected by converter mappings.
- `data.aws_caller_identity` is mapped to STS `POST_GetCallerIdentity` instead
  of being treated as provider-local metadata.
- SigV4 credential bindings preserve AWS provider aliases, such as
  `aws_west_hmac`, instead of collapsing all aliases into `aws_hmac`.
- The loose AWS OpenAPI import fallback is scoped to APIs.guru
  `amazonaws.com` sources rather than arbitrary downloaded documents.

The regenerated strict Lambda/IAM/STS package using the real downloaded
OpenAPI documents passed OpenUdon quality with no failing checks.
