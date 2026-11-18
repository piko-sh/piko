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
	"maps"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

type nodeCommands struct {
	names []string
	mu sync.Mutex
}

func (c *nodeCommands) record(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !slices.Contains(c.names, name) {
		c.names = append(c.names, name)
	}
}

func (c *nodeCommands) seen() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.names)
}

func TestValkeyClusterAdapter_ProbeEntryReportsSubMillisecondTTL(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, redisServer := newMiniredisClusterAdapter(t)
	require.NoError(t, adapter.SetWithTTL(ctx, "key", 1, time.Minute))
	keyString, err := adapter.encodeKey("key")
	require.NoError(t, err)
	redisServer.SetTTL(keyString, 500*time.Microsecond)

	entry, found, err := adapter.ProbeEntry(ctx, "key")
	require.NoError(t, err)
	require.True(t, found)

	assert.NotZero(t, entry.ExpiresAtNano, "a TTL under one millisecond must not read as no expiry")
}

func TestExpiryFromPTTL(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	testCases := []struct {
		name            string
		ttlMilliseconds int64
		want            int64
	}{
		{name: "no TTL", ttlMilliseconds: -1, want: 0},
		{name: "missing key", ttlMilliseconds: -2, want: 0},
		{name: "under one millisecond left", ttlMilliseconds: 0, want: now.Add(time.Millisecond).UnixNano()},
		{name: "a TTL", ttlMilliseconds: 1500, want: now.Add(1500 * time.Millisecond).UnixNano()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, expiryFromPTTL(now, testCase.ttlMilliseconds))
		})
	}
}

func TestIsPrimaryRole(t *testing.T) {
	t.Parallel()

	assert.True(t, isPrimaryRole("master"))
	assert.True(t, isPrimaryRole("primary"))
	assert.False(t, isPrimaryRole("slave"))
	assert.False(t, isPrimaryRole("replica"))
	assert.False(t, isPrimaryRole("sentinel"))
}

func TestDropSeenKeysYieldsEachKeyOnceAcrossNodes(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	primary := dropSeenKeys([]string{"a", "b", "c"}, seen)
	replica := dropSeenKeys([]string{"c", "b", "a", "d"}, seen)

	assert.Equal(t, []string{"a", "b", "c"}, primary)
	assert.Equal(t, []string{"d"}, replica, "keys a replica repeats must not be yielded again")
}

func TestValkeyClusterAdapter_AllYieldsEachKeyOnce(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, _ := newMiniredisClusterAdapter(t)
	for index, key := range []string{"a", "b", "c"} {
		require.NoError(t, adapter.Set(ctx, key, float64(index)))
	}

	counts := make(map[string]int)
	for key := range adapter.Keys() {
		counts[key]++
	}

	assert.Equal(t, map[string]int{"a": 1, "b": 1, "c": 1}, counts)
	assert.Equal(t, 3, adapter.EstimatedSize())
}

func TestValkeyClusterAdapter_SetExpiresAfterIgnoresNonPositiveDurations(t *testing.T) {
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
			adapter, redisServer := newMiniredisClusterAdapter(t)
			require.NoError(t, adapter.SetWithTTL(ctx, "k", 1, time.Minute))
			keyString, err := adapter.encodeKey("k")
			require.NoError(t, err)

			require.NoError(t, adapter.SetExpiresAfter(ctx, "k", testCase.expiresAfter))

			assert.True(t, redisServer.Exists(keyString), "a non-positive duration must not delete the key")
			assert.Equal(t, time.Minute, redisServer.TTL(keyString), "a non-positive duration must not change the TTL")
		})
	}
}

func TestValkeyClusterAdapter_KeyspaceWalksSkipReplicas(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, primary := newMiniredisClusterAdapter(t)
	for index, key := range []string{"a", "b", "c"} {
		require.NoError(t, adapter.Set(ctx, key, float64(index)))
	}
	primaryCommands := answerAsNode(t, primary, "master", 3, 1, 1000)
	replica := newReplicaOf(t, adapter, primary)
	replicaCommands := answerAsNode(t, replica, "slave", 50, 20, 4000)

	counts := make(map[string]int)
	for key := range adapter.Keys() {
		counts[key]++
	}

	assert.Equal(t, map[string]int{"a": 1, "b": 1, "c": 1}, counts)
	assert.Equal(t, 3, adapter.EstimatedSize())
	assert.Equal(t, uint64(3), adapter.Stats().Hits)
	assert.Equal(t, uint64(1), adapter.Stats().Misses)
	assert.Equal(t, uint64(1000), adapter.WeightedSize())
	assert.Contains(t, primaryCommands.seen(), "SCAN")
	assert.Equal(t, []string{"ROLE"}, replicaCommands.seen(), "a replica must only be asked its role")
}

func TestValkeyClusterAdapter_AllDropsKeysHeldByTwoNodes(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, primary := newMiniredisClusterAdapter(t)
	for index, key := range []string{"a", "b", "c"} {
		require.NoError(t, adapter.Set(ctx, key, float64(index)))
	}
	newReplicaOf(t, adapter, primary)

	counts := make(map[string]int)
	for key := range adapter.Keys() {
		counts[key]++
	}

	assert.Equal(t, map[string]int{"a": 1, "b": 1, "c": 1}, counts,
		"a node whose role cannot be read is walked, so its copies must be dropped")
}

func answerAsNode(t *testing.T, node *miniredis.Miniredis, role string, hits, misses, usedMemory int) *nodeCommands {
	t.Helper()

	commands := &nodeCommands{}
	node.Server().SetPreHook(func(peer *server.Peer, command string, arguments ...string) bool {
		switch command {
		case "ROLE":
			commands.record(command)
			writeRoleReply(peer, role)
			return true
		case "INFO":
			commands.record(command)
			if len(arguments) == 1 && arguments[0] == "memory" {
				peer.WriteBulk("# Memory\r\nused_memory:" + strconv.Itoa(usedMemory) + "\r\n")
				return true
			}
			peer.WriteBulk("# Stats\r\nkeyspace_hits:" + strconv.Itoa(hits) + "\r\nkeyspace_misses:" + strconv.Itoa(misses) + "\r\n")
			return true
		case "SCAN", "DBSIZE", "FLUSHDB", "FLUSHALL", "KEYS", "DEL":
			commands.record(command)
		}
		return false
	})
	return commands
}

func writeRoleReply(peer *server.Peer, role string) {
	if role != "slave" {
		peer.WriteLen(3)
		peer.WriteBulk(role)
		peer.WriteInt(0)
		peer.WriteLen(0)
		return
	}
	peer.WriteLen(5)
	peer.WriteBulk(role)
	peer.WriteBulk("127.0.0.1")
	peer.WriteInt(6379)
	peer.WriteBulk("connected")
	peer.WriteInt(0)
}

func newReplicaOf(t *testing.T, adapter *ValkeyClusterAdapter[string, float64], primary *miniredis.Miniredis) *miniredis.Miniredis {
	t.Helper()

	replica := miniredis.RunT(t)
	for _, key := range primary.Keys() {
		value, err := primary.Get(key)
		require.NoError(t, err)
		require.NoError(t, replica.Set(key, value))
	}

	replicaClient, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:       []string{replica.Addr()},
		ForceSingleClient: true,
		DisableCache:      true,
	})
	require.NoError(t, err)
	t.Cleanup(replicaClient.Close)

	nodes := maps.Clone(adapter.client.Nodes())
	require.Len(t, nodes, 1)
	nodes[replica.Addr()] = replicaClient
	adapter.listNodes = func() map[string]valkey.Client { return nodes }
	return replica
}
