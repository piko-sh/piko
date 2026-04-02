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

import "sync/atomic"

// MockTelemetryProvider is a test double for TelemetryProvider where nil
// function fields return zero values and call counts are tracked atomically.
type MockTelemetryProvider struct {
	// GetMetricsFunc is the function called by
	// GetMetrics.
	GetMetricsFunc func() []MetricData

	// GetSpansFunc is the function called by GetSpans.
	GetSpansFunc func(limit int, errorsOnly bool) []SpanData

	// GetSpanByTraceIDFunc is the function called by
	// GetSpanByTraceID.
	GetSpanByTraceIDFunc func(traceID string) []SpanData

	// GetMetricsCallCount tracks how many times
	// GetMetrics was called.
	GetMetricsCallCount int64

	// GetSpansCallCount tracks how many times GetSpans
	// was called.
	GetSpansCallCount int64

	// GetSpanByTraceIDCallCount tracks how many times
	// GetSpanByTraceID was called.
	GetSpanByTraceIDCallCount int64
}

var _ TelemetryProvider = (*MockTelemetryProvider)(nil)

// GetMetrics delegates to GetMetricsFunc if set.
//
// Returns nil if GetMetricsFunc is nil.
func (m *MockTelemetryProvider) GetMetrics() []MetricData {
	atomic.AddInt64(&m.GetMetricsCallCount, 1)
	if m.GetMetricsFunc != nil {
		return m.GetMetricsFunc()
	}
	return nil
}

// GetSpans delegates to GetSpansFunc if set.
//
// Takes limit (int) which caps the number of spans returned.
// Takes errorsOnly (bool) which filters to only error spans when true.
//
// Returns nil if GetSpansFunc is nil.
func (m *MockTelemetryProvider) GetSpans(limit int, errorsOnly bool) []SpanData {
	atomic.AddInt64(&m.GetSpansCallCount, 1)
	if m.GetSpansFunc != nil {
		return m.GetSpansFunc(limit, errorsOnly)
	}
	return nil
}

// GetSpanByTraceID delegates to GetSpanByTraceIDFunc if set.
//
// Takes traceID (string) which identifies the trace to look up.
//
// Returns nil if GetSpanByTraceIDFunc is nil.
func (m *MockTelemetryProvider) GetSpanByTraceID(traceID string) []SpanData {
	atomic.AddInt64(&m.GetSpanByTraceIDCallCount, 1)
	if m.GetSpanByTraceIDFunc != nil {
		return m.GetSpanByTraceIDFunc(traceID)
	}
	return nil
}
