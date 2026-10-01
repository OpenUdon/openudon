package browserpackage

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

func RunCommand(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	fail := func(message string, code int) int { fmt.Fprintln(errOut, message); return code }
	usage := func() {
		fmt.Fprintln(out, "usage: openudon browser-author <plan|apply> --example DIR --request FILE|- [--expected-plan sha256:HEX --confirmed]")
	}
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		usage()
		return 0
	}
	if args[0] != "plan" && args[0] != "apply" {
		return fail("browser author command invalid", 2)
	}
	operation := args[0]
	fs := flag.NewFlagSet("browser-author", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var example, requestPath, expectedPlan string
	var confirmed bool
	fs.StringVar(&example, "example", "", "package directory")
	fs.StringVar(&requestPath, "request", "", "bounded request file or stdin")
	fs.StringVar(&expectedPlan, "expected-plan", "", "exact approved plan digest")
	fs.BoolVar(&confirmed, "confirmed", false, "exact plan authoring approval")
	seen := map[string]bool{}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") && arg != "-" {
			key := strings.TrimLeft(strings.SplitN(arg, "=", 2)[0], "-")
			if seen[key] {
				return fail("browser author duplicate option", 2)
			}
			seen[key] = true
		}
	}
	err := fs.Parse(args[1:])
	if errors.Is(err, flag.ErrHelp) {
		usage()
		return 0
	}
	if ctx == nil || in == nil || err != nil || fs.NArg() != 0 || example == "" || requestPath == "" || operation == "plan" && (expectedPlan != "" || confirmed) || operation == "apply" && (!confirmed || !validDigest(expectedPlan)) {
		return fail("browser author options invalid", 2)
	}
	var data []byte
	if requestPath == "-" {
		data, err = io.ReadAll(io.LimitReader(in, MaxRequestBytes+1))
	} else {
		data, _, err = evidencefile.ReadRegular(requestPath, MaxRequestBytes)
	}
	if err != nil || len(data) > MaxRequestBytes {
		return fail("browser author request unavailable", 2)
	}
	var result any
	code := 0
	if operation == "plan" {
		result, err = Prepare(ctx, example, data)
		if err != nil {
			return fail("browser author planning refused", 1)
		}
	} else {
		applied, applyErr := Apply(ctx, example, data, expectedPlan, confirmed)
		if applyErr != nil {
			if applied.Version == "" {
				return fail("browser author apply refused; inspect before a new proposal", 1)
			}
			code = 1
			fmt.Fprintln(errOut, "browser author native build failed after authoring; no automatic retry")
		}
		result = applied
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > MaxReportBytes {
		return fail("browser author report unavailable; inspect package", 1)
	}
	if _, err := fmt.Fprintln(out, string(encoded)); err != nil {
		return fail("browser author output lost; inspect package", 1)
	}
	return code
}
