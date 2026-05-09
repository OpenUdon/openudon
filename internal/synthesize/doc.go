// Package synthesize owns OpenUdon's deterministic artifact pipeline.
//
// Synthesize starts from a project brief and optional existing intent, then
// discovers OpenAPI inputs, renders intent, workflow HCL, UWS YAML, expected
// plans, review evidence, handoff manifests, refinement evidence, and quality
// reports. Build, Promote, and Assess expose narrower entry points over the
// same artifact set for repair loops after hand edits. The package is
// intentionally validation-first: generated artifacts are treated as untrusted
// until quality checks and review handoff gates pass.
package synthesize

