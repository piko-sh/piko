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
	"math"
	"runtime/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRuntimeMetricsCollector_RegistersAvailableMetrics(t *testing.T) {
	t.Parallel()

	collector := newRuntimeMetricsCollector()
	require.NotNil(t, collector)

	idx, ok := collector.indices[metricNameHeapObjects]
	assert.True(t, ok, "heap objects metric should be registered")
	if ok {
		assert.Equal(t, metricNameHeapObjects, collector.samples[idx].Name)
	}
}

func TestNewRuntimeMetricsCollector_HandlesUnknownMetricGracefully(t *testing.T) {
	t.Parallel()

	collector := &runtimeMetricsCollector{
		samples:   nil,
		indices:   map[string]int{},
		available: map[string]bool{metricNameHeapObjects: false},
	}

	snap := collector.sample(time.Unix(123, 0))
	assert.Equal(t, time.Unix(123, 0), snap.SampledAt)
	assert.Zero(t, snap.HeapObjectsBytes)
}

func TestRuntimeMetricsCollector_SampleLiveProcess(t *testing.T) {
	t.Parallel()

	collector := newRuntimeMetricsCollector()
	now := time.Now()
	snap := collector.sample(now)

	assert.Equal(t, now, snap.SampledAt)

	assert.Greater(t, snap.GoMaxProcs, int64(0), "GOMAXPROCS should be positive")
	assert.Greater(t, snap.Goroutines, int64(0), "live process has at least one goroutine")
	assert.Greater(t, snap.TotalMemoryBytes, uint64(0), "total Go-managed memory should be positive")
}

func TestHistogramQuantile_LinearBuckets(t *testing.T) {
	t.Parallel()

	hist := &metrics.Float64Histogram{

		Buckets: []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},

		Counts: []uint64{100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
	}

	p50 := histogramQuantile(hist, 0.50)
	p99 := histogramQuantile(hist, 0.99)

	assert.InDelta(t, 500*time.Millisecond, p50, float64(100*time.Millisecond), "p50 within tolerance")

	assert.GreaterOrEqual(t, p99, 900*time.Millisecond)
}

func TestHistogramQuantile_EmptyHistogram(t *testing.T) {
	t.Parallel()

	hist := &metrics.Float64Histogram{
		Buckets: []float64{0.0, 1.0, 2.0},
		Counts:  []uint64{0, 0},
	}

	assert.Zero(t, histogramQuantile(hist, 0.99), "empty histogram returns zero")
	assert.Zero(t, histogramQuantile(nil, 0.50), "nil histogram returns zero")
}

func TestHistogramQuantile_QuantileClamping(t *testing.T) {
	t.Parallel()

	hist := &metrics.Float64Histogram{
		Buckets: []float64{0.0, 1.0},
		Counts:  []uint64{1},
	}

	assert.Equal(t, histogramQuantile(hist, -0.5), histogramQuantile(hist, 0.0))
	assert.Equal(t, histogramQuantile(hist, 1.5), histogramQuantile(hist, 1.0))
}

func TestHistogramQuantile_OpenEndedTopBin(t *testing.T) {
	t.Parallel()

	hist := &metrics.Float64Histogram{

		Buckets: []float64{0.0, 0.5, math.Inf(1)},
		Counts:  []uint64{1, 100},
	}

	p99 := histogramQuantile(hist, 0.99)
	assert.Equal(t, 500*time.Millisecond, p99, "open-ended top bin uses last finite boundary")
}

func TestSecondsToDuration_ClampsHugeValues(t *testing.T) {
	t.Parallel()

	clamped := secondsToDuration(1e18)
	assert.Equal(t, time.Duration(math.MaxInt64), clamped, "huge values clamp to int64 max")

	assert.Zero(t, secondsToDuration(0), "zero seconds is zero duration")
	assert.Zero(t, secondsToDuration(-1), "negative seconds is zero duration")
	assert.Zero(t, secondsToDuration(math.NaN()), "NaN is zero duration")
	assert.Zero(t, secondsToDuration(math.Inf(1)), "+Inf is zero duration")
}

func TestHistogramQuantile_SingleBucket(t *testing.T) {
	t.Parallel()

	hist := &metrics.Float64Histogram{
		Buckets: []float64{0.0, 1.0},
		Counts:  []uint64{10},
	}

	assert.Equal(t, time.Second, histogramQuantile(hist, 0.50), "single-bucket histogram returns its upper boundary")
	assert.Equal(t, time.Second, histogramQuantile(hist, 0.99), "single-bucket histogram returns the same value for any q")
}

func TestHistogramQuantile_ExtremeBoundaries(t *testing.T) {
	t.Parallel()

	hist := &metrics.Float64Histogram{
		Buckets: []float64{0.0, 0.1, 0.2, 0.3},
		Counts:  []uint64{10, 10, 10},
	}

	q0 := histogramQuantile(hist, 0)
	q1 := histogramQuantile(hist, 1)

	assert.LessOrEqual(t, q0, q1, "p0 should be at or below p100")
	assert.NotZero(t, q1, "p100 should reach the top bucket boundary")
}

func TestHistogramQuantile_AsymmetricCountLengths(t *testing.T) {
	t.Parallel()

	hist := &metrics.Float64Histogram{
		Buckets: []float64{0.0, 1.0, 2.0, 3.0},
		Counts:  []uint64{1},
	}

	require.NotPanics(t, func() {
		_ = histogramQuantile(hist, 0.50)
	})
}

func TestRuntimeMetricsCollector_LastSnapshotMapPopulated(t *testing.T) {
	t.Parallel()

	collector := newRuntimeMetricsCollector()
	snap := collector.sample(time.Now())

	out := collector.lastSnapshotMap(snap)
	require.NotEmpty(t, out)

	if _, ok := out[metricNameGoroutines]; !ok {
		t.Fatalf("snapshot map missing %s", metricNameGoroutines)
	}

	if _, ok := out[metricNameGCPauses+":p99"]; !ok {
		t.Fatalf("snapshot map missing GC pause p99 key")
	}
}
