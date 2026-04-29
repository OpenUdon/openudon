# n8n Trello List Get All

## Goal

Represent the n8n Trello list get-all workflow as a Ramen intent workflow that lists board lists.

## Inputs

- `boardId`: required string containing the Trello board ID.
- `limit`: required integer containing the maximum number of lists to request.

## Outputs

- `lists`: Trello list response body returned by the list step.

## Data Flow

- Pass `inputs.boardId` to `list_board_lists.id`.
- Pass `inputs.limit` to `list_board_lists.limit`.
- Return `list_board_lists.received_body` as `lists`.

## External Systems and OpenAPI

- Trello API: use `openapi/trello.json`.
- The selected OpenAPI operation is `listTrelloBoardLists`.
- The OpenAPI document is copied from `../w8m/reducibility/specs/trello.json`.

## Runtime Policy

- `openapi` and `http` are allowed for the Trello API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need response cleanup.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `trello_api_token` for Trello authentication when this workflow is approved for execution.
- Do not include API tokens, board secrets, or production board data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Use sandbox Trello boards or test endpoints for any future proof run.
- Production execution requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the Trello OpenAPI document is unavailable.
- Stop if `boardId` or `limit` is unavailable.
- Stop if trusted execution approval is missing.
