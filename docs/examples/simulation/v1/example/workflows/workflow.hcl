# pending "report"

  uws = "1.12.0"
  info {
    title       = "report"
    description = "Preview a report"
    version     = "1.0.0"
  }
  operation = []
  workflow "main" {
    type        = "sequence"
    description = "Preview a report"
    step "report" {
      description = "Obtain a report"
      pending {
        purpose = "Obtain a report"
        effect  = "read"
        inputs {
          type = "object"
        }
        outputs {
          type = "object"
          properties "text" {
            type = "string"
          }
        }
      }
    }
  }