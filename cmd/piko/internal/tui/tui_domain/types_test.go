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
	"testing"
	"time"
)

func TestMetricSeries_Latest(t *testing.T) {
	testCases := []struct {
		name          string
		values        []MetricValue
		expectNil     bool
		expectedValue float64
	}{
		{
			name:      "empty series returns nil",
			values:    nil,
			expectNil: true,
		},
		{
			name:      "empty slice returns nil",
			values:    []MetricValue{},
			expectNil: true,
		},
		{
			name: "single value returns that value",
			values: []MetricValue{
				{Value: 42.0, Timestamp: time.Now()},
			},
			expectNil:     false,
			expectedValue: 42.0,
		},
		{
			name: "multiple values returns last",
			values: []MetricValue{
				{Value: 1.0, Timestamp: time.Now().Add(-2 * time.Second)},
				{Value: 2.0, Timestamp: time.Now().Add(-1 * time.Second)},
				{Value: 3.0, Timestamp: time.Now()},
			},
			expectNil:     false,
			expectedValue: 3.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			series := &MetricSeries{
				Name:   "test_metric",
				Values: tc.values,
			}

			result := series.Latest()

			if tc.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				if result.Value != tc.expectedValue {
					t.Errorf("expected value %f, got %f", tc.expectedValue, result.Value)
				}
			}
		})
	}
}

func TestSpanStatus_String(t *testing.T) {
	testCases := []struct {
		expected string
		status   SpanStatus
	}{
		{status: SpanStatusUnset, expected: "unset"},
		{status: SpanStatusOK, expected: "ok"},
		{status: SpanStatusError, expected: "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.status.String()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestSpan_IsRoot(t *testing.T) {
	testCases := []struct {
		name     string
		parentID string
		expected bool
	}{
		{name: "empty parent is root", parentID: "", expected: true},
		{name: "with parent is not root", parentID: "parent-123", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			span := &Span{ParentID: tc.parentID}
			result := span.IsRoot()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSpan_IsError(t *testing.T) {
	testCases := []struct {
		name     string
		status   SpanStatus
		expected bool
	}{
		{name: "unset is not error", status: SpanStatusUnset, expected: false},
		{name: "ok is not error", status: SpanStatusOK, expected: false},
		{name: "error is error", status: SpanStatusError, expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			span := &Span{Status: tc.status}
			result := span.IsError()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestResourceStatus_String(t *testing.T) {
	testCases := []struct {
		expected string
		status   ResourceStatus
	}{
		{status: ResourceStatusUnknown, expected: "unknown"},
		{status: ResourceStatusHealthy, expected: "healthy"},
		{status: ResourceStatusDegraded, expected: "degraded"},
		{status: ResourceStatusUnhealthy, expected: "unhealthy"},
		{status: ResourceStatusPending, expected: "pending"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.status.String()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestResource_HasChildren(t *testing.T) {
	testCases := []struct {
		name     string
		children []Resource
		expected bool
	}{
		{name: "nil children", children: nil, expected: false},
		{name: "empty children", children: []Resource{}, expected: false},
		{name: "with children", children: []Resource{{ID: "child-1"}}, expected: true},
		{name: "multiple children", children: []Resource{{ID: "child-1"}, {ID: "child-2"}}, expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resource := &Resource{Children: tc.children}
			result := resource.HasChildren()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestProviderStatus_String(t *testing.T) {
	testCases := []struct {
		expected string
		status   ProviderStatus
	}{
		{status: ProviderStatusDisconnected, expected: "disconnected"},
		{status: ProviderStatusConnecting, expected: "connecting"},
		{status: ProviderStatusConnected, expected: "connected"},
		{status: ProviderStatusError, expected: "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.status.String()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestHealthState_String(t *testing.T) {
	testCases := []struct {
		expected string
		state    HealthState
	}{
		{state: HealthStateUnknown, expected: "unknown"},
		{state: HealthStateHealthy, expected: "healthy"},
		{state: HealthStateDegraded, expected: "degraded"},
		{state: HealthStateUnhealthy, expected: "unhealthy"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.state.String()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestHealthStatus_IsHealthy(t *testing.T) {
	testCases := []struct {
		name     string
		state    HealthState
		expected bool
	}{
		{name: "unknown is not healthy", state: HealthStateUnknown, expected: false},
		{name: "healthy is healthy", state: HealthStateHealthy, expected: true},
		{name: "degraded is not healthy", state: HealthStateDegraded, expected: false},
		{name: "unhealthy is not healthy", state: HealthStateUnhealthy, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := &HealthStatus{State: tc.state}
			result := status.IsHealthy()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestHealthStatus_CountByState(t *testing.T) {
	testCases := []struct {
		expected     map[HealthState]int
		name         string
		dependencies []*HealthStatus
	}{
		{
			name:         "empty dependencies",
			dependencies: nil,
			expected:     map[HealthState]int{},
		},
		{
			name: "single healthy dependency",
			dependencies: []*HealthStatus{
				{State: HealthStateHealthy},
			},
			expected: map[HealthState]int{
				HealthStateHealthy: 1,
			},
		},
		{
			name: "mixed dependencies",
			dependencies: []*HealthStatus{
				{State: HealthStateHealthy},
				{State: HealthStateHealthy},
				{State: HealthStateDegraded},
				{State: HealthStateUnhealthy},
			},
			expected: map[HealthState]int{
				HealthStateHealthy:   2,
				HealthStateDegraded:  1,
				HealthStateUnhealthy: 1,
			},
		},
		{
			name: "all same state",
			dependencies: []*HealthStatus{
				{State: HealthStateUnhealthy},
				{State: HealthStateUnhealthy},
				{State: HealthStateUnhealthy},
			},
			expected: map[HealthState]int{
				HealthStateUnhealthy: 3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := &HealthStatus{Dependencies: tc.dependencies}
			result := status.CountByState()

			for state, count := range tc.expected {
				if result[state] != count {
					t.Errorf("for state %v: expected count %d, got %d", state, count, result[state])
				}
			}

			for state, count := range result {
				if tc.expected[state] != count {
					t.Errorf("unexpected state %v with count %d", state, count)
				}
			}
		})
	}
}
