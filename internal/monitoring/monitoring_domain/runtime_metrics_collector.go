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
	"strconv"
	"time"
)

const (
	// metricNameHeapObjects is the runtime/metrics path for live heap
	// object bytes (the canonical "heap alloc" replacement).
	metricNameHeapObjects = "/memory/classes/heap/objects:bytes"

	// metricNameHeapFree is the runtime/metrics path for heap memory free
	// for reuse but not returned to the OS.
	metricNameHeapFree = "/memory/classes/heap/free:bytes"

	// metricNameHeapReleased is the runtime/metrics path for heap memory
	// returned to the OS.
	metricNameHeapReleased = "/memory/classes/heap/released:bytes"

	// metricNameHeapStacks is the runtime/metrics path for memory backing
	// goroutine stacks.
	metricNameHeapStacks = "/memory/classes/heap/stacks:bytes"

	// metricNameHeapUnused is the runtime/metrics path for heap memory
	// reserved but neither in use nor free for reuse.
	metricNameHeapUnused = "/memory/classes/heap/unused:bytes"

	// metricNameMemoryTotal is the runtime/metrics path for the total
	// number of bytes obtained from the OS by the runtime.
	metricNameMemoryTotal = "/memory/classes/total:bytes"

	// metricNameAllocsBytes is the runtime/metrics path for cumulative
	// bytes allocated to the heap.
	metricNameAllocsBytes = "/gc/heap/allocs:bytes"

	// metricNameFreesBytes is the runtime/metrics path for cumulative
	// bytes freed from the heap.
	metricNameFreesBytes = "/gc/heap/frees:bytes"

	// metricNameAllocsObjects is the runtime/metrics path for cumulative
	// objects allocated.
	metricNameAllocsObjects = "/gc/heap/allocs:objects"

	// metricNameFreesObjects is the runtime/metrics path for cumulative
	// objects freed.
	metricNameFreesObjects = "/gc/heap/frees:objects"

	// metricNameLiveObjects is the runtime/metrics path for the current
	// live object count on the heap.
	metricNameLiveObjects = "/gc/heap/objects:objects"

	// metricNameGCCycles is the runtime/metrics path for the count of
	// completed automatic GC cycles.
	metricNameGCCycles = "/gc/cycles/automatic:gc-cycles"

	// metricNameGCCyclesForced is the runtime/metrics path for the count
	// of GC cycles forced by the application.
	metricNameGCCyclesForced = "/gc/cycles/forced:gc-cycles"

	// metricNameGCPauses is the runtime/metrics histogram of stop-the-world
	// GC pause durations.
	metricNameGCPauses = "/gc/pauses:seconds"

	// metricNameGCGoGC is the runtime/metrics path for the current GOGC
	// percent setting.
	metricNameGCGoGC = "/gc/gogc:percent"

	// metricNameGCGoMemLimit is the runtime/metrics path for the effective
	// GOMEMLIMIT in bytes.
	metricNameGCGoMemLimit = "/gc/gomemlimit:bytes"

	// metricNameSchedLatencies is the runtime/metrics histogram of
	// goroutine scheduling latencies in seconds.
	metricNameSchedLatencies = "/sched/latencies:seconds"

	// metricNameGoroutines is the runtime/metrics path for the current
	// goroutine count.
	metricNameGoroutines = "/sched/goroutines:goroutines"

	// metricNameGoMaxProcs is the runtime/metrics path for the configured
	// GOMAXPROCS thread count.
	metricNameGoMaxProcs = "/sched/gomaxprocs:threads"

	// metricNameMutexWait is the runtime/metrics path for the cumulative
	// mutex contention wait time.
	metricNameMutexWait = "/sync/mutex/wait/total:seconds"

	// metricNameCPUClassesGC is the runtime/metrics path for cumulative
	// CPU time spent in GC.
	metricNameCPUClassesGC = "/cpu/classes/gc/total:cpu-seconds"

	// metricNameCPUClassesUser is the runtime/metrics path for cumulative
	// CPU time spent in user code.
	metricNameCPUClassesUser = "/cpu/classes/user:cpu-seconds"

	// metricNameCPUClassesTotal is the runtime/metrics path for the
	// cumulative CPU time the runtime has used in total.
	metricNameCPUClassesTotal = "/cpu/classes/total:cpu-seconds"
)

const (
	// quantileP50 is the median quantile extracted from runtime
	// histograms (GC pauses, scheduler latencies) for sidecar metadata.
	quantileP50 = 0.50

	// quantileP95 is the 95th-percentile quantile extracted from runtime
	// histograms.
	quantileP95 = 0.95

	// quantileP99 is the 99th-percentile quantile extracted from runtime
	// histograms.
	quantileP99 = 0.99
)

// nanosPerSecond converts seconds to nanoseconds when computing histogram
// quantile durations.
const nanosPerSecond = 1e9

// requestedRuntimeMetrics is the canonical list of metrics this collector
// samples. The order is preserved across reads so callers can rely on
// indices rather than name lookups inside hot paths.
var requestedRuntimeMetrics = []string{
	metricNameHeapObjects,
	metricNameHeapFree,
	metricNameHeapReleased,
	metricNameHeapStacks,
	metricNameHeapUnused,
	metricNameMemoryTotal,
	metricNameAllocsBytes,
	metricNameFreesBytes,
	metricNameAllocsObjects,
	metricNameFreesObjects,
	metricNameLiveObjects,
	metricNameGCCycles,
	metricNameGCCyclesForced,
	metricNameGCPauses,
	metricNameGCGoGC,
	metricNameGCGoMemLimit,
	metricNameSchedLatencies,
	metricNameGoroutines,
	metricNameGoMaxProcs,
	metricNameMutexWait,
	metricNameCPUClassesGC,
	metricNameCPUClassesUser,
	metricNameCPUClassesTotal,
}

// runtimeMetricsSnapshot is the curated view of /proc-equivalent runtime
// telemetry produced by sampling runtime/metrics once per system collector
// tick. Fields default to zero when the underlying metric is unavailable on
// the running Go version.
type runtimeMetricsSnapshot struct {
	// SampledAt is the monotonic timestamp of the read.
	SampledAt time.Time

	// HeapObjectsBytes is the live heap object footprint in bytes (the
	// canonical "heap alloc" replacement under runtime/metrics).
	HeapObjectsBytes uint64

	// HeapFreeBytes is heap memory free for reuse but not returned to the
	// OS.
	HeapFreeBytes uint64

	// HeapReleasedBytes is heap memory returned to the OS.
	HeapReleasedBytes uint64

	// HeapStacksBytes is memory backing goroutine stacks.
	HeapStacksBytes uint64

	// HeapUnusedBytes is heap memory reserved but neither in use nor
	// free for reuse.
	HeapUnusedBytes uint64

	// TotalMemoryBytes is the total bytes obtained from the OS.
	TotalMemoryBytes uint64

	// AllocsBytes is the cumulative bytes allocated to the heap since
	// process start.
	AllocsBytes uint64

	// FreesBytes is the cumulative bytes freed from the heap since
	// process start.
	FreesBytes uint64

	// AllocsObjects is the cumulative number of heap objects allocated.
	AllocsObjects uint64

	// FreesObjects is the cumulative number of heap objects freed.
	FreesObjects uint64

	// LiveObjects is the post-GC live object count.
	LiveObjects uint64

	// NumGC is the count of completed automatic GC cycles since process
	// start.
	NumGC uint64

	// NumForcedGC is the count of GC cycles forced by the application.
	NumForcedGC uint64

	// GCPauseP50 is the median stop-the-world GC pause derived from the
	// /gc/pauses:seconds histogram. Zero when no samples are present.
	GCPauseP50 time.Duration

	// GCPauseP95 is the 95th-percentile stop-the-world GC pause derived
	// from the /gc/pauses:seconds histogram.
	GCPauseP95 time.Duration

	// GCPauseP99 is the 99th-percentile stop-the-world GC pause derived
	// from the /gc/pauses:seconds histogram.
	GCPauseP99 time.Duration

	// GoGC is the current GOGC percent setting.
	GoGC int64

	// GoMemLimit is the effective GOMEMLIMIT in bytes
	// (math.MaxInt64 when unlimited).
	GoMemLimit int64

	// SchedulerLatencyP50 is the median goroutine scheduling latency
	// derived from /sched/latencies:seconds.
	SchedulerLatencyP50 time.Duration

	// SchedulerLatencyP99 is the 99th-percentile goroutine scheduling
	// latency derived from /sched/latencies:seconds.
	SchedulerLatencyP99 time.Duration

	// Goroutines is the current goroutine count.
	Goroutines int64

	// GoMaxProcs is the configured GOMAXPROCS thread count.
	GoMaxProcs int64

	// MutexWaitTotalSeconds is the cumulative time spent waiting for
	// mutexes across all goroutines. Sample twice and divide the delta by
	// elapsed wall-clock time to derive a contention rate.
	MutexWaitTotalSeconds float64

	// CPUUserSeconds is the cumulative CPU time spent in user code.
	CPUUserSeconds float64

	// CPUGCSeconds is the cumulative CPU time spent in GC.
	CPUGCSeconds float64

	// CPUTotalSeconds is the cumulative CPU time the runtime has used in
	// total.
	CPUTotalSeconds float64
}

// runtimeMetricsCollector samples the curated set of runtime/metrics into a
// snapshot once per tick. It owns a pre-allocated samples slice so the hot
// path performs no allocations.
//
// sample is single-threaded by contract -- the SystemCollector tick goroutine
// drives it. The snapshot it produces is copied into
// SystemCollector.lastSnapshot under the collector's RWMutex; lastSnapshotMap
// reads only the snapshot passed to it (no shared state) and is therefore
// safe to call from any goroutine.
type runtimeMetricsCollector struct {
	// indices maps metric name to position in samples for the metrics that
	// were registered at construction. Metrics not advertised by the running
	// Go version are absent from this map.
	indices map[string]int

	// available reports whether each metric in requestedRuntimeMetrics was
	// found at construction. Used by extract to decide whether to read or
	// leave the snapshot field at zero.
	available map[string]bool

	// samples is reused across reads. metrics.Read writes into Sample.Value
	// in place; nothing else mutates the slice.
	samples []metrics.Sample
}

// newRuntimeMetricsCollector creates a collector configured for the metrics
// available on the running Go version. Metrics absent from metrics.All() are
// silently skipped -- the corresponding snapshot field stays zero on every
// read.
//
// Returns *runtimeMetricsCollector ready to be sampled. The collector is
// allocated once per SystemCollector and reused.
func newRuntimeMetricsCollector() *runtimeMetricsCollector {
	descs := metrics.All()
	advertised := make(map[string]struct{}, len(descs))
	for _, d := range descs {
		advertised[d.Name] = struct{}{}
	}

	indices := make(map[string]int, len(requestedRuntimeMetrics))
	available := make(map[string]bool, len(requestedRuntimeMetrics))
	samples := make([]metrics.Sample, 0, len(requestedRuntimeMetrics))

	for _, name := range requestedRuntimeMetrics {
		if _, ok := advertised[name]; !ok {
			available[name] = false
			continue
		}
		indices[name] = len(samples)
		samples = append(samples, metrics.Sample{Name: name})
		available[name] = true
	}

	return &runtimeMetricsCollector{
		samples:   samples,
		indices:   indices,
		available: available,
	}
}

// sample reads the runtime/metrics view and returns a populated snapshot.
// metrics.Read is lock-free in the runtime and significantly cheaper than
// runtime.ReadMemStats on large heaps because it does not stop the world.
//
// Takes now (time.Time) which is recorded in SampledAt for downstream
// observers.
//
// Returns runtimeMetricsSnapshot containing the populated curated view.
func (c *runtimeMetricsCollector) sample(now time.Time) runtimeMetricsSnapshot {
	if len(c.samples) == 0 {
		return runtimeMetricsSnapshot{SampledAt: now}
	}

	metrics.Read(c.samples)
	return c.extract(now)
}

// extract converts the latest sample slice into a populated snapshot.
//
// Takes now (time.Time) which is stored as the snapshot's SampledAt.
//
// Returns runtimeMetricsSnapshot built from the most recent samples.
func (c *runtimeMetricsCollector) extract(now time.Time) runtimeMetricsSnapshot {
	snap := runtimeMetricsSnapshot{SampledAt: now}

	snap.HeapObjectsBytes = c.uint64(metricNameHeapObjects)
	snap.HeapFreeBytes = c.uint64(metricNameHeapFree)
	snap.HeapReleasedBytes = c.uint64(metricNameHeapReleased)
	snap.HeapStacksBytes = c.uint64(metricNameHeapStacks)
	snap.HeapUnusedBytes = c.uint64(metricNameHeapUnused)
	snap.TotalMemoryBytes = c.uint64(metricNameMemoryTotal)

	snap.AllocsBytes = c.uint64(metricNameAllocsBytes)
	snap.FreesBytes = c.uint64(metricNameFreesBytes)
	snap.AllocsObjects = c.uint64(metricNameAllocsObjects)
	snap.FreesObjects = c.uint64(metricNameFreesObjects)
	snap.LiveObjects = c.uint64(metricNameLiveObjects)
	snap.NumGC = c.uint64(metricNameGCCycles)
	snap.NumForcedGC = c.uint64(metricNameGCCyclesForced)

	if hist := c.histogram(metricNameGCPauses); hist != nil {
		snap.GCPauseP50 = histogramQuantile(hist, quantileP50)
		snap.GCPauseP95 = histogramQuantile(hist, quantileP95)
		snap.GCPauseP99 = histogramQuantile(hist, quantileP99)
	}

	snap.GoGC = c.int64(metricNameGCGoGC)
	snap.GoMemLimit = c.int64(metricNameGCGoMemLimit)

	if hist := c.histogram(metricNameSchedLatencies); hist != nil {
		snap.SchedulerLatencyP50 = histogramQuantile(hist, quantileP50)
		snap.SchedulerLatencyP99 = histogramQuantile(hist, quantileP99)
	}

	snap.Goroutines = c.int64(metricNameGoroutines)
	snap.GoMaxProcs = c.int64(metricNameGoMaxProcs)
	snap.MutexWaitTotalSeconds = c.float64(metricNameMutexWait)

	snap.CPUUserSeconds = c.float64(metricNameCPUClassesUser)
	snap.CPUGCSeconds = c.float64(metricNameCPUClassesGC)
	snap.CPUTotalSeconds = c.float64(metricNameCPUClassesTotal)

	return snap
}

// uint64 returns the named metric's value as uint64, or 0 when the metric is
// unavailable or has a different kind.
//
// Takes name (string) which is the canonical runtime/metrics path.
//
// Returns uint64 value, or 0 when the metric is missing or wrong-kind.
func (c *runtimeMetricsCollector) uint64(name string) uint64 {
	idx, ok := c.indices[name]
	if !ok {
		return 0
	}
	s := c.samples[idx]
	if s.Value.Kind() != metrics.KindUint64 {
		return 0
	}
	return s.Value.Uint64()
}

// int64 returns the named metric's value as int64, accepting either Uint64
// or Float64 sources (the runtime declares /sched/gomaxprocs:threads as
// Uint64 but /gc/gogc:percent as Uint64; defensive against future kind
// changes).
//
// Takes name (string) which is the canonical runtime/metrics path.
//
// Returns int64 value, or 0 when the metric is missing or wrong-kind. The
// uint64 path saturates at math.MaxInt64 to avoid wrap.
func (c *runtimeMetricsCollector) int64(name string) int64 {
	idx, ok := c.indices[name]
	if !ok {
		return 0
	}
	s := c.samples[idx]
	switch s.Value.Kind() {
	case metrics.KindUint64:
		v := s.Value.Uint64()
		if v > uint64(math.MaxInt64) {
			return math.MaxInt64
		}
		return int64(v)
	case metrics.KindFloat64:
		return int64(s.Value.Float64())
	default:
		return 0
	}
}

// float64 returns the named metric's value as float64, or 0 when the metric
// is unavailable or has a different kind.
//
// Takes name (string) which is the canonical runtime/metrics path.
//
// Returns float64 value, or 0 when the metric is missing or wrong-kind.
func (c *runtimeMetricsCollector) float64(name string) float64 {
	idx, ok := c.indices[name]
	if !ok {
		return 0
	}
	s := c.samples[idx]
	if s.Value.Kind() != metrics.KindFloat64 {
		return 0
	}
	return s.Value.Float64()
}

// histogram returns the named metric's value as a Float64Histogram, or nil
// when the metric is unavailable or has a different kind.
//
// Takes name (string) which is the canonical runtime/metrics path.
//
// Returns *metrics.Float64Histogram or nil when the metric is missing or
// wrong-kind.
func (c *runtimeMetricsCollector) histogram(name string) *metrics.Float64Histogram {
	idx, ok := c.indices[name]
	if !ok {
		return nil
	}
	s := c.samples[idx]
	if s.Value.Kind() != metrics.KindFloat64Histogram {
		return nil
	}
	return s.Value.Float64Histogram()
}

// histogramQuantile computes a quantile from a Float64Histogram by walking
// the cumulative distribution.
//
// The histogram's Buckets slice describes open-ended bins via boundary
// values; Counts gives the count per bin. The quantile q must be in
// [0, 1]. For samples falling in the last bin (no upper bound), the
// function returns the value of the last finite boundary; this is
// conservative and under-reports tail values.
//
// Takes hist (*metrics.Float64Histogram) which is the runtime histogram
// being summarised; nil is treated as empty.
// Takes q (float64) which is the quantile in [0, 1]; out-of-range values
// are clamped.
//
// Returns time.Duration which is the bucket boundary at the quantile, or
// zero when the histogram is missing or has no samples.
func histogramQuantile(hist *metrics.Float64Histogram, q float64) time.Duration {
	if hist == nil || len(hist.Counts) == 0 {
		return 0
	}

	var total uint64
	for _, count := range hist.Counts {
		total += count
	}
	if total == 0 {
		return 0
	}

	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}

	target := uint64(float64(total) * q)
	if target == 0 {
		target = 1
	}

	var cumulative uint64
	for index, count := range hist.Counts {
		cumulative += count
		if cumulative < target {
			continue
		}

		boundary := hist.Buckets[index+1]
		if isInfOrNaN(boundary) {
			boundary = hist.Buckets[index]
		}
		return secondsToDuration(boundary)
	}

	return secondsToDuration(hist.Buckets[len(hist.Buckets)-1])
}

// secondsToDuration converts a fractional-seconds float into a time.Duration,
// clamping at MaxInt64 nanoseconds.
//
// Takes seconds (float64) which is the duration in fractional seconds.
//
// Returns time.Duration with overflow clamped to math.MaxInt64.
func secondsToDuration(seconds float64) time.Duration {
	if seconds <= 0 || isInfOrNaN(seconds) {
		return 0
	}
	nanos := seconds * nanosPerSecond

	if nanos >= float64(math.MaxInt64) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(nanos)
}

// isInfOrNaN reports whether the supplied float is +Inf, -Inf, or NaN.
// Uses math.IsNaN / math.IsInf so the inputs are checked via well-known
// helpers rather than the f != f idiom, which can confuse linters.
//
// Takes f (float64) which is the value under inspection.
//
// Returns bool which is true when f is non-finite.
func isInfOrNaN(f float64) bool {
	return math.IsNaN(f) || math.IsInf(f, 0)
}

// lastSnapshotMap returns a curated map of the most recent runtime metrics
// suitable for embedding in capture sidecar metadata. Histograms are
// summarised as p50/p95/p99 to keep the JSON small; counters are reported
// verbatim.
//
// Takes snap (runtimeMetricsSnapshot) which is the snapshot to project
// into a JSON-friendly map.
//
// Returns map[string]any keyed by canonical runtime/metrics names plus the
// derived percentile keys. Empty when the collector has not been sampled.
func (*runtimeMetricsCollector) lastSnapshotMap(snap runtimeMetricsSnapshot) map[string]any {
	out := map[string]any{
		metricNameHeapObjects:    snap.HeapObjectsBytes,
		metricNameHeapFree:       snap.HeapFreeBytes,
		metricNameHeapReleased:   snap.HeapReleasedBytes,
		metricNameHeapStacks:     snap.HeapStacksBytes,
		metricNameHeapUnused:     snap.HeapUnusedBytes,
		metricNameMemoryTotal:    snap.TotalMemoryBytes,
		metricNameAllocsBytes:    snap.AllocsBytes,
		metricNameFreesBytes:     snap.FreesBytes,
		metricNameAllocsObjects:  snap.AllocsObjects,
		metricNameFreesObjects:   snap.FreesObjects,
		metricNameLiveObjects:    snap.LiveObjects,
		metricNameGCCycles:       snap.NumGC,
		metricNameGCCyclesForced: snap.NumForcedGC,
		metricNameGCGoGC:         snap.GoGC,
		metricNameGCGoMemLimit:   snap.GoMemLimit,
		metricNameGoroutines:     snap.Goroutines,
		metricNameGoMaxProcs:     snap.GoMaxProcs,
		metricNameMutexWait:      strconv.FormatFloat(snap.MutexWaitTotalSeconds, 'f', 6, 64),
		metricNameCPUClassesUser: strconv.FormatFloat(snap.CPUUserSeconds, 'f', 6, 64),
		metricNameCPUClassesGC:   strconv.FormatFloat(snap.CPUGCSeconds, 'f', 6, 64),
	}

	out[metricNameGCPauses+":p50"] = snap.GCPauseP50.String()
	out[metricNameGCPauses+":p95"] = snap.GCPauseP95.String()
	out[metricNameGCPauses+":p99"] = snap.GCPauseP99.String()
	out[metricNameSchedLatencies+":p50"] = snap.SchedulerLatencyP50.String()
	out[metricNameSchedLatencies+":p99"] = snap.SchedulerLatencyP99.String()

	return out
}
