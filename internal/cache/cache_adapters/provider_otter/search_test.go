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

package provider_otter

import (
	"context"
	"fmt"
	"testing"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

type Product struct {
	ID          string
	Name        string
	Description string
	Category    string
	Tags        []string
	Price       float64
	Stock       int
	Rating      float64
}

func generateProducts(n int) []Product {
	categories := []string{"electronics", "clothing", "books", "home", "sports", "toys", "food", "health"}
	adjectives := []string{"premium", "basic", "professional", "compact", "deluxe", "ultra", "eco", "smart"}
	nouns := []string{"widget", "gadget", "device", "tool", "item", "product", "accessory", "kit"}

	products := make([]Product, n)
	for i := range n {
		adj := adjectives[i%len(adjectives)]
		noun := nouns[i%len(nouns)]
		cat := categories[i%len(categories)]

		products[i] = Product{
			ID:          fmt.Sprintf("prod-%d", i),
			Name:        fmt.Sprintf("%s %s %d", adj, noun, i),
			Description: fmt.Sprintf("This is a %s %s in the %s category. High quality and durable.", adj, noun, cat),
			Category:    cat,
			Price:       float64(10 + (i % 990)),
			Stock:       i % 1000,
			Tags:        []string{cat, adj},
			Rating:      float64(1 + (i % 5)),
		}
	}
	return products
}

func requireOtterAdapter(t *testing.T, adapter cache_domain.ProviderPort[string, Product]) *OtterAdapter[string, Product] {
	t.Helper()
	otter, ok := adapter.(*OtterAdapter[string, Product])
	if !ok {
		t.Fatalf("unexpected adapter type: %T", adapter)
	}
	return otter
}

func TestOtterAdapter_Search_BasicFullText(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.TextField("Description"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	products := []Product{
		{ID: "1", Name: "Premium Widget", Description: "High quality widget", Category: "electronics", Price: 100},
		{ID: "2", Name: "Basic Gadget", Description: "Simple gadget", Category: "electronics", Price: 50},
		{ID: "3", Name: "Premium Device", Description: "Professional premium device", Category: "electronics", Price: 200},
		{ID: "4", Name: "Basic Tool", Description: "Standard tool", Category: "tools", Price: 30},
	}

	for _, p := range products {
		_ = cache.Set(ctx, p.ID, p)
	}

	tests := []struct {
		name          string
		query         string
		expectedIDs   []string
		expectedCount int64
	}{
		{
			name:          "single term match",
			query:         "premium",
			expectedIDs:   []string{"1", "3"},
			expectedCount: 2,
		},
		{
			name:          "multiple term match",
			query:         "premium widget",
			expectedIDs:   []string{"1"},
			expectedCount: 1,
		},
		{
			name:          "no match",
			query:         "nonexistent",
			expectedIDs:   []string{},
			expectedCount: 0,
		},
		{
			name:          "match in description",
			query:         "professional",
			expectedIDs:   []string{"3"},
			expectedCount: 1,
		},
		{
			name:          "match electronics",
			query:         "gadget",
			expectedIDs:   []string{"2"},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cache.Search(ctx, tt.query, &cache_dto.SearchOptions{Limit: 10})
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if result.Total != tt.expectedCount {
				t.Errorf("expected total %d, got %d", tt.expectedCount, result.Total)
			}

			if len(result.Items) != len(tt.expectedIDs) {
				t.Errorf("expected %d results, got %d", len(tt.expectedIDs), len(result.Items))
			}

			foundIDs := make(map[string]bool)
			for _, item := range result.Items {
				foundIDs[item.Key] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("expected to find ID %s in results", expectedID)
				}
			}
		})
	}
}

func TestOtterAdapter_Search_Pagination(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.SortableNumericField("Price"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	for i := range 25 {
		p := Product{
			ID:    string(rune('A' + i)),
			Name:  "Widget Item",
			Price: float64(i),
		}
		_ = cache.Set(ctx, p.ID, p)
	}

	result1, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{
		Offset:    0,
		Limit:     10,
		SortBy:    "Price",
		SortOrder: cache_dto.SortAsc,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result1.Total != 25 {
		t.Errorf("expected total 25, got %d", result1.Total)
	}
	if len(result1.Items) != 10 {
		t.Errorf("expected 10 items on page 1, got %d", len(result1.Items))
	}
	if result1.Offset != 0 {
		t.Errorf("expected offset 0, got %d", result1.Offset)
	}
	if !result1.HasMore() {
		t.Error("expected HasMore to be true")
	}

	for i := range 9 {
		if result1.Items[i].Value.Price > result1.Items[i+1].Value.Price {
			t.Errorf("results not sorted: item %d price %f > item %d price %f",
				i, result1.Items[i].Value.Price, i+1, result1.Items[i+1].Value.Price)
		}
	}

	result2, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{
		Offset:    10,
		Limit:     10,
		SortBy:    "Price",
		SortOrder: cache_dto.SortAsc,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result2.Items) != 10 {
		t.Errorf("expected 10 items on page 2, got %d", len(result2.Items))
	}
	if result2.Offset != 10 {
		t.Errorf("expected offset 10, got %d", result2.Offset)
	}

	result3, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{
		Offset:    20,
		Limit:     10,
		SortBy:    "Price",
		SortOrder: cache_dto.SortAsc,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result3.Items) != 5 {
		t.Errorf("expected 5 items on page 3, got %d", len(result3.Items))
	}
	if result3.HasMore() {
		t.Error("expected HasMore to be false on last page")
	}

	result4, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{
		Offset: 30,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result4.Items) != 0 {
		t.Errorf("expected 0 items beyond last page, got %d", len(result4.Items))
	}
}

func TestOtterAdapter_Search_Filters(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.TagField("Category"),
		cache_dto.NumericField("Price"),
		cache_dto.NumericField("Stock"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	products := []Product{
		{ID: "1", Name: "Widget A", Category: "electronics", Price: 100, Stock: 50},
		{ID: "2", Name: "Widget B", Category: "electronics", Price: 200, Stock: 30},
		{ID: "3", Name: "Widget C", Category: "tools", Price: 150, Stock: 20},
		{ID: "4", Name: "Widget D", Category: "electronics", Price: 300, Stock: 10},
		{ID: "5", Name: "Widget E", Category: "tools", Price: 250, Stock: 40},
	}

	for _, p := range products {
		_ = cache.Set(ctx, p.ID, p)
	}

	tests := []struct {
		name        string
		query       string
		filters     []cache_dto.Filter
		expectedIDs []string
	}{
		{
			name:  "filter by category",
			query: "widget",
			filters: []cache_dto.Filter{
				cache_dto.Eq("Category", "electronics"),
			},
			expectedIDs: []string{"1", "2", "4"},
		},
		{
			name:  "filter by price greater than",
			query: "widget",
			filters: []cache_dto.Filter{
				cache_dto.Gt("Price", 150.0),
			},
			expectedIDs: []string{"2", "4", "5"},
		},
		{
			name:  "filter by price between",
			query: "widget",
			filters: []cache_dto.Filter{
				cache_dto.Between("Price", 100.0, 200.0),
			},
			expectedIDs: []string{"1", "2", "3"},
		},
		{
			name:  "multiple filters",
			query: "widget",
			filters: []cache_dto.Filter{
				cache_dto.Eq("Category", "electronics"),
				cache_dto.Lt("Price", 250.0),
			},
			expectedIDs: []string{"1", "2"},
		},
		{
			name:  "filter by in",
			query: "widget",
			filters: []cache_dto.Filter{
				cache_dto.In("Category", "electronics", "tools"),
			},
			expectedIDs: []string{"1", "2", "3", "4", "5"},
		},
		{
			name:  "filter greater than or equal",
			query: "widget",
			filters: []cache_dto.Filter{
				cache_dto.Ge("Price", 200.0),
			},
			expectedIDs: []string{"2", "4", "5"},
		},
		{
			name:  "filter less than or equal",
			query: "widget",
			filters: []cache_dto.Filter{
				cache_dto.Le("Stock", 30),
			},
			expectedIDs: []string{"2", "3", "4"},
		},
		{
			name:  "filter not equal",
			query: "widget",
			filters: []cache_dto.Filter{
				cache_dto.Ne("Category", "electronics"),
			},
			expectedIDs: []string{"3", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cache.Search(ctx, tt.query, &cache_dto.SearchOptions{
				Limit:   10,
				Filters: tt.filters,
			})
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(result.Items) != len(tt.expectedIDs) {
				t.Errorf("expected %d results, got %d", len(tt.expectedIDs), len(result.Items))
			}

			foundIDs := make(map[string]bool)
			for _, item := range result.Items {
				foundIDs[item.Key] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("expected to find ID %s in results", expectedID)
				}
			}
		})
	}
}

func TestOtterAdapter_Search_Sorting(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.SortableNumericField("Price"),
		cache_dto.SortableNumericField("Rating"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	products := []Product{
		{ID: "1", Name: "Widget A", Price: 300, Rating: 2.0},
		{ID: "2", Name: "Widget B", Price: 100, Rating: 5.0},
		{ID: "3", Name: "Widget C", Price: 200, Rating: 3.0},
		{ID: "4", Name: "Widget D", Price: 150, Rating: 4.0},
	}

	for _, p := range products {
		_ = cache.Set(ctx, p.ID, p)
	}

	tests := []struct {
		name        string
		sortBy      string
		expectedIDs []string
		sortOrder   cache_dto.SortOrder
	}{
		{
			name:        "sort by price ascending",
			sortBy:      "Price",
			sortOrder:   cache_dto.SortAsc,
			expectedIDs: []string{"2", "4", "3", "1"},
		},
		{
			name:        "sort by price descending",
			sortBy:      "Price",
			sortOrder:   cache_dto.SortDesc,
			expectedIDs: []string{"1", "3", "4", "2"},
		},
		{
			name:        "sort by rating ascending",
			sortBy:      "Rating",
			sortOrder:   cache_dto.SortAsc,
			expectedIDs: []string{"1", "3", "4", "2"},
		},
		{
			name:        "sort by rating descending",
			sortBy:      "Rating",
			sortOrder:   cache_dto.SortDesc,
			expectedIDs: []string{"2", "4", "3", "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{
				Limit:     10,
				SortBy:    tt.sortBy,
				SortOrder: tt.sortOrder,
			})
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(result.Items) != len(tt.expectedIDs) {
				t.Fatalf("expected %d results, got %d", len(tt.expectedIDs), len(result.Items))
			}

			for i, item := range result.Items {
				if item.Key != tt.expectedIDs[i] {
					t.Errorf("position %d: expected ID %s, got %s", i, tt.expectedIDs[i], item.Key)
				}
			}
		})
	}
}

func TestOtterAdapter_Query_NoSearch(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TagField("Category"),
		cache_dto.SortableNumericField("Price"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	products := []Product{
		{ID: "1", Category: "electronics", Price: 100},
		{ID: "2", Category: "electronics", Price: 200},
		{ID: "3", Category: "tools", Price: 150},
		{ID: "4", Category: "electronics", Price: 300},
	}

	for _, p := range products {
		_ = cache.Set(ctx, p.ID, p)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{
			cache_dto.Eq("Category", "electronics"),
		},
		SortBy:    "Price",
		SortOrder: cache_dto.SortAsc,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}

	if len(result.Items) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Items))
	}

	expectedIDs := []string{"1", "2", "4"}
	for i, item := range result.Items {
		if item.Key != expectedIDs[i] {
			t.Errorf("position %d: expected ID %s, got %s", i, expectedIDs[i], item.Key)
		}
	}
}

func TestOtterAdapter_Search_EmptyQuery(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.SortableNumericField("Price"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	products := []Product{
		{ID: "1", Name: "Widget A", Price: 100},
		{ID: "2", Name: "Widget B", Price: 200},
		{ID: "3", Name: "Widget C", Price: 150},
	}

	for _, p := range products {
		_ = cache.Set(ctx, p.ID, p)
	}

	result, err := cache.Search(ctx, "", &cache_dto.SearchOptions{
		Limit:     10,
		SortBy:    "Price",
		SortOrder: cache_dto.SortAsc,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}

	if len(result.Items) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Items))
	}
}

func TestOtterAdapter_Search_UpdateIndex(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	_ = cache.Set(ctx, "1", Product{ID: "1", Name: "Widget A"})

	result, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 result, got %d", result.Total)
	}

	_ = cache.Set(ctx, "1", Product{ID: "1", Name: "Gadget B"})

	result, err = cache.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 results for old term, got %d", result.Total)
	}

	result, err = cache.Search(ctx, "gadget", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 result for new term, got %d", result.Total)
	}
}

func TestOtterAdapter_Search_InvalidateRemovesFromIndex(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	_ = cache.Set(ctx, "1", Product{ID: "1", Name: "Widget A"})
	_ = cache.Set(ctx, "2", Product{ID: "2", Name: "Widget B"})

	result, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 results, got %d", result.Total)
	}

	_ = cache.Invalidate(ctx, "1")

	result, err = cache.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 result after invalidation, got %d", result.Total)
	}
	if result.Items[0].Key != "2" {
		t.Errorf("expected remaining item to be '2', got '%s'", result.Items[0].Key)
	}
}

func TestOtterAdapter_Search_WithoutSchema(t *testing.T) {
	options := cache_dto.Options[string, Product]{
		MaximumSize: 100,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	_ = cache.Set(ctx, "1", Product{ID: "1", Name: "Widget A"})

	_, err = cache.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	if err == nil {
		t.Error("expected error when searching without schema, got nil")
	}

	_, err = cache.Query(ctx, &cache_dto.QueryOptions{Limit: 10})
	if err == nil {
		t.Error("expected error when querying without schema, got nil")
	}

	if cache.SupportsSearch() {
		t.Error("SupportsSearch should return false without schema")
	}
}

func TestOtterAdapter_Search_CaseInsensitive(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	_ = cache.Set(ctx, "1", Product{ID: "1", Name: "Premium Widget"})

	queries := []string{"premium", "PREMIUM", "Premium", "PrEmIuM"}
	for _, query := range queries {
		result, err := cache.Search(ctx, query, &cache_dto.SearchOptions{Limit: 10})
		if err != nil {
			t.Fatalf("Search failed for query '%s': %v", query, err)
		}
		if result.Total != 1 {
			t.Errorf("query '%s': expected 1 result, got %d", query, result.Total)
		}
	}
}

func TestOtterAdapter_Search_DefaultLimit(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	for i := range 20 {
		_ = cache.Set(ctx, string(rune('A'+i)), Product{
			ID:   string(rune('A' + i)),
			Name: "Widget Item",
		})
	}

	result, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 0})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Total != 20 {
		t.Errorf("expected total 20, got %d", result.Total)
	}

	if len(result.Items) != 10 {
		t.Errorf("expected default limit of 10 items, got %d", len(result.Items))
	}
}

func TestOtterAdapter_Search_WithTextAnalyser(t *testing.T) {

	mockAnalyser := func(text string) []string {
		words := splitWords(text)
		seen := make(map[string]struct{})
		var result []string
		for _, w := range words {

			if len(w) > 4 && w[len(w)-3:] == "ing" {
				w = w[:len(w)-3]
			}
			if _, ok := seen[w]; !ok {
				seen[w] = struct{}{}
				result = append(result, w)
			}
		}
		return result
	}

	schema := cache_dto.NewSearchSchemaWithAnalyser(
		mockAnalyser,
		cache_dto.TextField("Name"),
		cache_dto.TextField("Description"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	products := []Product{
		{ID: "1", Name: "Running Shoes", Description: "Fast running shoes"},
		{ID: "2", Name: "Walking Boots", Description: "Comfortable walking boots"},
		{ID: "3", Name: "Swim Goggles", Description: "Professional swim equipment"},
	}

	for _, p := range products {
		_ = cache.Set(ctx, p.ID, p)
	}

	result, err := cache.Search(ctx, "running", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("expected total 1 for stemmed search 'running', got %d", result.Total)
	}

	if len(result.Items) > 0 && result.Items[0].Key != "1" {
		t.Errorf("expected ID '1', got '%s'", result.Items[0].Key)
	}

	if len(result.Items) > 0 && result.Items[0].Score <= 0 {
		t.Errorf("expected positive BM25 score, got %f", result.Items[0].Score)
	}
}

func TestOtterAdapter_Search_WithTextAnalyser_ScoresOrdered(t *testing.T) {

	simpleAnalyser := func(text string) []string {
		return splitWords(text)
	}

	schema := cache_dto.NewSearchSchemaWithAnalyser(
		simpleAnalyser,
		cache_dto.TextField("Name"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer func() { _ = adapter.Close(context.Background()) }()

	cache := requireOtterAdapter(t, adapter)
	ctx := context.Background()

	_ = cache.Set(ctx, "short", Product{ID: "short", Name: "widget"})
	_ = cache.Set(ctx, "long", Product{ID: "long", Name: "widget gadget device tool item product accessory kit"})

	result, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}

	if len(result.Items) >= 2 {
		if result.Items[0].Score <= result.Items[1].Score {
			t.Errorf("expected first result to have higher BM25 score: %f <= %f",
				result.Items[0].Score, result.Items[1].Score)
		}
	}
}

func splitWords(text string) []string {
	var result []string
	seen := make(map[string]struct{})
	word := ""
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			word += string(r)
		} else if r >= 'A' && r <= 'Z' {
			word += string(r + 32)
		} else {
			if len(word) >= 2 {
				if _, ok := seen[word]; !ok {
					seen[word] = struct{}{}
					result = append(result, word)
				}
			}
			word = ""
		}
	}
	if len(word) >= 2 {
		if _, ok := seen[word]; !ok {
			result = append(result, word)
		}
	}
	return result
}
