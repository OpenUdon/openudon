# A31 source publication

Named STG11_SOURCE_PUBLICATION normal fast-forward origin refs/heads/main now
contains exact cleanup code5a3bba865452da78f0ca64322116386b21c7a50a and clean
qualified artifact source39f4bfdfd830a6497f163336fe34e6b82f475c5e. Publication head
05f4aa010080213dda8a002c86f11d720f7fffb7 is independently observed by ls-remote
and fetch; merge-base verifies both exact sources are ancestors. Destination
 git@github.com-tabilet:OpenUdon/openudon.git is unchanged. Publication commits
use [skip ci] to avoid the automatic website deployment workflow. No hosted-CI
qualification, website/host deployment or live operation is claimed.

Clean standalone offline Go1.26.6/GOWORK=off build reproduces CLI SHA
29d0e70a1ed5e17252f035f66e7adbb88c697347051fdece5291840d99ef685a twice, with
actualVCSmodified=false. Separate udon-runner SHA
d99adb4d944c5ec20690587c9374f8481ee3c1e11ae7d37c149611f47a837a5a is retained.
[a31-clean-build.json](a31-clean-build.json) records exact modules/locks/source.
Public APIs/wires and Kinet P09 worker SDK/source/image pins remain unchanged.
No implicit new SDK adoption follows this source publication.

The cleanup removes one dead unexported adapter; all six live legacy packages,
CLI/browser/HCL consumers and frozen Ramen/W8M/browser contracts remain. Precise
Stage12 deletion gates in [transition inventory](a31-transition-cleanup.md) and
[complete machine graph](a31-consumer-inventory.json) are handed to Kinet M48.
Required owner/public/consumer fixtures and pre-publication whole review1 pass;
final whole review2/normal retirement remain required before accepted handoff.
