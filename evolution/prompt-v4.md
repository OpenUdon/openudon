# Prompt V4 - Fnct Helper Authoring Boundary

OpenUdon should author public pure `fnct` helper selectors without committing to
runtime execution. Use Gmail raw-message rendering as the first case: preserve a
workflow-local render step, select `gmail.render_raw`, pass helper fields in
the request body, and let a trusted runtime register and execute the helper.

Keep side-effectful Gmail send as the HTTP/API step, and keep recipient
selection explicit through workflow inputs when the user only says “to me”.
