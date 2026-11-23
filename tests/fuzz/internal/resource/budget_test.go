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
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdmissionReservesBothHostAndCgroupHeadroom(t *testing.T) {
	budget := Budget{minFree: 50, disk: 1000}
	sample := Sample{Available: 100, Headroom: 40, DiskFree: 1000}
	assert.Equal(t, "cgroup memory headroom", budget.holds(sample, 41))
	assert.Equal(t, "host memory reserve", budget.holds(sample, 51))
	assert.Empty(t, budget.holds(sample, 40))
	sample.DiskUsed = 900
	assert.Equal(t, diskHoldReason, budget.holds(sample, 40))
}

func TestOOMRequiresIncreasedCountInSameGroup(t *testing.T) {
	before := Sample{Cgroup: "campaign", OOMKills: 4}
	assert.False(t, before.OOMSince(before), "historical OOM attributed to new phase")
	sample := Sample{Cgroup: "campaign", OOMKills: 5}
	assert.True(t, sample.OOMSince(before), "phase OOM missed")
	sample.Cgroup = "other"
	assert.False(t, sample.OOMSince(before), "unrelated OOM attributed to phase")
}

func TestReadCachesRecentSample(t *testing.T) {
	var calls atomic.Int32
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		calls.Add(1)
		return Sample{Cgroup: "campaign", DiskUsed: 7}, nil
	})
	first, err := budget.read(t.Context(), false)
	require.NoError(t, err)
	second, err := budget.read(t.Context(), false)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Equal(t, int32(1), calls.Load())

	_, err = budget.read(t.Context(), true)
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load(), "forced read reused the cache")
}

func TestReadScansDiskOnlyWhenDue(t *testing.T) {
	var scans atomic.Int32
	budget := fakeBudget(func(_ context.Context, scan bool) (Sample, error) {
		if scan {
			scans.Add(1)
			return Sample{DiskUsed: 42}, nil
		}
		return Sample{DiskUsed: 0}, nil
	})
	sample, err := budget.read(t.Context(), false)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), sample.DiskUsed)
	forced, err := budget.read(t.Context(), true)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), forced.DiskUsed, "unscanned reading lost the cached disk use")
	assert.Equal(t, int32(1), scans.Load())
}

func TestReadCollapsesConcurrentProbes(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		calls.Add(1)
		<-release
		return Sample{Cgroup: "campaign"}, nil
	})
	const readers = 8
	var started sync.WaitGroup
	var group sync.WaitGroup
	errs := make(chan error, readers)
	for range readers {
		started.Add(1)
		group.Go(func() {
			started.Done()
			_, err := budget.read(t.Context(), false)
			errs <- err
		})
	}
	started.Wait()
	require.Eventually(t, func() bool { return calls.Load() == 1 }, 5*time.Second, time.Millisecond)
	close(release)
	group.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	assert.LessOrEqual(t, calls.Load(), int32(2))
}

func TestReadDoesNotHoldMutexDuringProbe(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		close(entered)
		<-release
		return Sample{}, nil
	})
	done := make(chan error, 1)
	go func() {
		_, err := budget.read(t.Context(), true)
		done <- err
	}()
	<-entered
	locked := budget.mu.TryLock()
	if locked {
		budget.mu.Unlock()
	}
	close(release)
	require.NoError(t, <-done)
	assert.True(t, locked, "budget mutex held across the probe")
}

func TestReadReportsCallerCancellation(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	budget := fakeBudget(func(ctx context.Context, _ bool) (Sample, error) {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return Sample{}, errors.New("probe stopped")
	})
	cause := errors.New("caller gave up")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)
	_, err := budget.read(ctx, false)
	require.ErrorIs(t, err, cause)
}

func TestProbeTimeoutReportsCause(t *testing.T) {
	budget := fakeBudget(func(ctx context.Context, _ bool) (Sample, error) {
		<-ctx.Done()
		return Sample{}, ctx.Err()
	})
	ctx, cancel := context.WithDeadlineCause(t.Context(), time.Now(), errors.New("test probe deadline"))
	defer cancel()
	_, err := budget.probeOnce(ctx, false)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestAdmitReturnsCallerCauseWithoutLimit(t *testing.T) {
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		return Sample{}, errors.New("probe must not run")
	})
	cause := errors.New("operator interrupt")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)

	err := budget.Admit(ctx, 0)

	require.ErrorIs(t, err, cause)
	assert.NotErrorIs(t, err, ErrLimit)
}

func TestAdmissionStoppedWrapsLimitOnlyForAdmissionTimeout(t *testing.T) {
	timedOut, cancel := context.WithCancelCause(t.Context())
	cancel(errAdmissionTimeout)

	err := admissionStopped(timedOut, "memory pressure")
	require.ErrorIs(t, err, ErrLimit)
	require.ErrorIs(t, err, errAdmissionTimeout)
	assert.Equal(t, "fuzz resource limit: admission (memory pressure): resource admission timed out", err.Error())

	err = admissionStopped(timedOut, "")
	assert.Equal(t, "fuzz resource limit: admission: resource admission timed out", err.Error())

	interrupted, stop := context.WithCancel(t.Context())
	stop()
	err = admissionStopped(interrupted, "memory pressure")
	require.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, ErrLimit)
}

func TestOpenRequiresLauncherEnvironment(t *testing.T) {
	t.Setenv("PIKO_FUZZ_ROOT", "")
	_, err := Open(t.Context())
	require.Error(t, err)

	t.Setenv("PIKO_FUZZ_ROOT", "/root")
	t.Setenv("PIKO_FUZZ_CPUS", "nope")
	_, err = Open(t.Context())
	require.ErrorContains(t, err, "PIKO_FUZZ_CPUS")
}

func TestSampledMemoryIncludesIntermediatePeakAndSurvivesMissingRead(t *testing.T) {
	var readCount int
	sampled := make(chan struct{})
	failure := errors.New("trial failed")
	action := func() error {
		select {
		case <-sampled:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "no intermediate memory read")
		}
		return failure
	}
	memory, err := measurePhase(func() (uint64, error) {
		readCount++
		switch readCount {
		case 1:
			return 10, nil
		case 2:
			return 0, errors.New("transient read error")
		case 3:
			close(sampled)
			return 85, nil
		default:
			return 20, nil
		}
	}, time.Millisecond, action)
	require.ErrorIs(t, err, failure)
	assert.True(t, memory.StartKnown)
	assert.True(t, memory.EndKnown)
	assert.Equal(t, uint64(10), memory.StartBytes)
	assert.Equal(t, uint64(20), memory.EndBytes)
	assert.Equal(t, uint64(85), memory.SampledPeakBytes)
	assert.GreaterOrEqual(t, memory.Samples, uint64(3))
	assert.Equal(t, uint64(1), memory.ReadErrors)
}

func TestMeasurePhaseReadsCurrentIndependentlyOfKernelPeak(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "memory.current"), []byte("48\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "memory.peak"), []byte("1000\n"), 0o600))
	memory, err := MeasurePhase(directory, func() error { return nil })
	require.NoError(t, err)
	assert.Equal(t, uint64(48), memory.SampledPeakBytes)
	assert.GreaterOrEqual(t, memory.Samples, uint64(2))
	assert.Zero(t, memory.ReadErrors)

	missing, err := MeasurePhase("", func() error { return nil })
	require.NoError(t, err)
	assert.Zero(t, missing.Samples)
	assert.GreaterOrEqual(t, missing.ReadErrors, uint64(2))
	assert.False(t, missing.StartKnown)
	assert.False(t, missing.EndKnown)
}

func fakeBudget(read func(context.Context, bool) (Sample, error)) *Budget {
	return &Budget{
		readKernel: read,
		minFree:    0,
		disk:       1 << 30,
	}
}

func TestSampleReturnsCachedReadingAndSampleNowProbesAgain(t *testing.T) {
	var calls atomic.Int32
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		return Sample{Cgroup: "campaign", OOMKills: uint64(calls.Add(1))}, nil
	})

	first, err := budget.Sample(t.Context())
	require.NoError(t, err)
	cached, err := budget.Sample(t.Context())
	require.NoError(t, err)
	fresh, err := budget.SampleNow(t.Context())
	require.NoError(t, err)

	assert.Equal(t, uint64(1), first.OOMKills)
	assert.Equal(t, first, cached)
	assert.Equal(t, uint64(2), fresh.OOMKills)
}

func TestSampleNowReportsProbeFailure(t *testing.T) {
	failure := errors.New("probe broke")
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		return Sample{}, failure
	})

	_, err := budget.SampleNow(t.Context())

	require.ErrorIs(t, err, failure)
	assert.ErrorContains(t, err, "reading kernel resource limits")
}

func TestAdmitAcceptsReservationThatFits(t *testing.T) {
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		return Sample{Available: 1000, Headroom: 1000, DiskFree: 1073741824}, nil
	})

	assert.NoError(t, budget.Admit(t.Context(), 100))
}

func TestAdmitReturnsProbeFailure(t *testing.T) {
	failure := errors.New("probe unavailable")
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		return Sample{}, failure
	})

	assert.ErrorIs(t, budget.Admit(t.Context(), 0), failure)
}

func TestAdmitReturnsCallerCauseWhileHeld(t *testing.T) {
	cause := errors.New("campaign deadline")
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		cancel(cause)
		return Sample{Available: 10, Headroom: 10, DiskFree: 1073741824}, nil
	})

	err := budget.Admit(ctx, 100)

	require.ErrorIs(t, err, cause)
	assert.NotErrorIs(t, err, ErrLimit)
}

func TestMonitorCancelsWhenBaselineUnavailable(t *testing.T) {
	failure := errors.New("no accounting")
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		return Sample{}, failure
	})
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	budget.Monitor(ctx, cancel)

	assert.ErrorIs(t, context.Cause(ctx), failure)
}

func TestMonitorCancelsOnNewOOMKill(t *testing.T) {
	var calls atomic.Int32
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		if calls.Add(1) == 1 {
			return Sample{Cgroup: "campaign", Available: 1000, Headroom: 1000, DiskFree: 1073741824}, nil
		}
		return Sample{Cgroup: "campaign", OOMKills: 1, Available: 1000, Headroom: 1000, DiskFree: 1073741824}, nil
	})
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	budget.Monitor(ctx, cancel)

	require.ErrorIs(t, context.Cause(ctx), ErrLimit)
	assert.ErrorContains(t, context.Cause(ctx), "cgroup OOM kill")
}

func TestMonitorCancelsImmediatelyOnDiskHold(t *testing.T) {
	var calls atomic.Int32
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		if calls.Add(1) == 1 {
			return Sample{Available: 1000, Headroom: 1000, DiskFree: 1073741824}, nil
		}
		return Sample{Available: 1000, Headroom: 1000, DiskFree: 0}, nil
	})
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	budget.Monitor(ctx, cancel)

	require.ErrorIs(t, context.Cause(ctx), ErrLimit)
	assert.ErrorContains(t, context.Cause(ctx), diskHoldReason)
}

func TestMonitorCancelsWhenLaterReadFails(t *testing.T) {
	failure := errors.New("accounting vanished")
	var calls atomic.Int32
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		if calls.Add(1) == 1 {
			return Sample{Available: 1000, Headroom: 1000, DiskFree: 1073741824}, nil
		}
		return Sample{}, failure
	})
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	budget.Monitor(ctx, cancel)

	assert.ErrorIs(t, context.Cause(ctx), failure)
}

func TestMonitorReturnsWhenContextEnds(t *testing.T) {
	budget := fakeBudget(func(context.Context, bool) (Sample, error) {
		return Sample{Available: 1000, Headroom: 1000, DiskFree: 1073741824}, nil
	})
	ctx, cancel := context.WithCancelCause(t.Context())
	stop := errors.New("phase finished")
	done := make(chan struct{})
	go func() {
		defer close(done)
		budget.Monitor(ctx, cancel)
	}()

	cancel(stop)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "monitor did not return after cancellation")
	}
	assert.ErrorIs(t, context.Cause(ctx), stop)
}

func TestOpenRejectsRelativePathsAndZeroBudgets(t *testing.T) {
	t.Setenv("PIKO_FUZZ_ROOT", t.TempDir())
	t.Setenv("PIKO_FUZZ_STATE", "relative")
	t.Setenv("PIKO_FUZZ_CPUS", "2")
	t.Setenv("PIKO_FUZZ_MEMORY_BYTES", "4096")
	t.Setenv("PIKO_FUZZ_MIN_FREE_BYTES", "0")
	t.Setenv("PIKO_FUZZ_DISK_BYTES", "8192")
	_, err := Open(t.Context())
	require.ErrorContains(t, err, "invalid fuzz launcher paths or budgets")

	t.Setenv("PIKO_FUZZ_STATE", t.TempDir())
	t.Setenv("PIKO_FUZZ_CPUS", "0")
	_, err = Open(t.Context())
	assert.ErrorContains(t, err, "invalid fuzz launcher paths or budgets")
}

func TestMeasurePhaseCountsUnparsableMemoryReading(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "memory.current"), []byte("lots\n"), 0o600))

	memory, err := MeasurePhase(directory, func() error { return nil })

	require.NoError(t, err)
	assert.Zero(t, memory.Samples)
	assert.GreaterOrEqual(t, memory.ReadErrors, uint64(2))
}
