# n8n Google Drive File Upload

## Goal

Represent the n8n Google Drive file upload workflow as a Ramen intent workflow that uploads one file.

## Inputs

- `name`: required string containing the uploaded file name.
- `data`: required string naming the binary input field or encoded test content.

## Outputs

- `file`: Google Drive upload response body returned by the upload step.

## Data Flow

- Pass `inputs.name` to `upload_file.name`.
- Pass `inputs.data` to `upload_file.data`.
- Return `upload_file.received_body` as `file`.

## External Systems and OpenAPI

- Google Drive API: use `openapi/google_drive.json`.
- The selected OpenAPI operation is `uploadFile`.
- The OpenAPI document is copied from `../w8m/reducibility/specs/google_drive.json`.

## Runtime Policy

- `openapi` and `http` are allowed for the Google Drive API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need binary envelope preparation.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `google_drive_oauth_token` for Google Drive authentication when this workflow is approved for execution.
- Do not include OAuth tokens, real file IDs, or production file content in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Do not run the Google Drive upload workflow from this eval fixture.
- Uploading a real file is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox drives or test endpoints for any future proof run.

## Fallback Behavior

- Stop if the Google Drive OpenAPI document is unavailable.
- Stop if `name` or `data` is unavailable.
- Stop if trusted execution approval is missing.
