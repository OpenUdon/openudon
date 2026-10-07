# P09 qualification

Qualified implementation source is a6a3ef010fe27f277f8191204c1c81ea1cc0334b.
The clean ordinary clone passed GOWORK=off/GOPROXY=off standalone make check
and full vet using pinned cached Go 1.26.6 with GOTOOLCHAIN=local. No workspace
or directory replacement enters the owner build. The complete selected graph
has 91 modules and 90 hashed ordinary archives, all resolved offline. See
[pinned build](p09-qualified-build.json) and [archive manifest](p09-module-archives.json).

The CLI builds reproduce SHA-256
ffa7dc3be990755da7ee9af1fa9fd08642bb423e4491a9cfae564925270aefdb.
udon-runner SHA-256 is
f089f467142952fc0f7b84e412af41ae59e9cf37e6bc9539776c24a1bb0836eb.
The actual version probe names the full qualified revision, Go 1.26.6 and
vcs.modified=false. Source archive SHA-256 is
1efbe6c5236b7d1ff05ed94bb1f4f2039d02320bb3e9b8ff09bbf380232b6b4e.
These are owner artifacts; Kinet's browser/old runtime/media pins remain frozen.

The public package/schema/API/identity/authority golden tests, all eight native
families, exact numeric input/output contracts, stale/tampered/rehashed source
and report/security claims, private values, missing artifacts, version/cancel/
bounds and read-only history/refusal suites pass. Compatible focused staticcheck
and public package/trust/authority/approval/evidence races pass. Existing M98
surface/wire/schema fixtures remain unchanged. Legacy current-owner regressions
pass with only exact accepted M82/M08 dependency adoption; frozen browser locks
and qualifications are preserved. An actual rootless network-none/read-only
probe retains the separately pinned browser binary, without requalifying capture.

Review 1 found output/reference admission and absent expression-contract proof;
review 2 found dropped restricting parent constraints. Their regressions failed
before fixes and now pass. The complete pre-publication review 3 is recorded in
the active owner status. Final source publication, ordinary published SDK/public
and private consumers and closing milestone acceptance remain pending.

The public SDK never imports private Udon. The preliminary actual private
adapter uses ordinary published Udon M48 da43e57be37af4e18e633558580f740c525f139d,
one local SDK bootstrap and synthetic worker hashes. It independently reproduces
catalogs and calls non-effectful Compile/CheckSupported, public package/plan/
approval checks with zero effects. This is not final SDK/worker qualification.
Kinet M47 owns actual private-worker identity/isolation; Kinet W18/W19 own exact
confirmation/publication/conversion and fresh authority. No source digest,
metadata flag, conversion or audit creates a grant.
