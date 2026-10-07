package runevidence

import (
	"errors"
	"fmt"
	"github.com/OpenUdon/openudon/digest"
	"strings"
)

var ErrUnsupportedBrowser = errors.New("browser evidence requires the retained exact CLI verifier")

func Validate(evidence RunEvidence) error {
	if evidence.Browser != nil {
		return ErrUnsupportedBrowser
	}
	if err := validateBrokerEvidence(evidence); err != nil {
		return err
	}
	if evidence.Version != Version && evidence.Version != StepVersion && evidence.Version != BrokerVersion && evidence.Version != LegacyVersion {
		return fmt.Errorf("run evidence version must be %s or read-only legacy %s", Version, LegacyVersion)
	}
	if strings.TrimSpace(evidence.Scope) == "" {
		return fmt.Errorf("run evidence scope is required")
	}
	if strings.TrimSpace(evidence.PackageSHA256) == "" {
		return fmt.Errorf("run evidence package_sha256 is required")
	}
	if strings.TrimSpace(evidence.RunConfigPath) == "" {
		return fmt.Errorf("run evidence run_config_path is required")
	}
	if strings.TrimSpace(evidence.WorkDir) == "" {
		return fmt.Errorf("run evidence workdir is required")
	}
	if evidence.Version == Version || evidence.Version == StepVersion || evidence.Version == BrokerVersion {
		if strings.TrimSpace(evidence.RunID) == "" || !digest.ValidSHA256(evidence.PackageSHA256) ||
			!digest.ValidSHA256(evidence.HandoffSHA256) || !digest.ValidSHA256(evidence.ApprovalSHA256) ||
			!digest.ValidSHA256(evidence.RunConfigSHA256) {
			return fmt.Errorf("run evidence v2 requires run_id and full package, handoff, approval, and config SHA-256 digests")
		}
		if err := validateRunEvidenceGates(evidence); err != nil {
			return err
		}

	}
	if evidence.Version == StepVersion || evidence.Version == BrokerVersion {
		if evidence.StepExecution == nil || evidence.Browser != nil {
			return fmt.Errorf("v3 requires HTTP step execution observation")
		}
		if err := evidence.StepExecution.Validate(); err != nil {
			return err
		}
		if evidence.StepExecution.RunID != evidence.RunID || (evidence.DryRun != (evidence.StepExecution.State == "dry_run")) {
			return fmt.Errorf("v3 observation identity/posture mismatch")
		}
	} else if evidence.StepExecution != nil {
		return fmt.Errorf("legacy evidence cannot contain v3 step observations")
	}
	return nil
}

func validateRunEvidenceGates(evidence RunEvidence) error {
	statuses := make(map[string]string, len(evidence.Gates))
	for _, gate := range evidence.Gates {
		name := strings.TrimSpace(gate.Name)
		status := strings.TrimSpace(gate.Status)
		if name == "" || (status != "pass" && status != "fail") {
			return fmt.Errorf("run evidence gate names are required and statuses must be pass or fail")
		}
		if _, exists := statuses[name]; exists {
			return fmt.Errorf("duplicate run evidence gate: %s", name)
		}
		statuses[name] = status
	}
	for _, name := range []string{"handoff_package", "manifest_policy", "stored_quality", "current_quality", "approval", "run_config", "staged_digest"} {
		if statuses[name] != "pass" {
			return fmt.Errorf("run evidence requires passing %s gate", name)
		}
	}
	executorStatus, hasExecutorGate := statuses["executor_invocation"]
	if evidence.DryRun {
		if hasExecutorGate || evidence.Executor.Invoked || evidence.Executor.Mode != "dry-run" || evidence.StageKind != "dry-run" {
			return fmt.Errorf("dry-run evidence must use the non-invoked dry-run execution posture")
		}
		return nil
	}
	if !hasExecutorGate || (executorStatus != "pass" && executorStatus != "fail") {
		return fmt.Errorf("non-dry-run evidence requires an executor_invocation gate")
	}
	if !evidence.Executor.Invoked {
		return fmt.Errorf("non-dry-run evidence must record an invoked executor")
	}
	switch evidence.Executor.Mode {
	case "internal-runner":
		if evidence.StageKind != "executor" {
			return fmt.Errorf("internal-runner evidence must use the executor stage")
		}
	case "external-runner":
		if evidence.StageKind != "preflight" {
			return fmt.Errorf("external-runner evidence must use the preflight stage")
		}
	default:
		return fmt.Errorf("non-dry-run evidence has unsupported executor mode %q", evidence.Executor.Mode)
	}
	return nil
}

func validateBrokerEvidence(e RunEvidence) error {
	a := e.Broker
	if e.Version != BrokerVersion {
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
