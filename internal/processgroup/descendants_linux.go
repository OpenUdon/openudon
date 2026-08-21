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

	mu    sync.Mutex
	known map[int]uint64
	stop  chan struct{}
	done  chan struct{}
	once  sync.Once
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
	processes := readProcIdentities()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if root, ok := processes[tracker.rootPID]; ok {
		if recorded, exists := tracker.known[tracker.rootPID]; !exists || recorded == root.startTime {
			tracker.known[tracker.rootPID] = root.startTime
		}
	}
	// Repeat because /proc ordering does not guarantee parents precede children.
	for changed := true; changed; {
		changed = false
		for pid, process := range processes {
			if recorded, exists := tracker.known[pid]; exists && recorded == process.startTime {
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
			tracker.known[pid] = process.startTime
			changed = true
		}
	}
}

func (tracker *descendantTracker) terminateAndVerify(timeout time.Duration) error {
	tracker.stopMonitoring()
	tracker.observe()

	deadline := time.Now().Add(timeout)
	for {
		remaining := tracker.liveIdentities()
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

func (tracker *descendantTracker) liveIdentities() []procIdentity {
	processes := readProcIdentities()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	live := make([]procIdentity, 0, len(tracker.known))
	for pid, startTime := range tracker.known {
		process, exists := processes[pid]
		if !exists || process.startTime != startTime || process.state == 'Z' {
			continue
		}
		live = append(live, process)
	}
	return live
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
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		process, err := parseProcIdentity(pid, string(data))
		if err == nil {
			processes[pid] = process
		}
	}
	return processes
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
