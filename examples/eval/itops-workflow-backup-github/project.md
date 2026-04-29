# IT Ops Workflow Backup to GitHub

## Goal

Back up one n8n workflow definition into a GitHub repository path for audit and recovery.

## Inputs

- `workflowId`: required string identifying the n8n workflow to export.
- `repoOwner`: required string containing the GitHub repository owner.
- `repoName`: required string containing the GitHub repository name.
- `repoPath`: required string containing the target folder path in the repository.

## Outputs

- `backup_commit`: GitHub content update response.

## Data Flow

- Pass `inputs.workflowId` to `get_workflow.workflowId`.
- Pass `inputs.repoOwner` and `inputs.repoName` to GitHub content lookup and update steps.
- Bind exported workflow data into `render_backup_file.workflow`.
- Bind `render_backup_file.received_body.path` and `render_backup_file.received_body.content` into the GitHub update request.
- Bind `get_existing_backup.received_body.sha` into `upsert_backup_file.sha`.
- Pass credential binding `n8n_api_key` to the n8n API security field.

## External Systems and OpenAPI

- n8n API: use `openapi/n8n.yaml`.
- GitHub Contents API: use `openapi/github.yaml`.
- The fixture is inspired by the public n8n workflow-backup template and is modeled as a Ramen-native eval brief.

## Runtime Policy

- `openapi` and `http` are allowed for n8n and GitHub API requests.
- `fnct` is allowed for deterministic backup file rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `render_backup_file`
  - Inputs: exported workflow object, repository path.
  - Outputs: file path and serialized content.
  - Side effects: none.

## Credentials and Secrets

- Use credential binding name `n8n_api_key` for n8n API authentication.
- Use credential binding name `github_token` for GitHub authentication.
- Do not include API keys, GitHub tokens, workflow secrets, or production workflow payloads in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Writing to GitHub is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox n8n instances and test repositories for proof runs.

## Fallback Behavior

- Stop if the n8n workflow cannot be fetched.
- Stop if the GitHub file lookup fails with an unexpected error.
- Stop if trusted execution approval is missing.
