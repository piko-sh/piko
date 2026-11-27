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

//go:build linux

package resource

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
	"piko.sh/safeconv"
)

const (
	// cgroupRoot is the cgroup v2 mount point.
	cgroupRoot = "/sys/fs/cgroup"

	// procSelfCgroup lists this process's cgroup memberships.
	procSelfCgroup = "/proc/self/cgroup"

	// procMeminfo reports host memory.
	procMeminfo = "/proc/meminfo"

	// hostPressureFile reports host-wide memory pressure.
	hostPressureFile = "/proc/pressure/memory"

	// cgroupUnlimited is the cgroup value meaning no limit.
	cgroupUnlimited = "max"

	// cgroupMemoryMaxFile is the cgroup file holding the memory ceiling.
	cgroupMemoryMaxFile = "memory.max"

	// cgroupCPUMaxFile is the cgroup file holding the CPU quota and period.
	cgroupCPUMaxFile = "cpu.max"

	// memoryHighNumerator is the numerator of the memory.high fraction.
	memoryHighNumerator = 4

	// memoryHighDenominator is the denominator of the memory.high fraction.
	memoryHighDenominator = 5

	// kibibyteShift converts /proc/meminfo kibibytes to bytes.
	kibibyteShift = 10

	// cpuSetWordBits is the number of CPUs each CPUSet word holds.
	cpuSetWordBits = 64
)

var (
	// errNoCgroup reports a process outside the cgroup v2 mount.
	errNoCgroup = errors.New("cannot locate this process in the cgroup v2 mount")

	// errMemoryCapUnverified reports a missing or loose memory, swap or task ceiling.
	errMemoryCapUnverified = errors.New("memory.max, memory.high, swap or pids ceiling unverified")

	// errMissingPressure reports a PSI file without a "some" avg10 value.
	errMissingPressure = errors.New("missing PSI 'some' avg10")
)

// ancestorLimits collects the limits set on a cgroup and its ancestors.
type ancestorLimits struct {
	// memory holds memory.max values.
	memory []uint64

	// high holds memory.high values.
	high []uint64

	// swap holds memory.swap.max values.
	swap []uint64

	// tasks holds pids.max values.
	tasks []uint64

	// quotas holds cpu.max quotas in cores.
	quotas []float64

	// headroom holds the remaining allowance under each memory.max.
	headroom []uint64
}

// readKernel verifies the process tree's cgroup limits before sampling resource use.
//
// Takes limits (kernelLimits) which holds the launcher's requested ceilings.
// Takes scan (bool) which requests a state-tree disk measurement.
//
// Returns Sample which holds the verified reading.
// Returns error which reports unverified limits, failed accounting reads or cancellation.
func readKernel(ctx context.Context, limits kernelLimits, scan bool) (Sample, error) {
	group, err := OwnCgroup()
	if err != nil {
		return Sample{}, err
	}
	found, err := collectAncestorLimits(group)
	if err != nil {
		return Sample{}, err
	}
	affinity, err := Affinity()
	if err != nil {
		return Sample{}, err
	}
	if err := verifyLimits(found, affinity, limits); err != nil {
		return Sample{}, err
	}

	var sample Sample
	sample.Cgroup, sample.CPUs = group, affinity
	sample.QuotaCPUs, sample.Headroom = slices.Min(found.quotas), slices.Min(found.headroom)
	sample.MemoryMax, sample.MemoryHigh = slices.Min(found.memory), slices.Min(found.high)
	sample.SwapMax, sample.TasksMax = slices.Min(found.swap), slices.Min(found.tasks)
	if err := readUsage(group, &sample); err != nil {
		return Sample{}, err
	}
	if scan {
		if sample.DiskUsed, err = DiskUsed(ctx, limits.state); err != nil {
			return Sample{}, err
		}
	}
	if sample.DiskFree, err = DiskFree(limits.state); err != nil {
		return Sample{}, err
	}
	return sample, nil
}

// readUsage fills in a cgroup's current memory, events, CPU time and pressure, and the
// host's available memory.
//
// Takes group (string) which is the cgroup directory.
// Takes sample (*Sample) which receives the readings.
//
// Returns error which reports unreadable or malformed accounting files.
func readUsage(group string, sample *Sample) error {
	var err error
	if sample.MemoryCurrent, _, err = cgroupLimit(group, cgroupMemoryFile); err != nil {
		return err
	}
	if sample.MemoryPeak, _, err = cgroupLimit(group, "memory.peak"); err != nil {
		return err
	}
	events, err := cgroupKeyed(group, "memory.events")
	if err != nil {
		return err
	}
	sample.OOMKills, sample.HighEvents = events["oom_kill"], events["high"]
	cpu, err := cgroupKeyed(group, "cpu.stat")
	if err != nil {
		return err
	}
	sample.CPUUsec, sample.ThrottledUsec = cpu["usage_usec"], cpu["throttled_usec"]
	if sample.Available, err = MemInfo("MemAvailable"); err != nil {
		return err
	}
	sample.PSI, err = pressure(group)
	return err
}

// verifyLimits checks that effective CPU, memory, swap and task limits are enforced.
//
// The memory.high check allows the kernel to round to the next page boundary.
//
// Takes found (ancestorLimits) which holds the limits on the cgroup and its ancestors.
// Takes affinity ([]int) which lists the allowed CPUs.
// Takes limits (kernelLimits) which holds the requested ceilings.
//
// Returns error which reports a ceiling the kernel does not enforce.
func verifyLimits(found ancestorLimits, affinity []int, limits kernelLimits) error {
	if len(found.quotas) == 0 || slices.Min(found.quotas) > float64(limits.cpus) ||
		len(affinity) == 0 || safeconv.IntToUint64(len(affinity)) > limits.cpus {
		quota := cgroupUnlimited
		if len(found.quotas) > 0 {
			quota = strconv.FormatFloat(slices.Min(found.quotas), 'f', -1, 64)
		}
		return fmt.Errorf("CPU cap unverified (quota=%s, affinity=%v)", quota, affinity)
	}
	highCeiling := limits.memory*memoryHighNumerator/memoryHighDenominator + safeconv.IntToUint64(os.Getpagesize()) - 1
	switch {
	case len(found.memory) == 0 || slices.Min(found.memory) == 0 || slices.Min(found.memory) > limits.memory,
		len(found.high) == 0 || slices.Min(found.high) == 0 || slices.Min(found.high) > highCeiling,
		len(found.swap) == 0 || slices.Min(found.swap) != 0,
		len(found.tasks) == 0 || slices.Min(found.tasks) > TasksMax:
		return errMemoryCapUnverified
	}
	return nil
}

// OwnCgroup locates this process's cgroup directory in the cgroup v2 mount.
//
// A private cgroup namespace can expose its root as "/".
//
// Returns string which is the absolute cgroup directory.
// Returns error which reports an unreadable membership file or missing cgroup.
func OwnCgroup() (string, error) {
	data, err := os.ReadFile(procSelfCgroup)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", procSelfCgroup, err)
	}
	for line := range strings.Lines(string(data)) {
		relative, ok := strings.CutPrefix(strings.TrimSpace(line), "0::/")
		if !ok {
			continue
		}
		if slices.Contains(strings.Split(relative, "/"), "..") {
			break
		}
		group := filepath.Join(cgroupRoot, relative)
		if info, err := os.Stat(filepath.Join(group, "cgroup.procs")); err == nil && info.Mode().IsRegular() {
			return group, nil
		}
	}
	return "", errNoCgroup
}

// EnclosingLimit finds the nearest cgroup, from this process's own upwards, that already
// caps memory or CPU.
//
// A new user scope may move the process sideways in the hierarchy, so the launcher
// records this group and the supervisor checks it still runs beneath it.
//
// Returns string which is the limiting cgroup, or empty when none limits the process.
// Returns error which reports unreadable cgroup files.
func EnclosingLimit() (string, error) {
	group, err := OwnCgroup()
	if err != nil {
		return "", err
	}
	for ancestor := range ancestors(group) {
		_, capped, err := cgroupLimit(ancestor, cgroupMemoryMaxFile)
		if err != nil {
			return "", err
		}
		_, quoted, err := cgroupQuota(ancestor)
		if err != nil {
			return "", err
		}
		if capped || quoted {
			return ancestor, nil
		}
	}
	return "", nil
}

// collectAncestorLimits reads the limits on a cgroup and every ancestor.
//
// Takes group (string) which is the starting cgroup directory.
//
// Returns ancestorLimits which holds the limits found.
// Returns error which reports unreadable cgroup files.
func collectAncestorLimits(group string) (ancestorLimits, error) {
	var found ancestorLimits
	for ancestor := range ancestors(group) {
		if err := found.add(ancestor); err != nil {
			return found, err
		}
	}
	return found, nil
}

// add records the limits set on one cgroup.
//
// Takes group (string) which is the cgroup directory.
//
// Returns error which reports unreadable cgroup files.
func (found *ancestorLimits) add(group string) error {
	capBytes, capped, err := cgroupLimit(group, cgroupMemoryMaxFile)
	if err != nil {
		return err
	}
	if capped {
		current, _, err := cgroupLimit(group, cgroupMemoryFile)
		if err != nil {
			return err
		}
		found.memory = append(found.memory, capBytes)
		found.headroom = append(found.headroom, capBytes-min(capBytes, current))
	}
	for name, values := range map[string]*[]uint64{"memory.high": &found.high, "memory.swap.max": &found.swap, "pids.max": &found.tasks} {
		value, limited, err := cgroupLimit(group, name)
		if err != nil {
			return err
		}
		if limited {
			*values = append(*values, value)
		}
	}
	quota, quoted, err := cgroupQuota(group)
	if quoted {
		found.quotas = append(found.quotas, quota)
	}
	return err
}

// ancestors yields a cgroup directory and each parent up to the cgroup root.
//
// Takes group (string) which is the starting cgroup directory.
//
// Returns func(yield func(string) bool) which iterates the directories.
func ancestors(group string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for current := group; ; current = filepath.Dir(current) {
			if !yield(current) || current == cgroupRoot || current == filepath.Dir(current) {
				return
			}
		}
	}
}

// readOptionalCgroupFile reads a cgroup file that may be absent.
//
// Takes group (string) which is the cgroup directory.
// Takes name (string) which is the file name.
//
// Returns []byte which holds the contents, or nil when the file is absent.
// Returns error which reports an unreadable file.
func readOptionalCgroupFile(group, name string) ([]byte, error) {
	data, err := readCgroupFile(group, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filepath.Join(group, name), err)
	}
	return data, nil
}

// cgroupLimit reads a single-value cgroup file.
//
// Takes group (string) which is the cgroup directory.
// Takes name (string) which is the file name.
//
// Returns uint64 which is the value, or zero when unlimited or absent.
// Returns bool which is true when the file holds a number.
// Returns error which reports an unreadable or malformed file.
func cgroupLimit(group, name string) (uint64, bool, error) {
	data, err := readOptionalCgroupFile(group, name)
	if err != nil || data == nil {
		return 0, false, err
	}
	value, limited, err := parseLimit(data)
	if err != nil {
		return 0, false, fmt.Errorf("parsing %s: %w", filepath.Join(group, name), err)
	}
	return value, limited, nil
}

// cgroupQuota reads a cgroup's cpu.max quota.
//
// Takes group (string) which is the cgroup directory.
//
// Returns float64 which is the quota in cores, or zero when unlimited or absent.
// Returns bool which is true when a quota is set.
// Returns error which reports an unreadable or malformed file.
func cgroupQuota(group string) (float64, bool, error) {
	data, err := readOptionalCgroupFile(group, cgroupCPUMaxFile)
	if err != nil || data == nil {
		return 0, false, err
	}
	quota, quoted, err := parseQuota(data)
	if err != nil {
		return 0, false, fmt.Errorf("parsing %s: %w", filepath.Join(group, cgroupCPUMaxFile), err)
	}
	return quota, quoted, nil
}

// cgroupKeyed reads a "key value" cgroup file such as memory.events.
//
// Takes group (string) which is the cgroup directory.
// Takes name (string) which is the file name.
//
// Returns map[string]uint64 which maps keys to values, empty when the file is absent.
// Returns error which reports an unreadable or malformed file.
func cgroupKeyed(group, name string) (map[string]uint64, error) {
	data, err := readOptionalCgroupFile(group, name)
	if err != nil {
		return nil, err
	}
	values, err := parseKeyed(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filepath.Join(group, name), err)
	}
	return values, nil
}

// parseLimit parses a single cgroup value, where "max" means no limit.
//
// Takes data ([]byte) which is the file contents.
//
// Returns uint64 which is the value, or zero when unlimited.
// Returns bool which is true when the contents are a number.
// Returns error which reports malformed contents.
func parseLimit(data []byte) (uint64, bool, error) {
	text := strings.TrimSpace(string(data))
	if text == cgroupUnlimited {
		return 0, false, nil
	}
	value, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

// parseQuota parses cpu.max contents into cores.
//
// Takes data ([]byte) which is the file contents, a quota and a period.
//
// Returns float64 which is the quota in cores, or zero when unlimited.
// Returns bool which is true when a quota is set.
// Returns error which reports malformed contents or a non-positive period.
func parseQuota(data []byte) (float64, bool, error) {
	fields := strings.Fields(string(data))
	if len(fields) != 2 {
		return 0, false, errors.New("want quota and period")
	}
	if fields[0] == cgroupUnlimited {
		return 0, false, nil
	}
	quota, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false, err
	}
	period, err := strconv.ParseFloat(fields[1], 64)
	if err != nil || period <= 0 {
		return 0, false, fmt.Errorf("invalid period %q", fields[1])
	}
	return quota / period, true, nil
}

// parseKeyed parses "key value" lines, skipping lines of any other shape.
//
// Takes data ([]byte) which is the file contents.
//
// Returns map[string]uint64 which maps keys to values.
// Returns error which reports a non-numeric value.
func parseKeyed(data []byte) (map[string]uint64, error) {
	values := map[string]uint64{}
	for line := range strings.Lines(string(data)) {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, err
		}
		values[fields[0]] = value
	}
	return values, nil
}

// parsePressure reads the avg10 value of the "some" line of a PSI file.
//
// Takes data ([]byte) which is the file contents.
//
// Returns float64 which is the 10-second stall percentage.
// Returns error which reports a missing or malformed "some" avg10 value.
func parsePressure(data []byte) (float64, error) {
	for line := range strings.Lines(string(data)) {
		rest, ok := strings.CutPrefix(line, "some ")
		if !ok {
			continue
		}
		for field := range strings.FieldsSeq(rest) {
			if value, ok := strings.CutPrefix(field, "avg10="); ok {
				return strconv.ParseFloat(value, 64)
			}
		}
	}
	return 0, errMissingPressure
}

// pressure returns the higher of host and cgroup memory pressure.
//
// Takes group (string) which is the cgroup directory.
//
// Returns float64 which is the 10-second "some" stall percentage.
// Returns error which reports unreadable or malformed pressure files.
func pressure(group string) (float64, error) {
	data, err := os.ReadFile(hostPressureFile)
	if err != nil {
		return 0, err
	}
	host, err := parsePressure(data)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", hostPressureFile, err)
	}
	if data, err = readCgroupFile(group, "memory.pressure"); err != nil {
		return 0, fmt.Errorf("reading %s memory.pressure: %w", group, err)
	}
	own, err := parsePressure(data)
	if err != nil {
		return 0, fmt.Errorf("%s memory.pressure: %w", group, err)
	}
	return max(host, own), nil
}

// MemInfo converts a KiB-valued /proc/meminfo field to bytes.
//
// Takes key (string) which is the field name, such as MemAvailable.
//
// Returns uint64 which is the value in bytes.
// Returns error which reports an unreadable file or a missing field.
func MemInfo(key string) (uint64, error) {
	data, err := os.ReadFile(procMeminfo)
	if err != nil {
		return 0, err
	}
	values, err := parseMemInfo(data)
	if err != nil {
		return 0, err
	}
	value, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("%s has no %s", procMeminfo, key)
	}
	return value, nil
}

// parseMemInfo reads numeric /proc/meminfo fields and treats their values as KiB.
//
// Takes data ([]byte) which is the file contents.
//
// Returns map[string]uint64 which maps field names to bytes.
// Returns error which reports a non-numeric value.
func parseMemInfo(data []byte) (map[string]uint64, error) {
	values := map[string]uint64{}
	for line := range strings.Lines(string(data)) {
		name, value, ok := strings.Cut(line, ":")
		fields := strings.Fields(value)
		if !ok || len(fields) == 0 {
			continue
		}
		kibibytes, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing %s %s: %w", procMeminfo, name, err)
		}
		values[name] = kibibytes << kibibyteShift
	}
	return values, nil
}

// Affinity lists the CPUs this process may run on.
//
// Returns []int which lists the allowed CPU IDs in ascending order.
// Returns error which reports a failed affinity query.
func Affinity() ([]int, error) {
	var set unix.CPUSet
	if err := unix.SchedGetaffinity(0, &set); err != nil {
		return nil, fmt.Errorf("reading CPU affinity: %w", err)
	}
	var cpus []int
	for cpu := range len(set) * cpuSetWordBits {
		if set.IsSet(cpu) {
			cpus = append(cpus, cpu)
		}
	}
	return cpus, nil
}

// DiskUsed sums logical file sizes without following symbolic links.
//
// Files that disappear during the scan do not count toward the total.
//
// Takes root (string) which is the directory to measure.
//
// Returns uint64 which is the total size in bytes, zero when root is absent.
// Returns error which reports cancellation or a failed directory or file read.
func DiskUsed(ctx context.Context, root string) (uint64, error) {
	var total uint64
	err := filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		total += safeconv.Int64ToUint64(info.Size())
		return nil
	})
	return total, err
}

// DiskFree reports the space available to this user on the filesystem holding path, or on
// its nearest existing ancestor.
//
// Takes path (string) which is the directory whose filesystem is measured.
//
// Returns uint64 which is the available space in bytes.
// Returns error which reports a failed filesystem query.
func DiskFree(path string) (uint64, error) {
	existing := path
	for {
		if _, err := os.Stat(existing); err == nil || existing == filepath.Dir(existing) {
			break
		}
		existing = filepath.Dir(existing)
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(existing, &stat); err != nil {
		return 0, fmt.Errorf("measuring free space at %s: %w", existing, err)
	}
	return stat.Bavail * safeconv.Int64ToUint64(stat.Frsize), nil
}
