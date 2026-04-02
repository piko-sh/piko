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

//go:build bench

package provider_otter

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

func generateSearchTerms(n int) []string {
	baseTerms := []string{
		"premium",
		"widget",
		"electronics",
		"quality",
		"premium widget",
		"high quality durable",
		"professional device",
		"compact accessory",
		"smart gadget electronics",
		"basic tool home",
	}

	terms := make([]string, n)
	for i := range n {
		terms[i] = baseTerms[i%len(baseTerms)]
	}
	return terms
}

func createSearchableCache(b *testing.B, size int) *OtterAdapter[string, Product] {
	b.Helper()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.TextField("Description"),
		cache_dto.TagField("Category"),
		cache_dto.SortableNumericField("Price"),
		cache_dto.SortableNumericField("Rating"),
		cache_dto.NumericField("Stock"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  size * 2,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	if err != nil {
		b.Fatalf("failed to create adapter: %v", err)
	}

	products := generateProducts(size)
	for _, p := range products {
		adapter.Set(context.Background(), p.ID, p)
	}

	otter, ok := adapter.(*OtterAdapter[string, Product])
	if !ok {
		b.Fatalf("unexpected adapter type: %T", adapter)
	}
	return otter
}

func BenchmarkInvertedIndex_Add(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			products := generateProducts(size)
			index := NewInvertedIndex[string]()

			for i := range size / 2 {
				p := products[i]
				index.Add(p.ID, []string{p.Name, p.Description})
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; b.Loop(); i++ {

				p := products[i%size]
				index.Add(p.ID, []string{p.Name, p.Description})
			}
		})
	}
}

func BenchmarkInvertedIndex_Add_LongText(b *testing.B) {
	index := NewInvertedIndex[string]()

	var builder strings.Builder
	for i := range 100 {
		fmt.Fprintf(&builder, "word%d sentence%d paragraph%d ", i, i*2, i*3)
	}
	longText := builder.String()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("doc-%d", i%1000)
		index.Add(key, []string{longText})
	}
}

func BenchmarkInvertedIndex_Search_SingleTerm(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			products := generateProducts(size)
			index := NewInvertedIndex[string]()

			for _, p := range products {
				index.Add(p.ID, []string{p.Name, p.Description})
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = index.Search("premium")
			}
		})
	}
}

func BenchmarkInvertedIndex_Search_MultiTerm(b *testing.B) {
	sizes := []int{1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			products := generateProducts(size)
			index := NewInvertedIndex[string]()

			for _, p := range products {
				index.Add(p.ID, []string{p.Name, p.Description})
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = index.Search("premium widget quality")
			}
		})
	}
}

func BenchmarkInvertedIndex_Search_NoResults(b *testing.B) {
	products := generateProducts(10000)
	index := NewInvertedIndex[string]()

	for _, p := range products {
		index.Add(p.ID, []string{p.Name, p.Description})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = index.Search("nonexistent term xyz123")
	}
}

func BenchmarkInvertedIndex_Search_CommonTerm(b *testing.B) {
	products := generateProducts(10000)
	index := NewInvertedIndex[string]()

	for _, p := range products {
		index.Add(p.ID, []string{p.Name, p.Description})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {

		_ = index.Search("quality")
	}
}

func BenchmarkInvertedIndex_Remove(b *testing.B) {
	sizes := []int{1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			products := generateProducts(size)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				b.StopTimer()
				index := NewInvertedIndex[string]()
				for _, p := range products {
					index.Add(p.ID, []string{p.Name, p.Description})
				}
				b.StartTimer()

				for i := 0; i < size/2; i++ {
					index.Remove(products[i].ID)
				}
			}
		})
	}
}

func BenchmarkInvertedIndex_Concurrent_AddSearch(b *testing.B) {
	products := generateProducts(10000)
	index := NewInvertedIndex[string]()

	for _, p := range products[:5000] {
		index.Add(p.ID, []string{p.Name, p.Description})
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {

				_ = index.Search("premium")
			} else {

				p := products[(5000+i)%len(products)]
				index.Add(p.ID, []string{p.Name, p.Description})
			}
			i++
		}
	})
}

func BenchmarkSortedIndex_Add(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				b.StopTimer()
				index := NewSortedIndex[string]()
				b.StartTimer()

				for i := range size {
					index.Add(fmt.Sprintf("key-%d", i), float64(i))
				}
			}
		})
	}
}

func BenchmarkSortedIndex_Add_Sequential(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			index := NewSortedIndex[string]()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; b.Loop(); i++ {
				index.Add(fmt.Sprintf("key-%d", i), float64(i%size))
			}
		})
	}
}

func BenchmarkSortedIndex_Add_RandomOrder(b *testing.B) {
	sizes := []int{1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {

			values := make([]float64, size)
			for i := range size {
				values[i] = float64(i)
			}
			rand.Shuffle(size, func(i, j int) {
				values[i], values[j] = values[j], values[i]
			})

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				b.StopTimer()
				index := NewSortedIndex[string]()
				b.StartTimer()

				for i, v := range values {
					index.Add(fmt.Sprintf("key-%d", i), v)
				}
			}
		})
	}
}

func BenchmarkSortedIndex_Add_Update(b *testing.B) {
	index := NewSortedIndex[string]()

	for i := range 1000 {
		index.Add(fmt.Sprintf("key-%d", i), float64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {

		key := fmt.Sprintf("key-%d", i%1000)
		index.Add(key, float64(rand.IntN(10000)))
	}
}

func BenchmarkSortedIndex_Keys(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d/asc", size), func(b *testing.B) {
			index := NewSortedIndex[string]()
			for i := range size {
				index.Add(fmt.Sprintf("key-%d", i), float64(i))
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = index.Keys(true)
			}
		})

		b.Run(fmt.Sprintf("size=%d/desc", size), func(b *testing.B) {
			index := NewSortedIndex[string]()
			for i := range size {
				index.Add(fmt.Sprintf("key-%d", i), float64(i))
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = index.Keys(false)
			}
		})
	}
}

func BenchmarkSortedIndex_KeysFiltered(b *testing.B) {
	testCases := []struct {
		name       string
		totalSize  int
		filterSize int
	}{
		{name: "1000_filter_100", totalSize: 1000, filterSize: 100},
		{name: "1000_filter_500", totalSize: 1000, filterSize: 500},
		{name: "10000_filter_100", totalSize: 10000, filterSize: 100},
		{name: "10000_filter_1000", totalSize: 10000, filterSize: 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			index := NewSortedIndex[string]()
			for i := range tc.totalSize {
				index.Add(fmt.Sprintf("key-%d", i), float64(i))
			}

			filter := make(map[string]struct{}, tc.filterSize)
			for i := range tc.filterSize {
				filter[fmt.Sprintf("key-%d", i*tc.totalSize/tc.filterSize)] = struct{}{}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = index.KeysFiltered(filter, true)
			}
		})
	}
}

func BenchmarkSortedIndex_Remove(b *testing.B) {
	sizes := []int{1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				b.StopTimer()
				index := NewSortedIndex[string]()
				for i := range size {
					index.Add(fmt.Sprintf("key-%d", i), float64(i))
				}
				b.StartTimer()

				for i := size / 4; i < size*3/4; i++ {
					index.Remove(fmt.Sprintf("key-%d", i))
				}
			}
		})
	}
}

func BenchmarkOtterAdapter_Search_FullText(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			adapter := createSearchableCache(b, size)
			defer adapter.Close(context.Background())

			ctx := context.Background()
			opts := &cache_dto.SearchOptions{
				Limit: 10,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Search(ctx, "premium widget", opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Search_SingleTerm(b *testing.B) {
	sizes := []int{1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			adapter := createSearchableCache(b, size)
			defer adapter.Close(context.Background())

			ctx := context.Background()
			opts := &cache_dto.SearchOptions{
				Limit: 10,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Search(ctx, "premium", opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Search_MultiTerm(b *testing.B) {
	sizes := []int{1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			adapter := createSearchableCache(b, size)
			defer adapter.Close(context.Background())

			ctx := context.Background()
			opts := &cache_dto.SearchOptions{
				Limit: 10,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Search(ctx, "premium widget electronics quality", opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Search_NoResults(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()
	opts := &cache_dto.SearchOptions{
		Limit: 10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = adapter.Search(ctx, "nonexistent xyz123", opts)
	}
}

func BenchmarkOtterAdapter_Search_CommonTerm(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()
	opts := &cache_dto.SearchOptions{
		Limit: 10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {

		_, _ = adapter.Search(ctx, "quality", opts)
	}
}

func BenchmarkOtterAdapter_Search_WithPagination(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()

	testCases := []struct {
		name   string
		offset int
		limit  int
	}{
		{name: "page1_limit10", offset: 0, limit: 10},
		{name: "page10_limit10", offset: 90, limit: 10},
		{name: "page100_limit10", offset: 990, limit: 10},
		{name: "page1_limit100", offset: 0, limit: 100},
		{name: "page5_limit100", offset: 400, limit: 100},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			opts := &cache_dto.SearchOptions{
				Offset: tc.offset,
				Limit:  tc.limit,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Search(ctx, "premium", opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Search_WithSort(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()

	testCases := []struct {
		name  string
		field string
		order cache_dto.SortOrder
	}{
		{name: "price_asc", field: "Price", order: cache_dto.SortAsc},
		{name: "price_desc", field: "Price", order: cache_dto.SortDesc},
		{name: "rating_asc", field: "Rating", order: cache_dto.SortAsc},
		{name: "rating_desc", field: "Rating", order: cache_dto.SortDesc},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			opts := &cache_dto.SearchOptions{
				Limit:     50,
				SortBy:    tc.field,
				SortOrder: tc.order,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Search(ctx, "widget", opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Search_WithFilters(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()

	testCases := []struct {
		name    string
		filters []cache_dto.Filter
	}{
		{
			name:    "eq_category",
			filters: []cache_dto.Filter{cache_dto.Eq("Category", "electronics")},
		},
		{
			name:    "gt_price",
			filters: []cache_dto.Filter{cache_dto.Gt("Price", 500.0)},
		},
		{
			name:    "between_price",
			filters: []cache_dto.Filter{cache_dto.Between("Price", 100.0, 500.0)},
		},
		{
			name: "multiple_filters",
			filters: []cache_dto.Filter{
				cache_dto.Eq("Category", "electronics"),
				cache_dto.Gt("Price", 100.0),
				cache_dto.Lt("Stock", 500),
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			opts := &cache_dto.SearchOptions{
				Limit:   50,
				Filters: tc.filters,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Search(ctx, "widget", opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Query_NoSearch(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			adapter := createSearchableCache(b, size)
			defer adapter.Close(context.Background())

			ctx := context.Background()
			opts := &cache_dto.QueryOptions{
				Filters: []cache_dto.Filter{
					cache_dto.Eq("Category", "electronics"),
				},
				Limit: 50,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Query(ctx, opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Query_WithSort(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()
	opts := &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{
			cache_dto.Gt("Price", 100.0),
		},
		SortBy:    "Price",
		SortOrder: cache_dto.SortDesc,
		Limit:     100,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = adapter.Query(ctx, opts)
	}
}

func BenchmarkOtterAdapter_Query_InFilter(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()
	opts := &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{
			cache_dto.In("Category", "electronics", "clothing", "books"),
		},
		Limit: 100,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = adapter.Query(ctx, opts)
	}
}

func BenchmarkOtterAdapter_Search_Concurrent(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()
	searchTerms := generateSearchTerms(100)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			term := searchTerms[i%len(searchTerms)]
			opts := &cache_dto.SearchOptions{Limit: 10}
			_, _ = adapter.Search(ctx, term, opts)
			i++
		}
	})
}

func BenchmarkOtterAdapter_Mixed_SetAndSearch(b *testing.B) {
	adapter := createSearchableCache(b, 5000)
	defer adapter.Close(context.Background())

	ctx := context.Background()
	newProducts := generateProducts(5000)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%5 == 0 {

				p := newProducts[i%len(newProducts)]
				p.ID = "new-" + strconv.Itoa(i)
				adapter.Set(context.Background(), p.ID, p)
			} else {

				opts := &cache_dto.SearchOptions{Limit: 10}
				_, _ = adapter.Search(ctx, "premium", opts)
			}
			i++
		}
	})
}

func BenchmarkFieldExtractor_ExtractTextFields(b *testing.B) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.TextField("Description"),
	)

	extractor := NewFieldExtractor[Product](schema)
	product := Product{
		ID:          "test-1",
		Name:        "Premium Widget",
		Description: "A high quality premium widget for professional use",
		Category:    "electronics",
		Price:       99.99,
		Stock:       100,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = extractor.ExtractTextFields(product)
	}
}

func BenchmarkFieldExtractor_ExtractSortableValue(b *testing.B) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.SortableNumericField("Price"),
	)

	extractor := NewFieldExtractor[Product](schema)
	product := Product{
		ID:    "test-1",
		Price: 99.99,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = extractor.ExtractSortableValue(product, "Price")
	}
}

func BenchmarkFieldExtractor_ExtractAny_NestedField(b *testing.B) {
	type Address struct {
		City    string
		Country string
	}
	type User struct {
		Name    string
		Address Address
	}

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Address.City"),
	)

	extractor := NewFieldExtractor[User](schema)
	user := User{
		Name: "Test User",
		Address: Address{
			City:    "London",
			Country: "UK",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = extractor.extractAny(user, "Address.City")
	}
}

func BenchmarkInvertedIndex_MemoryPressure(b *testing.B) {

	index := NewInvertedIndex[string]()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("doc-%d", i%10000)

		text := fmt.Sprintf("unique%d term%d word%d", i, i*2, i*3)
		index.Add(key, []string{text})

		if i%10 == 0 {
			_ = index.Search(fmt.Sprintf("unique%d", i/10))
		}
	}
}

func BenchmarkSortedIndex_MemoryPressure(b *testing.B) {
	index := NewSortedIndex[string]()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("key-%d", i%10000)

		index.Add(key, float64(rand.IntN(1000000)))

		if i%10 == 0 {
			_ = index.Keys(true)
		}
	}
}

func BenchmarkOtterAdapter_Search_LargeResultSet(b *testing.B) {
	adapter := createSearchableCache(b, 100000)
	defer adapter.Close(context.Background())

	ctx := context.Background()

	testCases := []struct {
		name  string
		limit int
	}{
		{name: "limit_10", limit: 10},
		{name: "limit_100", limit: 100},
		{name: "limit_1000", limit: 1000},
		{name: "limit_10000", limit: 10000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			opts := &cache_dto.SearchOptions{
				Limit: tc.limit,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {

				_, _ = adapter.Search(ctx, "quality", opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Search_DeepPagination(b *testing.B) {
	adapter := createSearchableCache(b, 100000)
	defer adapter.Close(context.Background())

	ctx := context.Background()

	testCases := []struct {
		name   string
		offset int
		limit  int
	}{
		{name: "page_1", offset: 0, limit: 100},
		{name: "page_100", offset: 9900, limit: 100},
		{name: "page_500", offset: 49900, limit: 100},
		{name: "page_1000", offset: 99900, limit: 100},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			opts := &cache_dto.SearchOptions{
				Offset: tc.offset,
				Limit:  tc.limit,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Search(ctx, "quality", opts)
			}
		})
	}
}

func BenchmarkOtterAdapter_Search_ComplexQuery(b *testing.B) {
	adapter := createSearchableCache(b, 10000)
	defer adapter.Close(context.Background())

	ctx := context.Background()
	opts := &cache_dto.SearchOptions{
		Limit:     50,
		SortBy:    "Price",
		SortOrder: cache_dto.SortDesc,
		Filters: []cache_dto.Filter{
			cache_dto.Eq("Category", "electronics"),
			cache_dto.Between("Price", 100.0, 500.0),
			cache_dto.Gt("Rating", 3.0),
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = adapter.Search(ctx, "premium widget", opts)
	}
}

func BenchmarkOtterAdapter_Query_RangeFilter(b *testing.B) {
	sizes := []int{1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d/gt", size), func(b *testing.B) {
			adapter := createSearchableCache(b, size)
			defer adapter.Close(context.Background())

			ctx := context.Background()
			opts := &cache_dto.QueryOptions{
				Limit: 100,
				Filters: []cache_dto.Filter{
					cache_dto.Gt("Price", 500.0),
				},
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Query(ctx, opts)
			}
		})

		b.Run(fmt.Sprintf("size=%d/between", size), func(b *testing.B) {
			adapter := createSearchableCache(b, size)
			defer adapter.Close(context.Background())

			ctx := context.Background()
			opts := &cache_dto.QueryOptions{
				Limit: 100,
				Filters: []cache_dto.Filter{
					cache_dto.Between("Price", 200.0, 800.0),
				},
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Query(ctx, opts)
			}
		})

		b.Run(fmt.Sprintf("size=%d/lt", size), func(b *testing.B) {
			adapter := createSearchableCache(b, size)
			defer adapter.Close(context.Background())

			ctx := context.Background()
			opts := &cache_dto.QueryOptions{
				Limit: 100,
				Filters: []cache_dto.Filter{
					cache_dto.Lt("Price", 200.0),
				},
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.Query(ctx, opts)
			}
		})
	}
}
