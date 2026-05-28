# source = "graphql/graphiql-starwars-schema.graphql"
# http "get_hero"

  uws = "1.4.0"
  info {
    title       = "graphql_starwars_hero"
    description = "Read the reviewed Star Wars hero field using a package-local GraphQL schema artifact."
    version     = "1.0.0"
  }
  sourceDescription "graphiql_starwars_schema" {
    url  = "graphql/graphiql-starwars-schema.graphql"
    type = "graphql"
  }
  operation "get_hero" {
    sourceDescription = "graphiql_starwars_schema"
    sourceOperationId = "query.hero"
    description       = "Read the Star Wars hero field."
  }
  workflow "main" {
    type        = "sequence"
    description = "Read the reviewed Star Wars hero field using a package-local GraphQL schema artifact."
    outputs = {
      hero_result = "get_hero.received_body"
    }
    step "get_hero" {
      description  = "Read the Star Wars hero field."
      operationRef = "get_hero"
    }
  }