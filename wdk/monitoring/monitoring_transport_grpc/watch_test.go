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

package monitoring_transport_grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/wdk/clock"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

type mockHealthWatchStream struct {
	ctx     context.Context
	sendErr error
	sent    []*pb.HealthUpdate
}

func (m *mockHealthWatchStream) Send(update *pb.HealthUpdate) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, update)
	return nil
}

func (m *mockHealthWatchStream) Context() context.Context     { return m.ctx }
func (m *mockHealthWatchStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockHealthWatchStream) SendHeader(metadata.MD) error { return nil }
func (m *mockHealthWatchStream) SetTrailer(metadata.MD)       {}
func (m *mockHealthWatchStream) SendMsg(any) error            { return nil }
func (m *mockHealthWatchStream) RecvMsg(any) error            { return nil }

type mockOrchestratorWatchStream struct {
	ctx     context.Context
	sendErr error
	sent    []*pb.TasksUpdate
}

func (m *mockOrchestratorWatchStream) Send(update *pb.TasksUpdate) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, update)
	return nil
}

func (m *mockOrchestratorWatchStream) Context() context.Context     { return m.ctx }
func (m *mockOrchestratorWatchStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockOrchestratorWatchStream) SendHeader(metadata.MD) error { return nil }
func (m *mockOrchestratorWatchStream) SetTrailer(metadata.MD)       {}
func (m *mockOrchestratorWatchStream) SendMsg(any) error            { return nil }
func (m *mockOrchestratorWatchStream) RecvMsg(any) error            { return nil }

type mockRegistryWatchStream struct {
	ctx     context.Context
	sendErr error
	sent    []*pb.ArtefactsUpdate
}

func (m *mockRegistryWatchStream) Send(update *pb.ArtefactsUpdate) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, update)
	return nil
}

func (m *mockRegistryWatchStream) Context() context.Context     { return m.ctx }
func (m *mockRegistryWatchStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockRegistryWatchStream) SendHeader(metadata.MD) error { return nil }
func (m *mockRegistryWatchStream) SetTrailer(metadata.MD)       {}
func (m *mockRegistryWatchStream) SendMsg(any) error            { return nil }
func (m *mockRegistryWatchStream) RecvMsg(any) error            { return nil }

type mockMetricsWatchStream struct {
	ctx     context.Context
	sendErr error
	sent    []*pb.MetricsUpdate
}

func (m *mockMetricsWatchStream) Send(update *pb.MetricsUpdate) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, update)
	return nil
}

func (m *mockMetricsWatchStream) Context() context.Context     { return m.ctx }
func (m *mockMetricsWatchStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockMetricsWatchStream) SendHeader(metadata.MD) error { return nil }
func (m *mockMetricsWatchStream) SetTrailer(metadata.MD)       {}
func (m *mockMetricsWatchStream) SendMsg(any) error            { return nil }
func (m *mockMetricsWatchStream) RecvMsg(any) error            { return nil }

func TestHealthService_WatchHealth_ContextCancellation(t *testing.T) {
	t.Parallel()

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	service := NewHealthService(nil, WithHealthServiceClock(mock))

	ctx, cancel := context.WithCancelCause(context.Background())
	baseline := mock.TimerCount()

	errCh := make(chan error, 1)
	stream := &mockHealthWatchStream{ctx: ctx}
	go func() {
		errCh <- service.WatchHealth(&pb.WatchHealthRequest{IntervalMs: 100}, stream)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := <-errCh
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestHealthService_WatchHealth_WithProbe(t *testing.T) {
	t.Parallel()

	probe := &mockHealthProbeService{
		liveness: monitoring_domain.HealthProbeStatus{
			Name:  "liveness",
			State: "HEALTHY",
		},
		readiness: monitoring_domain.HealthProbeStatus{
			Name:  "readiness",
			State: "HEALTHY",
		},
	}

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	service := NewHealthService(probe, WithHealthServiceClock(mock))

	ctx, cancel := context.WithCancelCause(context.Background())
	baseline := mock.TimerCount()
	stream := &mockHealthWatchStream{ctx: ctx}

	done := make(chan struct{})
	go func() {
		_ = service.WatchHealth(&pb.WatchHealthRequest{IntervalMs: 100}, stream)
		close(done)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	mock.Advance(100 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))
	<-done

	assert.NotEmpty(t, stream.sent, "expected at least one health update")
	if len(stream.sent) > 0 {
		assert.Equal(t, "HEALTHY", stream.sent[0].Liveness.State)
		assert.Equal(t, "HEALTHY", stream.sent[0].Readiness.State)
	}
}

func TestHealthService_WatchHealth_NilProbe(t *testing.T) {
	t.Parallel()

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	service := NewHealthService(nil, WithHealthServiceClock(mock))

	ctx, cancel := context.WithCancelCause(context.Background())
	baseline := mock.TimerCount()
	stream := &mockHealthWatchStream{ctx: ctx}

	done := make(chan struct{})
	go func() {
		_ = service.WatchHealth(&pb.WatchHealthRequest{IntervalMs: 100}, stream)
		close(done)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	mock.Advance(100 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))
	<-done

	assert.NotEmpty(t, stream.sent, "expected at least one health update")
	if len(stream.sent) > 0 {
		assert.Equal(t, "piko", stream.sent[0].Liveness.Name)
		assert.Equal(t, "HEALTHY", stream.sent[0].Liveness.State)
		assert.Equal(t, "Health probe not configured", stream.sent[0].Liveness.Message)
	}
}

func TestHealthService_WatchHealth_MinInterval(t *testing.T) {
	t.Parallel()

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	service := NewHealthService(nil, WithHealthServiceClock(mock))

	ctx, cancel := context.WithCancelCause(context.Background())
	baseline := mock.TimerCount()
	stream := &mockHealthWatchStream{ctx: ctx}

	done := make(chan struct{})
	go func() {
		_ = service.WatchHealth(&pb.WatchHealthRequest{IntervalMs: 10}, stream)
		close(done)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	mock.Advance(200 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))
	<-done

	assert.Empty(t, stream.sent)
}

func TestOrchestratorInspectorService_WatchTasks_ContextCancellation(t *testing.T) {
	t.Parallel()

	inspector := &mockOrchestratorInspector{
		taskSummaryReturn: []orchestrator_domain.TaskSummary{
			{Status: "PENDING", Count: 5},
		},
		recentTasksReturn: []orchestrator_domain.TaskListItem{
			{ID: "task-1"},
		},
	}

	service := NewOrchestratorInspectorService(inspector)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 350*time.Millisecond, fmt.Errorf("test: orchestrator watch stream deadline"))
	defer cancel()

	stream := &mockOrchestratorWatchStream{ctx: ctx}
	err := service.WatchTasks(&pb.WatchTasksRequest{IntervalMs: 100}, stream)

	require.Error(t, err)
	assert.NotEmpty(t, stream.sent)
}

func TestRegistryInspectorService_WatchArtefacts_ContextCancellation(t *testing.T) {
	t.Parallel()

	inspector := &mockRegistryInspector{
		artefactSummaryReturn: []registry_domain.ArtefactSummary{
			{Status: "READY", Count: 10},
		},
		recentArtefactsReturn: []registry_domain.ArtefactListItem{
			{ID: "art-1"},
		},
	}

	service := NewRegistryInspectorService(inspector)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 350*time.Millisecond, fmt.Errorf("test: registry watch stream deadline"))
	defer cancel()

	stream := &mockRegistryWatchStream{ctx: ctx}
	err := service.WatchArtefacts(&pb.WatchArtefactsRequest{IntervalMs: 100}, stream)

	require.Error(t, err)
	assert.NotEmpty(t, stream.sent)
}

func TestMetricsService_WatchMetrics_NilTelemetry(t *testing.T) {
	t.Parallel()

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	service := NewMetricsService(nil, nil, nil, nil, WithMetricsServiceClock(mock))

	ctx, cancel := context.WithCancelCause(context.Background())

	errCh := make(chan error, 1)
	stream := &mockMetricsWatchStream{ctx: ctx}
	go func() {
		errCh <- service.WatchMetrics(&pb.WatchMetricsRequest{IntervalMs: 100}, stream)
	}()

	err := <-errCh
	cancel(fmt.Errorf("test: cleanup"))
	assert.NoError(t, err)
}

func TestMetricsService_WatchMetrics_WithTelemetry(t *testing.T) {
	t.Parallel()

	telemetry := &monitoring_domain.MockTelemetryProvider{
		GetMetricsFunc: func() []monitoring_domain.MetricData {
			return []monitoring_domain.MetricData{
				{
					Name: "test.metric",
					Type: "counter",
					DataPoints: []monitoring_domain.MetricDataPoint{
						{TimestampMs: 1000, Value: 42},
					},
				},
			}
		},
	}

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	service := NewMetricsService(telemetry, nil, nil, nil, WithMetricsServiceClock(mock))

	ctx, cancel := context.WithCancelCause(context.Background())
	baseline := mock.TimerCount()
	stream := &mockMetricsWatchStream{ctx: ctx}

	done := make(chan struct{})
	go func() {
		_ = service.WatchMetrics(&pb.WatchMetricsRequest{IntervalMs: 100}, stream)
		close(done)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	mock.Advance(100 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))
	<-done

	assert.NotEmpty(t, stream.sent)
	if len(stream.sent) > 0 {
		assert.Len(t, stream.sent[0].Metrics, 1)
		assert.Equal(t, "test.metric", stream.sent[0].Metrics[0].Name)
	}
}

func TestMetricsService_WatchMetrics_MinInterval(t *testing.T) {
	t.Parallel()

	telemetry := &monitoring_domain.MockTelemetryProvider{}
	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	service := NewMetricsService(telemetry, nil, nil, nil, WithMetricsServiceClock(mock))

	ctx, cancel := context.WithCancelCause(context.Background())
	baseline := mock.TimerCount()
	stream := &mockMetricsWatchStream{ctx: ctx}

	done := make(chan struct{})
	go func() {
		_ = service.WatchMetrics(&pb.WatchMetricsRequest{IntervalMs: 10}, stream)
		close(done)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	mock.Advance(200 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))
	<-done

	assert.Empty(t, stream.sent)
}
