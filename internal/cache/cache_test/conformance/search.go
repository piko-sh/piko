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

package conformance

import (
	"context"
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

// runSearchTests runs the search functionality test suite.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache product configuration.
func runSearchTests(t *testing.T, config ProductConfig) {
	t.Helper()

	t.Run("SupportsSearch", func(t *testing.T) {
		t.Parallel()
		testSupportsSearch(t, config)
	})

	t.Run("GetSchema", func(t *testing.T) {
		t.Parallel()
		testGetSchema(t, config)
	})

	t.Run("Search_EmptyCache", func(t *testing.T) {
		t.Parallel()
		testSearchEmptyCache(t, config)
	})

	t.Run("Search_BasicQuery", func(t *testing.T) {
		t.Parallel()
		testSearchBasicQuery(t, config)
	})

	t.Run("Search_NoMatches", func(t *testing.T) {
		t.Parallel()
		testSearchNoMatches(t, config)
	})

	t.Run("Query_FilterEq", func(t *testing.T) {
		t.Parallel()
		testQueryFilterEq(t, config)
	})

	t.Run("Query_FilterNe", func(t *testing.T) {
		t.Parallel()
		testQueryFilterNe(t, config)
	})

	t.Run("Query_FilterGt", func(t *testing.T) {
		t.Parallel()
		testQueryFilterGt(t, config)
	})

	t.Run("Query_FilterGe", func(t *testing.T) {
		t.Parallel()
		testQueryFilterGe(t, config)
	})

	t.Run("Query_FilterLt", func(t *testing.T) {
		t.Parallel()
		testQueryFilterLt(t, config)
	})

	t.Run("Query_FilterLe", func(t *testing.T) {
		t.Parallel()
		testQueryFilterLe(t, config)
	})

	t.Run("Query_FilterIn", func(t *testing.T) {
		t.Parallel()
		testQueryFilterIn(t, config)
	})

	t.Run("Query_FilterBetween", func(t *testing.T) {
		t.Parallel()
		testQueryFilterBetween(t, config)
	})

	t.Run("Query_FilterPrefix", func(t *testing.T) {
		t.Parallel()
		testQueryFilterPrefix(t, config)
	})

	t.Run("Query_MultipleFilters", func(t *testing.T) {
		t.Parallel()
		testQueryMultipleFilters(t, config)
	})

	t.Run("Query_SortAscending", func(t *testing.T) {
		t.Parallel()
		testQuerySortAscending(t, config)
	})

	t.Run("Query_SortDescending", func(t *testing.T) {
		t.Parallel()
		testQuerySortDescending(t, config)
	})

	t.Run("Query_Pagination", func(t *testing.T) {
		t.Parallel()
		testQueryPagination(t, config)
	})

	t.Run("Query_HasMore", func(t *testing.T) {
		t.Parallel()
		testQueryHasMore(t, config)
	})
}

// testSupportsSearch verifies that a search-configured cache reports search
// support.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory to test.
func testSupportsSearch(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())

	if !cache.SupportsSearch() {
		t.Error("Search-configured cache should support search")
	}
}

// testGetSchema verifies that GetSchema returns a non-nil schema for a
// search-configured cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testGetSchema(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())

	schema := cache.GetSchema()
	if schema == nil {
		t.Error("GetSchema should return non-nil for search-configured cache")
	}
}

// testSearchEmptyCache verifies that searching an empty cache returns an empty
// result without error.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testSearchEmptyCache(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	result, err := cache.Search(ctx, "laptop", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Errorf("Search on empty cache failed: %v", err)
	}

	if !result.IsEmpty() {
		t.Error("Search on empty cache should return empty result")
	}
}

// testSearchBasicQuery verifies that basic search queries return expected results.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testSearchBasicQuery(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Laptop Pro", Description: "High-performance laptop", Price: 1299.99, Category: "electronics"}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "Desktop Computer", Description: "Powerful desktop", Price: 999.99, Category: "electronics"}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}

	result, err := cache.Search(ctx, "laptop", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}

	if result.IsEmpty() {
		t.Error("Search for 'laptop' should find results")
	}

	found := false
	for _, hit := range result.Items {
		if hit.Value.ID == "1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Search should find the laptop product")
	}
}

// testSearchNoMatches verifies that searching for a non-existent term returns
// an empty result set.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory for testing.
func testSearchNoMatches(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Laptop Pro", Price: 1299.99}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}

	result, err := cache.Search(ctx, "xyznonexistent", &cache_dto.SearchOptions{Limit: 10})
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}

	if !result.IsEmpty() {
		t.Errorf("Search for non-existent term should return empty: got %d", len(result.Items))
	}
}

// testQueryFilterEq tests that the Eq filter correctly matches items by field
// value.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testQueryFilterEq(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Laptop", Category: "electronics"}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "Chair", Category: "furniture"}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{cache_dto.Eq("category", "electronics")},
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	for _, hit := range result.Items {
		if hit.Value.Category != "electronics" {
			t.Errorf("Filter Eq should only return electronics: got %s", hit.Value.Category)
		}
	}
}

// testQueryFilterNe verifies that the not-equal filter excludes matching items.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testQueryFilterNe(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Laptop", Category: "electronics"}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "Chair", Category: "furniture"}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{cache_dto.Ne("category", "electronics")},
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	for _, hit := range result.Items {
		if hit.Value.Category == "electronics" {
			t.Error("Filter Ne should not return electronics")
		}
	}
}

// runProductQueryTest runs a product query test with the given filter.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory.
// Takes filter (cache_dto.Filter) which specifies the query filter to test.
// Takes check (func(...)) which validates each returned product.
// Takes errorMessage (string) which is the error format string for failures.
func runProductQueryTest(t *testing.T, config ProductConfig, filter cache_dto.Filter, check func(Product) bool, errorMessage string) {
	t.Helper()
	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Cheap", Price: 10.00}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "Medium", Price: 50.00}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}
	if err := cache.Set(ctx, "3", Product{ID: "3", Name: "Expensive", Price: 100.00}); err != nil {
		t.Fatalf("Set 3 failed: %v", err)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{filter},
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	for _, hit := range result.Items {
		if !check(hit.Value) {
			t.Errorf(errorMessage, hit.Value.Price)
		}
	}
}

// testQueryFilterGt tests the greater-than query filter.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the test configuration.
func testQueryFilterGt(t *testing.T, config ProductConfig) {
	runProductQueryTest(t, config, cache_dto.Gt("price", 50.00), func(p Product) bool {
		return p.Price > 50.00
	}, "Filter Gt(50) should only return price > 50: got %f")
}

// testQueryFilterGe verifies that the greater-than-or-equal filter works.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the product test configuration.
func testQueryFilterGe(t *testing.T, config ProductConfig) {
	runProductQueryTest(t, config, cache_dto.Ge("price", 50.00), func(p Product) bool {
		return p.Price >= 50.00
	}, "Filter Ge(50) should only return price >= 50: got %f")
}

// testQueryFilterLt verifies that the less-than filter returns only products
// with prices below the threshold.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the test configuration.
func testQueryFilterLt(t *testing.T, config ProductConfig) {
	runProductQueryTest(t, config, cache_dto.Lt("price", 50.00), func(p Product) bool {
		return p.Price < 50.00
	}, "Filter Lt(50) should only return price < 50: got %f")
}

// testQueryFilterLe tests the less-than-or-equal filter on product price.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the test configuration.
func testQueryFilterLe(t *testing.T, config ProductConfig) {
	runProductQueryTest(t, config, cache_dto.Le("price", 50.00), func(p Product) bool {
		return p.Price <= 50.00
	}, "Filter Le(50) should only return price <= 50: got %f")
}

// testQueryFilterIn verifies that the cache query correctly filters products
// using the In filter operator.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testQueryFilterIn(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Laptop", Category: "electronics"}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "Chair", Category: "furniture"}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}
	if err := cache.Set(ctx, "3", Product{ID: "3", Name: "Shirt", Category: "clothing"}); err != nil {
		t.Fatalf("Set 3 failed: %v", err)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{cache_dto.In("category", "electronics", "furniture")},
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	for _, hit := range result.Items {
		if hit.Value.Category != "electronics" && hit.Value.Category != "furniture" {
			t.Errorf("Filter In should only return electronics or furniture: got %s", hit.Value.Category)
		}
	}
}

// testQueryFilterBetween tests that the Between filter returns only products
// within the specified price range.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory for testing.
func testQueryFilterBetween(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Cheap", Price: 10.00}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "Medium", Price: 50.00}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}
	if err := cache.Set(ctx, "3", Product{ID: "3", Name: "Expensive", Price: 100.00}); err != nil {
		t.Fatalf("Set 3 failed: %v", err)
	}
	if err := cache.Set(ctx, "4", Product{ID: "4", Name: "Premium", Price: 200.00}); err != nil {
		t.Fatalf("Set 4 failed: %v", err)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{cache_dto.Between("price", 25.00, 150.00)},
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	for _, hit := range result.Items {
		if hit.Value.Price < 25.00 || hit.Value.Price > 150.00 {
			t.Errorf("Filter Between(25, 150) should only return prices in range: got %f", hit.Value.Price)
		}
	}
}

// testQueryFilterPrefix verifies that prefix filtering returns only matching
// items.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testQueryFilterPrefix(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Laptop", Category: "electronics"}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "Chair", Category: "furniture"}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}
	if err := cache.Set(ctx, "3", Product{ID: "3", Name: "Phone", Category: "electronics-mobile"}); err != nil {
		t.Fatalf("Set 3 failed: %v", err)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{cache_dto.Prefix("category", "elec")},
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	for _, hit := range result.Items {
		if len(hit.Value.Category) < 4 || hit.Value.Category[:4] != "elec" {
			t.Errorf("Filter Prefix should only return categories starting with 'elec': got %s", hit.Value.Category)
		}
	}
}

// testQueryMultipleFilters verifies that multiple query filters are combined
// with AND logic.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testQueryMultipleFilters(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "Cheap Laptop", Category: "electronics", Price: 500.00}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "Expensive Laptop", Category: "electronics", Price: 2000.00}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}
	if err := cache.Set(ctx, "3", Product{ID: "3", Name: "Chair", Category: "furniture", Price: 300.00}); err != nil {
		t.Fatalf("Set 3 failed: %v", err)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{
			cache_dto.Eq("category", "electronics"),
			cache_dto.Lt("price", 1000.00),
		},
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	for _, hit := range result.Items {
		if hit.Value.Category != "electronics" || hit.Value.Price >= 1000.00 {
			t.Errorf("Multiple filters should apply AND: got category=%s, price=%f", hit.Value.Category, hit.Value.Price)
		}
	}
}

// runProductSortTest runs a product cache query test with the given sort order.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
// Takes sortOrder (cache_dto.SortOrder) which specifies ascending or descending.
// Takes check (func(...)) which validates that two products are correctly ordered.
// Takes errorMessage (string) which is the format string for comparison failures.
func runProductSortTest(t *testing.T, config ProductConfig, sortOrder cache_dto.SortOrder, check func(p1, p2 Product) bool, errorMessage string) {
	t.Helper()
	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "1", Product{ID: "1", Name: "C", Price: 300.00}); err != nil {
		t.Fatalf("Set 1 failed: %v", err)
	}
	if err := cache.Set(ctx, "2", Product{ID: "2", Name: "A", Price: 100.00}); err != nil {
		t.Fatalf("Set 2 failed: %v", err)
	}
	if err := cache.Set(ctx, "3", Product{ID: "3", Name: "B", Price: 200.00}); err != nil {
		t.Fatalf("Set 3 failed: %v", err)
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		SortBy:    "price",
		SortOrder: sortOrder,
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	if len(result.Items) < 2 {
		return
	}

	for i := 1; i < len(result.Items); i++ {
		if !check(result.Items[i-1].Value, result.Items[i].Value) {
			t.Errorf(errorMessage, result.Items[i-1].Value.Price, result.Items[i].Value.Price)
		}
	}
}

// testQuerySortAscending verifies that products are sorted by price in
// ascending order.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the test configuration.
func testQuerySortAscending(t *testing.T, config ProductConfig) {
	runProductSortTest(t, config, cache_dto.SortAsc, func(p1, p2 Product) bool {
		return p1.Price <= p2.Price
	}, "Sort ascending should order by price: got %f before %f")
}

// testQuerySortDescending tests that query results are sorted in descending
// order by price.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the product cache configuration.
func testQuerySortDescending(t *testing.T, config ProductConfig) {
	runProductSortTest(t, config, cache_dto.SortDesc, func(p1, p2 Product) bool {
		return p1.Price >= p2.Price
	}, "Sort descending should order by price: got %f before %f")
}

// testQueryPagination verifies that cache queries respect limit and offset.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testQueryPagination(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	for i := range 10 {
		if err := cache.Set(ctx, string(rune('a'+i)), Product{ID: string(rune('a' + i)), Name: "Product", Price: float64(i * 10)}); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	result1, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Limit:  3,
		Offset: 0,
	})
	if err != nil {
		t.Errorf("Query page 1 failed: %v", err)
	}

	if len(result1.Items) > 3 {
		t.Errorf("First page should have at most 3 items: got %d", len(result1.Items))
	}

	result2, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Limit:  3,
		Offset: 3,
	})
	if err != nil {
		t.Errorf("Query page 2 failed: %v", err)
	}

	if len(result2.Items) > 3 {
		t.Errorf("Second page should have at most 3 items: got %d", len(result2.Items))
	}
}

// testQueryHasMore verifies that query results correctly report when more
// results are available.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which provides the cache factory and settings.
func testQueryHasMore(t *testing.T, config ProductConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultProductOptions())
	ctx := context.Background()

	for i := range 10 {
		if err := cache.Set(ctx, string(rune('a'+i)), Product{ID: string(rune('a' + i)), Name: "Product"}); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Limit:  5,
		Offset: 0,
	})
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	if !result.HasMore() {
		t.Error("HasMore should return true when more results exist")
	}

	resultAll, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		t.Errorf("Query all failed: %v", err)
	}

	if resultAll.HasMore() {
		t.Error("HasMore should return false when all results are returned")
	}
}
