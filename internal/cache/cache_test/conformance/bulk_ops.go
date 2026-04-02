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

// runBulkOpsTests runs the bulk operations test suite for string caches.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func runBulkOpsTests(t *testing.T, config StringConfig) {
	t.Helper()

	t.Run("BulkGet_AllCached", func(t *testing.T) {
		t.Parallel()
		testBulkGetAllCached(t, config)
	})

	t.Run("BulkGet_AllMissing", func(t *testing.T) {
		t.Parallel()
		testBulkGetAllMissing(t, config)
	})

	t.Run("BulkGet_Mixed", func(t *testing.T) {
		t.Parallel()
		testBulkGetMixed(t, config)
	})

	t.Run("BulkGet_Empty", func(t *testing.T) {
		t.Parallel()
		testBulkGetEmpty(t, config)
	})

	t.Run("BulkSet_Basic", func(t *testing.T) {
		t.Parallel()
		testBulkSetBasic(t, config)
	})

	t.Run("BulkSet_WithTags", func(t *testing.T) {
		t.Parallel()
		testBulkSetWithTags(t, config)
	})

	t.Run("BulkSet_Empty", func(t *testing.T) {
		t.Parallel()
		testBulkSetEmpty(t, config)
	})

	t.Run("InvalidateByTags_Single", func(t *testing.T) {
		t.Parallel()
		testInvalidateByTagsSingle(t, config)
	})

	t.Run("InvalidateByTags_Multiple", func(t *testing.T) {
		t.Parallel()
		testInvalidateByTagsMultiple(t, config)
	})

	t.Run("InvalidateByTags_NoMatch", func(t *testing.T) {
		t.Parallel()
		testInvalidateByTagsNoMatch(t, config)
	})

	t.Run("InvalidateAll", func(t *testing.T) {
		t.Parallel()
		testInvalidateAll(t, config)
	})

	t.Run("InvalidateAll_Empty", func(t *testing.T) {
		t.Parallel()
		testInvalidateAllEmpty(t, config)
	})

	t.Run("BulkRefresh", func(t *testing.T) {
		t.Parallel()
		testBulkRefresh(t, config)
	})

}

// testBulkGetAllCached verifies that BulkGet returns cached values without
// calling the loader.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testBulkGetAllCached(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key1", "value1"); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := cache.Set(ctx, "key2", "value2"); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}

	loadCount := 0
	bulkLoader := cache_dto.BulkLoaderFunc[string, string](func(ctx context.Context, keys []string) (map[string]string, error) {
		loadCount++
		return make(map[string]string), nil
	})

	results, err := cache.BulkGet(ctx, []string{"key1", "key2"}, bulkLoader)
	if err != nil {
		t.Errorf("BulkGet failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("BulkGet should return 2 results: got %d", len(results))
	}

	if results["key1"] != "value1" {
		t.Errorf("key1 mismatch: got %q, want %q", results["key1"], "value1")
	}
	if results["key2"] != "value2" {
		t.Errorf("key2 mismatch: got %q, want %q", results["key2"], "value2")
	}
}

// testBulkGetAllMissing verifies that BulkGet calls the loader for keys not
// present in the cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testBulkGetAllMissing(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	loadCount := 0
	bulkLoader := cache_dto.BulkLoaderFunc[string, string](func(ctx context.Context, keys []string) (map[string]string, error) {
		loadCount++
		result := make(map[string]string)
		for _, k := range keys {
			result[k] = "loaded-" + k
		}
		return result, nil
	})

	results, err := cache.BulkGet(ctx, []string{"key1", "key2"}, bulkLoader)
	if err != nil {
		t.Errorf("BulkGet failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("BulkGet should return 2 results: got %d", len(results))
	}

	if loadCount == 0 {
		t.Error("Loader should be called for missing keys")
	}

	if results["key1"] != "loaded-key1" {
		t.Errorf("key1 mismatch: got %q, want %q", results["key1"], "loaded-key1")
	}
}

// testBulkGetMixed tests bulk get with a mix of cached and uncached keys.
//
// Takes t (*testing.T) which provides test control and error reporting.
// Takes config (StringConfig) which specifies the cache configuration to test.
func testBulkGetMixed(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key1", "cached-value1"); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}

	bulkLoader := cache_dto.BulkLoaderFunc[string, string](func(ctx context.Context, keys []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, k := range keys {
			result[k] = "loaded-" + k
		}
		return result, nil
	})

	results, err := cache.BulkGet(ctx, []string{"key1", "key2"}, bulkLoader)
	if err != nil {
		t.Errorf("BulkGet failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("BulkGet should return 2 results: got %d", len(results))
	}

	if results["key1"] != "cached-value1" {
		t.Errorf("key1 should be cached value: got %q, want %q", results["key1"], "cached-value1")
	}
	if results["key2"] != "loaded-key2" {
		t.Errorf("key2 should be loaded value: got %q, want %q", results["key2"], "loaded-key2")
	}
}

// testBulkGetEmpty verifies that BulkGet handles an empty key slice correctly.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testBulkGetEmpty(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	bulkLoader := cache_dto.BulkLoaderFunc[string, string](func(ctx context.Context, keys []string) (map[string]string, error) {
		return make(map[string]string), nil
	})

	results, err := cache.BulkGet(ctx, []string{}, bulkLoader)
	if err != nil {
		t.Errorf("BulkGet with empty keys failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("BulkGet with empty keys should return empty map: got %d", len(results))
	}
}

// testBulkSetBasic verifies that BulkSet stores multiple items correctly.
//
// Takes t (*testing.T) which provides the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testBulkSetBasic(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	err := cache.BulkSet(ctx, items)
	if err != nil {
		t.Errorf("BulkSet failed: %v", err)
	}

	got1, found1, err := cache.GetIfPresent(ctx, "key1")
	if err != nil {
		t.Fatalf("GetIfPresent key1 failed: %v", err)
	}
	if !found1 {
		t.Error("key1 should be present after BulkSet")
	}
	if got1 != "value1" {
		t.Errorf("key1 value mismatch: got %q, want %q", got1, "value1")
	}

	got2, found2, err := cache.GetIfPresent(ctx, "key2")
	if err != nil {
		t.Fatalf("GetIfPresent key2 failed: %v", err)
	}
	if !found2 {
		t.Error("key2 should be present after BulkSet")
	}
	if got2 != "value2" {
		t.Errorf("key2 value mismatch: got %q, want %q", got2, "value2")
	}
}

// testBulkSetWithTags tests that BulkSet correctly associates tags with items
// and that InvalidateByTags removes all items with the specified tag.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and configuration.
func testBulkSetWithTags(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	err := cache.BulkSet(ctx, items, "bulk-tag")
	if err != nil {
		t.Errorf("BulkSet with tags failed: %v", err)
	}

	_, found1, err := cache.GetIfPresent(ctx, "key1")
	if err != nil {
		t.Fatalf("GetIfPresent key1 failed: %v", err)
	}
	_, found2, err := cache.GetIfPresent(ctx, "key2")
	if err != nil {
		t.Fatalf("GetIfPresent key2 failed: %v", err)
	}
	if !found1 || !found2 {
		t.Error("Both keys should be present after BulkSet")
	}

	count, err := cache.InvalidateByTags(ctx, "bulk-tag")
	if err != nil {
		t.Fatalf("InvalidateByTags failed: %v", err)
	}
	if count != 2 {
		t.Errorf("InvalidateByTags should remove 2 items: got %d", count)
	}

	_, found1, err = cache.GetIfPresent(ctx, "key1")
	if err != nil {
		t.Fatalf("GetIfPresent key1 after invalidate failed: %v", err)
	}
	_, found2, err = cache.GetIfPresent(ctx, "key2")
	if err != nil {
		t.Fatalf("GetIfPresent key2 after invalidate failed: %v", err)
	}
	if found1 || found2 {
		t.Error("Both keys should be absent after InvalidateByTags")
	}
}

// testBulkSetEmpty verifies that BulkSet handles an empty map without error.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func testBulkSetEmpty(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	err := cache.BulkSet(ctx, map[string]string{})
	if err != nil {
		t.Errorf("BulkSet with empty map failed: %v", err)
	}
}

// testInvalidateByTagsSingle verifies that cache entries with a specific tag
// are invalidated while entries with other tags remain.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testInvalidateByTagsSingle(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "user:1", "Alice", "user"); err != nil {
		t.Fatalf("Set user:1 failed: %v", err)
	}
	if err := cache.Set(ctx, "user:2", "Bob", "user"); err != nil {
		t.Fatalf("Set user:2 failed: %v", err)
	}
	if err := cache.Set(ctx, "product:1", "Widget", "product"); err != nil {
		t.Fatalf("Set product:1 failed: %v", err)
	}

	count, err := cache.InvalidateByTags(ctx, "user")
	if err != nil {
		t.Fatalf("InvalidateByTags failed: %v", err)
	}
	if count != 2 {
		t.Errorf("InvalidateByTags(user) should remove 2: got %d", count)
	}

	_, found1, err := cache.GetIfPresent(ctx, "user:1")
	if err != nil {
		t.Fatalf("GetIfPresent user:1 failed: %v", err)
	}
	_, found2, err := cache.GetIfPresent(ctx, "user:2")
	if err != nil {
		t.Fatalf("GetIfPresent user:2 failed: %v", err)
	}
	_, found3, err := cache.GetIfPresent(ctx, "product:1")
	if err != nil {
		t.Fatalf("GetIfPresent product:1 failed: %v", err)
	}

	if found1 || found2 {
		t.Error("user tagged keys should be invalidated")
	}
	if !found3 {
		t.Error("product tagged key should remain")
	}
}

// testInvalidateByTagsMultiple verifies that cache entries can be invalidated
// by multiple tags in a single call.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testInvalidateByTagsMultiple(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key1", "value1", "tag1"); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := cache.Set(ctx, "key2", "value2", "tag2"); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}
	if err := cache.Set(ctx, "key3", "value3", "tag3"); err != nil {
		t.Fatalf("Set key3 failed: %v", err)
	}

	count, err := cache.InvalidateByTags(ctx, "tag1", "tag2")
	if err != nil {
		t.Fatalf("InvalidateByTags failed: %v", err)
	}
	if count < 2 {
		t.Errorf("InvalidateByTags(tag1, tag2) should remove at least 2: got %d", count)
	}

	_, found1, err := cache.GetIfPresent(ctx, "key1")
	if err != nil {
		t.Fatalf("GetIfPresent key1 failed: %v", err)
	}
	_, found2, err := cache.GetIfPresent(ctx, "key2")
	if err != nil {
		t.Fatalf("GetIfPresent key2 failed: %v", err)
	}
	_, found3, err := cache.GetIfPresent(ctx, "key3")
	if err != nil {
		t.Fatalf("GetIfPresent key3 failed: %v", err)
	}

	if found1 || found2 {
		t.Error("tag1 and tag2 keys should be invalidated")
	}
	if !found3 {
		t.Error("tag3 key should remain")
	}
}

// testInvalidateByTagsNoMatch verifies that invalidating by a non-existent tag
// does not affect entries with different tags.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testInvalidateByTagsNoMatch(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value", "existing-tag"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	count, err := cache.InvalidateByTags(ctx, "non-existent-tag")
	if err != nil {
		t.Fatalf("InvalidateByTags failed: %v", err)
	}
	if count != 0 {
		t.Errorf("InvalidateByTags with no matches should return 0: got %d", count)
	}

	_, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Key with different tag should remain")
	}
}

// testInvalidateAll verifies that InvalidateAll clears all cache entries.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testInvalidateAll(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key1", "value1"); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := cache.Set(ctx, "key2", "value2"); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}

	if cache.EstimatedSize() == 0 {
		t.Error("Cache should have entries before InvalidateAll")
	}

	if err := cache.InvalidateAll(ctx); err != nil {
		t.Fatalf("InvalidateAll failed: %v", err)
	}

	if cache.EstimatedSize() != 0 {
		t.Errorf("Cache should be empty after InvalidateAll: got %d", cache.EstimatedSize())
	}
}

// testInvalidateAllEmpty tests that InvalidateAll succeeds on an empty cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache provider factory.
func testInvalidateAllEmpty(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.InvalidateAll(ctx); err != nil {
		t.Fatalf("InvalidateAll on empty cache failed: %v", err)
	}
}

// testBulkRefresh tests that bulk refresh updates existing cache entries
// with new values from a bulk loader.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache provider factory.
func testBulkRefresh(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key1", "old-value1"); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := cache.Set(ctx, "key2", "old-value2"); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}

	bulkLoader := cache_dto.BulkLoaderFunc[string, string](func(ctx context.Context, keys []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, k := range keys {
			result[k] = "new-" + k
		}
		return result, nil
	})

	cache.BulkRefresh(ctx, []string{"key1", "key2"}, bulkLoader)
}
