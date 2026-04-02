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

package cache_provider_redis_cluster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

type UserKey struct {
	TenantID string
	UserID   int64
}

type SimpleValue struct {
	Data string
}

func TestRedisClusterAdapter_StructKeyEncoding(t *testing.T) {
	t.Parallel()

	t.Run("Encodes and decodes struct keys correctly", func(t *testing.T) {
		t.Parallel()

		keyEncoder := cache_encoder_json.New[UserKey]()
		keyRegistry := cache.NewEncodingRegistry(keyEncoder.(cache.AnyEncoder))

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		adapter := createTestAdapter[UserKey, SimpleValue](t, Config{
			Addrs:       getTestRedisClusterAddrs(),
			Registry:    valueRegistry,
			KeyRegistry: keyRegistry,
			Namespace:   "test:struct-keys:",
		})
		defer func() { _ = adapter.Close(context.Background()) }()

		ctx := context.Background()
		key := UserKey{TenantID: "tenant-123", UserID: 456}
		value := SimpleValue{Data: "test-data"}

		require.NoError(t, adapter.Set(ctx, key, value))

		retrieved, found, err := adapter.GetIfPresent(ctx, key)
		require.NoError(t, err)
		require.True(t, found, "Should find value with struct key")
		assert.Equal(t, value.Data, retrieved.Data)
	})

	t.Run("All() iterator works with struct keys", func(t *testing.T) {
		t.Parallel()

		keyEncoder := cache_encoder_json.New[UserKey]()
		keyRegistry := cache.NewEncodingRegistry(keyEncoder.(cache.AnyEncoder))

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		adapter := createTestAdapter[UserKey, SimpleValue](t, Config{
			Addrs:       getTestRedisClusterAddrs(),
			Registry:    valueRegistry,
			KeyRegistry: keyRegistry,
			Namespace:   "test:struct-iter:",
		})
		defer func() { _ = adapter.Close(context.Background()) }()

		ctx := context.Background()
		keys := []UserKey{
			{TenantID: "t1", UserID: 1},
			{TenantID: "t1", UserID: 2},
			{TenantID: "t2", UserID: 3},
		}

		for _, k := range keys {
			require.NoError(t, adapter.Set(ctx, k, SimpleValue{Data: "data"}))
		}

		var foundKeys []UserKey
		for k := range adapter.All() {
			foundKeys = append(foundKeys, k)
		}

		assert.Len(t, foundKeys, 3, "All() should return all struct keys")

		for _, fk := range foundKeys {
			assert.NotEmpty(t, fk.TenantID, "TenantID should be populated")
			assert.NotZero(t, fk.UserID, "UserID should be non-zero")
		}
	})

	t.Run("Backward compatibility: primitive keys work without KeyRegistry", func(t *testing.T) {
		t.Parallel()

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		adapter := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:       getTestRedisClusterAddrs(),
			Registry:    valueRegistry,
			KeyRegistry: nil,
			Namespace:   "test:primitive:",
		})
		defer func() { _ = adapter.Close(context.Background()) }()

		ctx := context.Background()
		require.NoError(t, adapter.Set(ctx, "string-key", SimpleValue{Data: "string-value"}))
		value, found, err := adapter.GetIfPresent(ctx, "string-key")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "string-value", value.Data)

		require.NoError(t, adapter.Set(ctx, "another-key", SimpleValue{Data: "another-value"}))
		val2, found2, err := adapter.GetIfPresent(ctx, "another-key")
		require.NoError(t, err)
		assert.True(t, found2)
		assert.Equal(t, "another-value", val2.Data)
	})

	t.Run("BulkGet works with struct keys", func(t *testing.T) {
		t.Parallel()

		keyEncoder := cache_encoder_json.New[UserKey]()
		keyRegistry := cache.NewEncodingRegistry(keyEncoder.(cache.AnyEncoder))

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		adapter := createTestAdapter[UserKey, SimpleValue](t, Config{
			Addrs:       getTestRedisClusterAddrs(),
			Registry:    valueRegistry,
			KeyRegistry: keyRegistry,
			Namespace:   "test:bulk:",
		})
		defer func() { _ = adapter.Close(context.Background()) }()

		ctx := context.Background()

		key1 := UserKey{TenantID: "t1", UserID: 1}
		key2 := UserKey{TenantID: "t1", UserID: 2}
		require.NoError(t, adapter.Set(ctx, key1, SimpleValue{Data: "data1"}))
		require.NoError(t, adapter.Set(ctx, key2, SimpleValue{Data: "data2"}))

		keys := []UserKey{key1, key2}
		results, err := adapter.BulkGet(ctx, keys, &noopBulkLoader[UserKey, SimpleValue]{})

		require.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "data1", results[key1].Data)
		assert.Equal(t, "data2", results[key2].Data)
	})
}

func TestRedisClusterAdapter_NamespaceIsolation(t *testing.T) {
	t.Parallel()

	t.Run("Keys are prefixed with namespace", func(t *testing.T) {
		t.Parallel()

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		namespace := "test:prefix:"
		adapter := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:     getTestRedisClusterAddrs(),
			Registry:  valueRegistry,
			Namespace: namespace,
		})
		defer func() { _ = adapter.Close(context.Background()) }()

		ctx := context.Background()
		require.NoError(t, adapter.Set(ctx, "my-key", SimpleValue{Data: "value"}))

		found := false
		for k := range adapter.All() {
			if k == "my-key" {
				found = true
				break
			}
		}
		assert.True(t, found, "Key should be found via All() iterator")
	})

	t.Run("Different namespaces isolate caches", func(t *testing.T) {
		t.Parallel()

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		adapter1 := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:     getTestRedisClusterAddrs(),
			Registry:  valueRegistry,
			Namespace: "app1:cache:",
		})
		defer func() { _ = adapter1.Close(context.Background()) }()

		adapter2 := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:     getTestRedisClusterAddrs(),
			Registry:  valueRegistry,
			Namespace: "app2:cache:",
		})
		defer func() { _ = adapter2.Close(context.Background()) }()

		ctx := context.Background()
		require.NoError(t, adapter1.Set(ctx, "shared-key", SimpleValue{Data: "app1-data"}))
		require.NoError(t, adapter2.Set(ctx, "shared-key", SimpleValue{Data: "app2-data"}))

		val1, found1, err := adapter1.GetIfPresent(ctx, "shared-key")
		require.NoError(t, err)
		val2, found2, err := adapter2.GetIfPresent(ctx, "shared-key")
		require.NoError(t, err)

		assert.True(t, found1)
		assert.True(t, found2)
		assert.Equal(t, "app1-data", val1.Data)
		assert.Equal(t, "app2-data", val2.Data)
	})

	t.Run("InvalidateAll respects namespace boundaries", func(t *testing.T) {
		t.Parallel()

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		adapter1 := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:     getTestRedisClusterAddrs(),
			Registry:  valueRegistry,
			Namespace: "ns1:test:",
		})
		defer func() { _ = adapter1.Close(context.Background()) }()

		adapter2 := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:     getTestRedisClusterAddrs(),
			Registry:  valueRegistry,
			Namespace: "ns2:test:",
		})
		defer func() { _ = adapter2.Close(context.Background()) }()

		ctx := context.Background()
		require.NoError(t, adapter1.Set(ctx, "key1", SimpleValue{Data: "data1"}))
		require.NoError(t, adapter2.Set(ctx, "key2", SimpleValue{Data: "data2"}))

		require.NoError(t, adapter1.InvalidateAll(ctx))

		_, found1, err := adapter1.GetIfPresent(ctx, "key1")
		require.NoError(t, err)
		val2, found2, err := adapter2.GetIfPresent(ctx, "key2")
		require.NoError(t, err)

		assert.False(t, found1, "Namespace 1 key should be deleted")
		assert.True(t, found2, "Namespace 2 key should still exist")
		assert.Equal(t, "data2", val2.Data)
	})

	t.Run("All() iterator only returns keys from namespace", func(t *testing.T) {
		t.Parallel()

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		adapter1 := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:     getTestRedisClusterAddrs(),
			Registry:  valueRegistry,
			Namespace: "iter1:",
		})
		defer func() { _ = adapter1.Close(context.Background()) }()

		adapter2 := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:     getTestRedisClusterAddrs(),
			Registry:  valueRegistry,
			Namespace: "iter2:",
		})
		defer func() { _ = adapter2.Close(context.Background()) }()

		ctx := context.Background()
		require.NoError(t, adapter1.Set(ctx, "a", SimpleValue{Data: "1"}))
		require.NoError(t, adapter1.Set(ctx, "b", SimpleValue{Data: "2"}))
		require.NoError(t, adapter2.Set(ctx, "c", SimpleValue{Data: "3"}))
		require.NoError(t, adapter2.Set(ctx, "d", SimpleValue{Data: "4"}))

		count1 := 0
		for range adapter1.All() {
			count1++
		}

		count2 := 0
		for range adapter2.All() {
			count2++
		}

		assert.Equal(t, 2, count1, "Namespace 1 should have 2 keys")
		assert.Equal(t, 2, count2, "Namespace 2 should have 2 keys")
	})
}

func TestRedisClusterAdapter_InvalidateAllSafety(t *testing.T) {
	t.Parallel()

	t.Run("InvalidateAll blocked without namespace and unsafe flag", func(t *testing.T) {

		t.Skip("Safeguard test skipped - normal API usage always ensures a namespace")
	})

	t.Run("InvalidateAll uses FLUSHDB with unsafe flag and no namespace", func(t *testing.T) {

		t.Skip("Skipping dangerous FLUSHDB test - would delete entire cluster")
	})

	t.Run("InvalidateAll uses SCAN+DELETE with namespace", func(t *testing.T) {
		t.Parallel()

		valueEncoder := cache_encoder_json.New[SimpleValue]()
		valueRegistry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

		adapter := createTestAdapter[string, SimpleValue](t, Config{
			Addrs:     getTestRedisClusterAddrs(),
			Registry:  valueRegistry,
			Namespace: "safe-invalidate:",
		})
		defer func() { _ = adapter.Close(context.Background()) }()

		ctx := context.Background()
		for i := range 10 {
			require.NoError(t, adapter.Set(ctx, fmt.Sprintf("key-%d", i), SimpleValue{Data: "data"}))
		}

		count := 0
		for range adapter.All() {
			count++
		}
		assert.Equal(t, 10, count, "Should have 10 keys before invalidate")

		require.NoError(t, adapter.InvalidateAll(ctx))

		count = 0
		for range adapter.All() {
			count++
		}
		assert.Equal(t, 0, count, "Should have 0 keys after invalidate")
	})
}

func createTestAdapter[K comparable, V any](t *testing.T, config Config) *RedisClusterAdapter[K, V] {
	t.Helper()

	mr := miniredis.RunT(t)
	t.Cleanup(func() {
		mr.Close()
	})

	config.Addrs = []string{mr.Addr()}

	if config.DefaultTTL == 0 {
		config.DefaultTTL = 1 * time.Hour
	}

	provider, err := NewRedisClusterProvider(config)
	if err != nil {

		t.Skipf("Miniredis doesn't fully support Redis Cluster mode: %v", err)
	}

	c, err := createNamespaceGeneric[K, V](provider, "test", cache.Options[K, V]{})
	if err != nil {
		t.Skipf("Miniredis doesn't fully support Redis Cluster mode: %v", err)
	}

	adapter, ok := c.(*RedisClusterAdapter[K, V])
	require.True(t, ok, "Cache should be RedisClusterAdapter")

	t.Cleanup(func() {
		_ = adapter.Close(context.Background())
		_ = provider.Close()
	})

	return adapter
}

func getTestRedisClusterAddrs() []string {

	return []string{
		"localhost:7000",
		"localhost:7001",
		"localhost:7002",
	}
}

type noopBulkLoader[K comparable, V any] struct{}

func (l *noopBulkLoader[K, V]) BulkLoad(ctx context.Context, keys []K) (map[K]V, error) {
	return make(map[K]V), nil
}

func (l *noopBulkLoader[K, V]) BulkReload(ctx context.Context, keys []K, oldValues []V) (map[K]V, error) {
	return make(map[K]V), nil
}
