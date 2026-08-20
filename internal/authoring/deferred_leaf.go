package authoring

import (
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/credentialpolicy"
)

// LeafOptions configures a review-only leaf adapter.
type LeafOptions struct {
	Name                    string                  `json:"name,omitempty"`
	Source                  string                  `json:"source,omitempty"`
	DeferredExecutionPolicy DeferredExecutionPolicy `json:"deferred_execution_policy,omitempty"`
}

// DeferredExecutionPolicy records the generic safety boundary for a leaf.
type DeferredExecutionPolicy struct {
	ReviewOnly            bool     `json:"review_only,omitempty"`
	RuntimeDeferred       bool     `json:"runtime_deferred,omitempty"`
	DirectExecutionDenied bool     `json:"direct_execution_denied,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

// LeafAdapter is an embeddable concrete object for review-only,
// runtime-deferred leaf behavior.
type LeafAdapter struct {
	Name                    string                  `json:"name,omitempty"`
	Source                  string                  `json:"source,omitempty"`
	ArtifactSet             ArtifactSet             `json:"artifact_set,omitempty"`
	ReviewPackage           ReviewPackage           `json:"review_package,omitempty"`
	DeferredExecutionPolicy DeferredExecutionPolicy `json:"deferred_execution_policy,omitempty"`
}

// ReviewPackage is prompt-safe review metadata derived from an ArtifactSet.
type ReviewPackage struct {
	Name                    string                  `json:"name,omitempty"`
	Source                  string                  `json:"source,omitempty"`
	Artifacts               []ArtifactReview        `json:"artifacts,omitempty"`
	Diagnostics             []Diagnostic            `json:"diagnostics,omitempty"`
	ReadinessIssues         []ReadinessIssue        `json:"readiness_issues,omitempty"`
	SymbolicBindings        []SymbolicBinding       `json:"symbolic_bindings,omitempty"`
	BindingNames            []string                `json:"binding_names,omitempty"`
	Assumptions             []Assumption            `json:"assumptions,omitempty"`
	QuestionPlan            QuestionPlan            `json:"question_plan,omitempty"`
	TranscriptSummary       []string                `json:"transcript_summary,omitempty"`
	CredentialAudit         BindingAudit            `json:"credential_audit,omitempty"`
	RequiredReviewActions   []string                `json:"required_review_actions,omitempty"`
	DeferredExecutionPolicy DeferredExecutionPolicy `json:"deferred_execution_policy,omitempty"`
}

// ArtifactReview describes an artifact without duplicating its content.
type ArtifactReview struct {
	Path      string `json:"path"`
	MediaType string `json:"media_type,omitempty"`
	SizeBytes int    `json:"size_bytes,omitempty"`
}

// BindingAudit reports symbolic bindings and literal credential findings.
type BindingAudit struct {
	DeclaredSymbolicBindings     []string     `json:"declared_symbolic_bindings,omitempty"`
	LiteralCredentialDiagnostics []Diagnostic `json:"literal_credential_diagnostics,omitempty"`
}

// NewLeafAdapter returns an embeddable adapter with review metadata derived
// from the supplied artifact set. It does not bind credentials or execute APIs.
func NewLeafAdapter(set ArtifactSet, opts LeafOptions) LeafAdapter {
	policy := opts.DeferredExecutionPolicy
	if !policy.ReviewOnly && !policy.RuntimeDeferred && !policy.DirectExecutionDenied && len(policy.Notes) == 0 {
		policy = defaultDeferredExecutionPolicy()
	}
	leaf := LeafAdapter{
		Name:                    strings.TrimSpace(opts.Name),
		Source:                  strings.TrimSpace(opts.Source),
		ArtifactSet:             cloneArtifactSet(set),
		DeferredExecutionPolicy: policy,
	}
	leaf.ReviewPackage = leaf.MinimumReviewPackage()
	return leaf
}

func defaultDeferredExecutionPolicy() DeferredExecutionPolicy {
	return DeferredExecutionPolicy{
		ReviewOnly:            true,
		RuntimeDeferred:       true,
		DirectExecutionDenied: true,
		Notes: []string{
			"Artifacts are review-only until a caller-specific renderer validates them.",
			"Credential values must be supplied by a trusted runtime binding layer.",
			"OpenUdon authoring does not execute APIs or workflows.",
		},
	}
}

// ErrorDiagnostics returns error-severity diagnostics.
func (leaf LeafAdapter) ErrorDiagnostics() []Diagnostic {
	var out []Diagnostic
	for _, diagnostic := range leaf.ArtifactSet.Diagnostics {
		if strings.EqualFold(diagnostic.Severity, "error") {
			out = append(out, diagnostic)
		}
	}
	out = append(out, leaf.CredentialValueDiagnostics()...)
	return out
}

// BlockingReadinessIssues returns readiness issues that block rendering or use.
func (leaf LeafAdapter) BlockingReadinessIssues() []ReadinessIssue {
	var out []ReadinessIssue
	for _, issue := range leaf.ArtifactSet.ReadinessIssues {
		if blockingSeverity(issue.Severity) {
			out = append(out, issue)
		}
	}
	return out
}

// HasBlockingIssues reports whether diagnostics, readiness, or credentials
// require review before caller-specific rendering.
func (leaf LeafAdapter) HasBlockingIssues() bool {
	return len(leaf.ErrorDiagnostics()) > 0 || len(leaf.BlockingReadinessIssues()) > 0
}

// RequiredBindings returns declared symbolic runtime bindings only.
func (leaf LeafAdapter) RequiredBindings() []SymbolicBinding {
	byName := map[string]SymbolicBinding{}
	for _, binding := range leaf.ArtifactSet.SymbolicBindings {
		name := strings.TrimSpace(binding.Name)
		if name != "" {
			binding.Name = name
			byName[name] = binding
		}
	}
	for _, slot := range leaf.ArtifactSet.Slots {
		addBindingRef(byName, slot.BindingRef, slot.Source)
	}
	for _, assumption := range leaf.ArtifactSet.Assumptions {
		addBindingRef(byName, assumption.Binding, assumption.Source)
	}
	out := make([]SymbolicBinding, 0, len(byName))
	for _, binding := range byName {
		out = append(out, binding)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// BindingNames returns declared symbolic binding names.
func (leaf LeafAdapter) BindingNames() []string {
	bindings := leaf.RequiredBindings()
	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		names = append(names, binding.Name)
	}
	sort.Strings(names)
	return names
}

// BindingAudit returns a symbolic-only binding audit.
func (leaf LeafAdapter) BindingAudit() BindingAudit {
	return BindingAudit{
		DeclaredSymbolicBindings:     leaf.BindingNames(),
		LiteralCredentialDiagnostics: leaf.CredentialValueDiagnostics(),
	}
}

// CredentialValueDiagnostics flags likely literal credential values in artifact
// content without resolving or testing credentials.
func (leaf LeafAdapter) CredentialValueDiagnostics() []Diagnostic {
	return ScanCredentialValues(leaf.ArtifactSet.Artifacts)
}

// ScanCredentialValues flags likely literal credential values in artifact
// content without resolving or testing credentials.
func ScanCredentialValues(artifacts []Artifact) []Diagnostic {
	var diagnostics []Diagnostic
	for _, artifact := range artifacts {
		if len(artifact.Content) == 0 {
			continue
		}
		if ContainsLikelyCredentialValue(artifact.Content) {
			diagnostics = append(diagnostics, Diagnostic{
				Severity:    "error",
				Code:        "leaf.literal_credential",
				Message:     "artifact content appears to contain a literal credential value",
				Path:        artifact.Path,
				Remediation: "Replace literal values with symbolic binding names and provide secrets only through the trusted runtime.",
			})
		}
	}
	return diagnostics
}

// MinimumReviewPackage builds prompt-safe review metadata from the leaf.
func (leaf LeafAdapter) MinimumReviewPackage() ReviewPackage {
	artifacts := make([]ArtifactReview, 0, len(leaf.ArtifactSet.Artifacts))
	for _, artifact := range leaf.ArtifactSet.Artifacts {
		artifacts = append(artifacts, artifactReview(artifact))
	}
	seenArtifacts := make(map[string]bool, len(artifacts))
	for _, artifact := range artifacts {
		seenArtifacts[artifact.Path] = true
	}
	for _, artifact := range leaf.ReviewPackage.Artifacts {
		if !seenArtifacts[artifact.Path] {
			artifacts = append(artifacts, artifact)
			seenArtifacts[artifact.Path] = true
		}
	}
	sort.SliceStable(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	pkg := ReviewPackage{
		Name:                    leaf.Name,
		Source:                  leaf.Source,
		Artifacts:               artifacts,
		Diagnostics:             append([]Diagnostic(nil), leaf.ArtifactSet.Diagnostics...),
		ReadinessIssues:         append([]ReadinessIssue(nil), leaf.ArtifactSet.ReadinessIssues...),
		SymbolicBindings:        leaf.RequiredBindings(),
		BindingNames:            leaf.BindingNames(),
		Assumptions:             append([]Assumption(nil), leaf.ArtifactSet.Assumptions...),
		QuestionPlan:            leaf.ArtifactSet.QuestionPlan,
		TranscriptSummary:       transcriptSummary(leaf.ArtifactSet.Transcript),
		CredentialAudit:         leaf.BindingAudit(),
		DeferredExecutionPolicy: leaf.DeferredExecutionPolicy,
	}
	pkg.RequiredReviewActions = requiredReviewActions(leaf, pkg)
	return pkg
}

func artifactReview(artifact Artifact) ArtifactReview {
	return ArtifactReview{
		Path:      artifact.Path,
		MediaType: artifact.MediaType,
		SizeBytes: len(artifact.Content),
	}
}

func addBindingRef(out map[string]SymbolicBinding, ref BindingRef, source string) {
	name := strings.TrimSpace(ref.Name)
	if name == "" {
		return
	}
	if _, ok := out[name]; ok {
		return
	}
	out[name] = SymbolicBinding{
		Name:        name,
		Kind:        ref.Kind,
		Source:      source,
		Description: ref.Description,
	}
}

func blockingSeverity(severity string) bool {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error", "critical", "blocking", "fatal":
		return true
	default:
		return false
	}
}

func requiredReviewActions(leaf LeafAdapter, pkg ReviewPackage) []string {
	actions := []string{
		"Review all artifacts before caller-specific rendering.",
		"Validate artifacts with the downstream renderer and policy checks.",
		"Keep credential values out of prompts, artifacts, logs, and committed files.",
	}
	if len(pkg.BindingNames) > 0 {
		actions = append(actions, "Map symbolic binding names to trusted runtime bindings outside generated artifacts.")
	}
	if len(pkg.ReadinessIssues) > 0 || len(pkg.Diagnostics) > 0 || leaf.HasBlockingIssues() {
		actions = append(actions, "Resolve blocking diagnostics and readiness issues before execution-capable handoff.")
	}
	if len(pkg.QuestionPlan.Questions) > 0 {
		actions = append(actions, "Answer clarification questions before approving downstream artifacts.")
	}
	return actions
}

func transcriptSummary(transcript *Transcript) []string {
	if transcript == nil {
		return nil
	}
	out := make([]string, 0, len(transcript.Turns))
	for _, turn := range transcript.Turns {
		content := strings.TrimSpace(turn.Content)
		if content == "" {
			continue
		}
		if len(content) > 240 {
			content = content[:240] + "..."
		}
		if turn.Source != "" {
			content = turn.Source + ": " + content
		}
		out = append(out, content)
	}
	return out
}

func cloneArtifactSet(set ArtifactSet) ArtifactSet {
	out := set
	out.Artifacts = append([]Artifact(nil), set.Artifacts...)
	for i := range out.Artifacts {
		out.Artifacts[i] = cloneArtifact(out.Artifacts[i])
	}
	out.Diagnostics = append([]Diagnostic(nil), set.Diagnostics...)
	out.SymbolicBindings = append([]SymbolicBinding(nil), set.SymbolicBindings...)
	out.ReadinessIssues = append([]ReadinessIssue(nil), set.ReadinessIssues...)
	out.Slots = append([]Slot(nil), set.Slots...)
	out.Assumptions = append([]Assumption(nil), set.Assumptions...)
	if set.Transcript != nil {
		transcript := *set.Transcript
		transcript.Turns = append([]TranscriptTurn(nil), set.Transcript.Turns...)
		out.Transcript = &transcript
	}
	return out
}

func cloneArtifact(artifact Artifact) Artifact {
	out := artifact
	out.Content = append([]byte(nil), artifact.Content...)
	return out
}

// ContainsLikelyCredentialValue reports whether data contains a likely concrete
// credential value. Symbolic workflow references and binding names are allowed.
func ContainsLikelyCredentialValue(data []byte) bool {
	return credentialpolicy.ContainsLikelyValue(data)
}
