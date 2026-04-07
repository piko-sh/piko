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

package collection_dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createTestItem(id string, metadata map[string]any) *ContentItem {
	return &ContentItem{
		ID:       id,
		Metadata: metadata,
	}
}

func TestApplyFilter_Equals(t *testing.T) {
	item := createTestItem("1", map[string]any{
		"title":    "Hello World",
		"featured": true,
		"views":    100,
	})

	tests := []struct {
		filter   Filter
		name     string
		expected bool
	}{
		{
			name:     "string equals - match",
			filter:   Filter{Field: "title", Operator: FilterOpEquals, Value: "Hello World"},
			expected: true,
		},
		{
			name:     "string equals - no match",
			filter:   Filter{Field: "title", Operator: FilterOpEquals, Value: "Goodbye"},
			expected: false,
		},
		{
			name:     "bool equals - match",
			filter:   Filter{Field: "featured", Operator: FilterOpEquals, Value: true},
			expected: true,
		},
		{
			name:     "bool equals - no match",
			filter:   Filter{Field: "featured", Operator: FilterOpEquals, Value: false},
			expected: false,
		},
		{
			name:     "number equals - match",
			filter:   Filter{Field: "views", Operator: FilterOpEquals, Value: 100},
			expected: true,
		},
		{
			name:     "field not exists",
			filter:   Filter{Field: "nonexistent", Operator: FilterOpEquals, Value: "test"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyFilter(item, tt.filter)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestApplyFilter_Comparison(t *testing.T) {
	item := createTestItem("1", map[string]any{
		"views":       100,
		"publishedAt": "2025-01-15",
	})

	tests := []struct {
		filter   Filter
		name     string
		expected bool
	}{
		{
			name:     "greater than - true",
			filter:   Filter{Field: "views", Operator: FilterOpGreaterThan, Value: 50},
			expected: true,
		},
		{
			name:     "greater than - false",
			filter:   Filter{Field: "views", Operator: FilterOpGreaterThan, Value: 150},
			expected: false,
		},
		{
			name:     "greater or equal - true (greater)",
			filter:   Filter{Field: "views", Operator: FilterOpGreaterEqual, Value: 50},
			expected: true,
		},
		{
			name:     "greater or equal - true (equal)",
			filter:   Filter{Field: "views", Operator: FilterOpGreaterEqual, Value: 100},
			expected: true,
		},
		{
			name:     "less than - true",
			filter:   Filter{Field: "views", Operator: FilterOpLessThan, Value: 150},
			expected: true,
		},
		{
			name:     "less or equal - true",
			filter:   Filter{Field: "views", Operator: FilterOpLessEqual, Value: 100},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyFilter(item, tt.filter)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestApplyFilter_StringOperations(t *testing.T) {
	item := createTestItem("1", map[string]any{
		"title": "Hello World Example",
		"slug":  "hello-world-example",
	})

	tests := []struct {
		filter   Filter
		name     string
		expected bool
	}{
		{
			name:     "contains - match",
			filter:   Filter{Field: "title", Operator: FilterOpContains, Value: "World"},
			expected: true,
		},
		{
			name:     "contains - no match",
			filter:   Filter{Field: "title", Operator: FilterOpContains, Value: "Goodbye"},
			expected: false,
		},
		{
			name:     "starts with - match",
			filter:   Filter{Field: "title", Operator: FilterOpStartsWith, Value: "Hello"},
			expected: true,
		},
		{
			name:     "starts with - no match",
			filter:   Filter{Field: "title", Operator: FilterOpStartsWith, Value: "World"},
			expected: false,
		},
		{
			name:     "ends with - match",
			filter:   Filter{Field: "title", Operator: FilterOpEndsWith, Value: "Example"},
			expected: true,
		},
		{
			name:     "ends with - no match",
			filter:   Filter{Field: "title", Operator: FilterOpEndsWith, Value: "Hello"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyFilter(item, tt.filter)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestApplyFilter_In(t *testing.T) {
	item := createTestItem("1", map[string]any{
		"category": "programming",
		"status":   "published",
	})

	tests := []struct {
		filter   Filter
		name     string
		expected bool
	}{
		{
			name:     "in - match",
			filter:   Filter{Field: "category", Operator: FilterOpIn, Value: []string{"programming", "tutorial", "guide"}},
			expected: true,
		},
		{
			name:     "in - no match",
			filter:   Filter{Field: "category", Operator: FilterOpIn, Value: []string{"news", "update"}},
			expected: false,
		},
		{
			name:     "not in - match",
			filter:   Filter{Field: "status", Operator: FilterOpNotIn, Value: []string{"draft", "archived"}},
			expected: true,
		},
		{
			name:     "not in - no match",
			filter:   Filter{Field: "status", Operator: FilterOpNotIn, Value: []string{"published", "archived"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyFilter(item, tt.filter)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestApplyFilter_Exists(t *testing.T) {
	item := createTestItem("1", map[string]any{
		"title": "Hello World",
	})

	tests := []struct {
		filter   Filter
		name     string
		expected bool
	}{
		{
			name:     "exists - true",
			filter:   Filter{Field: "title", Operator: FilterOpExists, Value: true},
			expected: true,
		},
		{
			name:     "exists - false",
			filter:   Filter{Field: "nonexistent", Operator: FilterOpExists, Value: true},
			expected: false,
		},
		{
			name:     "not exists - true",
			filter:   Filter{Field: "nonexistent", Operator: FilterOpExists, Value: false},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyFilter(item, tt.filter)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestApplyFilterGroup_AND(t *testing.T) {
	item := createTestItem("1", map[string]any{
		"featured": true,
		"views":    100,
		"category": "programming",
	})

	group := &FilterGroup{
		Filters: []Filter{
			{Field: "featured", Operator: FilterOpEquals, Value: true},
			{Field: "views", Operator: FilterOpGreaterThan, Value: 50},
			{Field: "category", Operator: FilterOpEquals, Value: "programming"},
		},
		Logic: "AND",
	}

	result := ApplyFilterGroup(item, group)
	if !result {
		t.Error("Expected item to match AND filter group")
	}

	groupFail := &FilterGroup{
		Filters: []Filter{
			{Field: "featured", Operator: FilterOpEquals, Value: true},
			{Field: "views", Operator: FilterOpGreaterThan, Value: 150},
		},
		Logic: "AND",
	}

	result = ApplyFilterGroup(item, groupFail)
	if result {
		t.Error("Expected item NOT to match AND filter group with failing condition")
	}
}

func TestApplyFilterGroup_OR(t *testing.T) {
	item := createTestItem("1", map[string]any{
		"featured": false,
		"views":    100,
	})

	group := &FilterGroup{
		Filters: []Filter{
			{Field: "featured", Operator: FilterOpEquals, Value: true},
			{Field: "views", Operator: FilterOpGreaterThan, Value: 50},
		},
		Logic: "OR",
	}

	result := ApplyFilterGroup(item, group)
	if !result {
		t.Error("Expected item to match OR filter group (one condition true)")
	}

	groupFail := &FilterGroup{
		Filters: []Filter{
			{Field: "featured", Operator: FilterOpEquals, Value: true},
			{Field: "views", Operator: FilterOpLessThan, Value: 50},
		},
		Logic: "OR",
	}

	result = ApplyFilterGroup(item, groupFail)
	if result {
		t.Error("Expected item NOT to match OR filter group with all failing conditions")
	}
}

func TestApplyFilterGroup_Empty(t *testing.T) {
	item := createTestItem("1", map[string]any{
		"title": "Test",
	})

	result := ApplyFilterGroup(item, nil)
	if !result {
		t.Error("Expected nil filter group to match")
	}

	emptyGroup := &FilterGroup{
		Filters: []Filter{},
		Logic:   "AND",
	}

	result = ApplyFilterGroup(item, emptyGroup)
	if !result {
		t.Error("Expected empty filter group to match")
	}
}

func TestCompareFuzzy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fieldVal  any
		searchVal any
		name      string
		expected  bool
	}{
		{name: "identical strings", fieldVal: "hello", searchVal: "hello", expected: true},
		{name: "high similarity match", fieldVal: "introduction", searchVal: "introductoin", expected: true},
		{name: "completely different strings", fieldVal: "hello", searchVal: "zzzzz", expected: false},
		{name: "non-string field value", fieldVal: 42, searchVal: "hello", expected: false},
		{name: "non-string search value", fieldVal: "hello", searchVal: 42, expected: false},
		{name: "both non-string", fieldVal: true, searchVal: 42, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compareFuzzy(tt.fieldVal, tt.searchVal)
			if got != tt.expected {
				t.Errorf("compareFuzzy(%v, %v) = %v, want %v", tt.fieldVal, tt.searchVal, got, tt.expected)
			}
		})
	}
}

func TestApplyFilter_FuzzyMatch(t *testing.T) {
	t.Parallel()

	item := createTestItem("1", map[string]any{
		"title": "introduction",
		"views": 100,
	})

	tests := []struct {
		filter   Filter
		name     string
		expected bool
	}{
		{
			name:     "fuzzy match - close match",
			filter:   Filter{Field: "title", Operator: FilterOpFuzzyMatch, Value: "introductoin"},
			expected: true,
		},
		{
			name:     "fuzzy match - no match",
			filter:   Filter{Field: "title", Operator: FilterOpFuzzyMatch, Value: "zzzzz"},
			expected: false,
		},
		{
			name:     "fuzzy match - non-string field",
			filter:   Filter{Field: "views", Operator: FilterOpFuzzyMatch, Value: "100"},
			expected: false,
		},
		{
			name:     "fuzzy match - missing field",
			filter:   Filter{Field: "nonexistent", Operator: FilterOpFuzzyMatch, Value: "hello"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := applyFilter(item, tt.filter)
			if got != tt.expected {
				t.Errorf("applyFilter fuzzy = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompareGreaterThan(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-time.Hour)

	tests := []struct {
		a        any
		b        any
		name     string
		expected bool
	}{
		{name: "int greater than int", a: 100, b: 50, expected: true},
		{name: "int not greater when equal", a: 100, b: 100, expected: false},
		{name: "int not greater when less", a: 50, b: 100, expected: false},
		{name: "float64 greater", a: 3.14, b: 2.71, expected: true},
		{name: "float64 not greater", a: 2.71, b: 3.14, expected: false},
		{name: "float32 greater", a: float32(3.14), b: float32(2.71), expected: true},
		{name: "int64 greater", a: int64(100), b: int64(50), expected: true},
		{name: "int32 greater", a: int32(100), b: int32(50), expected: true},
		{name: "numeric strings parsed as numbers", a: "100", b: "50", expected: true},
		{name: "alphabetical string greater", a: "banana", b: "apple", expected: true},
		{name: "alphabetical string not greater", a: "apple", b: "banana", expected: false},
		{name: "time.Time a after b", a: now, b: earlier, expected: true},
		{name: "time.Time a before b", a: earlier, b: now, expected: false},
		{name: "incompatible types return false", a: true, b: "hello", expected: false},
		{name: "nil values return false", a: nil, b: nil, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compareGreaterThan(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("compareGreaterThan(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   any
		name    string
		wantVal float64
		wantNil bool
	}{
		{name: "float64 value", input: float64(3.14), wantVal: 3.14},
		{name: "float32 value", input: float32(2.5), wantVal: 2.5},
		{name: "int value", input: 42, wantVal: 42.0},
		{name: "int32 value", input: int32(42), wantVal: 42.0},
		{name: "int64 value", input: int64(42), wantVal: 42.0},
		{name: "string numeric", input: "3.14", wantVal: 3.14},
		{name: "string integer", input: "42", wantVal: 42.0},
		{name: "string non-numeric", input: "hello", wantNil: true},
		{name: "bool value", input: true, wantNil: true},
		{name: "nil value", input: nil, wantNil: true},
		{name: "negative float", input: float64(-1.5), wantVal: -1.5},
		{name: "zero int", input: 0, wantVal: 0.0},
		{name: "empty string", input: "", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toFloat64(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("toFloat64(%v) = %v, want nil", tt.input, *got)
				}
				return
			}
			require.NotNil(t, got, "toFloat64(%v) = nil, want %v", tt.input, tt.wantVal)

			diff := *got - tt.wantVal
			if diff < -0.01 || diff > 0.01 {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.input, *got, tt.wantVal)
			}
		})
	}
}

func TestToTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   any
		name    string
		wantNil bool
	}{
		{name: "time.Time value", input: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)},
		{name: "RFC3339 string", input: "2026-01-15T10:30:00Z"},
		{name: "date-only string", input: "2026-01-15"},
		{name: "datetime string", input: "2026-01-15 10:30:00"},
		{name: "invalid string format", input: "not a date", wantNil: true},
		{name: "empty string", input: "", wantNil: true},
		{name: "integer value", input: 42, wantNil: true},
		{name: "nil value", input: nil, wantNil: true},
		{name: "bool value", input: true, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toTime(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("toTime(%v) = %v, want nil", tt.input, got)
				}
			} else {
				if got == nil {
					t.Errorf("toTime(%v) = nil, want non-nil", tt.input)
				}
			}
		})
	}
}

func TestCompareContains_NonStringValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fieldVal  any
		searchVal any
		name      string
		expected  bool
	}{
		{name: "non-string field value", fieldVal: 42, searchVal: "4", expected: false},
		{name: "non-string search value", fieldVal: "hello", searchVal: 42, expected: false},
		{name: "both non-string", fieldVal: 42, searchVal: true, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compareContains(tt.fieldVal, tt.searchVal)
			if got != tt.expected {
				t.Errorf("compareContains(%v, %v) = %v, want %v", tt.fieldVal, tt.searchVal, got, tt.expected)
			}
		})
	}
}

func TestCompareStartsWith_NonStringValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fieldVal  any
		searchVal any
		name      string
		expected  bool
	}{
		{name: "non-string field value", fieldVal: 42, searchVal: "4", expected: false},
		{name: "non-string search value", fieldVal: "hello", searchVal: 42, expected: false},
		{name: "both non-string", fieldVal: 42, searchVal: true, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compareStartsWith(tt.fieldVal, tt.searchVal)
			if got != tt.expected {
				t.Errorf("compareStartsWith(%v, %v) = %v, want %v", tt.fieldVal, tt.searchVal, got, tt.expected)
			}
		})
	}
}

func TestCompareEndsWith_NonStringValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fieldVal  any
		searchVal any
		name      string
		expected  bool
	}{
		{name: "non-string field value", fieldVal: 42, searchVal: "2", expected: false},
		{name: "non-string search value", fieldVal: "hello", searchVal: 42, expected: false},
		{name: "both non-string", fieldVal: 42, searchVal: true, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compareEndsWith(tt.fieldVal, tt.searchVal)
			if got != tt.expected {
				t.Errorf("compareEndsWith(%v, %v) = %v, want %v", tt.fieldVal, tt.searchVal, got, tt.expected)
			}
		})
	}
}

func TestApplyFilter_UnknownOperator(t *testing.T) {
	t.Parallel()

	item := createTestItem("1", map[string]any{"title": "Hello"})
	got := applyFilter(item, Filter{Field: "title", Operator: "unknown", Value: "Hello"})
	if got {
		t.Error("expected unknown operator to return false")
	}
}

func TestApplyFilter_ExistsNonBoolValue(t *testing.T) {
	t.Parallel()

	item := createTestItem("1", map[string]any{"title": "Hello"})
	got := applyFilter(item, Filter{Field: "title", Operator: FilterOpExists, Value: "not-a-bool"})
	if got {
		t.Error("expected FilterOpExists with non-bool value to return false")
	}
}
