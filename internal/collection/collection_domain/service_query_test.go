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

package collection_domain

import (
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestApplyQueryOptions(t *testing.T) {
	service := newTestCollectionService(t)

	items := []collection_dto.ContentItem{
		{ID: "1", Locale: "en", Metadata: map[string]any{"title": "Alpha", "order": 2}},
		{ID: "2", Locale: "fr", Metadata: map[string]any{"title": "Beta", "order": 1}},
		{ID: "3", Locale: "en", Metadata: map[string]any{"title": "Gamma", "order": 3}},
	}

	t.Run("NoFiltersNoLocale", func(t *testing.T) {
		options := &collection_dto.FetchOptions{Filters: map[string]any{}}
		result := service.applyQueryOptions(items, options)
		if len(result) != 3 {
			t.Errorf("expected 3 items, got %d", len(result))
		}
	})

	t.Run("WithLocaleFilter", func(t *testing.T) {
		options := &collection_dto.FetchOptions{
			Locale:  "en",
			Filters: map[string]any{},
		}
		result := service.applyQueryOptions(items, options)
		if len(result) != 2 {
			t.Errorf("expected 2 en items, got %d", len(result))
		}
	})

	t.Run("WithSorting", func(t *testing.T) {
		options := &collection_dto.FetchOptions{
			Filters: map[string]any{
				"_sort": []collection_dto.SortOption{
					{Field: "title", Order: collection_dto.SortDesc},
				},
			},
		}
		result := service.applyQueryOptions(items, options)
		if len(result) != 3 {
			t.Fatalf("expected 3 items, got %d", len(result))
		}
		if result[0].Metadata["title"] != "Gamma" {
			t.Errorf("expected first item 'Gamma' (desc sort), got %v", result[0].Metadata["title"])
		}
	})

	t.Run("WithPagination", func(t *testing.T) {
		options := &collection_dto.FetchOptions{
			Filters: map[string]any{
				"_pagination": &collection_dto.PaginationOptions{
					Page:     1,
					PageSize: 2,
				},
			},
		}
		result := service.applyQueryOptions(items, options)
		if len(result) != 2 {
			t.Errorf("expected 2 items (page 1, size 2), got %d", len(result))
		}
	})
}

func TestConvertToPointerSlice(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		result := convertToPointerSlice(nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %d", len(result))
		}
	})

	t.Run("Multiple", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{ID: "1"},
			{ID: "2"},
		}
		result := convertToPointerSlice(items)
		if len(result) != 2 {
			t.Fatalf("expected 2, got %d", len(result))
		}
		if result[0].ID != "1" || result[1].ID != "2" {
			t.Error("pointer values do not match originals")
		}
	})
}

func TestConvertToValueSlice(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		result := convertToValueSlice(nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %d", len(result))
		}
	})

	t.Run("Multiple", func(t *testing.T) {
		a := collection_dto.ContentItem{ID: "a"}
		b := collection_dto.ContentItem{ID: "b"}
		result := convertToValueSlice([]*collection_dto.ContentItem{&a, &b})
		if len(result) != 2 {
			t.Fatalf("expected 2, got %d", len(result))
		}
		if result[0].ID != "a" || result[1].ID != "b" {
			t.Error("value copies do not match originals")
		}
	})
}

func TestApplyLocaleFilter(t *testing.T) {
	items := []*collection_dto.ContentItem{
		{Locale: "en"},
		{Locale: "fr"},
		{Locale: "en"},
	}

	t.Run("EmptyLocale", func(t *testing.T) {
		result := applyLocaleFilter(items, "")
		if len(result) != 3 {
			t.Errorf("expected all 3 items, got %d", len(result))
		}
	})

	t.Run("Matching", func(t *testing.T) {
		result := applyLocaleFilter(items, "en")
		if len(result) != 2 {
			t.Errorf("expected 2 en items, got %d", len(result))
		}
	})

	t.Run("NoMatch", func(t *testing.T) {
		result := applyLocaleFilter(items, "de")
		if len(result) != 0 {
			t.Errorf("expected 0 items, got %d", len(result))
		}
	})
}

func TestApplyCustomFilters(t *testing.T) {
	items := []*collection_dto.ContentItem{
		{Metadata: map[string]any{"status": "published"}},
		{Metadata: map[string]any{"status": "draft"}},
	}

	t.Run("NoKey", func(t *testing.T) {
		result := applyCustomFilters(items, map[string]any{})
		if len(result) != 2 {
			t.Errorf("expected 2 items unchanged, got %d", len(result))
		}
	})

	t.Run("WrongType", func(t *testing.T) {
		result := applyCustomFilters(items, map[string]any{"_filterGroup": "not-a-filter"})
		if len(result) != 2 {
			t.Errorf("expected 2 items unchanged, got %d", len(result))
		}
	})

	t.Run("Valid", func(t *testing.T) {
		fg := &collection_dto.FilterGroup{
			Logic: "AND",
			Filters: []collection_dto.Filter{
				{Field: "status", Operator: "eq", Value: "published"},
			},
		}
		result := applyCustomFilters(items, map[string]any{"_filterGroup": fg})
		if len(result) != 1 {
			t.Errorf("expected 1 published item, got %d", len(result))
		}
	})
}

func TestApplySorting(t *testing.T) {
	makeItems := func() []*collection_dto.ContentItem {
		return []*collection_dto.ContentItem{
			{Metadata: map[string]any{"title": "Banana"}},
			{Metadata: map[string]any{"title": "Apple"}},
			{Metadata: map[string]any{"title": "Cherry"}},
		}
	}

	t.Run("NoKey", func(t *testing.T) {
		items := makeItems()
		applySorting(items, map[string]any{})
		if items[0].Metadata["title"] != "Banana" {
			t.Error("expected items unchanged when no _sort key")
		}
	})

	t.Run("WrongType", func(t *testing.T) {
		items := makeItems()
		applySorting(items, map[string]any{"_sort": "not-a-slice"})
		if items[0].Metadata["title"] != "Banana" {
			t.Error("expected items unchanged for wrong _sort type")
		}
	})

	t.Run("EmptyOptions", func(t *testing.T) {
		items := makeItems()
		applySorting(items, map[string]any{"_sort": []collection_dto.SortOption{}})
		if items[0].Metadata["title"] != "Banana" {
			t.Error("expected items unchanged for empty sort options")
		}
	})

	t.Run("Valid", func(t *testing.T) {
		items := makeItems()
		applySorting(items, map[string]any{
			"_sort": []collection_dto.SortOption{
				{Field: "title", Order: collection_dto.SortAsc},
			},
		})
		if items[0].Metadata["title"] != "Apple" {
			t.Errorf("expected first item 'Apple' after asc sort, got %v", items[0].Metadata["title"])
		}
	})
}

func TestApplyPagination(t *testing.T) {
	items := []*collection_dto.ContentItem{
		{ID: "1"},
		{ID: "2"},
		{ID: "3"},
		{ID: "4"},
	}

	t.Run("NoKey", func(t *testing.T) {
		result := applyPagination(items, map[string]any{})
		if len(result) != 4 {
			t.Errorf("expected 4 items, got %d", len(result))
		}
	})

	t.Run("WrongType", func(t *testing.T) {
		result := applyPagination(items, map[string]any{"_pagination": 42})
		if len(result) != 4 {
			t.Errorf("expected 4 items unchanged, got %d", len(result))
		}
	})

	t.Run("Valid", func(t *testing.T) {
		result := applyPagination(items, map[string]any{
			"_pagination": &collection_dto.PaginationOptions{
				Page:     2,
				PageSize: 2,
			},
		})
		if len(result) != 2 {
			t.Errorf("expected 2 items (page 2 of 4), got %d", len(result))
		}
	})
}
