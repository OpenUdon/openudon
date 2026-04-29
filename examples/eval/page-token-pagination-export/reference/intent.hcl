openapi = "openapi/devices.yaml"

workflow {
  name        = "page_token_pagination_export"
  description = "Fetch two page-token-paginated pages of device inventory and merge them into one export."
}

step "list_devices_first_page" {
  type      = "http"
  do        = "Fetch the first page of devices."
  operation = "listDevices"
  with = {
    pageSize      = "100"
    Authorization = "devices_bearer_token"
  }
}

step "list_devices_second_page" {
  type       = "http"
  do         = "Fetch the second page of devices using the next page token."
  operation  = "listDevices"
  depends_on = ["list_devices_first_page"]
  with = {
    pageSize      = "100"
    Authorization = "devices_bearer_token"
  }
  bind {
    from = "list_devices_first_page"
    fields = {
      pageToken = "received_body.nextPageToken"
    }
  }
}

step "merge_device_pages" {
  type       = "fnct"
  do         = "Merge device records from both pages."
  depends_on = ["list_devices_first_page", "list_devices_second_page"]
  bind {
    from = "list_devices_first_page"
    fields = {
      page_1 = "received_body.devices"
    }
  }
  bind {
    from = "list_devices_second_page"
    fields = {
      page_2 = "received_body.devices"
    }
  }
}

output "inventory_export" {
  from = "merge_device_pages.received_body"
}
