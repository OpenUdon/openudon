# GraphQL Star Wars Hero Eval

## Goal

Read the reviewed Star Wars hero field using a package-local GraphQL schema artifact

## Inputs

- No runtime inputs are required for this provider-free fixture.

## Outputs

- `hero_result`: hero query result from the trusted runtime.

## External Systems and OpenAPI

- Use `graphql/graphiql-starwars-schema.graphql` for GraphQL source metadata.
- OpenAPI: none required.
- OpenUdon validates and packages the GraphQL source-bound workflow, but GraphQL execution belongs to the trusted runtime.

## Runtime Policy

- `openapi` and `http` are allowed for source-bound API operations.
- `fnct`, `cmd`, and `ssh` are not required.

## Function Contracts

- No `fnct` helper contracts are required.

## Data Flow

- Call the reviewed `query.hero` source operation without storing credentials or live endpoint details.

## Credentials and Secrets

- No credential values are stored in this provider-free fixture.
- Do not place GraphQL endpoint credentials, API keys, OAuth tokens, or other secret values in prompts, artifacts, or examples.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Real GraphQL calls require human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the GraphQL source operation is unavailable.
