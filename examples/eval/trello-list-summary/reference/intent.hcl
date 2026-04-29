openapi = "openapi/trello.yaml"

workflow {
  name        = "trello_list_summary"
  description = "Fetch board lists and summarize the returned array locally."
}

input "boardId" {
  type     = "string"
  required = true
}

step "list_board_lists" {
  type      = "http"
  do        = "Fetch all lists for one board."
  operation = "listBoardLists"
  with = {
    boardId = "inputs.boardId"
  }
}

step "summarize_lists" {
  type       = "fnct"
  do         = "Summarize the returned board lists."
  depends_on = ["list_board_lists"]
  bind {
    from = "list_board_lists"
    fields = {
      lists = "received_body.lists"
    }
  }
}

output "list_summary" {
  from = "summarize_lists.received_body"
}
