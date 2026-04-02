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

package provider_grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

var _ tui_domain.TracesProvider = (*TracesProvider)(nil)

// defaultTracesLimit is the default number of traces to fetch per query.
const defaultTracesLimit = 100

// TracesProvider implements tui_domain.TracesProvider using gRPC.
type TracesProvider struct {
	// conn holds the gRPC connection with health and metrics clients.
	conn *Connection

	// spans holds the cached span data populated by Refresh.
	spans []tui_domain.Span

	// errors holds spans that contain error status codes.
	errors []tui_domain.Span

	// mu guards concurrent access to spans and errors fields.
	mu sync.RWMutex

	// interval is the refresh duration between trace updates.
	interval time.Duration
}

// NewTracesProvider creates a new TracesProvider.
//
// Takes conn (*Connection) which is the shared gRPC connection.
// Takes interval (time.Duration) which is the refresh interval.
//
// Returns *TracesProvider which is the configured provider.
func NewTracesProvider(conn *Connection, interval time.Duration) *TracesProvider {
	return &TracesProvider{
		conn:     conn,
		spans:    nil,
		errors:   nil,
		mu:       sync.RWMutex{},
		interval: interval,
	}
}

// Name returns the provider name.
//
// Returns string which is the identifier "grpc-traces".
func (*TracesProvider) Name() string {
	return "grpc-traces"
}

// Health checks if the gRPC connection is healthy.
//
// Returns error when the health check fails or the connection is unavailable.
func (p *TracesProvider) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking traces provider health via gRPC: %w", err)
	}
	return nil
}

// Close releases resources.
//
// Returns error when resources cannot be released.
func (*TracesProvider) Close() error {
	return nil
}

// RefreshInterval returns the refresh interval.
//
// Returns time.Duration which is the interval between trace refreshes.
func (p *TracesProvider) RefreshInterval() time.Duration {
	return p.interval
}

// Refresh fetches the latest traces via gRPC.
//
// Returns error when the gRPC call fails or the connection is unavailable.
//
// Safe for concurrent use. Uses a mutex to protect the internal spans and
// errors state during updates.
func (p *TracesProvider) Refresh(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	return instrumentedCall(ctx, func() error {
		response, err := p.conn.metricsClient.GetTraces(ctx, &pb.GetTracesRequest{
			Limit:      defaultTracesLimit,
			ErrorsOnly: false,
			TraceId:    "",
		})
		if err != nil {
			l.Debug("Failed to fetch traces", logger.Error(err))
			return fmt.Errorf("fetching traces: %w", err)
		}

		spans, errors := convertSpans(response.GetSpans())

		p.mu.Lock()
		p.spans = spans
		p.errors = errors
		p.mu.Unlock()

		return nil
	})
}

// Recent fetches the N most recent traces.
//
// Takes limit (int) which specifies the maximum number of spans to return.
//
// Returns []tui_domain.Span which contains the most recent spans up to limit.
// Returns error when retrieval fails.
//
// Safe for concurrent use. Uses a read lock to protect access to the spans.
func (p *TracesProvider) Recent(_ context.Context, limit int) ([]tui_domain.Span, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit > len(p.spans) {
		limit = len(p.spans)
	}

	result := make([]tui_domain.Span, limit)
	copy(result, p.spans[:limit])
	return result, nil
}

// Errors fetches recent error spans.
//
// Takes limit (int) which specifies the maximum number of spans to return.
//
// Returns []tui_domain.Span which contains the most recent error spans.
// Returns error which is always nil.
//
// Safe for concurrent use. Uses a read lock to protect access to the error
// spans.
func (p *TracesProvider) Errors(_ context.Context, limit int) ([]tui_domain.Span, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit > len(p.errors) {
		limit = len(p.errors)
	}

	result := make([]tui_domain.Span, limit)
	copy(result, p.errors[:limit])
	return result, nil
}

// Get fetches all spans for a specific trace.
//
// Takes traceID (string) which identifies the trace to retrieve.
//
// Returns []tui_domain.Span which contains all spans belonging to the trace.
// Returns error when no spans are found for the given trace ID.
//
// Safe for concurrent use; protected by a read lock.
func (p *TracesProvider) Get(_ context.Context, traceID string) ([]tui_domain.Span, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []tui_domain.Span
	for i := range p.spans {
		if p.spans[i].TraceID == traceID {
			result = append(result, p.spans[i])
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("trace %q not found", traceID)
	}

	return result, nil
}

// convertSpans converts protobuf spans to domain format.
//
// Takes pbSpans ([]*pb.Span) which contains the protobuf span data to convert.
//
// Returns spans ([]tui_domain.Span) which contains all converted spans.
// Returns errors ([]tui_domain.Span) which contains only spans with error status.
func convertSpans(pbSpans []*pb.Span) (spans []tui_domain.Span, errors []tui_domain.Span) {
	spans = make([]tui_domain.Span, 0, len(pbSpans))
	errors = make([]tui_domain.Span, 0)

	for _, s := range pbSpans {
		span := tui_domain.Span{
			TraceID:       s.GetTraceId(),
			SpanID:        s.GetSpanId(),
			ParentID:      s.GetParentSpanId(),
			Name:          s.GetName(),
			Service:       s.GetServiceName(),
			Status:        convertSpanStatus(s.GetStatus()),
			StatusMessage: s.GetStatusMessage(),
			StartTime:     time.UnixMilli(s.GetStartTimeMs()),
			Duration:      time.Duration(s.GetDurationNs()),
			Attributes:    s.GetAttributes(),
			Children:      nil,
		}

		spans = append(spans, span)

		if span.Status == tui_domain.SpanStatusError {
			errors = append(errors, span)
		}
	}

	return spans, errors
}

// convertSpanStatus converts a protobuf span status to domain format.
//
// Takes status (string) which is the protobuf status value to convert.
//
// Returns tui_domain.SpanStatus which is the corresponding domain status.
func convertSpanStatus(status string) tui_domain.SpanStatus {
	switch status {
	case "OK":
		return tui_domain.SpanStatusOK
	case "ERROR":
		return tui_domain.SpanStatusError
	default:
		return tui_domain.SpanStatusUnset
	}
}
