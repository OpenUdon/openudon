package browsersystem

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/OpenUdon/openudon/internal/browserscenario"
)

// Private diagnostics are never report evidence or part of an error string.
// Keep only a bounded tail of each stream, where test failures normally appear.
const diagnosticStreamLimit = 1 << 20

type commandFailure struct {
	reason         string
	stdout, stderr []byte
	truncated      bool
}

func (*commandFailure) Error() string { return "component_failed" }

// Scenario evaluation is in-process, so there are no child streams to retain.
// Preserve its returned report and cause only through the existing bounded
// private diagnostic channel; neither can become successful aggregate evidence.
func scenarioFailure(report *browserscenario.Report, cause error) error {
	if cause == nil {
		return nil
	}
	data, _ := json.Marshal(report)
	detail := []byte(cause.Error())
	if diagnostic, err := browserscenario.AuthoringFailureDiagnostic(report); err != nil {
		detail = []byte("scenario_diagnostic_invalid")
	} else if len(diagnostic) != 0 {
		detail = diagnostic
	}
	return &commandFailure{reason: "scenario_evaluation", stdout: data, stderr: detail}
}

func retainFailureDiagnostic(out, stage string, cause error, progress io.Writer) {
	diagnostic := struct {
		Version   string `json:"version"`
		Stage     string `json:"stage"`
		Reason    string `json:"reason"`
		Stdout    string `json:"private_stdout"`
		Stderr    string `json:"private_stderr"`
		Truncated bool   `json:"truncated"`
	}{Version: "openudon.browser-system-diagnostic.v1", Stage: stage, Reason: "source_or_runtime_binding"}
	if cause != nil {
		diagnostic.Reason = "component_validation"
	}
	var failure *commandFailure
	if errors.As(cause, &failure) {
		diagnostic.Reason = failure.reason
		diagnostic.Truncated = failure.truncated
		tail := func(data []byte) string {
			if len(data) > diagnosticStreamLimit {
				diagnostic.Truncated = true
				data = data[len(data)-diagnosticStreamLimit:]
			}
			return string(data)
		}
		diagnostic.Stdout, diagnostic.Stderr = tail(failure.stdout), tail(failure.stderr)
	}
	data, err := json.Marshal(diagnostic)
	path := out + ".diagnostic.json"
	if err == nil {
		var file *os.File
		// Refuse stale files and final symlinks, including earlier failed runs.
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			_, writeErr := file.Write(append(data, '\n'))
			err = errors.Join(writeErr, file.Sync(), file.Close())
		}
	}
	if progress != nil {
		if err != nil {
			fmt.Fprintln(progress, "browser-system-eval: private diagnostic unavailable")
		} else {
			fmt.Fprintf(progress, "browser-system-eval: private diagnostic %q\n", path)
		}
	}
}

// ComponentFailureCode exposes only fixed qualification phases. Dynamic errors,
// including executor paths and private input values, never enter CLI output.
func ComponentFailureCode(cause error) string {
	if cause == nil {
		return "none"
	}
	message := cause.Error()
	for _, entry := range []struct{ prefix, code string }{
		{"BRP public capture qualification:", "registration_capture"},
		{"BRP attested runtime execution failed", "registration_execution"},
		{"registration private UI qualification: apply readiness", "registration_apply_readiness"},
		{"registration private UI checkpoint qualification", "registration_checkpoint"},
		{"registration private UI completion or artifact privacy failed", "registration_completion"},
		{"registration private UI teardown", "registration_teardown"},
		{"registration private UI qualification:", "registration_input_setup"},
		{"BRP", "registration_package"},
		{"BAP", "authenticated_package"},
	} {
		if strings.HasPrefix(message, entry.prefix) {
			return entry.code
		}
	}
	return "qualification_failed"
}
