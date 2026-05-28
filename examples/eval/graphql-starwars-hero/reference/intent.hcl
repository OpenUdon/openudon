source = "graphql/graphiql-starwars-schema.graphql"
workflow {
  name        = "graphql_starwars_hero"
  description = "Read the reviewed Star Wars hero field using a package-local GraphQL schema artifact."
}
step "get_hero" {
  type      = "http"
  do        = "Read the Star Wars hero field."
  source    = "graphql/graphiql-starwars-schema.graphql"
  operation = "query.hero"
}
output "hero_result" {
  from = "get_hero.received_body"
}
