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

// Package resource admits fuzz phases under the aggregate campaign and host limits.
package resource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	// pollInterval is the ordinary resource-sampling interval.
	pollInterval = time.Second

	// diskInterval is the state-tree disk-sampling interval.
	diskInterval = 15 * time.Second

	// recoveryInterval is the resource-recovery retry interval.
	recoveryInterval = 3 * time.Second

	// admissionTimeout bounds resource-admission waiting.
	admissionTimeout = 30 * time.Second

	// probeTimeout bounds one kernel reading, including a state-tree scan.
	probeTimeout = 20 * time.Second

	// pressureGrace is how long non-disk pressure may persist before Monitor cancels work.
	pressureGrace = 5 * time.Second

	// phaseMemoryInterval is the active-phase memory-sampling interval.
	phaseMemoryInterval = 250 * time.Millisecond

	// sharedProbeKey identifies the collapsible cached-reading probe.
	sharedProbeKey = "sample"

	// diskHoldReason is the admission hold caused by the disk budget.
	diskHoldReason = "disk budget or free-space reserve"

	// pressureLimit is the memory-pressure stall percentage that holds admission.
	pressureLimit = 10

	// diskReserveCeiling caps the disk reserve kept free beneath the budget in bytes.
	diskReserveCeiling = 256 << 20

	// diskReserveDivisor sets the disk reserve as a fraction of the disk budget.
	diskReserveDivisor = 10

	// cgroupMemoryFile is the cgroup file reporting current memory use.
	cgroupMemoryFile = "memory.current"

	// TasksMax caps the number of tasks in the campaign scope.
	TasksMax = 2048

	// EnvPrefix prefixes the fuzz launcher environment variables.
	EnvPrefix = "PIKO_FUZZ_"
)

var (
	// ErrLimit identifies an admission or measured resource limit.
	ErrLimit = errors.New("fuzz resource limit")

	// errProbeTimeout is the cancellation cause of a resource probe that overran.
	errProbeTimeout = errors.New("resource probe timed out")

	// errAdmissionTimeout is the cancellation cause of an admission wait that overran.
	errAdmissionTimeout = errors.New("resource admission timed out")

	// errMissingCgroup reports a memory read without a verified cgroup.
	errMissingCgroup = errors.New("missing cgroup path")
)

// Sample records kernel accounting for the entire run and current host headroom.
type Sample struct {
	// Cgroup is the verified process-tree cgroup.
	Cgroup string `json:"cgroup"`

	// CPUs lists the allowed CPU IDs.
	CPUs []int `json:"cpus"`

	// QuotaCPUs is the effective CPU quota in cores.
	QuotaCPUs float64 `json:"quota_cpus"`

	// MemoryMax is the cgroup memory ceiling in bytes.
	MemoryMax uint64 `json:"memory_max"`

	// MemoryHigh is the cgroup memory-throttling boundary in bytes.
	MemoryHigh uint64 `json:"memory_high"`

	// SwapMax is the cgroup swap ceiling in bytes.
	SwapMax uint64 `json:"swap_max"`

	// TasksMax is the cgroup task ceiling.
	TasksMax uint64 `json:"tasks_max"`

	// MemoryCurrent is current cgroup memory use in bytes.
	MemoryCurrent uint64 `json:"memory_current"`

	// MemoryPeak is the kernel's cumulative memory peak in bytes.
	MemoryPeak uint64 `json:"memory_peak"`

	// Headroom is the remaining cgroup memory allowance in bytes.
	Headroom uint64 `json:"headroom"`

	// Available is available host memory in bytes.
	Available uint64 `json:"available"`

	// PSI is the measured memory pressure.
	PSI float64 `json:"psi"`

	// OOMKills counts cgroup out-of-memory kills.
	OOMKills uint64 `json:"oom_kills"`

	// HighEvents counts cgroup memory-high events.
	HighEvents uint64 `json:"high_events"`

	// CPUUsec is measured CPU time in microseconds.
	CPUUsec uint64 `json:"cpu_usage_usec"`

	// ThrottledUsec is measured CPU throttling time in microseconds.
	ThrottledUsec uint64 `json:"throttled_usec"`

	// DiskUsed is the measured state-tree size in bytes.
	DiskUsed uint64 `json:"disk_used"`

	// DiskFree is available filesystem space in bytes.
	DiskFree uint64 `json:"disk_free"`
}

// Budget is shared by scheduling and the active-phase monitor.
type Budget struct {
	// updated is the start time of the probe that produced last.
	updated time.Time

	// diskChecked is the start time of the latest probe that scanned the state tree.
	diskChecked time.Time

	// flight shares one refresh among concurrent readers of the cached sample.
	flight singleflight.Group

	// readKernel verifies limits and optionally measures the state tree's disk use.
	readKernel func(ctx context.Context, scan bool) (Sample, error)

	// last holds the latest resource reading.
	last Sample

	// minFree is the required host memory reserve in bytes.
	minFree uint64

	// disk is the state-tree disk budget in bytes.
	disk uint64

	// mu protects updated, diskChecked and last.
	mu sync.Mutex
}

// PhaseMemory describes memory.current for the entire campaign cgroup, including the
// supervisor, page cache and children.
//
// Short-lived peaks between reads can be missed. The kernel's cumulative memory.peak
// remains in the run's Sample.
type PhaseMemory struct {
	// Samples counts successful memory reads.
	Samples uint64

	// ReadErrors counts failed memory reads.
	ReadErrors uint64

	// StartBytes is the cgroup memory reading at phase entry in bytes.
	StartBytes uint64

	// EndBytes is the cgroup memory reading at phase exit in bytes.
	EndBytes uint64

	// SampledPeakBytes is the highest sampled cgroup memory reading in bytes.
	SampledPeakBytes uint64

	// StartKnown reports whether the entry memory read succeeded.
	StartKnown bool

	// EndKnown reports whether the exit memory read succeeded.
	EndKnown bool
}

// kernelLimits holds the ceilings the launcher asked the kernel to enforce.
type kernelLimits struct {
	// state is the campaign state directory whose tree counts against the disk budget.
	state string

	// cpus is the CPU ceiling.
	cpus uint64

	// memory is the memory ceiling in bytes.
	memory uint64
}

// probeResult records a kernel sample, its start time and whether it measured disk use.
type probeResult struct {
	// started is when the probe began.
	started time.Time

	// sample holds the kernel measurements.
	sample Sample

	// scanned reports whether the probe measured the state tree.
	scanned bool
}

// NewBudget sets the memory reserve and disk limit for a resource reader.
//
// Takes read (func(ctx context.Context, scan bool) (Sample, error)) which samples the
// kernel and measures disk use when scan is true.
// Takes minFree (uint64) which is the host memory reserve in bytes.
// Takes disk (uint64) which is the state-tree disk budget in bytes.
//
// Returns *Budget which samples through read.
func NewBudget(read func(ctx context.Context, scan bool) (Sample, error), minFree, disk uint64) *Budget {
	return &Budget{
		updated:     time.Time{},
		diskChecked: time.Time{},
		flight:      singleflight.Group{},
		readKernel:  read,
		last:        Sample{},
		minFree:     minFree,
		disk:        disk,
		mu:          sync.Mutex{},
	}
}

// Sample returns the latest measurements.
//
// Returns Sample which holds the whole-process-tree resource reading.
// Returns error which reports failed accounting, unverified limits or cancellation.
func (b *Budget) Sample(ctx context.Context) (Sample, error) {
	return b.read(ctx, false)
}

// SampleNow reads fresh kernel event counters for attributing a worker exit.
//
// Returns Sample which holds the whole-process-tree resource reading.
// Returns error which reports failed accounting, unverified limits or cancellation.
func (b *Budget) SampleNow(ctx context.Context) (Sample, error) {
	return b.read(ctx, true)
}

// MinFree returns the configured host memory reserve.
//
// Returns uint64 which is the reserve in bytes.
func (b *Budget) MinFree() uint64 {
	return b.minFree
}

// OOMSince reports a kernel OOM kill in the same cgroup during this phase.
//
// Takes before (Sample) which holds whole-process-tree resource readings.
//
// Returns bool which reports a new kernel OOM kill in the same cgroup.
func (s Sample) OOMSince(before Sample) bool {
	return s.Cgroup != "" && s.Cgroup == before.Cgroup && s.OOMKills > before.OOMKills
}

// Admit waits for a phase reservation to fit, allowing sustained recovery before
// restarting.
//
// Takes estimate (uint64) which is the estimated incremental memory requirement in bytes.
//
// Returns error which reports unavailable measurements, the caller's cancellation cause,
// or an admission timeout wrapping ErrLimit, or nil on success.
func (b *Budget) Admit(ctx context.Context, estimate uint64) error {
	ctx, cancel := context.WithTimeoutCause(ctx, admissionTimeout, errAdmissionTimeout)
	defer cancel()
	var held time.Time
	var reason string
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return admissionStopped(ctx, reason)
		}
		sample, err := b.read(ctx, false)
		if err != nil {
			if ctx.Err() != nil {
				return admissionStopped(ctx, reason)
			}
			return err
		}
		if current := b.holds(sample, estimate); current != "" {
			held = time.Now()
			reason = current
		} else if held.IsZero() || time.Since(held) >= recoveryInterval {
			return nil
		}
		select {
		case <-ctx.Done():
			return admissionStopped(ctx, reason)
		case <-ticker.C:
		}
	}
}

// Monitor cancels the current operation if host pressure persists, evidence space runs
// low, an OOM occurs, or measurements become unavailable.
//
// Takes cancel (context.CancelCauseFunc) which stops active work with its resource-limit
// cause.
func (b *Budget) Monitor(ctx context.Context, cancel context.CancelCauseFunc) {
	baseline, err := b.read(ctx, false)
	if err != nil {
		cancel(err)
		return
	}
	var firstHold time.Time
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sample, err := b.read(ctx, false)
			if err != nil {
				cancel(err)
				return
			}
			if sample.OOMSince(baseline) {
				cancel(fmt.Errorf("%w: cgroup OOM kill", ErrLimit))
				return
			}
			reason := b.holds(sample, 0)
			switch {
			case reason == "":
				firstHold = time.Time{}
			case reason == diskHoldReason || (!firstHold.IsZero() && time.Since(firstHold) >= pressureGrace):
				cancel(fmt.Errorf("%w: %s", ErrLimit, reason))
				return
			case firstHold.IsZero():
				firstHold = time.Now()
			}
		}
	}
}

// read samples resource use or reuses a recent cached reading.
//
// At most one probe refreshes cached readings at a time. Concurrent callers share its
// result. A forced read always runs its own probe so that it observes counters taken
// after the call began.
//
// Takes force (bool) which bypasses the cached resource reading.
//
// Returns Sample which holds the whole-process-tree resource reading.
// Returns error which reports failed accounting, unverified limits or cancellation.
//
// Concurrency: Safe for concurrent use. The budget mutex guards only the cache snapshot
// and update. The kernel read runs without it.
func (b *Budget) read(ctx context.Context, force bool) (Sample, error) {
	b.mu.Lock()
	now := time.Now()
	scan := !force && now.Sub(b.diskChecked) >= diskInterval
	if !force && !scan && now.Sub(b.updated) < pollInterval {
		cached := b.last
		b.mu.Unlock()
		return cached, nil
	}
	b.mu.Unlock()
	if ctx.Err() != nil {
		return Sample{}, context.Cause(ctx)
	}
	if force {
		result, err := b.probeOnce(ctx, false)
		if err != nil {
			return Sample{}, err
		}
		return b.store(result), nil
	}
	channel := b.flight.DoChan(sharedProbeKey, func() (any, error) {
		result, err := b.probeOnce(context.WithoutCancel(ctx), scan)
		if err != nil {
			return Sample{}, err
		}
		return b.store(result), nil
	})
	select {
	case <-ctx.Done():
		return Sample{}, context.Cause(ctx)
	case outcome := <-channel:
		if outcome.Err != nil {
			return Sample{}, outcome.Err
		}
		sample, ok := outcome.Val.(Sample)
		if !ok {
			return Sample{}, fmt.Errorf("resource probe returned %T", outcome.Val)
		}
		return sample, nil
	}
}

// probeOnce takes one kernel reading.
//
// Takes scan (bool) which requests a state-tree disk measurement.
//
// Returns probeResult which holds the reading.
// Returns error which reports unverified limits, unreadable accounting or a timeout.
func (b *Budget) probeOnce(ctx context.Context, scan bool) (probeResult, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeoutCause(ctx, probeTimeout, errProbeTimeout)
	defer cancel()
	sample, err := b.readKernel(ctx, scan)
	if err != nil {
		if ctx.Err() != nil {
			return probeResult{}, fmt.Errorf("reading kernel resource limits: %w", errors.Join(context.Cause(ctx), err))
		}
		return probeResult{}, fmt.Errorf("reading kernel resource limits: %w", err)
	}
	return probeResult{started: started, sample: sample, scanned: scan}, nil
}

// store merges a probe reading into the cache unless a newer probe already landed.
//
// Takes result (probeResult) which holds the decoded reading.
//
// Returns Sample which is the reading with the cached disk use filled in when the probe
// did not scan.
//
// Concurrency: Safe for concurrent use. Takes the budget mutex.
func (b *Budget) store(result probeResult) Sample {
	b.mu.Lock()
	defer b.mu.Unlock()
	sample := result.sample
	if result.scanned {
		if result.started.After(b.diskChecked) {
			b.diskChecked = result.started
		}
	} else {
		sample.DiskUsed = b.last.DiskUsed
	}
	if !result.started.Before(b.updated) {
		b.updated = result.started
		b.last = sample
	}
	return sample
}

// holds identifies the resource condition preventing admission.
//
// Takes sample (Sample) which holds whole-process-tree resource readings.
// Takes estimate (uint64) which is the estimated incremental memory requirement in bytes.
//
// Returns string which describes the admission hold, or is empty when resources permit
// work.
func (b *Budget) holds(sample Sample, estimate uint64) string {
	switch {
	case sample.Available < b.minFree || sample.Available-b.minFree < estimate:
		return "host memory reserve"
	case sample.Headroom < estimate:
		return "cgroup memory headroom"
	case sample.PSI >= pressureLimit:
		return "memory pressure"
	}
	reserve := DiskReserve(b.disk)
	if sample.DiskUsed >= b.disk-reserve || sample.DiskFree <= reserve {
		return diskHoldReason
	}
	return ""
}

// Open verifies the campaign's kernel limits and creates its resource budget.
//
// Returns *Budget which holds verified launcher limits and the resource sampler.
// Returns error which reports invalid resource limits or inaccessible accounting state,
// or nil on success.
func Open(ctx context.Context) (*Budget, error) {
	root := os.Getenv(EnvPrefix + "ROOT")
	if root == "" {
		return nil, errors.New("run through hack/fuzz/fuzz.sh to establish CPU and memory limits")
	}
	values := make(map[string]uint64)
	for _, key := range []string{"CPUS", "MEMORY_BYTES", "MIN_FREE_BYTES", "DISK_BYTES"} {
		value, err := strconv.ParseUint(os.Getenv(EnvPrefix+key), 10, 63)
		if err != nil {
			return nil, fmt.Errorf("invalid %s%s: %w", EnvPrefix, key, err)
		}
		values[key] = value
	}
	limits := kernelLimits{state: os.Getenv(EnvPrefix + "STATE"), cpus: values["CPUS"], memory: values["MEMORY_BYTES"]}
	if !filepath.IsAbs(root) || !filepath.IsAbs(limits.state) || limits.cpus == 0 || limits.memory == 0 || values["DISK_BYTES"] == 0 {
		return nil, errors.New("invalid fuzz launcher paths or budgets")
	}
	if err := checkEnclosingLimit(os.Getenv(EnvPrefix + "ENCLOSING_LIMIT")); err != nil {
		return nil, err
	}
	budget := NewBudget(func(ctx context.Context, scan bool) (Sample, error) {
		return readKernel(ctx, limits, scan)
	}, values["MIN_FREE_BYTES"], values["DISK_BYTES"])
	if _, err := budget.Sample(ctx); err != nil {
		return nil, err
	}
	return budget, nil
}

// checkEnclosingLimit checks that the supervisor retained the launcher's cgroup limit.
//
// Takes enclosing (string) which is the launcher's limiting cgroup, or empty for none.
//
// Returns error which reports an unreadable cgroup or scope outside that limit.
func checkEnclosingLimit(enclosing string) error {
	if enclosing == "" {
		return nil
	}
	group, err := OwnCgroup()
	if err != nil {
		return err
	}
	if group != enclosing && !strings.HasPrefix(group, enclosing+string(filepath.Separator)) {
		return fmt.Errorf("the campaign scope left the pre-existing aggregate limit at %s", enclosing)
	}
	return nil
}

// DiskReserve returns a tenth of the disk budget, capped at 256 MiB.
//
// Takes disk (uint64) which is the disk budget in bytes.
//
// Returns uint64 which is the reserve in bytes.
func DiskReserve(disk uint64) uint64 {
	return min(uint64(diskReserveCeiling), disk/diskReserveDivisor)
}

// MeasurePhase samples the verified cgroup at entry, every 250 ms, and at exit.
//
// Takes cgroup (string) which is the verified process-tree cgroup.
// Takes action (func() error) which performs the measured operation.
//
// Returns PhaseMemory which holds sampled cgroup memory around the operation.
// Returns error which is the measured operation's error, independent of sampling
// failures.
func MeasurePhase(cgroup string, action func() error) (PhaseMemory, error) {
	read := func() (uint64, error) {
		if cgroup == "" {
			return 0, errMissingCgroup
		}
		data, err := readCgroupFile(cgroup, cgroupMemoryFile)
		if err != nil {
			return 0, fmt.Errorf("read memory.current: %w", err)
		}
		value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse memory.current: %w", err)
		}
		return value, nil
	}
	return measurePhase(read, phaseMemoryInterval, action)
}

// measurePhase samples cgroup memory around the supplied operation.
//
// Takes read (func() (uint64, error)) which returns current cgroup memory use in bytes.
// Takes interval (time.Duration) which selects the memory-sampling period.
// Takes action (func() error) which performs the measured operation.
//
// Returns PhaseMemory which holds sampled cgroup memory around the operation.
// Returns error which is the measured operation's error, independent of sampling
// failures.
//
// Concurrency: Spawns a memory-sampling goroutine and joins it before returning.
func measurePhase(read func() (uint64, error), interval time.Duration, action func() error) (memory PhaseMemory, err error) {
	take := func() (uint64, bool) {
		value, readErr := read()
		if readErr != nil {
			memory.ReadErrors++
			return 0, false
		}
		memory.Samples++
		memory.SampledPeakBytes = max(memory.SampledPeakBytes, value)
		return value, true
	}
	memory.StartBytes, memory.StartKnown = take()
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				take()
			}
		}
	}()
	defer func() {
		close(stop)
		<-done
		memory.EndBytes, memory.EndKnown = take()
	}()
	err = action()
	return memory, err
}

// admissionStopped describes why a cancelled admission wait ended.
//
// Only the admission's own timeout is a resource limit. A cancelled caller, such as an
// interrupt or the campaign deadline, is returned as its own cause.
//
// Takes reason (string) which is the last resource hold, or empty when none was seen.
//
// Returns error which wraps ErrLimit for an admission timeout, or is the caller's
// cancellation cause.
func admissionStopped(ctx context.Context, reason string) error {
	cause := context.Cause(ctx)
	if !errors.Is(cause, errAdmissionTimeout) {
		return cause
	}
	if reason == "" {
		return fmt.Errorf("%w: admission: %w", ErrLimit, cause)
	}
	return fmt.Errorf("%w: admission (%s): %w", ErrLimit, reason, cause)
}
