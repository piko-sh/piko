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
	"sync"
	"testing"

	"piko.sh/piko/internal/cache/cache_domain"
)

func BenchmarkConcurrent_Reads(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, goroutines := range []int{4, 8, 16, 32} {
			b.Run(fmt.Sprintf("Provider=%s/Goroutines=%d", provider, goroutines), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				key := "concurrent-read-key"
				data := generateStringData(1024)
				cache.Set(ctx, key, data)

				b.ResetTimer()
				b.ReportAllocs()

				runConcurrentBenchmark(b, goroutines, func(id, iteration int) {
					_, _ = cache.Get(ctx, key, noopLoader[string, string]())
				})
			})
		}
	}
}

func BenchmarkConcurrent_Writes(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, goroutines := range []int{4, 8, 16, 32} {
			b.Run(fmt.Sprintf("Provider=%s/Goroutines=%d", provider, goroutines), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 10000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				b.ResetTimer()
				b.ReportAllocs()

				runConcurrentBenchmark(b, goroutines, func(id, iteration int) {
					key := fmt.Sprintf("write-key-%d-%d", id, iteration)
					cache.Set(ctx, key, data)
				})
			})
		}
	}
}

func BenchmarkConcurrent_MixedReadWrite(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, goroutines := range []int{4, 8, 16, 32} {
			b.Run(fmt.Sprintf("Provider=%s/Goroutines=%d", provider, goroutines), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 10000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				for i := range 100 {
					key := fmt.Sprintf("mixed-key-%d", i)
					cache.Set(ctx, key, data)
				}

				b.ResetTimer()
				b.ReportAllocs()

				runConcurrentBenchmark(b, goroutines, func(id, iteration int) {

					if iteration%5 == 0 {

						key := fmt.Sprintf("mixed-key-write-%d-%d", id, iteration)
						cache.Set(ctx, key, data)
					} else {

						key := fmt.Sprintf("mixed-key-%d", iteration%100)
						_, _ = cache.Get(ctx, key, simpleLoader[string, string](data))
					}
				})
			})
		}
	}
}

func BenchmarkConcurrent_Invalidations(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, goroutines := range []int{4, 8, 16, 32} {
			b.Run(fmt.Sprintf("Provider=%s/Goroutines=%d", provider, goroutines), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 10000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				i := 0
				for b.Loop() {
					key := fmt.Sprintf("invalidate-key-%d", i)
					cache.Set(ctx, key, data)
					i++
				}

				b.ResetTimer()
				b.ReportAllocs()

				var counter int64
				var mu sync.Mutex
				runConcurrentBenchmark(b, goroutines, func(id, iteration int) {
					mu.Lock()
					index := counter
					counter++
					mu.Unlock()

					if int(index) < b.N {
						key := fmt.Sprintf("invalidate-key-%d", index)
						cache.Invalidate(ctx, key)
					}
				})
			})
		}
	}
}

func BenchmarkConcurrent_GetIfPresent(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, goroutines := range []int{4, 8, 16, 32} {
			b.Run(fmt.Sprintf("Provider=%s/Goroutines=%d", provider, goroutines), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				key := "concurrent-present-key"
				data := generateStringData(1024)
				cache.Set(ctx, key, data)

				b.ResetTimer()
				b.ReportAllocs()

				runConcurrentBenchmark(b, goroutines, func(id, iteration int) {
					_, _, _ = cache.GetIfPresent(ctx, key)
				})
			})
		}
	}
}
