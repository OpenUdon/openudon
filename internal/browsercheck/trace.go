// Package browsercheck contains development-only caching and diagnostic timing.
// It does not grant runtime authority or retain browser state.
package browsercheck

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type traceKey struct{}
type stageKey struct{}
type Trace struct {
	mu   sync.Mutex
	file *os.File
	err  error
}

func TraceFile(ctx context.Context, path string) (context.Context, *Trace, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return ctx, nil, errors.New("timing_output")
	}
	t := &Trace{file: f}
	return context.WithValue(ctx, traceKey{}, t), t, nil
}
func Stage(ctx context.Context, stage string) context.Context {
	return context.WithValue(ctx, stageKey{}, stage)
}
func Span(ctx context.Context, phase string) func(error) {
	t, _ := ctx.Value(traceKey{}).(*Trace)
	if t == nil {
		return func(error) {}
	}
	started := time.Now()
	return func(err error) {
		stage, _ := ctx.Value(stageKey{}).(string)
		status := "pass"
		if err != nil {
			status = "fail"
		}
		row := struct {
			Version      string `json:"version"`
			Stage        string `json:"stage"`
			Phase        string `json:"phase"`
			Started      string `json:"started_at"`
			Milliseconds int64  `json:"duration_ms"`
			Status       string `json:"status"`
		}{"openudon.browser-check-timing.v1", stage, phase, started.UTC().Format(time.RFC3339Nano), time.Since(started).Milliseconds(), status}
		t.mu.Lock()
		defer t.mu.Unlock()
		if e := json.NewEncoder(t.file).Encode(row); e != nil {
			t.err = errors.New("timing_output")
		}
	}
}
func (t *Trace) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return errors.Join(t.err, t.file.Sync(), t.file.Close())
}
