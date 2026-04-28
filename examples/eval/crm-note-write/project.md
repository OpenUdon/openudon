# CRM Note Write

## Goal

Create an internal note on a CRM contact after a reviewed support interaction.

## Inputs

- `contactId`: required string from the reviewed support interaction.
- `noteText`: required string containing the reviewed note body.

## Outputs

- `note`: created CRM note response.

## External Systems and OpenAPI

- CRM API: use `openapi/crm.yaml`.
- OpenAPI is required for the note creation request.

## Data Flow

- Pass `inputs.contactId` to `create_contact_note.contactId`.
- Pass `inputs.noteText` to `create_contact_note.text`.
- Use visibility `internal`.

## Runtime Policy

- `openapi` and `http` are allowed for the CRM API.
- `cmd` and `ssh` are not allowed.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- The note creation workflow is side-effectful and must only run through an approved trusted runner.
- Use sandbox CRM endpoints for proof runs.

## Fallback Behavior

- Stop if the CRM OpenAPI document is unavailable.
- Stop if `contactId` or `noteText` is unavailable.
