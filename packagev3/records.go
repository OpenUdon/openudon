// Package packagev3 owns explicit-byte non-browser workflow packages. Record
// validation proves structure/identity only, never source semantics or authority.
package packagev3

import (
	"encoding/json"
	"errors"
	"github.com/OpenUdon/evidence/artifact"
	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/trust"
	"github.com/OpenUdon/openudon/wire"
	"reflect"
	"sort"
	"strings"
)

const PackageVersion = "openudon.package.v3"
const HandoffVersion = "openudon.review-handoff.v3"
const AssessmentVersion = "openudon.assessment.v3"
const ShapeVersion = "uws.shape-table.v1"
const WorkflowPath = "workflows/workflow.uws.yaml"
const DataPath = "expected/data.json"
const ShapesPath = "expected/operation-shapes.json"
const ManifestPath = "expected/package.json"
const AssessmentPath = "expected/assessment.json"
const HandoffPath = "expected/review-handoff.json"
const MaxFiles = 512
const MaxFileBytes = 8 << 20
const MaxTotalBytes = 32 << 20
const MaxSources = 33

var ErrRecord = errors.New("invalid v3 package record")

type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Source struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Artifact Artifact `json:"artifact"`
}
type Manifest struct {
	Version      string   `json:"version"`
	Scope        string   `json:"scope"`
	Workflow     Artifact `json:"workflow"`
	Data         Artifact `json:"data"`
	Shapes       Artifact `json:"shapes"`
	ShapeVersion string   `json:"shape_version"`
	Sources      []Source `json:"sources"`
}
type Handoff struct {
	Version          string     `json:"version"`
	Scope            string     `json:"scope"`
	InputsSHA256     string     `json:"inputs_sha256"`
	ManifestSHA256   string     `json:"manifest_sha256"`
	AssessmentSHA256 string     `json:"assessment_sha256"`
	Artifacts        []Artifact `json:"artifacts"`
	ReviewState      string     `json:"review_state"`
	Credentials      []string   `json:"credentials"`
}
type Finding struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Outcome string `json:"outcome"`
}
type Assessment struct {
	Version        string    `json:"version"`
	Scope          string    `json:"scope"`
	InputsSHA256   string    `json:"inputs_sha256"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	Outcome        string    `json:"outcome"`
	Findings       []Finding `json:"findings"`
	Truncated      bool      `json:"truncated"`
}

func digestValid(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func publicPath(value string) bool {
	clean, err := artifact.CleanRelativePath(value, artifact.Options{})
	return err == nil && clean == value && len(value) <= 256 && !strings.HasSuffix(strings.ToLower(value), ".hcl") && !strings.Contains(value, ".icot") && !strings.HasPrefix(value, "browser/") && !strings.HasPrefix(value, "browser-")
}
func (a Artifact) Valid() bool { return publicPath(a.Path) && digestValid(a.SHA256) }
func scopeValid(scope string) bool {
	clean, err := artifact.CleanRelativePath(scope, artifact.Options{})
	return err == nil && clean == scope && len(scope) <= 2048
}
func kindValid(kind string) bool {
	switch kind {
	case "openapi", "google-discovery", "aws-smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "odata", RuntimeSourceKind:
		return true
	}
	return false
}

func (m Manifest) Validate() error {
	if m.Version != PackageVersion || !scopeValid(m.Scope) || m.ShapeVersion != ShapeVersion || m.Workflow.Path != WorkflowPath || m.Data.Path != DataPath || m.Shapes.Path != ShapesPath || !m.Workflow.Valid() || !m.Data.Valid() || !m.Shapes.Valid() || m.Sources == nil || len(m.Sources) > MaxSources {
		return ErrRecord
	}
	apiSources, runtimeSources := 0, 0
	ids := map[string]bool{}
	paths := map[string]bool{WorkflowPath: true, DataPath: true, ShapesPath: true, ManifestPath: true, AssessmentPath: true, HandoffPath: true}
	for _, source := range m.Sources {
		if !authority.Identifier(source.ID) || !kindValid(source.Kind) || !source.Artifact.Valid() || ids[source.ID] || paths[source.Artifact.Path] || !strings.HasPrefix(source.Artifact.Path, "sources/"+source.Kind+"/") {
			return ErrRecord
		}
		if source.Kind == RuntimeSourceKind {
			runtimeSources++
			if source.ID != RuntimeSourceID {
				return ErrRecord
			}
		} else {
			apiSources++
		}
		if apiSources > MaxAPISources || runtimeSources > 1 {
			return ErrRecord
		}
		ids[source.ID] = true
		paths[source.Artifact.Path] = true
	}
	return nil
}

// InputArtifacts is an independent sorted snapshot of exactly the reviewed
// YAML/data/shape/source inputs. Reports/handoff have separate identities.
func (m Manifest) InputArtifacts() ([]Artifact, error) {
	if m.Validate() != nil {
		return nil, ErrRecord
	}
	files := []Artifact{m.Workflow, m.Data, m.Shapes}
	for _, s := range m.Sources {
		files = append(files, s.Artifact)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// InputDigest retains the existing digest-v1 envelope/scope/ordering rules.
// It hashes metadata; later verification must independently reproduce sources.
func (m Manifest) InputDigest() (string, error) {
	files, err := m.InputArtifacts()
	if err != nil {
		return "", err
	}
	var digestFiles []handoff.DigestFile
	for _, f := range files {
		digestFiles = append(digestFiles, handoff.DigestFile{Path: f.Path, SHA256: f.SHA256})
	}
	sum, err := handoff.DigestFiles(m.Scope, trust.PackageDigestVersion, digestFiles)
	if err != nil {
		return "", ErrRecord
	}
	return sum, nil
}

func (h Handoff) Validate() error {
	if h.Version != HandoffVersion || !scopeValid(h.Scope) || !digestValid(h.InputsSHA256) || !digestValid(h.ManifestSHA256) || !digestValid(h.AssessmentSHA256) || h.ReviewState != "review_required" || len(h.Artifacts) < 5 || len(h.Artifacts) > MaxFiles || h.Credentials == nil || len(h.Credentials) > 64 {
		return ErrRecord
	}
	paths := map[string]bool{}
	previous := ""
	for _, f := range h.Artifacts {
		if !f.Valid() || paths[f.Path] || f.Path <= previous || f.Path == HandoffPath {
			return ErrRecord
		}
		paths[f.Path] = true
		previous = f.Path
		if f.Path == ManifestPath && f.SHA256 != h.ManifestSHA256 || f.Path == AssessmentPath && f.SHA256 != h.AssessmentSHA256 {
			return ErrRecord
		}
	}
	for _, p := range []string{WorkflowPath, DataPath, ShapesPath, ManifestPath, AssessmentPath} {
		if !paths[p] {
			return ErrRecord
		}
	}
	seen := map[string]bool{}
	for _, name := range h.Credentials {
		if !authority.Identifier(name) || seen[name] {
			return ErrRecord
		}
		seen[name] = true
	}
	return nil
}
func (a Assessment) Validate() error {
	if a.Version != AssessmentVersion || !scopeValid(a.Scope) || !digestValid(a.InputsSHA256) || !digestValid(a.ManifestSHA256) || a.Findings == nil || len(a.Findings) > 128 {
		return ErrRecord
	}
	if a.Outcome != "compatible" && a.Outcome != "incompatible" && a.Outcome != "indeterminate" {
		return ErrRecord
	}
	for _, f := range a.Findings {
		if len(f.Code) == 0 || len(f.Code) > 64 || f.Path != "" && !publicPath(f.Path) || f.Outcome != "compatible" && f.Outcome != "incompatible" && f.Outcome != "indeterminate" {
			return ErrRecord
		}
		for _, c := range f.Code {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
				return ErrRecord
			}
		}
	}
	return nil
}

func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if decodeRecord(data, &m) != nil || m.Validate() != nil {
		return Manifest{}, ErrRecord
	}
	return m, nil
}
func ParseHandoff(data []byte) (Handoff, error) {
	var h Handoff
	if decodeRecord(data, &h) != nil || h.Validate() != nil {
		return Handoff{}, ErrRecord
	}
	return h, nil
}
func ParseAssessment(data []byte) (Assessment, error) {
	var a Assessment
	if decodeRecord(data, &a) != nil || a.Validate() != nil {
		return Assessment{}, ErrRecord
	}
	return a, nil
}

// Keep canonical field spellings, required fields and arrays aligned with the
// closed public schemas. SDK wire owns byte/node/depth/duplicate limits.
func decodeRecord(data []byte, out any) error {
	if wire.DecodeStrictNumbers(data, out) != nil {
		return ErrRecord
	}
	return recordShape(data, reflect.TypeOf(out))
}
func recordShape(data []byte, t reflect.Type) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if strings.TrimSpace(string(data)) == "null" {
		return ErrRecord
	}
	switch t.Kind() {
	case reflect.Struct:
		var object map[string]json.RawMessage
		if json.Unmarshal(data, &object) != nil || object == nil || len(object) != t.NumField() {
			return ErrRecord
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			value, ok := object[name]
			if !ok || recordShape(value, field.Type) != nil {
				return ErrRecord
			}
		}
	case reflect.Slice:
		var values []json.RawMessage
		if json.Unmarshal(data, &values) != nil || values == nil {
			return ErrRecord
		}
		for _, value := range values {
			if recordShape(value, t.Elem()) != nil {
				return ErrRecord
			}
		}
	}
	return nil
}
func (m Manifest) Marshal() ([]byte, error) {
	if m.Validate() != nil {
		return nil, ErrRecord
	}
	return json.Marshal(m)
}
func (h Handoff) Marshal() ([]byte, error) {
	if h.Validate() != nil {
		return nil, ErrRecord
	}
	return json.Marshal(h)
}
func (a Assessment) Marshal() ([]byte, error) {
	if a.Validate() != nil {
		return nil, ErrRecord
	}
	return json.Marshal(a)
}
