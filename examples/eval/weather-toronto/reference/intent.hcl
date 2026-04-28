openapi = "openapi/weather.yaml"

workflow {
  name        = "weather_toronto"
  description = "Resolve Toronto coordinates and fetch current weather."
}

step "get_coordinates" {
  type      = "http"
  do        = "Resolve Toronto, Canada to coordinates."
  operation = "direct_get"
  with = {
    q = "Toronto,CA"
  }
}

step "get_weather" {
  type       = "http"
  do         = "Fetch weather for the resolved coordinates."
  operation  = "getWeatherData"
  depends_on = ["get_coordinates"]
  with = {
    appid = "weather_appid"
  }
  bind {
    from = "get_coordinates"
    fields = {
      lat = "received_body[0].lat"
      lon = "received_body[0].lon"
    }
  }
}

output "weather" {
  from = "get_weather.received_body"
}
