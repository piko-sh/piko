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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

func TestTelemetryStore_RecordMetric(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		recordMetrics func(store *TelemetryStore)
		expectedAttrs map[string]string
		name          string
		expectedName  string
		expectedCount int
		expectedValue float64
	}{
		{
			name: "records single metric",
			recordMetrics: func(store *TelemetryStore) {
				store.RecordMetric("test.metric", "A test metric", "count", "counter", 42.0, nil)
			},
			expectedCount: 1,
			expectedName:  "test.metric",
			expectedValue: 42.0,
			expectedAttrs: nil,
		},
		{
			name: "records metric with attributes",
			recordMetrics: func(store *TelemetryStore) {
				attrs := map[string]string{"method": "GET", "path": "/api"}
				store.RecordMetric("http.requests", "HTTP requests", "count", "counter", 10.0, attrs)
			},
			expectedCount: 1,
			expectedName:  "http.requests",
			expectedValue: 10.0,
			expectedAttrs: map[string]string{"method": "GET", "path": "/api"},
		},
		{
			name: "different attributes create separate metrics",
			recordMetrics: func(store *TelemetryStore) {
				store.RecordMetric("http.requests", "HTTP requests", "count", "counter", 5.0,
					map[string]string{"method": "GET"})
				store.RecordMetric("http.requests", "HTTP requests", "count", "counter", 3.0,
					map[string]string{"method": "POST"})
			},
			expectedCount: 2,
			expectedName:  "",
			expectedValue: 0,
			expectedAttrs: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
			store := NewTelemetryStore(WithStoreClock(mockClock))

			tc.recordMetrics(store)

			metrics := store.GetMetrics()
			if len(metrics) != tc.expectedCount {
				t.Errorf("expected %d metrics, got %d", tc.expectedCount, len(metrics))
			}

			if tc.expectedName != "" && len(metrics) > 0 {
				found := false
				for _, m := range metrics {
					if m.Name == tc.expectedName {
						found = true
						if len(m.DataPoints) == 0 {
							t.Error("expected at least one data point")
							break
						}
						dp := m.DataPoints[0]
						if dp.Value != tc.expectedValue {
							t.Errorf("expected value %v, got %v", tc.expectedValue, dp.Value)
						}
						if tc.expectedAttrs != nil {
							for k, v := range tc.expectedAttrs {
								if dp.Attributes[k] != v {
									t.Errorf("expected attribute %s=%s, got %s", k, v, dp.Attributes[k])
								}
							}
						}
						break
					}
				}
				if !found {
					t.Errorf("metric %s not found", tc.expectedName)
				}
			}
		})
	}
}

func TestTelemetryStore_MetricAgeTrimming(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := NewTelemetryStore(
		WithStoreClock(mockClock),
		WithMaxMetricAge(5*time.Minute),
	)

	store.RecordMetric("test.metric", "", "", "gauge", 1.0, nil)

	mockClock.Advance(3 * time.Minute)
	store.RecordMetric("test.metric", "", "", "gauge", 2.0, nil)

	metrics := store.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if len(metrics[0].DataPoints) != 2 {
		t.Errorf("expected 2 data points, got %d", len(metrics[0].DataPoints))
	}

	mockClock.Advance(3 * time.Minute)
	store.RecordMetric("test.metric", "", "", "gauge", 3.0, nil)

	metrics = store.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	if len(metrics[0].DataPoints) != 2 {
		t.Errorf("expected 2 data points after trimming, got %d", len(metrics[0].DataPoints))
	}
}

func TestTelemetryStore_RecordSpan(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore(WithMaxSpans(10))

	span := InternalSpanData{
		TraceID:       "trace-123",
		SpanID:        "span-456",
		ParentSpanID:  "",
		Name:          "test-span",
		Kind:          "SERVER",
		Status:        "OK",
		StatusMessage: "",
		ServiceName:   "test-service",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(100 * time.Millisecond),
		Duration:      100 * time.Millisecond,
		Attributes:    map[string]string{"key": "value"},
	}
	store.RecordSpan(span)

	spans := store.GetSpans(10, false)
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].TraceID != "trace-123" {
		t.Errorf("expected trace ID trace-123, got %s", spans[0].TraceID)
	}
	if spans[0].Name != "test-span" {
		t.Errorf("expected name test-span, got %s", spans[0].Name)
	}
}

func TestTelemetryStore_SpanRingBuffer(t *testing.T) {
	t.Parallel()

	maxSpans := 5
	store := NewTelemetryStore(WithMaxSpans(maxSpans))

	for i := range 10 {
		span := InternalSpanData{
			TraceID:   "trace",
			SpanID:    "span",
			Name:      "span-" + string(rune('0'+i)),
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		store.RecordSpan(span)
	}

	spans := store.GetSpans(100, false)
	if len(spans) != maxSpans {
		t.Errorf("expected %d spans (ring buffer size), got %d", maxSpans, len(spans))
	}

	expectedNames := []string{"span-9", "span-8", "span-7", "span-6", "span-5"}
	for i, expected := range expectedNames {
		if i < len(spans) && spans[i].Name != expected {
			t.Errorf("at index %d: expected %s, got %s", i, expected, spans[i].Name)
		}
	}
}

func TestTelemetryStore_GetSpansWithLimit(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore(WithMaxSpans(100))

	for i := range 20 {
		span := InternalSpanData{
			TraceID:   "trace",
			SpanID:    "span",
			Name:      "span",
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		_ = i
		store.RecordSpan(span)
	}

	spans := store.GetSpans(5, false)
	if len(spans) != 5 {
		t.Errorf("expected 5 spans with limit, got %d", len(spans))
	}
}

func TestTelemetryStore_GetSpansErrorsOnly(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore(WithMaxSpans(100))

	for i := range 10 {
		status := "OK"
		if i%2 == 0 {
			status = "ERROR"
		}
		span := InternalSpanData{
			TraceID:   "trace",
			SpanID:    "span",
			Name:      "span",
			Status:    status,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		store.RecordSpan(span)
	}

	spans := store.GetSpans(100, true)
	if len(spans) != 5 {
		t.Errorf("expected 5 error spans, got %d", len(spans))
	}

	for _, s := range spans {
		if s.Status != "ERROR" {
			t.Errorf("expected ERROR status, got %s", s.Status)
		}
	}
}

func TestTelemetryStore_GetSpanByTraceID(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore(WithMaxSpans(100))

	for i := range 5 {
		for j := range 3 {
			span := InternalSpanData{
				TraceID:   "trace-" + string(rune('A'+i)),
				SpanID:    "span-" + string(rune('0'+j)),
				Name:      "span",
				StartTime: time.Now(),
				EndTime:   time.Now(),
			}
			store.RecordSpan(span)
		}
	}

	spans := store.GetSpanByTraceID("trace-C")
	if len(spans) != 3 {
		t.Errorf("expected 3 spans for trace-C, got %d", len(spans))
	}

	for _, s := range spans {
		if s.TraceID != "trace-C" {
			t.Errorf("expected trace ID trace-C, got %s", s.TraceID)
		}
	}

	spans = store.GetSpanByTraceID("non-existent")
	if len(spans) != 0 {
		t.Errorf("expected 0 spans for non-existent trace, got %d", len(spans))
	}
}

func TestTelemetryStore_SpanEventConversion(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore()

	eventTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	span := InternalSpanData{
		TraceID:   "trace-123",
		SpanID:    "span-456",
		Name:      "test-span",
		StartTime: eventTime,
		EndTime:   eventTime.Add(time.Second),
		Events: []InternalSpanEvent{
			{
				Name:       "event-1",
				Timestamp:  eventTime.Add(100 * time.Millisecond),
				Attributes: map[string]string{"key": "value"},
			},
		},
	}
	store.RecordSpan(span)

	spans := store.GetSpans(1, false)
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if len(spans[0].Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spans[0].Events))
	}

	event := spans[0].Events[0]
	if event.Name != "event-1" {
		t.Errorf("expected event name event-1, got %s", event.Name)
	}
	if event.Attributes["key"] != "value" {
		t.Errorf("expected event attribute key=value, got %s", event.Attributes["key"])
	}
}

func TestWithMaxMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{
			name:     "sets positive value",
			n:        100,
			expected: 100,
		},
		{
			name:     "sets zero",
			n:        0,
			expected: 0,
		},
		{
			name:     "sets large value",
			n:        50000,
			expected: 50000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := StoreConfig{
				Clock:        nil,
				MaxSpans:     0,
				MaxMetrics:   0,
				MaxMetricAge: 0,
			}
			opt := WithMaxMetrics(tt.n)
			opt(&config)

			assert.Equal(t, tt.expected, config.MaxMetrics)
		})
	}
}

func TestTelemetryStore_GetSpans_ZeroLimit(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore(WithMaxSpans(100))

	for i := range 5 {
		span := InternalSpanData{
			TraceID:       "trace-" + string(rune('A'+i)),
			SpanID:        "span-" + string(rune('0'+i)),
			ParentSpanID:  "",
			Name:          "span",
			Kind:          "SERVER",
			Status:        "OK",
			StatusMessage: "",
			ServiceName:   "test",
			StartTime:     time.Now(),
			EndTime:       time.Now(),
			Duration:      time.Millisecond,
			Attributes:    nil,
			Events:        nil,
		}
		store.RecordSpan(span)
	}

	spans := store.GetSpans(0, false)
	assert.Len(t, spans, 5)
}

func TestTelemetryStore_GetSpans_NegativeLimit(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore(WithMaxSpans(100))

	for i := range 3 {
		span := InternalSpanData{
			TraceID:       "trace-" + string(rune('A'+i)),
			SpanID:        "span-" + string(rune('0'+i)),
			ParentSpanID:  "",
			Name:          "span",
			Kind:          "SERVER",
			Status:        "OK",
			StatusMessage: "",
			ServiceName:   "test",
			StartTime:     time.Now(),
			EndTime:       time.Now(),
			Duration:      time.Millisecond,
			Attributes:    nil,
			Events:        nil,
		}
		store.RecordSpan(span)
	}

	spans := store.GetSpans(-1, false)
	assert.Len(t, spans, 3)
}

func TestTelemetryStore_GetSpans_Empty(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore(WithMaxSpans(100))

	spans := store.GetSpans(10, false)
	assert.Empty(t, spans)
}

func TestTelemetryStore_RecordSpan_WithEvents(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := NewTelemetryStore(
		WithStoreClock(mockClock),
		WithMaxSpans(10),
	)

	eventTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	span := InternalSpanData{
		TraceID:       "trace-events",
		SpanID:        "span-events",
		ParentSpanID:  "parent-span",
		Name:          "with-events",
		Kind:          "CLIENT",
		Status:        "ERROR",
		StatusMessage: "something failed",
		ServiceName:   "event-service",
		StartTime:     eventTime,
		EndTime:       eventTime.Add(500 * time.Millisecond),
		Duration:      500 * time.Millisecond,
		Attributes:    map[string]string{"http.method": "GET", "http.path": "/api/v1"},
		Events: []InternalSpanEvent{
			{
				Name:       "exception",
				Timestamp:  eventTime.Add(100 * time.Millisecond),
				Attributes: map[string]string{"exception.type": "NullPointerException"},
			},
			{
				Name:       "retry",
				Timestamp:  eventTime.Add(200 * time.Millisecond),
				Attributes: map[string]string{"attempt": "2"},
			},
		},
	}
	store.RecordSpan(span)

	spans := store.GetSpans(1, false)
	require.Len(t, spans, 1)

	s := spans[0]
	assert.Equal(t, "trace-events", s.TraceID)
	assert.Equal(t, "span-events", s.SpanID)
	assert.Equal(t, "parent-span", s.ParentSpanID)
	assert.Equal(t, "with-events", s.Name)
	assert.Equal(t, "CLIENT", s.Kind)
	assert.Equal(t, "ERROR", s.Status)
	assert.Equal(t, "something failed", s.StatusMessage)
	assert.Equal(t, "event-service", s.ServiceName)
	assert.Equal(t, eventTime.UnixMilli(), s.StartTimeMs)
	assert.Equal(t, eventTime.Add(500*time.Millisecond).UnixMilli(), s.EndTimeMs)
	assert.Equal(t, (500 * time.Millisecond).Nanoseconds(), s.DurationNs)
	assert.Equal(t, "GET", s.Attributes["http.method"])
	assert.Equal(t, "/api/v1", s.Attributes["http.path"])

	require.Len(t, s.Events, 2)
	assert.Equal(t, "exception", s.Events[0].Name)
	assert.Equal(t, "NullPointerException", s.Events[0].Attributes["exception.type"])
	assert.Equal(t, "retry", s.Events[1].Name)
}

func TestTelemetryStore_RecordMetric_Accumulates(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := NewTelemetryStore(
		WithStoreClock(mockClock),
		WithMaxMetricAge(10*time.Minute),
	)

	store.RecordMetric("cpu.usage", "CPU usage", "percent", "gauge", 45.0, nil)

	mockClock.Advance(time.Minute)
	store.RecordMetric("cpu.usage", "CPU usage", "percent", "gauge", 55.0, nil)

	mockClock.Advance(time.Minute)
	store.RecordMetric("cpu.usage", "CPU usage", "percent", "gauge", 65.0, nil)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)

	assert.Equal(t, "cpu.usage", metrics[0].Name)
	assert.Equal(t, "CPU usage", metrics[0].Description)
	assert.Equal(t, "percent", metrics[0].Unit)
	assert.Equal(t, "gauge", metrics[0].Type)
	assert.Len(t, metrics[0].DataPoints, 3)
}

func TestTelemetryStore_AttributesToKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		attrs   map[string]string
		name    string
		isEmpty bool
	}{
		{
			name:    "nil attributes",
			attrs:   nil,
			isEmpty: true,
		},
		{
			name:    "empty attributes",
			attrs:   map[string]string{},
			isEmpty: true,
		},
		{
			name:    "single attribute",
			attrs:   map[string]string{"key": "value"},
			isEmpty: false,
		},
		{
			name:    "multiple attributes",
			attrs:   map[string]string{"a": "1", "b": "2"},
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := attributesToKey(tt.attrs)
			if tt.isEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Contains(t, result, "|")
				assert.Contains(t, result, "=")
			}
		})
	}
}

func TestTelemetryStore_SpanRingBuffer_WrapsAround(t *testing.T) {
	t.Parallel()

	maxSpans := 3
	store := NewTelemetryStore(WithMaxSpans(maxSpans))

	for i := range 7 {
		span := InternalSpanData{
			TraceID:       "trace",
			SpanID:        "span",
			ParentSpanID:  "",
			Name:          "span-" + string(rune('A'+i)),
			Kind:          "SERVER",
			Status:        "OK",
			StatusMessage: "",
			ServiceName:   "test",
			StartTime:     time.Now(),
			EndTime:       time.Now(),
			Duration:      time.Millisecond,
			Attributes:    nil,
			Events:        nil,
		}
		store.RecordSpan(span)
	}

	spans := store.GetSpans(0, false)
	assert.Len(t, spans, maxSpans)

	names := make([]string, len(spans))
	for i, s := range spans {
		names[i] = s.Name
	}

	assert.Contains(t, names, "span-G")
	assert.Contains(t, names, "span-F")
	assert.Contains(t, names, "span-E")
}

func TestTelemetryStore_GetSpans_ErrorsOnlyWithLimit(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore(WithMaxSpans(100))

	for i := range 20 {
		status := "OK"
		if i%3 == 0 {
			status = "ERROR"
		}
		span := InternalSpanData{
			TraceID:       "trace",
			SpanID:        "span",
			ParentSpanID:  "",
			Name:          "span",
			Kind:          "SERVER",
			Status:        status,
			StatusMessage: "",
			ServiceName:   "test",
			StartTime:     time.Now(),
			EndTime:       time.Now(),
			Duration:      time.Millisecond,
			Attributes:    nil,
			Events:        nil,
		}
		store.RecordSpan(span)
	}

	spans := store.GetSpans(2, true)
	assert.Len(t, spans, 2)

	for _, s := range spans {
		assert.Equal(t, "ERROR", s.Status)
	}
}

func TestTelemetryStore_GetMetrics_MultipleMetricTypes(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := NewTelemetryStore(WithStoreClock(mockClock))

	store.RecordMetric("http.request.count", "Total requests", "count", "counter", 100.0, nil)
	store.RecordMetric("http.request.duration", "Request duration", "ms", "histogram", 42.5, nil)
	store.RecordMetric("memory.usage", "Memory usage", "bytes", "gauge", 1024.0, nil)

	metrics := store.GetMetrics()
	assert.Len(t, metrics, 3)

	metricNames := make(map[string]bool)
	for _, m := range metrics {
		metricNames[m.Name] = true
	}

	assert.True(t, metricNames["http.request.count"])
	assert.True(t, metricNames["http.request.duration"])
	assert.True(t, metricNames["memory.usage"])
}

func TestTelemetryStore_DefaultConfig(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore()

	require.NotNil(t, store)
	assert.Equal(t, DefaultMaxSpans, store.config.MaxSpans)
	assert.Equal(t, DefaultMaxMetrics, store.config.MaxMetrics)
	assert.Equal(t, DefaultMaxMetricAge, store.config.MaxMetricAge)
	assert.NotNil(t, store.clock)
	assert.NotNil(t, store.metrics)
	assert.NotNil(t, store.spans)
	assert.Equal(t, 0, store.spanIndex)
}

func TestTelemetryStore_ConvertSpanToDomain_AllFields(t *testing.T) {
	t.Parallel()

	store := NewTelemetryStore()

	startTime := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Second)

	span := InternalSpanData{
		TraceID:       "abc123",
		SpanID:        "def456",
		ParentSpanID:  "parent789",
		Name:          "GET /api",
		Kind:          "SERVER",
		Status:        "OK",
		StatusMessage: "success",
		ServiceName:   "api-gateway",
		StartTime:     startTime,
		EndTime:       endTime,
		Duration:      2 * time.Second,
		Attributes:    map[string]string{"http.status_code": "200"},
		Events:        nil,
	}

	result := store.convertSpanToDomain(span)

	assert.Equal(t, "abc123", result.TraceID)
	assert.Equal(t, "def456", result.SpanID)
	assert.Equal(t, "parent789", result.ParentSpanID)
	assert.Equal(t, "GET /api", result.Name)
	assert.Equal(t, "SERVER", result.Kind)
	assert.Equal(t, "OK", result.Status)
	assert.Equal(t, "success", result.StatusMessage)
	assert.Equal(t, "api-gateway", result.ServiceName)
	assert.Equal(t, startTime.UnixMilli(), result.StartTimeMs)
	assert.Equal(t, endTime.UnixMilli(), result.EndTimeMs)
	assert.Equal(t, (2 * time.Second).Nanoseconds(), result.DurationNs)
	assert.Equal(t, "200", result.Attributes["http.status_code"])
	assert.Empty(t, result.Events)
}

func TestRecordMetric_DeterministicKeyForSameAttributes(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := NewTelemetryStore(WithStoreClock(mockClock))

	attrs := map[string]string{
		"alpha":   "one",
		"beta":    "two",
		"gamma":   "three",
		"delta":   "four",
		"epsilon": "five",
	}

	for range 1000 {
		store.RecordMetric("http.requests", "HTTP requests", "count", "counter", 1.0, attrs)
	}

	store.metricsMutex.RLock()
	storedCount := len(store.metrics)
	store.metricsMutex.RUnlock()

	assert.Equal(t, 1, storedCount, "identical attributes must collapse to a single metric entry")

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Len(t, metrics[0].DataPoints, 1000, "each call should append one data point to the single entry")
}

func TestRecordMetric_EnforcesMaxMetricsCap(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := NewTelemetryStore(
		WithStoreClock(mockClock),
		WithMaxMetrics(2),
	)

	store.RecordMetric("first", "", "", "counter", 1.0, nil)
	store.RecordMetric("second", "", "", "counter", 2.0, nil)
	store.RecordMetric("third", "", "", "counter", 3.0, nil)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 2, "third metric should be dropped because cap is 2")

	names := make(map[string]float64)
	for _, m := range metrics {
		require.Len(t, m.DataPoints, 1)
		names[m.Name] = m.DataPoints[0].Value
	}

	assert.Contains(t, names, "first", "first metric must survive")
	assert.Contains(t, names, "second", "second metric must survive")
	assert.NotContains(t, names, "third", "third metric must be dropped, not evict an existing entry")
	assert.InDelta(t, 1.0, names["first"], 0.0001)
	assert.InDelta(t, 2.0, names["second"], 0.0001)
}

func TestRecordMetric_CapAdmitsExistingEntryDataPoints(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := NewTelemetryStore(
		WithStoreClock(mockClock),
		WithMaxMetrics(1),
	)

	store.RecordMetric("seeded", "", "", "gauge", 1.0, nil)
	store.RecordMetric("rejected", "", "", "gauge", 99.0, nil)
	store.RecordMetric("seeded", "", "", "gauge", 2.0, nil)
	store.RecordMetric("seeded", "", "", "gauge", 3.0, nil)

	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "seeded", metrics[0].Name)
	assert.Len(t, metrics[0].DataPoints, 3, "existing entry must keep accepting data points after cap hit")
}

func TestAttributesToKey_OrderIndependent(t *testing.T) {
	t.Parallel()

	first := map[string]string{}
	first["zeta"] = "z"
	first["alpha"] = "a"
	first["mu"] = "m"
	first["kappa"] = "k"

	second := map[string]string{}
	second["alpha"] = "a"
	second["mu"] = "m"
	second["kappa"] = "k"
	second["zeta"] = "z"

	assert.Equal(t, attributesToKey(first), attributesToKey(second),
		"different insertion orders for the same key/value pairs must yield identical keys")

	keyFirst := attributesToKey(first)
	for range 50 {
		assert.Equal(t, keyFirst, attributesToKey(first),
			"repeated calls on the same map must be deterministic")
	}
}

func TestRecordMetric_NoCapWhenZero(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := NewTelemetryStore(
		WithStoreClock(mockClock),
		WithMaxMetrics(0),
	)

	const distinctMetrics = 250
	for index := range distinctMetrics {
		store.RecordMetric("metric.unbounded."+strconv.Itoa(index), "", "", "counter", float64(index), nil)
	}

	metrics := store.GetMetrics()
	assert.Len(t, metrics, distinctMetrics, "MaxMetrics of 0 should disable the cap")
}
