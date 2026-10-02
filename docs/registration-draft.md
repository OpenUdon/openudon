# Structural registration drafts

`openudon authoring registration-draft --request FILE|- [--at UTC_RFC3339]`
turns reviewed structural registration definitions into a canonical native
profile. This command starts no browser, writes no package, calls no model,
and grants no capture, import, package or runtime approval.

The strict request version is `openudon.registration-draft.v1`. Requests and
results are bounded to 256 KiB. Unknown or duplicate JSON fields, trailing
JSON, unsafe origins/navigation, invalid credential symbols and incomplete
native evidence are refused with a fixed diagnostic. The request contains:

- `start`: `profile_version` (`1.0`, `1.1` or `1.2`) and approved `origins`;
- `draft`: public title/provider/confidence/expiry, symbolic credential slots,
  typed input slots, a flow with observed candidates, reviewed call controls,
  and explicit success proof posture;
- `observation`, and for typed profiles `history` and `previews`: exact reduced
  native observation records. Never substitute credential values, cookies,
  raw page bodies or private browser state.

The result preserves the exact request digest, canonical `profile`, selected
`candidate_ids`, symbolic `credential_bindings`, and `disclosure`, including
retained structural query parameters and typed step/candidate relationships.
Absent `--at`, the evidence time is current UTC. A deferred success proof is
explicitly operator reviewed; it is never presented as an observed success.

Use the shared `internal/registrationdraft` builder for in-process consumers.
The public `browser-capture` command independently validates the resulting
profile against its own current native observation/history/preview and issued
review card. Import remains a separate exact decision. Ordinary browser
package authoring and package promotion remain separate approvals; registration
runtime submission requires its own trusted approval. The builder cannot confer
any of these permissions.

This replaces the former OpenUdon registration UI's field-definition renderer.
Kinet owns the interactive presentation and user decisions. Historical `.icot`
records and profile versions retain their original meanings.
