package elicitor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/openudon/internal/projectdoc"
	"github.com/OpenUdon/openudon/internal/projectwizard"
)

const (
	SessionVersion    = "openudon.icot-session.v2"
	TranscriptVersion = "openudon.icot-transcript.v2"
)

// WorkflowBoundary is the confirmed delivery boundary for the one active
// workflow. Candidate workflows are deliberately kept outside this boundary.
type WorkflowBoundary struct {
	Outcome         string   `json:"outcome" yaml:"outcome"`
	Actor           string   `json:"actor" yaml:"actor"`
	Trigger         string   `json:"trigger" yaml:"trigger"`
	SuccessEvidence []string `json:"success_evidence" yaml:"success_evidence"`
	NonGoals        []string `json:"non_goals,omitempty" yaml:"non_goals,omitempty"`
	Confirmed       bool     `json:"confirmed" yaml:"confirmed"`
}

// SourceMaterialization records one reviewed source and its eventual package
// target. SourcePath is inspected during the interview; TargetPath is not
// written until the complete proposal or incomplete draft is approved.
type SourceMaterialization struct {
	Kind           string `json:"kind" yaml:"kind"`
	ID             string `json:"id" yaml:"id"`
	SourcePath     string `json:"source_path" yaml:"source_path"`
	TargetPath     string `json:"target_path" yaml:"target_path"`
	SHA256         string `json:"sha256" yaml:"sha256"`
	Title          string `json:"title,omitempty" yaml:"title,omitempty"`
	OperationCount int    `json:"operation_count" yaml:"operation_count"`
	Provenance     string `json:"provenance" yaml:"provenance"`
}

// CandidateWorkflow is an unnumbered future direction with no source,
// operation, mapping, or implementation breakdown.
type CandidateWorkflow = projectdoc.CandidateWorkflow

// FileAction is one exact proposed filesystem mutation.
type FileAction struct {
	Action string `json:"action" yaml:"action"`
	Path   string `json:"path" yaml:"path"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

func normalizeV2Session(session *Session) {
	if session == nil {
		return
	}
	if strings.TrimSpace(session.Version) == "" {
		session.Version = SessionVersion
	}
	session.Version = strings.TrimSpace(session.Version)
	session.Boundary.Outcome = strings.TrimSpace(firstNonEmpty(session.Boundary.Outcome, session.Project.Goal, session.IntentDescription()))
	session.Boundary.Actor = strings.TrimSpace(session.Boundary.Actor)
	session.Boundary.Trigger = strings.TrimSpace(session.Boundary.Trigger)
	session.Boundary.SuccessEvidence = dedupeStrings(session.Boundary.SuccessEvidence)
	session.Boundary.NonGoals = dedupeStrings(session.Boundary.NonGoals)
	session.CandidateWorkflows = projectdoc.NormalizeCandidateWorkflows(append(session.CandidateWorkflows, session.Project.CandidateWorkflows...))
	session.Project.CandidateWorkflows = append([]projectdoc.CandidateWorkflow(nil), session.CandidateWorkflows...)
	session.SourcePlan = normalizeSourcePlan(session.SourcePlan)
	syncLegacyEvidenceLedger(session)
	session.Interview = publicinterview.Normalize(session.Interview)
}

func normalizeSourcePlan(sources []SourceMaterialization) []SourceMaterialization {
	byKey := map[string]SourceMaterialization{}
	for _, source := range sources {
		source.Kind = strings.ToLower(strings.TrimSpace(source.Kind))
		source.ID = strings.TrimSpace(source.ID)
		source.SourcePath = filepath.Clean(strings.TrimSpace(source.SourcePath))
		source.TargetPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(source.TargetPath)))
		source.SHA256 = strings.ToLower(strings.TrimSpace(source.SHA256))
		source.Title = strings.TrimSpace(source.Title)
		source.Provenance = strings.TrimSpace(source.Provenance)
		if source.SourcePath == "." || source.TargetPath == "." {
			continue
		}
		key := source.Kind + "\x00" + source.ID + "\x00" + source.SHA256
		byKey[key] = source
	}
	out := make([]SourceMaterialization, 0, len(byKey))
	for _, source := range byKey {
		out = append(out, source)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TargetPath != out[j].TargetPath {
			return out[i].TargetPath < out[j].TargetPath
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func syncLegacyEvidenceLedger(session *Session) {
	if session == nil {
		return
	}
	existing := map[string]bool{}
	for _, evidence := range session.Interview.Evidence {
		existing[evidence.ID] = true
	}
	appendEvidence := func(kind, summary, value, source string, refs ...string) {
		summary = strings.TrimSpace(summary)
		value = strings.TrimSpace(value)
		if summary == "" {
			summary = value
		}
		if summary == "" {
			return
		}
		digest := sha256.Sum256([]byte(strings.Join([]string{kind, summary, value, source}, "\x00")))
		id := "evidence." + hex.EncodeToString(digest[:6])
		if existing[id] {
			return
		}
		existing[id] = true
		session.Interview.Evidence = append(session.Interview.Evidence, publicinterview.Evidence{
			ID: id, Kind: kind, Summary: summary, Value: value, Source: strings.TrimSpace(source), References: dedupeStrings(refs),
		})
	}
	for _, annotation := range session.Annotations {
		appendEvidence(publicinterview.EvidenceObservedFact, annotation.Evidence, annotation.Slot, annotation.Source, annotation.PromptVersion)
	}
	for _, assumption := range session.Assumptions {
		appendEvidence(publicinterview.EvidenceAssumption, firstNonEmpty(assumption.Reason, assumption.Evidence), assumption.Value, "legacy-assumption", assumption.Slot)
	}
	for _, classification := range session.Classifications {
		kind := publicinterview.EvidenceRecommendation
		if classification.Source == mappingSourceUser {
			kind = publicinterview.EvidenceUserDecision
		}
		appendEvidence(kind, firstNonEmpty(classification.Reason, classification.Evidence), classification.Value, classification.Source, classification.Slot)
	}
	for _, decision := range session.DecisionEvidence {
		kind := publicinterview.EvidenceRecommendation
		if decision.Source == mappingSourceUser {
			kind = publicinterview.EvidenceUserDecision
		}
		if decision.RequiresConfirmation {
			kind = publicinterview.EvidenceOpenDecision
		}
		appendEvidence(kind, firstNonEmpty(decision.Reason, decision.Evidence), decision.Value, decision.Source, decision.Stage, decision.Slot)
	}
}

func validateV2Session(session Session) error {
	if err := validateV2State(session); err != nil {
		return err
	}
	if strings.TrimSpace(session.Boundary.Outcome) == "" || strings.TrimSpace(session.Boundary.Actor) == "" || strings.TrimSpace(session.Boundary.Trigger) == "" || len(session.Boundary.SuccessEvidence) == 0 {
		return fmt.Errorf("active workflow boundary requires outcome, actor, trigger, and success evidence")
	}
	if len(session.CandidateWorkflows) > 0 && strings.TrimSpace(session.Interview.Metadata["active_workflow_selected"]) == "" {
		return fmt.Errorf("broad requests require explicit active-workflow selection")
	}
	if strings.TrimSpace(firstNonEmpty(session.Fallback, session.Project.Fallback)) == "" {
		return fmt.Errorf("active workflow requires explicit fallback behavior")
	}
	if projectwizard.NormalizeSideEffectScope(session.SideEffectScope) == "" {
		return fmt.Errorf("active workflow requires an explicit side-effect and approval posture")
	}
	return nil
}

func validateV2State(session Session) error {
	if session.Version != SessionVersion {
		return fmt.Errorf("unsupported iCoT session version %q; want %q (v1 inputs are not accepted)", session.Version, SessionVersion)
	}
	if err := publicinterview.Validate(session.Interview); err != nil {
		return err
	}
	for _, source := range session.SourcePlan {
		if source.Kind == "" || source.ID == "" || source.SourcePath == "" || source.TargetPath == "" || len(source.SHA256) != 64 || source.Provenance == "" {
			return fmt.Errorf("source materialization %q must include kind, id, source path, target path, SHA-256, and provenance", source.ID)
		}
	}
	for _, candidate := range session.CandidateWorkflows {
		if candidate.Title == "" || candidate.Outcome == "" || candidate.DeferralReason == "" || candidate.PromotionTrigger == "" {
			return fmt.Errorf("candidate workflow %q must include title, outcome, deferral reason, and promotion trigger", candidate.Title)
		}
	}
	return nil
}
