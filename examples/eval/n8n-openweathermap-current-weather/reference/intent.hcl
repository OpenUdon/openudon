openapi = "openapi/openweathermap.json"

workflow {
  name        = "n8n_openweathermap_current_weather"
  description = "Represent the n8n OpenWeatherMap current weather workflow as a OpenUdon intent workflow."
}

input "cityName" {
  type     = "string"
  required = true
}

input "language" {
  type     = "string"
  required = true
}

step "get_current_weather" {
  type      = "http"
  do        = "Fetch current weather for the requested city."
  operation = "getOpenWeatherMapCurrentWeather"
  with = {
    q    = "inputs.cityName"
    lang = "inputs.language"
  }
}

output "weather" {
  from = "get_current_weather.received_body"
}
