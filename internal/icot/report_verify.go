package icot

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type reportVersionProbe struct {
	Version string `json:"version"`
}

func runReport(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] != "verify" {
		fmt.Fprintln(errOut, "Usage: icot report verify --file report.json")
		return 2
	}
	return runReportVerify(args[1:], out, errOut)
}

func runReportVerify(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot report verify", flag.ContinueOnError)
	fs.SetOutput(out)
	path := fs.String("file", "", "scorecard.json or authoring-eval.json report to verify")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot report verify --file report.json\n\n")
		fmt.Fprintf(fs.Output(), "Verifies iCoT scorecard or authoring-eval report JSON, counters, expected top issue diagnostics, failure categories, and .sha256 digest sidecar.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if strings.TrimSpace(*path) == "" {
		fmt.Fprintln(errOut, "--file is required")
		return 2
	}
	version, err := verifyReportFile(*path)
	if err != nil {
		fmt.Fprintf(errOut, "icot report verify: fail %s - %v\n", *path, err)
		return 1
	}
	fmt.Fprintf(out, "icot report verify: pass %s (%s)\n", *path, version)
	return 0
}

func verifyReportFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var probe reportVersionProbe
	if err := json.Unmarshal(data, &probe); err != nil {
		return "", fmt.Errorf("parse report JSON: %w", err)
	}
	if err := verifyJSONReportDigest(path); err != nil {
		return probe.Version, err
	}
	switch probe.Version {
	case scorecardReportVersion:
		var report scorecardReport
		if err := json.Unmarshal(data, &report); err != nil {
			return probe.Version, fmt.Errorf("parse scorecard report: %w", err)
		}
		if err := validateScorecardReport(report); err != nil {
			return probe.Version, err
		}
	case authoringEvalReportVersion:
		var report authoringEvalReport
		if err := json.Unmarshal(data, &report); err != nil {
			return probe.Version, fmt.Errorf("parse authoring-eval report: %w", err)
		}
		if err := validateAuthoringEvalReport(report); err != nil {
			return probe.Version, err
		}
	default:
		return probe.Version, fmt.Errorf("unsupported report version %q", probe.Version)
	}
	return probe.Version, nil
}
