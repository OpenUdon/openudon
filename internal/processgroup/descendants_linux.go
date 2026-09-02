//go:build linux

package processgroup

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// descendantTracker records Linux PID/start-time identities while the group
// leader is alive. That lets cleanup follow children that later call setpgid
// or setsid without risking a signal to an unrelated process after PID reuse.
type descendantTracker struct {
	rootPID int

	mu          sync.Mutex
	terminateMu sync.Mutex
	known       map[int]uint64
	healthy     bool
	stop        chan struct{}
	done        chan struct{}
	once        sync.Once
}

type procIdentity struct {
	pid       int
	parentPID int
	groupPID  int
	state     byte
	startTime uint64
}

func startDescendantTracker(rootPID int) *descendantTracker {
	tracker := &descendantTracker{
		rootPID: rootPID,
		known:   make(map[int]uint64),
		healthy: true,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	tracker.observe()
	go tracker.monitor()
	return tracker
}

func (tracker *descendantTracker) monitor() {
	defer close(tracker.done)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tracker.observe()
		case <-tracker.stop:
			return
		}
	}
}

func (tracker *descendantTracker) stopMonitoring() {
	tracker.once.Do(func() { close(tracker.stop) })
	<-tracker.done
}

func (tracker *descendantTracker) observe() {
	// Read the kernel's direct-child lists first. This is substantially cheaper
	// than a complete /proc scan and closes the scheduling window for a child
	// that detaches and whose leader exits quickly under host load.
	tracker.observeChildLists()

	processes := readProcIdentities()
	if processes == nil {
		tracker.mu.Lock()
		tracker.healthy = false
		tracker.mu.Unlock()
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if root, ok := processes[tracker.rootPID]; ok {
		recordStableDescendantIdentity(tracker.known, tracker.rootPID, root.startTime)
	}
	// Repeat because /proc ordering does not guarantee parents precede children.
	for changed := true; changed; {
		changed = false
		for pid, process := range processes {
			if recorded, exists := tracker.known[pid]; exists {
				if recorded == process.startTime {
					continue
				}
				// A recorded PID is one immutable PID/start-time identity. Never
				// adopt a later process that reused the numeric PID.
				continue
			}
			_, parentKnown := tracker.known[process.parentPID]
			inOriginalGroup := process.groupPID == tracker.rootPID
			if !parentKnown && !inOriginalGroup {
				continue
			}
			if parentKnown {
				if parent, exists := processes[process.parentPID]; !exists || tracker.known[process.parentPID] != parent.startTime {
					continue
				}
			}
			recordStableDescendantIdentity(tracker.known, pid, process.startTime)
			changed = true
		}
	}
}

func (tracker *descendantTracker) observeChildLists() {
	tracker.mu.Lock()
	seeds := make(map[int]uint64, len(tracker.known)+1)
	for pid, startTime := range tracker.known {
		seeds[pid] = startTime
	}
	tracker.mu.Unlock()
	if _, ok := seeds[tracker.rootPID]; !ok {
		if root, err := readProcIdentity(tracker.rootPID); err == nil {
			seeds[tracker.rootPID] = root.startTime
		}
	}

	queue := make([]int, 0, len(seeds))
	for pid := range seeds {
		queue = append(queue, pid)
	}
	seen := make(map[int]bool, len(queue))
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		process, err := readProcIdentity(pid)
		if err != nil || seeds[pid] != 0 && seeds[pid] != process.startTime {
			continue
		}
		tracker.mu.Lock()
		accepted := recordStableDescendantIdentity(tracker.known, pid, process.startTime)
		tracker.mu.Unlock()
		if !accepted {
			continue
		}

		data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "task", strconv.Itoa(pid), "children"))
		if err != nil {
			continue
		}
		for _, field := range strings.Fields(string(data)) {
			childPID, err := strconv.Atoi(field)
			if err != nil || seen[childPID] {
				continue
			}
			child, err := readProcIdentity(childPID)
			if err != nil || child.parentPID != pid {
				continue
			}
			seeds[childPID] = child.startTime
			queue = append(queue, childPID)
		}
	}
}

func recordStableDescendantIdentity(known map[int]uint64, pid int, startTime uint64) bool {
	if recorded, exists := known[pid]; exists && recorded != startTime {
		return false
	}
	known[pid] = startTime
	return true
}

// killRootGroupIfLive uses the process group only while its leader's immutable
// identity is still present. After Wait reaps the leader, numeric PGID reuse
// makes group signaling unsafe and cleanup must remain identity-only.
func (tracker *descendantTracker) killRootGroupIfLive() {
	tracker.mu.Lock()
	startTime, ok := tracker.known[tracker.rootPID]
	tracker.mu.Unlock()
	if !ok {
		return
	}
	process, err := readProcIdentity(tracker.rootPID)
	if err != nil || process.startTime != startTime || process.state == 'Z' || process.groupPID != tracker.rootPID {
		return
	}
	_ = syscall.Kill(-tracker.rootPID, syscall.SIGKILL)
}

func (tracker *descendantTracker) terminateAndVerify(timeout time.Duration) error {
	tracker.terminateMu.Lock()
	defer tracker.terminateMu.Unlock()
	tracker.stopMonitoring()
	tracker.observe()

	deadline := time.Now().Add(timeout)
	for {
		remaining, err := tracker.liveIdentities()
		if err != nil {
			return ErrTerminationTimeout
		}
		if len(remaining) == 0 {
			return nil
		}
		for _, process := range remaining {
			_ = syscall.Kill(process.pid, syscall.SIGKILL)
		}
		if time.Now().After(deadline) {
			return ErrTerminationTimeout
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (tracker *descendantTracker) liveIdentities() ([]procIdentity, error) {
	processes := readProcIdentities()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if processes == nil || !tracker.healthy {
		return nil, ErrTerminationTimeout
	}
	live := make([]procIdentity, 0, len(tracker.known))
	for pid, startTime := range tracker.known {
		process, exists := processes[pid]
		if !exists || process.startTime != startTime || process.state == 'Z' {
			continue
		}
		live = append(live, process)
	}
	return live, nil
}

func readProcIdentities() map[int]procIdentity {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	processes := make(map[int]procIdentity, len(entries))
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		process, err := readProcIdentity(pid)
		if err == nil {
			processes[pid] = process
		}
	}
	return processes
}

func readProcIdentity(pid int) (procIdentity, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return procIdentity{}, err
	}
	return parseProcIdentity(pid, string(data))
}

func parseProcIdentity(pid int, stat string) (procIdentity, error) {
	closing := strings.LastIndexByte(stat, ')')
	if closing < 0 || closing+2 >= len(stat) {
		return procIdentity{}, errors.New("invalid proc stat")
	}
	fields := strings.Fields(stat[closing+2:])
	// fields starts at proc(5) field 3 (state); starttime is field 22.
	if len(fields) < 20 || len(fields[0]) != 1 {
		return procIdentity{}, errors.New("incomplete proc stat")
	}
	parentPID, err := strconv.Atoi(fields[1])
	if err != nil {
		return procIdentity{}, err
	}
	groupPID, err := strconv.Atoi(fields[2])
	if err != nil {
		return procIdentity{}, err
	}
	startTime, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return procIdentity{}, err
	}
	return procIdentity{pid: pid, parentPID: parentPID, groupPID: groupPID, state: fields[0][0], startTime: startTime}, nil
}
