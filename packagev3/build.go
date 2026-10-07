package packagev3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"errors"
	"github.com/OpenUdon/uws/uws1"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/credentialpolicy"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/trust"
	"github.com/OpenUdon/openudon/wire"
	"github.com/OpenUdon/uws/binding"
	"github.com/OpenUdon/uws/expressions"
)

var ErrPackage = errors.New("invalid, unsafe or bounded v3 package")

type SourceInput struct {
	ID    string
	Kind  string
	Path  string
	Bytes []byte
}

// BuildOptions contains only explicit reviewed bytes. Privacy is a host policy
// predicate, never a credential loader. The host must bind and isolate its calls.
type BuildOptions struct {
	Scope        string
	WorkflowYAML []byte
	DataJSON     []byte
	Sources      []SourceInput
	Private      func([]byte) bool
}
type Package struct {
	Manifest   Manifest
	Assessment Assessment
	Handoff    Handoff
	Files      map[string][]byte
	SHA256     string
}

func hashBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func artifactFor(path string, data []byte) Artifact {
	return Artifact{Path: path, SHA256: hashBytes(data)}
}

// Build constructs an independent v3 byte snapshot without synthesis, filesystem
// access, approval or runtime binding. Even incompatible/pending workflows can
// be packaged for review; Assess returns their closed refusal outcome.
func Build(ctx context.Context, options BuildOptions) (Package, error) {
	if ctx == nil {
		return Package{}, ErrPackage
	}
	if ctx.Err() != nil {
		return Package{}, ctx.Err()
	}
	if !scopeValid(options.Scope) || len(options.Sources) > MaxSources {
		return Package{}, ErrPackage
	}
	files := map[string][]byte{}
	total := 0
	put := func(path string, data []byte) error {
		if !publicPath(path) || len(data) == 0 || len(data) > MaxFileBytes || len(files) >= MaxFiles || len(data) > MaxTotalBytes-total {
			return ErrPackage
		}
		if _, exists := files[path]; exists {
			return ErrPackage
		}
		if credentialpolicy.ContainsLikelyValue(data) || options.Private != nil && options.Private(data) {
			return ErrPackage
		}
		files[path] = append([]byte(nil), data...)
		total += len(data)
		return nil
	}
	if put(WorkflowPath, options.WorkflowYAML) != nil || put(DataPath, options.DataJSON) != nil {
		return Package{}, ErrPackage
	}
	var data map[string]any
	if wire.DecodeStrictNumbers(files[DataPath], &data) != nil || data == nil {
		return Package{}, ErrPackage
	}
	if _, _, err := DecodeWorkflow(ctx, files[WorkflowPath]); err != nil {
		if ctx.Err() != nil {
			return Package{}, ctx.Err()
		}
		return Package{}, ErrPackage
	}
	manifest := Manifest{Version: PackageVersion, Scope: options.Scope, Workflow: artifactFor(WorkflowPath, files[WorkflowPath]), Data: artifactFor(DataPath, files[DataPath]), ShapeVersion: ShapeVersion, Sources: []Source{}}
	sourceOptions := apitools.OperationShapeOptions{MaxBytes: MaxFileBytes}
	for _, source := range options.Sources {
		if !authorityID(source.ID) || !kindValid(source.Kind) || put(source.Path, source.Bytes) != nil {
			return Package{}, ErrPackage
		}
		manifest.Sources = append(manifest.Sources, Source{ID: source.ID, Kind: source.Kind, Artifact: artifactFor(source.Path, files[source.Path])})
		sourceOptions.Sources = append(sourceOptions.Sources, apitools.ShapeSourceInput{ID: source.ID, OperationSourceInput: apitools.OperationSourceInput{Kind: apitools.OperationSourceKind(source.Kind), Content: files[source.Path]}})
	}
	var table binding.ShapeTable
	var err error
	if len(sourceOptions.Sources) == 0 {
		table = binding.ShapeTable{Version: ShapeVersion, Sources: []binding.Source{}, Operations: []binding.OperationShape{}}
	} else {
		table, err = apitools.BuildOperationShapeTable(ctx, sourceOptions)
		if err == nil {
			err = apitools.VerifyOperationShapeTable(ctx, sourceOptions, table)
		}
	}
	if ctx.Err() != nil {
		return Package{}, ctx.Err()
	}
	if err != nil {
		return Package{}, ErrPackage
	}
	shapes, err := table.Marshal()
	if err != nil || put(ShapesPath, shapes) != nil {
		return Package{}, ErrPackage
	}
	manifest.Shapes = artifactFor(ShapesPath, files[ShapesPath])
	sort.Slice(manifest.Sources, func(i, j int) bool { return manifest.Sources[i].ID < manifest.Sources[j].ID })
	bytes, err := manifest.Marshal()
	if err != nil || put(ManifestPath, bytes) != nil {
		return Package{}, ErrPackage
	}
	assessment, err := Assess(ctx, manifest, files)
	if err != nil {
		return Package{}, err
	}
	bytes, err = assessment.Marshal()
	if err != nil || put(AssessmentPath, bytes) != nil {
		return Package{}, ErrPackage
	}
	review := Handoff{Version: HandoffVersion, Scope: options.Scope, InputsSHA256: assessment.InputsSHA256, ManifestSHA256: hashBytes(files[ManifestPath]), AssessmentSHA256: hashBytes(files[AssessmentPath]), ReviewState: "review_required", Credentials: []string{}, Artifacts: []Artifact{}}
	for path, data := range files {
		review.Artifacts = append(review.Artifacts, artifactFor(path, data))
	}
	sort.Slice(review.Artifacts, func(i, j int) bool { return review.Artifacts[i].Path < review.Artifacts[j].Path })
	bytes, err = review.Marshal()
	if err != nil || put(HandoffPath, bytes) != nil {
		return Package{}, ErrPackage
	}
	digestFiles := make([]handoff.DigestFile, 0, len(files))
	for path, data := range files {
		digestFiles = append(digestFiles, handoff.DigestFile{Path: path, SHA256: hashBytes(data)})
	}
	digest, err := handoff.DigestFiles(options.Scope, trust.PackageDigestVersion, digestFiles)
	if err != nil || ctx.Err() != nil {
		if ctx.Err() != nil {
			return Package{}, ctx.Err()
		}
		return Package{}, ErrPackage
	}
	return Package{Manifest: manifest, Assessment: assessment, Handoff: review, Files: files, SHA256: digest}, nil
}
func authorityID(value string) bool { return authority.Identifier(value) }

// Assess independently reproduces input identities and API shapes. It supplies
// structural/binding review only: compatible never grants executable authority.
func Assess(ctx context.Context, manifest Manifest, files map[string][]byte) (Assessment, error) {
	if ctx == nil || manifest.Validate() != nil {
		return Assessment{}, ErrPackage
	}
	if ctx.Err() != nil {
		return Assessment{}, ctx.Err()
	}
	inputs, err := manifest.InputArtifacts()
	if err != nil {
		return Assessment{}, ErrPackage
	}
	total := 0
	sourceOptions := apitools.OperationShapeOptions{MaxBytes: MaxFileBytes}
	for _, input := range inputs {
		data, ok := files[input.Path]
		if !ok || len(data) == 0 || len(data) > MaxFileBytes || len(data) > MaxTotalBytes-total || hashBytes(data) != input.SHA256 || credentialpolicy.ContainsLikelyValue(data) {
			return Assessment{}, ErrPackage
		}
		total += len(data)
	}
	var data map[string]any
	if wire.DecodeStrictNumbers(files[DataPath], &data) != nil || data == nil {
		return Assessment{}, ErrPackage
	}
	for _, source := range manifest.Sources {
		sourceOptions.Sources = append(sourceOptions.Sources, apitools.ShapeSourceInput{ID: source.ID, OperationSourceInput: apitools.OperationSourceInput{Kind: apitools.OperationSourceKind(source.Kind), Content: files[source.Artifact.Path]}})
	}
	table, err := binding.ParseTable(files[ShapesPath])
	if err != nil {
		return Assessment{}, ErrPackage
	}
	if len(sourceOptions.Sources) > 0 {
		if apitools.VerifyOperationShapeTable(ctx, sourceOptions, table) != nil {
			if ctx.Err() != nil {
				return Assessment{}, ctx.Err()
			}
			return Assessment{}, ErrPackage
		}
	} else if len(table.Sources) != 0 || len(table.Operations) != 0 {
		return Assessment{}, ErrPackage
	}
	encoded, err := manifest.Marshal()
	if err != nil {
		return Assessment{}, ErrPackage
	}
	sum, err := manifest.InputDigest()
	if err != nil {
		return Assessment{}, ErrPackage
	}
	assessment := Assessment{Version: AssessmentVersion, Scope: manifest.Scope, InputsSHA256: sum, ManifestSHA256: hashBytes(encoded), Outcome: "compatible", Findings: []Finding{}}
	add := func(code string, outcome string) {
		if len(assessment.Findings) < 128 {
			assessment.Findings = append(assessment.Findings, Finding{Code: code, Path: WorkflowPath, Outcome: outcome})
		} else {
			assessment.Truncated = true
		}
		if outcome == "incompatible" || outcome == "indeterminate" && assessment.Outcome == "compatible" {
			assessment.Outcome = outcome
		}
	}
	doc, raw, err := DecodeWorkflow(ctx, files[WorkflowPath])
	if err != nil {
		if ctx.Err() != nil {
			return Assessment{}, ctx.Err()
		}
		return Assessment{}, ErrPackage
	}
	if validateWorkflowSchema(raw) != nil || doc.Validate() != nil {
		add("workflow.invalid", "incompatible")
		return assessment, nil
	}
	if len(expressions.CheckPortability(doc)) > 0 {
		add("workflow.nonportable", "incompatible")
	}
	if doc.ValidateExecutable() != nil {
		add("workflow.not_executable", "incompatible")
	}
	if doc.ValidateExecutionEntrypoint() != nil {
		add("workflow.entrypoint", "incompatible")
	}
	sourceByID := map[string]Source{}
	for _, s := range manifest.Sources {
		sourceByID[s.ID] = s
	}
	if len(doc.SourceDescriptions) != len(manifest.Sources) {
		add("source.inventory", "incompatible")
	}
	for _, s := range doc.SourceDescriptions {
		source, ok := sourceByID[s.Name]
		if !ok || string(s.EffectiveType()) != source.Kind || s.URL != source.Artifact.Path {
			add("source.inventory", "incompatible")
		}
	}
	resolver, err := binding.NewResolver(table)
	if err != nil {
		return Assessment{}, ErrPackage
	}
	for _, op := range doc.Operations {
		if !op.HasSourceBinding() {
			add("binding.profile_unproved", "indeterminate")
			continue
		}
		source, ok := sourceByID[op.SourceDescription]
		if !ok {
			add("binding.source_missing", "incompatible")
			continue
		}
		selectorKind, selector := "id", op.SourceOperationID
		if selector == "" {
			selector = op.OpenAPIOperationID
		}
		if op.SourceOperationRef != "" {
			selectorKind, selector = "ref", op.SourceOperationRef
		}
		if op.OpenAPIOperationRef != "" {
			selectorKind, selector = "ref", op.OpenAPIOperationRef
		}
		request := binding.Request{Binding: binding.Binding{Source: binding.Source{ID: source.ID, Kind: source.Kind, SHA256: source.Artifact.SHA256}, SelectorKind: selectorKind, SelectorValue: selector}, ExpressionContext: expressions.Context{Version: doc.UWS}}
		for location, value := range op.Request {
			if location == "body" {
				request.Inputs = append(request.Inputs, binding.BoundInput{Location: location, Name: "body", Value: value})
				continue
			}
			fields, ok := value.(map[string]any)
			if !ok {
				add("binding.request_unproved", "indeterminate")
				continue
			}
			for name, value := range fields {
				request.Inputs = append(request.Inputs, binding.BoundInput{Location: location, Name: name, Value: value})
			}
		}
		sort.Slice(request.Inputs, func(i, j int) bool {
			a, b := request.Inputs[i], request.Inputs[j]
			return a.Location+"/"+a.Name < b.Location+"/"+b.Name
		})
		for _, output := range op.Outputs {
			parsed, err := expressions.Parse(output, expressions.Context{Version: doc.UWS, Field: expressions.Value})
			if err != nil {
				continue
			}
			source := parsed.Source()
			if source == "$response.body" || strings.HasPrefix(source, "$response.body#") {
				pointer := strings.TrimPrefix(source, "$response.body")
				request.OutputReferences = append(request.OutputReferences, binding.OutputReference{Location: "body", Name: "body", Pointer: pointer})
			} else if strings.HasPrefix(source, "$response.header.") {
				request.OutputReferences = append(request.OutputReferences, binding.OutputReference{Location: "header", Name: strings.TrimPrefix(source, "$response.header.")})
			}
		}
		report, err := binding.ValidateBinding(ctx, resolver, request)
		if err != nil {
			if ctx.Err() != nil {
				return Assessment{}, ctx.Err()
			}
			return Assessment{}, ErrPackage
		}
		for _, finding := range report.Diagnostics {
			add(finding.Code, string(finding.Outcome))
		}
		if report.Outcome != "compatible" && len(report.Diagnostics) == 0 {
			add("binding.unproved", string(report.Outcome))
		}
	}
	var walkSteps func([]*uws1.Step)
	walkSteps = func(steps []*uws1.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			if len(step.Inputs) > 0 || len(step.Body) > 0 {
				add("binding.step_values_unproved", "indeterminate")
			}
			walkSteps(step.Steps)
			walkSteps(step.Default)
			for _, branch := range step.Cases {
				if branch != nil {
					walkSteps(branch.Steps)
				}
			}
		}
	}
	for _, workflow := range doc.Workflows {
		walkSteps(workflow.Steps)
		walkSteps(workflow.Default)
		for _, branch := range workflow.Cases {
			if branch != nil {
				walkSteps(branch.Steps)
			}
		}
	}
	// Never expose flow document paths or parser excerpts: map observations to
	// stable artifact paths. Warnings remain explicit, not executable proof.
	flow, err := binding.AnalyzeFlow(ctx, doc)
	if err != nil {
		return Assessment{}, err
	}
	for _, finding := range flow.Findings {
		outcome := "compatible"
		switch finding.Code {
		case "flow.effect_unknown", "flow.loop_unbounded", "flow.entry_indeterminate", "flow.depth":
			outcome = "indeterminate"
		}
		add(finding.Code, outcome)
	}
	if flow.Truncated {
		assessment.Truncated = true
		add("flow.truncated", "indeterminate")
	}
	return assessment, nil
}
