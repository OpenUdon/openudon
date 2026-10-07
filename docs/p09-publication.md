# P09 source publication and ordinary consumers

Named STG11_SOURCE_PUBLICATION pushed normal origin refs/heads/main to
 da6653145c8624f862e3c21adb413345d0cb4be2 at unchanged destination
 git@github.com-tabilet:OpenUdon/openudon.git. Read-only ls-remote and fetch
independently observed that full head; merge-base verifies exact qualified
source a6a3ef010fe27f277f8191204c1c81ea1cc0334b is reachable. Last publication
commit uses [skip ci] to avoid the existing docs-deployment workflow; source
publication grants no website/host deployment or hosted-CI qualification.

Ordinary module v0.1.1-0.20261007094531-a6a3ef010fe2 independently resolves to
that full origin hash. Module sum h1:Tomt06dU+DCTEqrTNlDlcFnFZGl/r8fvpIXaSUOwu7s=;
go.mod sum h1:kol8tnW9phAZtwTfHJ2z9Oa8eUnfUYcQIwcxPcyvl4g=;
archive SHA-256 380499a3209b780a62115ee53a922298cb8a25f011d56c103a6738762de219d8.
[Consumer proof](p09-consumer-proof.json) records exact versions/closure manifests.

Public standalone consumer passes package/source/shape verification, exact
plan/broker/approval, read-only history and tamper refusal using the published
SDK, pinned Go 1.26.6, GOWORK=off and 73 selected modules with no replacements.
Private standalone consumer passes actual RuntimeFunctionCatalog reproduction,
Compile/CheckSupported admission, public construction/verification/plan/approval
against the published SDK and exact Udon M48 module. Its complete selected graph
has 171 modules, no directory replacements and the already qualified
 github.com/docker/docker -> github.com/moby/moby v24.0.7+incompatible version
replacement required by M48. The SDK root remains public-only.

Both consumers use synthetic worker identity/time/approval and perform zero
effects. They qualify the ordinary library/adapter source closure, not Kinet's
future real execution-worker binary or host permissions. A single already
selected older x/telemetry module metadata/archive was resolved under exact
build-closure fetching authority so the public full graph could be recorded;
no dependency version was upgraded. Both then pass with GOPROXY=off. Local
bootstrap evidence is superseded by these ordinary published-module results.

P09.5 publication task is complete; whole closing review, downstream exact-source
reconciliation and literal retirement are still required before Kinet adoption.
No deployment/live ledger/provider/model/API/mail/registration change occurred.
