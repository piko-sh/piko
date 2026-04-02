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

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

var _ sdktrace.SpanProcessor = (*SpanProcessor)(nil)

// SpanProcessor stores completed spans in a TelemetryStore.
// It implements sdktrace.SpanProcessor.
type SpanProcessor struct {
	// store records telemetry spans for later retrieval.
	store *monitoring_domain.TelemetryStore
}

// NewSpanProcessor creates a new span processor that writes to the given store.
//
// Takes store (*monitoring_domain.TelemetryStore) which receives the processed
// span data.
//
// Returns *SpanProcessor which is ready to process spans.
func NewSpanProcessor(store *monitoring_domain.TelemetryStore) *SpanProcessor {
	return &SpanProcessor{store: store}
}

// OnStart is called when a span starts but performs no action.
func (*SpanProcessor) OnStart(_ context.Context, _ sdktrace.ReadWriteSpan) {
}

// OnEnd is called when a span ends.
//
// Takes s (sdktrace.ReadOnlySpan) which provides the completed span data to
// process.
func (p *SpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	var parentSpanID string
	if s.Parent().HasSpanID() {
		parentSpanID = s.Parent().SpanID().String()
	}

	var serviceName string
	if resource := s.Resource(); resource != nil {
		for _, attr := range resource.Attributes() {
			if string(attr.Key) == "service.name" {
				serviceName = attr.Value.AsString()
				break
			}
		}
	}

	span := monitoring_domain.InternalSpanData{
		StartTime:     s.StartTime(),
		EndTime:       s.EndTime(),
		Attributes:    make(map[string]string),
		TraceID:       s.SpanContext().TraceID().String(),
		SpanID:        s.SpanContext().SpanID().String(),
		ParentSpanID:  parentSpanID,
		Name:          s.Name(),
		Kind:          s.SpanKind().String(),
		Status:        statusToString(s.Status().Code),
		StatusMessage: s.Status().Description,
		ServiceName:   serviceName,
		Events:        make([]monitoring_domain.InternalSpanEvent, 0, len(s.Events())),
		Duration:      s.EndTime().Sub(s.StartTime()),
	}

	for _, attr := range s.Attributes() {
		span.Attributes[string(attr.Key)] = attr.Value.Emit()
	}

	for _, event := range s.Events() {
		eventAttrs := make(map[string]string)
		for _, attr := range event.Attributes {
			eventAttrs[string(attr.Key)] = attr.Value.Emit()
		}
		e := monitoring_domain.InternalSpanEvent{
			Timestamp:  event.Time,
			Attributes: eventAttrs,
			Name:       event.Name,
		}
		span.Events = append(span.Events, e)
	}

	p.store.RecordSpan(span)
}

// Shutdown shuts down the span processor.
//
// Returns error when shutdown fails.
func (*SpanProcessor) Shutdown(_ context.Context) error {
	return nil
}

// ForceFlush forces a flush of any buffered spans.
//
// Returns error when the flush fails.
func (*SpanProcessor) ForceFlush(_ context.Context) error {
	return nil
}

// statusToString converts an OTEL status code to a string.
//
// Takes code (codes.Code) which is the OTEL status code to convert.
//
// Returns string which is the human-readable status name.
func statusToString(code codes.Code) string {
	switch code {
	case codes.Unset:
		return "UNSET"
	case codes.Ok:
		return "OK"
	case codes.Error:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
