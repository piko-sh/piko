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

package logger_domain

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OTelSlogHandler is a slog.Handler that adds OpenTelemetry trace context to
// log records and sends log events to active spans.
type OTelSlogHandler struct {
	slog.Handler
}

// NewOTelSlogHandler creates a handler that adds trace data to log records.
//
// Takes next (slog.Handler) which is the handler to wrap.
//
// Returns *OTelSlogHandler which wraps next and adds OpenTelemetry trace and
// span IDs to each log record.
func NewOTelSlogHandler(next slog.Handler) *OTelSlogHandler {
	return &OTelSlogHandler{Handler: next}
}

// Handle processes a log record by adding OpenTelemetry trace context,
// recording the log as a span event, and passing it to the wrapped handler.
// Implements the slog.Handler interface.
//
// Takes r (slog.Record) which contains the log entry to process.
//
// Returns error when the wrapped handler fails to process the record.
//
//nolint:gocritic // slog.Handler requires value receiver
func (h *OTelSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)

	if !span.IsRecording() {
		return h.Handler.Handle(ctx, r)
	}

	newRecord := r.Clone()
	newRecord.AddAttrs(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	finalAttrs := make([]slog.Attr, 0, newRecord.NumAttrs())
	newRecord.Attrs(func(a slog.Attr) bool {
		finalAttrs = append(finalAttrs, a)
		return true
	})

	if len(finalAttrs) > 0 {
		kvsPtr, ok := otelAttrPool.Get().(*[]attribute.KeyValue)
		if !ok {
			kvsPtr = new(make([]attribute.KeyValue, 0, len(finalAttrs)))
		}
		kvs := (*kvsPtr)[:0]
		for _, a := range finalAttrs {
			addAttrRecursive(&kvs, "", a)
		}
		span.AddEvent(r.Message, trace.WithAttributes(kvs...))
		*kvsPtr = kvs[:0]
		otelAttrPool.Put(kvsPtr)
	} else {
		span.AddEvent(r.Message)
	}

	if r.Level >= slog.LevelError {
		errMessage := r.Message
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "error" {
				switch v := a.Value.Any().(type) {
				case error:
					errMessage = v.Error()
				case string:
					errMessage = v
				}
				return false
			}
			return true
		})
		span.SetStatus(codes.Error, errMessage)
	}

	return h.Handler.Handle(ctx, newRecord)
}

// WithAttrs returns a new OTelSlogHandler with the given attributes added
// to the underlying handler. This preserves the OTEL wrapper when
// attributes are added.
//
// Takes attrs ([]slog.Attr) which specifies the attributes to add.
//
// Returns slog.Handler which is the new handler with the attributes added.
func (h *OTelSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OTelSlogHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

// WithGroup returns a new OTelSlogHandler with the given group added to the
// underlying handler. This preserves the OTEL wrapper when groups
// are added.
//
// Takes name (string) which specifies the group name to add.
//
// Returns slog.Handler which is a new handler with the group applied.
func (h *OTelSlogHandler) WithGroup(name string) slog.Handler {
	return &OTelSlogHandler{
		Handler: h.Handler.WithGroup(name),
	}
}
