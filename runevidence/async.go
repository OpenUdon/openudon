package runevidence

import (
	"fmt"
	asyncevidence "github.com/OpenUdon/evidence/async"
)

func validateAsyncEvidenceBundle(bundle AsyncEvidenceBundle) error {
	if bundle.Version != AsyncEvidenceVersion {
		return fmt.Errorf("async evidence bundle version must be %s", AsyncEvidenceVersion)
	}
	if len(bundle.Records) == 0 {
		return fmt.Errorf("async evidence bundle requires records")
	}
	for i, record := range bundle.Records {
		switch record.Kind {
		case "execution_request":
			if record.ExecutionRequest == nil {
				return fmt.Errorf("async evidence record %d missing execution_request", i)
			}
			if asyncRecordPayloadCount(record) != 1 {
				return fmt.Errorf("async evidence record %d has unexpected payload for execution_request", i)
			}
			if diagnostics := asyncevidence.ValidateExecutionRequest(*record.ExecutionRequest); len(diagnostics) != 0 {
				return fmt.Errorf("async evidence request record %d is invalid: %s", i, diagnostics[0].Code)
			}
		case "execution_response":
			if record.ExecutionResponse == nil {
				return fmt.Errorf("async evidence record %d missing execution_response", i)
			}
			if asyncRecordPayloadCount(record) != 1 {
				return fmt.Errorf("async evidence record %d has unexpected payload for execution_response", i)
			}
			if diagnostics := asyncevidence.ValidateExecutionResponse(*record.ExecutionResponse); len(diagnostics) != 0 {
				return fmt.Errorf("async evidence response record %d is invalid: %s", i, diagnostics[0].Code)
			}
		case "status_observation":
			if record.StatusObservation == nil {
				return fmt.Errorf("async evidence record %d missing status_observation", i)
			}
			if asyncRecordPayloadCount(record) != 1 {
				return fmt.Errorf("async evidence record %d has unexpected payload for status_observation", i)
			}
			if diagnostics := asyncevidence.ValidateStatusObservation(*record.StatusObservation); len(diagnostics) != 0 {
				return fmt.Errorf("async evidence status record %d is invalid: %s", i, diagnostics[0].Code)
			}
		case "confirmation_read_observation":
			if record.ConfirmationReadObservation == nil {
				return fmt.Errorf("async evidence record %d missing confirmation_read_observation", i)
			}
			if asyncRecordPayloadCount(record) != 1 {
				return fmt.Errorf("async evidence record %d has unexpected payload for confirmation_read_observation", i)
			}
			if diagnostics := asyncevidence.ValidateConfirmationReadObservation(*record.ConfirmationReadObservation); len(diagnostics) != 0 {
				return fmt.Errorf("async evidence confirmation-read record %d is invalid: %s", i, diagnostics[0].Code)
			}
		default:
			return fmt.Errorf("unsupported async evidence record kind %q", record.Kind)
		}
	}
	return nil
}

func asyncRecordPayloadCount(record AsyncEvidenceRecord) int {
	count := 0
	if record.ExecutionRequest != nil {
		count++
	}
	if record.ExecutionResponse != nil {
		count++
	}
	if record.StatusObservation != nil {
		count++
	}
	if record.ConfirmationReadObservation != nil {
		count++
	}
	return count
}
