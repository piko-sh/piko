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
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestRedisClusterAdapter_KeysValuesAndEntries(t *testing.T) {
	t.Parallel()

	adapter, _ := newStringAdapter(t, "iter", false)
	ctx := t.Context()
	require.NoError(t, adapter.Set(ctx, "a", "alpha", "tagged"))
	require.NoError(t, adapter.SetWithTTL(ctx, "b", "beta", time.Minute))

	assert.ElementsMatch(t, []string{"a", "b"}, slices.Collect(adapter.Keys()))
	assert.ElementsMatch(t, []string{"alpha", "beta"}, slices.Collect(adapter.Values()))
	assert.Equal(t, map[string]string{"a": "alpha", "b": "beta"}, maps.Collect(adapter.All()))

	for range adapter.Keys() {
		break
	}
	for range adapter.Values() {
		break
	}

	entry, found, err := adapter.GetEntry(ctx, "b")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "beta", entry.Value)
	assert.Positive(t, entry.ExpiresAtNano)

	_, found, err = adapter.GetEntry(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRedisClusterAdapter_ScanPatternFollowsTheNamespace(t *testing.T) {
	t.Parallel()

	adapter := &RedisClusterAdapter[string, string]{namespace: "users:"}
	assert.Equal(t, "users:*", adapter.buildScanPattern())

	adapter.namespace = ""
	assert.Equal(t, "*", adapter.buildScanPattern())
}

func TestRedisClusterAdapter_EstimatedSizeWithoutNamespaceCountsEveryKey(t *testing.T) {
	t.Parallel()

	adapter, db := newStringAdapter(t, "size", false)
	require.NoError(t, db.Set("other", "value"))
	require.NoError(t, adapter.Set(t.Context(), "a", "alpha"))
	adapter.namespace = ""

	assert.Equal(t, 2, adapter.EstimatedSize())
}

func TestParseRedisStatsInfo(t *testing.T) {
	t.Parallel()

	adapter := &RedisClusterAdapter[string, string]{}

	hits, misses := adapter.parseRedisStatsInfo(t.Context(), "# Stats\r\nkeyspace_hits:12\r\nkeyspace_misses:3\r\nother:1")
	assert.Equal(t, int64(12), hits)
	assert.Equal(t, int64(3), misses)

	hits, misses = adapter.parseRedisStatsInfo(t.Context(), "keyspace_hits:many\r\nkeyspace_misses:few")
	assert.Zero(t, hits)
	assert.Zero(t, misses)
}

func TestRedisClusterAdapter_ServerStatistics(t *testing.T) {
	t.Parallel()

	adapter, db := newStringAdapter(t, "stats", false)
	require.NoError(t, db.Server().Register("CONFIG", func(peer *server.Peer, _ string, arguments []string) {
		if len(arguments) == 2 && arguments[1] == "maxmemory" {
			peer.WriteMapLen(1)
			peer.WriteBulk("maxmemory")
			peer.WriteBulk("1048576")
			return
		}
		peer.WriteError("unsupported")
	}))

	assert.Equal(t, uint64(1048576), adapter.GetMaximum())
	assert.Equal(t, uint64(0), adapter.Stats().Hits)
	assert.Zero(t, adapter.WeightedSize(), "miniredis has no INFO memory section, so the size is unknown")
	adapter.SetMaximum(10)
	assert.Equal(t, uint64(1048576), adapter.GetMaximum())
	require.NoError(t, adapter.SetRefreshableAfter(t.Context(), "a", time.Minute))
}

func TestRedisClusterAdapter_ServerStatisticsOnAClosedServer(t *testing.T) {
	t.Parallel()

	adapter, db := newStringAdapter(t, "stats", false)
	adapter.operationTimeout = 200 * time.Millisecond
	db.Close()

	assert.Equal(t, uint64(0), adapter.GetMaximum())
	assert.Equal(t, uint64(0), adapter.Stats().Hits)
	assert.Equal(t, uint64(0), adapter.WeightedSize())
	assert.Zero(t, adapter.EstimatedSize())
}
