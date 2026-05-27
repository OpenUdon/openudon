source = "asyncapi/streetlights-mqtt.yml"

workflow {
  name        = "asyncapi_streetlights_mqtt"
  description = "Publish a reviewed Streetlights MQTT dim-light command using a package-local AsyncAPI source."
}

input "streetlight_id" {
  type     = "string"
  required = true
}

input "dim_percentage" {
  type     = "number"
  required = true
}

input "sent_at" {
  type     = "string"
  required = true
}

input "app_header_value" {
  type     = "string"
  required = true
}

step "dim_streetlight" {
  type      = "http"
  do        = "Publish the Streetlights MQTT dim-light command."
  source    = "asyncapi/streetlights-mqtt.yml"
  operation = "dimLight"
  with = {
    "path.streetlightId"    = "inputs.streetlight_id"
    "body.percentage"      = "inputs.dim_percentage"
    "body.sentAt"          = "inputs.sent_at"
    "header.my-app-header" = "inputs.app_header_value"
  }
}

output "dim_result" {
  from = "dim_streetlight.received_body"
}
