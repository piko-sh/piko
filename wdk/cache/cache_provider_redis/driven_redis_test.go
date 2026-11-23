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
package cache_provider_redis

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

func TestRedisVectorSearchReturnsOnlyTheRequestedPage(t *testing.T) {
	db := miniredis.RunT(t)
	commands := make(chan []string, 1)
	require.NoError(t, db.Server().Register("FT.SEARCH", func(peer *server.Peer, _ string, args []string) {
		commands <- slices.Clone(args)
		peer.WriteLen(1)
		peer.WriteInt(0)
	}))
	client := redis.NewClient(&redis.Options{Addr: db.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	adapter := &RedisAdapter[string, string]{client: client, indexName: "docs"}
	_, err := adapter.vectorSearchWithRediSearch(t.Context(), "", &cache.SearchOptions{Vector: []float32{1, 0}, VectorField: "vec", TopK: 30, Offset: 3, Limit: 2})
	require.NoError(t, err)
	arguments := <-commands
	require.Contains(t, arguments, "(*)=>[KNN 30 @vec $vec]")
	start := slices.Index(arguments, "LIMIT")
	require.NotEqual(t, -1, start)
	require.Equal(t, []string{"3", "2"}, arguments[start+1:start+3])
}

func TestRedisAdapter_WriteExpiry(t *testing.T) {
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

			adapter := &RedisAdapter[string, string]{perEntry: testCase.perEntry}

			assert.Equal(t, testCase.want, adapter.writeExpiry(testCase.ttl, testCase.fallback))
		})
	}
}

func TestExpiryFromPTTL(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	testCases := []struct {
		name string
		ttl  time.Duration
		want int64
	}{
		{name: "no TTL", ttl: -1, want: 0},
		{name: "missing key", ttl: -2, want: 0},
		{name: "under one millisecond left", ttl: 0, want: now.Add(time.Millisecond).UnixNano()},
		{name: "a TTL", ttl: 1500 * time.Millisecond, want: now.Add(1500 * time.Millisecond).UnixNano()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, expiryFromPTTL(now, testCase.ttl))
		})
	}
}

type fixedExpiry[K comparable, V any] struct {
	ttl time.Duration
}

func (f fixedExpiry[K, V]) ExpireAfterCreate(cache.Entry[K, V]) time.Duration { return f.ttl }

func (f fixedExpiry[K, V]) ExpireAfterUpdate(cache.Entry[K, V], V) time.Duration { return f.ttl }

func (f fixedExpiry[K, V]) ExpireAfterRead(cache.Entry[K, V]) time.Duration { return f.ttl }

func newStringAdapter(t *testing.T, options cache.Options[string, string]) (*RedisAdapter[string, string], *miniredis.Miniredis) {
	t.Helper()

	db := miniredis.RunT(t)
	encoder, ok := cache_encoder_json.New[string]().(cache.AnyEncoder)
	require.True(t, ok)
	provider, err := NewRedisProvider(Config{Address: db.Addr(), Registry: cache.NewEncodingRegistry(encoder), DefaultTTL: time.Hour})
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	created, err := createNamespaceGeneric(provider, "plain", options)
	require.NoError(t, err)
	adapter, ok := created.(*RedisAdapter[string, string])
	require.True(t, ok)
	return adapter, db
}

func TestRedisAdapter_BulkSetWritesEveryItemWithTags(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, db := newStringAdapter(t, cache.Options[string, string]{})

	require.NoError(t, adapter.BulkSet(ctx, nil))
	require.NoError(t, adapter.BulkSet(ctx, map[string]string{"a": "1", "b": "2"}, "group", "other"))

	assert.Equal(t, time.Hour, db.TTL("plain:a"))
	members, err := db.Members("plain:" + tagPrefix + "group")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"plain:a", "plain:b"}, members)
	keyTags, err := db.Members(keyTagsPrefix + "plain:a")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"group", "other"}, keyTags)

	removed, err := adapter.InvalidateByTags(ctx, "group")
	require.NoError(t, err)
	assert.Equal(t, 2, removed)
	assert.False(t, db.Exists("plain:a"))
}

func TestRedisAdapter_BulkSetUsesTheExpiryCalculator(t *testing.T) {
	t.Parallel()

	adapter, db := newStringAdapter(t, cache.Options[string, string]{ExpiryCalculator: fixedExpiry[string, string]{ttl: time.Minute}})

	require.NoError(t, adapter.BulkSet(t.Context(), map[string]string{"a": "1"}))

	assert.Equal(t, time.Minute, db.TTL("plain:a"))
}

func TestRedisAdapter_BulkSetSkipsValuesWithoutAnEncoder(t *testing.T) {
	t.Parallel()

	db := miniredis.RunT(t)
	encoder, ok := cache_encoder_json.New[string]().(cache.AnyEncoder)
	require.True(t, ok)
	provider, err := NewRedisProvider(Config{Address: db.Addr(), Registry: cache.NewEncodingRegistry(encoder)})
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })
	created, err := createNamespaceGeneric(provider, "ints", cache.Options[string, int]{})
	require.NoError(t, err)
	adapter, ok := created.(*RedisAdapter[string, int])
	require.True(t, ok)

	require.NoError(t, adapter.BulkSet(t.Context(), map[string]int{"a": 1}))

	assert.False(t, db.Exists("ints:a"))
}

func TestRedisAdapter_RefreshLoadsAndStores(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, _ := newStringAdapter(t, cache.Options[string, string]{})

	result := <-adapter.Refresh(ctx, "a", cache.LoaderFunc[string, string](func(context.Context, string) (string, error) {
		return "fresh", nil
	}))
	require.NoError(t, result.Err)
	assert.Equal(t, "fresh", result.Value)

	value, found, err := adapter.GetIfPresent(ctx, "a")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "fresh", value)

	failure := <-adapter.Refresh(ctx, "b", cache.LoaderFunc[string, string](func(context.Context, string) (string, error) {
		return "", errors.New("load failed")
	}))
	require.ErrorContains(t, failure.Err, "load failed")
	_, found, err = adapter.GetIfPresent(ctx, "b")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRedisAdapter_BulkRefreshStoresLoadedValues(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, db := newStringAdapter(t, cache.Options[string, string]{})

	adapter.BulkRefresh(ctx, []string{"a", "b"}, cache.BulkLoaderFunc[string, string](func(_ context.Context, keys []string) (map[string]string, error) {
		loaded := make(map[string]string, len(keys))
		for _, key := range keys {
			loaded[key] = "v-" + key
		}
		return loaded, nil
	}))

	require.Eventually(t, func() bool {
		return db.Exists("plain:a") && db.Exists("plain:b")
	}, 5*time.Second, 5*time.Millisecond)

	failed := make(chan struct{})
	adapter.BulkRefresh(ctx, []string{"c"}, cache.BulkLoaderFunc[string, string](func(context.Context, []string) (map[string]string, error) {
		defer close(failed)
		return nil, errors.New("load failed")
	}))
	<-failed
	assert.False(t, db.Exists("plain:c"))
}

func TestRedisAdapter_GetEntry(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, _ := newStringAdapter(t, cache.Options[string, string]{})
	require.NoError(t, adapter.SetWithTTL(ctx, "a", "1", time.Minute))

	entry, found, err := adapter.GetEntry(ctx, "a")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "a", entry.Key)
	assert.Equal(t, "1", entry.Value)
	assert.NotZero(t, entry.ExpiresAtNano)

	_, found, err = adapter.GetEntry(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRedisAdapter_ServerStatistics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		stats        string
		memory       string
		maxmemory    string
		wantHits     uint64
		wantMisses   uint64
		wantUsed     uint64
		wantMaximum  uint64
		failCommands bool
	}{
		{
			name:        "parsed",
			stats:       "# Stats\r\nkeyspace_hits:5\r\nkeyspace_misses:3\r\n",
			memory:      "# Memory\r\nused_memory:2048\r\n",
			maxmemory:   "1024",
			wantHits:    5,
			wantMisses:  3,
			wantUsed:    2048,
			wantMaximum: 1024,
		},
		{
			name:      "unparseable values read as zero",
			stats:     "keyspace_hits:many\r\nkeyspace_misses:few\r\n",
			memory:    "used_memory:lots\r\n",
			maxmemory: "unlimited",
		},
		{
			name:      "missing values read as zero",
			stats:     "# Stats\r\n",
			memory:    "# Memory\r\n",
			maxmemory: "",
		},
		{
			name:         "server errors read as zero",
			failCommands: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, db := newStringAdapter(t, cache.Options[string, string]{})
			db.Server().SetPreHook(func(peer *server.Peer, cmd string, args ...string) bool {
				if cmd != "INFO" && cmd != "CONFIG" {
					return false
				}
				if testCase.failCommands {
					peer.WriteError("ERR unavailable")
					return true
				}
				switch {
				case cmd == "CONFIG" && testCase.maxmemory == "":
					peer.WriteMapLen(0)
				case cmd == "CONFIG":
					peer.WriteMapLen(1)
					peer.WriteBulk("maxmemory")
					peer.WriteBulk(testCase.maxmemory)
				case len(args) == 1 && strings.EqualFold(args[0], "memory"):
					peer.WriteBulk(testCase.memory)
				default:
					peer.WriteBulk(testCase.stats)
				}
				return true
			})

			stats := adapter.Stats()
			assert.Equal(t, testCase.wantHits, stats.Hits)
			assert.Equal(t, testCase.wantMisses, stats.Misses)
			assert.Equal(t, testCase.wantUsed, adapter.WeightedSize())
			assert.Equal(t, testCase.wantMaximum, adapter.GetMaximum())
		})
	}
}

func TestRedisAdapter_UnsupportedSettersDoNothing(t *testing.T) {
	t.Parallel()

	adapter, db := newStringAdapter(t, cache.Options[string, string]{})
	require.NoError(t, adapter.SetWithTTL(t.Context(), "a", "1", time.Minute))

	adapter.SetMaximum(10)
	require.NoError(t, adapter.SetRefreshableAfter(t.Context(), "a", time.Second))

	assert.Equal(t, time.Minute, db.TTL("plain:a"))
}

func TestRedisAdapter_ApplyJSONExpiry(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expiration time.Duration
		want       time.Duration
	}{
		{name: "KeepTTL leaves the TTL alone", expiration: redis.KeepTTL, want: time.Hour},
		{name: "a positive expiration replaces the TTL", expiration: time.Minute, want: time.Minute},
		{name: "no expiration removes the TTL", expiration: 0, want: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, db := newStringAdapter(t, cache.Options[string, string]{})
			require.NoError(t, db.Set("document", "{}"))
			db.SetTTL("document", time.Hour)

			adapter.applyJSONExpiry(t.Context(), "document", testCase.expiration)

			assert.True(t, db.Exists("document"))
			assert.Equal(t, testCase.want, db.TTL("document"))
		})
	}
}

func TestRedisAdapter_InvalidateAllWithoutNamespace(t *testing.T) {
	t.Parallel()

	db := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: db.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, db.Set("unrelated", "1"))

	blocked := &RedisAdapter[string, string]{client: client, flushTimeout: time.Minute}
	require.ErrorContains(t, blocked.InvalidateAll(t.Context()), "InvalidateAll blocked")
	assert.True(t, db.Exists("unrelated"))

	unsafe := &RedisAdapter[string, string]{client: client, flushTimeout: time.Minute, allowUnsafeFLUSHDB: true}
	require.NoError(t, unsafe.InvalidateAll(t.Context()))
	assert.False(t, db.Exists("unrelated"))
}
