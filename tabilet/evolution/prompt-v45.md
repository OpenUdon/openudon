# Stage 1 source provisioning and explicit field mapping

Approved direction, 2026-09-28, in the coordinated remediation sequence
Kinet M07 → APItools M79 → OpenUdon M89 → Kinet W04 → Kinet A04 → Kinet U02.

OpenUdon M89 extends the existing step-authoring boundary so Kinet can
provision user-selected local API documents through `openudon step source add`
after exact user confirmation. OpenUdon owns bounded source copying,
APItools-backed validation, package-local provenance, and the versioned
command contract. It does not fetch source URLs or execute workflows.

Support explicitly proven nested and renamed request/response mappings,
including type, format, requiredness, digest, and source evidence. Preserve
Kinet's output mappings outside UWS intent and require them again for later
checks. Unknown operation effects and unsupported root extensions stay
fail-closed. Keep package-root confinement when Kinet invokes a checkout-built
OpenUdon binary from KINET_HOME or a private staged root.

APItools M79 is the exact upstream prerequisite. Kinet W04 remains downstream
of an accepted, published OpenUdon M89 revision; M89 execution does not
authorize that publication, any external workflow run, or Stage 2 promotion.
