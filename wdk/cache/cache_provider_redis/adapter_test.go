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

package cache_provider_redis_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_provider_redis"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, string) {
	t.Helper()

	mr := miniredis.RunT(t)

	t.Cleanup(func() {
		mr.Close()
	})

	return mr, mr.Addr()
}

func createRedisCache[K comparable, V any](t *testing.T, defaultTTL time.Duration) (*cache_provider_redis.RedisAdapter[K, V], *miniredis.Miniredis) {
	t.Helper()

	mr, addr := setupMiniredis(t)

	valueEncoder := cache_encoder_json.New[V]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

	config := cache_provider_redis.Config{
		Address:            addr,
		Password:           "",
		DB:                 0,
		DefaultTTL:         defaultTTL,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
	}

	provider, err := cache_provider_redis.NewRedisProvider(config)
	if err != nil {
		t.Fatalf("failed to create redis provider: %v", err)
	}

	cacheAny, err := provider.CreateNamespaceTyped("test", cache.Options[K, V]{})
	if err != nil {
		t.Fatalf("failed to create redis cache: %v", err)
	}

	c, ok := cacheAny.(*cache_provider_redis.RedisAdapter[K, V])
	if !ok {
		t.Fatalf("expected *cache_provider_redis.RedisAdapter[K, V], got %T", cacheAny)
	}

	t.Cleanup(func() {
		_ = c.Close(context.Background())
		_ = provider.Close()
	})

	return c, mr
}

func TestRedis_GetSetInvalidate(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	value, found, err := c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if !found {
		t.Fatal("expected key to be present")
	}
	assert.Equal(t, value, "value1", "value mismatch")

	require.NoError(t, c.Invalidate(ctx, "key1"))

	_, found, err = c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if found {
		t.Error("expected key to be absent after invalidation")
	}
}

func TestRedis_GetWithLoader(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	loadCount := 0
	loader := cache.LoaderFunc[string, string](func(ctx context.Context, key string) (string, error) {
		loadCount++
		return "loaded-" + key, nil
	})

	value, err := c.Get(ctx, "key1", loader)
	require.NoError(t, err, "Get with loader")
	assert.Equal(t, value, "loaded-key1", "loaded value")
	assert.Equal(t, loadCount, 1, "load count")

	value, err = c.Get(ctx, "key1", loader)
	require.NoError(t, err, "Get with loader (cached)")
	assert.Equal(t, value, "loaded-key1", "cached value")
	assert.Equal(t, loadCount, 1, "load count should not increase")
}

func TestRedis_TTLExpiration(t *testing.T) {
	t.Parallel()

	c, mr := createRedisCache[string, string](t, 100*time.Millisecond)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	value, found, err := c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if !found || value != "value1" {
		t.Fatal("key should be present initially")
	}

	mr.FastForward(200 * time.Millisecond)

	_, found, err = c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if found {
		t.Error("key should be expired after TTL")
	}
}

func TestRedis_TagBasedInvalidation(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "user:1", "Alice", "user", "active"))
	require.NoError(t, c.Set(ctx, "user:2", "Bob", "user", "inactive"))
	require.NoError(t, c.Set(ctx, "product:1", "Widget", "product"))

	_, found, err := c.GetIfPresent(ctx, "user:1")
	require.NoError(t, err)
	if !found {
		t.Error("user:1 should be present")
	}
	_, found, err = c.GetIfPresent(ctx, "user:2")
	require.NoError(t, err)
	if !found {
		t.Error("user:2 should be present")
	}
	_, found, err = c.GetIfPresent(ctx, "product:1")
	require.NoError(t, err)
	if !found {
		t.Error("product:1 should be present")
	}

	count, err := c.InvalidateByTags(ctx, "user")
	require.NoError(t, err)
	assert.Equal(t, count, 2, "invalidated count")

	_, found, err = c.GetIfPresent(ctx, "user:1")
	require.NoError(t, err)
	if found {
		t.Error("user:1 should be invalidated")
	}
	_, found, err = c.GetIfPresent(ctx, "user:2")
	require.NoError(t, err)
	if found {
		t.Error("user:2 should be invalidated")
	}
	_, found, err = c.GetIfPresent(ctx, "product:1")
	require.NoError(t, err)
	if !found {
		t.Error("product:1 should still be present")
	}
}

func TestRedis_InvalidateAll(t *testing.T) {
	t.Parallel()

	c, mr := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	for i := range 10 {
		require.NoError(t, c.Set(ctx, string(rune('a'+i)), "value"))
	}

	if len(mr.DB(0).Keys()) == 0 {
		t.Error("cache should have entries")
	}

	require.NoError(t, c.InvalidateAll(ctx))

	keys := mr.DB(0).Keys()
	if len(keys) != 0 {
		t.Errorf("cache should be empty, got %d keys", len(keys))
	}
}

func TestRedis_BulkGet(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))
	require.NoError(t, c.Set(ctx, "key2", "value2"))

	bulkLoader := cache.BulkLoaderFunc[string, string](func(ctx context.Context, keys []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, key := range keys {
			result[key] = "loaded-" + key
		}
		return result, nil
	})

	keys := []string{"key1", "key2", "key3", "key4"}
	results, err := c.BulkGet(ctx, keys, bulkLoader)
	require.NoError(t, err, "BulkGet")

	assert.Equal(t, results["key1"], "value1", "existing key1")
	assert.Equal(t, results["key2"], "value2", "existing key2")
	assert.Equal(t, results["key3"], "loaded-key3", "loaded key3")
	assert.Equal(t, results["key4"], "loaded-key4", "loaded key4")
}

func TestRedis_Compute(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, int](t, time.Hour)
	ctx := context.Background()

	value, found, err := c.Compute(ctx, "counter", func(oldValue int, exists bool) (int, cache.ComputeAction) {
		if !exists {
			return 1, cache.ComputeActionSet
		}
		return oldValue + 1, cache.ComputeActionSet
	})
	require.NoError(t, err)

	if !found {
		t.Error("expected Compute to return found=true after setting")
	}
	assert.Equal(t, value, 1, "initial value")

	value, found, err = c.Compute(ctx, "counter", func(oldValue int, exists bool) (int, cache.ComputeAction) {
		if !exists {
			return 1, cache.ComputeActionSet
		}
		return oldValue + 1, cache.ComputeActionSet
	})
	require.NoError(t, err)

	if !found {
		t.Error("expected Compute to return found=true")
	}
	assert.Equal(t, value, 2, "incremented value")
}

func TestRedis_ComputeIfAbsent(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	value, computed, err := c.ComputeIfAbsent(ctx, "key1", func() string {
		return "computed-value"
	})
	require.NoError(t, err)

	if !computed {
		t.Error("expected computation to occur for absent key")
	}
	assert.Equal(t, value, "computed-value", "computed value")

	value, _, err = c.ComputeIfAbsent(ctx, "key1", func() string {
		return "different-value"
	})
	require.NoError(t, err)

	assert.Equal(t, value, "computed-value", "original value should be retained")
}

func TestRedis_ComputeIfPresent(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	_, found, err := c.ComputeIfPresent(ctx, "key1", func(oldValue string) (string, cache.ComputeAction) {
		return "should-not-happen", cache.ComputeActionSet
	})
	require.NoError(t, err)

	if found {
		t.Error("expected no computation for absent key")
	}

	require.NoError(t, c.Set(ctx, "key1", "original"))

	value, found, err := c.ComputeIfPresent(ctx, "key1", func(oldValue string) (string, cache.ComputeAction) {
		return oldValue + "-modified", cache.ComputeActionSet
	})
	require.NoError(t, err)

	if !found {
		t.Error("expected computation for present key")
	}
	assert.Equal(t, value, "original-modified", "modified value")
}

func TestRedis_ComputeDelete(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	value, found, err := c.Compute(ctx, "key1", func(oldValue string, exists bool) (string, cache.ComputeAction) {
		return "", cache.ComputeActionDelete
	})
	require.NoError(t, err)

	if found {
		t.Error("expected found=false after deletion")
	}
	_ = value

	_, found, err = c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if found {
		t.Error("key should be deleted")
	}
}

func TestRedis_ComputeNoop(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "original"))

	value, found, err := c.Compute(ctx, "key1", func(oldValue string, exists bool) (string, cache.ComputeAction) {
		return "should-be-ignored", cache.ComputeActionNoop
	})
	require.NoError(t, err)

	if !found {
		t.Error("expected key to still be present")
	}
	assert.Equal(t, value, "original", "value should be unchanged")
}

func TestRedis_UpdateExistingValue(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	value, found, err := c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if !found || value != "value1" {
		t.Fatal("initial value not set correctly")
	}

	require.NoError(t, c.Set(ctx, "key1", "value2"))

	value, found, err = c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if !found || value != "value2" {
		t.Error("value not updated correctly")
	}
}

func TestRedis_EstimatedSize(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	assert.Equal(t, c.EstimatedSize(), 0, "initial size")

	require.NoError(t, c.Set(ctx, "key1", "value1"))
	require.NoError(t, c.Set(ctx, "key2", "value2"))

	size := c.EstimatedSize()
	if size < 2 {
		t.Errorf("expected size >= 2, got %d", size)
	}

	require.NoError(t, c.Invalidate(ctx, "key1"))

	size = c.EstimatedSize()
	if size < 1 {
		t.Errorf("expected size >= 1 after one invalidation, got %d", size)
	}
}

func TestRedis_Keys(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	expectedKeys := map[string]bool{
		"key1": true,
		"key2": true,
		"key3": true,
	}

	for key := range expectedKeys {
		require.NoError(t, c.Set(ctx, key, "value"))
	}

	foundKeys := make(map[string]bool)
	for key := range c.Keys() {
		foundKeys[key] = true
	}

	for expected := range expectedKeys {
		if !foundKeys[expected] {
			t.Errorf("expected key %q not found in iteration", expected)
		}
	}
}

func TestRedis_Values(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))
	require.NoError(t, c.Set(ctx, "key2", "value2"))
	require.NoError(t, c.Set(ctx, "key3", "value3"))

	count := 0
	for range c.Values() {
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 values, got %d", count)
	}
}

func TestRedis_All(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range expected {
		require.NoError(t, c.Set(ctx, k, v))
	}

	found := maps.Collect(c.All())

	for k, expectedV := range expected {
		if foundV, ok := found[k]; !ok {
			t.Errorf("key %q not found in iteration", k)
		} else if foundV != expectedV {
			t.Errorf("key %q: expected %q, got %q", k, expectedV, foundV)
		}
	}
}

func TestRedis_Concurrent(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, int](t, time.Hour)
	ctx := context.Background()

	const numGoroutines = 50
	const opsPerGoroutine = 100

	runConcurrent(t, numGoroutines, func(id int) {
		for i := range opsPerGoroutine {
			key := string(rune('a' + (id % 26)))
			_ = c.Set(ctx, key, id*opsPerGoroutine+i)
		}
	})

	runConcurrent(t, numGoroutines, func(id int) {
		for range opsPerGoroutine {
			key := string(rune('a' + (id % 26)))
			_, _, _ = c.GetIfPresent(ctx, key)
		}
	})

	runConcurrent(t, numGoroutines/10, func(id int) {
		_ = c.Set(ctx, string(rune('A'+id)), 999, "test-tag")
	})

	_, _ = c.InvalidateByTags(ctx, "test-tag")

}

func TestRedis_ConnectionHandling(t *testing.T) {
	t.Parallel()

	mr, addr := setupMiniredis(t)

	jsonEncoder := cache_encoder_json.New[any]()
	registry := cache.NewEncodingRegistry(jsonEncoder.(cache.AnyEncoder))

	config := cache_provider_redis.Config{
		Address:    addr,
		DefaultTTL: time.Hour,
		Registry:   registry,
	}

	provider, err := cache_provider_redis.NewRedisProvider(config)
	require.NoError(t, err, "provider creation")

	cacheAny, err := provider.CreateNamespaceTyped("test", cache.Options[string, string]{})
	require.NoError(t, err, "cache creation")

	c, ok := cacheAny.(cache.Cache[string, string])
	if !ok {
		t.Fatalf("expected cache.Cache[string, string], got %T", cacheAny)
	}

	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	value, found, err := c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if !found || value != "value1" {
		t.Error("should be able to get value after connection")
	}

	require.NoError(t, provider.Close())

	if !mr.Exists("test:key1") {
		t.Error("key should still exist in Redis after cache close")
	}
}

func TestRedis_MiniredisIntegration(t *testing.T) {
	t.Parallel()

	c, mr := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	if !mr.Exists("test:key1") {
		t.Error("key should exist in miniredis")
	}

	storedValue, err := mr.Get("test:key1")
	require.NoError(t, err, "miniredis Get")

	if storedValue == "" {
		t.Error("stored value should not be empty")
	}

	ttl := mr.TTL("test:key1")
	if ttl == 0 {
		t.Error("TTL should be set")
	}
}

func TestRedis_TaggedKeys(t *testing.T) {
	t.Parallel()

	c, mr := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "user:1", "Alice", "user", "active"))

	tagKey := "test:tag:user"
	if !mr.Exists(tagKey) {
		t.Errorf("tag set %q should exist", tagKey)
	}

	_, err := c.InvalidateByTags(ctx, "user")
	require.NoError(t, err)

	if mr.Exists("test:user:1") {
		t.Error("key should be invalidated")
	}
}

func TestRedis_Close(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	require.NoError(t, c.Close(ctx))

}

func TestRedis_EmptyTagList(t *testing.T) {
	t.Parallel()

	c, _ := createRedisCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	value, found, err := c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if !found || value != "value1" {
		t.Error("key without tags should work normally")
	}

	count, err := c.InvalidateByTags(ctx)
	require.NoError(t, err)
	assert.Equal(t, count, 0, "no tags means no invalidation")

	_, found, err = c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if !found {
		t.Error("key should still exist")
	}
}
