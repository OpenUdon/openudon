# Browser Status Read Eval

## Goal

Read the reviewed status banner from a web page whose provider exposes no
adequate API capability.

## Inputs

- `item`: required status item name.

## Outputs

- Reviewed status text.

## External Systems and OpenAPI

OpenAPI: none required

- The UI-only capability is defined by `browser-profiles/status.json`.
- API sources remain preferred if a later reviewed API exposes the same action.

## Runtime Policy

- `browser` is allowed only through the verified Browsertools profile and the
  trusted Udon browser driver boundary.
- Browser sessions, credentials, cookies, raw DOM, screenshots, and driver
  configuration are not OpenUdon artifacts.

## Data Flow

- Pass `inputs.item` to the reviewed `read_status` browser action.
- Return `read.received_body.status` as the workflow output.

## Function Contracts

- No function runtime is required.

## Credentials and Secrets

- None.

## Safety and Approval Boundary

- Read-only browser action; generate and validate artifacts only.

## Fallback Behavior

- Stop cleanly if the reviewed browser action cannot be verified or executed.
