package browsersystem

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
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
