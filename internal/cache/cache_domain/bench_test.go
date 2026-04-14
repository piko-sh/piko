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

package cache_domain

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

func generateBenchProducts(n int) []Product {
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

func BenchmarkInvertedIndex_Add(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			products := generateBenchProducts(size)
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
			products := generateBenchProducts(size)
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
			products := generateBenchProducts(size)
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
	products := generateBenchProducts(10000)
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
	products := generateBenchProducts(10000)
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
			products := generateBenchProducts(size)

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
	products := generateBenchProducts(10000)
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
		_, _ = extractor.ExtractAny(user, "Address.City")
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
