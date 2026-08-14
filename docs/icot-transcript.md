# iCoT v2 Transcript Format

iCoT writes ignored local transcript evidence to
`<example>/.icot/transcript.json` unless `--no-transcript` or `--print` is used.
It is not required by `openudon build` and must not contain secrets.

```json
{
  "version": "openudon.icot-transcript.v2",
  "time_utc": "2026-08-14T00:00:00Z",
  "turns": [
    {
      "label": "Which workflow should be active?",
      "answer": "Toronto weather report"
    },
    {
      "label": "Type approve, edit <slot>, explain <assumption-id>, or cancel",
      "answer": "approve"
    }
  ],
  "events": [
    {
      "kind": "frontier_planned",
      "data": {
        "round": 1,
        "nodes": ["boundary.actor_trigger", "boundary.success"]
      }
    },
    {
      "kind": "round_applied",
      "data": {
        "answers": 2
      }
    }
  ],
  "session": {
    "version": "openudon.icot-session.v2",
    "boundary": {},
    "interview": {
      "version": "authoring.interview.v1",
      "nodes": [],
      "evidence": []
    },
    "project": {},
    "intent": {}
  }
}
```

- `version` is `openudon.icot-transcript.v2`.
- `turns` records prompt labels and answers. Normal/fast defaults may be
  recorded even when collection is automatic; the final proposal approval is
  always explicit unless `--yes` was supplied.
- `events` records whole-frontier planning, round application, autosave,
  source discovery, operation detail requests, confirmation edits, advisory
  flow review, and bounded repair attempts.
- `session` is the final v2 session snapshot. Its
  `interview.evidence` collection is the unified public evidence ledger for
  observed facts, decisions, recommendations, assumptions, open decisions,
  deferrals, and inapplicable branches.

Transcript evidence may name operation IDs, source paths/digests, symbolic
credential bindings, candidate workflows, and concise review rationale. It must
not include credential values, pasted secrets, or hidden model
chain-of-thought.
