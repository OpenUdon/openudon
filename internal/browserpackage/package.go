package browserpackage

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/artifactwriter"
	engine "github.com/OpenUdon/openudon/internal/authoringengine"
	"github.com/OpenUdon/openudon/internal/browserauthoring"
	"github.com/OpenUdon/openudon/internal/browsercapture"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/elicitor"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/synthesize"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type fileRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

type importedFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type receipt struct {
	Version     string                         `json:"version"`
	StartSHA256 string                         `json:"start_sha256"`
	Transaction browsertransaction.Transaction `json:"transaction"`
	Effect      string                         `json:"effect"`
	Files       []importedFile                 `json:"files"`
}

var invalidEvidence = errors.New("browser author evidence invalid")

// InputDigest reads a bounded package inventory without following symlinks.
// This is a byte observation, not source semantics or approval authority.
func InputDigest(ctx context.Context, root string) (string, error) {
	return inputDigest(ctx, root, nil)
}

func inputDigest(ctx context.Context, root string, ignored map[string]bool) (string, error) {
	if ctx == nil {
		return "", invalidEvidence
	}
	var files []fileRecord
	total := 0
	err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return invalidEvidence
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if p == root {
			if !e.IsDir() {
				return invalidEvidence
			}
		}
		if ignored[p] {
			return nil
		}
		if e.Name() == ".git" {
			if e.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if e.Type()&os.ModeSymlink != 0 {
			return invalidEvidence
		}
		if e.IsDir() {
			info, err := e.Info()
			if err != nil {
				return invalidEvidence
			}
			if st, ok := info.Sys().(*syscall.Stat_t); !ok || st.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0022 != 0 {
				return invalidEvidence
			}
			return nil
		}
		if !e.Type().IsRegular() || len(files) >= 512 {
			return invalidEvidence
		}
		data, info, err := evidencefile.ReadRegular(p, 8<<20)
		if err != nil {
			return invalidEvidence
		}
		if st, ok := info.Sys().(*syscall.Stat_t); !ok || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 || info.Mode().Perm()&0022 != 0 {
			return invalidEvidence
		}
		total += len(data)
		if total > 32<<20 {
			return invalidEvidence
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return invalidEvidence
		}
		files = append(files, fileRecord{filepath.ToSlash(rel), Digest(data), len(data)})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	data, err := json.Marshal(files)
	if err != nil {
		return "", err
	}
	return Digest(data), nil
}

func resolveRoot(example string) (string, error) {
	if example == "" {
		return "", invalidEvidence
	}
	root, err := filepath.Abs(example)
	if err != nil {
		return "", invalidEvidence
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil || real != root {
		return "", invalidEvidence
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", invalidEvidence
	}
	wd, err = filepath.EvalSymlinks(wd)
	if err != nil {
		return "", invalidEvidence
	}
	rel, err := filepath.Rel(wd, root)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", invalidEvidence
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", invalidEvidence
	}
	if st, ok := info.Sys().(*syscall.Stat_t); !ok || st.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0022 != 0 {
		return "", invalidEvidence
	}
	return root, nil
}

func readImport(root string, r Request, at time.Time) (elicitor.VirtualBrowserTransactionInput, error) {
	read := func(rel string) ([]byte, error) {
		p, err := artifactwriter.SafeExampleTarget(root, rel)
		if err != nil {
			return nil, invalidEvidence
		}
		data, _, err := evidencefile.ReadRegular(p, 8<<20)
		if err != nil {
			return nil, invalidEvidence
		}
		return data, nil
	}
	data, err := read(r.ReceiptPath)
	if err != nil || evidencefile.SHA256(data) != r.ReceiptSHA256 {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	var rec receipt
	if evidencefile.DecodeStrict(data, &rec) != nil || rec.Version != "openudon.browser-capture-import.v1" || rec.Effect != "write" || rec.StartSHA256 != evidencefile.SHA256(r.Start) || rec.Transaction.State != browsertransaction.StateReviewed || rec.Transaction.Validate() != nil || r.ReceiptPath != "expected/browser-capture/"+rec.Transaction.ID+".json" {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	digest, err := browsertransaction.Digest(rec.Transaction)
	if err != nil || digest != r.TransactionSHA256 {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	start, err := browsercapture.DecodeStart(r.Start)
	if err != nil {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	var origins []string
	if start.Authentication != nil {
		origins = start.Authentication.Origins
	} else if start.Registration != nil {
		origins = start.Registration.Origins
	}
	origins = slices.Clone(origins)
	for i, origin := range origins {
		origins[i], err = profile.ParseOrigin(strings.TrimSpace(origin))
		if err != nil {
			return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
		}
	}
	slices.Sort(origins)
	transactionOrigins := slices.Clone(rec.Transaction.Provenance.Origins)
	slices.Sort(transactionOrigins)
	if !slices.Equal(origins, transactionOrigins) {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	paths := map[string][]byte{}
	for _, file := range rec.Files {
		if !safeRelative(file.Path) || !evidencefile.ValidSHA256(file.SHA256) {
			return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
		}
		if _, exists := paths[file.Path]; exists {
			return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
		}
		content, err := read(file.Path)
		if err != nil || evidencefile.SHA256(content) != file.SHA256 {
			return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
		}
		paths[file.Path] = content
	}
	if start.Mode == browsercapture.Authenticated {
		if rec.Transaction.Kind != browsertransaction.KindAuthenticationCapability || len(paths) != 3 {
			return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
		}
		a := start.Authentication
		cfg := browserauthoring.LiveConfig{ProfileID: a.ProfileID, URL: a.URL, DashboardURL: a.DashboardURL, GoalURL: a.GoalURL, Goal: a.Goal, Origins: a.Origins, GoalContext: a.GoalContext, GoalRole: a.GoalRole, GoalLabel: a.GoalLabel, AfterAuthentication: a.AfterAuthentication}
		candidate, err := browserauthoring.ImportedAuthenticationCandidate(cfg, rec.Transaction, paths["browser-authentication/"+rec.Transaction.ID+"-auth.json"], paths["browser-profiles/"+rec.Transaction.ID+".json"], paths[".icot/authenticated-browser-authoring.json"], at)
		if err != nil {
			return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
		}
		p, err := authprofile.Parse(candidate.Authentication())
		if err != nil {
			return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
		}
		if *r.ExpectedTOTP {
			found := false
			for _, step := range p.Flows[candidate.Flow()].Sequence {
				if step.Challenge != nil && step.Challenge.Kind == "totp" {
					found = true
				}
			}
			if !found {
				return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
			}
		}
		return engine.AuthenticationCapabilityVirtualBrowserTransaction(candidate, true)
	}
	if rec.Transaction.Kind != browsertransaction.KindRegistration || len(paths) != 2 || rec.Transaction.ID != start.Registration.TransactionID {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	p, err := registrationprofile.Parse(paths["browser-registration/"+rec.Transaction.ID+".json"])
	if err != nil {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	source, err := registrationprofile.MarshalJSON(p)
	if err != nil {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	flow := r.Flow
	if flow == "" {
		names := registrationprofile.SortedFlowNames(p)
		if len(names) != 1 {
			return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
		}
		flow = names[0]
	}
	review := paths["browser-registration/"+rec.Transaction.ID+".review.json"]
	// Native review digests use their typed field order, not generic-map order.
	// Preserve compact original order here; native discovery validates its schema.
	var raw json.RawMessage
	if json.Unmarshal(review, &raw) != nil {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	review, err = json.Marshal(raw)
	if err != nil {
		return elicitor.VirtualBrowserTransactionInput{}, invalidEvidence
	}
	return elicitor.VirtualBrowserTransactionInput{Transaction: rec.Transaction, Sources: []elicitor.VirtualBrowserSourceInput{{Kind: browsertransaction.CandidateRegistration, Flow: flow, CleanupDisposition: r.CleanupDisposition, Source: source, Review: review}}}, nil
}

type preparation struct {
	plan    Plan
	request Request
	root    string
	files   artifactwriter.Prepared
}

func prepare(ctx context.Context, example string, data []byte) (preparation, error) {
	if ctx == nil {
		return preparation{}, invalidEvidence
	}
	if err := ctx.Err(); err != nil {
		return preparation{}, err
	}
	r, err := DecodeRequest(data)
	if err != nil {
		return preparation{}, err
	}
	root, err := resolveRoot(example)
	if err != nil {
		return preparation{}, err
	}
	inputSHA, err := InputDigest(ctx, root)
	if err != nil || r.InputSHA256 != "" && inputSHA != r.InputSHA256 {
		return preparation{}, invalidEvidence
	}
	at := time.Now().UTC()
	input, err := readImport(root, r, at)
	if err != nil {
		return preparation{}, err
	}
	discovery, err := elicitor.DiscoverVirtualBrowserSources([]elicitor.VirtualBrowserTransactionInput{input}, at)
	if err != nil {
		return preparation{}, invalidEvidence
	}
	plan := Plan{Version: Version, Kind: "plan", RequestID: r.RequestID, RequestSHA256: Digest(data), InputSHA256: inputSHA, ReceiptSHA256: r.ReceiptSHA256, TransactionSHA256: r.TransactionSHA256, Candidates: discovery.Candidates, Operations: []Operation{}, Blockers: []string{}, WriteConflicts: []engine.WriteConflict{}, FileActions: []elicitor.FileAction{}}
	for _, c := range discovery.Candidates {
		for _, doc := range discovery.Docs {
			if doc.RelativePath != c.TargetPath {
				continue
			}
			for _, op := range doc.Operations {
				x := Operation{CandidateID: c.ID, Name: op.OperationID, Type: strings.ToLower(op.Method)}
				if op.RequestBody != nil {
					for _, field := range op.RequestBody.Fields {
						x.Inputs = append(x.Inputs, field.Path)
					}
				}
				plan.Operations = append(plan.Operations, x)
			}
		}
	}
	result := preparation{request: r, root: root}
	if r.Flow == "" || input.Transaction.Kind == browsertransaction.KindAuthenticationCapability && r.Action == "" {
		plan.Blockers = append(plan.Blockers, "operation_selection_required")
	} else {
		goal := "Review the captured account registration recipe without submitting it."
		start, _ := browsercapture.DecodeStart(r.Start)
		if start.Authentication != nil {
			goal = start.Authentication.Goal
		}
		intent := rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: r.WorkflowName, Description: goal}}
		for _, i := range r.Inputs {
			intent.Inputs = append(intent.Inputs, &rollout.Input{Name: i.Name, Type: i.Type, Sensitive: i.Sensitive, Required: true})
		}
		seed, err := elicitor.SessionFromIntent(&intent, projectwizard.Answers{Safety: "sandbox-only; exact approval required", Fallback: "stop cleanly without retry", SideEffectScope: "sandbox"})
		if err != nil {
			return preparation{}, invalidEvidence
		}
		seed, err = elicitor.ReviewedVirtualWorkflow(seed, discovery, r.Flow, r.Action, r.InputBindings)
		if err != nil {
			return preparation{}, invalidEvidence
		}
		issues := elicitor.CheckReadiness(seed, discovery.Docs)
		for _, issue := range issues {
			if issue.Severity == "blocking" {
				plan.Blockers = append(plan.Blockers, issue.Code)
			}
		}
		artifacts, err := elicitor.RenderArtifacts(seed)
		if err != nil {
			return preparation{}, invalidEvidence
		}
		prepared, err := artifactwriter.Prepare(root, artifacts, r.AllowOverwrite, at)
		if err != nil {
			return preparation{}, invalidEvidence
		}
		plan.WriteConflicts, err = artifactwriter.WriteConflicts(prepared)
		if err != nil {
			return preparation{}, invalidEvidence
		}
		plan.FileActions = artifactwriter.ProposedFileActions(prepared)
		plan.ArtifactSHA256 = map[string]string{}
		for _, file := range prepared.Files {
			rel, err := filepath.Rel(root, file.Path)
			if err != nil || !safeRelative(filepath.ToSlash(rel)) {
				return preparation{}, invalidEvidence
			}
			plan.ArtifactSHA256[filepath.ToSlash(rel)] = Digest([]byte(file.Content))
		}
		if len(plan.WriteConflicts) > 0 && !r.AllowOverwrite {
			plan.Blockers = append(plan.Blockers, "overwrite_authority_required")
		}
		plan.Preview = &engine.Preview{ProjectMD: artifacts.ProjectMD, IntentHCL: artifacts.IntentHCL, ProjectPath: filepath.Join(root, "project.md"), IntentPath: filepath.Join(root, rollout.IntentPath)}
		plan.Ready = len(plan.Blockers) == 0
		result.files = prepared
	}
	sort.Slice(plan.Operations, func(i, j int) bool {
		a, b := plan.Operations[i], plan.Operations[j]
		if a.CandidateID != b.CandidateID {
			return a.CandidateID < b.CandidateID
		}
		return a.Name < b.Name
	})
	plan, err = SealPlan(plan)
	if err != nil {
		return preparation{}, err
	}
	// Detect changes during independent receipt/source validation and rendering.
	observed, err := InputDigest(ctx, root)
	if err != nil || observed != inputSHA {
		return preparation{}, invalidEvidence
	}
	result.plan = plan
	return result, nil
}

func Prepare(ctx context.Context, example string, data []byte) (Plan, error) {
	p, err := prepare(ctx, example, data)
	return p.plan, err
}

func Apply(ctx context.Context, example string, data []byte, expectedPlan string, confirmed bool) (Result, error) {
	if !confirmed || !validDigest(expectedPlan) {
		return Result{}, errors.New("browser author exact confirmation required")
	}
	p, err := prepare(ctx, example, data)
	if err != nil {
		return Result{}, err
	}
	if p.request.InputSHA256 == "" || !p.plan.Ready || p.plan.PlanSHA256 != expectedPlan {
		return Result{}, errors.New("browser author plan changed or not ready")
	}
	guard := func(transient []string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ignored := map[string]bool{}
		for _, name := range transient {
			ignored[name] = true
		}
		sha, err := inputDigest(ctx, p.root, ignored)
		if err != nil || sha != p.plan.InputSHA256 {
			return invalidEvidence
		}
		_, err = readImport(p.root, p.request, time.Now().UTC())
		return err
	}
	written, err := artifactwriter.CommitCheckedObserved(p.files, p.request.AllowOverwrite, guard)
	if err != nil {
		return Result{}, errors.New("browser author commit failed; inspect before another proposal")
	}
	result := Result{Version: Version, Kind: "result", RequestID: p.request.RequestID, RequestSHA256: p.plan.RequestSHA256, PlanSHA256: p.plan.PlanSHA256, Outcome: "build_failed", Written: []string{}, QualityStatus: "fail", CleanupRequired: len(written.CleanupWarnings) != 0}
	for _, name := range written.Written {
		rel, err := filepath.Rel(p.root, name)
		if err != nil {
			return result, invalidEvidence
		}
		result.Written = append(result.Written, filepath.ToSlash(rel))
	}
	_, quality, buildErr := synthesize.PackageFromIntent(ctx, synthesize.Options{ExampleDir: p.root})
	if quality != nil {
		result.QualityStatus = quality.Status
	}
	if buildErr != nil || quality == nil || !quality.Passed() {
		return result, errors.New("browser author native build failed; inspect committed authoring")
	}
	result.Outcome = "authored"
	return result, nil
}
