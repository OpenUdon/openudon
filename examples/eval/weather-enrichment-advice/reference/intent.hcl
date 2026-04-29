openapi = "openapi/weather.yaml"

workflow {
  name        = "weather_enrichment_advice"
  description = "Fetch current weather and render local advice from the response."
}

input "city" {
  type     = "string"
  required = true
}

step "get_weather" {
  type      = "http"
  do        = "Fetch current weather for one city."
  operation = "getWeather"
  with = {
    city = "inputs.city"
  }
}

step "render_weather_advice" {
  type       = "fnct"
  do         = "Render local advice from the weather response."
  depends_on = ["get_weather"]
  bind {
    from = "get_weather"
    fields = {
      city      = "received_body.city"
      tempC     = "received_body.tempC"
      condition = "received_body.condition"
    }
  }
}

output "advice" {
  from = "render_weather_advice.received_body"
}
