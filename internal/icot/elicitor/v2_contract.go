package elicitor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/openudon/internal/projectdoc"
	"github.com/OpenUdon/openudon/internal/projectwizard"
)

const (
	SessionVersion    = "openudon.icot-session.v2"
	TranscriptVersion = "openudon.icot-transcript.v2"

	evidenceAttrRecord               = "openudon.record"
	evidenceRecordSourceAnnotation   = "source_annotation"
	evidenceRecordAssumption         = "assumption"
	evidenceRecordMapping            = "mapping_classification"
	evidenceRecordDecision           = "decision_evidence"
	evidenceAttrSlot                 = "slot"
	evidenceAttrStage                = "stage"
	evidenceAttrConfidence           = "confidence"
	evidenceAttrReason               = "reason"
	evidenceAttrEvidence             = "evidence"
	evidenceAttrRequiresConfirmation = "requires_confirmation"
	evidenceAttrAlternatives         = "alternatives_json"
	evidenceAttrRisk                 = "risk"
	evidenceAttrLegacyID             = "legacy_id"
	evidenceAttrPromptVersion        = "prompt_version"
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
	session.Interview = publicinterview.Normalize(session.Interview)
	restoreLegacyEvidenceFromLedger(session)
	session.Annotations = normalizeSourceAnnotations(session.Annotations)
	session.Assumptions = mergeAssumptions(nil, session.Assumptions)
	session.Classifications = normalizeMappingClassifications(session.Classifications)
	session.DecisionEvidence = normalizeDecisionEvidenceList(session.DecisionEvidence)
	session.SourcePlan = normalizeSourcePlan(session.SourcePlan)
	syncLegacyEvidenceLedger(session)
	session.Interview = publicinterview.Normalize(session.Interview)
}

func normalizeSourcePlan(sources []SourceMaterialization) []SourceMaterialization {
	normalized := make([]SourceMaterialization, 0, len(sources))
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
		normalized = append(normalized, source)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].TargetPath != normalized[j].TargetPath {
			return normalized[i].TargetPath < normalized[j].TargetPath
		}
		if normalized[i].SHA256 != normalized[j].SHA256 {
			return normalized[i].SHA256 < normalized[j].SHA256
		}
		if normalized[i].Kind != normalized[j].Kind {
			return normalized[i].Kind < normalized[j].Kind
		}
		if normalized[i].ID != normalized[j].ID {
			return normalized[i].ID < normalized[j].ID
		}
		return normalized[i].SourcePath < normalized[j].SourcePath
	})
	out := make([]SourceMaterialization, 0, len(normalized))
	seenContent := map[string]bool{}
	for _, source := range normalized {
		key := source.TargetPath + "\x00" + source.SHA256
		if seenContent[key] {
			continue
		}
		seenContent[key] = true
		out = append(out, source)
	}
	return out
}

func syncLegacyEvidenceLedger(session *Session) {
	if session == nil {
		return
	}
	ledger := make([]publicinterview.Evidence, 0, len(session.Interview.Evidence)+len(session.Annotations)+len(session.Assumptions)+len(session.Classifications)+len(session.DecisionEvidence))
	for _, evidence := range session.Interview.Evidence {
		if evidence.Attributes[evidenceAttrRecord] == "" {
			ledger = append(ledger, evidence)
		}
	}
	appendEvidence := func(recordType, kind, summary, value, source string, attributes map[string]string, refs ...string) {
		summary = strings.TrimSpace(summary)
		value = strings.TrimSpace(value)
		if summary == "" {
			summary = value
		}
		if summary == "" {
			summary = recordType
		}
		if attributes == nil {
			attributes = map[string]string{}
		}
		attributes[evidenceAttrRecord] = recordType
		identity := []string{recordType, attributes[evidenceAttrStage], attributes[evidenceAttrSlot], value, strings.TrimSpace(source), attributes[evidenceAttrLegacyID]}
		digest := sha256.Sum256([]byte(strings.Join(identity, "\x00")))
		ledger = append(ledger, publicinterview.Evidence{
			ID: "evidence." + hex.EncodeToString(digest[:8]), Kind: kind, Summary: summary, Value: value,
			Source: strings.TrimSpace(source), References: dedupeStrings(refs), Attributes: attributes,
		})
	}
	for _, annotation := range session.Annotations {
		appendEvidence(evidenceRecordSourceAnnotation, publicinterview.EvidenceObservedFact, annotation.Evidence, annotation.Slot, annotation.Source, map[string]string{
			evidenceAttrSlot: annotation.Slot, evidenceAttrEvidence: annotation.Evidence, evidenceAttrPromptVersion: annotation.PromptVersion,
		}, annotation.PromptVersion)
	}
	for _, assumption := range session.Assumptions {
		appendEvidence(evidenceRecordAssumption, publicinterview.EvidenceAssumption, firstNonEmpty(assumption.Reason, assumption.Evidence), assumption.Value, "", map[string]string{
			evidenceAttrLegacyID: assumption.ID, evidenceAttrSlot: assumption.Slot, evidenceAttrReason: assumption.Reason,
			evidenceAttrEvidence: assumption.Evidence, evidenceAttrRisk: assumption.Risk,
			evidenceAttrRequiresConfirmation: strconv.FormatBool(assumption.RequiresConfirmation),
		}, assumption.Slot)
	}
	for _, classification := range session.Classifications {
		kind := publicinterview.EvidenceRecommendation
		if classification.Source == mappingSourceUser {
			kind = publicinterview.EvidenceUserDecision
		} else if classification.RequiresConfirmation {
			kind = publicinterview.EvidenceOpenDecision
		}
		appendEvidence(evidenceRecordMapping, kind, firstNonEmpty(classification.Reason, classification.Evidence), classification.Value, classification.Source, map[string]string{
			evidenceAttrSlot: classification.Slot, evidenceAttrConfidence: classification.Confidence,
			evidenceAttrReason: classification.Reason, evidenceAttrEvidence: classification.Evidence,
			evidenceAttrRequiresConfirmation: strconv.FormatBool(classification.RequiresConfirmation),
		}, classification.Slot)
	}
	for _, decision := range session.DecisionEvidence {
		kind := publicinterview.EvidenceRecommendation
		if decision.Source == mappingSourceUser {
			kind = publicinterview.EvidenceUserDecision
		}
		if decision.RequiresConfirmation {
			kind = publicinterview.EvidenceOpenDecision
		}
		alternatives, _ := json.Marshal(decision.Alternatives)
		appendEvidence(evidenceRecordDecision, kind, firstNonEmpty(decision.Reason, decision.Evidence), decision.Value, decision.Source, map[string]string{
			evidenceAttrStage: decision.Stage, evidenceAttrSlot: decision.Slot, evidenceAttrConfidence: decision.Confidence,
			evidenceAttrReason: decision.Reason, evidenceAttrEvidence: decision.Evidence,
			evidenceAttrRequiresConfirmation: strconv.FormatBool(decision.RequiresConfirmation),
			evidenceAttrAlternatives:         string(alternatives),
		}, decision.Stage, decision.Slot)
	}
	session.Interview.Evidence = ledger
}

func restoreLegacyEvidenceFromLedger(session *Session) {
	if session == nil {
		return
	}
	restoreAnnotations := len(session.Annotations) == 0
	restoreAssumptions := len(session.Assumptions) == 0
	restoreMappings := len(session.Classifications) == 0
	restoreDecisions := len(session.DecisionEvidence) == 0
	for _, evidence := range session.Interview.Evidence {
		attributes := evidence.Attributes
		switch attributes[evidenceAttrRecord] {
		case evidenceRecordSourceAnnotation:
			if !restoreAnnotations {
				continue
			}
			session.Annotations = append(session.Annotations, SourceAnnotation{
				Slot: attributes[evidenceAttrSlot], Source: evidence.Source,
				PromptVersion: attributes[evidenceAttrPromptVersion], Evidence: attributes[evidenceAttrEvidence],
			})
		case evidenceRecordAssumption:
			if !restoreAssumptions {
				continue
			}
			session.Assumptions = append(session.Assumptions, Assumption{
				ID: attributes[evidenceAttrLegacyID], Slot: attributes[evidenceAttrSlot], Value: evidence.Value,
				Reason: attributes[evidenceAttrReason], Evidence: attributes[evidenceAttrEvidence], Risk: attributes[evidenceAttrRisk],
				RequiresConfirmation: attributes[evidenceAttrRequiresConfirmation] == "true",
			})
		case evidenceRecordMapping:
			if !restoreMappings {
				continue
			}
			session.Classifications = append(session.Classifications, MappingClassification{
				Slot: attributes[evidenceAttrSlot], Value: evidence.Value, Source: evidence.Source,
				Confidence: attributes[evidenceAttrConfidence], Reason: attributes[evidenceAttrReason],
				Evidence: attributes[evidenceAttrEvidence], RequiresConfirmation: attributes[evidenceAttrRequiresConfirmation] == "true",
			})
		case evidenceRecordDecision:
			if !restoreDecisions {
				continue
			}
			var alternatives []DecisionAlternative
			_ = json.Unmarshal([]byte(attributes[evidenceAttrAlternatives]), &alternatives)
			session.DecisionEvidence = append(session.DecisionEvidence, DecisionEvidence{
				Stage: attributes[evidenceAttrStage], Slot: attributes[evidenceAttrSlot], Value: evidence.Value, Source: evidence.Source,
				Confidence: attributes[evidenceAttrConfidence], Reason: attributes[evidenceAttrReason],
				Evidence: attributes[evidenceAttrEvidence], Alternatives: alternatives,
				RequiresConfirmation: attributes[evidenceAttrRequiresConfirmation] == "true",
			})
		}
	}
}

func normalizeSourceAnnotations(annotations []SourceAnnotation) []SourceAnnotation {
	seen := map[string]bool{}
	out := make([]SourceAnnotation, 0, len(annotations))
	for _, annotation := range annotations {
		annotation.Slot = strings.TrimSpace(annotation.Slot)
		annotation.Source = strings.TrimSpace(annotation.Source)
		annotation.PromptVersion = strings.TrimSpace(annotation.PromptVersion)
		annotation.Evidence = strings.TrimSpace(annotation.Evidence)
		key := strings.Join([]string{annotation.Slot, annotation.Source, annotation.PromptVersion, annotation.Evidence}, "\x00")
		if key == "\x00\x00\x00" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, annotation)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Source < out[j].Source
	})
	return out
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
	if err := validateSourcePlanTargets(session.SourcePlan); err != nil {
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

func validateSourcePlanTargets(sources []SourceMaterialization) error {
	digests := map[string]string{}
	for _, source := range normalizeSourcePlan(sources) {
		target := filepath.ToSlash(strings.TrimSpace(source.TargetPath))
		if target == "" || target == "." {
			continue
		}
		if prior, ok := digests[target]; ok && prior != source.SHA256 {
			return fmt.Errorf("source materialization target %q has conflicting SHA-256 digests %s and %s", target, prior, source.SHA256)
		}
		digests[target] = source.SHA256
	}
	return nil
}
