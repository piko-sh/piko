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

package metric_collector_grpcfb

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	// DefaultInterval is the sampling cadence when none is supplied.
	DefaultInterval = 15 * time.Second

	// unitBytes is the metric unit marking a byte-valued gauge for the dashboards.
	unitBytes = "bytes"

	// unitCount is the metric unit marking a dimensionless count for the dashboards.
	unitCount = "1"

	// kindGauge is the metric kind marking an instantaneous gauge sample.
	kindGauge = "gauge"

	// kindCounter is the metric kind marking a monotonic counter sample.
	kindCounter = "counter"
)

// Runtime samples process/runtime gauges and forwards them to a telemetry client.
type Runtime struct {
	// client is the telemetry transport that the sampled gauges are enqueued onto.
	client *telemetry_grpcfb.Client

	// clock supplies timestamps and the sampling ticker for deterministic testing.
	clock clock.Clock

	// interval is the cadence between successive samples.
	interval time.Duration

	// ownsClient reports whether this Runtime built and must close the client.
	ownsClient bool
}

// Option configures a Runtime.
type Option func(*Runtime)

// WithClock injects the clock used for timestamps and the sampling ticker, so the loop is
// deterministically testable. Defaults to the real system clock.
//
// Takes c (clock.Clock) which is the clock to drive timestamps and the ticker.
//
// Returns Option which sets the supplied clock on a Runtime.
func WithClock(c clock.Clock) Option {
	return func(r *Runtime) {
		if c != nil {
			r.clock = c
		}
	}
}

// NewRuntime wraps a shared telemetry client. The caller owns the client's lifecycle; Run
// launches the sampling loop and returns when its context is cancelled.
//
// Takes client (*telemetry_grpcfb.Client) which is the shared telemetry transport.
// Takes interval (time.Duration) which is the sampling cadence, defaulted when not
// positive.
// Takes opts (...Option) which adjust the Runtime before use.
//
// Returns *Runtime which samples runtime gauges over the shared client.
func NewRuntime(client *telemetry_grpcfb.Client, interval time.Duration, opts ...Option) *Runtime {
	if interval <= 0 {
		interval = DefaultInterval
	}
	r := &Runtime{client: client, clock: clock.RealClock(), interval: interval}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

// Run samples immediately, then on each interval tick, until ctx is cancelled. It is
// non-blocking on the telemetry client (Add* enqueues, lossy by design).
func (r *Runtime) Run(ctx context.Context) {
	if r == nil || r.client == nil {
		return
	}
	r.collect(ctx)
	t := r.clock.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C():
			if ctx.Err() != nil {
				return
			}
			r.collect(ctx)
		}
	}
}

// Close releases the underlying telemetry client when this Runtime owns it (i.e. it was
// built via Dial).
//
// It is a no-op for the shared-client case (NewRuntime), where the caller owns the
// client's lifecycle. Stop Run() by cancelling its context first.
//
// Returns error which is non-nil when closing an owned client fails.
func (r *Runtime) Close(ctx context.Context) error {
	if r != nil && r.ownsClient && r.client != nil {
		return r.client.Close(ctx)
	}
	return nil
}

// collect reads one snapshot of runtime + process stats and emits a MetricPoint per
// gauge.
func (r *Runtime) collect(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	defer func() {
		if rec := recover(); rec != nil {
			l.Error("metric collector recovered from panic", logger_domain.String("panic", fmt.Sprint(rec)))
		}
	}()
	nowMs := r.clock.Now().UnixMilli()
	emit := func(name string, val float64, unit, kind string, labels ...telemetry_grpcfb.KV) {
		r.client.AddMetric(ctx, telemetry_grpcfb.MetricPoint{
			Name:        name,
			Value:       val,
			Unit:        unit,
			Kind:        kind,
			TimestampMs: nowMs,
			Labels:      labels,
		})
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	emit("runtime.go.mem.heap_alloc", float64(ms.HeapAlloc), unitBytes, kindGauge)
	emit("runtime.go.mem.heap_sys", float64(ms.HeapSys), unitBytes, kindGauge)
	emit("runtime.go.mem.heap_idle", float64(ms.HeapIdle), unitBytes, kindGauge)
	emit("runtime.go.mem.heap_inuse", float64(ms.HeapInuse), unitBytes, kindGauge)
	emit("runtime.go.mem.heap_released", float64(ms.HeapReleased), unitBytes, kindGauge)
	emit("runtime.go.mem.heap_objects", float64(ms.HeapObjects), unitCount, kindGauge)
	emit("runtime.go.mem.stack_inuse", float64(ms.StackInuse), unitBytes, kindGauge)
	emit("runtime.go.mem.stack_sys", float64(ms.StackSys), unitBytes, kindGauge)
	emit("runtime.go.gc.count", float64(ms.NumGC), unitCount, kindCounter)
	emit("runtime.go.gc.cpu_fraction", ms.GCCPUFraction, unitCount, kindGauge)
	emit("runtime.go.goroutines", float64(runtime.NumGoroutine()), unitCount, kindGauge)

	if threads, ok := procThreads(); ok {
		emit("process.threads", float64(threads), unitCount, kindGauge)
	}
	for cat, n := range fdCategories() {
		emit("process.open_fds", float64(n), unitCount, kindGauge, telemetry_grpcfb.KV{Key: "category", Value: cat})
	}
	if lim, ok := fdLimit(); ok {
		emit("process.max_fds", float64(lim), unitCount, kindGauge)
	}
}

// fdCategories counts this process's open file descriptors by kind (socket/pipe/anon/
// file/other) from /proc/self/fd.
//
// Returns map[string]int which counts descriptors per kind, empty without /proc.
func fdCategories() map[string]int {
	out := map[string]int{}
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return out
	}
	for _, e := range entries {
		target, err := os.Readlink("/proc/self/fd/" + e.Name())
		cat := "other"
		switch {
		case err != nil:
			cat = "other"
		case strings.HasPrefix(target, "socket:"):
			cat = "socket"
		case strings.HasPrefix(target, "pipe:"):
			cat = "pipe"
		case strings.HasPrefix(target, "anon_inode:"):
			cat = "anon"
		case strings.HasPrefix(target, "/"):
			cat = "file"
		}
		out[cat]++
	}
	return out
}

// fdLimit (the soft RLIMIT_NOFILE) is platform-specific: see fdlimit_unix.go and
// fdlimit_other.go (the syscall.Rlimit API is Unix-only, so it must be build-tagged out
// of the Windows build).

// procThreads reads the OS thread count from /proc/self/status.
//
// Returns int which is the thread count, zero when unavailable.
// Returns bool which is false when /proc/self/status cannot be read or parsed.
func procThreads() (int, bool) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, false
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		rest, ok := strings.CutPrefix(sc.Text(), "Threads:")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return 0, false
		}
		return n, true
	}

	if sc.Err() != nil {
		return 0, false
	}
	return 0, false
}

// Dial creates a runtime collector backed by its own telemetry client and connection.
//
// Takes target (string) which is the telemetry endpoint address to dial.
// Takes config (telemetry_grpcfb.Config) which configures the new telemetry client.
// Takes interval (time.Duration) which is the sampling cadence, defaulted when not
// positive.
// Takes dialOpts (...grpc.DialOption) which tune the gRPC connection.
//
// Returns *Runtime which owns and samples over the dialled client.
// Returns error which is non-nil when dialling the telemetry endpoint fails.
func Dial(target string, config telemetry_grpcfb.Config, interval time.Duration, dialOpts ...grpc.DialOption) (*Runtime, error) {
	client, err := telemetry_grpcfb.Dial(target, config, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("metric_collector_grpcfb: dial %q: %w", target, err)
	}
	if interval <= 0 {
		interval = DefaultInterval
	}
	return &Runtime{client: client, clock: clock.RealClock(), interval: interval, ownsClient: true}, nil
}
