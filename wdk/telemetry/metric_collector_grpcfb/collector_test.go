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
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

type sink struct {
	batches []*telemetry_grpcfb.Batch
	mu      sync.Mutex
}

func (s *sink) Ingest(stream telemetry_grpcfb.IngestStream) error {
	for {
		b, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.batches = append(s.batches, b)
		s.mu.Unlock()
	}
	return stream.SendAndClose(&telemetry_grpcfb.IngestAck{OK: true})
}

func (s *sink) metrics() []telemetry_grpcfb.MetricPoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []telemetry_grpcfb.MetricPoint
	for _, b := range s.batches {
		out = append(out, b.Metrics...)
	}
	return out
}

type harness struct {
	srv    *grpc.Server
	conn   *grpc.ClientConn
	client *telemetry_grpcfb.Client
	sink   *sink
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})),
	)
	require.NoError(t, err)

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{
		SiteID:        "s",
		FlushSize:     1 << 20,
		FlushInterval: time.Hour,
	})

	client.Start(context.Background())
	return &harness{srv: srv, conn: conn, client: client, sink: snk}
}

func (h *harness) drain(t *testing.T) []telemetry_grpcfb.MetricPoint {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, h.client.Close(ctx))
	require.NoError(t, h.conn.Close())
	h.srv.Stop()
	return h.sink.metrics()
}

func names(points []telemetry_grpcfb.MetricPoint) map[string]bool {
	out := make(map[string]bool, len(points))
	for _, p := range points {
		out[p.Name] = true
	}
	return out
}

var (
	coreGaugeNames = []string{
		"runtime.go.mem.heap_alloc",
		"runtime.go.mem.heap_sys",
		"runtime.go.mem.heap_idle",
		"runtime.go.mem.heap_inuse",
		"runtime.go.mem.heap_released",
		"runtime.go.mem.heap_objects",
		"runtime.go.mem.stack_inuse",
		"runtime.go.mem.stack_sys",
		"runtime.go.gc.count",
		"runtime.go.gc.cpu_fraction",
		"runtime.go.goroutines",
	}
)

func TestCollectEmitsWellFormedMetrics(t *testing.T) {
	h := newHarness(t)
	r := NewRuntime(h.client, time.Hour, WithClock(clock.NewMockClock(time.Unix(1_700_000_000, 0))))

	r.collect(context.Background())

	points := h.drain(t)
	require.NotEmpty(t, points)

	seen := names(points)
	for _, want := range coreGaugeNames {
		assert.True(t, seen[want], "expected metric %q to be emitted", want)
	}

	wantMs := time.Unix(1_700_000_000, 0).UnixMilli()
	for _, p := range points {
		assert.NotEmpty(t, p.Name, "metric name must not be empty")
		assert.Equal(t, wantMs, p.TimestampMs, "metric %q has wrong timestamp", p.Name)
		assert.Contains(t, []string{unitBytes, unitCount}, p.Unit, "metric %q has unexpected unit", p.Name)
		assert.Contains(t, []string{kindGauge, kindCounter}, p.Kind, "metric %q has unexpected kind", p.Name)
	}
}

func TestCollectEmitsExpectedUnitsAndKinds(t *testing.T) {
	h := newHarness(t)
	r := NewRuntime(h.client, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())
	points := h.drain(t)

	byName := make(map[string]telemetry_grpcfb.MetricPoint, len(points))
	for _, p := range points {
		byName[p.Name] = p
	}

	tests := []struct {
		name string
		unit string
		kind string
	}{
		{"runtime.go.mem.heap_alloc", unitBytes, kindGauge},
		{"runtime.go.mem.heap_objects", unitCount, kindGauge},
		{"runtime.go.gc.count", unitCount, kindCounter},
		{"runtime.go.gc.cpu_fraction", unitCount, kindGauge},
		{"runtime.go.goroutines", unitCount, kindGauge},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, ok := byName[tc.name]
			require.True(t, ok, "metric %q not emitted", tc.name)
			assert.Equal(t, tc.unit, p.Unit)
			assert.Equal(t, tc.kind, p.Kind)
		})
	}
}

type signallingClock struct {
	clock.Clock
	collects chan struct{}
}

func (c *signallingClock) Now() time.Time {
	select {
	case c.collects <- struct{}{}:
	default:
	}
	return c.Clock.Now()
}

func TestRunSamplesImmediatelyAndOnTick(t *testing.T) {
	h := newHarness(t)
	const interval = time.Hour
	mock := clock.NewMockClock(time.Unix(1_700_000_000, 0))
	sc := &signallingClock{Clock: mock, collects: make(chan struct{}, 8)}
	r := NewRuntime(h.client, interval, WithClock(sc))

	ctx, cancel := context.WithCancel(context.Background())

	baseline := mock.TimerCount()
	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	select {
	case <-sc.collects:
	case <-time.After(5 * time.Second):
		cancel()
		<-done
		t.Fatal("Run never took the immediate sample")
	}

	require.True(t, mock.AwaitTimerSetup(baseline, 5*time.Second), "ticker was never installed")

	mock.Advance(interval)
	select {
	case <-sc.collects:
	case <-time.After(5 * time.Second):
		cancel()
		<-done
		t.Fatal("Run never took the tick sample")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}

	points := h.drain(t)
	require.NotEmpty(t, points)
	seen := names(points)
	for _, want := range coreGaugeNames {
		assert.True(t, seen[want], "expected metric %q after run", want)
	}

	var heapAllocCount int
	for _, p := range points {
		if p.Name == "runtime.go.mem.heap_alloc" {
			heapAllocCount++
		}
	}
	assert.GreaterOrEqual(t, heapAllocCount, 2, "expected immediate sample and tick sample")
}

type panicClock struct {
	clock.Clock
}

func (panicClock) Now() time.Time {
	panic("boom")
}

func TestCollectRecoversFromPanic(t *testing.T) {
	h := newHarness(t)
	defer h.drain(t)
	r := NewRuntime(h.client, time.Hour, WithClock(panicClock{Clock: clock.NewMockClock(time.Unix(0, 0))}))

	assert.NotPanics(t, func() { r.collect(context.Background()) },
		"collect must not propagate a panic to the sampling loop")
}

func TestRunSkipsTickAfterContextCancelled(t *testing.T) {
	h := newHarness(t)
	const interval = time.Hour
	mock := clock.NewMockClock(time.Unix(1_700_000_000, 0))
	sc := &signallingClock{Clock: mock, collects: make(chan struct{}, 8)}
	r := NewRuntime(h.client, interval, WithClock(sc))

	ctx, cancel := context.WithCancel(context.Background())

	baseline := mock.TimerCount()
	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	select {
	case <-sc.collects:
	case <-time.After(5 * time.Second):
		cancel()
		<-done
		t.Fatal("Run never took the immediate sample")
	}
	require.True(t, mock.AwaitTimerSetup(baseline, 5*time.Second), "ticker was never installed")

	cancel()
	mock.Advance(interval)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}

	select {
	case <-sc.collects:
		t.Fatal("Run collected after the context was cancelled")
	default:
	}

	h.drain(t)
}

func TestRunReturnsImmediatelyOnNilClient(t *testing.T) {
	r := NewRuntime(nil, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	done := make(chan struct{})
	go func() {
		r.Run(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run with nil client did not return")
	}
}

func TestRunOnNilRuntimeIsSafe(t *testing.T) {
	var r *Runtime
	assert.NotPanics(t, func() { r.Run(context.Background()) })
}

func TestRunStopsWhenContextAlreadyCancelled(t *testing.T) {
	h := newHarness(t)
	defer h.drain(t)
	mock := clock.NewMockClock(time.Unix(0, 0))
	r := NewRuntime(h.client, time.Hour, WithClock(mock))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return for a cancelled context")
	}
}

func TestNewRuntimeDefaultsNonPositiveInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
	}{
		{"zero", 0},
		{"negative", -time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRuntime(nil, tc.interval)
			assert.Equal(t, DefaultInterval, r.interval)
		})
	}
}

func TestNewRuntimeKeepsPositiveInterval(t *testing.T) {
	r := NewRuntime(nil, 42*time.Second)
	assert.Equal(t, 42*time.Second, r.interval)
}

func TestWithClockIgnoresNil(t *testing.T) {
	r := NewRuntime(nil, time.Hour, WithClock(nil))
	assert.NotNil(t, r.clock, "nil clock option must leave a usable default clock")
}

func TestWithClockOverridesDefault(t *testing.T) {
	mock := clock.NewMockClock(time.Unix(123, 0))
	r := NewRuntime(nil, time.Hour, WithClock(mock))
	assert.Equal(t, mock, r.clock)
}

func TestNewRuntimeToleratesNilOption(t *testing.T) {
	assert.NotPanics(t, func() { _ = NewRuntime(nil, time.Hour, nil) })
}

func TestCloseSharedClientIsNoOp(t *testing.T) {
	h := newHarness(t)
	r := NewRuntime(h.client, time.Hour)

	require.NoError(t, r.Close(context.Background()))

	r2 := NewRuntime(h.client, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))
	r2.collect(context.Background())

	points := h.drain(t)
	assert.NotEmpty(t, points, "shared client should still accept metrics after Runtime.Close")
}

func TestCloseOwnedClientClosesIt(t *testing.T) {
	h := newHarness(t)

	r := &Runtime{client: h.client, clock: clock.NewMockClock(time.Unix(0, 0)), interval: time.Hour, ownsClient: true}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, r.Close(ctx))

	require.NoError(t, r.Close(ctx))

	require.NoError(t, h.conn.Close())
	h.srv.Stop()
}

func TestCloseNilRuntimeIsSafe(t *testing.T) {
	var r *Runtime
	assert.NoError(t, r.Close(context.Background()))
}

func TestCloseRuntimeWithNilClientIsSafe(t *testing.T) {
	r := &Runtime{ownsClient: true}
	assert.NoError(t, r.Close(context.Background()))
}

func TestProcThreadsReturnsPositiveOnLinux(t *testing.T) {
	if _, err := os.Stat("/proc/self/status"); err != nil {
		t.Skip("/proc/self/status not available on this platform")
	}
	n, ok := procThreads()
	require.True(t, ok, "procThreads should succeed when /proc/self/status exists")
	assert.Positive(t, n, "a running process must have at least one thread")
}

func TestFdCategoriesNonEmptyOnLinux(t *testing.T) {
	if _, err := os.Stat("/proc/self/fd"); err != nil {
		t.Skip("/proc/self/fd not available on this platform")
	}
	cats := fdCategories()
	require.NotEmpty(t, cats, "a running process must have open file descriptors")
	total := 0
	for cat, n := range cats {
		assert.Contains(t, []string{"socket", "pipe", "anon", "file", "other"}, cat)
		assert.Positive(t, n)
		total += n
	}
	assert.Positive(t, total)
}

func TestFdLimitReturnsOkOnUnix(t *testing.T) {
	lim, ok := fdLimit()
	require.True(t, ok, "fdLimit should be available on unix")
	assert.Positive(t, lim, "the soft RLIMIT_NOFILE should be positive")
}
