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

	"charm.land/lipgloss/v2"
)

func TestNewRoutesPanel(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "routes" {
		t.Errorf("expected ID 'routes', got %q", panel.ID())
	}
	if panel.Title() != "Routes" {
		t.Errorf("expected Title 'Routes', got %q", panel.Title())
	}
}

func TestNewRoutesPanel_NilClock(t *testing.T) {
	panel := NewRoutesPanel(nil, nil)

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.clock == nil {
		t.Error("expected non-nil clock even when nil passed")
	}
}

func TestRoutesPanel_HandleRefreshMessage(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())

	routes := []RouteStats{
		{Path: "/api/users", Count: 100, AverageMs: 50},
		{Path: "/api/orders", Count: 50, AverageMs: 100},
	}

	panel.handleRefreshMessage(RoutesRefreshMessage{
		Routes:      routes,
		TotalCount:  150,
		TotalErrors: 5,
	})

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	panel.stateMutex.RLock()
	totalCount := panel.totalCount
	totalErrors := panel.totalErrors
	panel.stateMutex.RUnlock()

	if totalCount != 150 {
		t.Errorf("expected totalCount 150, got %d", totalCount)
	}
	if totalErrors != 5 {
		t.Errorf("expected totalErrors 5, got %d", totalErrors)
	}

	panel.handleRefreshMessage(RoutesRefreshMessage{Err: ErrConnectionFailed})
	if !errors.Is(panel.err, ErrConnectionFailed) {
		t.Error("expected error to be set")
	}
}

func TestRoutesPanel_CycleSortBy(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())

	if panel.sortBy != "count" {
		t.Errorf("expected initial sortBy 'count', got %q", panel.sortBy)
	}

	panel.cycleSortBy()
	if panel.sortBy != "avg" {
		t.Errorf("expected sortBy 'avg', got %q", panel.sortBy)
	}

	panel.cycleSortBy()
	if panel.sortBy != "errors" {
		t.Errorf("expected sortBy 'errors', got %q", panel.sortBy)
	}

	panel.cycleSortBy()
	if panel.sortBy != "path" {
		t.Errorf("expected sortBy 'path', got %q", panel.sortBy)
	}

	panel.cycleSortBy()
	if panel.sortBy != "count" {
		t.Errorf("expected sortBy 'count', got %q", panel.sortBy)
	}
}

func TestRoutesPanel_View(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	view := panel.View(100, 24)
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "No routes") {
		t.Error("expected empty state message")
	}
}

func TestRoutesPanel_ViewWithData(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	panel.handleRefreshMessage(RoutesRefreshMessage{
		Routes: []RouteStats{
			{Path: "/api/users", Count: 100, AverageMs: 50, P50Ms: 45},
			{Path: "/api/orders", Count: 50, AverageMs: 100, P50Ms: 90},
		},
		TotalCount:  150,
		TotalErrors: 0,
	})

	view := panel.View(100, 24)
	if !strings.Contains(view, "Total:") {
		t.Error("expected 'Total:' in view")
	}
}

func TestAverage(t *testing.T) {
	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{name: "empty", values: nil, expected: 0},
		{name: "single", values: []float64{10}, expected: 10},
		{name: "multiple", values: []float64{10, 20, 30}, expected: 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := average(tc.values)
			if result != tc.expected {
				t.Errorf("expected %f, got %f", tc.expected, result)
			}
		})
	}
}

func TestPercentile(t *testing.T) {
	testCases := []struct {
		name     string
		sorted   []float64
		p        int
		expected float64
	}{
		{name: "empty", sorted: nil, p: 50, expected: 0},
		{name: "p0", sorted: []float64{10, 20, 30}, p: 0, expected: 10},
		{name: "p100", sorted: []float64{10, 20, 30}, p: 100, expected: 30},
		{name: "p50", sorted: []float64{10, 20, 30, 40}, p: 50, expected: 25},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := percentile(tc.sorted, tc.p)
			if result != tc.expected {
				t.Errorf("expected %f, got %f", tc.expected, result)
			}
		})
	}
}

func TestAggregateSpansToRoutes(t *testing.T) {
	spans := []Span{
		{Name: "op1", Attributes: map[string]string{"path": "/api/users", "method": "GET"}, Status: SpanStatusOK},
		{Name: "op2", Attributes: map[string]string{"path": "/api/users", "method": "GET"}, Status: SpanStatusError},
		{Name: "op3", Attributes: map[string]string{"path": "/api/orders", "method": "POST"}, Status: SpanStatusOK},
	}

	routeMap, totalCount, totalErrors := aggregateSpansToRoutes(spans)

	if totalCount != 3 {
		t.Errorf("expected totalCount 3, got %d", totalCount)
	}
	if totalErrors != 1 {
		t.Errorf("expected totalErrors 1, got %d", totalErrors)
	}
	if len(routeMap) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routeMap))
	}

	usersRoute := routeMap["/api/users"]
	if usersRoute.Count != 2 {
		t.Errorf("expected count 2 for /api/users, got %d", usersRoute.Count)
	}
	if usersRoute.ErrorCount != 1 {
		t.Errorf("expected 1 error for /api/users, got %d", usersRoute.ErrorCount)
	}
}

func TestRoutesRenderer_GetID(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())
	renderer := &routesRenderer{panel: panel}

	route := RouteStats{Path: "/api/users"}
	id := renderer.GetID(route)

	if id != "/api/users" {
		t.Errorf("expected '/api/users', got %q", id)
	}
}

func TestRoutesRenderer_MatchesFilter(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())
	renderer := &routesRenderer{panel: panel}

	route := RouteStats{Path: "/api/users"}

	testCases := []struct {
		query    string
		expected bool
	}{
		{query: "api", expected: true},
		{query: "users", expected: true},
		{query: "xyz", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			result := renderer.MatchesFilter(route, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestRoutesRenderer_IsExpandable(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())
	renderer := &routesRenderer{panel: panel}

	route := RouteStats{Path: "/api/users"}
	if !renderer.IsExpandable(route) {
		t.Error("all routes should be expandable")
	}
}

func TestRoutesRenderer_ExpandedLineCount(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())
	renderer := &routesRenderer{panel: panel}

	testCases := []struct {
		name     string
		route    RouteStats
		expected int
	}{
		{
			name:     "no errors no spans",
			route:    RouteStats{Count: 10, ErrorCount: 0, RecentSpans: nil},
			expected: 2,
		},
		{
			name:     "with errors",
			route:    RouteStats{Count: 10, ErrorCount: 5, RecentSpans: nil},
			expected: 3,
		},
		{
			name:     "with spans",
			route:    RouteStats{Count: 10, RecentSpans: []Span{{}, {}}},
			expected: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.ExpandedLineCount(tc.route)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestRoutesPanel_RenderSpanLine(t *testing.T) {
	panel := NewRoutesPanel(nil, newTestClock())
	panel.SetSize(100, 24)

	span := Span{
		StartTime:  time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Duration:   50 * time.Millisecond,
		Status:     SpanStatusOK,
		Attributes: map[string]string{"method": "GET"},
	}

	result := panel.renderSpanLine(span, "  ", new(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))))
	if !strings.Contains(result, "OK") {
		t.Error("expected 'OK' in span line")
	}
	if !strings.Contains(result, "GET") {
		t.Error("expected 'GET' in span line")
	}
}
