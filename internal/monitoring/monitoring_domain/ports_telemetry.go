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

// MetricData holds a metric and its recent values.
type MetricData struct {
	// Name is the metric identifier used in protocol buffer responses.
	Name string `json:"name"`

	// Description provides a human-readable explanation of the metric.
	Description string `json:"description,omitempty"`

	// Unit is the measurement unit for the metric values.
	Unit string `json:"unit,omitempty"`

	// Type is the metric type, such as "counter" or "gauge".
	Type string `json:"type"`

	// DataPoints holds the time-series data for this metric.
	DataPoints []MetricDataPoint `json:"data_points"`
}

// MetricDataPoint represents a single metric value at a point in time.
type MetricDataPoint struct {
	// Attributes contains key-value pairs of metadata for this data point.
	Attributes map[string]string `json:"attributes,omitempty"`

	// TimestampMs is the data point timestamp in milliseconds since Unix epoch.
	TimestampMs int64 `json:"timestamp_ms"`

	// Value is the numeric measurement for this data point.
	Value float64 `json:"value"`
}

// SpanData holds the details of a single trace span for monitoring.
type SpanData struct {
	// Attributes holds key-value pairs of span metadata.
	Attributes map[string]string `json:"attributes,omitempty"`

	// Kind is the span type (e.g. "client", "server", "internal").
	Kind string `json:"kind"`

	// TraceID is the unique identifier for the trace this span belongs to.
	TraceID string `json:"trace_id"`

	// SpanID is the unique identifier for this span within the trace.
	SpanID string `json:"span_id"`

	// ParentSpanID is the span ID of the parent span; empty for root spans.
	ParentSpanID string `json:"parent_span_id,omitempty"`

	// Name is the human-readable identifier for the span.
	Name string `json:"name"`

	// Status indicates the outcome of the span (e.g. "OK", "ERROR").
	Status string `json:"status"`

	// StatusMessage provides additional status details.
	StatusMessage string `json:"status_message,omitempty"`

	// ServiceName identifies the service that produced this span.
	ServiceName string `json:"service_name,omitempty"`

	// Events contains the timestamped events recorded during the span.
	Events []SpanEvent `json:"events,omitempty"`

	// StartTimeMs is the span start time in milliseconds since Unix epoch.
	StartTimeMs int64 `json:"start_time_ms"`

	// EndTimeMs is the span end time in milliseconds since Unix epoch.
	EndTimeMs int64 `json:"end_time_ms"`

	// DurationNs is the span duration in nanoseconds.
	DurationNs int64 `json:"duration_ns"`
}

// SpanEvent represents a notable occurrence within a span.
type SpanEvent struct {
	// Attributes contains key-value pairs of metadata for this event.
	Attributes map[string]string `json:"attributes,omitempty"`

	// Name is the identifier for this span event.
	Name string `json:"name"`

	// TimestampMs is when the event occurred, in milliseconds since Unix epoch.
	TimestampMs int64 `json:"timestamp_ms"`
}
