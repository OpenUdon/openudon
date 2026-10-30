workflow {
  name        = "report"
  description = "Preview a report"
}
step "report" {
  type = "pending"
  pending {
    purpose = "Obtain a report"
    effect  = "read"
    inputs  = "{\"type\":\"object\"}"
    outputs = "{\"type\":\"object\",\"properties\":{\"text\":{\"type\":\"string\"}}}"
  }
  do = "Obtain a report"
}
