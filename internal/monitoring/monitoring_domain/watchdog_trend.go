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

package monitoring_domain

import (
	"context"
	"math"
	"strconv"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultTrendWindowSize is the number of heap samples to retain for
	// linear regression (120 samples at 500ms = 60 seconds of history).
	defaultTrendWindowSize = 120

	// defaultTrendEvaluationInterval is the period between trend regression
	// computations.
	defaultTrendEvaluationInterval = 30 * time.Second

	// defaultTrendWarningHorizon is the projected time-to-breach below which
	// a heap trend warning is emitted.
	defaultTrendWarningHorizon = 5 * time.Minute

	// minimumTrendSamples is the minimum number of samples required before a
	// slope computation is meaningful.
	minimumTrendSamples = 10
)

// heapTrendBuffer is a fixed-capacity ring buffer of uint64 heap allocation
// samples used for linear regression analysis.
type heapTrendBuffer struct {
	// samples holds the ring buffer of heap allocation values.
	samples []uint64

	// head is the index of the next write position in the ring buffer.
	head int

	// count is the number of valid samples currently stored.
	count int

	// capacity is the maximum number of samples the buffer can hold.
	capacity int
}

// newHeapTrendBuffer creates a ring buffer with the given capacity.
//
// Takes capacity (int) which is the maximum number of samples to retain.
//
// Returns *heapTrendBuffer which is an empty ring buffer ready for use.
func newHeapTrendBuffer(capacity int) *heapTrendBuffer {
	return &heapTrendBuffer{
		samples:  make([]uint64, capacity),
		capacity: capacity,
	}
}

// add appends a heap allocation sample to the ring buffer, overwriting the
// oldest sample when the buffer is full.
//
// Takes value (uint64) which is the heap allocation sample in bytes.
func (b *heapTrendBuffer) add(value uint64) {
	b.samples[b.head] = value
	b.head = (b.head + 1) % b.capacity

	if b.count < b.capacity {
		b.count++
	}
}

// isFull reports whether the buffer has reached its capacity.
//
// Returns bool which is true when the buffer contains capacity samples.
func (b *heapTrendBuffer) isFull() bool {
	return b.count >= b.capacity
}

// slope computes the least-squares linear regression slope of the buffered
// samples in bytes per sample index.
//
// Returns float64 which is the slope in bytes per sample index, or 0 when
// fewer than two samples are recorded.
//
// The slope formula is:
//
//	slope = (n*sum(i*y_i) - sum(i)*sum(y_i)) / (n*sum(i^2) - sum(i)^2)
//
// where i is the sample index and y_i is the heap allocation at that index.
func (b *heapTrendBuffer) slope() float64 {
	if b.count < 2 {
		return 0
	}

	n := float64(b.count)

	var sumIndex float64
	var sumValue float64
	var sumIndexValue float64
	var sumIndexSquared float64

	start := 0
	if b.count == b.capacity {
		start = b.head
	}

	for step := range b.count {
		index := float64(step)
		sampleIndex := (start + step) % b.capacity
		value := float64(b.samples[sampleIndex])

		sumIndex += index
		sumValue += value
		sumIndexValue += index * value
		sumIndexSquared += index * index
	}

	denominator := n*sumIndexSquared - sumIndex*sumIndex
	if denominator == 0 {
		return 0
	}

	return (n*sumIndexValue - sumIndex*sumValue) / denominator
}

// evaluateHeapTrend appends the current heap allocation to the ring buffer
// and periodically computes a linear regression to project whether the heap
// will breach the memory limit within the configured warning horizon.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes stats (*SystemStats) which contains the current system metrics
// including heap allocation.
func (w *Watchdog) evaluateHeapTrend(ctx context.Context, now time.Time, stats *SystemStats) {
	if ctx.Err() != nil || w.heapTrendBuffer == nil {
		return
	}

	w.heapTrendBuffer.add(stats.Memory.HeapAlloc)

	if now.Sub(w.lastTrendEvaluation) < w.config.TrendEvaluationInterval {
		return
	}

	w.lastTrendEvaluation = now

	if w.heapTrendBuffer.count < minimumTrendSamples {
		return
	}

	slopePerSample := w.heapTrendBuffer.slope()
	if slopePerSample <= 0 {
		return
	}

	slopePerSecond := slopePerSample / w.config.CheckInterval.Seconds()

	watchdogHeapGrowthRateBytesPerSecond.Record(ctx, int64(slopePerSecond))

	effectiveLimit := w.resolveEffectiveMemoryLimit(stats)
	if effectiveLimit == 0 {
		return
	}

	currentHeap := stats.Memory.HeapAlloc
	if currentHeap >= effectiveLimit {
		return
	}

	remainingBytes := effectiveLimit - currentHeap
	secondsToBreach := time.Duration(int64(float64(remainingBytes)/slopePerSecond)) * time.Second

	if secondsToBreach >= w.config.TrendWarningHorizon {
		return
	}

	if !w.tryAdmitWarning(now, "heap_trend") {
		return
	}

	w.emitHeapTrendWarning(ctx, slopePerSecond, currentHeap, effectiveLimit, secondsToBreach)
}

// resolveEffectiveMemoryLimit returns the effective memory limit from
// GOMEMLIMIT or the cgroup memory limit.
//
// Takes stats (*SystemStats) which provides the cgroup memory limit as a
// fallback.
//
// Returns uint64 which is the effective limit in bytes, or 0 when no limit
// is available.
func (w *Watchdog) resolveEffectiveMemoryLimit(stats *SystemStats) uint64 {
	if w.gomemlimit > 0 && w.gomemlimit < math.MaxInt64 {
		return uint64(w.gomemlimit)
	}

	return stats.Process.CgroupMemoryLimit
}

// emitHeapTrendWarning logs and notifies about projected memory limit breach.
//
// Takes slopePerSecond (float64) which is the heap growth rate in bytes per
// second.
// Takes currentHeap (uint64) which is the current heap allocation in bytes.
// Takes effectiveLimit (uint64) which is the effective memory limit in bytes.
// Takes secondsToBreach (time.Duration) which is the projected time until the
// limit is reached.
func (w *Watchdog) emitHeapTrendWarning(
	ctx context.Context,
	slopePerSecond float64,
	currentHeap, effectiveLimit uint64,
	secondsToBreach time.Duration,
) {
	_, l := logger_domain.From(ctx, log)
	l.Warn("Heap growth rate projects a memory limit breach within the warning horizon",
		logger_domain.Float64("growth_rate_bytes_per_second", slopePerSecond),
		logger_domain.Uint64("current_heap_bytes", currentHeap),
		logger_domain.Uint64("effective_limit_bytes", effectiveLimit),
		String("projected_time_to_breach", secondsToBreach.String()),
	)

	watchdogHeapTrendWarningCount.Add(ctx, 1)

	w.sendNotification(ctx, WatchdogEvent{
		EventType: WatchdogEventHeapTrendWarning,
		Priority:  WatchdogPriorityHigh,
		Message:   "Heap growth rate projects a memory limit breach within the warning horizon",
		Fields: map[string]string{
			"growth_rate_bytes_per_second": strconv.FormatFloat(slopePerSecond, 'f', 2, 64),
			"current_heap_bytes":           strconv.FormatUint(currentHeap, 10),
			"effective_limit_bytes":        strconv.FormatUint(effectiveLimit, 10),
			"projected_time_to_breach":     secondsToBreach.String(),
		},
	})
}
