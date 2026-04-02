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

func BenchmarkGet_Hit(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, data := range dataSizes() {
			b.Run(fmt.Sprintf("Provider=%s/Size=%s", provider, data.Name), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				key := "benchmark-key"
				cache.Set(ctx, key, data.Data)

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					_, _ = cache.Get(ctx, key, noopLoader[string, string]())
				}
			})
		}
	}
}

func BenchmarkGet_Miss(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, data := range dataSizes() {
			b.Run(fmt.Sprintf("Provider=%s/Size=%s", provider, data.Name), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				loader := simpleLoader[string, string](data.Data)

				b.ResetTimer()
				b.ReportAllocs()

				i := 0
				for b.Loop() {

					key := fmt.Sprintf("miss-key-%d", i)
					_, _ = cache.Get(ctx, key, loader)
					i++
				}
			})
		}
	}
}

func BenchmarkSet(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, data := range dataSizes() {
			b.Run(fmt.Sprintf("Provider=%s/Size=%s", provider, data.Name), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 10000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				b.ResetTimer()
				b.ReportAllocs()

				i := 0
				for b.Loop() {
					key := fmt.Sprintf("set-key-%d", i)
					cache.Set(ctx, key, data.Data)
					i++
				}
			})
		}
	}
}

func BenchmarkGetIfPresent(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, data := range dataSizes() {
			b.Run(fmt.Sprintf("Provider=%s/Size=%s", provider, data.Name), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				key := "present-key"
				cache.Set(ctx, key, data.Data)

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					_, _, _ = cache.GetIfPresent(ctx, key)
				}
			})
		}
	}
}

func BenchmarkInvalidate(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, data := range dataSizes() {
			b.Run(fmt.Sprintf("Provider=%s/Size=%s", provider, data.Name), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 10000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				i := 0
				for b.Loop() {
					key := fmt.Sprintf("invalidate-key-%d", i)
					cache.Set(ctx, key, data.Data)
					i++
				}

				b.ResetTimer()
				b.ReportAllocs()

				i = 0
				for b.Loop() {
					key := fmt.Sprintf("invalidate-key-%d", i)
					cache.Invalidate(ctx, key)
					i++
				}
			})
		}
	}
}
