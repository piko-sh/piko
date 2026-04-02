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
)

func TestSortItems_SingleField(t *testing.T) {
	items := []*ContentItem{
		createTestItem("3", map[string]any{"title": "Zebra"}),
		createTestItem("1", map[string]any{"title": "Apple"}),
		createTestItem("2", map[string]any{"title": "Banana"}),
	}

	sortOptions := []SortOption{
		{Field: "title", Order: SortAsc},
	}

	SortItems(items, sortOptions)

	expectedOrder := []string{"Apple", "Banana", "Zebra"}
	for i, item := range items {
		if item.Metadata["title"] != expectedOrder[i] {
			t.Errorf("Expected %s at position %d, got %s", expectedOrder[i], i, item.Metadata["title"])
		}
	}
}

func TestSortItems_Descending(t *testing.T) {
	items := []*ContentItem{
		createTestItem("1", map[string]any{"views": 100}),
		createTestItem("2", map[string]any{"views": 500}),
		createTestItem("3", map[string]any{"views": 250}),
	}

	sortOptions := []SortOption{
		{Field: "views", Order: SortDesc},
	}

	SortItems(items, sortOptions)

	expectedOrder := []int{500, 250, 100}
	for i, item := range items {
		if item.Metadata["views"] != expectedOrder[i] {
			t.Errorf("Expected %d at position %d, got %v", expectedOrder[i], i, item.Metadata["views"])
		}
	}
}

func TestSortItems_MultipleFields(t *testing.T) {
	items := []*ContentItem{
		createTestItem("1", map[string]any{"category": "tech", "title": "Zebra"}),
		createTestItem("2", map[string]any{"category": "tech", "title": "Apple"}),
		createTestItem("3", map[string]any{"category": "news", "title": "Banana"}),
		createTestItem("4", map[string]any{"category": "news", "title": "Apple"}),
	}

	sortOptions := []SortOption{
		{Field: "category", Order: SortAsc},
		{Field: "title", Order: SortAsc},
	}

	SortItems(items, sortOptions)

	expected := []struct {
		category string
		title    string
	}{
		{category: "news", title: "Apple"},
		{category: "news", title: "Banana"},
		{category: "tech", title: "Apple"},
		{category: "tech", title: "Zebra"},
	}

	for i, item := range items {
		category, ok := item.Metadata["category"].(string)
		if !ok {
			t.Fatalf("Expected category to be string at position %d", i)
		}
		title, ok := item.Metadata["title"].(string)
		if !ok {
			t.Fatalf("Expected title to be string at position %d", i)
		}

		if category != expected[i].category || title != expected[i].title {
			t.Errorf("Expected %s/%s at position %d, got %s/%s",
				expected[i].category, expected[i].title, i, category, title)
		}
	}
}

func TestSortItems_MissingFields(t *testing.T) {
	items := []*ContentItem{
		createTestItem("1", map[string]any{"title": "Has Title"}),
		createTestItem("2", map[string]any{}),
		createTestItem("3", map[string]any{"title": "Another Title"}),
	}

	sortOptions := []SortOption{
		{Field: "title", Order: SortAsc},
	}

	SortItems(items, sortOptions)

	if _, exists := items[0].Metadata["title"]; exists {
		t.Error("Expected item without title to be first")
	}
}

func TestSortItems_EmptyOptions(t *testing.T) {
	items := []*ContentItem{
		createTestItem("1", map[string]any{"title": "Zebra"}),
		createTestItem("2", map[string]any{"title": "Apple"}),
	}

	originalOrder := []string{
		items[0].Metadata["title"].(string),
		items[1].Metadata["title"].(string),
	}

	SortItems(items, []SortOption{})

	for i, item := range items {
		if item.Metadata["title"] != originalOrder[i] {
			t.Error("Order should not change with empty sort options")
		}
	}
}

func TestPaginateItems_PageBased(t *testing.T) {
	items := make([]*ContentItem, 25)
	for i := range 25 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	pagination := &PaginationOptions{
		Page:     2,
		PageSize: 10,
	}

	result := PaginateItems(items, pagination)

	if len(result) != 10 {
		t.Errorf("Expected 10 items, got %d", len(result))
	}

	if result[0].Metadata["index"] != 10 {
		t.Errorf("Expected first item to have index 10, got %v", result[0].Metadata["index"])
	}

	if result[9].Metadata["index"] != 19 {
		t.Errorf("Expected last item to have index 19, got %v", result[9].Metadata["index"])
	}
}

func TestPaginateItems_LastPage(t *testing.T) {
	items := make([]*ContentItem, 25)
	for i := range 25 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	pagination := &PaginationOptions{
		Page:     3,
		PageSize: 10,
	}

	result := PaginateItems(items, pagination)

	if len(result) != 5 {
		t.Errorf("Expected 5 items, got %d", len(result))
	}

	if result[0].Metadata["index"] != 20 {
		t.Errorf("Expected first item to have index 20, got %v", result[0].Metadata["index"])
	}
}

func TestPaginateItems_OffsetBased(t *testing.T) {
	items := make([]*ContentItem, 25)
	for i := range 25 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	pagination := &PaginationOptions{
		Offset: 5,
		Limit:  10,
	}

	result := PaginateItems(items, pagination)

	if len(result) != 10 {
		t.Errorf("Expected 10 items, got %d", len(result))
	}

	if result[0].Metadata["index"] != 5 {
		t.Errorf("Expected first item to have index 5, got %v", result[0].Metadata["index"])
	}
}

func TestPaginateItems_BeyondRange(t *testing.T) {
	items := make([]*ContentItem, 10)
	for i := range 10 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	pagination := &PaginationOptions{
		Page:     10,
		PageSize: 10,
	}

	result := PaginateItems(items, pagination)

	if len(result) != 0 {
		t.Errorf("Expected 0 items, got %d", len(result))
	}
}

func TestPaginateItems_NilPagination(t *testing.T) {
	items := make([]*ContentItem, 10)
	for i := range 10 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	result := PaginateItems(items, nil)

	if len(result) != 10 {
		t.Errorf("Expected 10 items, got %d", len(result))
	}
}

func TestCalculatePaginationMeta(t *testing.T) {
	pagination := &PaginationOptions{
		Page:     2,
		PageSize: 10,
	}

	meta := CalculatePaginationMeta(25, pagination)

	if meta.CurrentPage != 2 {
		t.Errorf("Expected CurrentPage 2, got %d", meta.CurrentPage)
	}

	if meta.PageSize != 10 {
		t.Errorf("Expected PageSize 10, got %d", meta.PageSize)
	}

	if meta.TotalItems != 25 {
		t.Errorf("Expected TotalItems 25, got %d", meta.TotalItems)
	}

	if meta.TotalPages != 3 {
		t.Errorf("Expected TotalPages 3, got %d", meta.TotalPages)
	}

	if !meta.HasNextPage {
		t.Error("Expected HasNextPage to be true")
	}

	if !meta.HasPrevPage {
		t.Error("Expected HasPrevPage to be true")
	}
}

func TestCalculatePaginationMeta_FirstPage(t *testing.T) {
	pagination := &PaginationOptions{
		Page:     1,
		PageSize: 10,
	}

	meta := CalculatePaginationMeta(25, pagination)

	if meta.HasPrevPage {
		t.Error("Expected HasPrevPage to be false on first page")
	}

	if !meta.HasNextPage {
		t.Error("Expected HasNextPage to be true")
	}
}

func TestCalculatePaginationMeta_LastPage(t *testing.T) {
	pagination := &PaginationOptions{
		Page:     3,
		PageSize: 10,
	}

	meta := CalculatePaginationMeta(25, pagination)

	if !meta.HasPrevPage {
		t.Error("Expected HasPrevPage to be true")
	}

	if meta.HasNextPage {
		t.Error("Expected HasNextPage to be false on last page")
	}
}

func TestCompareStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal strings", a: "hello", b: "hello", want: 0},
		{name: "case insensitive equal", a: "Hello", b: "hello", want: 0},
		{name: "a less than b", a: "apple", b: "banana", want: -1},
		{name: "a greater than b", a: "zebra", b: "apple", want: 1},
		{name: "empty strings equal", a: "", b: "", want: 0},
		{name: "empty less than non-empty", a: "", b: "a", want: -1},
		{name: "non-empty greater than empty", a: "a", b: "", want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := compareStrings(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("compareStrings(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestCompareNumbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    float64
		b    float64
		want int
	}{
		{name: "equal", a: 1.0, b: 1.0, want: 0},
		{name: "a less than b", a: 1.0, b: 2.0, want: -1},
		{name: "a greater than b", a: 5.0, b: 2.0, want: 1},
		{name: "negative values", a: -3.0, b: -1.0, want: -1},
		{name: "zero equal", a: 0.0, b: 0.0, want: 0},
		{name: "zero less than positive", a: 0.0, b: 1.0, want: -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := compareNumbers(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("compareNumbers(%f, %f) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestCompareTimes(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-time.Hour)
	later := now.Add(time.Hour)

	tests := []struct {
		a    time.Time
		b    time.Time
		name string
		want int
	}{
		{name: "equal times", a: now, b: now, want: 0},
		{name: "a before b", a: earlier, b: later, want: -1},
		{name: "a after b", a: later, b: earlier, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := compareTimes(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("compareTimes() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCompareBools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    bool
		b    bool
		want int
	}{
		{name: "both true", a: true, b: true, want: 0},
		{name: "both false", a: false, b: false, want: 0},
		{name: "false before true", a: false, b: true, want: -1},
		{name: "true after false", a: true, b: false, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := compareBools(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("compareBools(%v, %v) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil returns empty", value: nil, want: ""},
		{name: "string passthrough", value: "hello", want: "hello"},
		{name: "bool true", value: true, want: "true"},
		{name: "bool false", value: false, want: "false"},
		{name: "int value", value: 65, want: "A"},
		{name: "unsupported type returns empty", value: 3.14, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := toString(tc.value)
			if got != tc.want {
				t.Errorf("toString(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-time.Hour)

	tests := []struct {
		a    any
		b    any
		name string
		want int
	}{
		{name: "both nil", a: nil, b: nil, want: 0},
		{name: "a nil b not nil", a: nil, b: "hello", want: -1},
		{name: "a not nil b nil", a: "hello", b: nil, want: 1},
		{name: "string comparison", a: "apple", b: "banana", want: -1},
		{name: "number comparison", a: 100, b: 200, want: -1},
		{name: "time comparison", a: earlier, b: now, want: -1},
		{name: "bool comparison", a: false, b: true, want: -1},
		{name: "mixed types fall back to string", a: true, b: "true", want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := compareValues(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("compareValues(%v, %v) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestCompareField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a    *ContentItem
		b    *ContentItem
		name string
		want int
	}{
		{
			name: "both missing field",
			a:    createTestItem("1", map[string]any{}),
			b:    createTestItem("2", map[string]any{}),
			want: 0,
		},
		{
			name: "a missing field sorts first",
			a:    createTestItem("1", map[string]any{}),
			b:    createTestItem("2", map[string]any{"title": "hello"}),
			want: -1,
		},
		{
			name: "b missing field sorts first",
			a:    createTestItem("1", map[string]any{"title": "hello"}),
			b:    createTestItem("2", map[string]any{}),
			want: 1,
		},
		{
			name: "equal values",
			a:    createTestItem("1", map[string]any{"title": "same"}),
			b:    createTestItem("2", map[string]any{"title": "same"}),
			want: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := compareField(tc.a, tc.b, "title")
			if got != tc.want {
				t.Errorf("compareField() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPaginateItems_LimitOnly(t *testing.T) {
	items := make([]*ContentItem, 20)
	for i := range 20 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	result := PaginateItems(items, &PaginationOptions{
		Limit: 5,
	})

	if len(result) != 5 {
		t.Errorf("Expected 5 items, got %d", len(result))
	}
}

func TestPaginateItems_EmptySlice(t *testing.T) {
	result := PaginateItems([]*ContentItem{}, &PaginationOptions{
		Page:     1,
		PageSize: 10,
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 items, got %d", len(result))
	}
}

func TestPaginateItems_InvalidPagination(t *testing.T) {
	items := make([]*ContentItem, 10)
	for i := range 10 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	result := PaginateItems(items, &PaginationOptions{})

	if len(result) != 10 {
		t.Errorf("Expected original 10 items, got %d", len(result))
	}
}

func TestCalculatePaginationMeta_NilPagination(t *testing.T) {
	meta := CalculatePaginationMeta(25, nil)

	if meta.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage 1, got %d", meta.CurrentPage)
	}
	if meta.TotalPages != 1 {
		t.Errorf("Expected TotalPages 1, got %d", meta.TotalPages)
	}
	if meta.PageSize != 25 {
		t.Errorf("Expected PageSize 25, got %d", meta.PageSize)
	}
	if meta.HasNextPage {
		t.Error("Expected HasNextPage to be false")
	}
	if meta.HasPrevPage {
		t.Error("Expected HasPrevPage to be false")
	}
}

func TestCalculatePaginationMeta_ZeroPageDefaultsToOne(t *testing.T) {
	meta := CalculatePaginationMeta(25, &PaginationOptions{
		Page:     0,
		PageSize: 10,
	})

	if meta.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage 1, got %d", meta.CurrentPage)
	}
}

func TestSortItems_RandomOrder(t *testing.T) {
	items := make([]*ContentItem, 20)
	for i := range 20 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	sortOptions := []SortOption{
		{Field: "index", Order: SortRandom},
	}

	SortItems(items, sortOptions)

	if len(items) != 20 {
		t.Errorf("Expected 20 items after random sort, got %d", len(items))
	}
}

func TestPaginateItems_Page0WithPageSize(t *testing.T) {
	t.Parallel()

	items := make([]*ContentItem, 10)
	for i := range 10 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	result := PaginateItems(items, &PaginationOptions{Page: 0, PageSize: 10})
	if len(result) != 10 {
		t.Errorf("Expected all 10 items (Page=0 is invalid), got %d", len(result))
	}
}

func TestPaginateItems_OffsetExceedsTotal(t *testing.T) {
	t.Parallel()

	items := make([]*ContentItem, 5)
	for i := range 5 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	result := PaginateItems(items, &PaginationOptions{Offset: 10, Limit: 5})
	if len(result) != 0 {
		t.Errorf("Expected 0 items when offset exceeds total, got %d", len(result))
	}
}

func TestPaginateItems_OffsetClipsEnd(t *testing.T) {
	t.Parallel()

	items := make([]*ContentItem, 10)
	for i := range 10 {
		items[i] = createTestItem(string(rune(i)), map[string]any{"index": i})
	}

	result := PaginateItems(items, &PaginationOptions{Offset: 7, Limit: 10})
	if len(result) != 3 {
		t.Errorf("Expected 3 items (10 - 7), got %d", len(result))
	}
	if result[0].Metadata["index"] != 7 {
		t.Errorf("Expected first item index 7, got %v", result[0].Metadata["index"])
	}
}
