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
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

var (
	errLoaderFailed = errors.New("loader failed")
)

func newStringAdapter(t *testing.T, namespace string, allowUnsafeFLUSHDB bool) (*RedisClusterAdapter[string, string], *miniredis.Miniredis) {
	t.Helper()

	db := miniredis.RunT(t)
	encoder, ok := cache_encoder_json.New[string]().(cache.AnyEncoder)
	require.True(t, ok)
	provider, err := NewRedisClusterProvider(Config{
		Addrs:              []string{db.Addr()},
		Registry:           cache.NewEncodingRegistry(encoder),
		DefaultTTL:         time.Hour,
		AllowUnsafeFLUSHDB: allowUnsafeFLUSHDB,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, provider.Close()) })

	created, err := createNamespaceGeneric(provider, namespace, cache.Options[string, string]{})
	require.NoError(t, err)
	adapter, ok := created.(*RedisClusterAdapter[string, string])
	require.True(t, ok)
	return adapter, db
}

func TestRedisClusterAdapter_WriteExpiry(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		ttl      time.Duration
		fallback time.Duration
		want     time.Duration
		perEntry bool
	}{
		{name: "a positive TTL is used", ttl: time.Minute, fallback: time.Hour, want: time.Minute},
		{name: "a sub-millisecond TTL rounds up", ttl: time.Microsecond, want: time.Millisecond},
		{name: "NoExpiry sets none", ttl: cache.NoExpiry, fallback: time.Hour, want: 0},
		{name: "no TTL under per-entry expiry keeps the TTL", ttl: 0, fallback: time.Hour, perEntry: true, want: redis.KeepTTL},
		{name: "no TTL otherwise applies the default", ttl: 0, fallback: time.Hour, want: time.Hour},
		{name: "a sub-millisecond default rounds up", ttl: 0, fallback: time.Microsecond, want: time.Millisecond},
		{name: "a default of minus one sets none rather than keeping the TTL", ttl: 0, fallback: -1, want: 0},
		{name: "a negative default sets none", ttl: 0, fallback: -time.Second, want: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter := &RedisClusterAdapter[string, string]{perEntry: testCase.perEntry}

			assert.Equal(t, testCase.want, adapter.writeExpiry(testCase.ttl, testCase.fallback))
		})
	}
}

func TestRedisClusterAdapter_BulkSet(t *testing.T) {
	t.Parallel()

	adapter, db := newStringAdapter(t, "bulk", false)
	ctx := t.Context()

	require.NoError(t, adapter.BulkSet(ctx, nil))
	require.NoError(t, adapter.BulkSet(ctx, map[string]string{"a": "alpha", "b": "beta"}, "letters", "greek"))

	values, err := adapter.BulkGet(ctx, []string{"a", "b"}, cache.BulkLoaderFunc[string, string](func(context.Context, []string) (map[string]string, error) {
		return nil, errLoaderFailed
	}))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "alpha", "b": "beta"}, values)
	assert.Equal(t, time.Hour, db.TTL("bulk:a"))

	count, err := adapter.InvalidateByTags(ctx, "greek")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.False(t, db.Exists("bulk:a"))
}

func TestRedisClusterAdapter_BulkSetStopsOnACancelledContext(t *testing.T) {
	t.Parallel()

	adapter, db := newStringAdapter(t, "bulk", false)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := adapter.BulkSet(ctx, map[string]string{"a": "alpha"})

	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, db.Exists("bulk:a"))
}

func TestRedisClusterAdapter_InvalidateRemovesTagMembership(t *testing.T) {
	t.Parallel()

	adapter, _ := newStringAdapter(t, "tagged", false)
	ctx := t.Context()

	require.NoError(t, adapter.Set(ctx, "a", "alpha", "first"))
	require.NoError(t, adapter.Set(ctx, "b", "beta", "first"))
	require.NoError(t, adapter.Invalidate(ctx, "a"))

	count, err := adapter.InvalidateByTags(ctx, "first")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRedisClusterAdapter_Refresh(t *testing.T) {
	t.Parallel()

	adapter, _ := newStringAdapter(t, "refresh", false)
	ctx := t.Context()

	result := <-adapter.Refresh(ctx, "a", cache.LoaderFunc[string, string](func(context.Context, string) (string, error) {
		return "loaded", nil
	}))
	require.NoError(t, result.Err)
	assert.Equal(t, "loaded", result.Value)

	value, found, err := adapter.GetIfPresent(ctx, "a")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "loaded", value)

	result = <-adapter.Refresh(ctx, "b", cache.LoaderFunc[string, string](func(context.Context, string) (string, error) {
		return "", errLoaderFailed
	}))
	require.ErrorIs(t, result.Err, errLoaderFailed)
	_, found, err = adapter.GetIfPresent(ctx, "b")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRedisClusterAdapter_BulkRefresh(t *testing.T) {
	t.Parallel()

	adapter, db := newStringAdapter(t, "refresh", false)
	ctx := t.Context()
	failed := make(chan struct{})

	adapter.BulkRefresh(ctx, []string{"a", "b"}, cache.BulkLoaderFunc[string, string](func(_ context.Context, keys []string) (map[string]string, error) {
		loaded := make(map[string]string, len(keys))
		for _, key := range keys {
			loaded[key] = "value-" + key
		}
		return loaded, nil
	}))
	adapter.BulkRefresh(ctx, []string{"c"}, cache.BulkLoaderFunc[string, string](func(context.Context, []string) (map[string]string, error) {
		defer close(failed)
		return nil, errLoaderFailed
	}))

	require.Eventually(t, func() bool { return db.Exists("refresh:a") && db.Exists("refresh:b") }, 5*time.Second, 10*time.Millisecond)
	<-failed
	value, found, err := adapter.GetIfPresent(ctx, "b")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value-b", value)
	assert.False(t, db.Exists("refresh:c"))
}

func TestRedisClusterAdapter_InvalidateAllWithoutNamespace(t *testing.T) {
	t.Parallel()

	t.Run("flushes every node when unsafe flushing is allowed", func(t *testing.T) {
		t.Parallel()

		adapter, db := newStringAdapter(t, "flush", true)
		adapter.namespace = ""
		require.NoError(t, db.Set("other", "value"))

		require.NoError(t, adapter.InvalidateAll(t.Context()))

		assert.False(t, db.Exists("other"))
	})

	t.Run("refuses when unsafe flushing is not allowed", func(t *testing.T) {
		t.Parallel()

		adapter, db := newStringAdapter(t, "flush", false)
		adapter.namespace = ""
		require.NoError(t, db.Set("other", "value"))

		require.ErrorContains(t, adapter.InvalidateAll(t.Context()), "AllowUnsafeFLUSHDB is false")

		assert.True(t, db.Exists("other"))
	})
}

func TestRedisClusterProvider_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "redis-cluster", (&RedisClusterProvider{}).Name())
}
