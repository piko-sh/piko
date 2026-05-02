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

//go:build crosslang

package harness

import (
	"math"
	"sort"
)

// Summarise reduces a slice of nanosecond samples into a single Aggregate record. Caller
// is responsible for filtering to only successful runs.
// Returns Aggregate with Runs=0 if samples is empty.
//
// `samples` is the per-iteration wall/inner runtime (Mode-dependent). `compileSamples` is
// the per-run CompileNanos values; their median is reported as MedianCompileNanos. Pass
// nil if compile timing was not captured. `kInner` is the spec.KInner the inner-loop ran
// for; used to derive per-iteration runtime when computing ColdStartNanos in
// ModeInnerLoop. For ModeEndToEnd kInner is ignored.
func Summarise(
	benchmark string,
	runner RunnerKind,
	mode RunMode,
	samples []int64,
	rssSamples []int64,
	compileSamples []int64,
	kInner int,
) Aggregate {
	if len(samples) == 0 {
		return Aggregate{Benchmark: benchmark, Runner: runner, Mode: mode}
	}
	sorted := append([]int64(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	medianCompile := medianInt64Unsorted(compileSamples)
	median := medianInt64(sorted)
	coldStart := medianCompile
	if mode == ModeInnerLoop && kInner > 0 {
		coldStart += median / int64(kInner)
	} else {
		coldStart += median
	}
	return Aggregate{
		Benchmark:          benchmark,
		Runner:             runner,
		Mode:               mode,
		Runs:               len(sorted),
		MedianNanos:        median,
		MeanNanos:          meanInt64(sorted),
		StddevNanos:        stddevInt64(sorted),
		MinNanos:           sorted[0],
		P95Nanos:           percentileInt64(sorted, 0.95),
		PeakRSSKB:          medianInt64Unsorted(rssSamples),
		MedianCompileNanos: medianCompile,
		ColdStartNanos:     coldStart,
	}
}

// medianInt64 returns the median of a pre-sorted slice. Even-length inputs return the
// average of the two middle elements (rounded toward zero).
func medianInt64(sorted []int64) int64 {
	count := len(sorted)
	if count == 0 {
		return 0
	}
	if count%2 == 1 {
		return sorted[count/2]
	}
	return (sorted[count/2-1] + sorted[count/2]) / 2
}

// medianInt64Unsorted sorts a copy before computing the median.
func medianInt64Unsorted(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	sorted := append([]int64(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return medianInt64(sorted)
}

// meanInt64 returns the arithmetic mean rounded toward zero.
func meanInt64(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	var total int64
	for _, sample := range samples {
		total += sample
	}
	return total / int64(len(samples))
}

// stddevInt64 returns the population standard deviation, rounded toward zero. Returns 0
// for single-sample inputs.
func stddevInt64(samples []int64) int64 {
	count := len(samples)
	if count < 2 {
		return 0
	}
	mean := meanInt64(samples)
	var sumSquares float64
	for _, sample := range samples {
		delta := float64(sample - mean)
		sumSquares += delta * delta
	}
	return int64(math.Sqrt(sumSquares / float64(count)))
}

// percentileInt64 returns the nearest-rank percentile of a pre-sorted slice using the
// simple ceil-of-(p * N) convention. p must be in [0, 1].
func percentileInt64(sorted []int64, p float64) int64 {
	count := len(sorted)
	if count == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[count-1]
	}
	rank := int(math.Ceil(p * float64(count)))
	if rank < 1 {
		rank = 1
	}
	if rank > count {
		rank = count
	}
	return sorted[rank-1]
}
