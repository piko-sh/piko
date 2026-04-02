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

package tui_domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewTracesPanel(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "traces" {
		t.Errorf("expected ID 'traces', got %q", panel.ID())
	}
	if panel.Title() != "Traces" {
		t.Errorf("expected Title 'Traces', got %q", panel.Title())
	}
}

func TestNewTracesPanel_NilClock(t *testing.T) {
	panel := NewTracesPanel(nil, nil)

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.clock == nil {
		t.Error("expected non-nil clock even when nil passed")
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name     string
		spanTime time.Time
		expected string
	}{
		{
			name:     "seconds",
			spanTime: now.Add(-30 * time.Second),
			expected: "30s",
		},
		{
			name:     "minutes",
			spanTime: now.Add(-5 * time.Minute),
			expected: "5m",
		},
		{
			name:     "hours",
			spanTime: now.Add(-3 * time.Hour),
			expected: "3h",
		},
		{
			name:     "days",
			spanTime: now.Add(-48 * time.Hour),
			expected: "2d",
		},
		{
			name:     "just under a minute",
			spanTime: now.Add(-59 * time.Second),
			expected: "59s",
		},
		{
			name:     "just under an hour",
			spanTime: now.Add(-59 * time.Minute),
			expected: "59m",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatTimeAgo(tc.spanTime, now)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestSpanRenderer_GetID(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	renderer := &spanRenderer{panel: panel}

	span := Span{SpanID: "span-123"}
	id := renderer.GetID(span)

	if id != "span-123" {
		t.Errorf("expected 'span-123', got %q", id)
	}
}

func TestSpanRenderer_MatchesFilter(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	renderer := &spanRenderer{panel: panel}

	span := Span{
		Name:    "HTTP GET /users",
		Service: "api-gateway",
		TraceID: "trace-abc-123",
	}

	testCases := []struct {
		query    string
		expected bool
	}{
		{query: "http", expected: true},
		{query: "gateway", expected: true},
		{query: "abc", expected: true},
		{query: "xyz", expected: false},
		{query: "users", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			result := renderer.MatchesFilter(span, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSpanRenderer_IsExpandable(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	renderer := &spanRenderer{panel: panel}

	span := Span{Name: "test"}
	if renderer.IsExpandable(span) {
		t.Error("spans should not be expandable")
	}
}

func TestSpanRenderer_ExpandedLineCount(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	renderer := &spanRenderer{panel: panel}

	span := Span{Name: "test"}
	if renderer.ExpandedLineCount(span) != 0 {
		t.Error("expanded line count should be 0")
	}
}

func TestSpanRenderer_RenderExpanded(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	renderer := &spanRenderer{panel: panel}

	span := Span{Name: "test"}
	lines := renderer.RenderExpanded(span, 80)
	if lines != nil {
		t.Error("expected nil expanded lines")
	}
}

func TestTracesPanel_HandleRefreshMessage(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())

	spans := []Span{
		{SpanID: "span-1", Name: "op1"},
		{SpanID: "span-2", Name: "op2"},
	}
	panel.handleRefreshMessage(TracesRefreshMessage{Spans: spans})

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	panel.handleRefreshMessage(TracesRefreshMessage{Err: ErrConnectionFailed})
	if !errors.Is(panel.err, ErrConnectionFailed) {
		t.Error("expected error to be set")
	}
}

func TestTracesPanel_SpanDeduplication(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())

	panel.handleRefreshMessage(TracesRefreshMessage{
		Spans: []Span{
			{SpanID: "span-1", Name: "op1"},
			{SpanID: "span-2", Name: "op2"},
		},
	})

	panel.handleRefreshMessage(TracesRefreshMessage{
		Spans: []Span{
			{SpanID: "span-1", Name: "op1"},
			{SpanID: "span-3", Name: "op3"},
		},
	})

	items := panel.Items()
	if len(items) != 3 {
		t.Errorf("expected 3 unique spans, got %d", len(items))
	}

	if items[0].SpanID != "span-3" {
		t.Errorf("expected span-3 first, got %s", items[0].SpanID)
	}
}

func TestTracesPanel_View(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	view := panel.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "No traces") {
		t.Error("expected empty state message")
	}
}

func TestTracesPanel_ViewWithSpans(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	panel.handleRefreshMessage(TracesRefreshMessage{
		Spans: []Span{
			{SpanID: "span-1", Name: "GET /users", Service: "api", Status: SpanStatusOK, Duration: 50 * time.Millisecond},
			{SpanID: "span-2", Name: "POST /orders", Service: "api", Status: SpanStatusError, Duration: 150 * time.Millisecond},
		},
	})

	view := panel.View(80, 24)
	if !strings.Contains(view, "GET /users") {
		t.Error("expected span name in view")
	}
}

func TestTracesPanel_ErrorsOnlyToggle(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())

	panel.stateMutex.RLock()
	initialState := panel.errorsOnly
	panel.stateMutex.RUnlock()

	if initialState {
		t.Error("expected errorsOnly to be false initially")
	}

	keyMessage := createTestKeyMessage("e")
	_, _ = panel.handleKey(keyMessage)

	panel.stateMutex.RLock()
	newState := panel.errorsOnly
	panel.stateMutex.RUnlock()

	if !newState {
		t.Error("expected errorsOnly to be true after toggle")
	}
}

func TestTracesPanel_ErrorsOnlyHeader(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	panel.SetSize(80, 24)

	panel.stateMutex.Lock()
	panel.errorsOnly = true
	panel.stateMutex.Unlock()

	header := panel.renderHeader()
	if !strings.Contains(header, "Errors Only") {
		t.Error("expected 'Errors Only' in header")
	}
}

func TestTracesPanel_EmptyStateWithErrorsOnly(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	panel.SetSize(80, 24)

	panel.stateMutex.Lock()
	panel.errorsOnly = true
	panel.stateMutex.Unlock()

	var content strings.Builder
	panel.renderTracesEmptyState(&content)
	result := content.String()

	if !strings.Contains(result, "No error traces") {
		t.Errorf("expected 'No error traces', got %q", result)
	}
}

func TestTracesPanel_TableHeader(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	panel.SetSize(120, 24)

	header := panel.renderTableHeader()

	if !strings.Contains(header, "Name") {
		t.Error("expected 'Name' in table header")
	}
	if !strings.Contains(header, "Service") {
		t.Error("expected 'Service' in table header")
	}
	if !strings.Contains(header, "Duration") {
		t.Error("expected 'Duration' in table header")
	}
}

func TestTracesPanel_CalculateColumnWidths(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	panel.SetSize(120, 24)

	nameW, serviceW := panel.calculateColumnWidths()

	if nameW < tracesMinNameWidth {
		t.Errorf("expected nameW >= %d, got %d", tracesMinNameWidth, nameW)
	}
	if serviceW < tracesMinServiceW {
		t.Errorf("expected serviceW >= %d, got %d", tracesMinServiceW, serviceW)
	}
}

func TestTracesPanel_RenderSpanLine(t *testing.T) {
	panel := NewTracesPanel(nil, newTestClock())
	panel.SetSize(120, 24)
	panel.SetFocused(true)

	testCases := []struct {
		name     string
		contains string
		span     Span
	}{
		{
			name: "OK status",
			span: Span{
				Name:     "GET /health",
				Service:  "api",
				Status:   SpanStatusOK,
				Duration: 10 * time.Millisecond,
			},
			contains: "GET /health",
		},
		{
			name: "Error status",
			span: Span{
				Name:     "POST /error",
				Service:  "api",
				Status:   SpanStatusError,
				Duration: 200 * time.Millisecond,
			},
			contains: "POST /error",
		},
		{
			name: "Slow request",
			span: Span{
				Name:     "Slow query",
				Service:  "db",
				Status:   SpanStatusOK,
				Duration: 2 * time.Second,
			},
			contains: "Slow query",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			line := panel.renderSpanLine(tc.span, 0)
			if !strings.Contains(line, tc.contains) {
				t.Errorf("expected line to contain %q", tc.contains)
			}
		})
	}
}
