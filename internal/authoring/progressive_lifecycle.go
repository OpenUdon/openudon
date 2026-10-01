package authoring

import (
	"context"
	"io"
	"strings"

	sharedengine "github.com/OpenUdon/authoring/engine"
)

// ProgressiveLifecycleOptions adds draft/transcript lifecycle behavior
// around RunProgressiveICOT.
type ProgressiveLifecycleOptions[S, D, A any] struct {
	ExampleDir           string
	DraftPath            string
	TranscriptPath       string
	TranscriptVersion    string
	DeleteDraftOnSuccess bool
	Normalize            func(*S)
	LooksLikeSession     func(S) bool
	Opening              func(S) string
	TranscriptSession    func(A) any
}

// RunProgressiveWithLifecycle binds caller-specific progressive iCoT hooks to
// the local draft, autosave, transcript, and cleanup lifecycle.
func RunProgressiveWithLifecycle[S, D, A any](ctx context.Context, in io.Reader, out io.Writer, hooks ProgressiveLoopHooks[S, D, A], opts ProgressiveLifecycleOptions[S, D, A]) (A, error) {
	draftPath := strings.TrimSpace(opts.DraftPath)
	if draftPath == "" && strings.TrimSpace(opts.ExampleDir) != "" {
		draftPath = DraftPath(opts.ExampleDir)
	}
	return sharedengine.RunInteractiveWithLifecycle(ctx, in, out, hooks, sharedengine.InteractiveLifecycleOptions[S, D, A]{
		DraftPath:            draftPath,
		TranscriptPath:       opts.TranscriptPath,
		TranscriptVersion:    firstNonEmpty(opts.TranscriptVersion, "openudon.icot-transcript.v2"),
		DeleteDraftOnSuccess: opts.DeleteDraftOnSuccess,
		Normalize:            opts.Normalize,
		LooksLikeSession:     opts.LooksLikeSession,
		Opening:              opts.Opening,
		TranscriptSession:    opts.TranscriptSession,
		LoadDraft:            LoadDraft[S],
		SaveDraft:            SaveDraft[S],
		DeleteDraft:          DeleteDraft,
	})
}
