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

package cache_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_dto"
)

func createOtterCache[K comparable, V any](t *testing.T, options cache_dto.Options[K, V]) *provider_otter.OtterAdapter[K, V] {
	t.Helper()

	cache, err := provider_otter.OtterProviderFactory(options)
	if err != nil {
		t.Fatalf("failed to create otter cache: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close(context.Background())
	})

	otterCache, ok := cache.(*provider_otter.OtterAdapter[K, V])
	if !ok {
		t.Fatalf("expected *provider_otter.OtterAdapter[K, V], got %T", cache)
	}

	return otterCache
}

func TestOtter_GetSetInvalidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key1", "value1")

	value, found, _ := cache.GetIfPresent(ctx, "key1")
	if !found {
		t.Fatal("expected key to be present")
	}
	Equal(t, value, "value1", "value mismatch")

	_ = cache.Invalidate(ctx, "key1")

	_, found, _ = cache.GetIfPresent(ctx, "key1")
	if found {
		t.Error("expected key to be absent after invalidation")
	}
}

func TestOtter_GetWithLoader(t *testing.T) {
	t.Parallel()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	ctx := context.Background()
	loadCount := 0

	loader := cache_dto.LoaderFunc[string, string](func(ctx context.Context, key string) (string, error) {
		loadCount++
		return "loaded-" + key, nil
	})

	value, err := cache.Get(ctx, "key1", loader)
	NoError(t, err, "Get with loader")
	Equal(t, value, "loaded-key1", "loaded value")
	Equal(t, loadCount, 1, "load count")

	value, err = cache.Get(ctx, "key1", loader)
	NoError(t, err, "Get with loader (cached)")
	Equal(t, value, "loaded-key1", "cached value")
	Equal(t, loadCount, 1, "load count should not increase")
}

func TestOtter_TagBasedInvalidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "user:1", "Alice", "user", "active")
	_ = cache.Set(ctx, "user:2", "Bob", "user", "inactive")
	_ = cache.Set(ctx, "product:1", "Widget", "product")

	_, found, _ := cache.GetIfPresent(ctx, "user:1")
	if !found {
		t.Error("user:1 should be present")
	}
	_, found, _ = cache.GetIfPresent(ctx, "user:2")
	if !found {
		t.Error("user:2 should be present")
	}
	_, found, _ = cache.GetIfPresent(ctx, "product:1")
	if !found {
		t.Error("product:1 should be present")
	}

	count, _ := cache.InvalidateByTags(ctx, "user")
	Equal(t, count, 2, "invalidated count")

	_, found, _ = cache.GetIfPresent(ctx, "user:1")
	if found {
		t.Error("user:1 should be invalidated")
	}
	_, found, _ = cache.GetIfPresent(ctx, "user:2")
	if found {
		t.Error("user:2 should be invalidated")
	}
	_, found, _ = cache.GetIfPresent(ctx, "product:1")
	if !found {
		t.Error("product:1 should still be present")
	}
}

func TestOtter_TagBasedInvalidation_MultipleTags(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key1", "value1", "tag1", "tag2")
	_ = cache.Set(ctx, "key2", "value2", "tag2", "tag3")
	_ = cache.Set(ctx, "key3", "value3", "tag3")

	count, _ := cache.InvalidateByTags(ctx, "tag1", "tag3")

	if count < 2 {
		t.Errorf("expected at least 2 keys invalidated, got %d", count)
	}
}

func TestOtter_InvalidateAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	for i := range 10 {
		_ = cache.Set(ctx, string(rune('a'+i)), "value")
	}

	if cache.EstimatedSize() == 0 {
		t.Error("cache should have entries")
	}

	_ = cache.InvalidateAll(ctx)

	if cache.EstimatedSize() != 0 {
		t.Errorf("cache should be empty, got size %d", cache.EstimatedSize())
	}
}

func TestOtter_MaximumSize(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	const maxSize = 10

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: maxSize,
	})

	for i := range maxSize * 2 {
		_ = cache.Set(ctx, string(rune('a'+i)), "value")
	}

	time.Sleep(100 * time.Millisecond)

	size := cache.EstimatedSize()
	if size > maxSize*2 {
		t.Errorf("cache size %d exceeds reasonable limit (max %d)", size, maxSize)
	}
}

func TestOtter_Concurrent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, int]{
		MaximumSize: 1000,
	})

	const numGoroutines = 50
	const opsPerGoroutine = 100

	RunConcurrent(t, numGoroutines, func(id int) {
		for i := range opsPerGoroutine {
			key := string(rune('a' + (id % 26)))
			_ = cache.Set(ctx, key, id*opsPerGoroutine+i)
		}
	})

	RunConcurrent(t, numGoroutines, func(id int) {
		for range opsPerGoroutine {
			key := string(rune('a' + (id % 26)))
			_, _, _ = cache.GetIfPresent(ctx, key)
		}
	})

	RunConcurrent(t, numGoroutines/10, func(id int) {
		_ = cache.Set(ctx, string(rune('A'+id)), 999, "test-tag")
	})

	_, _ = cache.InvalidateByTags(ctx, "test-tag")
}

func TestOtter_Stats(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key1", "value1")
	_ = cache.Set(ctx, "key2", "value2")

	_, _, _ = cache.GetIfPresent(ctx, "key1")
	_, _, _ = cache.GetIfPresent(ctx, "key1")

	_, _, _ = cache.GetIfPresent(ctx, "key-nonexistent")

	stats := cache.Stats()

	if stats.HitRatio() < 0 || stats.HitRatio() > 1 {
		t.Errorf("invalid hit ratio: %f", stats.HitRatio())
	}

	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {

		t.Logf("Stats recorded: Hits=%d, Misses=%d, Total=%d",
			stats.Hits, stats.Misses, totalRequests)
	} else {
		t.Log("Stats recording not enabled (no StatsRecorder configured)")
	}
}

func TestOtter_EstimatedSize(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	Equal(t, cache.EstimatedSize(), 0, "initial size")

	_ = cache.Set(ctx, "key1", "value1")
	_ = cache.Set(ctx, "key2", "value2")

	size := cache.EstimatedSize()
	if size < 2 {
		t.Errorf("expected size >= 2, got %d", size)
	}

	_ = cache.Invalidate(ctx, "key1")

	size = cache.EstimatedSize()
	if size < 1 {
		t.Errorf("expected size >= 1 after one invalidation, got %d", size)
	}
}

func TestOtter_Keys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	expectedKeys := map[string]bool{
		"key1": true,
		"key2": true,
		"key3": true,
	}

	for key := range expectedKeys {
		_ = cache.Set(ctx, key, "value")
	}

	foundKeys := make(map[string]bool)
	for key := range cache.Keys() {
		foundKeys[key] = true
	}

	for expected := range expectedKeys {
		if !foundKeys[expected] {
			t.Errorf("expected key %q not found in iteration", expected)
		}
	}
}

func TestOtter_Values(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key1", "value1")
	_ = cache.Set(ctx, "key2", "value2")
	_ = cache.Set(ctx, "key3", "value3")

	count := 0
	for range cache.Values() {
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 values, got %d", count)
	}
}

func TestOtter_All(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range expected {
		_ = cache.Set(ctx, k, v)
	}

	found := maps.Collect(cache.All())

	for k, expectedV := range expected {
		if foundV, ok := found[k]; !ok {
			t.Errorf("key %q not found in iteration", k)
		} else if foundV != expectedV {
			t.Errorf("key %q: expected %q, got %q", k, expectedV, foundV)
		}
	}
}

func TestOtter_GetEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key1", "value1")

	entry, found, _ := cache.GetEntry(ctx, "key1")
	if !found {
		t.Fatal("expected entry to be found")
	}

	Equal(t, entry.Key, "key1", "entry key")
	Equal(t, entry.Value, "value1", "entry value")
}

func TestOtter_Compute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, int]{
		MaximumSize: 100,
	})

	value, found, _ := cache.Compute(ctx, "counter", func(oldValue int, exists bool) (int, cache_dto.ComputeAction) {
		if !exists {
			return 1, cache_dto.ComputeActionSet
		}
		return oldValue + 1, cache_dto.ComputeActionSet
	})

	if !found {
		t.Error("expected Compute to return found=true after setting")
	}
	Equal(t, value, 1, "initial value")

	value, found, _ = cache.Compute(ctx, "counter", func(oldValue int, exists bool) (int, cache_dto.ComputeAction) {
		if !exists {
			return 1, cache_dto.ComputeActionSet
		}
		return oldValue + 1, cache_dto.ComputeActionSet
	})

	if !found {
		t.Error("expected Compute to return found=true")
	}
	Equal(t, value, 2, "incremented value")
}

func TestOtter_ComputeIfAbsent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	value, computed, _ := cache.ComputeIfAbsent(ctx, "key1", func() string {
		return "computed-value"
	})

	if !computed {
		t.Error("expected computation to occur for absent key")
	}
	Equal(t, value, "computed-value", "computed value")

	value, _, _ = cache.ComputeIfAbsent(ctx, "key1", func() string {
		return "different-value"
	})

	Equal(t, value, "computed-value", "original value should be retained")
}

func TestOtter_ComputeIfPresent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_, found, _ := cache.ComputeIfPresent(ctx, "key1", func(oldValue string) (string, cache_dto.ComputeAction) {
		return "should-not-happen", cache_dto.ComputeActionSet
	})

	if found {
		t.Error("expected no computation for absent key")
	}

	_ = cache.Set(ctx, "key1", "original")

	value, found, _ := cache.ComputeIfPresent(ctx, "key1", func(oldValue string) (string, cache_dto.ComputeAction) {
		return oldValue + "-modified", cache_dto.ComputeActionSet
	})

	if !found {
		t.Error("expected computation for present key")
	}
	Equal(t, value, "original-modified", "modified value")
}

func TestOtter_SetMaximum(t *testing.T) {
	t.Parallel()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	initialMax := cache.GetMaximum()
	if initialMax != 100 {
		t.Errorf("expected initial max 100, got %d", initialMax)
	}

	cache.SetMaximum(200)

	newMax := cache.GetMaximum()
	if newMax != 200 {
		t.Errorf("expected new max 200, got %d", newMax)
	}
}

func TestOtter_BulkGet(t *testing.T) {
	t.Parallel()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	ctx := context.Background()

	_ = cache.Set(ctx, "key1", "value1")
	_ = cache.Set(ctx, "key2", "value2")

	bulkLoader := cache_dto.BulkLoaderFunc[string, string](func(ctx context.Context, keys []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, key := range keys {
			result[key] = "loaded-" + key
		}
		return result, nil
	})

	keys := []string{"key1", "key2", "key3", "key4"}
	results, err := cache.BulkGet(ctx, keys, bulkLoader)
	NoError(t, err, "BulkGet")

	Equal(t, results["key1"], "value1", "existing key1")
	Equal(t, results["key2"], "value2", "existing key2")
	Equal(t, results["key3"], "loaded-key3", "loaded key3")
	Equal(t, results["key4"], "loaded-key4", "loaded key4")
}

func TestOtter_UpdateExistingValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key1", "value1")

	value, found, _ := cache.GetIfPresent(ctx, "key1")
	if !found || value != "value1" {
		t.Fatal("initial value not set correctly")
	}

	_ = cache.Set(ctx, "key1", "value2")

	value, found, _ = cache.GetIfPresent(ctx, "key1")
	if !found || value != "value2" {
		t.Error("value not updated correctly")
	}
}

func TestOtter_Close(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key1", "value1")

	_ = cache.Close(ctx)
}
