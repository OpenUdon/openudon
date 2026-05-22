package elicitor

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/openudon/internal/authoring"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const maxCatalogPlanCandidates = 16

type CatalogPlanRequest struct {
	Opening        string                        `json:"opening"`
	SessionSummary CatalogPlanSessionSummary     `json:"session_summary,omitempty"`
	Candidates     []CatalogPlanCandidateContext `json:"candidates"`
}

type CatalogPlanSessionSummary struct {
	WorkflowName        string   `json:"workflow_name,omitempty"`
	WorkflowDescription string   `json:"workflow_description,omitempty"`
	ExistingSteps       []string `json:"existing_steps,omitempty"`
}

type CatalogPlanCandidateContext struct {
	ProviderID   string   `json:"provider_id"`
	ProviderName string   `json:"provider_name,omitempty"`
	AuthStatus   string   `json:"auth_status,omitempty"`
	ArtifactKey  string   `json:"artifact_key"`
	RelativePath string   `json:"relative_path"`
	Kind         string   `json:"kind,omitempty"`
	Protocol     string   `json:"protocol,omitempty"`
	FollowUps    []string `json:"follow_ups,omitempty"`
}

type CatalogPlanResponse struct {
	SelectedArtifacts []CatalogPlanArtifactSelection `json:"selected_artifacts,omitempty"`
	ProposedSteps     []CatalogPlanStep              `json:"proposed_steps,omitempty"`
	Blockers          []string                       `json:"blockers,omitempty"`
	Assumptions       []string                       `json:"assumptions,omitempty"`
}

type CatalogPlanArtifactSelection struct {
	ProviderID  string `json:"provider_id,omitempty"`
	ArtifactKey string `json:"artifact_key,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type CatalogPlanStep struct {
	Name      string             `json:"name,omitempty"`
	Type      string             `json:"type,omitempty"`
	Provider  string             `json:"provider,omitempty"`
	OpenAPI   string             `json:"openapi,omitempty"`
	Do        string             `json:"do,omitempty"`
	DependsOn flexibleStringList `json:"depends_on,omitempty"`
}

type flexibleStringList []string

func (list *flexibleStringList) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var out []string
	add := func(value any) {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				out = append(out, v)
			}
		case float64:
			out = append(out, fmt.Sprintf("%.0f", v))
		}
	}
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			add(item)
		}
	default:
		add(v)
	}
	*list = flexibleStringList(dedupeStrings(out))
	return nil
}

type catalogPlanApplication struct {
	Result       CatalogMigrationResult
	Applied      bool
	Rejected     []string
	SelectedKeys []string
}

func BuildCatalogPlanRequest(opening string, session Session, hints []CatalogHint, exampleDir string) CatalogPlanRequest {
	candidates := catalogPlanCandidateContexts(hints, exampleDir)
	request := CatalogPlanRequest{
		Opening:    strings.TrimSpace(opening),
		Candidates: candidates,
	}
	if session.Intent.Workflow != nil {
		request.SessionSummary.WorkflowName = strings.TrimSpace(session.Intent.Workflow.Name)
		request.SessionSummary.WorkflowDescription = strings.TrimSpace(session.Intent.Workflow.Description)
	}
	for _, step := range session.Intent.Steps {
		if step == nil || strings.TrimSpace(step.Name) == "" {
			continue
		}
		request.SessionSummary.ExistingSteps = append(request.SessionSummary.ExistingSteps, step.Name)
	}
	return request
}

func catalogPlanCandidateContexts(hints []CatalogHint, exampleDir string) []CatalogPlanCandidateContext {
	candidates := CatalogMigrationCandidates(hints, exampleDir)
	byProvider := map[string][]CatalogMigrationCandidate{}
	for _, candidate := range candidates {
		byProvider[candidate.ProviderID] = append(byProvider[candidate.ProviderID], candidate)
	}
	var out []CatalogPlanCandidateContext
	for _, hint := range hints {
		for _, candidate := range byProvider[hint.Provider.ID] {
			if len(out) >= maxCatalogPlanCandidates {
				return out
			}
			out = append(out, CatalogPlanCandidateContext{
				ProviderID:   candidate.ProviderID,
				ProviderName: candidate.ProviderName,
				AuthStatus:   string(hint.AuthStatus),
				ArtifactKey:  catalogPlanArtifactKey(candidate),
				RelativePath: candidate.RelativePath,
				Kind:         string(candidate.Kind),
				Protocol:     firstNonEmpty(candidate.Protocol, catalogPlanProtocol(candidate)),
				FollowUps:    append([]string(nil), hint.FollowUps...),
			})
		}
	}
	return out
}

func applyCatalogPlanResponse(out io.Writer, session *Session, hints []CatalogHint, exampleDir string, response CatalogPlanResponse) (catalogPlanApplication, error) {
	if session == nil {
		return catalogPlanApplication{}, nil
	}
	candidates := CatalogMigrationCandidates(hints, exampleDir)
	byKey := map[string]CatalogMigrationCandidate{}
	byProvider := map[string][]CatalogMigrationCandidate{}
	for _, candidate := range candidates {
		key := catalogPlanArtifactKey(candidate)
		byKey[key] = candidate
		byProvider[candidate.ProviderID] = append(byProvider[candidate.ProviderID], candidate)
	}
	selected := map[string]CatalogMigrationCandidate{}
	selectedProviders := map[string]bool{}
	var rejected []string
	for _, selection := range response.SelectedArtifacts {
		providerID := strings.TrimSpace(selection.ProviderID)
		key := strings.TrimSpace(selection.ArtifactKey)
		candidate, ok := byKey[key]
		switch {
		case providerID == "":
			rejected = append(rejected, firstNonEmpty(key, "<empty>")+" (missing provider)")
		case !ok:
			rejected = append(rejected, firstNonEmpty(key, "<empty>"))
		case candidate.ProviderID != providerID:
			rejected = append(rejected, key+" (provider "+providerID+")")
		case !catalogPlanCandidateMigratable(candidate):
			rejected = append(rejected, key)
		default:
			selected[key] = candidate
			selectedProviders[candidate.ProviderID] = true
		}
	}
	for providerID := range selectedProviders {
		for _, candidate := range byProvider[providerID] {
			if candidate.Kind == catalog.SpecKind("advisory-overlay") {
				selected[catalogPlanArtifactKey(candidate)] = candidate
			}
		}
	}
	sort.Strings(rejected)
	for _, item := range rejected {
		fmt.Fprintf(out, "icot: rejected unknown catalog artifact %s\n", item)
	}
	if len(selected) == 0 {
		session.DraftEvents = append(session.DraftEvents, TranscriptEvent{Kind: "catalog_plan_result", Data: map[string]any{
			"selected_artifacts": response.SelectedArtifacts,
			"proposed_steps":     response.ProposedSteps,
			"blockers":           response.Blockers,
			"assumptions":        response.Assumptions,
			"rejected":           rejected,
		}})
		recordCatalogPlanRejections(session, rejected)
		return catalogPlanApplication{Rejected: rejected}, nil
	}
	selectedCandidates := make([]CatalogMigrationCandidate, 0, len(selected))
	selectedKeys := make([]string, 0, len(selected))
	for key, candidate := range selected {
		selectedKeys = append(selectedKeys, key)
		selectedCandidates = append(selectedCandidates, candidate)
	}
	sort.Slice(selectedCandidates, func(i, j int) bool {
		if selectedCandidates[i].ProviderID != selectedCandidates[j].ProviderID {
			return selectedCandidates[i].ProviderID < selectedCandidates[j].ProviderID
		}
		return selectedCandidates[i].RelativePath < selectedCandidates[j].RelativePath
	})
	sort.Strings(selectedKeys)
	result, err := migrateSelectedCatalogCandidates(selectedCandidates)
	if err != nil {
		return catalogPlanApplication{Rejected: rejected, SelectedKeys: selectedKeys}, err
	}
	for _, candidate := range append(result.Existing, result.Copied...) {
		fmt.Fprintf(out, "icot: selected %s API document from catalog plan: %s\n", candidate.ProviderName, candidate.RelativePath)
	}
	applyCatalogPlanSteps(session, response.ProposedSteps, selectedCandidates)
	recordCatalogPlanAssumptions(session, response, selectedCandidates, rejected)
	recordCatalogPlanRejections(session, rejected)
	return catalogPlanApplication{Result: result, Applied: true, Rejected: rejected, SelectedKeys: selectedKeys}, nil
}

func migrateSelectedCatalogCandidates(candidates []CatalogMigrationCandidate) (CatalogMigrationResult, error) {
	var result CatalogMigrationResult
	for _, candidate := range candidates {
		if candidate.ExistingLocal {
			result.Existing = append(result.Existing, candidate)
			continue
		}
		if err := copyCatalogArtifact(candidate.SourcePath, candidate.TargetPath); err != nil {
			return result, err
		}
		result.Copied = append(result.Copied, candidate)
	}
	return result, nil
}

func applyCatalogPlanSteps(session *Session, proposed []CatalogPlanStep, candidates []CatalogMigrationCandidate) {
	if session == nil || len(session.Intent.Steps) > 0 {
		return
	}
	byProvider := selectedCatalogArtifactsByProvider(candidates)
	providerAllowed := map[string]bool{}
	for providerID := range byProvider {
		providerAllowed[providerID] = true
	}
	var steps []*rollout.Step
	acceptedNames := map[string]bool{}
	for _, step := range proposed {
		provider := strings.TrimSpace(step.Provider)
		if provider == "" || !providerAllowed[provider] {
			continue
		}
		name := uniqueCatalogPlanStepName(slugIdent(firstNonEmpty(step.Name, provider)), acceptedNames)
		if name == "" {
			continue
		}
		dependsOn := safeCatalogPlanDependsOn([]string(step.DependsOn), acceptedNames)
		acceptedNames[name] = true
		steps = append(steps, &rollout.Step{
			Name:      name,
			Type:      "http",
			Provider:  provider,
			OpenAPI:   catalogPlanStepOpenAPI(step.OpenAPI, byProvider[provider]),
			Do:        firstLine(firstNonEmpty(step.Do, "Use "+provider+" for this workflow capability.")),
			DependsOn: dependsOn,
		})
	}
	if len(steps) == 0 {
		for _, candidate := range candidates {
			provider := strings.TrimSpace(candidate.ProviderID)
			if provider == "" || acceptedNames[provider] {
				continue
			}
			name := uniqueCatalogPlanStepName(slugIdent(provider), acceptedNames)
			acceptedNames[name] = true
			steps = append(steps, &rollout.Step{
				Name:     name,
				Type:     "http",
				Provider: provider,
				OpenAPI:  catalogPlanStepOpenAPI("", byProvider[provider]),
				Do:       "Use " + firstNonEmpty(candidate.ProviderName, provider) + " for this workflow capability.",
			})
		}
	}
	session.Intent.Steps = steps
}

func selectedCatalogArtifactsByProvider(candidates []CatalogMigrationCandidate) map[string][]CatalogMigrationCandidate {
	out := map[string][]CatalogMigrationCandidate{}
	for _, candidate := range candidates {
		out[candidate.ProviderID] = append(out[candidate.ProviderID], candidate)
	}
	return out
}

func catalogPlanStepOpenAPI(value string, candidates []CatalogMigrationCandidate) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	if value != "" {
		for _, candidate := range candidates {
			if candidate.RelativePath == value {
				return value
			}
		}
	}
	for _, candidate := range candidates {
		if isLocalAPIDocumentRef(candidate.RelativePath) {
			return candidate.RelativePath
		}
	}
	return ""
}

func safeCatalogPlanDependsOn(values []string, acceptedNames map[string]bool) []string {
	var out []string
	for _, value := range values {
		name := slugIdent(value)
		if name != "" && acceptedNames[name] {
			out = append(out, name)
		}
	}
	return dedupeStrings(out)
}

func uniqueCatalogPlanStepName(name string, used map[string]bool) string {
	name = slugIdent(name)
	if name == "" {
		return ""
	}
	if !used[name] {
		return name
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s_%d", name, i)
		if !used[candidate] {
			return candidate
		}
	}
	return ""
}

func recordCatalogPlanAssumptions(session *Session, response CatalogPlanResponse, candidates []CatalogMigrationCandidate, rejected []string) {
	if session == nil {
		return
	}
	var providerNames []string
	for _, candidate := range candidates {
		providerNames = append(providerNames, firstNonEmpty(candidate.ProviderName, candidate.ProviderID))
	}
	providerNames = dedupeStrings(providerNames)
	session.Assumptions = mergeAssumptions(session.Assumptions, []Assumption{{
		ID:                   "catalog_plan_api_docs_migrated",
		Slot:                 "intent.openapi",
		Value:                strings.Join(providerNames, " -> "),
		Reason:               "LLM catalog planning selected validated local catalog artifacts; iCoT still requires operationId selection from local metadata.",
		Evidence:             strings.Join(providerNames, " -> "),
		Risk:                 "review",
		RequiresConfirmation: true,
	}})
	session.DraftEvents = append(session.DraftEvents, TranscriptEvent{Kind: "catalog_plan_result", Data: map[string]any{
		"selected_artifacts": response.SelectedArtifacts,
		"proposed_steps":     response.ProposedSteps,
		"blockers":           response.Blockers,
		"assumptions":        response.Assumptions,
		"rejected":           rejected,
	}})
}

func recordCatalogPlanRejections(session *Session, rejected []string) {
	if session == nil || len(rejected) == 0 {
		return
	}
	session.Assumptions = mergeAssumptions(session.Assumptions, []Assumption{{
		ID:                   "catalog_plan_rejected",
		Slot:                 "catalog_plan.selected_artifacts",
		Value:                strings.Join(rejected, ", "),
		Reason:               "Rejected unknown, mismatched, or non-migratable catalog artifacts from the LLM catalog plan.",
		Evidence:             strings.Join(rejected, ", "),
		Risk:                 "warning",
		RequiresConfirmation: true,
	}})
	session.DraftEvents = append(session.DraftEvents, TranscriptEvent{Kind: "catalog_plan_rejected", Data: map[string]any{
		"artifacts": rejected,
	}})
}

func catalogPlanArtifactKey(candidate CatalogMigrationCandidate) string {
	return candidate.ProviderID + ":" + candidate.RelativePath
}

func catalogPlanCandidateMigratable(candidate CatalogMigrationCandidate) bool {
	return strings.TrimSpace(candidate.SourcePath) != "" && strings.TrimSpace(candidate.TargetPath) != "" && strings.TrimSpace(candidate.RelativePath) != ""
}

func catalogPlanProtocol(candidate CatalogMigrationCandidate) string {
	switch candidate.Kind {
	case catalog.SpecKindGoogleDiscovery:
		return string(catalog.SpecProtocolGoogleDiscovery)
	case catalog.SpecKindOpenAPI, catalog.SpecKind("advisory-overlay"):
		return string(catalog.SpecProtocolOpenAPI)
	default:
		return string(catalog.SpecProtocolUnknown)
	}
}

func catalogPlanEvents(session Session, reported *int) []authoring.PromptEvent {
	if reported == nil {
		return nil
	}
	if *reported < 0 || *reported > len(session.DraftEvents) {
		*reported = 0
	}
	raw := session.DraftEvents[*reported:]
	*reported = len(session.DraftEvents)
	out := make([]authoring.PromptEvent, 0, len(raw))
	for _, event := range raw {
		out = append(out, authoring.PromptEvent{Kind: event.Kind, Data: event.Data})
	}
	return out
}
