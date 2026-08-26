// Package engine coordinates the value-free browser-profile transaction
// lifecycle. It has no browser, credential, prompting, or runtime execution
// authority; frontends supply explicit revisions and human authorization.
package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

const Version = "openudon.browser-transaction-engine.v1"

type Operation string

const (
	OperationStart           Operation = "start"
	OperationObserve         Operation = "observe"
	OperationReview          Operation = "review"
	OperationPrepare         Operation = "prepare"
	OperationPromote         Operation = "promote"
	OperationCancel          Operation = "cancel"
	OperationInspectRecovery Operation = "inspect_recovery"
	OperationRecover         Operation = "recover"
	OperationInspectSelected Operation = "inspect_selected"
)

type ErrorCode string

const (
	ErrorInvalidRequest         ErrorCode = "invalid_request"
	ErrorTransactionInvalid     ErrorCode = "transaction_invalid"
	ErrorInactive               ErrorCode = "transaction_inactive"
	ErrorInvalidState           ErrorCode = "invalid_state"
	ErrorStaleRevision          ErrorCode = "stale_revision"
	ErrorAuthorizationRequired  ErrorCode = "authorization_required"
	ErrorDigestMismatch         ErrorCode = "digest_mismatch"
	ErrorCandidateStale         ErrorCode = "candidate_stale"
	ErrorPreparationFailed      ErrorCode = "preparation_failed"
	ErrorQualificationFailed    ErrorCode = "qualification_failed"
	ErrorPromotionFailed        ErrorCode = "promotion_failed"
	ErrorPromotionRolledBack    ErrorCode = "promotion_rolled_back"
	ErrorPromotionIndeterminate ErrorCode = "promotion_indeterminate"
	ErrorRecoveryRequired       ErrorCode = "recovery_required"
	ErrorRecoveryDrift          ErrorCode = "recovery_drift"
	ErrorInspectionFailed       ErrorCode = "inspection_failed"
	ErrorCanceled               ErrorCode = "canceled"
)

// Error is the shared, path-free frontend failure surface. The underlying
// package error is deliberately not exposed through Error or JSON.
type Error struct {
	Class     browsertransaction.FailureClass `json:"class"`
	Code      ErrorCode                       `json:"code"`
	Operation Operation                       `json:"operation"`
	Retryable bool                            `json:"retryable"`
	cause     error
}

func (err *Error) Error() string {
	if err == nil {
		return "browser transaction engine failed"
	}
	return "browser transaction engine failed: " + string(err.Code)
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

// ErrorDetails returns the stable frontend error fields.
func ErrorDetails(err error) (browsertransaction.FailureClass, ErrorCode, Operation, bool, bool) {
	var typed *Error
	if !errors.As(err, &typed) {
		return "", "", "", false, false
	}
	return typed.Class, typed.Code, typed.Operation, typed.Retryable, true
}

type Authority struct {
	ExpectedRevision          string `json:"revision"`
	ExpectedTransactionSHA256 string `json:"transaction_sha256"`
	HumanApproved             bool   `json:"human_approved"`
}

type StartRequest struct {
	ExpectedRevision          string          `json:"revision"`
	ExpectedTransactionSHA256 string          `json:"transaction_sha256"`
	TransactionJSON           json.RawMessage `json:"transaction"`
}

type ReviewRequest struct{ Authority }
type PrepareRequest struct{ Authority }

type PromoteRequest struct {
	Authority
	ExpectedPreparationSHA256   string `json:"preparation_sha256"`
	ExpectedQualificationSHA256 string `json:"qualification_sha256"`
}

type CancelRequest struct{ Authority }

type InspectRecoveryRequest struct {
	ExpectedRevision string `json:"revision"`
}

type RecoverRequest struct {
	Authority
	ExpectedRecoverySHA256 string `json:"recovery_sha256"`
}

type InspectSelectedRequest struct {
	ExpectedRevision        string `json:"revision"`
	ExpectedSelectionSHA256 string `json:"selection_sha256"`
}

type PreparationEvidence struct {
	PreparationSHA256   string `json:"preparation_sha256"`
	InputSHA256         string `json:"input_sha256"`
	PackageSHA256       string `json:"package_sha256"`
	HandoffSHA256       string `json:"handoff_sha256"`
	QualitySHA256       string `json:"quality_sha256"`
	QualificationSHA256 string `json:"qualification_sha256"`
}

type PromotionEvidence struct {
	GenerationSHA256         string `json:"generation_sha256"`
	SelectionSHA256          string `json:"selection_sha256"`
	BaselineSelectionSHA256  string `json:"baseline_selection_sha256,omitempty"`
	SelectedGenerationSHA256 string `json:"selected_generation_sha256"`
	PriorGenerationSHA256    string `json:"prior_generation_sha256,omitempty"`
}

type RecoveryEvidence struct {
	Report         *packagepipeline.RecoveryReport `json:"report,omitempty"`
	Reconciliation *packagepipeline.Reconciliation `json:"reconciliation,omitempty"`
}

type OperationFailure struct {
	Class                  browsertransaction.FailureClass `json:"class"`
	Code                   ErrorCode                       `json:"code"`
	Operation              Operation                       `json:"operation"`
	Retryable              bool                            `json:"retryable"`
	PromotionState         packagepipeline.PromotionState  `json:"promotion_state,omitempty"`
	TargetGenerationSHA256 string                          `json:"target_generation_sha256,omitempty"`
}

// Snapshot is the complete value-free state shared by terminal and UI
// adapters. It intentionally has no filesystem, source-body, result-path,
// worker-output, credential-value, or runtime-execution fields.
type Snapshot struct {
	Version                   string                           `json:"version"`
	Revision                  string                           `json:"revision"`
	Transaction               *browsertransaction.Transaction  `json:"transaction,omitempty"`
	TransactionSHA256         string                           `json:"transaction_sha256,omitempty"`
	Preparation               *PreparationEvidence             `json:"preparation,omitempty"`
	Promotion                 *PromotionEvidence               `json:"promotion,omitempty"`
	Recovery                  *RecoveryEvidence                `json:"recovery,omitempty"`
	Inspection                *trustedrunner.PackageInspection `json:"inspection,omitempty"`
	LastFailure               *OperationFailure                `json:"last_failure,omitempty"`
	AllowedOperations         []Operation                      `json:"allowed_operations"`
	RuntimeExecutionSupported bool                             `json:"runtime_execution_supported"`
}

type Config struct {
	Package packagepipeline.CurrentOptions
	Now     func() time.Time

	operations packageOperations
}

type Engine struct {
	mu        sync.Mutex
	config    Config
	sequence  uint64
	snapshot  Snapshot
	qualified *packagepipeline.Qualified
}

type packageOperations interface {
	prepareAndQualify(context.Context, packagepipeline.CurrentOptions) (packagepipeline.Qualified, error)
	promote(context.Context, packagepipeline.Qualified, packagepipeline.PromotionOptions) (packagepipeline.Promoted, error)
	readCurrent(context.Context, string) (packagepipeline.Promoted, error)
	inspectRecovery(context.Context, string) (packagepipeline.RecoveryReport, error)
	reconcile(context.Context, packagepipeline.ReconcileOptions) (packagepipeline.Reconciliation, error)
	inspectSelected(context.Context, string, string) (trustedrunner.PackageInspection, error)
}

type defaultPackageOperations struct{}

func (defaultPackageOperations) prepareAndQualify(ctx context.Context, opts packagepipeline.CurrentOptions) (packagepipeline.Qualified, error) {
	return packagepipeline.PrepareAndQualifyCurrent(ctx, opts)
}
func (defaultPackageOperations) promote(ctx context.Context, qualified packagepipeline.Qualified, opts packagepipeline.PromotionOptions) (packagepipeline.Promoted, error) {
	return packagepipeline.Promote(ctx, qualified, opts)
}
func (defaultPackageOperations) readCurrent(ctx context.Context, store string) (packagepipeline.Promoted, error) {
	return packagepipeline.ReadCurrent(ctx, store)
}
func (defaultPackageOperations) inspectRecovery(ctx context.Context, store string) (packagepipeline.RecoveryReport, error) {
	return packagepipeline.InspectRecovery(ctx, store)
}
func (defaultPackageOperations) reconcile(ctx context.Context, opts packagepipeline.ReconcileOptions) (packagepipeline.Reconciliation, error) {
	return packagepipeline.Reconcile(ctx, opts)
}
func (defaultPackageOperations) inspectSelected(ctx context.Context, store, selection string) (trustedrunner.PackageInspection, error) {
	return packagepipeline.InspectSelected(ctx, store, selection)
}

// New creates an inactive, observable engine without inspecting or changing
// any package, browser, or runtime state.
func New(config Config) (*Engine, Snapshot, error) {
	if strings.TrimSpace(config.Package.ExampleDir) == "" || strings.TrimSpace(config.Package.Scope) == "" ||
		strings.TrimSpace(config.Package.ScratchParent) == "" || strings.TrimSpace(config.Package.StoreDir) == "" {
		return nil, Snapshot{}, engineError(browsertransaction.FailureRejected, ErrorInvalidRequest, OperationStart, false, nil)
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.operations == nil {
		config.operations = defaultPackageOperations{}
	}
	engine := &Engine{config: config, snapshot: Snapshot{Version: Version}}
	if err := engine.refreshLocked(); err != nil {
		return nil, Snapshot{}, err
	}
	result, err := cloneSnapshot(engine.snapshot)
	return engine, result, err
}

// Observe returns a defensive copy and performs no filesystem or lifecycle
// mutation.
func (engine *Engine) Observe(ctx context.Context) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationObserve, false, nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorCanceled, OperationObserve, true, err)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	return cloneSnapshot(engine.snapshot)
}

// Start strictly decodes one digest-bound public candidate or reviewed
// transaction. Candidate and review bodies never enter the engine.
func (engine *Engine) Start(ctx context.Context, request StartRequest) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationStart, false, nil)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if err := engine.requireContext(ctx, OperationStart); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	if engine.snapshot.Transaction != nil {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureConflict, ErrorInvalidState, OperationStart, false, nil)
	}
	if err := engine.requireRevision(request.ExpectedRevision, OperationStart); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	transaction, err := browsertransaction.Decode(request.TransactionJSON)
	if err != nil {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureRejected, ErrorTransactionInvalid, OperationStart, false, err)
	}
	digest, err := browsertransaction.Digest(transaction)
	if err != nil || request.ExpectedTransactionSHA256 == "" || digest != strings.ToLower(request.ExpectedTransactionSHA256) {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureRejected, ErrorDigestMismatch, OperationStart, false, err)
	}
	if transaction.State != browsertransaction.StateCandidate && transaction.State != browsertransaction.StateReviewed {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureConflict, ErrorInvalidState, OperationStart, false, nil)
	}
	engine.snapshot.Transaction = &transaction
	engine.snapshot.TransactionSHA256 = digest
	engine.snapshot.LastFailure = nil
	engine.sequence++
	if err := engine.refreshLocked(); err != nil {
		return Snapshot{}, err
	}
	return cloneSnapshot(engine.snapshot)
}

func (engine *Engine) Review(ctx context.Context, request ReviewRequest) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationReview, false, nil)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if err := engine.requireMutation(ctx, request.Authority, OperationReview, browsertransaction.StateCandidate); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	next := cloneTransaction(*engine.snapshot.Transaction)
	next.State = browsertransaction.StateReviewed
	return engine.commitTransitionLocked(next, OperationReview)
}

func (engine *Engine) Prepare(ctx context.Context, request PrepareRequest) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationPrepare, false, nil)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if err := engine.requireMutation(ctx, request.Authority, OperationPrepare, browsertransaction.StateReviewed); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	qualified, err := engine.config.operations.prepareAndQualify(ctx, engine.config.Package)
	if err != nil {
		if canceled(ctx, err) {
			return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureOperational, ErrorCanceled, OperationPrepare, true, err)
		}
		code := ErrorPreparationFailed
		wireCode := browsertransaction.FailurePreparationFailed
		if _, ok := packagepipeline.QualificationFailureCode(err); ok {
			code = ErrorQualificationFailed
			wireCode = browsertransaction.FailureQualificationFailed
		}
		return engine.commitFailedLocked(OperationPrepare, code, wireCode, err)
	}
	manifest, report := qualified.Prepared().Manifest(), qualified.Report()
	next := cloneTransaction(*engine.snapshot.Transaction)
	next.State = browsertransaction.StatePrepared
	evidence := preparationEvidence(manifest, report)
	next.Preparation = &browsertransaction.Preparation{PackageSHA256: evidence.PackageSHA256, QualificationSHA256: evidence.QualificationSHA256}
	result, transitionErr := engine.commitTransitionLocked(next, OperationPrepare)
	if transitionErr != nil {
		return result, transitionErr
	}
	engine.qualified = &qualified
	engine.snapshot.Preparation = evidence
	engine.snapshot.LastFailure = nil
	if err := engine.refreshLocked(); err != nil {
		return Snapshot{}, err
	}
	return cloneSnapshot(engine.snapshot)
}

func (engine *Engine) Promote(ctx context.Context, request PromoteRequest) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationPromote, false, nil)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if err := engine.requireMutation(ctx, request.Authority, OperationPromote, browsertransaction.StatePrepared); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	if engine.snapshot.Preparation == nil || request.ExpectedPreparationSHA256 != engine.snapshot.Preparation.PreparationSHA256 ||
		request.ExpectedQualificationSHA256 != engine.snapshot.Preparation.QualificationSHA256 {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureRejected, ErrorDigestMismatch, OperationPromote, false, nil)
	}
	qualified, err := engine.exactQualifiedLocked(ctx)
	if err != nil {
		if canceled(ctx, err) {
			return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureOperational, ErrorCanceled, OperationPromote, true, err)
		}
		code := ErrorPreparationFailed
		wireCode := browsertransaction.FailurePreparationFailed
		if _, ok := packagepipeline.QualificationFailureCode(err); ok {
			code, wireCode = ErrorQualificationFailed, browsertransaction.FailureQualificationFailed
		}
		var typed *Error
		if errors.As(err, &typed) && typed.Code == ErrorDigestMismatch {
			return engine.commitFailureLocked(OperationPromote, browsertransaction.FailureRejected, ErrorDigestMismatch, browsertransaction.FailureDigestMismatch, false, err)
		}
		return engine.commitFailedLocked(OperationPromote, code, wireCode, err)
	}
	baseline, err := engine.config.operations.inspectRecovery(ctx, engine.config.Package.StoreDir)
	if err != nil || baseline.Resolution != packagepipeline.RecoveryClean {
		if canceled(ctx, err) {
			return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureOperational, ErrorCanceled, OperationPromote, true, err)
		}
		if err == nil {
			engine.snapshot.Recovery = &RecoveryEvidence{Report: &baseline}
		}
		engine.snapshot.LastFailure = &OperationFailure{Class: browsertransaction.FailureIndeterminate, Code: ErrorRecoveryRequired, Operation: OperationPromote, Retryable: false, PromotionState: packagepipeline.PromotionRecoveryRequiredState, TargetGenerationSHA256: baseline.TargetGenerationSHA256}
		engine.sequence++
		_ = engine.refreshLocked()
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureIndeterminate, ErrorRecoveryRequired, OperationPromote, false, err)
	}
	promoted, err := engine.config.operations.promote(ctx, qualified, packagepipeline.PromotionOptions{StoreDir: engine.config.Package.StoreDir})
	if err != nil {
		return engine.handlePromotionFailureLocked(ctx, err)
	}
	selection := promoted.Selection()
	next := cloneTransaction(*engine.snapshot.Transaction)
	next.State = browsertransaction.StatePromoted
	next.Promotion = &browsertransaction.Promotion{GenerationSHA256: selection.SelectedGenerationSHA256}
	result, transitionErr := engine.commitTransitionLocked(next, OperationPromote)
	if transitionErr != nil {
		return result, transitionErr
	}
	engine.snapshot.Promotion = promotionEvidence(selection, baseline.ObservedSelectionSHA256)
	engine.snapshot.Recovery = nil
	engine.snapshot.LastFailure = nil
	engine.qualified = nil
	if err := engine.refreshLocked(); err != nil {
		return Snapshot{}, err
	}
	return cloneSnapshot(engine.snapshot)
}

func (engine *Engine) Cancel(ctx context.Context, request CancelRequest) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationCancel, false, nil)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if err := engine.requireMutation(ctx, request.Authority, OperationCancel, browsertransaction.StateCandidate, browsertransaction.StateReviewed, browsertransaction.StatePrepared, browsertransaction.StateIndeterminate); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	next := cloneTransaction(*engine.snapshot.Transaction)
	next.State, next.Promotion, next.Failure = browsertransaction.StateCancelled, nil, nil
	result, err := engine.commitTransitionLocked(next, OperationCancel)
	if err == nil {
		engine.qualified = nil
		engine.snapshot.LastFailure = nil
		_ = engine.refreshLocked()
		result = cloneOrZero(engine.snapshot)
	}
	return result, err
}

func (engine *Engine) InspectRecovery(ctx context.Context, request InspectRecoveryRequest) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationInspectRecovery, false, nil)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if err := engine.requireContext(ctx, OperationInspectRecovery); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	if err := engine.requireRevision(request.ExpectedRevision, OperationInspectRecovery); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	if !engine.inState(browsertransaction.StatePrepared, browsertransaction.StateIndeterminate) {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureConflict, ErrorInvalidState, OperationInspectRecovery, false, nil)
	}
	report, err := engine.config.operations.inspectRecovery(ctx, engine.config.Package.StoreDir)
	if err != nil {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureOperational, ErrorInspectionFailed, OperationInspectRecovery, true, err)
	}
	engine.snapshot.Recovery = &RecoveryEvidence{Report: &report}
	engine.sequence++
	if err := engine.refreshLocked(); err != nil {
		return Snapshot{}, err
	}
	return cloneSnapshot(engine.snapshot)
}

func (engine *Engine) Recover(ctx context.Context, request RecoverRequest) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationRecover, false, nil)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if err := engine.requireMutation(ctx, request.Authority, OperationRecover, browsertransaction.StateIndeterminate); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	if engine.snapshot.Recovery == nil || engine.snapshot.Recovery.Report == nil || request.ExpectedRecoverySHA256 == "" ||
		request.ExpectedRecoverySHA256 != engine.snapshot.Recovery.Report.RecoverySHA256 {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureConflict, ErrorRecoveryDrift, OperationRecover, false, nil)
	}
	report := *engine.snapshot.Recovery.Report
	reconciliation, err := engine.config.operations.reconcile(ctx, packagepipeline.ReconcileOptions{StoreDir: engine.config.Package.StoreDir, ExpectedRecoverySHA256: request.ExpectedRecoverySHA256})
	if err != nil {
		if canceled(ctx, err) {
			return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureOperational, ErrorCanceled, OperationRecover, true, err)
		}
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureIndeterminate, ErrorRecoveryDrift, OperationRecover, false, err)
	}
	next := cloneTransaction(*engine.snapshot.Transaction)
	next.Failure = nil
	switch reconciliation.Resolution {
	case packagepipeline.RecoveryRolledBack:
		next.State = browsertransaction.StatePrepared
	case packagepipeline.RecoveryPromoted:
		if reconciliation.SelectedGenerationSHA256 == "" || report.ObservedSelectionSHA256 == "" {
			return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureIndeterminate, ErrorRecoveryDrift, OperationRecover, false, nil)
		}
		next.State = browsertransaction.StatePromoted
		next.Promotion = &browsertransaction.Promotion{GenerationSHA256: reconciliation.SelectedGenerationSHA256}
	case packagepipeline.RecoveryClean:
		selected, selectedErr := engine.config.operations.readCurrent(ctx, engine.config.Package.StoreDir)
		target := ""
		if engine.snapshot.LastFailure != nil {
			target = engine.snapshot.LastFailure.TargetGenerationSHA256
		}
		if selectedErr == nil && target != "" && selected.Selection().SelectedGenerationSHA256 == target {
			next.State = browsertransaction.StatePromoted
			next.Promotion = &browsertransaction.Promotion{GenerationSHA256: target}
			report.ObservedSelectionSHA256 = selected.Selection().SelectionSHA256
			reconciliation.SelectedGenerationSHA256 = target
			reconciliation.PriorGenerationSHA256 = selected.Selection().PriorGenerationSHA256
		} else {
			next.State = browsertransaction.StatePrepared
		}
	default:
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureIndeterminate, ErrorRecoveryDrift, OperationRecover, false, nil)
	}
	result, transitionErr := engine.commitTransitionLocked(next, OperationRecover)
	if transitionErr != nil {
		return result, transitionErr
	}
	engine.snapshot.Recovery = &RecoveryEvidence{Report: &report, Reconciliation: &reconciliation}
	engine.snapshot.LastFailure = nil
	if next.State == browsertransaction.StatePromoted {
		selection := packagepipeline.Selection{
			SelectedGenerationSHA256: reconciliation.SelectedGenerationSHA256,
			PriorGenerationSHA256:    reconciliation.PriorGenerationSHA256,
			SelectionSHA256:          report.ObservedSelectionSHA256,
		}
		engine.snapshot.Promotion = promotionEvidence(selection, report.BaselineSelectionSHA256)
		engine.qualified = nil
	}
	if err := engine.refreshLocked(); err != nil {
		return Snapshot{}, err
	}
	return cloneSnapshot(engine.snapshot)
}

func (engine *Engine) InspectSelected(ctx context.Context, request InspectSelectedRequest) (Snapshot, error) {
	if engine == nil {
		return Snapshot{}, engineError(browsertransaction.FailureOperational, ErrorInactive, OperationInspectSelected, false, nil)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if err := engine.requireContext(ctx, OperationInspectSelected); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	if err := engine.requireRevision(request.ExpectedRevision, OperationInspectSelected); err != nil {
		return cloneOrZero(engine.snapshot), err
	}
	if !engine.inState(browsertransaction.StatePromoted) || engine.snapshot.Promotion == nil || request.ExpectedSelectionSHA256 != engine.snapshot.Promotion.SelectionSHA256 {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureConflict, ErrorDigestMismatch, OperationInspectSelected, false, nil)
	}
	inspection, err := engine.config.operations.inspectSelected(ctx, engine.config.Package.StoreDir, request.ExpectedSelectionSHA256)
	if err != nil {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureOperational, ErrorInspectionFailed, OperationInspectSelected, true, err)
	}
	engine.snapshot.Inspection = &inspection
	engine.sequence++
	if err := engine.refreshLocked(); err != nil {
		return Snapshot{}, err
	}
	return cloneSnapshot(engine.snapshot)
}

func (engine *Engine) exactQualifiedLocked(ctx context.Context) (packagepipeline.Qualified, error) {
	if engine.qualified != nil {
		manifest, report := engine.qualified.Prepared().Manifest(), engine.qualified.Report()
		if evidenceMatches(engine.snapshot.Preparation, manifest, report) {
			return *engine.qualified, nil
		}
	}
	qualified, err := engine.config.operations.prepareAndQualify(ctx, engine.config.Package)
	if err != nil {
		return packagepipeline.Qualified{}, err
	}
	manifest, report := qualified.Prepared().Manifest(), qualified.Report()
	if !evidenceMatches(engine.snapshot.Preparation, manifest, report) {
		return packagepipeline.Qualified{}, engineError(browsertransaction.FailureRejected, ErrorDigestMismatch, OperationPromote, false, nil)
	}
	engine.qualified = &qualified
	return qualified, nil
}

func (engine *Engine) handlePromotionFailureLocked(ctx context.Context, cause error) (Snapshot, error) {
	if canceled(ctx, cause) {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureOperational, ErrorCanceled, OperationPromote, true, cause)
	}
	state, hasState := packagepipeline.PromotionFailureState(cause)
	target, _ := packagepipeline.PromotionFailureGeneration(cause)
	if state == packagepipeline.PromotionIndeterminateState || state == packagepipeline.PromotionRecoveryRequiredState {
		next := cloneTransaction(*engine.snapshot.Transaction)
		next.State = browsertransaction.StateIndeterminate
		next.Failure = &browsertransaction.Failure{Class: browsertransaction.FailureIndeterminate, Code: browsertransaction.FailurePromotionIndeterminate}
		result, err := engine.commitTransitionLocked(next, OperationPromote)
		if err != nil {
			return result, err
		}
		if report, inspectErr := engine.config.operations.inspectRecovery(ctx, engine.config.Package.StoreDir); inspectErr == nil {
			engine.snapshot.Recovery = &RecoveryEvidence{Report: &report}
			if target == "" {
				target = report.TargetGenerationSHA256
			}
		}
		engine.snapshot.LastFailure = &OperationFailure{Class: browsertransaction.FailureIndeterminate, Code: ErrorPromotionIndeterminate, Operation: OperationPromote, Retryable: false, PromotionState: state, TargetGenerationSHA256: target}
		_ = engine.refreshLocked()
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureIndeterminate, ErrorPromotionIndeterminate, OperationPromote, false, cause)
	}
	if hasState && state == packagepipeline.PromotionRolledBackState {
		engine.snapshot.LastFailure = &OperationFailure{Class: browsertransaction.FailureOperational, Code: ErrorPromotionRolledBack, Operation: OperationPromote, Retryable: true, PromotionState: state, TargetGenerationSHA256: target}
		if report, inspectErr := engine.config.operations.inspectRecovery(ctx, engine.config.Package.StoreDir); inspectErr == nil {
			engine.snapshot.Recovery = &RecoveryEvidence{Report: &report}
		}
		engine.sequence++
		_ = engine.refreshLocked()
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureOperational, ErrorPromotionRolledBack, OperationPromote, true, cause)
	}
	return engine.commitFailedLocked(OperationPromote, ErrorPromotionFailed, browsertransaction.FailurePromotionFailed, cause)
}

func (engine *Engine) commitFailedLocked(operation Operation, code ErrorCode, wireCode browsertransaction.FailureCode, cause error) (Snapshot, error) {
	return engine.commitFailureLocked(operation, browsertransaction.FailureOperational, code, wireCode, false, cause)
}

func (engine *Engine) commitFailureLocked(operation Operation, class browsertransaction.FailureClass, code ErrorCode, wireCode browsertransaction.FailureCode, retryable bool, cause error) (Snapshot, error) {
	next := cloneTransaction(*engine.snapshot.Transaction)
	next.State = browsertransaction.StateFailed
	next.Failure = &browsertransaction.Failure{Class: class, Code: wireCode}
	result, err := engine.commitTransitionLocked(next, operation)
	if err != nil {
		return result, err
	}
	engine.snapshot.LastFailure = &OperationFailure{Class: class, Code: code, Operation: operation, Retryable: retryable}
	_ = engine.refreshLocked()
	return cloneOrZero(engine.snapshot), engineError(class, code, operation, retryable, cause)
}

func (engine *Engine) commitTransitionLocked(next browsertransaction.Transaction, operation Operation) (Snapshot, error) {
	previous := *engine.snapshot.Transaction
	if err := browsertransaction.ValidateTransition(previous, next); err != nil {
		return cloneOrZero(engine.snapshot), engineError(browsertransaction.FailureRejected, ErrorInvalidState, operation, false, err)
	}
	engine.snapshot.Transaction = &next
	engine.snapshot.TransactionSHA256, _ = browsertransaction.Digest(next)
	engine.snapshot.Inspection = nil
	engine.sequence++
	if err := engine.refreshLocked(); err != nil {
		return Snapshot{}, err
	}
	return cloneSnapshot(engine.snapshot)
}

func (engine *Engine) requireMutation(ctx context.Context, authority Authority, operation Operation, states ...browsertransaction.State) error {
	if err := engine.requireContext(ctx, operation); err != nil {
		return err
	}
	if err := engine.requireRevision(authority.ExpectedRevision, operation); err != nil {
		return err
	}
	if engine.snapshot.Transaction == nil {
		return engineError(browsertransaction.FailureConflict, ErrorInactive, operation, false, nil)
	}
	if !engine.inState(states...) {
		return engineError(browsertransaction.FailureConflict, ErrorInvalidState, operation, false, nil)
	}
	if authority.ExpectedTransactionSHA256 == "" || authority.ExpectedTransactionSHA256 != engine.snapshot.TransactionSHA256 {
		return engineError(browsertransaction.FailureRejected, ErrorDigestMismatch, operation, false, nil)
	}
	if !authority.HumanApproved {
		return engineError(browsertransaction.FailureRejected, ErrorAuthorizationRequired, operation, false, nil)
	}
	if operation == OperationReview || operation == OperationPrepare {
		expires, err := time.Parse(time.RFC3339Nano, engine.snapshot.Transaction.Provenance.ExpiresAt)
		if err != nil || !engine.config.Now().Before(expires) {
			return engineError(browsertransaction.FailureRejected, ErrorCandidateStale, operation, false, err)
		}
	}
	return nil
}

func (engine *Engine) requireRevision(expected string, operation Operation) error {
	if expected == "" || expected != engine.snapshot.Revision {
		return engineError(browsertransaction.FailureConflict, ErrorStaleRevision, operation, true, nil)
	}
	return nil
}

func (engine *Engine) requireContext(ctx context.Context, operation Operation) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return engineError(browsertransaction.FailureOperational, ErrorCanceled, operation, true, err)
	}
	return nil
}

func (engine *Engine) inState(states ...browsertransaction.State) bool {
	if engine.snapshot.Transaction == nil {
		return false
	}
	for _, state := range states {
		if engine.snapshot.Transaction.State == state {
			return true
		}
	}
	return false
}

func (engine *Engine) refreshLocked() error {
	engine.snapshot.Version = Version
	engine.snapshot.RuntimeExecutionSupported = false
	engine.snapshot.AllowedOperations = []Operation{OperationObserve}
	if engine.snapshot.Transaction == nil {
		engine.snapshot.AllowedOperations = append(engine.snapshot.AllowedOperations, OperationStart)
	} else {
		switch engine.snapshot.Transaction.State {
		case browsertransaction.StateCandidate:
			engine.snapshot.AllowedOperations = append(engine.snapshot.AllowedOperations, OperationReview, OperationCancel)
		case browsertransaction.StateReviewed:
			engine.snapshot.AllowedOperations = append(engine.snapshot.AllowedOperations, OperationPrepare, OperationCancel)
		case browsertransaction.StatePrepared:
			engine.snapshot.AllowedOperations = append(engine.snapshot.AllowedOperations, OperationPromote, OperationCancel, OperationInspectRecovery)
		case browsertransaction.StateIndeterminate:
			engine.snapshot.AllowedOperations = append(engine.snapshot.AllowedOperations, OperationInspectRecovery, OperationCancel)
			if engine.snapshot.Recovery != nil && engine.snapshot.Recovery.Report != nil && engine.snapshot.Recovery.Report.Resolution != packagepipeline.RecoveryDrift {
				engine.snapshot.AllowedOperations = append(engine.snapshot.AllowedOperations, OperationRecover)
			}
		case browsertransaction.StatePromoted:
			engine.snapshot.AllowedOperations = append(engine.snapshot.AllowedOperations, OperationInspectSelected)
		}
	}
	payload := engine.snapshot
	payload.Revision = ""
	data, err := json.Marshal(struct {
		Sequence uint64   `json:"sequence"`
		Snapshot Snapshot `json:"snapshot"`
	}{Sequence: engine.sequence, Snapshot: payload})
	if err != nil {
		return engineError(browsertransaction.FailureOperational, ErrorInvalidRequest, OperationObserve, false, err)
	}
	sum := sha256.Sum256(data)
	engine.snapshot.Revision = "sha256:" + hex.EncodeToString(sum[:])
	return nil
}

func preparationEvidence(manifest packagepipeline.Manifest, report packagepipeline.QualificationReport) *PreparationEvidence {
	return &PreparationEvidence{
		PreparationSHA256: taggedSHA256(manifest.ManifestSHA256), InputSHA256: taggedSHA256(manifest.InputSHA256),
		PackageSHA256: taggedSHA256(manifest.PackageSHA256), HandoffSHA256: taggedSHA256(manifest.HandoffSHA256),
		QualitySHA256: taggedSHA256(manifest.QualitySHA256), QualificationSHA256: taggedSHA256(report.QualificationSHA256),
	}
}

func promotionEvidence(selection packagepipeline.Selection, baseline string) *PromotionEvidence {
	return &PromotionEvidence{
		GenerationSHA256: selection.SelectedGenerationSHA256, SelectionSHA256: selection.SelectionSHA256,
		BaselineSelectionSHA256: baseline, SelectedGenerationSHA256: selection.SelectedGenerationSHA256,
		PriorGenerationSHA256: selection.PriorGenerationSHA256,
	}
}

func evidenceMatches(expected *PreparationEvidence, manifest packagepipeline.Manifest, report packagepipeline.QualificationReport) bool {
	actual := preparationEvidence(manifest, report)
	return expected != nil && *expected == *actual
}

func cloneTransaction(transaction browsertransaction.Transaction) browsertransaction.Transaction {
	transaction.Candidates = append(make([]browsertransaction.Candidate, 0, len(transaction.Candidates)), transaction.Candidates...)
	transaction.Provenance.Origins = append(make([]string, 0, len(transaction.Provenance.Origins)), transaction.Provenance.Origins...)
	transaction.CredentialBindings = append(make([]browsertransaction.CredentialBinding, 0, len(transaction.CredentialBindings)), transaction.CredentialBindings...)
	if transaction.Preparation != nil {
		value := *transaction.Preparation
		transaction.Preparation = &value
	}
	if transaction.Promotion != nil {
		value := *transaction.Promotion
		transaction.Promotion = &value
	}
	if transaction.Failure != nil {
		value := *transaction.Failure
		transaction.Failure = &value
	}
	return transaction
}

func cloneSnapshot(snapshot Snapshot) (Snapshot, error) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return Snapshot{}, err
	}
	var result Snapshot
	if err := json.Unmarshal(data, &result); err != nil {
		return Snapshot{}, err
	}
	return result, nil
}

func cloneOrZero(snapshot Snapshot) Snapshot {
	result, _ := cloneSnapshot(snapshot)
	return result
}

func engineError(class browsertransaction.FailureClass, code ErrorCode, operation Operation, retryable bool, cause error) error {
	return &Error{Class: class, Code: code, Operation: operation, Retryable: retryable, cause: cause}
}

func canceled(ctx context.Context, err error) bool {
	return (ctx != nil && ctx.Err() != nil) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func taggedSHA256(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) == sha256.Size*2 {
		return "sha256:" + value
	}
	return value
}
