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

package logger_otel_sdk

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/clock"
)

func TestHistogramMeanInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dp       metricdata.HistogramDataPoint[int64]
		expected float64
	}{
		{
			name:     "zero count returns zero",
			dp:       metricdata.HistogramDataPoint[int64]{Count: 0, Sum: 100},
			expected: 0,
		},
		{
			name:     "single value",
			dp:       metricdata.HistogramDataPoint[int64]{Count: 1, Sum: 42},
			expected: 42.0,
		},
		{
			name:     "multiple values",
			dp:       metricdata.HistogramDataPoint[int64]{Count: 4, Sum: 100},
			expected: 25.0,
		},
		{
			name:     "non-even division",
			dp:       metricdata.HistogramDataPoint[int64]{Count: 3, Sum: 10},
			expected: 10.0 / 3.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := histogramMeanInt64(tc.dp)
			assert.InDelta(t, tc.expected, result, 1e-9)
		})
	}
}

func TestHistogramMeanFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dp       metricdata.HistogramDataPoint[float64]
		expected float64
	}{
		{
			name:     "zero count returns zero",
			dp:       metricdata.HistogramDataPoint[float64]{Count: 0, Sum: 99.5},
			expected: 0,
		},
		{
			name:     "single value",
			dp:       metricdata.HistogramDataPoint[float64]{Count: 1, Sum: 3.14},
			expected: 3.14,
		},
		{
			name:     "multiple values",
			dp:       metricdata.HistogramDataPoint[float64]{Count: 2, Sum: 10.0},
			expected: 5.0,
		},
		{
			name:     "fractional mean",
			dp:       metricdata.HistogramDataPoint[float64]{Count: 3, Sum: 7.5},
			expected: 2.5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := histogramMeanFloat64(tc.dp)
			assert.InDelta(t, tc.expected, result, 1e-9)
		})
	}
}

func TestAttributeSetToMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		attrs    attribute.Set
		expected map[string]string
		name     string
	}{
		{
			name:     "empty attribute set",
			attrs:    *attribute.EmptySet(),
			expected: map[string]string{},
		},
		{
			name: "single string attribute",
			attrs: attribute.NewSet(
				attribute.String("key1", "value1"),
			),
			expected: map[string]string{"key1": "value1"},
		},
		{
			name: "multiple attributes of different types",
			attrs: attribute.NewSet(
				attribute.String("env", "production"),
				attribute.Int("port", 8080),
				attribute.Bool("debug", true),
			),
			expected: map[string]string{
				"env":   "production",
				"port":  "8080",
				"debug": "true",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := attributeSetToMap(tc.attrs)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewMetricsCollector(t *testing.T) {
	t.Parallel()

	t.Run("positive interval", func(t *testing.T) {
		t.Parallel()
		store := monitoring_domain.NewTelemetryStore()
		mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)
		require.NotNil(t, mc)
		assert.NotNil(t, mc.Reader())
		assert.Equal(t, DefaultMetricsCollectionInterval, mc.interval)
	})

	t.Run("zero interval uses default", func(t *testing.T) {
		t.Parallel()
		store := monitoring_domain.NewTelemetryStore()
		mc := NewMetricsCollector(store, 0)
		require.NotNil(t, mc)
		assert.Equal(t, DefaultMetricsCollectionInterval, mc.interval)
	})

	t.Run("negative interval uses default", func(t *testing.T) {
		t.Parallel()
		store := monitoring_domain.NewTelemetryStore()
		mc := NewMetricsCollector(store, -1)
		require.NotNil(t, mc)
		assert.Equal(t, DefaultMetricsCollectionInterval, mc.interval)
	})
}

func TestMetricsCollector_Reader(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)
	reader := mc.Reader()

	require.NotNil(t, reader, "Reader should return a non-nil metric.Reader")
}

func TestMetricsCollector_CollectBeforeRegistered(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	err := mc.collect(t.Context())
	assert.NoError(t, err)
}

func TestMetricsCollector_RecordMetric_SumInt64(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	m := metricdata.Metrics{
		Name:        "test.counter.int64",
		Description: "A test counter",
		Unit:        "1",
		Data: metricdata.Sum[int64]{
			DataPoints: []metricdata.DataPoint[int64]{
				{
					Value:      42,
					Attributes: attribute.NewSet(attribute.String("env", "test")),
				},
			},
		},
	}

	mc.recordMetric(m)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "test.counter.int64", metrics[0].Name)
	assert.Equal(t, "A test counter", metrics[0].Description)
	assert.Equal(t, "1", metrics[0].Unit)
	assert.Equal(t, "counter", metrics[0].Type)
	require.Len(t, metrics[0].DataPoints, 1)
	assert.Equal(t, float64(42), metrics[0].DataPoints[0].Value)
}

func TestMetricsCollector_RecordMetric_SumFloat64(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	m := metricdata.Metrics{
		Name:        "test.counter.float64",
		Description: "A float64 counter",
		Unit:        "ms",
		Data: metricdata.Sum[float64]{
			DataPoints: []metricdata.DataPoint[float64]{
				{Value: 3.14},
			},
		},
	}

	mc.recordMetric(m)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "test.counter.float64", metrics[0].Name)
	assert.Equal(t, "counter", metrics[0].Type)
	assert.InDelta(t, 3.14, metrics[0].DataPoints[0].Value, 1e-9)
}

func TestMetricsCollector_RecordMetric_GaugeInt64(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	m := metricdata.Metrics{
		Name: "test.gauge.int64",
		Data: metricdata.Gauge[int64]{
			DataPoints: []metricdata.DataPoint[int64]{
				{Value: 100},
			},
		},
	}

	mc.recordMetric(m)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "gauge", metrics[0].Type)
	assert.Equal(t, float64(100), metrics[0].DataPoints[0].Value)
}

func TestMetricsCollector_RecordMetric_GaugeFloat64(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	m := metricdata.Metrics{
		Name: "test.gauge.float64",
		Data: metricdata.Gauge[float64]{
			DataPoints: []metricdata.DataPoint[float64]{
				{Value: 99.9},
			},
		},
	}

	mc.recordMetric(m)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "gauge", metrics[0].Type)
	assert.InDelta(t, 99.9, metrics[0].DataPoints[0].Value, 1e-9)
}

func TestMetricsCollector_RecordMetric_HistogramInt64(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	m := metricdata.Metrics{
		Name: "test.histogram.int64",
		Data: metricdata.Histogram[int64]{
			DataPoints: []metricdata.HistogramDataPoint[int64]{
				{Sum: 100, Count: 4},
			},
		},
	}

	mc.recordMetric(m)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "histogram", metrics[0].Type)
	assert.Equal(t, 25.0, metrics[0].DataPoints[0].Value)
}

func TestMetricsCollector_RecordMetric_HistogramFloat64(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	m := metricdata.Metrics{
		Name: "test.histogram.float64",
		Data: metricdata.Histogram[float64]{
			DataPoints: []metricdata.HistogramDataPoint[float64]{
				{Sum: 7.5, Count: 3},
			},
		},
	}

	mc.recordMetric(m)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "histogram", metrics[0].Type)
	assert.InDelta(t, 2.5, metrics[0].DataPoints[0].Value, 1e-9)
}

func TestMetricsCollector_Stop(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))
	mc.Start(ctx)
	mc.Stop()
}

func TestMetricsCollector_Start_CollectsMetrics(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	mc := NewMetricsCollector(store, 50*time.Millisecond, WithMetricsCollectorClock(mock))
	baseline := mock.TimerCount()

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(mc.manualReader),
	)

	meter := provider.Meter("test")
	counter, err := meter.Int64Counter("test.start.counter")
	require.NoError(t, err)
	counter.Add(context.Background(), 1)

	ctx, cancel := context.WithCancelCause(context.Background())
	mc.Start(ctx)

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	mock.Advance(50 * time.Millisecond)

	require.Eventually(t, func() bool {
		return len(store.GetMetrics()) > 0
	}, time.Second, 5*time.Millisecond, "expected metrics to be collected via Start")

	cancel(fmt.Errorf("test: cleanup"))

	_ = provider.Shutdown(context.Background())
}

func TestMetricsCollector_Start_ContextCancellation(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	mc := NewMetricsCollector(store, 50*time.Millisecond, WithMetricsCollectorClock(mock))
	baseline := mock.TimerCount()

	ctx, cancel := context.WithCancelCause(context.Background())
	mc.Start(ctx)

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	cancel(fmt.Errorf("test: simulating cancelled context"))

	time.Sleep(10 * time.Millisecond)
}

func TestMetricsCollector_Shutdown(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	err := mc.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestMetricsCollector_RecordMetric_MultipleDataPoints(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	m := metricdata.Metrics{
		Name: "test.multi",
		Data: metricdata.Sum[int64]{
			DataPoints: []metricdata.DataPoint[int64]{
				{
					Value:      10,
					Attributes: attribute.NewSet(attribute.String("method", "GET")),
				},
				{
					Value:      20,
					Attributes: attribute.NewSet(attribute.String("method", "POST")),
				},
			},
		},
	}

	mc.recordMetric(m)

	metrics := store.GetMetrics()

	assert.GreaterOrEqual(t, len(metrics), 1)
}

func TestMetricsCollector_CollectWithRegisteredProvider(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	mc := NewMetricsCollector(store, DefaultMetricsCollectionInterval)

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(mc.manualReader),
	)
	defer func() { _ = provider.Shutdown(context.Background()) }()

	meter := provider.Meter("test")
	counter, err := meter.Int64Counter("test.requests")
	require.NoError(t, err)

	counter.Add(context.Background(), 5,
		otelmetric.WithAttributes(attribute.String("status", "200")),
	)

	err = mc.collect(context.Background())
	require.NoError(t, err)

	metrics := store.GetMetrics()
	require.NotEmpty(t, metrics)

	var found bool
	for _, m := range metrics {
		if m.Name == "test.requests" {
			found = true
			assert.Equal(t, "counter", m.Type)
			break
		}
	}
	assert.True(t, found, "expected to find test.requests metric")
}
