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
	"piko.sh/piko/internal/cache/cache_dto"
)

func BenchmarkCompute(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, scenario := range []string{"Set", "Delete", "Noop"} {
			b.Run(fmt.Sprintf("Provider=%s/Action=%s", provider, scenario), func(b *testing.B) {
				var cache cache_domain.Cache[string, int]
				if provider == "Otter" {
					cache = setupOtterCache[string, int](b, 1000)
				} else {
					cache = setupRedisCache[string, int](b)
				}

				key := "compute-key"
				cache.Set(ctx, key, 100)

				var computeFunction func(oldValue int, found bool) (int, cache_dto.ComputeAction)
				switch scenario {
				case "Set":
					computeFunction = func(oldValue int, found bool) (int, cache_dto.ComputeAction) {
						return oldValue + 1, cache_dto.ComputeActionSet
					}
				case "Delete":
					computeFunction = func(oldValue int, found bool) (int, cache_dto.ComputeAction) {
						return 0, cache_dto.ComputeActionDelete
					}
				case "Noop":
					computeFunction = func(oldValue int, found bool) (int, cache_dto.ComputeAction) {
						return oldValue, cache_dto.ComputeActionNoop
					}
				}

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					cache.Compute(ctx, key, computeFunction)
				}
			})
		}
	}
}

func BenchmarkComputeIfAbsent(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, scenario := range []string{"Present", "Absent"} {
			b.Run(fmt.Sprintf("Provider=%s/State=%s", provider, scenario), func(b *testing.B) {
				var cache cache_domain.Cache[string, string]
				if provider == "Otter" {
					cache = setupOtterCache[string, string](b, 1000)
				} else {
					cache = setupRedisCache[string, string](b)
				}

				data := generateStringData(1024)

				if scenario == "Present" {

					cache.Set(ctx, "compute-absent-key", data)
				}

				computeFunction := func() string {
					return data
				}

				b.ResetTimer()
				b.ReportAllocs()

				i := 0
				for b.Loop() {
					if scenario == "Present" {
						cache.ComputeIfAbsent(ctx, "compute-absent-key", computeFunction)
					} else {

						key := fmt.Sprintf("compute-absent-key-%d", i)
						cache.ComputeIfAbsent(ctx, key, computeFunction)
					}
					i++
				}
			})
		}
	}
}

func BenchmarkComputeIfPresent(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, scenario := range []string{"Present", "Absent"} {
			b.Run(fmt.Sprintf("Provider=%s/State=%s", provider, scenario), func(b *testing.B) {
				var cache cache_domain.Cache[string, int]
				if provider == "Otter" {
					cache = setupOtterCache[string, int](b, 1000)
				} else {
					cache = setupRedisCache[string, int](b)
				}

				key := "compute-present-key"

				if scenario == "Present" {

					cache.Set(ctx, key, 100)
				}

				computeFunction := func(oldValue int) (int, cache_dto.ComputeAction) {
					return oldValue + 1, cache_dto.ComputeActionSet
				}

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					cache.ComputeIfPresent(ctx, key, computeFunction)
				}
			})
		}
	}
}

func BenchmarkCompute_Concurrent(b *testing.B) {
	ctx := context.Background()

	for _, provider := range []string{"Otter", "Redis"} {
		for _, goroutines := range []int{4, 8, 16} {
			b.Run(fmt.Sprintf("Provider=%s/Goroutines=%d", provider, goroutines), func(b *testing.B) {
				var cache cache_domain.Cache[string, int]
				if provider == "Otter" {
					cache = setupOtterCache[string, int](b, 1000)
				} else {
					cache = setupRedisCache[string, int](b)
				}

				key := "concurrent-compute-key"
				cache.Set(ctx, key, 0)

				computeFunction := func(oldValue int, found bool) (int, cache_dto.ComputeAction) {
					return oldValue + 1, cache_dto.ComputeActionSet
				}

				b.ResetTimer()
				b.ReportAllocs()

				runConcurrentBenchmark(b, goroutines, func(id, iteration int) {
					cache.Compute(ctx, key, computeFunction)
				})
			})
		}
	}
}
