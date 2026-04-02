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

package cache_provider_redis_cluster_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_provider_redis_cluster"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

func createRedisClusterCache[K comparable, V any](t *testing.T, defaultTTL time.Duration) (*cache_provider_redis_cluster.RedisClusterAdapter[K, V], *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	t.Cleanup(func() {
		mr.Close()
	})

	valueEncoder := cache_encoder_json.New[V]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

	config := cache_provider_redis_cluster.Config{
		Addrs:              []string{mr.Addr()},
		Password:           "",
		DefaultTTL:         defaultTTL,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
	}

	provider, err := cache_provider_redis_cluster.NewRedisClusterProvider(config)
	if err != nil {

		t.Skip("Miniredis doesn't fully support Redis Cluster mode - skipping cluster adapter test")
	}

	cacheAny, err := provider.CreateNamespaceTyped("test", cache.Options[K, V]{})
	if err != nil {
		t.Skip("Miniredis doesn't fully support Redis Cluster mode - skipping cluster adapter test")
	}

	c, ok := cacheAny.(*cache_provider_redis_cluster.RedisClusterAdapter[K, V])
	if !ok {
		t.Fatalf("expected *cache_provider_redis_cluster.RedisClusterAdapter[K, V], got %T", cacheAny)
	}

	t.Cleanup(func() {
		_ = c.Close(context.Background())
		_ = provider.Close()
	})

	return c, mr
}

func TestRedisCluster_GetSetInvalidate(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_GetWithLoader(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_TTLExpiration(t *testing.T) {
	t.Parallel()

	c, mr := createRedisClusterCache[string, string](t, 100*time.Millisecond)
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

func TestRedisCluster_TagBasedInvalidation(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_TagMemoryLeak(t *testing.T) {
	t.Parallel()

	c, mr := createRedisClusterCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "user:1", "Alice", "user", "active"))

	tagKey := "tag:{user}"
	keyTagsKey := "keytags:user:1"

	if !mr.Exists(tagKey) {
		t.Skip("Miniredis may not show cluster hash tag keys correctly - skipping")
	}

	require.NoError(t, c.Invalidate(ctx, "user:1"))

	if mr.Exists("user:1") {
		t.Error("key should be deleted")
	}

	if mr.Exists(keyTagsKey) {
		t.Error("reverse tag mapping should be cleaned up to prevent memory leak")
	}
}

func TestRedisCluster_InvalidateAll(t *testing.T) {
	t.Parallel()

	c, mr := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_BulkGet(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_Compute(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, int](t, time.Hour)
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

func TestRedisCluster_ComputeIfAbsent(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_ComputeIfPresent(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_EstimatedSize(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_All(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
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

func TestRedisCluster_Concurrent(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, int](t, time.Hour)
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

func TestRedisCluster_ConfigurableTimeouts(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

	config := cache_provider_redis_cluster.Config{
		Addrs:                  []string{mr.Addr()},
		DefaultTTL:             time.Hour,
		Registry:               registry,
		OperationTimeout:       500 * time.Millisecond,
		AtomicOperationTimeout: 2 * time.Second,
		BulkOperationTimeout:   5 * time.Second,
		FlushTimeout:           10 * time.Second,
		MaxComputeRetries:      5,
	}

	provider, err := cache_provider_redis_cluster.NewRedisClusterProvider(config)
	if err != nil {
		t.Skip("Miniredis doesn't fully support cluster mode")
	}

	cacheAny, err := provider.CreateNamespaceTyped("test", cache.Options[string, string]{})
	if err != nil {
		t.Skip("Miniredis doesn't fully support cluster mode")
	}

	c, ok := cacheAny.(cache.Cache[string, string])
	if !ok {
		t.Fatalf("expected cache.Cache[string, string], got %T", cacheAny)
	}

	ctx := context.Background()
	defer func() {
		_ = c.Close(ctx)
		_ = provider.Close()
	}()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	value, found, err := c.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	if !found || value != "value1" {
		t.Error("cache should work with custom timeouts")
	}
}

func TestRedisCluster_Close(t *testing.T) {
	t.Parallel()

	c, _ := createRedisClusterCache[string, string](t, time.Hour)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "key1", "value1"))

	require.NoError(t, c.Close(ctx))

}
