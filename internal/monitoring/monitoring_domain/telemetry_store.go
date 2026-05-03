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
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/wdk/clock"
)

var _ TelemetryProvider = (*TelemetryStore)(nil)

const (
	// DefaultMaxSpans is the default maximum number of spans to retain.
	// Increased from 1000 to support Routes panel aggregation.
	DefaultMaxSpans = 10000

	// DefaultMaxMetrics is the default maximum number of metrics to retain.
	// Increased from 500 to support per-path metric cardinality.
	DefaultMaxMetrics = 2000

	// DefaultMaxMetricAge is the default maximum age for metric data points.
	// Increased from 5 minutes to support longer-term trend analysis.
	DefaultMaxMetricAge = 15 * time.Minute
)

// InternalMetricDataPoint represents a single metric value at a point in time.
// This is the internal storage type; GetMetrics converts to domain types.
type InternalMetricDataPoint struct {
	// Timestamp is when the data point was recorded.
	Timestamp time.Time

	// Attributes holds key-value pairs of metadata for this data point.
	Attributes map[string]string

	// Value is the numeric measurement recorded at this data point.
	Value float64
}

// InternalMetricData represents a metric with its recent values.
// This is the internal storage type; GetMetrics converts it to domain types.
type InternalMetricData struct {
	// Name string // Name is the identifier for this metric.
	Name string

	// Description is a human-readable explanation of what this metric measures.
	Description string

	// Unit specifies the measurement unit for the metric value.
	Unit string

	// Type indicates the metric kind: counter, gauge, or histogram.
	Type string

	// DataPoints holds the recorded metric values, trimmed to MaxMetricAge.
	DataPoints []InternalMetricDataPoint
}

// InternalSpanEvent represents an event that occurred within a span.
type InternalSpanEvent struct {
	// Timestamp is when the event occurred.
	Timestamp time.Time

	// Attributes contains key-value pairs of metadata for this event.
	Attributes map[string]string

	// Name string // Name is the human-readable event identifier.
	Name string
}

// InternalSpanData represents a trace span.
// This is the internal storage type; GetSpans converts to domain types.
type InternalSpanData struct {
	// StartTime is when the span began.
	StartTime time.Time

	// EndTime is when the span finished.
	EndTime time.Time

	// Attributes maps attribute keys to their string values.
	Attributes map[string]string

	// TraceID string // TraceID is the unique trace identifier linking related spans.
	TraceID string

	// SpanID string // SpanID is the unique identifier for this span within the trace.
	SpanID string

	// ParentSpanID is the span ID of the parent span; empty for root spans.
	ParentSpanID string

	// Name string // Name is the span operation name.
	Name string

	// Kind specifies the span type such as client, server, or internal.
	Kind string

	// Status indicates the span outcome; "ERROR" marks failed spans.
	Status string

	// StatusMessage is the human-readable message explaining the span status.
	StatusMessage string

	// ServiceName is the name of the service that produced this span.
	ServiceName string

	// Events holds the collection of events recorded during the span's lifetime.
	Events []InternalSpanEvent

	// Duration is the total execution time of the span.
	Duration time.Duration
}

// StoreConfig holds settings for the telemetry store.
type StoreConfig struct {
	// Clock provides time functions; nil uses the real system clock.
	Clock clock.Clock

	// MaxSpans is the maximum number of spans to store; uses a ring buffer when full.
	MaxSpans int

	// MaxMetrics is the maximum number of metrics to store.
	//
	// New entries past the cap are dropped (fail-closed); existing
	// entries are not evicted. A value of 0 disables the cap.
	MaxMetrics int

	// MaxMetricAge is the maximum age for metric data points; older points are
	// trimmed during recording.
	MaxMetricAge time.Duration
}

// StoreOption configures the telemetry store.
type StoreOption func(*StoreConfig)

// TelemetryStore provides safe in-memory storage for metrics and spans.
// It implements TelemetryProvider and can be used by multiple goroutines.
type TelemetryStore struct {
	// clock provides the time source for recording metric timestamps.
	clock clock.Clock

	// metrics maps metric keys to their recorded data.
	metrics map[string]*InternalMetricData

	// spans stores recorded span data in a ring buffer.
	spans []InternalSpanData

	// config holds the store settings that control retention limits.
	config StoreConfig

	// spanIndex is the current position in the ring buffer for spans.
	spanIndex int

	// metricCapWarnOnce ensures the cap-hit warning is logged at most once
	// per store lifetime, preventing log floods when a misbehaving caller
	// drives high cardinality.
	metricCapWarnOnce sync.Once

	// metricsMutex sync.RWMutex // metricsMutex guards access to the metrics map.
	metricsMutex sync.RWMutex

	// spansMutex guards access to the spans slice.
	spansMutex sync.RWMutex
}

// NewTelemetryStore creates a new telemetry store.
//
// Takes opts (...StoreOption) which configures the store behaviour.
//
// Returns *TelemetryStore which is ready to collect metrics and spans.
func NewTelemetryStore(opts ...StoreOption) *TelemetryStore {
	config := StoreConfig{
		Clock:        nil,
		MaxSpans:     DefaultMaxSpans,
		MaxMetrics:   DefaultMaxMetrics,
		MaxMetricAge: DefaultMaxMetricAge,
	}

	for _, opt := range opts {
		opt(&config)
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	return &TelemetryStore{
		clock:             clk,
		metrics:           make(map[string]*InternalMetricData),
		spans:             make([]InternalSpanData, 0, config.MaxSpans),
		config:            config,
		spanIndex:         0,
		metricCapWarnOnce: sync.Once{},
		metricsMutex:      sync.RWMutex{},
		spansMutex:        sync.RWMutex{},
	}
}

// RecordMetric records a metric data point.
//
// Takes name (string) which identifies the metric.
// Takes description (string) which explains what the metric measures.
// Takes unit (string) which specifies the measurement unit.
// Takes metricType (string) which indicates the type of metric.
// Takes value (float64) which is the metric value to record.
// Takes attributes (map[string]string) which provides additional labels.
//
// Safe for concurrent use. Protected by a mutex.
//
// New entries are dropped (fail-closed) once the configured MaxMetrics
// cap is reached; MaxMetrics of 0 disables the cap. Existing entries
// continue to accumulate data points so cardinality cannot escape via
// later calls. The cap-hit is logged once per store lifetime to avoid
// log floods.
//
// TODO: the per-call DataPoints slice rebuild allocates fresh storage
// every recording. If profiling flags this on a hot path, switch to
// in-place compaction or a ring buffer.
func (s *TelemetryStore) RecordMetric(name, description, unit, metricType string, value float64, attributes map[string]string) {
	s.metricsMutex.Lock()
	defer s.metricsMutex.Unlock()

	key := name
	if len(attributes) > 0 {
		key = name + attributesToKey(attributes)
	}

	metric, exists := s.metrics[key]
	if !exists {
		if s.config.MaxMetrics > 0 && len(s.metrics) >= s.config.MaxMetrics {
			s.metricCapWarnOnce.Do(func() {
				log.Warn("Telemetry metric cap reached, dropping new entries",
					Int("max_metrics", s.config.MaxMetrics),
					String("dropped_metric", name),
				)
			})
			return
		}
		metric = &InternalMetricData{
			Name:        name,
			Description: description,
			Unit:        unit,
			Type:        metricType,
			DataPoints:  make([]InternalMetricDataPoint, 0),
		}
		s.metrics[key] = metric
	}

	now := s.clock.Now()
	metric.DataPoints = append(metric.DataPoints, InternalMetricDataPoint{
		Timestamp:  now,
		Attributes: attributes,
		Value:      value,
	})

	cutoff := now.Add(-s.config.MaxMetricAge)
	trimmed := make([]InternalMetricDataPoint, 0, len(metric.DataPoints))
	for _, dp := range metric.DataPoints {
		if dp.Timestamp.After(cutoff) {
			trimmed = append(trimmed, dp)
		}
	}
	metric.DataPoints = trimmed
}

// RecordSpan records a span to the telemetry store.
//
// Takes span (InternalSpanData) which contains the span data to record.
//
// Safe for concurrent use. Uses a mutex to protect the internal ring buffer.
func (s *TelemetryStore) RecordSpan(span InternalSpanData) {
	s.spansMutex.Lock()
	defer s.spansMutex.Unlock()

	if len(s.spans) < s.config.MaxSpans {
		s.spans = append(s.spans, span)
	} else {
		s.spans[s.spanIndex] = span
		s.spanIndex = (s.spanIndex + 1) % s.config.MaxSpans
	}
}

// GetMetrics returns all current metrics in domain format.
// Implements TelemetryProvider.
//
// Returns []MetricData which contains a snapshot of all stored metrics.
//
// Safe for concurrent use; holds a read lock for the duration of the call.
func (s *TelemetryStore) GetMetrics() []MetricData {
	s.metricsMutex.RLock()
	defer s.metricsMutex.RUnlock()

	result := make([]MetricData, 0, len(s.metrics))
	for _, m := range s.metrics {
		dataPoints := make([]MetricDataPoint, len(m.DataPoints))
		for i, dp := range m.DataPoints {
			dataPoints[i] = MetricDataPoint{
				TimestampMs: dp.Timestamp.UnixMilli(),
				Attributes:  dp.Attributes,
				Value:       dp.Value,
			}
		}

		result = append(result, MetricData{
			Name:        m.Name,
			Description: m.Description,
			Unit:        m.Unit,
			Type:        m.Type,
			DataPoints:  dataPoints,
		})
	}

	return result
}

// GetSpans returns recent spans in domain format.
// Implements TelemetryProvider.
//
// Takes limit (int) which specifies the maximum number of spans to return.
// A value of zero or less returns all spans.
// Takes errorsOnly (bool) which filters to only spans with error status.
//
// Returns []SpanData which contains spans in reverse chronological order.
//
// Safe for concurrent use; protected by a read lock.
func (s *TelemetryStore) GetSpans(limit int, errorsOnly bool) []SpanData {
	s.spansMutex.RLock()
	defer s.spansMutex.RUnlock()

	if limit <= 0 {
		limit = len(s.spans)
	}

	result := make([]SpanData, 0, min(limit, len(s.spans)))

	total := len(s.spans)
	if total == 0 {
		return result
	}

	startIndex := s.spanIndex - 1
	if startIndex < 0 {
		startIndex = total - 1
	}

	for i := 0; i < total && len(result) < limit; i++ {
		index := (startIndex - i + total) % total
		span := s.spans[index]

		if errorsOnly && span.Status != "ERROR" {
			continue
		}

		result = append(result, s.convertSpanToDomain(span))
	}

	return result
}

// GetSpanByTraceID returns all spans for a given trace ID in domain format.
// Implements TelemetryProvider.
//
// Takes traceID (string) which identifies the trace to retrieve spans for.
//
// Returns []SpanData which contains all spans matching the given trace ID.
//
// Safe for concurrent use; protected by a read lock.
func (s *TelemetryStore) GetSpanByTraceID(traceID string) []SpanData {
	s.spansMutex.RLock()
	defer s.spansMutex.RUnlock()

	result := make([]SpanData, 0)
	for i := range s.spans {
		if s.spans[i].TraceID == traceID {
			result = append(result, s.convertSpanToDomain(s.spans[i]))
		}
	}

	return result
}

// convertSpanToDomain converts internal span data to domain format.
//
// Takes span (InternalSpanData) which contains the internal span data to
// convert.
//
// Returns SpanData which is the converted span in domain format.
func (*TelemetryStore) convertSpanToDomain(span InternalSpanData) SpanData {
	events := make([]SpanEvent, len(span.Events))
	for i, e := range span.Events {
		events[i] = SpanEvent{
			Name:        e.Name,
			TimestampMs: e.Timestamp.UnixMilli(),
			Attributes:  e.Attributes,
		}
	}

	return SpanData{
		TraceID:       span.TraceID,
		SpanID:        span.SpanID,
		ParentSpanID:  span.ParentSpanID,
		Name:          span.Name,
		Kind:          span.Kind,
		Status:        span.Status,
		StatusMessage: span.StatusMessage,
		ServiceName:   span.ServiceName,
		StartTimeMs:   span.StartTime.UnixMilli(),
		EndTimeMs:     span.EndTime.UnixMilli(),
		DurationNs:    span.Duration.Nanoseconds(),
		Attributes:    span.Attributes,
		Events:        events,
	}
}

// WithMaxSpans sets the maximum number of spans to retain.
//
// Takes n (int) which specifies the maximum span count.
//
// Returns StoreOption which configures the span limit on a StoreConfig.
func WithMaxSpans(n int) StoreOption {
	return func(c *StoreConfig) {
		c.MaxSpans = n
	}
}

// WithMaxMetrics sets the maximum number of metrics to retain.
//
// Takes n (int) which specifies the maximum number of metrics to keep.
//
// Returns StoreOption which configures the metric retention limit.
func WithMaxMetrics(n int) StoreOption {
	return func(c *StoreConfig) {
		c.MaxMetrics = n
	}
}

// WithMaxMetricAge sets the maximum age for metric data points.
//
// Takes d (time.Duration) which specifies how long metric data points are
// retained before expiry.
//
// Returns StoreOption which configures the maximum metric age on a store.
func WithMaxMetricAge(d time.Duration) StoreOption {
	return func(c *StoreConfig) {
		c.MaxMetricAge = d
	}
}

// WithStoreClock sets the clock for the telemetry store.
//
// Takes clk (clock.Clock) which provides the time source for timestamps.
//
// Returns StoreOption which configures the store to use the given clock.
func WithStoreClock(clk clock.Clock) StoreOption {
	return func(c *StoreConfig) {
		c.Clock = clk
	}
}

// attributesToKey creates a unique key suffix from attributes.
//
// Iteration order of a Go map is randomised, so the suffix is built
// from keys collected via slices.Sorted(maps.Keys(...)) to guarantee
// the same attribute set always produces the same key. Without this,
// callers re-passing identical attributes would mint fresh map entries
// indefinitely (cardinality bomb).
//
// Takes attrs (map[string]string) which contains the key-value pairs to encode.
//
// Returns string which is the encoded key suffix, or empty if attrs is empty.
func attributesToKey(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}

	keys := slices.Sorted(maps.Keys(attrs))

	var builder strings.Builder
	for _, k := range keys {
		_ = builder.WriteByte('|')
		builder.WriteString(k)
		_ = builder.WriteByte('=')
		builder.WriteString(attrs[k])
	}
	return builder.String()
}
