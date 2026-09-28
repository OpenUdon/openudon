# M89 Stage 1 source-provisioning contract

The approved direction is implemented and verified in OpenUdon M89. M89.1
through M89.4 are recorded as task-level commits on local `main`. Publication
of the exact M89 implementation and Kinet W04's exact-revision handoff remain
pending; M89.5 is still in progress.

APItools M79 is pinned at commit
`e3625f6ef52ea54b7f78b7a4a4f1993bf8a06a46` and module
`v0.0.0-20260928033144-e3625f6ef52e`. `step source add` copies selected local
documents and a digest-bound source manifest atomically. Its response exposes
both the caller-selected manifest ID and the path-derived candidate source ID.
Explicit nested and renamed input/output mappings are checked against source
metadata; Kinet retains output mapping evidence outside UWS intent. Unknown
effects and unsupported root extensions remain fail-closed.

Full repository tests, the OpenUdon boundary and artifact checks, the checkout
built Kinet consumer check, and a disposable KINET_HOME add/bind/check probe
pass against the implementation. M89.5 remains pending publication, followed
by Kinet W04's exact-revision reconciliation. No workflow or account mutation
was performed.
