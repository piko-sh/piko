// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package resource

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"piko.sh/safeconv"
	"piko.sh/safedisk"
)

const (
	// dirEntryLimit bounds the entries DirBytes visits before giving up.
	dirEntryLimit = 100_000

	// cgroupCPUFile is the cgroup file reporting CPU use and throttling.
	cgroupCPUFile = "cpu.stat"
)

var (
	// errDirTooLarge reports a directory scan that exceeded dirEntryLimit.
	errDirTooLarge = errors.New("measurement exceeds the directory entry limit")
)

// PhaseCost records measured work for the whole verified cgroup.
//
// Disk counts logical file bytes left in the named private work directory at phase exit.
// This snapshot excludes shared build caches and cannot see deleted temporaries.
type PhaseCost struct {
	// CPUUsec records CPU use in microseconds.
	CPUUsec uint64

	// ThrottledUsec records CPU throttling time in microseconds.
	ThrottledUsec uint64

	// EndWorkDirBytes is the logical size of files left at phase end.
	EndWorkDirBytes uint64

	// CPUKnown reports whether CPU accounting succeeded.
	CPUKnown bool

	// WorkDirKnown reports whether directory-size accounting succeeded.
	WorkDirKnown bool
}

// CPUUsage returns a fresh aggregate cgroup usage counter in microseconds.
//
// Takes group (string) which is the cgroup accounting directory.
//
// Returns uint64 which is current cgroup CPU use in microseconds.
// Returns bool which reports whether the counter was read.
func CPUUsage(group string) (uint64, bool) {
	usage, _, ok := cpuTimes(group)
	return usage, ok
}

// cpuTimes reads CPU use and throttling time from cgroup accounting.
//
// Takes group (string) which is the cgroup accounting directory.
//
// Returns usage (uint64) which is CPU use in microseconds.
// Returns throttled (uint64) which is CPU throttling time in microseconds.
// Returns ok (bool) which reports whether the usage counter was read without an
// accounting error.
func cpuTimes(group string) (usage uint64, throttled uint64, ok bool) {
	if group == "" {
		return 0, 0, false
	}
	data, err := readCgroupFile(group, cgroupCPUFile)
	if err != nil {
		return 0, 0, false
	}
	seen := false
	for line := range strings.Lines(string(data)) {
		key, raw, found := strings.Cut(strings.TrimRight(line, "\n"), " ")
		if !found || (key != "usage_usec" && key != "throttled_usec") {
			continue
		}
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return 0, 0, false
		}
		if key == "usage_usec" {
			usage, seen = value, true
		} else {
			throttled = value
		}
	}
	return usage, throttled, seen
}

// DirBytes counts logical retained bytes without following symbolic links.
//
// An unreasonably large tree reports an unknown measurement instead of holding up
// shutdown. The independent launcher still enforces its full disk budget.
//
// Takes directory (string) which is the directory whose logical file bytes are measured.
//
// Returns uint64 which is the scanned logical file size in bytes, possibly partial when
// the scan fails.
// Returns bool which reports whether the bounded scan completed.
func DirBytes(directory string) (uint64, bool) {
	if directory == "" {
		return 0, false
	}
	var total uint64
	entries := 0
	err := filepath.WalkDir(directory, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries++
		if entries > dirEntryLimit {
			return fmt.Errorf("%w of %d", errDirTooLarge, dirEntryLimit)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += safeconv.Int64ToUint64(info.Size())
		return nil
	})
	return total, err == nil
}

// MeasureCheck preserves the operation's original error even when a measurement is
// unavailable.
//
// Takes group (string) which is the cgroup accounting directory.
// Takes work (string) which is the invocation-private working directory.
// Takes action (func() error) which performs the selected operation.
//
// Returns PhaseMemory which records sampled cgroup memory before, during and after the
// check.
// Returns PhaseCost which holds CPU use, throttling and final work-directory size
// readings.
// Returns error which is the measured operation's error, independent of sampling
// failures.
func MeasureCheck(group, work string, action func() error) (PhaseMemory, PhaseCost, error) {
	start, throttleStart, startOK := cpuTimes(group)
	memory, err := MeasurePhase(group, action)
	end, throttleEnd, endOK := cpuTimes(group)
	cost := PhaseCost{}
	if startOK && endOK && end >= start && throttleEnd >= throttleStart {
		cost.CPUKnown = true
		cost.CPUUsec = end - start
		cost.ThrottledUsec = throttleEnd - throttleStart
	}
	cost.EndWorkDirBytes, cost.WorkDirKnown = DirBytes(work)
	return memory, cost, err
}

// readCgroupFile reads one accounting file from a cgroup directory through a read-only
// sandbox rooted at that directory.
//
// Takes group (string) which is the cgroup accounting directory.
// Takes name (string) which is the accounting file within it.
//
// Returns []byte which is the file content.
// Returns error which reports a sandbox, read or close failure.
func readCgroupFile(group, name string) ([]byte, error) {
	sandbox, err := safedisk.NewSandbox(group, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("open cgroup %s: %w", group, err)
	}
	data, readErr := sandbox.ReadFile(name)
	if closeErr := sandbox.Close(); closeErr != nil {
		readErr = errors.Join(readErr, fmt.Errorf("close cgroup %s: %w", group, closeErr))
	}
	if readErr != nil {
		return nil, readErr
	}
	return data, nil
}
