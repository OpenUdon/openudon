# iCoT Transcript Format

iCoT writes an ignored local transcript to `<example>/.icot/transcript.json` unless `--no-transcript`
or `--print` is used. The transcript is local review/debug evidence; it is not required for
`openudon build`, and it should not contain secrets.

```json
{
  "version": "openudon.icot-transcript.v1",
  "time_utc": "2026-05-23T00:00:00Z",
  "turns": [
    {
      "label": "Workflow goal",
      "answer": "Get weather in Toronto and Gmail me the report."
    }
  ],
  "events": [
    {
      "kind": "catalog_plan_call",
      "data": {
        "candidates": ["gmail:google-discovery/gmail-discovery-v1.json"]
      }
    },
    {
      "kind": "draft_flow_review_result",
      "data": {
        "issues": [
          {
            "severity": "warning",
            "code": "llm_flow_review_disconnected_report",
            "message": "Gmail does not consume the report body.",
            "slot": "steps.gmail.with.raw",
            "evidence": "raw is not bound to the render step"
          }
        ]
      }
    }
  ],
  "session": {
    "project": {},
    "intent": {}
  }
}
```

Fields:

- `version`: transcript wire name; current value is `openudon.icot-transcript.v1`.
- `time_utc`: write time in UTC.
- `turns`: local prompt labels and answers, including automatically accepted defaults in
  `normal` and `fast` mode.
- `events`: structured stage observations such as catalog planning, request-mapping drafts,
  operation-detail requests, confirmation edits, and advisory flow-review results.
- `session`: the final iCoT session snapshot used to render `project.md` and `intent.hcl`.

Transcript events may include operation IDs, API document paths, symbolic credential binding names,
and LLM warning evidence. They must not include credential values or pasted secrets.
