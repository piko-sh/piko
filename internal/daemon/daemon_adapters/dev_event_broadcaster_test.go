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

package daemon_adapters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

func TestDevEventBroadcaster_ClientLifecycle(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer func() { _ = b.Close() }()

	assert.Equal(t, 0, b.ClientCount())

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)
	assert.Equal(t, 1, b.ClientCount())

	ch2 := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch2)
	assert.Equal(t, 2, b.ClientCount())

	b.removeClient(ch)
	assert.Equal(t, 1, b.ClientCount())

	b.removeClient(ch2)
	assert.Equal(t, 0, b.ClientCount())
}

func TestDevEventBroadcaster_BroadcastFanout(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer func() { _ = b.Close() }()

	ch1 := make(chan []byte, devSSEClientBuffer)
	ch2 := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch1)
	b.addClient(ch2)
	defer b.removeClient(ch1)
	defer b.removeClient(ch2)

	assert.Equal(t, 2, b.ClientCount())

	b.Broadcast(DevBuildEvent{
		Type:           "rebuild-complete",
		AffectedRoutes: []string{"/login"},
		TimestampMs:    1234567890,
	})

	msg1 := <-ch1
	msg2 := <-ch2

	assert.Contains(t, string(msg1), "event: rebuild-complete")
	assert.Contains(t, string(msg1), `"/login"`)
	assert.Equal(t, msg1, msg2, "both clients should receive identical messages")
}

func TestDevEventBroadcaster_SlowClientDrop(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer func() { _ = b.Close() }()

	ch := make(chan []byte, 1)
	b.addClient(ch)
	defer b.removeClient(ch)

	b.Broadcast(DevBuildEvent{Type: "rebuild-complete", AffectedRoutes: []string{"/a"}})

	done := make(chan struct{})
	go func() {
		b.Broadcast(DevBuildEvent{Type: "rebuild-complete", AffectedRoutes: []string{"/b"}})
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(time.Second):
		require.FailNow(t, "Broadcast blocked on a slow client")
	}
}

func TestDevEventBroadcaster_Close(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)

	assert.Equal(t, 1, b.ClientCount())

	require.NoError(t, b.Close())
	assert.Equal(t, 0, b.ClientCount())

	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after Close()")

	require.NoError(t, b.Close())
}

func TestDevEventBroadcaster_NotifyRebuildComplete(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer func() { _ = b.Close() }()

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)
	defer b.removeClient(ch)

	b.NotifyRebuildComplete(context.Background(), []string{
		"pages/login.pk",
		"pages/dashboard.pk",
		"partials/header.pk",
	})

	msg := <-ch
	s := string(msg)
	assert.Contains(t, s, `"/login"`)
	assert.Contains(t, s, `"/dashboard"`)
	assert.NotContains(t, s, "header")
}

func TestDevEventBroadcaster_NotifyRebuildComplete_OnlyPartials(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer func() { _ = b.Close() }()

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)
	defer b.removeClient(ch)

	b.NotifyRebuildComplete(context.Background(), []string{
		"partials/sidebar.pk",
	})

	msg := <-ch
	assert.Contains(t, string(msg), `"*"`)
}

func TestDevEventBroadcaster_ServeHTTP_InitialHeartbeat(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer func() { _ = b.Close() }()

	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second, errors.New("test timed out waiting for initial heartbeat"))
	defer cancel()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/_piko/dev/events", nil)
	w := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	go b.ServeHTTP(w, req)

	require.Eventually(t, func() bool {
		return strings.Contains(w.Body(), "event: connected")
	}, time.Second, 10*time.Millisecond)
}

func TestPathToRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"pages/login.pk", "/login", true},
		{"pages/dashboard.pk", "/dashboard", true},
		{"pages/environments/environment_id/blueprints.pk", "/environments/environment_id/blueprints", true},
		{"pages/index.pk", "/", true},
		{"pages/blog/index.pk", "/blog", true},
		{"partials/header.pk", "", false},
		{"emails/welcome.pk", "", false},
		{"random/file.go", "", false},
	}

	for _, tt := range tests {
		route, ok := pathToRoute(tt.input)
		assert.Equal(t, tt.ok, ok, "pathToRoute(%q) ok", tt.input)
		if ok {
			assert.Equal(t, tt.expected, route, "pathToRoute(%q) route", tt.input)
		}
	}
}

type flushRecorder struct {
	*httptest.ResponseRecorder
	mu sync.RWMutex
}

func (f *flushRecorder) Write(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ResponseRecorder.Write(b)
}

func (f *flushRecorder) WriteHeader(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ResponseRecorder.WriteHeader(code)
}

func (f *flushRecorder) Flush() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ResponseRecorder.Flush()
}

func (f *flushRecorder) Body() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.ResponseRecorder.Body.String()
}

type stubSystemStatsProvider struct{}

func (stubSystemStatsProvider) GetStats() monitoring_domain.SystemStats {
	return monitoring_domain.SystemStats{}
}

func TestDevEventBroadcaster_Close_DrainsStatsGoroutine(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	b.SetSystemStatsProvider(stubSystemStatsProvider{})

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)

	require.Eventually(t, func() bool {
		b.mu.RLock()
		defer b.mu.RUnlock()
		return b.statsRunning
	}, time.Second, 5*time.Millisecond, "stats goroutine should be running while a client is connected")

	require.NoError(t, b.Close(), "Close should drain the stats goroutine cleanly")

	b.mu.RLock()
	stillRunning := b.statsRunning
	b.mu.RUnlock()
	assert.False(t, stillRunning, "statsRunning should be cleared via cancel after Close")
}

func TestDevEventBroadcaster_Close_NoStatsGoroutine(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)

	require.NoError(t, b.Close(), "Close should succeed even when no stats provider is configured")
}

type broadcasterHealthProbe struct{}

func (broadcasterHealthProbe) CheckLiveness(context.Context) monitoring_domain.HealthProbeStatus {
	return monitoring_domain.HealthProbeStatus{Name: "liveness", State: "HEALTHY"}
}

func (broadcasterHealthProbe) CheckReadiness(context.Context) monitoring_domain.HealthProbeStatus {
	return monitoring_domain.HealthProbeStatus{Name: "readiness", State: "HEALTHY"}
}

type broadcasterResourceProvider struct{}

func (broadcasterResourceProvider) GetResources() monitoring_domain.ResourceData {
	return monitoring_domain.ResourceData{Total: 7}
}

type broadcasterProviderInspector struct {
	monitoring_domain.ProviderInfoInspector
}

func (broadcasterProviderInspector) ListResourceTypes(context.Context) []string {
	return []string{"cache", "broken"}
}

func (broadcasterProviderInspector) ListProviders(_ context.Context, resourceType string) (*monitoring_domain.ProviderListResult, error) {
	if resourceType == "broken" {
		return nil, errors.New("listing failed")
	}
	return &monitoring_domain.ProviderListResult{
		Rows: []provider_domain.ProviderListEntry{{Name: "otter"}, {Name: "redis"}},
	}, nil
}

func (broadcasterProviderInspector) DescribeProvider(_ context.Context, _, name string) (*provider_domain.ProviderDetail, error) {
	if name == "redis" {
		return nil, errors.New("describe failed")
	}
	return &provider_domain.ProviderDetail{Name: name}, nil
}

func (broadcasterProviderInspector) ListSubResources(_ context.Context, _, name string) (*monitoring_domain.ProviderListResult, error) {
	if name == "redis" {
		return nil, errors.New("sub-resources failed")
	}
	return &monitoring_domain.ProviderListResult{
		Rows: []provider_domain.ProviderListEntry{{Name: "namespace-a"}},
	}, nil
}

type broadcasterOrchestrator struct {
	err error
}

func (o broadcasterOrchestrator) ListTaskSummary(context.Context) ([]orchestrator_domain.TaskSummary, error) {
	return []orchestrator_domain.TaskSummary{}, o.err
}

func (o broadcasterOrchestrator) ListRecentTasks(context.Context, int) ([]orchestrator_domain.TaskListItem, error) {
	return []orchestrator_domain.TaskListItem{}, o.err
}

func (o broadcasterOrchestrator) ListWorkflowSummary(context.Context, int) ([]orchestrator_domain.WorkflowSummary, error) {
	return []orchestrator_domain.WorkflowSummary{}, o.err
}

type nonFlushingResponseWriter struct {
	http.ResponseWriter
}

func newFullyWiredBroadcaster() *DevEventBroadcaster {
	broadcaster := NewDevEventBroadcaster()
	broadcaster.SetHealthProbeService(broadcasterHealthProbe{})
	broadcaster.SetResourceProvider(broadcasterResourceProvider{})
	broadcaster.SetProviderInfoInspector(broadcasterProviderInspector{})
	broadcaster.SetOrchestratorInspector(broadcasterOrchestrator{})
	return broadcaster
}

func drainEvents(t *testing.T, channel chan []byte) string {
	t.Helper()

	var collected strings.Builder
	for {
		select {
		case message := <-channel:
			collected.Write(message)
		default:
			return collected.String()
		}
	}
}

func TestDevEventBroadcaster_ServeHTTP_RejectsWritersThatCannotStream(t *testing.T) {
	t.Parallel()

	broadcaster := NewDevEventBroadcaster()
	defer func() { _ = broadcaster.Close() }()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/_piko/dev/events", nil)

	broadcaster.ServeHTTP(nonFlushingResponseWriter{ResponseWriter: recorder}, request)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "streaming unsupported")
}

func TestDevEventBroadcaster_ServeHTTP_SendsInitialStateAndBroadcasts(t *testing.T) {
	t.Parallel()

	broadcaster := newFullyWiredBroadcaster()
	broadcaster.SetSystemStatsProvider(stubSystemStatsProvider{})
	defer func() { _ = broadcaster.Close() }()

	ctx, cancel := context.WithCancelCause(t.Context())
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/_piko/dev/events", nil)
	writer := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	done := make(chan struct{})
	go func() {
		defer close(done)
		broadcaster.ServeHTTP(writer, request)
	}()

	require.Eventually(t, func() bool { return broadcaster.ClientCount() == 1 }, time.Second, time.Millisecond)

	body := writer.Body()
	for _, event := range []string{"system-stats", "memory-detail", "build-update", "health-update", "resources-update", "providers-update"} {
		assert.Contains(t, body, "event: "+event, "a new client must receive %s straight away", event)
	}

	broadcaster.BroadcastBuildSummary(DevBuildSummary{ComponentCount: 3, DurationMs: 40, ErrorCount: 0, TimestampMs: 1})
	require.Eventually(t, func() bool {
		return strings.Contains(writer.Body(), "event: build-summary")
	}, time.Second, time.Millisecond)

	cancel(errors.New("client went away"))
	<-done
	assert.Equal(t, 0, broadcaster.ClientCount())
}

func TestDevEventBroadcaster_ServeHTTP_ReturnsWhenTheBroadcasterCloses(t *testing.T) {
	t.Parallel()

	broadcaster := NewDevEventBroadcaster()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/_piko/dev/events", nil)
	writer := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	done := make(chan struct{})
	go func() {
		defer close(done)
		broadcaster.ServeHTTP(writer, request)
	}()

	require.Eventually(t, func() bool { return broadcaster.ClientCount() == 1 }, time.Second, time.Millisecond)
	require.NoError(t, broadcaster.Close())
	<-done
}

func TestDevEventBroadcaster_BroadcastFullState(t *testing.T) {
	t.Parallel()

	broadcaster := newFullyWiredBroadcaster()
	defer func() { _ = broadcaster.Close() }()

	channel := make(chan []byte, devSSEClientBuffer)
	broadcaster.addClient(channel)
	defer broadcaster.removeClient(channel)

	broadcaster.broadcastFullState(t.Context(), stubSystemStatsProvider{})

	events := drainEvents(t, channel)
	for _, event := range []string{"build-update", "health-update", "resources-update", "providers-update", "memory-detail"} {
		assert.Contains(t, events, "event: "+event)
	}
	assert.Contains(t, events, `"otter"`)
	assert.Contains(t, events, "namespace-a")
	assert.NotContains(t, events, `"broken":`, "a resource type that cannot be listed is left out")
}

func TestDevEventBroadcaster_BroadcastFullState_SkipsMissingProviders(t *testing.T) {
	t.Parallel()

	broadcaster := NewDevEventBroadcaster()
	defer func() { _ = broadcaster.Close() }()

	channel := make(chan []byte, devSSEClientBuffer)
	broadcaster.addClient(channel)
	defer broadcaster.removeClient(channel)

	broadcaster.broadcastFullState(t.Context(), nil)

	assert.Empty(t, drainEvents(t, channel))
}

func TestDevEventBroadcaster_BroadcastSystemStats(t *testing.T) {
	t.Parallel()

	broadcaster := NewDevEventBroadcaster()
	defer func() { _ = broadcaster.Close() }()

	channel := make(chan []byte, devSSEClientBuffer)
	broadcaster.addClient(channel)
	defer broadcaster.removeClient(channel)

	broadcaster.broadcastSystemStats(stubSystemStatsProvider{})

	events := drainEvents(t, channel)
	assert.Contains(t, events, "event: system-stats")
	assert.Contains(t, events, `"goroutines"`)
}

func TestDevEventBroadcaster_BroadcastBuildUpdate_SkipsWhenTheOrchestratorFails(t *testing.T) {
	t.Parallel()

	broadcaster := NewDevEventBroadcaster()
	defer func() { _ = broadcaster.Close() }()

	channel := make(chan []byte, devSSEClientBuffer)
	broadcaster.addClient(channel)
	defer broadcaster.removeClient(channel)

	broadcaster.broadcastBuildUpdate(t.Context(), broadcasterOrchestrator{err: errors.New("database closed")})

	assert.Empty(t, drainEvents(t, channel))
}

func TestWriteInitialBuildUpdate_WritesNothingWhenEveryQueryFails(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	writeInitialBuildUpdate(t.Context(), recorder, broadcasterOrchestrator{err: errors.New("database closed")})

	assert.Empty(t, recorder.Body.String())
}

func TestDevEventBroadcaster_BroadcastBuildSummary_DropsForAFullClient(t *testing.T) {
	t.Parallel()

	broadcaster := NewDevEventBroadcaster()
	defer func() { _ = broadcaster.Close() }()

	channel := make(chan []byte, 1)
	broadcaster.addClient(channel)
	defer broadcaster.removeClient(channel)

	broadcaster.BroadcastBuildSummary(DevBuildSummary{ComponentCount: 1})
	broadcaster.BroadcastBuildSummary(DevBuildSummary{ComponentCount: 2})

	assert.Contains(t, string(<-channel), `"componentCount":1`)
	assert.Empty(t, drainEvents(t, channel), "the second summary is dropped for a full client")
}

func TestDevEventBroadcaster_NilReceiverIgnoresBroadcasts(t *testing.T) {
	t.Parallel()

	var broadcaster *DevEventBroadcaster

	assert.NotPanics(t, func() {
		broadcaster.Broadcast(DevBuildEvent{Type: "rebuild-complete"})
		broadcaster.BroadcastBuildSummary(DevBuildSummary{})
	})
}

func TestFormatSSEMessage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "event: ping\ndata: {\"ok\":true}\n\n", string(formatSSEMessage("ping", map[string]bool{"ok": true})))
	assert.Nil(t, formatSSEMessage("broken", make(chan int)), "a payload that cannot be encoded produces no message")
}

func TestDevEventBroadcaster_BroadcastEvent_SkipsAPayloadThatCannotBeEncoded(t *testing.T) {
	t.Parallel()

	broadcaster := NewDevEventBroadcaster()
	defer func() { _ = broadcaster.Close() }()

	channel := make(chan []byte, devSSEClientBuffer)
	broadcaster.addClient(channel)
	defer broadcaster.removeClient(channel)

	broadcaster.broadcastEvent("broken", make(chan int))

	assert.Empty(t, drainEvents(t, channel))
}

func TestResolveSSEFlusher_FallsBackWhenTheWriterCannotFlush(t *testing.T) {
	t.Parallel()

	fallback := httptest.NewRecorder()

	assert.Same(t, fallback, resolveSSEFlusher(&strings.Builder{}, fallback))
}
