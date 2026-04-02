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
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSpanProcessor struct{}

func (m *mockSpanProcessor) Shutdown(_ context.Context) error   { return nil }
func (m *mockSpanProcessor) ForceFlush(_ context.Context) error { return nil }

var _ SpanProcessor = (*mockSpanProcessor)(nil)

type mockMetricReader struct{}

func (m *mockMetricReader) Shutdown(_ context.Context) error { return nil }

type mockMetricsCollectorAdapter struct {
	startCalled  atomic.Bool
	stopCalled   atomic.Bool
	readerCalled atomic.Bool
}

func (m *mockMetricsCollectorAdapter) Start(_ context.Context) {
	m.startCalled.Store(true)
}

func (m *mockMetricsCollectorAdapter) Stop() {
	m.stopCalled.Store(true)
}

func (m *mockMetricsCollectorAdapter) Reader() MetricReader {
	m.readerCalled.Store(true)
	return &mockMetricReader{}
}

var _ MetricsCollectorAdapter = (*mockMetricsCollectorAdapter)(nil)

type mockTransportServer struct {
	address     string
	startCalled atomic.Bool
	stopCalled  atomic.Bool
}

func (m *mockTransportServer) Start(ctx context.Context) error {
	m.startCalled.Store(true)
	<-ctx.Done()
	return ctx.Err()
}

func (m *mockTransportServer) Stop(_ context.Context) {
	m.stopCalled.Store(true)
}

func (m *mockTransportServer) Address() string {
	return m.address
}

var _ TransportServer = (*mockTransportServer)(nil)

func newTestFactories(spanProcessor SpanProcessor, metricsAdapter *mockMetricsCollectorAdapter) ServiceFactories {
	return ServiceFactories{
		SpanProcessorFactory: func(_ *TelemetryStore) SpanProcessor {
			return spanProcessor
		},
		MetricsCollectorFactory: func(_ *TelemetryStore, _ time.Duration) MetricsCollectorAdapter {
			return metricsAdapter
		},
	}
}

func newMockTransportFactory(transport *mockTransportServer) TransportFactory {
	return func(_ MonitoringDeps, _ TransportConfig) (TransportServer, error) {
		return transport, nil
	}
}

func TestWithServiceAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "sets port only",
			address:  ":8080",
			expected: ":8080",
		},
		{
			name:     "sets full address",
			address:  "localhost:9091",
			expected: "localhost:9091",
		},
		{
			name:     "sets empty string",
			address:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := ServiceConfig{
				Address:                   "",
				BindAddress:               "",
				MaxSpans:                  0,
				MaxMetrics:                0,
				MaxMetricAge:              0,
				MetricsCollectionInterval: 0,
			}
			opt := WithServiceAddress(tt.address)
			opt(&config)

			assert.Equal(t, tt.expected, config.Address)
		})
	}
}

func TestWithServiceBindAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "sets localhost",
			address:  "127.0.0.1",
			expected: "127.0.0.1",
		},
		{
			name:     "sets all interfaces",
			address:  "0.0.0.0",
			expected: "0.0.0.0",
		},
		{
			name:     "sets empty string",
			address:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := ServiceConfig{
				Address:                   "",
				BindAddress:               "",
				MaxSpans:                  0,
				MaxMetrics:                0,
				MaxMetricAge:              0,
				MetricsCollectionInterval: 0,
			}
			opt := WithServiceBindAddress(tt.address)
			opt(&config)

			assert.Equal(t, tt.expected, config.BindAddress)
		})
	}
}

func TestWithServiceMaxSpans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{
			name:     "sets positive value",
			n:        500,
			expected: 500,
		},
		{
			name:     "sets zero",
			n:        0,
			expected: 0,
		},
		{
			name:     "sets large value",
			n:        100000,
			expected: 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := ServiceConfig{
				Address:                   "",
				BindAddress:               "",
				MaxSpans:                  0,
				MaxMetrics:                0,
				MaxMetricAge:              0,
				MetricsCollectionInterval: 0,
			}
			opt := WithServiceMaxSpans(tt.n)
			opt(&config)

			assert.Equal(t, tt.expected, config.MaxSpans)
		})
	}
}

func TestWithServiceMaxMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{
			name:     "sets positive value",
			n:        1000,
			expected: 1000,
		},
		{
			name:     "sets zero",
			n:        0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := ServiceConfig{
				Address:                   "",
				BindAddress:               "",
				MaxSpans:                  0,
				MaxMetrics:                0,
				MaxMetricAge:              0,
				MetricsCollectionInterval: 0,
			}
			opt := WithServiceMaxMetrics(tt.n)
			opt(&config)

			assert.Equal(t, tt.expected, config.MaxMetrics)
		})
	}
}

func TestWithServiceMaxMetricAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		d        time.Duration
		expected time.Duration
	}{
		{
			name:     "sets five minutes",
			d:        5 * time.Minute,
			expected: 5 * time.Minute,
		},
		{
			name:     "sets one hour",
			d:        time.Hour,
			expected: time.Hour,
		},
		{
			name:     "sets zero",
			d:        0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := ServiceConfig{
				Address:                   "",
				BindAddress:               "",
				MaxSpans:                  0,
				MaxMetrics:                0,
				MaxMetricAge:              0,
				MetricsCollectionInterval: 0,
			}
			opt := WithServiceMaxMetricAge(tt.d)
			opt(&config)

			assert.Equal(t, tt.expected, config.MaxMetricAge)
		})
	}
}

func TestWithServiceMetricsInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		d        time.Duration
		expected time.Duration
	}{
		{
			name:     "sets ten seconds",
			d:        10 * time.Second,
			expected: 10 * time.Second,
		},
		{
			name:     "sets one second",
			d:        time.Second,
			expected: time.Second,
		},
		{
			name:     "sets zero",
			d:        0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := ServiceConfig{
				Address:                   "",
				BindAddress:               "",
				MaxSpans:                  0,
				MaxMetrics:                0,
				MaxMetricAge:              0,
				MetricsCollectionInterval: 0,
			}
			opt := WithServiceMetricsInterval(tt.d)
			opt(&config)

			assert.Equal(t, tt.expected, config.MetricsCollectionInterval)
		})
	}
}

func TestNewService_DefaultConfig(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories)

	require.NotNil(t, service)
	assert.Equal(t, ":9091", service.config.Address)
	assert.Equal(t, "127.0.0.1", service.config.BindAddress)
	assert.Equal(t, DefaultMaxSpans, service.config.MaxSpans)
	assert.Equal(t, DefaultMaxMetrics, service.config.MaxMetrics)
	assert.Equal(t, DefaultMaxMetricAge, service.config.MaxMetricAge)
	assert.Equal(t, DefaultMetricsCollectionInterval, service.config.MetricsCollectionInterval)
	assert.NotNil(t, service.store)
	assert.NotNil(t, service.spanProcessor)
	assert.NotNil(t, service.metricsCollector)
	assert.NotNil(t, service.systemCollector)
	assert.NotNil(t, service.resourceCollector)
}

func TestNewService_WithOptions(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories,
		WithServiceAddress(":7777"),
		WithServiceBindAddress("0.0.0.0"),
		WithServiceMaxSpans(500),
		WithServiceMaxMetrics(100),
		WithServiceMaxMetricAge(3*time.Minute),
		WithServiceMetricsInterval(10*time.Second),
	)

	require.NotNil(t, service)
	assert.Equal(t, ":7777", service.config.Address)
	assert.Equal(t, "0.0.0.0", service.config.BindAddress)
	assert.Equal(t, 500, service.config.MaxSpans)
	assert.Equal(t, 100, service.config.MaxMetrics)
	assert.Equal(t, 3*time.Minute, service.config.MaxMetricAge)
	assert.Equal(t, 10*time.Second, service.config.MetricsCollectionInterval)
}

func TestNewService_FullAddressWithoutColon(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories,
		WithServiceAddress("10.0.0.1:9091"),
	)

	require.NotNil(t, service)
	assert.Equal(t, "10.0.0.1:9091", service.config.Address)
}

func TestService_SpanProcessor(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories)

	result := service.SpanProcessor()
	assert.Equal(t, spanProc, result)
}

func TestService_MetricsReader(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories)

	reader := service.MetricsReader()
	assert.NotNil(t, reader)
	assert.True(t, metricsAdapter.readerCalled.Load())
}

func TestService_Address_BeforeStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  string
		bind     string
		expected string
	}{
		{
			name:     "port only prepends bind address",
			address:  ":9091",
			bind:     "127.0.0.1",
			expected: "127.0.0.1:9091",
		},
		{
			name:     "full address returns as-is",
			address:  "10.0.0.1:9091",
			bind:     "127.0.0.1",
			expected: "10.0.0.1:9091",
		},
		{
			name:     "port only with custom bind",
			address:  ":8080",
			bind:     "0.0.0.0",
			expected: "0.0.0.0:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spanProc := &mockSpanProcessor{}
			metricsAdapter := &mockMetricsCollectorAdapter{}
			factories := newTestFactories(spanProc, metricsAdapter)

			deps := MonitoringDeps{
				OrchestratorInspector: nil,
				RegistryInspector:     nil,
				DispatcherInspector:   nil,
				RateLimiterInspector:  nil,
				TelemetryProvider:     nil,
				SystemStatsProvider:   nil,
				ResourceProvider:      nil,
				HealthProbeService:    nil,
				ProviderInfoInspector: nil,
			}

			service := NewService(deps, factories,
				WithServiceAddress(tt.address),
				WithServiceBindAddress(tt.bind),
			)

			assert.Equal(t, tt.expected, service.Address())
		})
	}
}

func TestService_SetInspectors(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories)

	assert.Nil(t, service.orchestratorInspector)
	assert.Nil(t, service.registryInspector)
	assert.Nil(t, service.healthProbeService)
	assert.Nil(t, service.dispatcherInspector)
	assert.Nil(t, service.rateLimiterInspector)

	service.SetInspectors(nil, nil, nil, nil, nil)

	assert.Nil(t, service.orchestratorInspector)
	assert.Nil(t, service.registryInspector)
	assert.Nil(t, service.healthProbeService)
	assert.Nil(t, service.dispatcherInspector)
	assert.Nil(t, service.rateLimiterInspector)
}

func TestService_SetProviderInfoInspector(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories)

	assert.Nil(t, service.providerInfoInspector)

	agg := NewProviderInfoAggregator()
	service.SetProviderInfoInspector(agg)

	assert.Equal(t, agg, service.providerInfoInspector)

	service.SetProviderInfoInspector(nil)
	assert.Nil(t, service.providerInfoInspector)
}

func TestWithServiceAutoNextPort(t *testing.T) {
	t.Parallel()

	config := ServiceConfig{}
	WithServiceAutoNextPort(true)(&config)
	assert.True(t, config.AutoNextPort)

	WithServiceAutoNextPort(false)(&config)
	assert.False(t, config.AutoNextPort)
}

func TestService_StartAndStop_WithTransport(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	mockTransport := &mockTransportServer{address: "127.0.0.1:9999"}

	deps := MonitoringDeps{}

	service := NewService(deps, factories,
		WithServiceAddress(":9091"),
		WithServiceBindAddress("127.0.0.1"),
		WithServiceTransportFactory(newMockTransportFactory(mockTransport)),
	)

	ctx, cancel := context.WithCancelCause(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- service.Start(ctx)
	}()

	require.Eventually(t, func() bool {
		return mockTransport.startCalled.Load()
	}, 5*time.Second, 5*time.Millisecond, "transport.Start was never called")

	assert.Equal(t, "127.0.0.1:9999", service.Address())

	cancel(fmt.Errorf("test: cleanup"))
	<-errCh
	service.Stop(context.Background())
	assert.True(t, mockTransport.stopCalled.Load())
}

func TestService_Stop_BeforeStart(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories)

	service.Stop(context.Background())

	assert.True(t, metricsAdapter.stopCalled.Load())
}

func TestService_StartAndStop_LocalOnly(t *testing.T) {
	t.Parallel()

	spanProc := &mockSpanProcessor{}
	metricsAdapter := &mockMetricsCollectorAdapter{}
	factories := newTestFactories(spanProc, metricsAdapter)

	deps := MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	service := NewService(deps, factories,
		WithServiceAddress(":0"),
		WithServiceBindAddress("127.0.0.1"),
	)

	ctx, cancel := context.WithCancelCause(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- service.Start(ctx)
	}()

	require.Eventually(t, func() bool {
		return metricsAdapter.startCalled.Load()
	}, 5*time.Second, 5*time.Millisecond, "metricsAdapter.Start was never called")

	cancel(fmt.Errorf("test: cleanup"))

	err := <-errCh
	assert.ErrorIs(t, err, context.Canceled)

	service.Stop(context.Background())
	assert.True(t, metricsAdapter.stopCalled.Load())
}
