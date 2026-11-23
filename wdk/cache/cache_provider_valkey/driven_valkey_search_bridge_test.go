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
package cache_provider_valkey

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
)

func TestValkeyVectorSearchReturnsOnlyTheRequestedPage(t *testing.T) {
	adapter, db := newMiniredisAdapter(t)
	commands := make(chan []string, 1)
	require.NoError(t, db.Server().Register("FT.SEARCH", func(peer *server.Peer, _ string, args []string) {
		commands <- slices.Clone(args)
		peer.WriteLen(1)
		peer.WriteInt(0)
	}))
	_, err := adapter.vectorSearchWithValkeySearch(t.Context(), "", &cache.SearchOptions{Vector: []float32{1, 0}, VectorField: "vec", TopK: 30, Offset: 3, Limit: 2})
	require.NoError(t, err)
	arguments := <-commands
	require.Contains(t, arguments, "(*)=>[KNN 30 @vec $vec]")
	start := slices.Index(arguments, "LIMIT")
	require.NotEqual(t, -1, start)
	require.Equal(t, []string{"3", "2"}, arguments[start+1:start+3])
}

func TestValkeyAdapter_SetExpiresAfterIgnoresNonPositiveDurations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		expiresAfter time.Duration
	}{
		{name: "zero", expiresAfter: 0},
		{name: "negative", expiresAfter: -time.Second},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			adapter, server := newMiniredisAdapter(t)
			require.NoError(t, adapter.SetWithTTL(ctx, "k", 1, time.Minute))
			keyString, err := adapter.encodeKey("k")
			require.NoError(t, err)

			require.NoError(t, adapter.SetExpiresAfter(ctx, "k", testCase.expiresAfter))

			assert.True(t, server.Exists(keyString), "a non-positive duration must not delete the key")
			assert.Equal(t, time.Minute, server.TTL(keyString), "a non-positive duration must not change the TTL")
		})
	}
}

func TestValkeyAdapter_SearchWithoutSchemaIsNotSupported(t *testing.T) {
	t.Parallel()

	adapter := &ValkeyAdapter[string, string]{}

	_, err := adapter.Search(t.Context(), "", nil)
	require.ErrorIs(t, err, cache.ErrSearchNotSupported)
	_, err = adapter.Query(t.Context(), nil)
	require.ErrorIs(t, err, cache.ErrSearchNotSupported)
	assert.False(t, adapter.SupportsSearch())
	assert.Nil(t, adapter.GetSchema())
}

func TestValkeyAdapter_SearchSchemaIsReported(t *testing.T) {
	t.Parallel()

	schema := vectorSchema()
	adapter := &ValkeyAdapter[string, string]{schema: schema}

	assert.True(t, adapter.SupportsSearch())
	assert.Same(t, schema, adapter.GetSchema())
}

func TestValkeyAdapter_ServerStatistics(t *testing.T) {
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

			adapter, db := newMiniredisAdapter(t)
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

func TestValkeyAdapter_UnsupportedSettersDoNothing(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, db := newMiniredisAdapter(t)
	require.NoError(t, adapter.SetWithTTL(ctx, "a", 1, time.Minute))
	keyString, err := adapter.encodeKey("a")
	require.NoError(t, err)

	adapter.SetMaximum(10)
	require.NoError(t, adapter.SetRefreshableAfter(ctx, "a", time.Second))

	assert.Equal(t, time.Minute, db.TTL(keyString))
}

func TestValkeyAdapter_SetExpiresAfter(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, db := newMiniredisAdapter(t)
	require.NoError(t, adapter.SetWithTTL(ctx, "a", 1, time.Minute))
	keyString, err := adapter.encodeKey("a")
	require.NoError(t, err)

	require.NoError(t, adapter.SetExpiresAfter(ctx, "a", time.Hour))
	assert.Equal(t, time.Hour, db.TTL(keyString))

	require.NoError(t, adapter.SetExpiresAfter(ctx, "a", time.Microsecond))
	assert.Equal(t, time.Millisecond, db.TTL(keyString))

	require.NoError(t, adapter.SetExpiresAfter(ctx, "a", cache.NoExpiry))
	assert.Zero(t, db.TTL(keyString))
	assert.True(t, db.Exists(keyString))

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, adapter.SetExpiresAfter(cancelled, "a", time.Hour), context.Canceled)
}

func TestValkeyAdapter_EstimatedSize(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, db := newMiniredisAdapter(t)
	require.NoError(t, db.Set("unrelated", "1"))
	for _, key := range []string{"a", "b", "c"} {
		require.NoError(t, adapter.Set(ctx, key, 1))
	}

	assert.Equal(t, 3, adapter.EstimatedSize())

	adapter.namespace = ""
	assert.Equal(t, 4, adapter.EstimatedSize())
}

func TestValkeyAdapter_CloseReportsADoneContext(t *testing.T) {
	t.Parallel()

	adapter, _ := newMiniredisAdapter(t)
	require.NoError(t, adapter.Close(t.Context()))

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, adapter.Close(cancelled), context.Canceled)
}
