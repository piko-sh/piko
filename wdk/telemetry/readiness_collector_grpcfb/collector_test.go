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

package readiness_collector_grpcfb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/telemetry/readiness"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

type fakeProbe struct {
	snapshot readiness.Snapshot
	calls    int
	panics   bool
	mu       sync.Mutex
}

func (p *fakeProbe) CheckReadiness(context.Context) readiness.Snapshot {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	if p.panics {
		panic("boom")
	}
	return p.snapshot
}

func (p *fakeProbe) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func sampleSnapshot() readiness.Snapshot {
	return readiness.Snapshot{
		Name:     "readiness",
		State:    readiness.StateHealthy,
		Message:  "",
		Duration: "3ms",
		Dependencies: []readiness.Dependency{
			{Name: "RegistryDatabase", State: readiness.StateHealthy, Message: "ok", Duration: "1.5ms"},
			{Name: "RedisCache", State: readiness.StateDegraded, Message: "slow", Duration: "500us"},
			{Name: "PaymentsAPI", State: readiness.StateUnhealthy, Message: "timeout", Duration: "bogus"},
		},
	}
}

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

func TestCollectEmitsDependencyMetrics(t *testing.T) {
	h := newHarness(t)
	probe := &fakeProbe{snapshot: sampleSnapshot()}
	mock := clock.NewMockClock(time.Unix(1_700_000_000, 0))
	r := NewReadiness(h.client, probe, time.Hour, WithClock(mock))

	r.collect(context.Background())

	points := h.drain(t)
	require.Len(t, points, 3, "one MetricPoint per dependency child")

	wantMs := time.Unix(1_700_000_000, 0).UnixMilli()
	byName := make(map[string]telemetry_grpcfb.MetricPoint, len(points))
	for _, p := range points {
		assert.Equal(t, kindDependency, p.Kind, "every sample uses the dependency discriminator")
		assert.Equal(t, unitMs, p.Unit)
		assert.Equal(t, wantMs, p.TimestampMs)
		byName[p.Name] = p
	}

	db := byName["RegistryDatabase"]
	assert.InDelta(t, 1.5, db.Value, 1e-9, "1.5ms parses to 1.5")
	assert.Equal(t, statusHealthy, labelValue(db, labelStatus))
	assert.Equal(t, "ok", labelValue(db, labelMessage))
	assert.Equal(t, "database", labelValue(db, labelIcon))

	cache := byName["RedisCache"]
	assert.InDelta(t, 0.5, cache.Value, 1e-9, "500us parses to 0.5ms")
	assert.Equal(t, statusDegraded, labelValue(cache, labelStatus))
	assert.Equal(t, "zap", labelValue(cache, labelIcon))

	api := byName["PaymentsAPI"]
	assert.InDelta(t, 0, api.Value, 1e-9, "unparseable duration yields 0")
	assert.Equal(t, statusDown, labelValue(api, labelStatus), "UNHEALTHY maps to down")
	assert.Equal(t, "globe", labelValue(api, labelIcon))
}

func labelValue(p telemetry_grpcfb.MetricPoint, key string) string {
	for _, kv := range p.Labels {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

func infoLabels(p telemetry_grpcfb.MetricPoint) []telemetry_grpcfb.KV {
	var out []telemetry_grpcfb.KV
	for _, kv := range p.Labels {
		if kv.Key == labelInfoDropped {
			continue
		}
		if strings.HasPrefix(kv.Key, labelInfoPrefix) {
			out = append(out, kv)
		}
	}
	return out
}

func shuffledInfo(n int) []readiness.InfoEntry {
	entries := make([]readiness.InfoEntry, n)
	for i := range entries {

		idx := (i*7 + 3) % n
		entries[i] = readiness.InfoEntry{
			Section: "Overview",
			Key:     fmt.Sprintf("k%03d", idx),
			Value:   fmt.Sprintf("v%03d", idx),
		}
	}
	return entries
}

func TestEmitAppendsBoundedSortedInfoLabels(t *testing.T) {
	h := newHarness(t)
	const total = maxInfoEntriesPerDep + 20
	dep := readiness.Dependency{
		Name:     "DatabaseService",
		State:    readiness.StateHealthy,
		Message:  "ok",
		Duration: "1ms",
		Info:     shuffledInfo(total),
	}
	probe := &fakeProbe{snapshot: readiness.Snapshot{
		Name: "readiness", State: readiness.StateHealthy, Duration: "1ms",
		Dependencies: []readiness.Dependency{dep},
	}}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())

	points := h.drain(t)
	require.Len(t, points, 1)
	p := points[0]

	assert.Equal(t, statusHealthy, labelValue(p, labelStatus))
	assert.Equal(t, "ok", labelValue(p, labelMessage))
	assert.Equal(t, "database", labelValue(p, labelIcon))

	infos := infoLabels(p)
	require.Len(t, infos, maxInfoEntriesPerDep, "info labels are capped at maxInfoEntriesPerDep")

	for i, kv := range infos {
		assert.Equal(t, infoLabelKey("Overview", fmt.Sprintf("k%03d", i)), kv.Key)
		assert.Equal(t, fmt.Sprintf("v%03d", i), kv.Value)
	}
}

func TestEmitSurfacesDroppedInfoCount(t *testing.T) {
	h := newHarness(t)
	const total = maxInfoEntriesPerDep + 20
	dep := readiness.Dependency{
		Name:     "DatabaseService",
		State:    readiness.StateHealthy,
		Message:  "ok",
		Duration: "1ms",
		Info:     shuffledInfo(total),
	}
	probe := &fakeProbe{snapshot: readiness.Snapshot{
		Name: "readiness", State: readiness.StateHealthy, Duration: "1ms",
		Dependencies: []readiness.Dependency{dep},
	}}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())

	points := h.drain(t)
	require.Len(t, points, 1)
	assert.Equal(t, strconv.Itoa(total-maxInfoEntriesPerDep), labelValue(points[0], labelInfoDropped),
		"the count of dropped info entries is surfaced as a label")
}

func TestEmitOmitsDroppedLabelWhenInfoFits(t *testing.T) {
	h := newHarness(t)
	dep := readiness.Dependency{
		Name: "DatabaseService", State: readiness.StateHealthy, Duration: "1ms",
		Info: shuffledInfo(maxInfoEntriesPerDep),
	}
	probe := &fakeProbe{snapshot: readiness.Snapshot{
		Name: "readiness", State: readiness.StateHealthy, Duration: "1ms",
		Dependencies: []readiness.Dependency{dep},
	}}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())

	points := h.drain(t)
	require.Len(t, points, 1)
	assert.Empty(t, labelValue(points[0], labelInfoDropped), "no dropped label when info fits under the cap")
}

func TestEmitInfoSubsetIsStableAcrossCalls(t *testing.T) {
	h := newHarness(t)
	const total = maxInfoEntriesPerDep + 20
	dep := readiness.Dependency{
		Name: "DatabaseService", State: readiness.StateHealthy, Duration: "1ms",
		Info: shuffledInfo(total),
	}
	probe := &fakeProbe{snapshot: readiness.Snapshot{
		Name: "readiness", State: readiness.StateHealthy, Duration: "1ms",
		Dependencies: []readiness.Dependency{dep},
	}}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())
	r.collect(context.Background())

	points := h.drain(t)
	require.Len(t, points, 2, "two samples")

	first := infoLabels(points[0])
	second := infoLabels(points[1])
	require.Len(t, first, maxInfoEntriesPerDep)
	assert.Equal(t, first, second, "the kept info subset is identical tick to tick")
}

func TestEmitTruncatesInfoValues(t *testing.T) {
	h := newHarness(t)
	longValue := strings.Repeat("z", maxInfoValueLen+200)
	dep := readiness.Dependency{
		Name: "DatabaseService", State: readiness.StateHealthy, Duration: "1ms",
		Info: []readiness.InfoEntry{{Section: "Engine Diagnostics", Key: "database_size", Value: longValue}},
	}
	probe := &fakeProbe{snapshot: readiness.Snapshot{
		Name: "readiness", State: readiness.StateHealthy, Duration: "1ms",
		Dependencies: []readiness.Dependency{dep},
	}}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())

	points := h.drain(t)
	require.Len(t, points, 1)
	infos := infoLabels(points[0])
	require.Len(t, infos, 1)
	assert.Equal(t, infoLabelKey("Engine Diagnostics", "database_size"), infos[0].Key)
	assert.LessOrEqual(t, len(infos[0].Value), maxInfoValueLen, "info value is truncated to maxInfoValueLen")
}

func TestEmitWithoutInfoKeepsOnlyReservedLabels(t *testing.T) {
	h := newHarness(t)
	probe := &fakeProbe{snapshot: sampleSnapshot()}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())

	points := h.drain(t)
	require.NotEmpty(t, points)
	for _, p := range points {
		assert.Empty(t, infoLabels(p), "dependencies without Info emit no info labels")
		assert.Len(t, p.Labels, 3, "only the three reserved labels are present")
	}
}

func TestInfoLabelKeyLowercasesSectionAndKey(t *testing.T) {
	assert.Equal(t, "info.engine diagnostics.database_size", infoLabelKey("Engine Diagnostics", "database_size"))
	assert.Equal(t, "info.overview.driver", infoLabelKey("Overview", "Driver"))
}

func TestCollectCapsDependencies(t *testing.T) {
	h := newHarness(t)
	deps := make([]readiness.Dependency, maxDeps+50)
	for i := range deps {
		deps[i] = readiness.Dependency{Name: "dep", State: readiness.StateHealthy, Message: "", Duration: "0s"}
	}
	probe := &fakeProbe{snapshot: readiness.Snapshot{
		Name: "readiness", State: readiness.StateHealthy, Message: "", Duration: "0s", Dependencies: deps,
	}}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())

	points := h.drain(t)
	assert.Len(t, points, maxDeps, "dependency count is capped at maxDeps")
}

func TestCollectBoundsLongStrings(t *testing.T) {
	h := newHarness(t)
	longName := strings.Repeat("a", maxStringLen+500)
	longMsg := strings.Repeat("b", maxStringLen+500)
	probe := &fakeProbe{snapshot: readiness.Snapshot{
		Name: "readiness", State: readiness.StateHealthy, Message: "", Duration: "0s",
		Dependencies: []readiness.Dependency{
			{Name: longName, State: readiness.StateHealthy, Message: longMsg, Duration: "0s"},
		},
	}}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	r.collect(context.Background())

	points := h.drain(t)
	require.Len(t, points, 1)
	assert.LessOrEqual(t, len(points[0].Name), maxStringLen, "name is truncated to the byte budget")
	assert.LessOrEqual(t, len(labelValue(points[0], labelMessage)), maxStringLen, "message is truncated")
}

func TestCollectRecoversFromProbePanic(t *testing.T) {
	h := newHarness(t)
	defer h.drain(t)
	probe := &fakeProbe{snapshot: sampleSnapshot(), panics: true}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	assert.NotPanics(t, func() { r.collect(context.Background()) }, "a panicking probe must not crash the host")
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
	probe := &fakeProbe{snapshot: sampleSnapshot()}
	r := NewReadiness(h.client, probe, interval, WithClock(sc))

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

	assert.GreaterOrEqual(t, probe.callCount(), 2, "expected an immediate sample and a tick sample")

	points := h.drain(t)
	require.NotEmpty(t, points)
	for _, p := range points {
		assert.Equal(t, kindDependency, p.Kind)
	}
}

func TestRunReturnsImmediatelyOnNilClient(t *testing.T) {
	r := NewReadiness(nil, &fakeProbe{snapshot: sampleSnapshot()}, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

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

func TestRunReturnsImmediatelyOnNilProbe(t *testing.T) {
	h := newHarness(t)
	defer h.drain(t)
	r := NewReadiness(h.client, nil, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

	done := make(chan struct{})
	go func() {
		r.Run(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run with nil probe did not return")
	}
}

func TestRunOnNilRuntimeIsSafe(t *testing.T) {
	var r *Runtime
	assert.NotPanics(t, func() { r.Run(context.Background()) })
}

func TestRunStopsWhenContextAlreadyCancelled(t *testing.T) {
	h := newHarness(t)
	defer h.drain(t)
	probe := &fakeProbe{snapshot: sampleSnapshot()}
	r := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))

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

func TestNewReadinessDefaultsNonPositiveInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
	}{
		{"zero", 0},
		{"negative", -time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewReadiness(nil, nil, tc.interval)
			assert.Equal(t, DefaultInterval, r.interval)
		})
	}
}

func TestNewReadinessKeepsPositiveInterval(t *testing.T) {
	r := NewReadiness(nil, nil, 42*time.Second)
	assert.Equal(t, 42*time.Second, r.interval)
}

func TestWithClockIgnoresNil(t *testing.T) {
	r := NewReadiness(nil, nil, time.Hour, WithClock(nil))
	assert.NotNil(t, r.clock, "nil clock option must leave a usable default clock")
}

func TestWithClockOverridesDefault(t *testing.T) {
	mock := clock.NewMockClock(time.Unix(123, 0))
	r := NewReadiness(nil, nil, time.Hour, WithClock(mock))
	assert.Equal(t, mock, r.clock)
}

func TestNewReadinessToleratesNilOption(t *testing.T) {
	assert.NotPanics(t, func() { _ = NewReadiness(nil, nil, time.Hour, nil) })
}

func TestCloseSharedClientIsNoOp(t *testing.T) {
	h := newHarness(t)
	probe := &fakeProbe{snapshot: sampleSnapshot()}
	r := NewReadiness(h.client, probe, time.Hour)

	require.NoError(t, r.Close(context.Background()))

	r2 := NewReadiness(h.client, probe, time.Hour, WithClock(clock.NewMockClock(time.Unix(0, 0))))
	r2.collect(context.Background())

	points := h.drain(t)
	assert.NotEmpty(t, points, "shared client should still accept metrics after Runtime.Close")
}

func TestCloseNilRuntimeIsSafe(t *testing.T) {
	var r *Runtime
	assert.NoError(t, r.Close(context.Background()))
}

func TestMapState(t *testing.T) {
	tests := []struct {
		name  string
		state readiness.State
		want  string
	}{
		{"healthy", readiness.StateHealthy, statusHealthy},
		{"degraded", readiness.StateDegraded, statusDegraded},
		{"unhealthy maps to down", readiness.StateUnhealthy, statusDown},
		{"unknown maps to down", readiness.State("WEIRD"), statusDown},
		{"empty maps to down", readiness.State(""), statusDown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, mapState(tc.state))
		})
	}
}

func TestLatencyMs(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     float64
	}{
		{"milliseconds", "1.5ms", 1.5},
		{"microseconds", "500us", 0.5},
		{"micro symbol", "500µs", 0.5},
		{"seconds", "2s", 2000},
		{"zero", "0s", 0},
		{"empty", "", 0},
		{"unparseable", "bogus", 0},
		{"nanoseconds", "1000000ns", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.want, latencyMs(tc.duration), 1e-9)
		})
	}
}

func TestIconFor(t *testing.T) {
	tests := []struct {
		name    string
		depName string
		want    string
	}{
		{"postgres", "PostgresPrimary", "database"},
		{"sql", "MySQLReplica", "database"},
		{"db", "RegistryDB", "database"},
		{"redis", "RedisCache", "zap"},
		{"cache", "RenderCache", "zap"},
		{"http", "HTTPProbe", "globe"},
		{"api", "PaymentsAPI", "globe"},
		{"default", "Orchestrator", defaultIcon},
		{"empty", "", defaultIcon},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dep := readiness.Dependency{Name: tc.depName, State: readiness.StateHealthy, Message: "", Duration: ""}
			assert.Equal(t, tc.want, iconFor(&dep))
		})
	}
}
