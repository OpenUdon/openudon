# openapi = "asyncapi/streetlights-mqtt.yml"
# http "dim_streetlight"

  uws = "1.3.0"
  info {
    title       = "asyncapi_streetlights_mqtt"
    description = "Publish a reviewed Streetlights MQTT dim-light command using a package-local AsyncAPI source."
    version     = "1.0.0"
  }
  sourceDescription "streetlights_mqtt" {
    url  = "asyncapi/streetlights-mqtt.yml"
    type = "asyncapi"
  }
  operation "dim_streetlight" {
    sourceDescription = "streetlights_mqtt"
    sourceOperationId = "dimLight"
    description       = "Publish the Streetlights MQTT dim-light command."
    request {
      path "streetlightId" {
        __dollar__expr = "variables.inputs.streetlight_id"
      }
      body "percentage" {
        __dollar__expr = "variables.inputs.dim_percentage"
      }
      body "sentAt" {
        __dollar__expr = "variables.inputs.sent_at"
      }
      header "my-app-header" {
        __dollar__expr = "variables.inputs.app_header_value"
      }
    }
  }
  workflow "main" {
    type        = "sequence"
    description = "Publish a reviewed Streetlights MQTT dim-light command using a package-local AsyncAPI source."
    outputs = {
      dim_result = "dim_streetlight.received_body"
    }
    step "dim_streetlight" {
      description  = "Publish the Streetlights MQTT dim-light command."
      operationRef = "dim_streetlight"
    }
  }