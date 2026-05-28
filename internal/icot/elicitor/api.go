package elicitor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/awssmithy"
	"github.com/OpenUdon/apitools/googlediscovery"
	"github.com/OpenUdon/openudon/internal/openapidisco"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const (
	// Keep draft prompts compact enough for small models while still exposing a
	// useful local operation shortlist.
	maxDraftOperationCandidates = 12
	// Bound LLM-requested detail expansion so one draft cannot pull the whole
	// local API catalog into prompt context.
	maxDraftRequestedOperations = 5
	maxRequestBodyFieldDepth    = 6
	maxRequestBodyFields        = 60
)

type APIDocument = apitools.AuthoringAPIDocument

type OperationDetailRef struct {
	DocumentPath string `json:"document_path"`
	OperationID  string `json:"operationId"`
}

type operationCatalogDocumentContext struct {
	DocumentPath string                         `json:"document_path"`
	Title        string                         `json:"title,omitempty"`
	Description  string                         `json:"description,omitempty"`
	Operations   []operationCatalogEntryContext `json:"operations,omitempty"`
}

type operationCatalogEntryContext struct {
	OperationID string `json:"operationId"`
	Method      string `json:"method,omitempty"`
	Path        string `json:"path,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Provenance  string `json:"provenance,omitempty"`
}

type operationPromptContext struct {
	OperationID    string                    `json:"operationId"`
	Method         string                    `json:"method"`
	Path           string                    `json:"path"`
	Summary        string                    `json:"summary,omitempty"`
	Description    string                    `json:"description,omitempty"`
	Tags           []string                  `json:"tags,omitempty"`
	Provenance     string                    `json:"provenance,omitempty"`
	RequiredFields []string                  `json:"required_fields,omitempty"`
	Parameters     []parameterPromptContext  `json:"parameters,omitempty"`
	RequestBody    *requestBodyPromptContext `json:"request_body,omitempty"`
	Security       securityPromptContext     `json:"security,omitempty"`
}

type parameterPromptContext struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Required    bool   `json:"required"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type requestBodyPromptContext struct {
	Required           bool                      `json:"required"`
	ContentType        string                    `json:"content_type,omitempty"`
	Type               string                    `json:"type,omitempty"`
	Description        string                    `json:"description,omitempty"`
	Fields             []requestBodyFieldContext `json:"fields,omitempty"`
	RequiredFieldPaths []string                  `json:"required_field_paths,omitempty"`
}

type requestBodyFieldContext struct {
	Path        string `json:"path"`
	Required    bool   `json:"required"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type securityPromptContext struct {
	Schemes          []string `json:"schemes,omitempty"`
	CredentialFields []string `json:"credential_fields,omitempty"`
}

func DiscoverLocalAPIs(exampleDir, projectText string) ([]APIDocument, error) {
	var docs []APIDocument
	openAPIDir := filepath.Join(exampleDir, "openapi")
	if _, err := os.Stat(openAPIDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		candidates, err := openapidisco.LocalFiles(openAPIDir, exampleDir, projectText)
		if err != nil {
			return nil, err
		}
		var inventoryDocs []apitools.InventoryDocument
		for _, candidate := range candidates {
			inventoryDocs = append(inventoryDocs, apitools.InventoryDocument{
				Name:         strings.TrimSuffix(filepath.Base(candidate.RelativePath), filepath.Ext(candidate.RelativePath)),
				Path:         candidate.Path,
				RelativePath: candidate.RelativePath,
			})
		}
		openAPIDocs, err := apitools.BuildAuthoringAPIDocuments(context.Background(), apitools.AuthoringAPIDocumentOptions{
			Documents: inventoryDocs,
			BaseDir:   exampleDir,
			Query:     projectText,
		})
		if err != nil {
			return nil, err
		}
		docs = append(docs, openAPIDocs...)
	}
	discoveryDocs, err := discoverLocalGoogleDiscoveryAPIs(exampleDir)
	if err != nil {
		return nil, err
	}
	docs = append(docs, discoveryDocs...)
	smithyDocs, err := discoverLocalAWSSmithyAPIs(exampleDir)
	if err != nil {
		return nil, err
	}
	docs = append(docs, smithyDocs...)
	asyncDocs, err := discoverLocalAsyncAPIs(exampleDir)
	if err != nil {
		return nil, err
	}
	docs = append(docs, asyncDocs...)
	graphqlDocs, err := discoverLocalParsedSourceAPIs(exampleDir, "graphql", []string{".graphql", ".gql"}, apitools.ParseGraphQLOperationSummaries)
	if err != nil {
		return nil, err
	}
	docs = append(docs, graphqlDocs...)
	openRPCDocs, err := discoverLocalParsedSourceAPIs(exampleDir, "openrpc", []string{".json"}, apitools.ParseOpenRPCOperationSummaries)
	if err != nil {
		return nil, err
	}
	docs = append(docs, openRPCDocs...)
	grpcDocs, err := discoverLocalParsedSourceAPIs(exampleDir, "grpc-protobuf", []string{".proto"}, apitools.ParseGRPCProtobufOperationSummaries)
	if err != nil {
		return nil, err
	}
	docs = append(docs, grpcDocs...)
	odataDocs, err := discoverLocalParsedSourceAPIs(exampleDir, "odata", []string{".xml", ".json"}, apitools.ParseODataOperationSummaries)
	if err != nil {
		return nil, err
	}
	docs = append(docs, odataDocs...)
	sort.Slice(docs, func(i, j int) bool {
		if apiDocumentPriority(docs[i]) != apiDocumentPriority(docs[j]) {
			return apiDocumentPriority(docs[i]) < apiDocumentPriority(docs[j])
		}
		return docs[i].RelativePath < docs[j].RelativePath
	})
	return docs, nil
}

type operationSummaryParser func([]byte, string) ([]apitools.OperationSummary, error)

func discoverLocalParsedSourceAPIs(exampleDir, dirName string, extensions []string, parse operationSummaryParser) ([]APIDocument, error) {
	var docs []APIDocument
	sourceDir := filepath.Join(exampleDir, dirName)
	if _, err := os.Stat(sourceDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	allowedExt := map[string]bool{}
	for _, ext := range extensions {
		allowedExt[strings.ToLower(ext)] = true
	}
	if err := filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if isAdvisorySecuritySidecarPath(path) || !allowedExt[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(exampleDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		operations, err := parse(data, rel)
		if err != nil {
			return fmt.Errorf("parse %s %s: %w", dirName, path, err)
		}
		docs = append(docs, APIDocument{
			ID:           strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			Path:         path,
			RelativePath: rel,
			Title:        parsedSourceDocumentTitle(dirName, operations),
			Operations:   operations,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return docs, nil
}

func parsedSourceDocumentTitle(dirName string, operations []apitools.OperationSummary) string {
	for _, op := range operations {
		if title := strings.TrimSpace(op.DocumentName); title != "" {
			return title
		}
	}
	return dirName
}

func apiDocumentPriority(doc APIDocument) int {
	if isAdvisoryAPIDocument(doc) {
		return 0
	}
	return 1
}

func isAdvisoryAPIDocument(doc APIDocument) bool {
	text := strings.ToLower(strings.Join([]string{
		doc.RelativePath,
		doc.Title,
		doc.Description,
	}, " "))
	return strings.Contains(text, "advisory") || strings.Contains(text, "overlay")
}

func discoverLocalAWSSmithyAPIs(exampleDir string) ([]APIDocument, error) {
	var docs []APIDocument
	smithyDir := filepath.Join(exampleDir, "aws-smithy")
	if _, err := os.Stat(smithyDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(smithyDir, "*.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range matches {
		if isAdvisorySecuritySidecarPath(path) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		model, err := awssmithy.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("parse AWS Smithy %s: %w", path, err)
		}
		rel, err := filepath.Rel(exampleDir, path)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		docs = append(docs, APIDocument{
			ID:           strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			Path:         path,
			RelativePath: rel,
			Title:        firstNonEmpty(model.Title, model.ServiceID, model.AWSServiceID),
			Description:  model.Description,
			Operations:   awsSmithyOperationSummaries(rel, model),
		})
	}
	return docs, nil
}

func discoverLocalAsyncAPIs(exampleDir string) ([]APIDocument, error) {
	var docs []APIDocument
	asyncDir := filepath.Join(exampleDir, "asyncapi")
	if _, err := os.Stat(asyncDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if err := filepath.WalkDir(asyncDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json", ".yaml", ".yml":
		default:
			return nil
		}
		if isAdvisorySecuritySidecarPath(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(exampleDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		operations, err := apitools.ParseAsyncAPIOperationSummaries(data, rel)
		if err != nil {
			return fmt.Errorf("parse AsyncAPI %s: %w", path, err)
		}
		docs = append(docs, APIDocument{
			ID:           strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			Path:         path,
			RelativePath: rel,
			Title:        asyncAPIDocumentTitle(operations),
			Operations:   operations,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return docs, nil
}

func asyncAPIDocumentTitle(operations []apitools.OperationSummary) string {
	for _, op := range operations {
		if title := strings.TrimSpace(op.DocumentName); title != "" {
			return title
		}
	}
	return ""
}

func discoverLocalGoogleDiscoveryAPIs(exampleDir string) ([]APIDocument, error) {
	var docs []APIDocument
	for _, dirName := range []string{"google-discovery", "discovery"} {
		discoveryDir := filepath.Join(exampleDir, dirName)
		if _, err := os.Stat(discoveryDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		matches, err := filepath.Glob(filepath.Join(discoveryDir, "*.json"))
		if err != nil {
			return nil, err
		}
		for _, path := range matches {
			if isAdvisorySecuritySidecarPath(path) {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			model, err := googlediscovery.Parse(data)
			if err != nil {
				return nil, fmt.Errorf("parse Google Discovery %s: %w", path, err)
			}
			rel, err := filepath.Rel(exampleDir, path)
			if err != nil {
				return nil, err
			}
			doc := APIDocument{
				ID:           strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
				Path:         path,
				RelativePath: filepath.ToSlash(rel),
				Title:        firstNonEmpty(model.Title, model.Name),
				Description:  model.Description,
				Operations:   googleDiscoveryOperationSummaries(filepath.ToSlash(rel), model),
			}
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func googleDiscoveryOperationSummaries(relativePath string, model *googlediscovery.Model) []apitools.OperationSummary {
	if model == nil {
		return nil
	}
	var operations []apitools.OperationSummary
	for _, op := range model.Operations {
		if op == nil || strings.TrimSpace(op.OperationID) == "" {
			continue
		}
		summary := apitools.OperationSummary{
			ID:                   op.ID,
			DocumentName:         model.Name,
			DocumentPath:         relativePath,
			DocumentRelativePath: relativePath,
			OperationID:          op.OperationID,
			Method:               op.HTTPMethod,
			Path:                 op.Path,
			Summary:              op.Summary,
			Description:          op.Description,
			Tags:                 append([]string(nil), op.Tags...),
			Parameters:           googleDiscoveryParameterSummaries(op.Parameters),
			Provenance:           "google-discovery",
		}
		if op.RequestRef != "" || op.RequestMediaType != "" {
			summary.RequestBody = googleDiscoveryRequestBodySummary(model, op)
		}
		operations = append(operations, summary)
	}
	return operations
}

func googleDiscoveryRequestBodySummary(model *googlediscovery.Model, op *googlediscovery.Operation) *apitools.RequestBodySummary {
	if op == nil {
		return nil
	}
	body := &apitools.RequestBodySummary{
		Required:     strings.TrimSpace(op.RequestRef) != "",
		Ref:          op.RequestRef,
		ContentTypes: []string{firstNonEmpty(op.RequestMediaType, "application/json")},
	}
	if model == nil || strings.TrimSpace(op.RequestRef) == "" {
		return body
	}
	schema := model.Schemas[strings.TrimPrefix(op.RequestRef, "#/components/schemas/")]
	if len(schema) == 0 {
		return body
	}
	body.Description = stringFromMap(schema, "description")
	body.Fields = discoveryRequestFields(schema, op.OperationID)
	var required []string
	for _, field := range body.Fields {
		if field.Required {
			required = append(required, field.Path)
		}
	}
	body.RequiredFieldPaths = required
	return body
}

func discoveryRequestFields(schema map[string]any, operationID string) []apitools.RequestFieldSummary {
	props := anyMap(schema["properties"])
	if len(props) == 0 {
		return nil
	}
	required := stringSet(stringSliceFromAny(schema["required"]))
	var out []apitools.RequestFieldSummary
	for _, name := range sortedAnyMapKeys(props) {
		prop := anyMap(props[name])
		if len(prop) == 0 {
			continue
		}
		out = append(out, apitools.RequestFieldSummary{
			Path:        name,
			Required:    required[name] || discoveryPropertyRequiredForOperation(prop, operationID),
			Type:        stringFromMap(prop, "type"),
			Format:      stringFromMap(prop, "format"),
			Ref:         stringFromMap(prop, "$ref"),
			Description: stringFromMap(prop, "description"),
		})
	}
	return out
}

func discoveryPropertyRequiredForOperation(prop map[string]any, operationID string) bool {
	annotations := anyMap(prop["annotations"])
	for _, required := range stringSliceFromAny(annotations["required"]) {
		if discoveryOperationIDMatches(required, operationID) {
			return true
		}
	}
	return false
}

func discoveryOperationIDMatches(candidate, operationID string) bool {
	candidate = strings.TrimSpace(candidate)
	operationID = strings.TrimSpace(operationID)
	if candidate == "" || operationID == "" {
		return false
	}
	return candidate == operationID || slugIdent(candidate) == slugIdent(operationID)
}

func isAdvisorySecuritySidecarPath(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strings.HasSuffix(base, ".security") || strings.HasSuffix(base, ".security-overlay")
}

func sortedAnyMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func googleDiscoveryParameterSummaries(params []*googlediscovery.Parameter) []apitools.ParameterSummary {
	var out []apitools.ParameterSummary
	for _, param := range params {
		if param == nil {
			continue
		}
		out = append(out, apitools.ParameterSummary{
			Name:        param.Name,
			In:          param.Location,
			Description: param.Description,
			Required:    param.Required,
			Type:        stringFromMap(param.Schema, "type"),
			Format:      stringFromMap(param.Schema, "format"),
			Ref:         stringFromMap(param.Schema, "$ref"),
		})
	}
	return out
}

func awsSmithyOperationSummaries(relativePath string, model *awssmithy.Model) []apitools.OperationSummary {
	if model == nil {
		return nil
	}
	var operations []apitools.OperationSummary
	for _, op := range model.Operations {
		if op == nil || strings.TrimSpace(op.Name) == "" {
			continue
		}
		summary := apitools.OperationSummary{
			ID:                   op.ID,
			DocumentName:         firstNonEmpty(model.ServiceID, model.AWSServiceID),
			DocumentPath:         relativePath,
			DocumentRelativePath: relativePath,
			OperationID:          op.Name,
			Method:               op.Method,
			Path:                 firstNonEmpty(op.Path, op.URI),
			Parameters:           awsSmithyParameterSummaries(op),
			Provenance:           "aws-smithy",
		}
		if op.Payload != nil || len(op.UnboundInput) > 0 || len(op.StaticPayload) > 0 {
			summary.RequestBody = &apitools.RequestBodySummary{
				Ref:          op.Input,
				ContentTypes: []string{firstNonEmpty(op.RequestMediaType, "application/json")},
			}
		}
		operations = append(operations, summary)
	}
	return operations
}

func awsSmithyParameterSummaries(op *awssmithy.Operation) []apitools.ParameterSummary {
	if op == nil {
		return nil
	}
	var out []apitools.ParameterSummary
	for _, binding := range op.InputBindings {
		if binding == nil {
			continue
		}
		location := binding.Location
		switch location {
		case "queryParams":
			location = "query"
		case "prefixHeaders":
			location = "header"
		}
		if location == "payload" {
			continue
		}
		out = append(out, apitools.ParameterSummary{
			Name:     firstNonEmpty(binding.WireName, binding.MemberName),
			In:       location,
			Required: binding.Required,
			Type:     smithyTargetType(binding.Target),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].In != out[j].In {
			return out[i].In < out[j].In
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func smithyTargetType(target string) string {
	switch {
	case strings.Contains(target, "#Boolean"):
		return "boolean"
	case strings.Contains(target, "#Byte"), strings.Contains(target, "#Short"), strings.Contains(target, "#Integer"), strings.Contains(target, "#Long"):
		return "integer"
	case strings.Contains(target, "#Float"), strings.Contains(target, "#Double"), strings.Contains(target, "#BigInteger"), strings.Contains(target, "#BigDecimal"):
		return "number"
	case strings.Contains(target, "#Blob"):
		return "string"
	default:
		return "string"
	}
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func anySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	default:
		return nil
	}
}

func operationLabel(op apitools.OperationSummary) string {
	return apitools.OperationLabel(op)
}

func operationPrompt(op apitools.OperationSummary) operationPromptContext {
	return operationPromptContext{
		OperationID:    op.OperationID,
		Method:         op.Method,
		Path:           op.Path,
		Summary:        op.Summary,
		Description:    op.Description,
		Tags:           append([]string(nil), op.Tags...),
		Provenance:     op.Provenance,
		RequiredFields: apitools.RequiredOperationFields(op),
		Parameters:     parameterPromptContexts(op.Parameters),
		RequestBody:    requestBodyPrompt(op.RequestBody),
		Security:       securityPrompt(op.Security),
	}
}

func operationCatalog(docs []APIDocument) []operationCatalogDocumentContext {
	catalog := make([]operationCatalogDocumentContext, 0, len(docs))
	for _, doc := range docs {
		entry := operationCatalogDocumentContext{
			DocumentPath: doc.RelativePath,
			Title:        doc.Title,
			Description:  firstLine(doc.Description),
		}
		for _, op := range doc.Operations {
			if strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			entry.Operations = append(entry.Operations, operationCatalogEntryContext{
				OperationID: op.OperationID,
				Method:      op.Method,
				Path:        op.Path,
				Summary:     firstLine(op.Summary),
				Provenance:  op.Provenance,
			})
		}
		catalog = append(catalog, entry)
	}
	return catalog
}

func rankedDraftDocuments(request DraftRequest) []APIDocument {
	return apitools.RankAuthoringAPIDocuments(request.Docs, apitools.AuthoringOperationRankingOptions{
		Query:              draftRankingText(request),
		SelectedOperations: selectedOperationRefs(request.Session),
		Limit:              maxDraftOperationCandidates,
	})
}

type operationRankCandidate struct {
	docIndex int
	opIndex  int
	op       apitools.OperationSummary
	score    int
	selected bool
	rank     int
}

func rankOperationCandidates(request DraftRequest) []operationRankCandidate {
	query := rankingTokenWeights(draftRankingText(request))
	selected := selectedOperationKeys(request.Session)
	var candidates []operationRankCandidate
	for docIndex, doc := range request.Docs {
		selectedDoc := selectedDocument(request.Session, doc)
		for opIndex, op := range doc.Operations {
			key := operationCandidateKey(doc.RelativePath, op.OperationID)
			isSelected := selected[key] || selected["\x00"+op.OperationID]
			candidates = append(candidates, operationRankCandidate{
				docIndex: docIndex,
				opIndex:  opIndex,
				op:       op,
				score:    operationRankScore(query, doc, op, selectedDoc),
				selected: isSelected,
			})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].docIndex != candidates[j].docIndex {
			return candidates[i].docIndex < candidates[j].docIndex
		}
		return candidates[i].opIndex < candidates[j].opIndex
	})
	for i := range candidates {
		candidates[i].rank = i
	}
	return candidates
}

func operationRankScore(query map[string]int, doc APIDocument, op apitools.OperationSummary, selectedDoc bool) int {
	score := 0
	if selectedDoc {
		score += 20
	}
	if isAdvisoryAPIDocument(doc) {
		score += 15
	}
	if query["weather"] > 0 && query["current"] > 0 && operationLooksCurrentWeatherLookup(op) {
		score += 50
	}
	score += rankingMatchScore(query, doc.RelativePath+" "+doc.Title+" "+doc.Description, 2)
	score += rankingMatchScore(query, op.OperationID, 12)
	score += rankingMatchScore(query, op.Summary, 8)
	score += rankingMatchScore(query, op.Path, 7)
	score += rankingMatchScore(query, strings.Join(op.Tags, " "), 6)
	score += rankingMatchScore(query, op.Description, 5)
	score += rankingMatchScore(query, strings.Join(apitools.RequiredOperationFields(op), " "), 4)
	score += rankingMatchScore(query, operationMethodHints(op.Method), 3)
	for _, parameter := range op.Parameters {
		score += rankingMatchScore(query, parameter.Name+" "+parameter.In+" "+parameter.Type, 4)
		score += rankingMatchScore(query, parameter.Description, 2)
	}
	if op.RequestBody != nil {
		for _, field := range op.RequestBody.Fields {
			score += rankingMatchScore(query, field.Path+" "+field.Type, 3)
			score += rankingMatchScore(query, field.Description, 2)
		}
	}
	return score
}

func draftRankingText(request DraftRequest) string {
	var parts []string
	parts = append(parts, request.Opening)
	if request.Session.Intent.Workflow != nil {
		parts = append(parts, request.Session.Intent.Workflow.Name, request.Session.Intent.Workflow.Description)
	}
	parts = append(parts,
		intentAPISourceRef(request.Session.Intent),
		request.Session.Project.ProjectName,
		request.Session.Project.Goal,
		request.Session.Project.Inputs,
		request.Session.Project.Outputs,
		request.Session.Project.DataFlow,
		request.Session.Project.FunctionContracts,
		request.Session.Project.OpenAPI,
		request.Session.Project.Safety,
		request.Session.Project.Fallback,
		request.Session.Safety,
		request.Session.Fallback,
	)
	for _, turn := range request.TranscriptTurns {
		parts = append(parts, turn.Label, turn.Answer)
	}
	for _, issue := range request.ReadinessFeedback {
		parts = append(parts, issue.Code, issue.Slot, issue.Message, issue.SuggestedAnswer)
	}
	for _, input := range request.Session.Intent.Inputs {
		if input != nil {
			parts = append(parts, input.Name, input.Type, input.Description)
		}
	}
	for _, output := range request.Session.Intent.Outputs {
		if output != nil {
			parts = append(parts, output.Name, output.From, output.Description)
		}
	}
	walkSteps(request.Session.Intent.Steps, func(step *rollout.Step) {
		parts = append(parts, step.Name, step.Type, step.Do, firstNonEmpty(step.Source, step.OpenAPI), step.Operation)
		for field, value := range step.With {
			parts = append(parts, field, value)
		}
	})
	return strings.Join(parts, " ")
}

func selectedOperationKeys(session Session) map[string]bool {
	out := map[string]bool{}
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Operation) == "" {
			return
		}
		docPath := stepAPISourceRef(session, step)
		out[operationCandidateKey(docPath, step.Operation)] = true
		if docPath == "" {
			out["\x00"+step.Operation] = true
		}
	})
	return out
}

func selectedOperationRefs(session Session) []apitools.AuthoringOperationRef {
	var refs []apitools.AuthoringOperationRef
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Operation) == "" {
			return
		}
		refs = append(refs, apitools.AuthoringOperationRef{
			DocumentPath: stepAPISourceRef(session, step),
			OperationID:  step.Operation,
		})
	})
	return refs
}

func selectedDocument(session Session, doc APIDocument) bool {
	selected := intentAPISourceRef(session.Intent)
	if selected == "" {
		selected = strings.TrimSpace(session.Project.OpenAPI)
	}
	if selected == "" {
		return false
	}
	return selected == doc.RelativePath || strings.Contains(selected, doc.RelativePath)
}

func operationCandidateKey(docPath, operationID string) string {
	return docPath + "\x00" + operationID
}

func rankingTokenWeights(text string) map[string]int {
	out := map[string]int{}
	for _, token := range rankingTokens(text) {
		out[token]++
	}
	return out
}

func rankingMatchScore(query map[string]int, text string, weight int) int {
	if len(query) == 0 || strings.TrimSpace(text) == "" {
		return 0
	}
	score := 0
	for _, token := range rankingTokens(text) {
		score += query[token] * weight
	}
	return score
}

func rankingTokens(text string) []string {
	var normalized []rune
	var prev rune
	for i, r := range text {
		if unicode.IsUpper(r) && i > 0 && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
			normalized = append(normalized, ' ')
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized = append(normalized, unicode.ToLower(r))
		} else {
			normalized = append(normalized, ' ')
		}
		prev = r
	}
	seen := map[string]bool{}
	var out []string
	for _, token := range strings.Fields(string(normalized)) {
		if rankingStopwords[token] {
			continue
		}
		addRankingToken(&out, seen, token)
		if len(token) > 3 && strings.HasSuffix(token, "s") {
			addRankingToken(&out, seen, strings.TrimSuffix(token, "s"))
		}
	}
	return out
}

func addRankingToken(out *[]string, seen map[string]bool, token string) {
	if len(token) < 2 || rankingStopwords[token] || seen[token] {
		return
	}
	seen[token] = true
	*out = append(*out, token)
}

func operationMethodHints(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET":
		return "get fetch list read search lookup"
	case "POST":
		return "post create add write submit send"
	case "PUT", "PATCH":
		return "update edit change replace write"
	case "DELETE":
		return "delete remove archive"
	default:
		return method
	}
}

var rankingStopwords = map[string]bool{
	"a": true, "an": true, "and": true, "api": true, "as": true, "by": true, "for": true,
	"from": true, "in": true, "into": true, "of": true, "on": true, "or": true, "the": true,
	"this": true, "to": true, "use": true, "with": true, "workflow": true,
}

func parameterPromptContexts(parameters []apitools.ParameterSummary) []parameterPromptContext {
	if len(parameters) == 0 {
		return nil
	}
	out := make([]parameterPromptContext, 0, len(parameters))
	for _, parameter := range parameters {
		out = append(out, parameterPromptContext{
			Name:        parameter.Name,
			Location:    parameter.In,
			Required:    parameter.Required,
			Type:        parameter.Type,
			Description: parameter.Description,
		})
	}
	return out
}

func requestBodyPrompt(body *apitools.RequestBodySummary) *requestBodyPromptContext {
	if body == nil {
		return nil
	}
	out := &requestBodyPromptContext{
		Required:           body.Required,
		ContentType:        firstNonEmpty(body.ContentTypes...),
		Description:        body.Description,
		RequiredFieldPaths: append([]string(nil), body.RequiredFieldPaths...),
	}
	if body.Schema != nil {
		out.Type = body.Schema.Type
		if out.Description == "" {
			out.Description = body.Schema.Description
		}
	}
	for _, field := range body.Fields {
		out.Fields = append(out.Fields, requestBodyFieldContext{
			Path:        field.Path,
			Required:    field.Required,
			Type:        field.Type,
			Description: field.Description,
		})
	}
	return out
}

func securityPrompt(security []apitools.SecuritySummary) securityPromptContext {
	if len(security) == 0 {
		return securityPromptContext{}
	}
	var schemes []string
	var fields []string
	for _, scheme := range security {
		schemes = append(schemes, scheme.Name)
		if field := apitools.SecurityCredentialFieldName(scheme); field != "" {
			fields = append(fields, field)
		}
	}
	return securityPromptContext{
		Schemes:          dedupeStrings(schemes),
		CredentialFields: dedupeStrings(fields),
	}
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func securityFieldName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "api") && strings.Contains(lower, "key") {
		return camelToSnake(name)
	}
	if strings.Contains(lower, "bearer") || strings.Contains(lower, "auth") || strings.Contains(lower, "token") {
		return "Authorization"
	}
	return camelToSnake(name)
}

func camelToSnake(value string) string {
	var out []rune
	var prev rune
	for i, r := range value {
		if r == '-' || r == ' ' {
			r = '_'
		}
		if unicode.IsUpper(r) {
			if i > 0 && prev != '_' && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
				out = append(out, '_')
			}
			r = unicode.ToLower(r)
		}
		out = append(out, r)
		prev = r
	}
	return slugIdent(string(out))
}

func operationByID(docs []APIDocument, docPath, operationID string) (*apitools.OperationSummary, bool) {
	for _, doc := range docs {
		if doc.RelativePath != docPath {
			continue
		}
		for i := range doc.Operations {
			if doc.Operations[i].OperationID == operationID {
				return &doc.Operations[i], true
			}
		}
	}
	return nil, false
}
