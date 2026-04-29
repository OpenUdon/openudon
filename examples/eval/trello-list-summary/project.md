# Trello List Summary

## Goal

Fetch lists for one Trello-style board and summarize the returned array locally.

## Inputs

- `boardId`: required string containing the board identifier.

## Outputs

- `list_summary`: rendered list summary payload.

## Data Flow

- Pass `inputs.boardId` to `list_board_lists`.
- Bind `list_board_lists.received_body.lists` into `summarize_lists`.

## External Systems and OpenAPI

- Trello-like Lists API: use `openapi/trello.yaml`.
- The selected OpenAPI operation is `listBoardLists`.

## Runtime Policy

- `openapi` and `http` are allowed for the Lists API.
- `fnct` is allowed for local list summarization.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `summarize_lists`
  - Inputs: lists.
  - Outputs: list summary payload.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Board-list lookup is read-only and may run against sandbox data after approval.

## Fallback Behavior

- Stop if board lists cannot be fetched.
