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

package cache_provider_valkey_cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

func TestValkeyClusterAdapter_WriteExpiry(t *testing.T) {
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
		{name: "no TTL under per-entry expiry keeps the TTL", ttl: 0, fallback: time.Hour, perEntry: true, want: keepTTL},
		{name: "no TTL otherwise applies the default", ttl: 0, fallback: time.Hour, want: time.Hour},
		{name: "a negative default sets none", ttl: 0, fallback: -time.Second, want: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter := &ValkeyClusterAdapter[string, string]{perEntry: testCase.perEntry}

			assert.Equal(t, testCase.want, adapter.writeExpiry(testCase.ttl, testCase.fallback))
		})
	}
}

func TestValkeyClusterAdapter_ApplyJSONExpiryRoundsUpSubMillisecond(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, server := newMiniredisClusterAdapter(t)
	require.NoError(t, server.Set("document", "{}"))

	require.NoError(t, adapter.applyJSONExpiry(ctx, "document", 500*time.Microsecond))

	assert.True(t, server.Exists("document"), "PEXPIRE 0 deleted the document")
	assert.Equal(t, time.Millisecond, server.TTL("document"))
}

func TestValkeyClusterAdapter_InvalidateAllSkipsReplicas(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		unsafe bool
	}{
		{name: "namespace invalidation", unsafe: false},
		{name: "unsafe flush without a namespace", unsafe: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			adapter, primary := newMiniredisClusterAdapter(t)
			for index, key := range []string{"a", "b", "c"} {
				require.NoError(t, adapter.Set(ctx, key, float64(index)))
			}
			answerAsNode(t, primary, "master", 0, 0, 0)
			replica := newReplicaOf(t, adapter, primary)
			replicaCommands := answerAsNode(t, replica, "slave", 0, 0, 0)
			if testCase.unsafe {
				adapter.namespace = ""
				adapter.allowUnsafeFLUSHDB = true
			}

			require.NoError(t, adapter.InvalidateAll(ctx))

			assert.Empty(t, primary.Keys())
			assert.Len(t, replica.Keys(), 3, "a replica is cleared by replication, not directly")
			assert.Equal(t, []string{"ROLE"}, replicaCommands.seen(), "a replica must only be asked its role")
		})
	}
}

func TestValkeyClusterAdapter_SetWithTTLKeepsSubSecondTTL(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, server := newMiniredisClusterAdapter(t)

	require.NoError(t, adapter.SetWithTTL(ctx, "k", 1, 500*time.Millisecond))

	keyString, err := adapter.encodeKey("k")
	require.NoError(t, err)
	assert.Equal(t, 500*time.Millisecond, server.TTL(keyString), "a sub-second TTL must be sent in milliseconds")
}

var (
	errLoaderFailed = errors.New("loader failed")
)

func newMiniredisClusterAdapter(t *testing.T) (*ValkeyClusterAdapter[string, float64], *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	encoder, ok := cache_encoder_json.New[float64]().(cache.AnyEncoder)
	require.True(t, ok)
	provider, err := NewValkeyClusterProvider(Config{InitAddress: []string{server.Addr()}, Registry: cache.NewEncodingRegistry(encoder)})
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	created, err := createNamespaceGeneric(provider, "test", cache.Options[string, float64]{})
	require.NoError(t, err)
	adapter, ok := created.(*ValkeyClusterAdapter[string, float64])
	require.True(t, ok)
	return adapter, server
}

func TestValkeyClusterAdapter_GetLoadsAndStoresMisses(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, _ := newMiniredisClusterAdapter(t)
	loads := 0
	loader := cache.LoaderFunc[string, float64](func(context.Context, string) (float64, error) {
		loads++
		return 7, nil
	})

	value, err := adapter.Get(ctx, "k", loader)
	require.NoError(t, err)
	assert.InDelta(t, 7.0, value, 0)

	value, err = adapter.Get(ctx, "k", loader)
	require.NoError(t, err)
	assert.InDelta(t, 7.0, value, 0)
	assert.Equal(t, 1, loads)

	_, err = adapter.Get(ctx, "other", cache.LoaderFunc[string, float64](func(context.Context, string) (float64, error) {
		return 0, errLoaderFailed
	}))
	require.ErrorIs(t, err, errLoaderFailed)
}

func TestValkeyClusterAdapter_BulkSetAndBulkGet(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, server := newMiniredisClusterAdapter(t)

	require.NoError(t, adapter.BulkSet(ctx, nil))
	require.NoError(t, adapter.BulkSet(ctx, map[string]float64{"a": 1, "b": 2}, "numbers"))
	require.NoError(t, server.Set("test:corrupt", "not json"))

	var requested []string
	values, err := adapter.BulkGet(ctx, []string{"a", "b", "c", "corrupt"}, cache.BulkLoaderFunc[string, float64](func(_ context.Context, keys []string) (map[string]float64, error) {
		requested = keys
		return map[string]float64{"c": 3}, nil
	}))
	require.NoError(t, err)

	assert.Equal(t, map[string]float64{"a": 1, "b": 2, "c": 3}, values)
	assert.ElementsMatch(t, []string{"c", "corrupt"}, requested)
	assert.True(t, server.Exists("test:c"), "loaded values are written back")

	empty, err := adapter.BulkGet(ctx, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	count, err := adapter.InvalidateByTags(ctx, "numbers")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.False(t, server.Exists("test:a"))
}

func TestValkeyClusterAdapter_BulkGetReportsLoaderFailure(t *testing.T) {
	t.Parallel()

	adapter, _ := newMiniredisClusterAdapter(t)
	require.NoError(t, adapter.Set(t.Context(), "a", 1))

	values, err := adapter.BulkGet(t.Context(), []string{"a", "b"}, cache.BulkLoaderFunc[string, float64](func(context.Context, []string) (map[string]float64, error) {
		return nil, errLoaderFailed
	}))

	require.ErrorIs(t, err, errLoaderFailed)
	assert.Equal(t, map[string]float64{"a": 1}, values)
}

func TestValkeyClusterAdapter_BulkSetReportsPipelineFailure(t *testing.T) {
	t.Parallel()

	adapter, server := newMiniredisClusterAdapter(t)
	require.NoError(t, server.Set(tagPrefix+clusterHashTag("test:numbers"), "not a set"))

	err := adapter.BulkSet(t.Context(), map[string]float64{"a": 1}, "numbers")

	require.ErrorContains(t, err, "bulk set pipeline failed")
}

func TestValkeyClusterAdapter_Refresh(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, server := newMiniredisClusterAdapter(t)

	result := <-adapter.Refresh(ctx, "a", cache.LoaderFunc[string, float64](func(context.Context, string) (float64, error) {
		return 5, nil
	}))
	require.NoError(t, result.Err)
	assert.InDelta(t, 5.0, result.Value, 0)
	assert.True(t, server.Exists("test:a"))

	result = <-adapter.Refresh(ctx, "b", cache.LoaderFunc[string, float64](func(context.Context, string) (float64, error) {
		return 0, errLoaderFailed
	}))
	require.ErrorIs(t, result.Err, errLoaderFailed)
	assert.False(t, server.Exists("test:b"))
}

func TestValkeyClusterAdapter_BulkRefresh(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, server := newMiniredisClusterAdapter(t)
	failed := make(chan struct{})

	adapter.BulkRefresh(ctx, []string{"a", "b"}, cache.BulkLoaderFunc[string, float64](func(_ context.Context, keys []string) (map[string]float64, error) {
		loaded := make(map[string]float64, len(keys))
		for index, key := range keys {
			loaded[key] = float64(index)
		}
		return loaded, nil
	}))
	adapter.BulkRefresh(ctx, []string{"c"}, cache.BulkLoaderFunc[string, float64](func(context.Context, []string) (map[string]float64, error) {
		defer close(failed)
		return nil, errLoaderFailed
	}))

	require.Eventually(t, func() bool { return server.Exists("test:a") && server.Exists("test:b") }, 5*time.Second, 10*time.Millisecond)
	<-failed
	assert.False(t, server.Exists("test:c"))
}

func TestValkeyClusterAdapter_InvalidateByTagsHonoursCancellation(t *testing.T) {
	t.Parallel()

	adapter, _ := newMiniredisClusterAdapter(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := adapter.InvalidateByTags(ctx, "numbers")

	require.ErrorIs(t, err, context.Canceled)
}
