package trustedrunner

import (
	"fmt"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/udonreport"
	"os"
	"path/filepath"
	"reflect"
)

func buildRunEvidenceV5Executor(opts runEvidenceOptions, argv []string) (RunEvidenceExecutor, *udonreport.ObservationV5, error) {
	e := RunEvidenceExecutor{Invoked: opts.Invoked, Mode: opts.Mode, RunnerPath: opts.RunnerPath, Argv: argv}
	if opts.Invoked && opts.Mode == "internal-runner" && !opts.Prepared.InvocationAttempted {
		return e, nil, fmt.Errorf("executor invocation was not attempted; no invoked v5 evidence")
	}
	if opts.Prepared.InventoryV5 == nil {
		return e, nil, fmt.Errorf("missing reviewed v5 inventory")
	}
	i := *opts.Prepared.InventoryV5
	state := "missing"
	if opts.Result.DryRun {
		state = "dry_run"
	}
	o := udonreport.UnknownV5(i, state)
	if state == "dry_run" {
		return e, &o, nil
	}
	data, info, err := evidencefile.ReadRegular(opts.Prepared.ExecutorReportPath, 256<<10)
	if err == nil {
		o = udonreport.ObserveV5(i, data)
		if o.State == "validated" {
			rel, err := filepath.Rel(opts.Result.WorkDir, opts.Prepared.ExecutorReportPath)
			if err != nil {
				return e, nil, err
			}
			clean, err := packageartifacts.CleanRelativePath(filepath.ToSlash(rel))
			if err != nil {
				return e, nil, err
			}
			e.ReportPath = clean
			e.ReportSHA256 = evidencefile.SHA256(data)
			e.ReportSize = info.Size()
		}
	} else if !os.IsNotExist(err) {
		o = udonreport.UnknownV5(i, "invalid")
	}
	if opts.ExecutorStatus == "pass" && (o.State != "validated" || o.ReportStatus != "success") {
		return e, nil, fmt.Errorf("successful executor requires validated exact-attempt v5 success report")
	}
	return e, &o, nil
}

func verifyStepExecutionV3(workdir string, e RunEvidence, success bool) error {
	o := e.StepExecution
	if o == nil {
		return fmt.Errorf("missing step observation")
	}
	if o.State != "validated" {
		if e.Executor.ReportPath != "" || e.Executor.ReportSHA256 != "" || e.Executor.ReportSize != 0 || success {
			return fmt.Errorf("unvalidated observation cannot reference a report or prove success")
		}
		return nil
	}
	clean, err := packageartifacts.CleanRelativePath(e.Executor.ReportPath)
	if err != nil || clean != e.Executor.ReportPath {
		return fmt.Errorf("invalid v5 report path")
	}
	data, info, err := evidencefile.ReadRegular(filepath.Join(workdir, filepath.FromSlash(clean)), 256<<10)
	if err != nil {
		return fmt.Errorf("read v5 report: %w", err)
	}
	if info.Size() != e.Executor.ReportSize || evidencefile.SHA256(data) != e.Executor.ReportSHA256 {
		return fmt.Errorf("v5 report size/digest mismatch")
	}
	observed := udonreport.ObserveV5(o.Inventory(), data)
	if !reflect.DeepEqual(observed, *o) || (success && o.ReportStatus != "success") {
		return fmt.Errorf("v5 report does not match exact run observation")
	}
	return nil
}
