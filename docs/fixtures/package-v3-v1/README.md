# Candidate package v3 contract fixtures

These synthetic fixtures freeze exact approved YAML/data/source bytes, package,
manifest/assessment/handoff identities, a selected worker/closure plan and the
unchanged broker-authority v1 wire. Worker hashes and authority identities are
synthetic controlled inputs, not actual worker qualification or permission.
The package includes the exact integer 9007199254740993. Field-order/canonical
bytes and raw artifact identities are compared independently in default tests;
all new schemas are compiled against the emitted records.

Regeneration is explicit: GOWORK=off go run ./tools/packagev3-fixtures. Review
changed semantics/hashes before replacing fixtures. Default tests never rewrite
files. The API surface fixture covers packagev3 and shared credentialpolicy;
M98's existing fixture stays frozen. P09 acceptance/publication is recorded in
its owner status/qualification, not inferred from these candidate files.
