package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
)

// RegistrationVirtualBrowserTransaction converts one immutable, path-free
// registration candidate into the generic virtual discovery input. When
// reviewed is true, the returned lifecycle snapshot records explicit
// acceptance; virtual discovery rejects registration candidates otherwise.
func RegistrationVirtualBrowserTransaction(candidate *browsercandidate.Registration, reviewed bool) (elicitor.VirtualBrowserTransactionInput, error) {
	if candidate == nil {
		return elicitor.VirtualBrowserTransactionInput{}, errors.New("registration candidate is required")
	}
	transaction := candidate.Transaction()
	if reviewed {
		var err error
		transaction, err = candidate.ReviewedTransaction()
		if err != nil {
			return elicitor.VirtualBrowserTransactionInput{}, err
		}
	}
	if transaction.Kind != browsertransaction.KindRegistration || len(transaction.Candidates) != 1 || transaction.Candidates[0].Kind != browsertransaction.CandidateRegistration {
		return elicitor.VirtualBrowserTransactionInput{}, errors.New("registration candidate transaction composition is invalid")
	}
	return elicitor.VirtualBrowserTransactionInput{
		Transaction: transaction,
		Sources: []elicitor.VirtualBrowserSourceInput{{
			Kind: browsertransaction.CandidateRegistration, Flow: candidate.Flow(), CleanupDisposition: candidate.CleanupDisposition(), Source: candidate.Source(), Review: candidate.Review(),
		}},
	}, nil
}

// AuthenticationCapabilityVirtualBrowserTransaction converts one immutable,
// path-free BAP+BCP candidate into the generic virtual discovery input. When
// reviewed is true, the returned lifecycle snapshot records the caller's
// explicit acceptance of the exact candidate and review digests.
func AuthenticationCapabilityVirtualBrowserTransaction(candidate *browsercandidate.AuthenticationCapability, reviewed bool) (elicitor.VirtualBrowserTransactionInput, error) {
	if candidate == nil {
		return elicitor.VirtualBrowserTransactionInput{}, errors.New("authentication-capability candidate is required")
	}
	transaction := candidate.Transaction()
	if reviewed {
		var err error
		transaction, err = candidate.ReviewedTransaction()
		if err != nil {
			return elicitor.VirtualBrowserTransactionInput{}, err
		}
	}
	if transaction.Kind != browsertransaction.KindAuthenticationCapability || len(transaction.Candidates) != 2 ||
		transaction.Candidates[0].Kind != browsertransaction.CandidateAuthentication || transaction.Candidates[1].Kind != browsertransaction.CandidateCapability {
		return elicitor.VirtualBrowserTransactionInput{}, errors.New("authentication-capability transaction composition is invalid")
	}
	return elicitor.VirtualBrowserTransactionInput{
		Transaction: transaction,
		Sources: []elicitor.VirtualBrowserSourceInput{
			{Kind: browsertransaction.CandidateAuthentication, Flow: candidate.Flow(), Source: candidate.Authentication(), Review: candidate.AuthenticationReview()},
			{Kind: browsertransaction.CandidateCapability, Source: candidate.Capability(), Review: candidate.CapabilityReview()},
		},
	}, nil
}

// ReplaceVirtualBrowserSources atomically replaces the in-memory catalog when
// expectedGeneration matches. It performs no workspace write and rejects a
// replacement that would make an already selected virtual source stale.
func (e *Engine) ReplaceVirtualBrowserSources(ctx context.Context, expectedGeneration uint64, inputs []elicitor.VirtualBrowserTransactionInput) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, operational(errors.New("engine is nil"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return Snapshot{}, err
	}
	if expectedGeneration != e.virtualGeneration {
		return Snapshot{}, conflict("virtual_sources_stale", fmt.Errorf("virtual browser catalog generation is %d, not %d", e.virtualGeneration, expectedGeneration))
	}
	if e.virtualGeneration == ^uint64(0) {
		return Snapshot{}, operational(errors.New("virtual browser catalog generation is exhausted"))
	}
	nextGeneration := e.virtualGeneration + 1
	virtual, err := elicitor.DiscoverVirtualBrowserSources(inputs, e.now())
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	if err := elicitor.RequireFreshVirtualBrowserSources(e.session.SourcePlan, virtual.Plans); err != nil {
		return Snapshot{}, conflict("virtual_sources_selected", err)
	}
	refreshed, err := e.buildRefreshedStateWithVirtualLocked(ctx, e.session, virtual, nextGeneration)
	if err != nil {
		return Snapshot{}, classifyRefreshFailure(err)
	}
	prospective, err := e.snapshotForStateLocked(ctx, refreshed, nil, nil)
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return Snapshot{}, err
	}
	paths := watchedPaths(e.workspaceRoot, prospective)
	baseline, err := captureWorkspace(ctx, e.workspaceRoot, paths)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	cached, err := cloneSnapshot(prospective)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	result, err := cloneSnapshot(prospective)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	e.applyRefreshedStateLocked(refreshed)
	e.cachedSnapshot = cached
	e.watchedPaths = paths
	e.workspaceBaseline = baseline
	return result, nil
}

// SelectVirtualBrowserSources replaces the selected virtual subset with the
// deterministic dependency closure of candidateIDs. Only the resumable,
// value-free session metadata is persisted; source bytes remain in memory
// until the ordinary explicit authoring approval boundary.
func (e *Engine) SelectVirtualBrowserSources(ctx context.Context, expectedGeneration uint64, candidateIDs []string) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, operational(errors.New("engine is nil"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return Snapshot{}, err
	}
	if expectedGeneration != e.virtualGeneration {
		return Snapshot{}, conflict("virtual_sources_stale", fmt.Errorf("virtual browser catalog generation is %d, not %d", e.virtualGeneration, expectedGeneration))
	}
	workspaceAtStart, err := e.observeMutationWorkspaceLocked(ctx)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	next, err := elicitor.SelectVirtualBrowserSources(e.session, e.virtual, candidateIDs)
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	return e.persistSessionMutationLocked(ctx, next, workspaceAtStart, nil)
}
