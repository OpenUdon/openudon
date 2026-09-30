# Explicit per-step executor handoff

Stage 4's approved OpenUdon M90 consumes Udon M44's payload-free report v5
through the external CLI boundary. Explicit selection alone adds run-evidence
v3 with exact attempt/workflow/complete inventory binding; legacy execution and
report/evidence defaults remain unchanged. Missing or rejected evidence cannot
prove unstarted steps. Observe incomplete durable inventories conservatively;
retry decisions remain downstream, under fresh approval.

Qualify only against the accepted frozen private source/build closure and a
provider-free disposable loopback service. Keep UWS 1.11 and all package/approval
and browser gates unchanged. Publishing the reviewed source requires separate
origin/main authority; no runtime adoption, hosted execution or target account
operation is promoted by this work.
