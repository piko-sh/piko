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
	"errors"
	"strings"
	"testing"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/wdk/cache"
)

func setupOtterCache[K comparable, V any](b *testing.B, maxSize int) cache.Cache[K, V] {
	b.Helper()

	c, err := provider_otter.OtterProviderFactory(cache.Options[K, V]{
		MaximumSize: maxSize,
	})
	if err != nil {
		b.Fatalf("failed to create otter cache: %v", err)
	}

	b.Cleanup(func() {
		c.Close(context.Background())
	})

	return c
}

func setupRedisCache[K comparable, V any](b *testing.B) cache.Cache[K, V] {
	b.Helper()
	b.Skip("Redis benchmarks are only available in the cache_provider_redis module")
	return nil
}

func generateStringData(size int) string {

	pattern := "The quick brown fox jumps over the lazy dog. "
	repeats := (size / len(pattern)) + 1
	data := strings.Repeat(pattern, repeats)
	return data[:size]
}

func generateByteData(size int) []byte {
	return []byte(generateStringData(size))
}

type TestData struct {
	Name string
	Data string
	Size int
}

func dataSizes() []TestData {
	return []TestData{
		{Name: "Size_100B", Data: generateStringData(100), Size: 100},
		{Name: "Size_1KB", Data: generateStringData(1024), Size: 1024},
		{Name: "Size_10KB", Data: generateStringData(10 * 1024), Size: 10 * 1024},
		{Name: "Size_100KB", Data: generateStringData(100 * 1024), Size: 100 * 1024},
	}
}

func runConcurrentBenchmark(b *testing.B, goroutines int, callback func(id int, iteration int)) {
	b.Helper()
	b.SetParallelism(goroutines)

	b.RunParallel(func(pb *testing.PB) {
		id := 0
		iteration := 0
		for pb.Next() {
			callback(id, iteration)
			iteration++
		}
	})
}

func noopLoader[K comparable, V any]() cache.Loader[K, V] {
	return cache.LoaderFunc[K, V](func(ctx context.Context, key K) (V, error) {
		var zero V
		return zero, errors.New("no loader")
	})
}

func simpleLoader[K comparable, V any](value V) cache.Loader[K, V] {
	return cache.LoaderFunc[K, V](func(ctx context.Context, key K) (V, error) {
		return value, nil
	})
}

func simpleBulkLoader[K comparable, V any](valueGen func(K) V) cache.BulkLoader[K, V] {
	return cache.BulkLoaderFunc[K, V](func(ctx context.Context, keys []K) (map[K]V, error) {
		result := make(map[K]V, len(keys))
		for _, key := range keys {
			result[key] = valueGen(key)
		}
		return result, nil
	})
}
