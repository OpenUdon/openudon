openapi = "openapi/inventory.yaml"

workflow {
  name        = "inventory_reorder_decision"
  description = "Fetch inventory for a SKU and render a reorder decision from the returned quantity and threshold."
}

input "sku" {
  type     = "string"
  required = true
}

step "get_inventory" {
  type      = "http"
  do        = "Fetch inventory details for the SKU."
  operation = "getInventory"
  with = {
    sku = "inputs.sku"
  }
}

step "decide_reorder" {
  type       = "fnct"
  do         = "Render a reorder decision from inventory fields."
  depends_on = ["get_inventory"]
  bind {
    from = "get_inventory"
    fields = {
      sku              = "received_body.sku"
      quantity         = "received_body.quantity"
      reorderThreshold = "received_body.reorderThreshold"
    }
  }
}

output "decision" {
  from = "decide_reorder.received_body"
}
