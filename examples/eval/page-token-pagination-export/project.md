# Page Token Pagination Export

## Goal

Fetch two page-token-paginated pages of device inventory and merge them into one export.

## Inputs

- No inputs are required.

## Outputs

- `inventory_export`: merged device inventory export.

## Data Flow

- Fetch the first device page without a page token.
- Bind `list_devices_first_page.received_body.nextPageToken` to `list_devices_second_page.pageToken`.
- Bind both device arrays into `merge_device_pages`.

## External Systems and OpenAPI

- Device Inventory API: use `openapi/devices.yaml`.
- The selected OpenAPI operation is `listDevices`.

## Runtime Policy

- `openapi` and `http` are allowed for device inventory lookup.
- `fnct` is allowed for local merge rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `merge_device_pages`
  - Inputs: page_1 and page_2 device arrays.
  - Outputs: merged inventory export.
  - Side effects: none.

## Credentials and Secrets

- Use credential binding name `devices_bearer_token`.
- Do not include tokens or production device data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox device inventory for proof runs.

## Fallback Behavior

- Stop if either device page cannot be fetched.
