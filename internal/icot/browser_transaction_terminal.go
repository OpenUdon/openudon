package icot

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/OpenUdon/openudon/internal/browsertransaction"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/browsertransaction/presentation"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
)

const browserTransactionTerminalVersion = "openudon.browser-transaction-terminal.v1"

type browserTransactionTerminalOptions struct {
	transactionPath string
	exampleDir      string
	scope           string
	scratchParent   string
	storeDir        string
	review          bool
	prepare         bool
	promote         bool
	recover         bool
	inspectSelected bool
}

type browserTransactionPackageOptions struct {
	exampleDir    string
	scope         string
	scratchParent string
	storeDir      string
}

type browserTransactionTerminalEvent struct {
	Version            string                `json:"version"`
	Event              string                `json:"event"`
	BrowserTransaction presentation.Resource `json:"browser_transaction"`
}

type browserTransactionTerminalFailure struct {
	Version   string                          `json:"version"`
	Class     browsertransaction.FailureClass `json:"class"`
	Code      transactionengine.ErrorCode     `json:"code"`
	Operation transactionengine.Operation     `json:"operation"`
	Retryable bool                            `json:"retryable"`
}

type browserTransactionLifecycle interface {
	Observe(context.Context) (transactionengine.Snapshot, error)
	Start(context.Context, transactionengine.StartRequest) (transactionengine.Snapshot, error)
	Review(context.Context, transactionengine.ReviewRequest) (transactionengine.Snapshot, error)
	Prepare(context.Context, transactionengine.PrepareRequest) (transactionengine.Snapshot, error)
	Promote(context.Context, transactionengine.PromoteRequest) (transactionengine.Snapshot, error)
	Cancel(context.Context, transactionengine.CancelRequest) (transactionengine.Snapshot, error)
	InspectRecovery(context.Context, transactionengine.InspectRecoveryRequest) (transactionengine.Snapshot, error)
	Recover(context.Context, transactionengine.RecoverRequest) (transactionengine.Snapshot, error)
	InspectSelected(context.Context, transactionengine.InspectSelectedRequest) (transactionengine.Snapshot, error)
}

func runBrowserTransaction(args []string, in io.Reader, out, errOut io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runBrowserTransactionContext(ctx, args, in, out, errOut)
}

func runBrowserTransactionContext(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot browser-transaction", flag.ContinueOnError)
	fs.SetOutput(out)
	options := browserTransactionTerminalOptions{}
	fs.StringVar(&options.transactionPath, "transaction", "", "Public browser-profile transaction v1 JSON file")
	fs.StringVar(&options.exampleDir, "example", "", "Reviewed package directory")
	fs.StringVar(&options.scope, "scope", "", "Explicit portable package scope")
	fs.StringVar(&options.scratchParent, "scratch", "", "Existing absolute restrictive-scratch parent")
	fs.StringVar(&options.storeDir, "store", "", "Existing package generation store")
	fs.BoolVar(&options.review, "review", false, "Request exact candidate review authorization")
	fs.BoolVar(&options.prepare, "prepare", false, "Request separate scratch preparation and qualification authorization")
	fs.BoolVar(&options.promote, "promote", false, "Request separate exact atomic promotion authorization")
	fs.BoolVar(&options.recover, "recover", false, "After indeterminate promotion, inspect and request exact recovery authorization")
	fs.BoolVar(&options.inspectSelected, "inspect-selected", false, "After promotion, inspect the exact selected package without runtime execution")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: icot browser-transaction --transaction FILE --example DIR --scope PORTABLE --scratch /absolute/dir --store /absolute/store [--review|--prepare|--promote] [--recover] [--inspect-selected]")
		fmt.Fprintln(fs.Output(), "Emits value-free NDJSON state on stdout. Exact digest-bound authorizations are read from stdin; no browser or runtime operation is available.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "icot browser-transaction: unexpected positional arguments")
		return 2
	}
	if options.promote {
		options.prepare, options.review = true, true
	}
	if options.prepare {
		options.review = true
	}
	if options.recover && !options.promote {
		fmt.Fprintln(errOut, "icot browser-transaction: --recover requires --promote")
		return 2
	}
	if options.inspectSelected && !options.promote {
		fmt.Fprintln(errOut, "icot browser-transaction: --inspect-selected requires --promote")
		return 2
	}
	if strings.TrimSpace(options.transactionPath) == "" || strings.TrimSpace(options.exampleDir) == "" || strings.TrimSpace(options.scope) == "" || strings.TrimSpace(options.scratchParent) == "" || strings.TrimSpace(options.storeDir) == "" {
		fmt.Fprintln(errOut, "icot browser-transaction: --transaction, --example, --scope, --scratch, and --store are required")
		return 2
	}

	lifecycle, started, err := openBrowserTransaction(ctx, options.transactionPath, browserTransactionPackageOptions{
		exampleDir: options.exampleDir, scope: options.scope, scratchParent: options.scratchParent, storeDir: options.storeDir,
	})
	if err != nil {
		fmt.Fprintln(errOut, "icot browser-transaction: public transaction input is unavailable or unstable")
		return writeBrowserTransactionEngineFailure(errOut, err)
	}
	if eventErr := writeBrowserTransactionTerminalEvent(out, "started", started); eventErr != nil {
		fmt.Fprintln(errOut, "icot browser-transaction: stdout write failed")
		return 1
	}
	authorizations := newBrowserTransactionAuthorizationReader(in)
	defer authorizations.Close()
	return runBrowserTransactionLifecycle(ctx, lifecycle, started, options, authorizations, out, errOut)
}

func openBrowserTransaction(ctx context.Context, transactionPath string, options browserTransactionPackageOptions) (*transactionengine.Engine, transactionengine.Snapshot, error) {
	transactionData, err := readBrowserTransactionFile(transactionPath)
	if err != nil {
		return nil, transactionengine.Snapshot{}, &transactionengine.Error{Class: browsertransaction.FailureRejected, Code: transactionengine.ErrorInvalidRequest, Operation: transactionengine.OperationStart}
	}
	transaction, err := browsertransaction.Decode(transactionData)
	if err != nil {
		return nil, transactionengine.Snapshot{}, &transactionengine.Error{Class: browsertransaction.FailureRejected, Code: transactionengine.ErrorTransactionInvalid, Operation: transactionengine.OperationStart}
	}
	transactionSHA256, err := browsertransaction.Digest(transaction)
	if err != nil {
		return nil, transactionengine.Snapshot{}, &transactionengine.Error{Class: browsertransaction.FailureRejected, Code: transactionengine.ErrorTransactionInvalid, Operation: transactionengine.OperationStart}
	}
	lifecycle, initial, err := transactionengine.New(transactionengine.Config{Package: packagepipeline.CurrentOptions{
		ExampleDir: options.exampleDir, Scope: options.scope, ScratchParent: options.scratchParent, StoreDir: options.storeDir,
	}})
	if err != nil {
		return nil, transactionengine.Snapshot{}, err
	}
	started, err := lifecycle.Start(ctx, transactionengine.StartRequest{
		ExpectedRevision: initial.Revision, ExpectedTransactionSHA256: transactionSHA256, TransactionJSON: transactionData,
	})
	if err != nil {
		return nil, started, err
	}
	return lifecycle, started, nil
}

func runBrowserTransactionLifecycle(ctx context.Context, lifecycle browserTransactionLifecycle, snapshot transactionengine.Snapshot, options browserTransactionTerminalOptions, authorizations *browserTransactionAuthorizationReader, out, errOut io.Writer) int {
	if !options.review {
		return 0
	}
	if snapshot.Transaction.State == browsertransaction.StateCandidate {
		decision, ok := requireBrowserTransactionAuthorization(ctx, authorizations, errOut, transactionengine.OperationReview,
			"review "+snapshot.TransactionSHA256, "cancel "+snapshot.TransactionSHA256)
		if !ok {
			return browserTransactionAuthorizationFailure(ctx, errOut, transactionengine.OperationReview)
		}
		if decision == "cancel" {
			return cancelBrowserTransaction(ctx, lifecycle, snapshot, out, errOut)
		}
		var err error
		snapshot, err = lifecycle.Review(ctx, transactionengine.ReviewRequest{Authority: browserTransactionAuthority(snapshot)})
		if eventErr := writeBrowserTransactionTerminalEvent(out, "review", snapshot); eventErr != nil {
			return 1
		}
		if err != nil {
			return writeBrowserTransactionEngineFailure(errOut, err)
		}
	}
	if !options.prepare {
		return 0
	}
	decision, ok := requireBrowserTransactionAuthorization(ctx, authorizations, errOut, transactionengine.OperationPrepare,
		"prepare "+snapshot.TransactionSHA256, "cancel "+snapshot.TransactionSHA256)
	if !ok {
		return browserTransactionAuthorizationFailure(ctx, errOut, transactionengine.OperationPrepare)
	}
	if decision == "cancel" {
		return cancelBrowserTransaction(ctx, lifecycle, snapshot, out, errOut)
	}
	var err error
	snapshot, err = lifecycle.Prepare(ctx, transactionengine.PrepareRequest{Authority: browserTransactionAuthority(snapshot)})
	if eventErr := writeBrowserTransactionTerminalEvent(out, "prepare", snapshot); eventErr != nil {
		return 1
	}
	if err != nil {
		return writeBrowserTransactionEngineFailure(errOut, err)
	}
	if !options.promote {
		return 0
	}
	decision, ok = requireBrowserTransactionAuthorization(ctx, authorizations, errOut, transactionengine.OperationPromote,
		strings.Join([]string{"promote", snapshot.TransactionSHA256, snapshot.Preparation.PreparationSHA256, snapshot.Preparation.QualificationSHA256}, " "),
		"cancel "+snapshot.TransactionSHA256)
	if !ok {
		return browserTransactionAuthorizationFailure(ctx, errOut, transactionengine.OperationPromote)
	}
	if decision == "cancel" {
		return cancelBrowserTransaction(ctx, lifecycle, snapshot, out, errOut)
	}
	snapshot, err = lifecycle.Promote(ctx, transactionengine.PromoteRequest{
		Authority: browserTransactionAuthority(snapshot), ExpectedPreparationSHA256: snapshot.Preparation.PreparationSHA256,
		ExpectedQualificationSHA256: snapshot.Preparation.QualificationSHA256,
	})
	if eventErr := writeBrowserTransactionTerminalEvent(out, "promote", snapshot); eventErr != nil {
		return 1
	}
	if err != nil {
		_, code, _, _, _ := transactionengine.ErrorDetails(err)
		writeBrowserTransactionEngineFailure(errOut, err)
		if !options.recover || (code != transactionengine.ErrorPromotionIndeterminate && code != transactionengine.ErrorRecoveryRequired) {
			return browserTransactionExitCode(ctx, err)
		}
		snapshot, err = lifecycle.InspectRecovery(ctx, transactionengine.InspectRecoveryRequest{ExpectedRevision: snapshot.Revision})
		if eventErr := writeBrowserTransactionTerminalEvent(out, "inspect_recovery", snapshot); eventErr != nil {
			return 1
		}
		if err != nil {
			return writeBrowserTransactionEngineFailure(errOut, err)
		}
		if snapshot.Recovery == nil || snapshot.Recovery.Report == nil {
			writeBrowserTransactionTerminalFailure(errOut, browsertransaction.FailureIndeterminate, transactionengine.ErrorRecoveryRequired, transactionengine.OperationRecover, false)
			return 1
		}
		decision, ok = requireBrowserTransactionAuthorization(ctx, authorizations, errOut, transactionengine.OperationRecover,
			strings.Join([]string{"recover", snapshot.TransactionSHA256, snapshot.Recovery.Report.RecoverySHA256}, " "),
			"cancel "+snapshot.TransactionSHA256)
		if !ok {
			return browserTransactionAuthorizationFailure(ctx, errOut, transactionengine.OperationRecover)
		}
		if decision == "cancel" {
			return cancelBrowserTransaction(ctx, lifecycle, snapshot, out, errOut)
		}
		snapshot, err = lifecycle.Recover(ctx, transactionengine.RecoverRequest{
			Authority: browserTransactionAuthority(snapshot), ExpectedRecoverySHA256: snapshot.Recovery.Report.RecoverySHA256,
		})
		if eventErr := writeBrowserTransactionTerminalEvent(out, "recover", snapshot); eventErr != nil {
			return 1
		}
		if err != nil {
			return writeBrowserTransactionEngineFailure(errOut, err)
		}
	}
	if options.inspectSelected {
		if snapshot.Promotion == nil {
			writeBrowserTransactionTerminalFailure(errOut, browsertransaction.FailureConflict, transactionengine.ErrorInvalidState, transactionengine.OperationInspectSelected, false)
			return 1
		}
		snapshot, err = lifecycle.InspectSelected(ctx, transactionengine.InspectSelectedRequest{
			ExpectedRevision: snapshot.Revision, ExpectedSelectionSHA256: snapshot.Promotion.SelectionSHA256,
		})
		if eventErr := writeBrowserTransactionTerminalEvent(out, "inspect_selected", snapshot); eventErr != nil {
			return 1
		}
		if err != nil {
			return writeBrowserTransactionEngineFailure(errOut, err)
		}
	}
	return 0
}

func browserTransactionAuthority(snapshot transactionengine.Snapshot) transactionengine.Authority {
	return transactionengine.Authority{ExpectedRevision: snapshot.Revision, ExpectedTransactionSHA256: snapshot.TransactionSHA256, HumanApproved: true}
}

func cancelBrowserTransaction(ctx context.Context, lifecycle browserTransactionLifecycle, snapshot transactionengine.Snapshot, out, errOut io.Writer) int {
	cancelled, err := lifecycle.Cancel(ctx, transactionengine.CancelRequest{Authority: browserTransactionAuthority(snapshot)})
	if eventErr := writeBrowserTransactionTerminalEvent(out, "cancel", cancelled); eventErr != nil {
		return 1
	}
	if err != nil {
		return writeBrowserTransactionEngineFailure(errOut, err)
	}
	return 0
}

func requireBrowserTransactionAuthorization(ctx context.Context, reader *browserTransactionAuthorizationReader, errOut io.Writer, operation transactionengine.Operation, expected, cancel string) (string, bool) {
	fmt.Fprintf(errOut, "browser transaction %s authorization required; type exactly %q or %q\n", operation, expected, cancel)
	line, err := reader.Read(ctx)
	if err != nil {
		return "", false
	}
	switch line {
	case expected:
		return string(operation), true
	case cancel:
		return "cancel", true
	default:
		return "", false
	}
}

func browserTransactionAuthorizationFailure(ctx context.Context, errOut io.Writer, operation transactionengine.Operation) int {
	if ctx.Err() != nil {
		writeBrowserTransactionTerminalFailure(errOut, browsertransaction.FailureOperational, transactionengine.ErrorCanceled, operation, true)
		return 130
	}
	writeBrowserTransactionTerminalFailure(errOut, browsertransaction.FailureRejected, transactionengine.ErrorAuthorizationRequired, operation, false)
	return 1
}

func writeBrowserTransactionTerminalEvent(out io.Writer, event string, snapshot transactionengine.Snapshot) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(browserTransactionTerminalEvent{Version: browserTransactionTerminalVersion, Event: event, BrowserTransaction: presentation.New(snapshot)})
}

func writeBrowserTransactionEngineFailure(errOut io.Writer, err error) int {
	class, code, operation, retryable, ok := transactionengine.ErrorDetails(err)
	if !ok {
		class, code, operation, retryable = browsertransaction.FailureOperational, transactionengine.ErrorInvalidRequest, transactionengine.OperationObserve, false
	}
	writeBrowserTransactionTerminalFailure(errOut, class, code, operation, retryable)
	return browserTransactionExitCode(context.Background(), err)
}

func writeBrowserTransactionTerminalFailure(errOut io.Writer, class browsertransaction.FailureClass, code transactionengine.ErrorCode, operation transactionengine.Operation, retryable bool) {
	encoder := json.NewEncoder(errOut)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(browserTransactionTerminalFailure{Version: browserTransactionTerminalVersion, Class: class, Code: code, Operation: operation, Retryable: retryable})
}

func browserTransactionExitCode(ctx context.Context, err error) int {
	_, code, _, _, _ := transactionengine.ErrorDetails(err)
	if ctx.Err() != nil || code == transactionengine.ErrorCanceled {
		return 130
	}
	return 1
}

func readBrowserTransactionFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() > browsertransaction.MaxBytes {
		return nil, errors.New("browser transaction input is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, browsertransaction.MaxBytes+1))
	if err != nil || len(data) > browsertransaction.MaxBytes {
		return nil, errors.New("browser transaction input is unreadable or oversized")
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, errors.New("browser transaction input changed while being read")
	}
	return data, nil
}

type browserTransactionAuthorizationResult struct {
	line string
	err  error
}

type browserTransactionAuthorizationReader struct {
	results <-chan browserTransactionAuthorizationResult
	done    chan struct{}
	once    sync.Once
}

func newBrowserTransactionAuthorizationReader(input io.Reader) *browserTransactionAuthorizationReader {
	results := make(chan browserTransactionAuthorizationResult, 1)
	done := make(chan struct{})
	reader := &browserTransactionAuthorizationReader{results: results, done: done}
	go func() {
		defer close(results)
		scanner := bufio.NewScanner(input)
		scanner.Buffer(make([]byte, 1024), 4096)
		for scanner.Scan() {
			result := browserTransactionAuthorizationResult{line: scanner.Text()}
			select {
			case results <- result:
			case <-done:
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case results <- browserTransactionAuthorizationResult{err: err}:
			case <-done:
			}
		}
	}()
	return reader
}

func (reader *browserTransactionAuthorizationReader) Read(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result, ok := <-reader.results:
		if !ok {
			return "", io.EOF
		}
		return result.line, result.err
	}
}

func (reader *browserTransactionAuthorizationReader) Close() {
	reader.once.Do(func() { close(reader.done) })
}
