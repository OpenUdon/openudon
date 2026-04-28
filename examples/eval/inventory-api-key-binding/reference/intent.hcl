openapi = "openapi/inventory.yaml"

workflow {
  name        = "inventory_api_key_binding"
  description = "Fetch inventory availability using a runtime credential binding."
}

input "sku" {
  type     = "string"
  required = true
}

step "get_inventory" {
  type      = "http"
  do        = "Fetch inventory availability for the requested SKU."
  operation = "getInventory"
  with = {
    sku     = "inputs.sku"
    api_key = "inventory_api_key"
  }
}

output "availability" {
  from = "get_inventory.received_body"
}
