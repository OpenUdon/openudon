workflow {
  name        = "m28_order_fulfillment_chain"
  description = "M28 sample: check customer and inventory details before creating a sandbox fulfillment order."
}

input "customerId" {
  type     = "string"
  required = true
}

input "sku" {
  type     = "string"
  required = true
}

input "quantity" {
  type     = "integer"
  required = true
}

step "get_customer" {
  type      = "http"
  do        = "Fetch the customer profile and default shipping address."
  openapi   = "openapi/customers.yaml"
  operation = "getCustomer"
  with = {
    customerId    = "inputs.customerId"
    Authorization = "customers_bearer_token"
  }
}

step "check_inventory" {
  type      = "http"
  do        = "Check inventory availability and preferred warehouse for the SKU."
  openapi   = "openapi/inventory.yaml"
  operation = "getInventory"
  with = {
    sku               = "inputs.sku"
    inventory_api_key = "inventory_api_key"
  }
}

step "create_fulfillment_order" {
  type       = "http"
  do         = "Create a sandbox fulfillment order request body from customer and inventory data."
  openapi    = "openapi/orders.yaml"
  operation  = "createFulfillmentOrder"
  depends_on = ["get_customer", "check_inventory"]
  with = {
    quantity      = "inputs.quantity"
    Authorization = "orders_bearer_token"
  }
  bind {
    from = "get_customer"
    fields = {
      customerEmail     = "received_body.email"
      shippingAddressId = "received_body.defaultShippingAddress.id"
    }
  }
  bind {
    from = "check_inventory"
    fields = {
      sku         = "received_body.sku"
      warehouseId = "received_body.preferredWarehouseId"
    }
  }
}

output "order_confirmation" {
  from = "create_fulfillment_order.received_body.confirmationNumber"
}
