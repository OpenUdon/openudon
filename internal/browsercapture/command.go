package browsercapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	engine "github.com/OpenUdon/openudon/internal/authoringengine"
	"github.com/OpenUdon/openudon/internal/browserauthor"
	"github.com/OpenUdon/openudon/internal/browserauthoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

// RunCommand consumes bounded JSONL on standard I/O. A reviewed start file and
// its exact byte digest are required before any browser worker is started.
// Credentials/codes are entered only through the private human browser path.
func RunCommand(ctx context.Context, args []string, in io.ReadCloser, out io.WriteCloser, errOut io.Writer) int {
	fs := flag.NewFlagSet("browser-capture", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var startPath, approval, example, private, driver string
	fs.StringVar(&startPath, "start", "", "reviewed capture start JSON")
	fs.StringVar(&approval, "approve-start-sha256", "", "exact reviewed start-file SHA-256")
	fs.StringVar(&example, "example", "", "workflow package directory")
	fs.StringVar(&private, "private-root", "", "existing restrictive private directory outside the package")
	fs.StringVar(&driver, "driver-dir", "", "existing Playwright driver cache")
	fail := func(message string, code int) int { fmt.Fprintln(errOut, message); return code }
	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		fmt.Fprintln(errOut, "usage: openudon browser-capture --start FILE --approve-start-sha256 SHA --example DIR --private-root DIR [--driver-dir DIR]")
		return 0
	}
	if ctx == nil || in == nil || out == nil || parseErr != nil || fs.NArg() != 0 || !evidencefile.ValidSHA256(approval) {
		return fail("browser-capture requires --start, --approve-start-sha256, --example and --private-root", 2)
	}
	data, _, err := evidencefile.ReadRegular(startPath, MaxMessageBytes)
	if err != nil {
		return fail("browser capture start unavailable", 2)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != approval {
		return fail("browser capture start approval mismatch", 2)
	}
	request, err := DecodeStart(data)
	if err != nil {
		return fail("browser capture start invalid", 2)
	}
	example, private, err = browserauthoring.NormalizeRoots(example, private)
	if err != nil {
		return fail("browser capture workspace/private roots invalid", 2)
	}
	guard, err := engine.ObserveWorkspace(ctx, example)
	if err != nil {
		return fail("browser capture workspace unavailable", 2)
	}
	if request.Mode == Authenticated {
		config, live, configErr := request.AuthenticationConfig(example, private, driver)
		if configErr != nil {
			return fail("browser capture authenticated authority invalid", 2)
		}
		_, err = runAuthenticated(ctx, config, in, out, func(ctx context.Context, config browserauthor.Config) (authenticationSession, error) {
			return browserauthor.Start(ctx, config)
		}, authenticationOptions{
			continuation: live.AfterAuthentication,
			complete: func(ctx context.Context, event browserauthor.Event) (*profileImport, error) {
				if event.Result == nil || event.Attestation == nil {
					return nil, errWorker
				}
				prepared, err := browserauthoring.PrepareAttestedImport(live, browserauthoring.ProtocolResult{ArtifactPath: event.Result.ArtifactPath, Digest: event.Result.Digest, Attestation: event.Attestation}, time.Now().UTC())
				if err != nil {
					return nil, err
				}
				native, err := engine.AuthenticationCapabilityVirtualBrowserTransaction(prepared.Candidate, true)
				if err != nil {
					return nil, err
				}
				return prepareProfileImport(ctx, example, native, prepared.Files, approval, guard)
			},
		})
	} else {
		config, start, configErr := request.RegistrationConfig(private, driver)
		if configErr != nil {
			return fail("browser capture registration authority invalid", 2)
		}
		_, err = runRegistration(ctx, config, start, in, out, func(ctx context.Context, config browserauthor.RegistrationConfig) (registrationSession, error) {
			return browserauthor.StartRegistration(ctx, config)
		},
			func(ctx context.Context, event browserauthor.RegistrationEvent) (*profileImport, error) {
				if event.State != "candidate" || event.Candidate == nil {
					return nil, errWorker
				}
				native, err := engine.RegistrationVirtualBrowserTransaction(event.Candidate, true)
				if err != nil {
					return nil, err
				}
				return prepareProfileImport(ctx, example, native, nil, approval, guard)
			})
	}
	if err != nil {
		return fail("browser capture did not complete; no automatic retry", 1)
	}
	return 0
}
