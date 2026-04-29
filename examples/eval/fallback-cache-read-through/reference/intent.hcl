openapi = "openapi/flags.yaml"

workflow {
  name        = "fallback_cache_read_through"
  description = "Fetch a feature flag from the primary API and prepare a cached fallback value when the primary value is unavailable."
}

input "flagKey" {
  type     = "string"
  required = true
}

step "get_primary_flag" {
  type      = "http"
  do        = "Fetch the feature flag from the primary API."
  operation = "getFeatureFlag"
  with = {
    flagKey       = "inputs.flagKey"
    Authorization = "flags_bearer_token"
  }
}

step "prepare_cached_fallback" {
  type       = "fnct"
  do         = "Prepare a cached fallback candidate for the feature flag."
  depends_on = ["get_primary_flag"]
  with = {
    flagKey = "inputs.flagKey"
  }
  bind {
    from = "get_primary_flag"
    fields = {
      primary = "received_body"
    }
  }
}

step "select_flag_result" {
  type       = "fnct"
  do         = "Select the primary flag result or cached fallback."
  depends_on = ["get_primary_flag", "prepare_cached_fallback"]
  bind {
    from = "get_primary_flag"
    fields = {
      primary = "received_body"
    }
  }
  bind {
    from = "prepare_cached_fallback"
    fields = {
      fallback = "received_body"
    }
  }
}

output "flag_result" {
  from = "select_flag_result.received_body"
}
