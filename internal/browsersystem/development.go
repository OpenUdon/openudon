package browsersystem

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/capture"
	"github.com/OpenUdon/openudon/internal/browsercheck"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const DevelopmentVersion = "openudon.browser-development.v1"

type DevelopmentOptions struct {
	Root, UdonRepo, Out, Mode, Stage, Cache string
	Reuse                                   bool
	Progress                                io.Writer
}
type DevelopmentReport struct {
	Version          string    `json:"version"`
	QualifiesRuntime bool      `json:"qualifies_runtime"`
	Mode             string    `json:"mode"`
	InputSHA256      string    `json:"input_sha256"`
	ExecutionID      string    `json:"execution_id"`
	ExecutedAt       time.Time `json:"executed_at"`
	DurationMS       int64     `json:"duration_ms"`
	Reused           bool      `json:"reused"`
	Stage            Stage     `json:"stage"`
}

// Development reports cannot be used by the qualification verifier. Reuse is
// explicit, bounded to 24 hours, and retains the original execution identity.
func validDevelopment(r DevelopmentReport, input, stage string, now time.Time) bool {
	return r.Version == DevelopmentVersion && !r.QualifiesRuntime && r.Mode == "smoke" &&
		r.InputSHA256 == input && evidencefile.ValidSHA256(input) && len(r.ExecutionID) == 32 && validID(r.ExecutionID) &&
		!r.ExecutedAt.IsZero() && !r.ExecutedAt.After(now) && now.Sub(r.ExecutedAt) <= 24*time.Hour &&
		r.DurationMS >= 0 && r.Stage.ID == stage && r.Stage.Status == "pass" && r.Stage.SHA256 == evidenceHash(r.Stage.Evidence) && validateProof(r.Stage, "loopback") == nil
}
func cacheableDevelopmentStage(id string) bool {
	return id == "registration_ui_handoff" || id == "bap_bcp_transaction"
}
func validID(id string) bool { _, err := hex.DecodeString(id); return err == nil }
func developmentStage(mode, stage string) (string, error) {
	if mode == "fast" {
		if stage != "" && stage != "openudon_unit" {
			return "", errors.New("development_stage")
		}
		return "openudon_unit", nil
	}
	if mode != "smoke" {
		return "", errors.New("development_mode")
	}
	if stage == "" {
		return "registration_ui_handoff", nil
	}
	for _, id := range inventory("loopback") {
		if id == stage {
			return id, nil
		}
	}
	return "", errors.New("development_stage")
}
func outsideDevelopmentSources(path, root, udon string) bool {
	for _, base := range []string{filepath.Dir(root), filepath.Dir(udon)} {
		rel, err := filepath.Rel(base, path)
		if err != nil || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return false
		}
	}
	return true
}

func RunDevelopment(ctx context.Context, o DevelopmentOptions) (report *DevelopmentReport, resultErr error) {
	id, err := developmentStage(o.Mode, o.Stage)
	if err != nil {
		return nil, err
	}
	if o.Reuse && (o.Mode != "smoke" || o.Cache == "" || !cacheableDevelopmentStage(id)) {
		return nil, errors.New("reuse_requires_transaction_smoke_cache")
	}
	root, err := filepath.Abs(o.Root)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return nil, errors.New("source_state")
	}
	udon := o.UdonRepo
	if udon == "" {
		udon = filepath.Join(filepath.Dir(root), "udon")
	}
	if o.Mode == "smoke" {
		udon, err = filepath.Abs(udon)
		if err == nil {
			udon, err = filepath.EvalSymlinks(udon)
		}
		if err != nil {
			return nil, errors.New("source_state")
		}
	} else {
		// Browser-free checks work without a private executor checkout.
		udon = root
	}
	if o.Out == "" {
		return nil, errors.New("output")
	}
	out, err := filepath.Abs(o.Out)
	if err != nil {
		return nil, errors.New("output")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(out))
	if err != nil {
		return nil, errors.New("output")
	}
	out = filepath.Join(parent, filepath.Base(out))
	if !outsideDevelopmentSources(out, root, udon) {
		return nil, errors.New("report_must_be_outside_workspace")
	}
	if o.Cache != "" && (!filepath.IsAbs(o.Cache) || !outsideDevelopmentSources(o.Cache, root, udon)) {
		return nil, errors.New("cache_must_be_outside_workspace")
	}
	var cache *browsercheck.Cache
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, errors.New("output")
	}
	defer func() {
		if report != nil {
			data, e := json.MarshalIndent(report, "", "  ")
			if e == nil {
				_, e = f.Write(append(data, '\n'))
			}
			resultErr = errors.Join(resultErr, e)
		}
		resultErr = errors.Join(resultErr, f.Sync(), f.Close())
		// Publish a reusable success only after report and timing writes complete.
		if resultErr == nil && report != nil && !report.Reused && cache != nil {
			data, e := json.Marshal(report)
			if e == nil {
				e = cache.WriteResult(id, data)
			}
			resultErr = errors.Join(resultErr, e)
		}
	}()
	ctx, trace, err := browsercheck.TraceFile(ctx, out+".timing.jsonl")
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, trace.Close()) }()
	ctx, cancel := context.WithTimeout(browsercheck.Stage(ctx, id), 20*time.Minute)
	defer cancel()
	// Fast uses Go's dependency-aware result cache. Browser development also binds
	// all source and installed runtime bytes before considering explicit reuse.
	input := ""
	if o.Mode == "smoke" {
		input, err = developmentInput(ctx, root, udon)
		if err != nil {
			return nil, err
		}
	}
	if o.Mode == "smoke" && o.Cache != "" && cacheableDevelopmentStage(id) {
		cache, err = browsercheck.OpenCache(o.Cache, input)
		if err != nil {
			return nil, err
		}
		ctx = browsercheck.WithCache(ctx, cache)
	}
	if o.Reuse {
		data, e := cache.ReadResult(id)
		if e == nil {
			var previous DevelopmentReport
			if evidencefile.DecodeStrict(data, &previous) != nil || !validDevelopment(previous, input, id, time.Now()) {
				return nil, errors.New("cached_development_invalid")
			}
			previous.Reused = true
			if o.Progress != nil {
				fmt.Fprintf(o.Progress, "browser-development: %s reused execution %s from %s\n", id, previous.ExecutionID, previous.ExecutedAt.Format(time.RFC3339))
			}
			return &previous, nil
		}
		if !os.IsNotExist(e) {
			return nil, errors.New("cached_development_invalid")
		}
	}
	started := time.Now().UTC()
	var token [16]byte
	if _, err = rand.Read(token[:]); err != nil {
		return nil, err
	}
	report = &DevelopmentReport{Version: DevelopmentVersion, Mode: o.Mode, InputSHA256: input, ExecutionID: hex.EncodeToString(token[:]), ExecutedAt: started}
	if o.Progress != nil {
		fmt.Fprintf(o.Progress, "browser-development: %s fresh started\n", id)
	}
	finish := browsercheck.Span(ctx, "stage")
	var value any
	switch {
	case o.Mode == "fast":
		value, err = goTestsMode(ctx, root, []string{"./..."}, nil, false, true)
	case id == "registration_ui_handoff" || id == "bap_bcp_transaction":
		value, err = RunComponent(ctx, root, udon, id)
	default:
		value, err = runStage(ctx, root, udon, id)
	}
	finish(err)
	report.DurationMS = time.Since(started).Milliseconds()
	if err == nil && o.Mode == "smoke" {
		after, e := developmentInput(ctx, root, udon)
		if e != nil || after != input {
			err = errors.New("development_input_changed")
		}
	}
	if err != nil {
		report.Stage = Stage{ID: id, Status: "fail"}
		retainFailureDiagnostic(out, id, err, o.Progress)
		return report, errors.New("development_stage_failed")
	}
	report.Stage = proof(id, value)
	if o.Mode == "smoke" && !validDevelopment(*report, input, id, time.Now()) {
		return report, errors.New("development_evidence")
	}
	if o.Progress != nil {
		fmt.Fprintf(o.Progress, "browser-development: %s fresh passed (%d ms)\n", id, report.DurationMS)
	}
	return report, nil
}

// Input includes dirty sources/fixtures, installed JS modules, Go dependency and
// toolchain files, the running checker, both Chromium distributions and sandbox.
// Environment values and local paths are hashed, never written to the report.
func developmentInput(ctx context.Context, root, udon string) (digest string, resultErr error) {
	finish := browsercheck.Span(ctx, "development_inputs")
	defer func() { finish(resultErr) }()
	ss, err := sources(ctx, root, udon, true)
	if err != nil {
		return "", err
	}
	values := map[string]string{}
	data, _ := json.Marshal(ss)
	values["sources"] = hash(data)
	values["roots"] = hash([]byte(root + "\x00" + udon))
	env := os.Environ()
	sort.Strings(env)
	values["environment"] = hash([]byte(strings.Join(env, "\x00")))
	addFile := func(path string) error {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("development_runtime")
		}
		values[path] = hash(append([]byte(fmt.Sprint(info.Mode())), b...))
		return nil
	}
	for _, name := range []string{"go", "node", "npm"} {
		path, err := exec.LookPath(name)
		if err != nil {
			return "", errors.New("development_runtime")
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil || addFile(path) != nil {
			return "", errors.New("development_runtime")
		}
	}
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(self)
	if err != nil {
		return "", err
	}
	values["checker"] = hash(b)
	if sandbox := os.Getenv("CHROME_DEVEL_SANDBOX"); sandbox != "" {
		if addFile(sandbox) != nil {
			return "", errors.New("development_sandbox")
		}
	}
	// Browsertools/Go UI also use a separately installed Playwright driver.
	installed, e := capture.PreflightPlaywrightDriver("")
	if e != nil {
		return "", errors.New("development_go_driver")
	}
	for _, dir := range []string{installed.DriverDirectory, filepath.Dir(installed.CLIPath)} {
		real, e := filepath.EvalSymlinks(dir)
		if e != nil {
			return "", errors.New("development_go_driver")
		}
		d, e := browsercheck.TreeDigest(real, true)
		if e != nil {
			return "", errors.New("development_go_driver")
		}
		values[real] = d
	}
	if addFile(installed.NodeExecutable) != nil {
		return "", errors.New("development_go_driver")
	}
	// Child environments can use HOME defaults instead of parent overrides.
	standard := filepath.Join(os.Getenv("HOME"), ".cache", "ms-playwright-go", installed.Version)
	if _, e := os.Stat(standard); e == nil {
		d, e := browsercheck.TreeDigest(standard, true)
		if e != nil {
			return "", errors.New("development_go_driver")
		}
		values[standard] = d
	} else if os.IsNotExist(e) {
		values[standard] = "missing"
	} else {
		return "", errors.New("development_go_driver")
	}
	for _, path := range []string{"/etc/os-release", "/var/lib/dpkg/status", "/proc/sys/kernel/osrelease"} {
		if _, e := os.Stat(path); e == nil {
			if addFile(path) != nil {
				return "", errors.New("development_host")
			}
		} else if os.IsNotExist(e) {
			values[path] = "missing"
		} else {
			return "", errors.New("development_host")
		}
	}
	driver := filepath.Join(filepath.Dir(root), "browserdriver")
	modules, err := filepath.EvalSymlinks(filepath.Join(driver, "node_modules"))
	if err != nil {
		return "", err
	}
	npmPath, err := exec.LookPath("npm")
	if err != nil {
		return "", errors.New("development_runtime")
	}
	npmPath, err = filepath.EvalSymlinks(npmPath)
	if err != nil {
		return "", errors.New("development_runtime")
	}
	values["npm_modules"], err = browsercheck.TreeDigest(filepath.Dir(filepath.Dir(npmPath)), true)
	if err != nil {
		return "", errors.New("development_modules")
	}
	values["node_modules"], err = browsercheck.TreeDigest(modules, true)
	if err != nil {
		return "", errors.New("development_modules")
	}
	// Resolve paths without launching a browser. Include the installed cache root
	// because Playwright may select Chromium's full or headless distribution.
	browser, err := command(ctx, driver, []string{"node", "--input-type=module", "--eval", `import {chromium} from 'playwright'; console.log(chromium.executablePath())`}, nil)
	if err != nil {
		return "", err
	}
	browserPath := strings.TrimSpace(string(browser))
	if !filepath.IsAbs(browserPath) {
		return "", errors.New("development_browser")
	}
	cacheRoot := filepath.Dir(filepath.Dir(filepath.Dir(browserPath)))
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return "", err
	}
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "chromium") || strings.HasPrefix(e.Name(), "ffmpeg") {
			d, err := browsercheck.TreeDigest(filepath.Join(cacheRoot, e.Name()), true)
			if err != nil {
				return "", errors.New("development_browser")
			}
			values["browser_"+e.Name()] = d
			count++
		}
	}
	if count == 0 {
		return "", errors.New("development_browser")
	}
	goRoot, err := command(ctx, root, []string{"go", "env", "GOROOT"}, nil)
	if err != nil {
		return "", err
	}
	for _, part := range []string{"pkg/tool", "pkg/include", "lib"} {
		path := filepath.Join(strings.TrimSpace(string(goRoot)), part)
		d, err := browsercheck.TreeDigest(path, true)
		if err != nil {
			return "", errors.New("development_go_toolchain")
		}
		values[path] = d
	}
	seen := map[string]bool{}
	for _, repo := range []string{root, udon, filepath.Join(filepath.Dir(root), "browsertools")} {
		data, err := command(ctx, repo, []string{"go", "list", "-deps", "-test", "-json", "./..."}, nil)
		if err != nil {
			return "", errors.New("development_go_dependencies")
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		for {
			var pkg struct {
				Dir                                                                        string
				GoFiles, CgoFiles, CFiles, CXXFiles, HFiles, SFiles, SysoFiles, EmbedFiles []string
			}
			err = decoder.Decode(&pkg)
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			for _, files := range [][]string{pkg.GoFiles, pkg.CgoFiles, pkg.CFiles, pkg.CXXFiles, pkg.HFiles, pkg.SFiles, pkg.SysoFiles, pkg.EmbedFiles} {
				for _, name := range files {
					path := name
					if !filepath.IsAbs(path) {
						path = filepath.Join(pkg.Dir, name)
					}
					if !seen[path] {
						if addFile(path) != nil {
							return "", errors.New("development_go_dependencies")
						}
						seen[path] = true
					}
				}
			}
		}
	}
	after, err := sources(ctx, root, udon, true)
	if err != nil {
		return "", err
	}
	beforeBytes, _ := json.Marshal(ss)
	afterBytes, _ := json.Marshal(after)
	if !bytes.Equal(beforeBytes, afterBytes) {
		return "", errors.New("development_input_changed")
	}
	data, err = json.Marshal(values)
	if err != nil {
		return "", err
	}
	return hash(data), nil
}
