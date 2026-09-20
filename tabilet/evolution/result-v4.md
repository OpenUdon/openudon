# Result V4 - Gmail Render Helper Authoring

OpenUdon now treats `gmail.render_raw` as a public pure helper selector for
weather-to-Gmail workflows.

iCoT finalization preserves `render_weather_report` as the local transform
step, selects the helper function, declares `recipient_email`, and maps Gmail
send `raw` from `render_weather_report.received_body`. Synthesis recognizes the
helper descriptor and emits request-body invocation without
`x-uws-runtime.arguments`.
