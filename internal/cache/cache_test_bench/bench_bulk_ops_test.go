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

func BenchmarkBulkGet(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, keyCount := range []int{10, 50, 100, 500} {
			b.Run(fmt.Sprintf("Provider=%s/Keys=%d", provider, keyCount), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				keys := make([]string, keyCount)
				for i := range keyCount {
					key := fmt.Sprintf("bulk-key-%d", i)
					keys[i] = key
					cache.Set(ctx, key, data)
				}

				bulkLoader := simpleBulkLoader(func(k string) string {
					return data
				})

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					_, _ = cache.BulkGet(ctx, keys, bulkLoader)
				}
			})
		}
	}
}

func BenchmarkBulkGet_PartialMiss(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, keyCount := range []int{10, 50, 100, 500} {
			b.Run(fmt.Sprintf("Provider=%s/Keys=%d", provider, keyCount), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				keys := make([]string, keyCount)
				for i := range keyCount {
					key := fmt.Sprintf("bulk-partial-%d", i)
					keys[i] = key
					if i%2 == 0 {
						cache.Set(ctx, key, data)
					}
				}

				bulkLoader := simpleBulkLoader(func(k string) string {
					return data
				})

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					_, _ = cache.BulkGet(ctx, keys, bulkLoader)
				}
			})
		}
	}
}

func BenchmarkBulkRefresh(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, keyCount := range []int{10, 50, 100} {
			b.Run(fmt.Sprintf("Provider=%s/Keys=%d", provider, keyCount), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				keys := make([]string, keyCount)
				for i := range keyCount {
					key := fmt.Sprintf("bulk-refresh-%d", i)
					keys[i] = key
					cache.Set(ctx, key, data)
				}

				bulkLoader := simpleBulkLoader(func(k string) string {
					return data
				})

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					cache.BulkRefresh(ctx, keys, bulkLoader)
				}
			})
		}
	}
}

func BenchmarkInvalidateByTags(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, keyCount := range []int{10, 50, 100} {
			b.Run(fmt.Sprintf("Provider=%s/Keys=%d", provider, keyCount), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 10000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				b.ResetTimer()
				b.ReportAllocs()

				i := 0
				for b.Loop() {

					tag := fmt.Sprintf("bench-tag-%d", i)
					for j := range keyCount {
						key := fmt.Sprintf("tagged-key-%d-%d", i, j)
						cache.Set(ctx, key, data, tag)
					}

					cache.InvalidateByTags(ctx, tag)
					i++
				}
			})
		}
	}
}
