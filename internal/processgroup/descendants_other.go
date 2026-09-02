//go:build !linux

package processgroup

import "time"

// Non-Linux Unix uses process-group signaling and Windows uses taskkill /T.
// Neither platform exposes the Linux /proc PID/start-time identity contract.
type descendantTracker struct{}

func startDescendantTracker(int) *descendantTracker { return &descendantTracker{} }

func (*descendantTracker) killRootGroupIfLive() {}

func (*descendantTracker) terminateAndVerify(time.Duration) error { return nil }
