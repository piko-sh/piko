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

package cli

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"

	"piko.sh/piko/wdk/safedisk"
)

const (
	// loadMaxIdleConnDuration is the maximum idle connection
	// lifetime for the HTTP client.
	loadMaxIdleConnDuration = 90 * time.Second

	// loadReadTimeout is the HTTP read timeout.
	loadReadTimeout = 30 * time.Second

	// loadWriteTimeout is the HTTP write timeout.
	loadWriteTimeout = 30 * time.Second

	// loadLatencyChBuffer is the buffer size for the latency sampling channel.
	loadLatencyChBuffer = 4096

	// loadSampleRate is the sampling rate for latencies in
	// continuous mode (every Nth request).
	loadSampleRate = 100

	// loadLiveMetricsWindow is the ring buffer size for latency
	// percentile computation.
	loadLiveMetricsWindow = 4000

	// loadMillisPerSec converts seconds to milliseconds.
	loadMillisPerSec = 1000

	// loadP50 is the 50th percentile threshold.
	loadP50 = 50

	// loadP80 is the 80th percentile threshold.
	loadP80 = 80

	// loadP99 is the 99th percentile threshold.
	loadP99 = 99

	// loadP100 is the 100th percentile threshold.
	loadP100 = 100

	// loadErrorLogPerms is the file permission for the error log.
	loadErrorLogPerms = 0o640

	// httpStatusOKMin is the lower bound (inclusive) of
	// successful HTTP status codes.
	httpStatusOKMin = 200

	// httpStatusOKMax is the upper bound (exclusive) of
	// successful HTTP status codes.
	httpStatusOKMax = 300
)

// metricsMessage carries a snapshot of live load metrics for the TUI.
type metricsMessage struct {
	// rps is the current requests per second.
	rps float64

	// meanLatencyMs is the cumulative mean latency in
	// milliseconds since the test started.
	meanLatencyMs float64

	// total is the cumulative number of completed requests.
	total int64

	// failed is the cumulative number of failed requests.
	failed int64

	// bytesReceived is the cumulative response bytes read.
	bytesReceived int64

	// Latency percentiles in milliseconds, computed from a sliding window
	// of sampled requests.
	p50Ms float64

	// p80Ms is the 80th percentile latency in milliseconds.
	p80Ms float64

	// p99Ms is the 99th percentile latency in milliseconds.
	p99Ms float64

	// p100Ms is the 100th percentile latency in milliseconds.
	p100Ms float64
}

// loadErrorRecord is a single error entry written to the JSONL error log.
type loadErrorRecord struct {
	// Time is the RFC 3339 timestamp of the error.
	Time string `json:"time"`

	// Phase identifies the profiling phase (e.g. "baseline", "cpu").
	Phase string `json:"phase"`

	// Kind is "transport" for network errors or "status" for non-2xx responses.
	Kind string `json:"kind"`

	// Error is the error message (only set when Kind is "transport").
	Error string `json:"error,omitempty"`

	// StatusCode is the HTTP status code (only set when Kind is "status").
	StatusCode int `json:"status_code,omitempty"`
}

// loadConfig configures the HTTP load generator.
type loadConfig struct {
	// headers are HTTP headers sent with every request.
	headers map[string]string

	// metricsCh receives periodic live metrics snapshots when non-nil and
	// metricsInterval is positive.
	metricsCh chan<- metricsMessage

	// errorCh receives error records for failed requests when non-nil.
	// Sends are non-blocking; records are silently dropped if the
	// channel buffer is full.
	errorCh chan<- loadErrorRecord

	// url is the target URL to send requests to.
	url string

	// phase identifies the current profiling phase for error records.
	phase string

	// concurrency is the number of concurrent HTTP workers.
	concurrency int

	// maxRequests is the total number of requests to send. Zero means
	// unlimited (run until the context is cancelled).
	maxRequests int

	// metricsInterval controls how often live metrics are emitted. Zero
	// disables live metrics.
	metricsInterval time.Duration
}

// loadResult holds the aggregated results of a load test run.
type loadResult struct {
	// latencies is a sorted slice of per-request latencies. In baseline mode
	// every request is recorded; in continuous mode latencies are sampled.
	latencies []time.Duration

	// totalRequests is the number of requests that completed (success or failure).
	totalRequests int64

	// failedRequests is the number of requests that returned a non-2xx status
	// or encountered a transport error.
	failedRequests int64

	// duration is the wall-clock time the load test ran for.
	duration time.Duration

	// bytesReceived is the total response body bytes read.
	bytesReceived int64
}

// requestsPerSecond returns the throughput as requests per second.
//
// Returns float64 which is the requests per second rate.
func (r *loadResult) requestsPerSecond() float64 {
	if r.duration == 0 {
		return 0
	}
	return float64(r.totalRequests) / r.duration.Seconds()
}

// percentile returns the latency at the given percentile (0-100).
//
// Takes p (float64) which is the percentile value from 0 to 100.
//
// Returns time.Duration which is the latency at that percentile.
func (r *loadResult) percentile(p float64) time.Duration {
	if len(r.latencies) == 0 {
		return 0
	}
	index := int(p / 100 * float64(len(r.latencies)))
	if index >= len(r.latencies) {
		index = len(r.latencies) - 1
	}
	return r.latencies[index]
}

// meanLatency returns the arithmetic mean of the recorded latencies.
//
// Returns time.Duration which is the mean latency, or zero if empty.
func (r *loadResult) meanLatency() time.Duration {
	if len(r.latencies) == 0 {
		return 0
	}
	var total time.Duration
	for _, l := range r.latencies {
		total += l
	}
	return total / time.Duration(len(r.latencies))
}

// loadWorkerState holds the shared mutable state accessed by load
// worker goroutines.
type loadWorkerState struct {
	// completedCount is the total number of requests that
	// have finished, whether successful or not.
	completedCount atomic.Int64

	// failedCount is the number of requests that returned a
	// non-2xx status or encountered a transport error.
	failedCount atomic.Int64

	// bytesCount is the cumulative response body bytes
	// received across all workers.
	bytesCount atomic.Int64

	// sampleCounter is a monotonically increasing counter
	// used to decide which requests are sampled for latency.
	sampleCounter atomic.Int64

	// remaining tracks how many requests are left to send
	// in baseline (fixed-count) mode.
	remaining atomic.Int64
}

// workerResult collects latencies recorded by a single load worker.
type workerResult struct {
	// latencies holds the per-request latencies recorded by this worker.
	latencies []time.Duration
}

// liveMetricsParams groups the parameters for emitLiveMetrics.
type liveMetricsParams struct {
	// completed tracks the total completed requests.
	completed *atomic.Int64

	// failed tracks the total failed requests.
	failed *atomic.Int64

	// bytes tracks the total received bytes.
	bytes *atomic.Int64

	// metricsChannel receives periodic metrics snapshots.
	metricsChannel chan<- metricsMessage

	// latencyCh provides sampled latencies for percentile computation.
	latencyCh <-chan time.Duration

	// start is the load test start time.
	start time.Time

	// interval controls how often metrics are emitted.
	interval time.Duration
}

// latencyWindow manages a ring buffer of latency samples and provides
// sorted percentile computation.
type latencyWindow struct {
	// ring is the fixed-size circular buffer of latency
	// samples.
	ring []time.Duration

	// sortBuf is a reusable scratch buffer for sorting the
	// current window contents without mutating ring.
	sortBuf []time.Duration

	// writePos is the next write index into ring, wrapping
	// modulo the buffer capacity.
	writePos int

	// filled is the number of valid samples currently held
	// in ring, capped at the buffer capacity.
	filled int
}

// drain reads all available samples from latencyChannel into the ring
// buffer.
//
// Takes latencyChannel (<-chan time.Duration) which provides latency
// samples.
func (w *latencyWindow) drain(latencyChannel <-chan time.Duration) {
	drainLatencyRing(latencyChannel, w.ring, &w.writePos, &w.filled)
}

// percentiles returns latency percentiles in milliseconds from the current
// window contents.
//
// Returns p50 (float64) which is the 50th percentile latency in milliseconds.
// Returns p80 (float64) which is the 80th percentile latency in milliseconds.
// Returns p99 (float64) which is the 99th percentile latency in milliseconds.
// Returns p100 (float64) which is the 100th percentile latency in milliseconds.
func (w *latencyWindow) percentiles() (p50, p80, p99, p100 float64) {
	if w.filled == 0 {
		return 0, 0, 0, 0
	}
	w.sortBuf = w.sortBuf[:w.filled]
	copy(w.sortBuf, w.ring[:w.filled])
	slices.Sort(w.sortBuf)
	return latencyPercentileMs(w.sortBuf, loadP50),
		latencyPercentileMs(w.sortBuf, loadP80),
		latencyPercentileMs(w.sortBuf, loadP99),
		latencyPercentileMs(w.sortBuf, loadP100)
}

// newLoadClient creates a fasthttp client configured for the
// load test.
//
// Takes concurrency (int) which is the maximum number of
// concurrent connections per host.
//
// Returns *fasthttp.Client which is the configured HTTP client.
func newLoadClient(concurrency int) *fasthttp.Client {
	return &fasthttp.Client{
		MaxConnsPerHost:               concurrency,
		MaxIdleConnDuration:           loadMaxIdleConnDuration,
		ReadTimeout:                   loadReadTimeout,
		WriteTimeout:                  loadWriteTimeout,
		DisableHeaderNamesNormalizing: true,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // local dev server
		},
	}
}

// newLoadTemplate creates the template request that workers
// copy per-iteration.
//
// Takes loadConfiguration (loadConfig) which provides the target URL and
// headers.
//
// Returns *fasthttp.Request which is the reusable template.
func newLoadTemplate(loadConfiguration loadConfig) *fasthttp.Request {
	templateReq := fasthttp.AcquireRequest()
	templateReq.SetRequestURI(loadConfiguration.url)
	templateReq.Header.SetMethod(fasthttp.MethodGet)
	for k, v := range loadConfiguration.headers {
		templateReq.Header.Set(k, v)
	}
	_ = templateReq.Header.Header()
	return templateReq
}

// collectWorkerLatencies merges and sorts per-worker latency
// slices.
//
// Takes results ([]workerResult) which holds per-worker
// latency recordings.
//
// Returns []time.Duration which is the merged and sorted
// latency slice.
func collectWorkerLatencies(results []workerResult) []time.Duration {
	total := 0
	for _, wr := range results {
		total += len(wr.latencies)
	}
	latencies := make([]time.Duration, 0, total)
	for _, wr := range results {
		latencies = append(latencies, wr.latencies...)
	}
	slices.Sort(latencies)
	return latencies
}

// reportTransportError sends a transport error record to the
// error channel if it is non-nil, without blocking.
//
// Takes loadConfiguration (loadConfig) which provides the error channel and
// current phase.
// Takes err (error) which is the transport error to report.
func reportTransportError(loadConfiguration loadConfig, err error) {
	if loadConfiguration.errorCh == nil {
		return
	}
	select {
	case loadConfiguration.errorCh <- loadErrorRecord{
		Time:  time.Now().Format(time.RFC3339Nano),
		Phase: loadConfiguration.phase,
		Kind:  "transport",
		Error: err.Error(),
	}:
	default:
	}
}

// reportStatusError sends a non-2xx status error record to the
// error channel if it is non-nil, without blocking.
//
// Takes loadConfiguration (loadConfig) which provides the error channel and
// current phase.
// Takes statusCode (int) which is the non-2xx HTTP status code.
func reportStatusError(loadConfiguration loadConfig, statusCode int) {
	if loadConfiguration.errorCh == nil {
		return
	}
	select {
	case loadConfiguration.errorCh <- loadErrorRecord{
		Time:       time.Now().Format(time.RFC3339Nano),
		Phase:      loadConfiguration.phase,
		Kind:       "status",
		StatusCode: statusCode,
	}:
	default:
	}
}

// runLoad executes an HTTP load test according to loadConfiguration using
// fasthttp for minimal per-request allocation.
//
// Takes loadConfiguration (loadConfig) which configures the load generator.
//
// Returns *loadResult which contains the aggregated test results.
//
// Spawns loadConfiguration.concurrency worker goroutines that send HTTP
// requests in parallel. An additional goroutine emits live metrics when
// loadConfiguration.metricsInterval is positive. All goroutines finish before
// the function returns.
func runLoad(ctx context.Context, loadConfiguration loadConfig) *loadResult {
	client := newLoadClient(loadConfiguration.concurrency)
	templateReq := newLoadTemplate(loadConfiguration)
	defer fasthttp.ReleaseRequest(templateReq)

	state := &loadWorkerState{}
	baseline := loadConfiguration.maxRequests > 0
	if baseline {
		state.remaining.Store(int64(loadConfiguration.maxRequests))
	}

	start := time.Now()

	metricsCtx, metricsCancel := context.WithCancelCause(ctx)
	defer metricsCancel(errors.New("load test completed"))

	var latencyCh chan time.Duration
	if loadConfiguration.metricsInterval > 0 && loadConfiguration.metricsCh != nil {
		latencyCh = make(chan time.Duration, loadLatencyChBuffer)
		go emitLiveMetrics(metricsCtx, liveMetricsParams{
			metricsChannel: loadConfiguration.metricsCh,
			interval:       loadConfiguration.metricsInterval,
			start:          start,
			completed:      &state.completedCount,
			failed:         &state.failedCount,
			bytes:          &state.bytesCount,
			latencyCh:      latencyCh,
		})
	}

	results := make([]workerResult, loadConfiguration.concurrency)
	var wg sync.WaitGroup
	for i := range loadConfiguration.concurrency {
		index := i
		wg.Go(func() {
			results[index].latencies = runLoadWorker(
				ctx, client, templateReq, loadConfiguration, state, baseline, latencyCh,
			)
		})
	}
	wg.Wait()
	duration := time.Since(start)

	return &loadResult{
		totalRequests:  state.completedCount.Load(),
		failedRequests: state.failedCount.Load(),
		duration:       duration,
		latencies:      collectWorkerLatencies(results),
		bytesReceived:  state.bytesCount.Load(),
	}
}

// runLoadWorker is the per-goroutine request loop.
//
// Takes client (*fasthttp.Client) which sends requests.
// Takes templateReq (*fasthttp.Request) which is copied for
// each iteration.
// Takes loadConfiguration (loadConfig) which configures request behaviour.
// Takes state (*loadWorkerState) which holds shared counters.
// Takes baseline (bool) which, when true, limits the worker
// to a fixed number of requests.
// Takes latencyCh (chan time.Duration) which receives sampled
// latencies for live metrics.
//
// Returns []time.Duration which holds the latencies recorded
// by this worker.
func runLoadWorker(
	ctx context.Context,
	client *fasthttp.Client,
	templateReq *fasthttp.Request,
	loadConfiguration loadConfig,
	state *loadWorkerState,
	baseline bool,
	latencyCh chan time.Duration,
) []time.Duration {
	var localLatencies []time.Duration
	if baseline {
		perWorker := loadConfiguration.maxRequests / loadConfiguration.concurrency
		localLatencies = make([]time.Duration, 0, perWorker+1)
	}

	for !baseline || state.remaining.Add(-1) >= 0 {
		select {
		case <-ctx.Done():
			return localLatencies
		default:
		}

		shouldTime := baseline || state.sampleCounter.Add(1)%loadSampleRate == 0
		elapsed, ok := executeAndTimeRequest(client, templateReq, loadConfiguration, state, shouldTime)
		if ok && shouldTime {
			localLatencies = append(localLatencies, elapsed)
			trySendLatency(latencyCh, elapsed)
		}
	}
	return localLatencies
}

// executeAndTimeRequest sends a single request and returns its
// elapsed time if shouldTime is true.
//
// Takes client (*fasthttp.Client) which sends the request.
// Takes templateReq (*fasthttp.Request) which is the request
// template.
// Takes loadConfiguration (loadConfig) which configures error reporting.
// Takes state (*loadWorkerState) which holds shared counters.
// Takes shouldTime (bool) which, when true, measures the
// request latency.
//
// Returns time.Duration which is the elapsed time, or zero on
// failure.
// Returns bool which is true when the request succeeded.
func executeAndTimeRequest(
	client *fasthttp.Client,
	templateReq *fasthttp.Request,
	loadConfiguration loadConfig,
	state *loadWorkerState,
	shouldTime bool,
) (time.Duration, bool) {
	var reqStart time.Time
	if shouldTime {
		reqStart = time.Now()
	}

	elapsed, ok := executeLoadRequest(client, templateReq, loadConfiguration, state)
	if !ok {
		return 0, false
	}

	if shouldTime && elapsed == 0 {
		elapsed = time.Since(reqStart)
	}
	return elapsed, ok
}

// trySendLatency attempts a non-blocking send of a latency
// sample.
//
// Takes latencyChannel (chan time.Duration) which receives the sample.
// Takes d (time.Duration) which is the latency to send.
func trySendLatency(latencyChannel chan time.Duration, d time.Duration) {
	if latencyChannel == nil {
		return
	}
	select {
	case latencyChannel <- d:
	default:
	}
}

// executeLoadRequest sends a single HTTP request using the
// template, updates counters, and reports errors.
//
// Takes client (*fasthttp.Client) which sends the request.
// Takes templateReq (*fasthttp.Request) which is the request
// template.
// Takes loadConfiguration (loadConfig) which configures error reporting.
// Takes state (*loadWorkerState) which holds shared counters.
//
// Returns time.Duration which is the elapsed time, or zero on
// transport error.
// Returns bool which is true when the request completed
// without a transport error.
func executeLoadRequest(
	client *fasthttp.Client,
	templateReq *fasthttp.Request,
	loadConfiguration loadConfig,
	state *loadWorkerState,
) (time.Duration, bool) {
	reqStart := time.Now()

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	templateReq.CopyTo(request)

	err := client.Do(request, response)
	if err != nil {
		fasthttp.ReleaseRequest(request)
		fasthttp.ReleaseResponse(response)
		state.failedCount.Add(1)
		state.completedCount.Add(1)
		reportTransportError(loadConfiguration, err)
		return 0, false
	}

	state.bytesCount.Add(int64(len(response.Body())))
	state.completedCount.Add(1)

	statusCode := response.StatusCode()
	if statusCode < httpStatusOKMin || statusCode >= httpStatusOKMax {
		state.failedCount.Add(1)
		reportStatusError(loadConfiguration, statusCode)
	}

	fasthttp.ReleaseRequest(request)
	fasthttp.ReleaseResponse(response)

	return time.Since(reqStart), true
}

// newLatencyWindow creates a latency window with the given
// capacity.
//
// Takes capacity (int) which is the maximum number of samples
// the ring buffer can hold.
//
// Returns *latencyWindow which is the initialised window.
func newLatencyWindow(capacity int) *latencyWindow {
	return &latencyWindow{
		ring:    make([]time.Duration, capacity),
		sortBuf: make([]time.Duration, 0, capacity),
	}
}

// emitLiveMetrics periodically reads the atomic counters and sends a
// metricsMessage on params.metricsChannel.
//
// Takes params (liveMetricsParams) which groups all required state.
func emitLiveMetrics(ctx context.Context, params liveMetricsParams) {
	ticker := time.NewTicker(params.interval)
	defer ticker.Stop()

	var previousCompleted int64
	previousTime := params.start
	window := newLatencyWindow(loadLiveMetricsWindow)

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			window.drain(params.latencyCh)
			message := buildMetricsSnapshot(params, window, now, previousTime, previousCompleted)
			select {
			case params.metricsChannel <- message:
			default:
			}
			previousCompleted = message.total
			previousTime = now
		}
	}
}

// buildMetricsSnapshot computes a single metricsMessage from the
// current counter values and latency window.
//
// Takes params (liveMetricsParams) which provides the atomic
// counters and start time.
// Takes window (*latencyWindow) which holds the sampled
// latencies.
// Takes now (time.Time) which is the current tick time.
// Takes previousTime (time.Time) which is the previous tick time.
// Takes previousCompleted (int64) which is the completed count at
// the previous tick.
//
// Returns metricsMessage which is the computed snapshot.
func buildMetricsSnapshot(
	params liveMetricsParams,
	window *latencyWindow,
	now, previousTime time.Time,
	previousCompleted int64,
) metricsMessage {
	curCompleted := params.completed.Load()
	dt := now.Sub(previousTime).Seconds()

	var rps float64
	if dt > 0 {
		rps = float64(curCompleted-previousCompleted) / dt
	}

	totalDur := now.Sub(params.start).Seconds()
	var meanMs float64
	if curCompleted > 0 && totalDur > 0 {
		meanMs = (totalDur / float64(curCompleted)) * loadMillisPerSec
	}

	p50, p80, p99, p100 := window.percentiles()

	return metricsMessage{
		rps:           rps,
		meanLatencyMs: meanMs,
		total:         curCompleted,
		failed:        params.failed.Load(),
		bytesReceived: params.bytes.Load(),
		p50Ms:         p50,
		p80Ms:         p80,
		p99Ms:         p99,
		p100Ms:        p100,
	}
}

// drainLatencyRing reads all available latency samples from the
// channel into the ring buffer.
//
// Takes latencyCh (<-chan time.Duration) which provides
// latency samples.
// Takes ring ([]time.Duration) which is the circular buffer.
// Takes writePos (*int) which tracks the next write index.
// Takes filled (*int) which tracks the number of valid
// samples in ring.
func drainLatencyRing(latencyCh <-chan time.Duration, ring []time.Duration, writePos, filled *int) {
	if latencyCh == nil {
		return
	}
	for {
		select {
		case d := <-latencyCh:
			ring[*writePos%loadLiveMetricsWindow] = d
			*writePos++
			if *filled < loadLiveMetricsWindow {
				*filled++
			}
		default:
			return
		}
	}
}

// latencyPercentileMs returns the percentile value in milliseconds
// from a sorted slice of durations.
//
// Takes sorted ([]time.Duration) which is the sorted latency samples.
// Takes p (float64) which is the percentile value from 0 to 100.
//
// Returns float64 which is the latency in milliseconds, or 0 if
// the slice is empty.
func latencyPercentileMs(sorted []time.Duration, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(p / 100 * float64(len(sorted)))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return float64(sorted[index]) / float64(time.Millisecond)
}

// writeErrorLog drains errorChannel and writes each record as a JSON line to
// errors.jsonl in the given sandbox. It returns when errorChannel is closed.
//
// Takes errorChannel (<-chan loadErrorRecord) which provides error records.
// Takes sandbox (safedisk.Sandbox) which scopes file writes.
//
// Returns error when the file cannot be created or written.
func writeErrorLog(errorChannel <-chan loadErrorRecord, sandbox safedisk.Sandbox) error {
	f, err := sandbox.OpenFile("errors.jsonl", os.O_WRONLY|os.O_CREATE|os.O_APPEND, loadErrorLogPerms)
	if err != nil {
		for range errorChannel { //nolint:revive // drain channel
		}
		return err
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)

	for record := range errorChannel {
		_ = enc.Encode(record)
	}

	return w.Flush()
}
