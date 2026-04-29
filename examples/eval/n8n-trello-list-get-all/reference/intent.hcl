openapi = "openapi/trello.json"

workflow {
  name        = "n8n_trello_list_get_all"
  description = "Represent the n8n Trello list get-all workflow as a Ramen intent workflow."
}

input "boardId" {
  type     = "string"
  required = true
}

input "limit" {
  type     = "integer"
  required = true
}

step "list_board_lists" {
  type      = "http"
  do        = "List Trello lists on a board."
  operation = "listTrelloBoardLists"
  with = {
    id    = "inputs.boardId"
    limit = "inputs.limit"
  }
}

output "lists" {
  from = "list_board_lists.received_body"
}
