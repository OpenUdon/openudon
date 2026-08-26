package packagepipeline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

const QualificationVersion = "openudon.package-qualification.v1"

var (
	qualificationMaterializedHook func(string)
	qualificationBeforeDryRunHook func() error
	qualificationCleanupHook      func(string) error
)

type QualificationCode string

const (
	QualificationInvalidPreparation QualificationCode = "invalid_preparation"
	QualificationScratchFailed      QualificationCode = "scratch_failed"
	QualificationMaterializeFailed  QualificationCode = "materialization_failed"
	QualificationStructureFailed    QualificationCode = "structure_failed"
	QualificationQualityFailed      QualificationCode = "quality_failed"
	QualificationPackageFailed      QualificationCode = "package_failed"
	QualificationDryRunFailed       QualificationCode = "dry_run_failed"
	QualificationCleanupFailed      QualificationCode = "cleanup_failed"
	QualificationCanceled           QualificationCode = "canceled"
)

// QualificationError is the closed pre-promotion failure surface.
type QualificationError struct {
	Code QualificationCode
	err  error
}

func (err *QualificationError) Error() string {
	if err == nil {
		return "package qualification failed"
	}
	return fmt.Sprintf("package qualification failed: %s", err.Code)
}

func (err *QualificationError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.err
}

// QualificationFailureCode returns the stable code for a qualification
// failure without exposing scratch paths or subprocess output.
func QualificationFailureCode(err error) (QualificationCode, bool) {
	var typed *QualificationError
	if !errors.As(err, &typed) {
		return "", false
	}
	return typed.Code, true
}

// QualifyOptions selects a canonical existing parent for same-filesystem
// restrictive scratch work. Now binds ephemeral approval/evidence timestamps.
type QualifyOptions struct {
	ScratchParent string
	Now           time.Time
}

type QualificationGate struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// QualificationReport is value-free evidence over one exact prepared
// generation. QualificationSHA256 hashes this structure with that field empty.
type QualificationReport struct {
	Version             string              `json:"version"`
	PreparationSHA256   string              `json:"preparation_sha256"`
	InputSHA256         string              `json:"input_sha256"`
	PackageSHA256       string              `json:"package_sha256"`
	QualitySHA256       string              `json:"quality_sha256"`
	ScratchPosture      string              `json:"scratch_posture"`
	NetworkPosture      string              `json:"network_posture"`
	DryRunPosture       string              `json:"dry_run_posture"`
	Gates               []QualificationGate `json:"gates"`
	QualificationSHA256 string              `json:"qualification_sha256"`
}

type Qualified struct {
	prepared Prepared
	report   QualificationReport
}

func (qualified Qualified) Prepared() Prepared { return clonePrepared(qualified.prepared) }

func (qualified Qualified) Report() QualificationReport {
	report := qualified.report
	report.Gates = append([]QualificationGate(nil), report.Gates...)
	return report
}

// Qualify materializes a prepared byte set only beneath a fresh mode-0700
// same-filesystem scratch root, performs every pre-promotion gate there, and
// removes the scratch tree before returning.
func Qualify(ctx context.Context, prepared Prepared, opts QualifyOptions) (Qualified, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Qualified{}, qualificationError(QualificationCanceled, err)
	}
	if err := validatePrepared(prepared); err != nil {
		return Qualified{}, qualificationError(QualificationInvalidPreparation, err)
	}
	parent, err := canonicalScratchParent(opts.ScratchParent)
	if err != nil {
		return Qualified{}, qualificationError(QualificationScratchFailed, err)
	}
	scratch, err := os.MkdirTemp(parent, ".openudon-package-qualification-")
	if err != nil {
		return Qualified{}, qualificationError(QualificationScratchFailed, errors.New("create restrictive scratch root"))
	}
	if err := os.Chmod(scratch, 0o700); err != nil {
		_ = removeScratch(parent, scratch)
		return Qualified{}, qualificationError(QualificationScratchFailed, errors.New("restrict scratch root"))
	}
	if same, ok := sameFilesystem(parent, scratch); !ok || !same {
		_ = removeScratch(parent, scratch)
		return Qualified{}, qualificationError(QualificationScratchFailed, errors.New("scratch root is not verifiably on the selected filesystem"))
	}
	qualified, qualificationErr := qualifyInScratch(ctx, prepared, scratch, opts.Now)
	cleanupErr := removeScratch(parent, scratch)
	if qualificationErr != nil {
		if cleanupErr != nil {
			return Qualified{}, qualificationError(QualificationCleanupFailed, errors.Join(qualificationErr, cleanupErr))
		}
		return Qualified{}, qualificationErr
	}
	if cleanupErr != nil {
		return Qualified{}, qualificationError(QualificationCleanupFailed, cleanupErr)
	}
	return qualified, nil
}

func qualifyInScratch(ctx context.Context, prepared Prepared, scratch string, at time.Time) (Qualified, error) {
	manifest := prepared.Manifest()
	packageRoot := filepath.Join(scratch, filepath.FromSlash(manifest.Scope))
	if err := materializePrepared(scratch, manifest.Scope, prepared.Files()); err != nil {
		return Qualified{}, qualificationError(QualificationMaterializeFailed, err)
	}
	if qualificationMaterializedHook != nil {
		qualificationMaterializedHook(packageRoot)
	}
	if err := validateScratchPackage(packageRoot, prepared.Manifest()); err != nil {
		return Qualified{}, qualificationError(QualificationStructureFailed, err)
	}
	readback, err := PrepareCurrent(ctx, PrepareOptions{ExampleDir: packageRoot, Scope: manifest.Scope, ExpectedInputSHA256: manifest.InputSHA256})
	if err != nil || !reflect.DeepEqual(readback.Manifest(), manifest) {
		if err == nil {
			err = errors.New("scratch preparation manifest differs from the approved byte set")
		}
		return Qualified{}, qualificationError(QualificationStructureFailed, err)
	}
	quality, err := synthesize.AssessCurrent(ctx, synthesize.Options{ExampleDir: packageRoot})
	if err != nil || quality == nil || !quality.Passed() {
		if err == nil {
			if quality == nil {
				err = errors.New("quality assessment returned no report")
			} else {
				err = fmt.Errorf("quality status is %q", quality.Status)
			}
		}
		return Qualified{}, qualificationError(QualificationQualityFailed, err)
	}
	assess := func(context.Context, synthesize.Options) (*synthesize.QualityReport, error) {
		copy := *quality
		copy.Checks = append([]synthesize.QualityCheck(nil), quality.Checks...)
		return &copy, nil
	}
	inspection, err := trustedrunner.InspectPackage(ctx, trustedrunner.TemplateOptions{
		RepoRoot: scratch, ExampleDir: packageRoot, Assess: assess,
	})
	if err != nil || inspection.PackageSHA256 != manifest.PackageSHA256 || inspection.HandoffSHA256 != manifest.HandoffSHA256 {
		if err == nil {
			err = errors.New("trusted package inspection digest differs from preparation")
		}
		return Qualified{}, qualificationError(QualificationPackageFailed, err)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	approval, err := trustedrunner.ApprovalTemplate(ctx, trustedrunner.TemplateOptions{
		RepoRoot: scratch, ExampleDir: packageRoot, State: trustedrunner.StateApprovedForSandbox,
		Reviewer: "OpenUdon package qualification", Now: func() time.Time { return at }, Assess: assess,
	})
	if err != nil {
		return Qualified{}, qualificationError(QualificationPackageFailed, err)
	}
	var approvalBytes bytes.Buffer
	if err := trustedrunner.WriteApproval(&approvalBytes, approval); err != nil {
		return Qualified{}, qualificationError(QualificationPackageFailed, err)
	}
	qualificationDir := filepath.Join(scratch, ".openudon-qualification-evidence")
	if err := os.Mkdir(qualificationDir, 0o700); err != nil {
		return Qualified{}, qualificationError(QualificationDryRunFailed, errors.New("create qualification evidence directory"))
	}
	approvalPath := filepath.Join(qualificationDir, "approval.json")
	if err := os.WriteFile(approvalPath, approvalBytes.Bytes(), 0o600); err != nil {
		return Qualified{}, qualificationError(QualificationDryRunFailed, errors.New("write ephemeral qualification approval"))
	}
	if qualificationBeforeDryRunHook != nil {
		if err := qualificationBeforeDryRunHook(); err != nil {
			return Qualified{}, qualificationError(QualificationDryRunFailed, err)
		}
	}
	run, err := trustedrunner.Run(ctx, trustedrunner.Options{
		RepoRoot: scratch, ExampleDir: packageRoot, Tier: trustedrunner.TierSandbox,
		ApprovalPath: approvalPath, WorkDir: filepath.Join(qualificationDir, "run"), DryRun: true,
		Now: func() time.Time { return at }, Env: []string{}, Assess: assess,
	})
	if err != nil || run == nil || !run.DryRun || run.PackageSHA256 != manifest.PackageSHA256 {
		if err == nil {
			err = errors.New("trusted dry-run result is incomplete or changed generation")
		}
		return Qualified{}, qualificationError(QualificationDryRunFailed, err)
	}
	if _, err := trustedrunner.VerifyRunEvidenceFile(run.RunEvidencePath); err != nil {
		return Qualified{}, qualificationError(QualificationDryRunFailed, err)
	}
	report := QualificationReport{
		Version: QualificationVersion, PreparationSHA256: manifest.ManifestSHA256, InputSHA256: manifest.InputSHA256,
		PackageSHA256: manifest.PackageSHA256, QualitySHA256: manifest.QualitySHA256,
		ScratchPosture: "same-filesystem mode-0700 root; package directories mode-0700 and files mode-0600 single-link",
		NetworkPosture: "offline package validation only", DryRunPosture: "trusted dry-run; executor not invoked",
		Gates: []QualificationGate{
			{Name: "prepared_readback", Status: "pass"}, {Name: "scratch_structure", Status: "pass"},
			{Name: "quality_and_secret_scan", Status: "pass"}, {Name: "package_and_handoff", Status: "pass"},
			{Name: "trusted_dry_run", Status: "pass"},
		},
	}
	data, err := json.Marshal(report)
	if err != nil {
		return Qualified{}, qualificationError(QualificationPackageFailed, err)
	}
	digest := sha256.Sum256(data)
	report.QualificationSHA256 = "sha256:" + hex.EncodeToString(digest[:])
	return Qualified{prepared: clonePrepared(prepared), report: report}, nil
}

func materializePrepared(scratch, scope string, files map[string][]byte) error {
	if firstPathComponent(scope) == ".openudon-qualification-evidence" {
		return errors.New("package scope collides with qualification evidence")
	}
	root, err := os.OpenRoot(scratch)
	if err != nil {
		return errors.New("open scratch root")
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.FromSlash(scope), 0o700); err != nil {
		return errors.New("create package scope")
	}
	caseAliases := map[string]string{}
	for _, path := range sortedFilePaths(files) {
		clean, err := packageartifacts.CleanRelativePath(path)
		if err != nil || clean != filepath.ToSlash(path) {
			return fmt.Errorf("prepared path %q is not canonical", path)
		}
		alias := strings.ToLower(clean)
		if prior := caseAliases[alias]; prior != "" && prior != clean {
			return fmt.Errorf("prepared paths %q and %q alias on case-insensitive filesystems", prior, clean)
		}
		caseAliases[alias] = clean
		joined := filepath.FromSlash(filepath.ToSlash(filepath.Join(scope, clean)))
		if err := root.MkdirAll(filepath.Dir(joined), 0o700); err != nil {
			return fmt.Errorf("create parent for %s", clean)
		}
		file, err := root.OpenFile(joined, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("create %s", clean)
		}
		data := files[path]
		written, writeErr := file.Write(data)
		syncErr, closeErr := file.Sync(), file.Close()
		if writeErr != nil || syncErr != nil || closeErr != nil || written != len(data) {
			return fmt.Errorf("write complete prepared file %s", clean)
		}
	}
	return nil
}

func validateScratchPackage(root string, manifest Manifest) error {
	expected := map[string]FileIdentity{}
	expectedDirectories := map[string]bool{".": true}
	for _, file := range manifest.Files {
		expected[file.Path] = file
		for directory := filepath.ToSlash(filepath.Dir(file.Path)); directory != "."; directory = filepath.ToSlash(filepath.Dir(directory)) {
			expectedDirectories[directory] = true
		}
	}
	found := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if info.Mode().Perm() != 0o700 {
				return fmt.Errorf("scratch directory mode is %04o, not 0700", info.Mode().Perm())
			}
			relative, err := filepath.Rel(root, path)
			if err != nil || !expectedDirectories[filepath.ToSlash(relative)] {
				return errors.New("scratch package contains an unsupported directory")
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
			return errors.New("scratch package contains an unsafe or unsupported member")
		}
		if links, ok := fileLinkCount(info); !ok || links != 1 {
			return errors.New("scratch package file link count is unsafe or unavailable")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		identity, ok := expected[relative]
		if !ok || found[relative] {
			return fmt.Errorf("scratch package contains unsupported or duplicate member %s", relative)
		}
		data, err := os.ReadFile(path)
		if err != nil || int64(len(data)) != identity.Bytes || evidenceSHA256(data) != identity.SHA256 {
			return fmt.Errorf("scratch package member %s differs from preparation", relative)
		}
		found[relative] = true
		return nil
	})
	if err != nil {
		return err
	}
	if len(found) != len(expected) {
		return errors.New("scratch package inventory is incomplete")
	}
	return nil
}

func validatePrepared(prepared Prepared) error {
	manifest := prepared.Manifest()
	if manifest.Version != PreparationVersion || manifest.Scope == "" || len(prepared.files) == 0 || len(manifest.Files) != len(prepared.files) {
		return errors.New("prepared package is empty or has an unsupported manifest")
	}
	rebuilt, err := buildManifest(context.Background(), manifest.Scope, prepared.files, prepared.handoff, prepared.quality)
	if err != nil || !reflect.DeepEqual(rebuilt, manifest) {
		return errors.New("prepared package bytes or review facts do not match the manifest")
	}
	return nil
}

func canonicalScratchParent(value string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(value))
	if clean == "." || !filepath.IsAbs(clean) {
		return "", errors.New("scratch parent must be an absolute canonical directory")
	}
	resolved, err := filepath.EvalSymlinks(clean)
	info, statErr := os.Lstat(clean)
	if err != nil || statErr != nil || resolved != clean || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("scratch parent must be an existing canonical non-symlink directory")
	}
	return clean, nil
}

func removeScratch(parent, scratch string) error {
	relative, err := filepath.Rel(parent, scratch)
	if err != nil || filepath.Dir(relative) != "." || !strings.HasPrefix(filepath.Base(relative), ".openudon-package-qualification-") {
		return errors.New("refuse unsafe scratch cleanup target")
	}
	if qualificationCleanupHook != nil {
		if err := qualificationCleanupHook(scratch); err != nil {
			return err
		}
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return errors.New("open scratch parent for cleanup")
	}
	defer root.Close()
	if err := root.RemoveAll(filepath.Base(relative)); err != nil {
		return errors.New("remove qualification scratch tree")
	}
	return nil
}

func fileLinkCount(info os.FileInfo) (uint64, bool) {
	if info == nil || info.Sys() == nil {
		return 0, false
	}
	value := reflect.Indirect(reflect.ValueOf(info.Sys()))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return 0, false
	}
	for _, name := range []string{"Nlink", "NumberOfLinks"} {
		field := value.FieldByName(name)
		if !field.IsValid() {
			continue
		}
		switch field.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return field.Uint(), true
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() >= 0 {
				return uint64(field.Int()), true
			}
		}
	}
	return 0, false
}

func sameFilesystem(first, second string) (bool, bool) {
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	if firstErr != nil || secondErr != nil {
		return false, false
	}
	firstDevice, firstOK := fileDevice(firstInfo)
	secondDevice, secondOK := fileDevice(secondInfo)
	return firstOK && secondOK && firstDevice == secondDevice, firstOK && secondOK
}

func fileDevice(info os.FileInfo) (uint64, bool) {
	if info == nil || info.Sys() == nil {
		return 0, false
	}
	value := reflect.Indirect(reflect.ValueOf(info.Sys()))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return 0, false
	}
	for _, name := range []string{"Dev", "VolumeSerialNumber"} {
		field := value.FieldByName(name)
		if !field.IsValid() {
			continue
		}
		switch field.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return field.Uint(), true
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() >= 0 {
				return uint64(field.Int()), true
			}
		}
	}
	return 0, false
}

func firstPathComponent(path string) string {
	path = strings.Trim(filepath.ToSlash(path), "/")
	if before, _, ok := strings.Cut(path, "/"); ok {
		return before
	}
	return path
}

func evidenceSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func clonePrepared(prepared Prepared) Prepared {
	return Prepared{
		root: prepared.root, manifest: cloneManifest(prepared.manifest), files: cloneFiles(prepared.files),
		quality: prepared.Quality(), handoff: prepared.Handoff(),
	}
}

func qualificationError(code QualificationCode, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = QualificationCanceled
	}
	return &QualificationError{Code: code, err: err}
}
