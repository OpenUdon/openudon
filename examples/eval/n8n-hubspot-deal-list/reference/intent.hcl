openapi = "openapi/hubspot.json"

workflow {
  name        = "n8n_hubspot_deal_list"
  description = "Represent the n8n HubSpot deal get-many workflow as a Ramen intent workflow."
}

input "limit" {
  type     = "integer"
  required = true
}

step "list_deals" {
  type      = "http"
  do        = "List HubSpot deals."
  operation = "listDeals"
  with = {
    limit = "inputs.limit"
  }
}

output "deals" {
  from = "list_deals.received_body"
}
