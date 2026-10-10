package packagev3

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/uws/binding"
	"github.com/OpenUdon/uws/browserauthentication"
	"github.com/OpenUdon/uws/browserregistration"
	"github.com/OpenUdon/uws/uws1"
)

const BrowserPath = "expected/browser.json"
const BrowserSupplementVersion = "openudon.browser-supplement.v1"

type BrowserSupplementV1 = browsercontract.BrowserSupplementV1
type BrowserCallV1 = browsercontract.BrowserCallV1

// BrowserPermission is a human-reviewed per-leaf session choice. Zero values
// select fresh contexts. The host separately confirms/enforces actual custody.
type BrowserPermission struct {
	ReuseAllowed bool
	SaveAllowed  bool
}
type BrowserReviewInput struct {
	Path  string
	Bytes []byte
}
type BrowserBuildOptions struct {
	Permissions map[string]BrowserPermission // exact browser step IDs only
	Reviews     []BrowserReviewInput
}

func browserRecipeSource(s Source, table binding.ShapeTable) bool {
	if s.Kind != "browser-profile" {
		return false
	}
	for _, op := range table.Operations {
		if op.Source.ID == s.ID && op.Browser != nil {
			return op.Browser.CallKind != "action"
		}
	}
	return false
}
func findBrowserCall(s *BrowserSupplementV1, operation string) *BrowserCallV1 {
	if s != nil {
		for i := range s.Calls {
			if s.Calls[i].OperationID == operation {
				return &s.Calls[i]
			}
		}
	}
	return nil
}
func browserOperationBinding(op *uws1.Operation, sources []Source) (binding.Binding, error) {
	b, err := operationBinding(op, sources)
	if err != nil {
		return b, err
	}
	if b.Source.Kind == "browser-profile" && b.SelectorKind == "ref" {
		const prefix = "#/actions/"
		if !strings.HasPrefix(b.SelectorValue, prefix) || strings.ContainsAny(strings.TrimPrefix(b.SelectorValue, prefix), "/~") {
			return binding.Binding{}, ErrPackage
		}
		b.SelectorKind, b.SelectorValue = "id", strings.TrimPrefix(b.SelectorValue, prefix)
	}
	return b, nil
}
func sourceForBrowserPath(path string, sources []Source) (Source, error) {
	for _, s := range sources {
		if s.Kind == "browser-profile" && s.Artifact.Path == path {
			return s, nil
		}
	}
	return Source{}, ErrPackage
}
func strictBrowserExtension(value any, out any) error {
	data, err := json.Marshal(value)
	if err != nil || browsercontract.Decode(data, out) != nil {
		return ErrPackage
	}
	return nil
}
func browserCall(ctx context.Context, step *uws1.Step, op *uws1.Operation, sources []Source, resolver binding.Resolver, sessions map[string]string) (*BrowserCallV1, error) {
	var selected binding.Binding
	credentials := map[string]string{}
	call := BrowserCallV1{StepID: step.StepID, OperationID: op.OperationID, InvocationID: step.StepID, CredentialSlots: []browsercontract.CredentialSlotBinding{}}
	switch {
	case op.Extensions[browserauthentication.ExtensionAuthentication] != nil:
		if op.HasSourceBinding() || len(op.Extensions) != 2 || (op.ExtensionProfile() != browserauthentication.CallProfileName && op.ExtensionProfile() != browserauthentication.ContextCallProfileName) {
			return nil, ErrPackage
		}
		var ext browserauthentication.OperationAuthentication
		if strictBrowserExtension(op.Extensions[browserauthentication.ExtensionAuthentication], &ext) != nil {
			return nil, ErrPackage
		}
		source, err := sourceForBrowserPath(ext.Profile, sources)
		if err != nil {
			return nil, err
		}
		selected = binding.Binding{Source: binding.Source{ID: source.ID, Kind: source.Kind, SHA256: source.Artifact.SHA256}, SelectorKind: "id", SelectorValue: ext.Flow}
		credentials = ext.CredentialBindings
		call.SessionName = ext.Session
	case op.Extensions[browserregistration.ExtensionRegistration] != nil:
		if op.HasSourceBinding() || len(op.Extensions) != 2 || !strings.HasPrefix(op.ExtensionProfile(), "uws.browser-registration-call.") {
			return nil, ErrPackage
		}
		var ext browserregistration.OperationRegistration
		if strictBrowserExtension(op.Extensions[browserregistration.ExtensionRegistration], &ext) != nil || ext.Approval == "" || ext.DuplicatePrevention != "operator_attestation" || ext.OnDuplicate != "fail" || ext.AmbiguousOutcome != "stop_without_retry" || ext.CleanupDisposition == "" {
			return nil, ErrPackage
		}
		source, err := sourceForBrowserPath(ext.Profile, sources)
		if err != nil {
			return nil, err
		}
		selected = binding.Binding{Source: binding.Source{ID: source.ID, Kind: source.Kind, SHA256: source.Artifact.SHA256}, SelectorKind: "id", SelectorValue: ext.Flow}
		credentials = ext.CredentialBindings
		call.RegistrationInputBinding = ext.InputBinding
	case op.HasSourceBinding():
		var err error
		selected, err = browserOperationBinding(op, sources)
		if err != nil {
			return nil, err
		}
		if selected.Source.Kind != "browser-profile" {
			return nil, nil
		}
		if len(op.Extensions) > 0 {
			if len(op.Extensions) != 1 || op.Extensions[browserauthentication.ExtensionSession] == nil {
				return nil, ErrPackage
			}
			var ext browserauthentication.OperationSession
			if strictBrowserExtension(op.Extensions[browserauthentication.ExtensionSession], &ext) != nil {
				return nil, ErrPackage
			}
			call.SessionName = ext.Session
		}
	default:
		return nil, nil
	}
	resolution, err := resolver.Resolve(ctx, selected)
	if err != nil || resolution.Status != binding.Resolved || resolution.Shape == nil || resolution.Shape.Browser == nil {
		return nil, ErrPackage
	}
	shape := resolution.Shape
	b := shape.Browser
	call.Kind = b.CallKind
	call.SourceID = shape.Source.ID
	call.SourceSHA256 = shape.Source.SHA256
	call.ProfileVersion = b.ProfileVersion
	call.Selector = browsercontract.BrowserSelector{Kind: shape.Selector.Kind, Value: shape.Selector.Value, Key: shape.Selector.Key}
	call.SelectedSHA256 = b.SelectedSHA256
	call.Origins = append([]string{}, b.Origins...)
	call.Effects = append(json.RawMessage{}, b.Effects...)
	if len(b.ConfirmationPolicy) > 0 {
		call.ConfirmationPolicySHA256 = browsercontract.SHA256(b.ConfirmationPolicy)
	}
	call.OuterProtocol, call.InnerProtocol = browsercontract.Protocols(call.ProfileVersion)
	if call.OuterProtocol == "" {
		return nil, ErrPackage
	}
	if call.Kind == "authentication" {
		if !authorityID(call.SessionName) || sessions[call.SessionName] != "" {
			return nil, ErrPackage
		}
		if op.ExtensionProfile() != strings.Replace(call.ProfileVersion, "browser-authentication.", "browser-authentication-call.", 1) {
			return nil, ErrPackage
		}
		sessions[call.SessionName] = call.SourceID
	} else if call.Kind == "registration" {
		if op.ExtensionProfile() != strings.Replace(call.ProfileVersion, "browser-registration.", "browser-registration-call.", 1) || (call.ProfileVersion != "uws.browser-registration.1.0" && !authorityID(call.RegistrationInputBinding)) {
			return nil, ErrPackage
		}
	} else {
		if b.AuthenticationRequired && call.SessionName == "" {
			return nil, ErrPackage
		}
		if call.SessionName != "" {
			call.AuthenticationSourceID = sessions[call.SessionName]
			if call.AuthenticationSourceID == "" {
				return nil, ErrPackage
			}
		}
	}
	if len(credentials) != len(b.CredentialSlots) {
		return nil, ErrPackage
	}
	for _, slot := range b.CredentialSlots {
		name, ok := credentials[slot.Name]
		if !ok || !authorityID(name) {
			return nil, ErrPackage
		}
		call.CredentialSlots = append(call.CredentialSlots, browsercontract.CredentialSlotBinding{Slot: slot.Name, Name: name})
	}
	return &call, nil
}

func deriveBrowserSupplement(ctx context.Context, m Manifest, files map[string][]byte, table binding.ShapeTable, permissions map[string]BrowserPermission, reviews []Artifact) (*BrowserSupplementV1, error) {
	sourceCount := 0
	for _, s := range m.Sources {
		if s.Kind == "browser-profile" {
			sourceCount++
		}
	}
	if sourceCount == 0 {
		if len(permissions)+len(reviews) != 0 {
			return nil, ErrPackage
		}
		return nil, nil
	}
	doc, _, err := DecodeWorkflow(ctx, files[WorkflowPath])
	if err != nil || len(doc.Workflows) != 1 {
		return nil, ErrPackage
	}
	resolver, err := binding.NewResolver(table)
	if err != nil {
		return nil, ErrPackage
	}
	supplement := &BrowserSupplementV1{Version: BrowserSupplementVersion, WorkflowSHA256: m.Workflow.SHA256, Sources: []browsercontract.BrowserSourceRef{}, Calls: []BrowserCallV1{}, ReviewArtifacts: []browsercontract.ArtifactRef{}}
	for _, s := range m.Sources {
		if s.Kind != "browser-profile" {
			continue
		}
		version := ""
		for _, op := range table.Operations {
			if op.Source.ID == s.ID && op.Browser != nil {
				version = op.Browser.ProfileVersion
				break
			}
		}
		if version == "" {
			return nil, ErrPackage
		}
		supplement.Sources = append(supplement.Sources, browsercontract.BrowserSourceRef{ID: s.ID, Artifact: browsercontract.ArtifactRef{Path: s.Artifact.Path, SHA256: s.Artifact.SHA256}, ProfileVersion: version})
	}
	sort.Slice(supplement.Sources, func(i, j int) bool { return supplement.Sources[i].ID < supplement.Sources[j].ID })
	for _, review := range reviews {
		supplement.ReviewArtifacts = append(supplement.ReviewArtifacts, browsercontract.ArtifactRef{Path: review.Path, SHA256: review.SHA256})
	}
	operations := map[string]*uws1.Operation{}
	for _, op := range doc.Operations {
		if op == nil || operations[op.OperationID] != nil {
			return nil, ErrPackage
		}
		operations[op.OperationID] = op
	}
	sessions := map[string]string{}
	used := map[string]bool{}
	seenPermissions := map[string]bool{}
	for _, step := range doc.Workflows[0].Steps {
		if !flatStep(step) || used[step.OperationRef] {
			return nil, ErrPackage
		}
		op := operations[step.OperationRef]
		if !flatOperation(op) {
			return nil, ErrPackage
		}
		used[step.OperationRef] = true
		call, err := browserCall(ctx, step, op, m.Sources, resolver, sessions)
		if err != nil {
			return nil, err
		}
		if call == nil {
			continue
		}
		if permission, ok := permissions[step.StepID]; ok {
			seenPermissions[step.StepID] = true
			call.ReuseAllowed, call.SaveAllowed = permission.ReuseAllowed, permission.SaveAllowed
		}
		if (call.ReuseAllowed || call.SaveAllowed) && (call.SessionName == "" || call.Kind == "registration") || call.SaveAllowed && call.Kind != "authentication" {
			return nil, ErrPackage
		}
		supplement.Calls = append(supplement.Calls, *call)
	}
	if len(seenPermissions) != len(permissions) || len(supplement.Calls) == 0 || len(supplement.Calls) > 100 {
		return nil, ErrPackage
	}
	for _, op := range doc.Operations {
		if !used[op.OperationID] {
			return nil, ErrPackage
		}
	}
	return supplement, nil
}
func addBrowserSupplement(ctx context.Context, m *Manifest, files map[string][]byte, table binding.ShapeTable, options *BrowserBuildOptions, put func(string, []byte) error) error {
	var permissions map[string]BrowserPermission
	reviews := []Artifact{}
	if options != nil {
		permissions = options.Permissions
		for _, review := range options.Reviews {
			if !strings.HasPrefix(review.Path, "expected/browser-review/") || put(review.Path, review.Bytes) != nil {
				return ErrPackage
			}
			reviews = append(reviews, artifactFor(review.Path, files[review.Path]))
		}
		sort.Slice(reviews, func(i, j int) bool { return reviews[i].Path < reviews[j].Path })
	}
	s, err := deriveBrowserSupplement(ctx, *m, files, table, permissions, reviews)
	if err != nil {
		return err
	}
	if s == nil {
		if options != nil {
			return ErrPackage
		}
		return nil
	}
	encoded, err := json.Marshal(s)
	if err != nil {
		return ErrPackage
	}
	encoded, err = browsercontract.CanonicalJSON(encoded)
	if err != nil || len(encoded) > browsercontract.MaxBytes || put(BrowserPath, encoded) != nil {
		return ErrPackage
	}
	ref := artifactFor(BrowserPath, files[BrowserPath])
	m.Browser = &ref
	m.BrowserReviews = reviews
	return nil
}
func verifyBrowserSupplement(ctx context.Context, m Manifest, files map[string][]byte, table binding.ShapeTable) (*BrowserSupplementV1, error) {
	if m.Browser == nil {
		return nil, nil
	}
	var s BrowserSupplementV1
	if browsercontract.Decode(files[BrowserPath], &s) != nil {
		return nil, ErrPackage
	}
	permissions := map[string]BrowserPermission{}
	for _, c := range s.Calls {
		if permissions[c.StepID] != (BrowserPermission{}) {
			return nil, ErrPackage
		}
		permissions[c.StepID] = BrowserPermission{ReuseAllowed: c.ReuseAllowed, SaveAllowed: c.SaveAllowed}
	}
	expected, err := deriveBrowserSupplement(ctx, m, files, table, permissions, m.BrowserReviews)
	if err != nil || !reflect.DeepEqual(expected, &s) {
		return nil, ErrPackage
	}
	encoded, err := json.Marshal(expected)
	if err != nil {
		return nil, ErrPackage
	}
	canonical, err := browsercontract.CanonicalJSON(encoded)
	if err != nil || !bytes.Equal(canonical, files[BrowserPath]) {
		return nil, ErrPackage
	}
	return &s, nil
}

// BrowserSupplement returns an independent copy derived from a verified snapshot.
func (v VerifiedPackage) BrowserSupplement() (*BrowserSupplementV1, error) {
	if v.files == nil {
		return nil, ErrPackage
	}
	if v.manifest.Browser == nil {
		return nil, nil
	}
	var s BrowserSupplementV1
	if browsercontract.Decode(v.files[BrowserPath], &s) != nil {
		return nil, ErrPackage
	}
	return &s, nil
}

func mergeBrowserCredentials(names []string, files map[string][]byte) ([]string, error) {
	if files[BrowserPath] == nil {
		return names, nil
	}
	var s BrowserSupplementV1
	if browsercontract.Decode(files[BrowserPath], &s) != nil {
		return nil, ErrPackage
	}
	seen := map[string]bool{}
	for _, name := range names {
		seen[name] = true
	}
	for _, call := range s.Calls {
		for _, slot := range call.CredentialSlots {
			seen[slot.Name] = true
		}
	}
	if len(seen) > 64 {
		return nil, ErrPackage
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}
