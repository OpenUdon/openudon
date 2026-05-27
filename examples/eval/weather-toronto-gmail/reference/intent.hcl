source = "openapi/openweathermap-one-call-3-overlay.json"
workflow {
  name        = "get_weather_in_toronto"
  description = "get weather in toronto, canada, and send me the report using Google gmail."
}
input "recipient_email" {
  type        = "string"
  description = "Email address that should receive the Gmail weather report."
  required    = true
}
step "geocode_openweathermap_location" {
  type = "http"
  do   = "Resolve toronto, canada to OpenWeatherMap coordinates."
  with = {
    appid                  = "credentials.openWeatherAPIKey"
    open_weather_a_p_i_key = "credentials.openWeatherAPIKey"
    q                      = "toronto, canada"
  }
  provider  = "openweathermap"
  openapi   = "openapi/openweathermap-one-call-3-overlay.json"
  operation = "geocodeOpenWeatherMapLocationName"
}
step "retrieve_weather" {
  type       = "http"
  do         = "Get the current weather for Toronto, Canada using OpenWeatherMap Current Weather."
  depends_on = ["geocode_openweathermap_location"]
  with = {
    appid                  = "credentials.openWeatherAPIKey"
    open_weather_a_p_i_key = "credentials.openWeatherAPIKey"
  }
  provider  = "openweathermap"
  openapi   = "openapi/openweathermap-one-call-3-overlay.json"
  operation = "getOpenWeatherMapCurrentWeather"
  bind {
    from = "geocode_openweathermap_location"
    fields = {
      lat = "received_body[0].lat"
      lon = "received_body[0].lon"
    }
  }
}
step "render_weather_report" {
  type       = "fnct"
  do         = "Render a reviewable local weather report from the weather response before Gmail delivery."
  depends_on = ["retrieve_weather"]
  with = {
    body_template = "Weather report: {{.}}"
    input         = "retrieve_weather.received_body"
    subject       = "Weather report"
    to            = "inputs.recipient_email"
  }
  operation = "gmail.render_raw"
}
step "email_report" {
  type       = "http"
  do         = "Send the weather report to me using Google Gmail."
  depends_on = ["render_weather_report"]
  with = {
    body   = "render_weather_report.received_body"
    raw    = "render_weather_report.received_body"
    userId = "me"
  }
  provider  = "gmail"
  openapi   = "google-discovery/gmail-discovery-v1.json"
  operation = "gmail_users_messages_send"
}
output "result" {
  from        = "render_weather_report.received_body"
  description = "result=email_report.received_body"
}
