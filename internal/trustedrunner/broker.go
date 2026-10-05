package trustedrunner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"

	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/OpenUdon/openudon/internal/udonrunner"
)

type BrokerPackageInspection struct {
	Version string `json:"version"`
	PackageInspection
	Plan udonrunner.BrokerPlan `json:"plan"`
}

// InspectBrokerPackage returns review metadata only. Its disposable staging
// never changes the package and supplies no approval, socket or capability.
func InspectBrokerPackage(ctx context.Context, opts TemplateOptions) (BrokerPackageInspection, error) {
	validated, err := validatePackage(ctx, packageOptions{RepoRoot: opts.RepoRoot, ExampleDir: opts.ExampleDir, Assess: opts.Assess})
	if err != nil {
		return BrokerPackageInspection{}, err
	}
	if err := validateManifestPolicy(validated.manifest); err != nil {
		return BrokerPackageInspection{}, err
	}
	plan, err := inspectBrokerValidated(ctx, validated)
	if err != nil {
		return BrokerPackageInspection{}, err
	}
	return BrokerPackageInspection{Version: "openudon.broker-package.v1", PackageInspection: PackageInspection{Scope: validated.paths.scope, PackageSHA256: validated.packageSHA256, HandoffSHA256: validated.handoffSHA256, ExecutionPolicy: validated.manifest.ExecutionPolicy, CredentialBindings: validated.manifest.CredentialBindings, ApprovalStates: validated.manifest.ApprovalStates}, Plan: plan}, nil
}

func inspectBrokerValidated(ctx context.Context, v validatedPackage) (udonrunner.BrokerPlan, error) {
	dir, err := os.MkdirTemp("", "broker-inspect-")
	if err != nil {
		return udonrunner.BrokerPlan{}, err
	}
	defer os.RemoveAll(dir)
	c, err := buildRunConfig(v.paths, v.manifest, v.snapshot, v.packageSHA256, TierSandbox, dir, "00000000000000000000000000000000", v.handoffSHA256, v.packageSHA256, nil)
	if err != nil {
		return udonrunner.BrokerPlan{}, err
	}
	if len(c.DataFiles) != 0 {
		return udonrunner.BrokerPlan{}, errors.New("broker profile excludes external data files")
	}
	c.ExecutorReportVersion = udonreport.VersionV5
	r, err := udonrunner.Prepare(ctx, c, udonrunner.Options{RepoRoot: v.paths.repoRoot, Env: []string{}})
	if err != nil {
		return udonrunner.BrokerPlan{}, err
	}
	return udonrunner.InspectBrokerPlan(r.StagePath, filepath.Join(r.StagePath, c.WorkflowPath), c.WorkflowFormat, c.APISourcePaths, *r.InventoryV5)
}

func bindBrokerConfig(c *RunConfig, approval Approval) error {
	if approval.Broker == nil {
		return nil
	}
	a := approval.Broker
	if a.RunID != c.RunID || a.PackageSHA256 != c.PackageSHA256 || a.HandoffSHA256 != c.HandoffSHA256 || c.Browser != nil || len(c.DataFiles) != 0 {
		return errors.New("broker approval differs from reviewed package or HTTP-only profile")
	}
	c.Version, c.Broker, c.ExecutorReportVersion = udonrunner.BrokerRunConfigVersion, a, udonreport.VersionV5
	return nil
}

func validateBrokerEvidence(e RunEvidence) error {
	a := e.Broker
	if e.Version != BrokerRunEvidenceVersion {
		if a != nil {
			return errors.New("legacy evidence cannot contain broker authority")
		}
		return nil
	}
	if a == nil || a.Validate() != nil || a.RunID != e.RunID || a.PackageSHA256 != e.PackageSHA256 || a.HandoffSHA256 != e.HandoffSHA256 || e.Browser != nil || len(e.CredentialEnvNames) != 0 || e.StepExecution == nil || len(e.StepExecution.Steps) != len(a.Operations) {
		return errors.New("broker evidence authority mismatch")
	}
	for i, op := range a.Operations {
		s := e.StepExecution.Steps[i]
		if s.StepID != op.StepID || s.OperationID != op.OperationID || s.InvocationID != op.InvocationID {
			return errors.New("broker evidence operation identity mismatch")
		}
	}
	return nil
}

func sameBrokerAuthority(c RunConfig, approval Approval) bool {
	return reflect.DeepEqual(c.Broker, approval.Broker)
}

func redactBrokerArgv(argv []string) []string {
	copy := append([]string(nil), argv...)
	for i := 0; i+1 < len(copy); i++ {
		if copy[i] == "--http-broker-config" {
			copy[i+1] = "[private broker config]"
		}
	}
	return copy
}
