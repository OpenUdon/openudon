# Shared Interview Settlement And Lifecycle-Ranking Boundary Result

OpenUdon now routes generic frontier planning and complete-round settlement
through Authoring's clone-based `icot.InterviewBinding`. Product callbacks
still own graph preparation, questions, intent mutation, normalization,
validation, and post-round behavior, and OpenUdon explicitly preserves its
`Workflow goal` wording.

Lifecycle sibling ranking now comes from `apitools/operationlifecycle` over
source-bearing `apitools.OperationSummary` values. Candidate lookup preserves
source ID plus operation ID, while Google Discovery upload normalization is
limited to explicit source provenance. Authoring's former lifecycle-ranking
package is no longer part of the dependency boundary.

Workspace tests, vet, the iCoT scorecard, and compatibility checks pass.
Apitools and Authoring were published in dependency order; OpenUdon pins both
revisions and passes standalone `GOWORK=off` test/vet without replacements.
