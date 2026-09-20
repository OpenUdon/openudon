# Result V9 - CLI-First Public Beta

OpenUdon now has a defined v0.1 CLI and versioned-artifact compatibility
boundary, local build metadata, public support and security policies, and
tag-driven release automation.

The standalone module pins the published Authoring lifecycle package and the
current public UWS revision. Public CI rejects local replacements, downloads
dependencies with workspace mode disabled, runs deterministic gates, and
cross-builds `openudon`, `icot`, and `udon-runner` for six platform targets.

The release workflow packages all three commands, injects the tag into
`openudon version --json`, runs a credential-free runtime-only package smoke,
publishes `SHA256SUMS`, and creates the GitHub release. iCoT/LLM/provider and
maintainer-helper behavior remains experimental, and live provider execution
is not a v0.1.0 release gate.

Annotated tag `v0.1.0` now points to OpenUdon commit `86b02af`. Public test run
`30166478994` and release run `30166536155` passed, all seven release assets
verified after download, and isolated tagged installs of all three commands
completed the credential-free author/build/assess/approval/dry-run path.
