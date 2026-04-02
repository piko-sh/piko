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

package cache_bench_test

import (
	"context"
	"fmt"
	"testing"

	"piko.sh/piko/internal/cache/cache_domain"
)

func BenchmarkAll(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, cacheSize := range []int{100, 500, 1000} {
			b.Run(fmt.Sprintf("Provider=%s/Size=%d", provider, cacheSize), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, cacheSize+100)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				for i := range cacheSize {
					key := fmt.Sprintf("iter-key-%d", i)
					cache.Set(ctx, key, data)
				}

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					count := 0
					for k, v := range cache.All() {
						_ = k
						_ = v
						count++
					}
					if count == 0 {
						b.Fatal("iterator returned no items")
					}
				}
			})
		}
	}
}

func BenchmarkValues(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, cacheSize := range []int{100, 500, 1000} {
			b.Run(fmt.Sprintf("Provider=%s/Size=%d", provider, cacheSize), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, cacheSize+100)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				for i := range cacheSize {
					key := fmt.Sprintf("values-key-%d", i)
					cache.Set(ctx, key, data)
				}

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					count := 0
					for v := range cache.Values() {
						_ = v
						count++
					}
					if count == 0 {
						b.Fatal("iterator returned no values")
					}
				}
			})
		}
	}
}

func BenchmarkAll_PartialIteration(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, breakAfter := range []int{10, 50, 100} {
			b.Run(fmt.Sprintf("Provider=%s/BreakAfter=%d", provider, breakAfter), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				for i := range 500 {
					key := fmt.Sprintf("partial-key-%d", i)
					cache.Set(ctx, key, data)
				}

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					count := 0
					for k, v := range cache.All() {
						_ = k
						_ = v
						count++
						if count >= breakAfter {
							break
						}
					}
				}
			})
		}
	}
}

func BenchmarkAll_WithFiltering(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		b.Run(fmt.Sprintf("Provider=%s", provider), func(b *testing.B) {
			var cache cache_domain.Cache[string, int]
			if provider == "Otter" {
				cache = setupOtterCache[string, int](b, 1000)
			} else {
				cache = setupRedisCache[string, int](b)
			}

			for i := range 500 {
				key := fmt.Sprintf("filter-key-%d", i)
				cache.Set(ctx, key, i)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				evenCount := 0
				for _, v := range cache.All() {
					if v%2 == 0 {
						evenCount++
					}
				}
				_ = evenCount
			}
		})
	}
}

func BenchmarkAll_Concurrent(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, goroutines := range []int{2, 4, 8} {
			b.Run(fmt.Sprintf("Provider=%s/Goroutines=%d", provider, goroutines), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				for i := range 500 {
					key := fmt.Sprintf("concurrent-iter-%d", i)
					cache.Set(ctx, key, data)
				}

				b.ResetTimer()
				b.ReportAllocs()

				runConcurrentBenchmark(b, goroutines, func(id, iteration int) {
					count := 0
					for k, v := range cache.All() {
						_ = k
						_ = v
						count++
					}
				})
			})
		}
	}
}
