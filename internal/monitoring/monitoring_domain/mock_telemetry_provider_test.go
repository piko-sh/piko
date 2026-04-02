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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockTelemetryProvider_GetMetrics(t *testing.T) {
	t.Parallel()

	t.Run("nil GetMetricsFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockTelemetryProvider{}

		result := mock.GetMetrics()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetMetricsCallCount))
	})

	t.Run("delegates to GetMetricsFunc", func(t *testing.T) {
		t.Parallel()

		expected := []MetricData{
			{
				Name:        "http.requests",
				Description: "Total HTTP requests",
				Unit:        "count",
				Type:        "counter",
				DataPoints: []MetricDataPoint{
					{Value: 42.0, TimestampMs: 1700000000000},
				},
			},
			{
				Name: "cpu.usage",
				Type: "gauge",
				DataPoints: []MetricDataPoint{
					{Value: 78.5, TimestampMs: 1700000001000, Attributes: map[string]string{"host": "node-1"}},
				},
			},
		}

		mock := &MockTelemetryProvider{
			GetMetricsFunc: func() []MetricData {
				return expected
			},
		}

		result := mock.GetMetrics()

		require.Len(t, result, 2)
		assert.Equal(t, expected, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetMetricsCallCount))
	})
}

func TestMockTelemetryProvider_GetSpans(t *testing.T) {
	t.Parallel()

	t.Run("nil GetSpansFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockTelemetryProvider{}

		result := mock.GetSpans(10, true)

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetSpansCallCount))
	})

	t.Run("delegates to GetSpansFunc", func(t *testing.T) {
		t.Parallel()

		expected := []SpanData{
			{
				TraceID:     "trace-abc",
				SpanID:      "span-123",
				Name:        "GET /api/v1/users",
				Kind:        "SERVER",
				Status:      "OK",
				StartTimeMs: 1700000000000,
				EndTimeMs:   1700000000500,
				DurationNs:  500000000,
			},
		}

		var capturedLimit int
		var capturedErrorsOnly bool

		mock := &MockTelemetryProvider{
			GetSpansFunc: func(limit int, errorsOnly bool) []SpanData {
				capturedLimit = limit
				capturedErrorsOnly = errorsOnly
				return expected
			},
		}

		result := mock.GetSpans(25, true)

		require.Len(t, result, 1)
		assert.Equal(t, expected, result)
		assert.Equal(t, 25, capturedLimit)
		assert.True(t, capturedErrorsOnly)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetSpansCallCount))
	})
}

func TestMockTelemetryProvider_GetSpanByTraceID(t *testing.T) {
	t.Parallel()

	t.Run("nil GetSpanByTraceIDFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockTelemetryProvider{}

		result := mock.GetSpanByTraceID("trace-xyz")

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetSpanByTraceIDCallCount))
	})

	t.Run("delegates to GetSpanByTraceIDFunc", func(t *testing.T) {
		t.Parallel()

		expected := []SpanData{
			{TraceID: "trace-abc", SpanID: "span-1", Name: "root"},
			{TraceID: "trace-abc", SpanID: "span-2", Name: "child", ParentSpanID: "span-1"},
		}

		var capturedTraceID string

		mock := &MockTelemetryProvider{
			GetSpanByTraceIDFunc: func(traceID string) []SpanData {
				capturedTraceID = traceID
				return expected
			},
		}

		result := mock.GetSpanByTraceID("trace-abc")

		require.Len(t, result, 2)
		assert.Equal(t, expected, result)
		assert.Equal(t, "trace-abc", capturedTraceID)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetSpanByTraceIDCallCount))
	})
}

func TestMockTelemetryProvider_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockTelemetryProvider

	metrics := mock.GetMetrics()
	assert.Nil(t, metrics)

	spans := mock.GetSpans(5, false)
	assert.Nil(t, spans)

	traceSpans := mock.GetSpanByTraceID("any-trace")
	assert.Nil(t, traceSpans)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetMetricsCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetSpansCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetSpanByTraceIDCallCount))
}

func TestMockTelemetryProvider_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockTelemetryProvider{
		GetMetricsFunc: func() []MetricData {
			return []MetricData{{Name: "m1"}}
		},
		GetSpansFunc: func(limit int, errorsOnly bool) []SpanData {
			return []SpanData{{Name: "s1"}}
		},
		GetSpanByTraceIDFunc: func(traceID string) []SpanData {
			return []SpanData{{TraceID: traceID}}
		},
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_ = mock.GetMetrics()
			_ = mock.GetSpans(10, false)
			_ = mock.GetSpanByTraceID("trace-concurrent")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetMetricsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetSpansCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetSpanByTraceIDCallCount))
}

func TestMockTelemetryProvider_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ TelemetryProvider = (*MockTelemetryProvider)(nil)
}
