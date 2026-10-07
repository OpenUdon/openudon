# M98 source publication

The confirmed STG11_SOURCE_PUBLICATION grant authorized only normal owner main
source/closure publication. Origin remains
`git@github.com-tabilet:OpenUdon/openudon.git`. No tags, force update, release
upload, website deployment or live operation was invoked.

Normal push moved main from the recorded baseline
`7cd7fbb837fb87e1ca4abea2a362790b0f434188` to
`642fddbcf960ff5d23edace6cbdc4d1202e1e291`. An independent `ls-remote` query
confirmed that exact head. It contains qualified implementation
`08a3839f357ec40c7e50c8668e4bd7c8d86bb55a` and passed pre-publication review 2.
The final commit uses `[skip ci]` to prevent the unrelated main docs workflow's
`mkdocs gh-deploy --force`; local qualification is recorded independently and
no hosted CI result is claimed. Release is tag-triggered; browser public
qualification is manual, and neither was invoked.

Independent ordinary module resolution/download reports:

- Version: `v0.1.1-0.20261007040813-08a3839f357e`.
- Origin: `https://github.com/OpenUdon/openudon`, full hash
  `08a3839f357ec40c7e50c8668e4bd7c8d86bb55a`.
- Module sum: `h1:Gn8HnUc5gPa7DnSTEnF2LbaRS5y/5NWfiiqVxutNy/g=`.
- go.mod sum: `h1:gYKkottLX/IoTptmggqMl1dMHIi/cg+Tg01OCXhzcGs=`.

The independent published consumer passes with `GOWORK=off GOPROXY=off`,
explicit Go 1.26.6 / `GOTOOLCHAIN=local`, 57 selected modules, and no replacements.
It exercises all eight public packages without internal/private imports.
Its snapshot identity is
`c964e68fbf42dae2418fc790f0744623b1d5fa7bb25f7416f0be5ef50a1f9621`,
and malformed report input remains an invalid/unknown observation. This
consumer is narrower than the 90-module owner build; the complete owner
closure remains in [qualification](m98-qualification.md).

The outside-module download helper auto-selected Go 1.26.8. No qualification
artifact uses that helper selection: owner builds and the published consumer
were explicitly pinned to cached Go 1.26.6. No source, dependency version or
operator tool configuration was upgraded.

Closing review 3, acceptance and consumer reconciliation passed; the complete record is [retired M98](../tabilet/docs/history/status-M98.md). Source closure publication is recorded below after independent observation.
