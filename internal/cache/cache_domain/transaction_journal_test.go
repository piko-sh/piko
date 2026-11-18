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

package cache_domain

import (
	"context"
	"errors"
	"iter"
	"maps"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/wdk/clock"
)

type testCache struct {
	data           map[string]string
	expires        map[string]int64
	ttls           map[string]time.Duration
	failures       map[string]error
	snapshotAtNano int64
}

func newTestCache() *testCache {
	return &testCache{
		data:     make(map[string]string),
		expires:  make(map[string]int64),
		ttls:     make(map[string]time.Duration),
		failures: make(map[string]error),
	}
}

func (c *testCache) GetIfPresent(_ context.Context, key string) (string, bool, error) {
	v, ok := c.data[key]
	return v, ok, nil
}

func (c *testCache) Get(ctx context.Context, key string, loader cache_dto.Loader[string, string]) (string, error) {
	v, ok, _ := c.GetIfPresent(ctx, key)
	if ok {
		return v, nil
	}
	loaded, err := loader.Load(ctx, key)
	if err != nil {
		return "", err
	}
	c.data[key] = loaded
	return loaded, nil
}

func (c *testCache) Set(_ context.Context, key string, value string, _ ...string) error {
	c.data[key] = value
	return nil
}

func (c *testCache) SetWithTTL(_ context.Context, key string, value string, ttl time.Duration, _ ...string) error {
	if err := c.failures[key]; err != nil {
		return err
	}
	c.data[key] = value
	c.ttls[key] = ttl
	if ttl == cache_dto.NoExpiry {
		c.expires[key] = 0
		return nil
	}
	c.expires[key] = time.Now().Add(ttl).UnixNano()
	return nil
}

func (c *testCache) Invalidate(_ context.Context, key string) error {
	if err := c.failures[key]; err != nil {
		return err
	}
	delete(c.data, key)
	delete(c.expires, key)
	return nil
}

func (c *testCache) Compute(_ context.Context, key string, fn func(string, bool) (string, cache_dto.ComputeAction)) (string, bool, error) {
	old, found := c.data[key]
	newVal, action := fn(old, found)
	switch action {
	case cache_dto.ComputeActionSet:
		c.data[key] = newVal
		return newVal, true, nil
	case cache_dto.ComputeActionDelete:
		delete(c.data, key)
		return "", false, nil
	default:
		return old, found, nil
	}
}

func (c *testCache) ComputeIfAbsent(_ context.Context, key string, fn func() string) (string, bool, error) {
	if v, ok := c.data[key]; ok {
		return v, false, nil
	}
	v := fn()
	c.data[key] = v
	return v, true, nil
}

func (c *testCache) ComputeIfPresent(_ context.Context, key string, fn func(string) (string, cache_dto.ComputeAction)) (string, bool, error) {
	old, found := c.data[key]
	if !found {
		return "", false, nil
	}
	newVal, action := fn(old)
	switch action {
	case cache_dto.ComputeActionSet:
		c.data[key] = newVal
		return newVal, true, nil
	case cache_dto.ComputeActionDelete:
		delete(c.data, key)
		return "", false, nil
	default:
		return old, true, nil
	}
}

func (c *testCache) ComputeWithTTL(_ context.Context, key string, fn func(string, bool) cache_dto.ComputeResult[string]) (string, bool, error) {
	old, found := c.data[key]
	result := fn(old, found)
	switch result.Action {
	case cache_dto.ComputeActionSet:
		c.data[key] = result.Value
		return result.Value, true, nil
	case cache_dto.ComputeActionDelete:
		delete(c.data, key)
		return "", false, nil
	default:
		return old, found, nil
	}
}

func (c *testCache) BulkSet(ctx context.Context, items map[string]string, tags ...string) error {
	for k, v := range items {
		if err := c.Set(ctx, k, v, tags...); err != nil {
			return err
		}
	}
	return nil
}

func (c *testCache) BulkGet(ctx context.Context, keys []string, loader cache_dto.BulkLoader[string, string]) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	var missing []string
	for _, k := range keys {
		if v, ok := c.data[k]; ok {
			result[k] = v
		} else {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 && loader != nil {
		loaded, err := loader.BulkLoad(ctx, missing)
		if err != nil {
			return result, err
		}
		for k, v := range loaded {
			c.data[k] = v
			result[k] = v
		}
	}
	return result, nil
}

func (c *testCache) InvalidateByTags(_ context.Context, _ ...string) (int, error) { return 0, nil }
func (c *testCache) InvalidateAll(_ context.Context) error {
	c.data = make(map[string]string)
	return nil
}

func (c *testCache) BulkRefresh(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, string]) {
}

func (c *testCache) Refresh(_ context.Context, _ string, _ cache_dto.Loader[string, string]) <-chan cache_dto.LoadResult[string] {
	ch := make(chan cache_dto.LoadResult[string], 1)
	close(ch)
	return ch
}

func (c *testCache) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for k, v := range c.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (c *testCache) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for k := range c.data {
			if !yield(k) {
				return
			}
		}
	}
}

func (c *testCache) Values() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, v := range c.data {
			if !yield(v) {
				return
			}
		}
	}
}

func (c *testCache) GetEntry(_ context.Context, key string) (cache_dto.Entry[string, string], bool, error) {
	v, ok := c.data[key]
	if !ok {
		return cache_dto.Entry[string, string]{}, false, nil
	}
	return cache_dto.Entry[string, string]{Key: key, Value: v, ExpiresAtNano: c.expires[key], SnapshotAtNano: c.snapshotAtNano}, true, nil
}

func (c *testCache) ProbeEntry(ctx context.Context, key string) (cache_dto.Entry[string, string], bool, error) {
	return c.GetEntry(ctx, key)
}

func (c *testCache) EstimatedSize() int            { return len(c.data) }
func (c *testCache) Stats() cache_dto.Stats        { return cache_dto.Stats{} }
func (c *testCache) Close(_ context.Context) error { return nil }
func (c *testCache) GetMaximum() uint64            { return 0 }
func (c *testCache) SetMaximum(_ uint64)           {}
func (c *testCache) WeightedSize() uint64          { return 0 }
func (c *testCache) SetExpiresAfter(_ context.Context, key string, expiresAfter time.Duration) error {
	if _, ok := c.data[key]; ok && expiresAfter > 0 {
		c.expires[key] = time.Now().Add(expiresAfter).UnixNano()
	}
	return nil
}
func (c *testCache) SetRefreshableAfter(_ context.Context, _ string, _ time.Duration) error {
	return nil
}

func (c *testCache) Search(_ context.Context, _ string, _ *cache_dto.SearchOptions) (cache_dto.SearchResult[string, string], error) {
	return cache_dto.SearchResult[string, string]{}, ErrSearchNotSupported
}

func (c *testCache) Query(_ context.Context, _ *cache_dto.QueryOptions) (cache_dto.SearchResult[string, string], error) {
	return cache_dto.SearchResult[string, string]{}, ErrSearchNotSupported
}

func (c *testCache) SupportsSearch() bool               { return false }
func (c *testCache) GetSchema() *cache_dto.SearchSchema { return nil }

func TestTransactionJournal_CommitPreservesMutations(t *testing.T) {
	tests := []struct {
		setup     map[string]string
		mutate    func(t *testing.T, tx TransactionCache[string, string])
		wantAfter map[string]string
		name      string
	}{
		{
			name:  "set new key and commit",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.Set(context.Background(), "a", "1"))
			},
			wantAfter: map[string]string{"a": "1"},
		},
		{
			name:  "overwrite existing key and commit",
			setup: map[string]string{"a": "old"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.Set(context.Background(), "a", "new"))
			},
			wantAfter: map[string]string{"a": "new"},
		},
		{
			name:  "invalidate existing key and commit",
			setup: map[string]string{"a": "1", "b": "2"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.Invalidate(context.Background(), "a"))
			},
			wantAfter: map[string]string{"b": "2"},
		},
		{
			name:  "multiple mutations and commit",
			setup: map[string]string{"a": "1"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				ctx := context.Background()
				require.NoError(t, tx.Set(ctx, "b", "2"))
				require.NoError(t, tx.Set(ctx, "c", "3"))
				require.NoError(t, tx.Invalidate(ctx, "a"))
			},
			wantAfter: map[string]string{"b": "2", "c": "3"},
		},
		{
			name:  "set with TTL and commit",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.SetWithTTL(context.Background(), "a", "1", time.Hour))
			},
			wantAfter: map[string]string{"a": "1"},
		},
		{
			name:  "bulk set and commit",
			setup: map[string]string{"a": "old"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.BulkSet(context.Background(), map[string]string{
					"a": "new",
					"b": "2",
				}))
			},
			wantAfter: map[string]string{"a": "new", "b": "2"},
		},
		{
			name:  "compute set action and commit",
			setup: map[string]string{"a": "old"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				_, _, err := tx.Compute(context.Background(), "a", func(old string, found bool) (string, cache_dto.ComputeAction) {
					require.True(t, found)
					require.Equal(t, "old", old)
					return "new", cache_dto.ComputeActionSet
				})
				require.NoError(t, err)
			},
			wantAfter: map[string]string{"a": "new"},
		},
		{
			name:  "compute delete action and commit",
			setup: map[string]string{"a": "1"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				_, _, err := tx.Compute(context.Background(), "a", func(string, bool) (string, cache_dto.ComputeAction) {
					return "", cache_dto.ComputeActionDelete
				})
				require.NoError(t, err)
			},
			wantAfter: map[string]string{},
		},
		{
			name:  "compute if absent creates new key and commits",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				val, computed, err := tx.ComputeIfAbsent(context.Background(), "a", func() string { return "1" })
				require.NoError(t, err)
				require.True(t, computed)
				require.Equal(t, "1", val)
			},
			wantAfter: map[string]string{"a": "1"},
		},
		{
			name:  "compute if present updates existing key and commits",
			setup: map[string]string{"a": "old"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				val, present, err := tx.ComputeIfPresent(context.Background(), "a", func(old string) (string, cache_dto.ComputeAction) {
					return "new", cache_dto.ComputeActionSet
				})
				require.NoError(t, err)
				require.True(t, present)
				require.Equal(t, "new", val)
			},
			wantAfter: map[string]string{"a": "new"},
		},
		{
			name:  "compute with TTL and commit",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				_, _, err := tx.ComputeWithTTL(context.Background(), "a", func(string, bool) cache_dto.ComputeResult[string] {
					return cache_dto.ComputeResult[string]{Value: "1", Action: cache_dto.ComputeActionSet, TTL: time.Hour}
				})
				require.NoError(t, err)
			},
			wantAfter: map[string]string{"a": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newTestCache()
			maps.Copy(cache.data, tt.setup)

			tx := newTransactionJournal[string, string](cache)
			tt.mutate(t, tx)

			err := tx.Commit(context.Background())
			require.NoError(t, err)

			require.Equal(t, tt.wantAfter, cache.data)
		})
	}
}

func TestTransactionJournal_RollbackRestoresState(t *testing.T) {
	tests := []struct {
		setup     map[string]string
		mutate    func(t *testing.T, tx TransactionCache[string, string])
		wantAfter map[string]string
		name      string
	}{
		{
			name:  "rollback new key removes it",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.Set(context.Background(), "a", "1"))
			},
			wantAfter: map[string]string{},
		},
		{
			name:  "rollback overwritten key restores old value",
			setup: map[string]string{"a": "old"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.Set(context.Background(), "a", "new"))
			},
			wantAfter: map[string]string{"a": "old"},
		},
		{
			name:  "rollback invalidated key restores it",
			setup: map[string]string{"a": "1", "b": "2"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.Invalidate(context.Background(), "a"))
			},
			wantAfter: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:  "rollback multiple mutations restores all",
			setup: map[string]string{"a": "1", "b": "2"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				ctx := context.Background()
				require.NoError(t, tx.Set(ctx, "c", "3"))
				require.NoError(t, tx.Set(ctx, "a", "modified"))
				require.NoError(t, tx.Invalidate(ctx, "b"))
			},
			wantAfter: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:  "rollback multiple mutations to same key restores original",
			setup: map[string]string{"a": "original"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				ctx := context.Background()
				require.NoError(t, tx.Set(ctx, "a", "first"))
				require.NoError(t, tx.Set(ctx, "a", "second"))
				require.NoError(t, tx.Set(ctx, "a", "third"))
			},
			wantAfter: map[string]string{"a": "original"},
		},
		{
			name:  "rollback set then invalidate same key restores original",
			setup: map[string]string{"a": "original"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				ctx := context.Background()
				require.NoError(t, tx.Set(ctx, "a", "modified"))
				require.NoError(t, tx.Invalidate(ctx, "a"))
			},
			wantAfter: map[string]string{"a": "original"},
		},
		{
			name:  "rollback invalidate then set same key restores absence",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				ctx := context.Background()
				require.NoError(t, tx.Invalidate(ctx, "a"))
				require.NoError(t, tx.Set(ctx, "a", "new"))
			},
			wantAfter: map[string]string{},
		},
		{
			name:  "rollback set with TTL restores absence",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.SetWithTTL(context.Background(), "a", "1", time.Hour))
			},
			wantAfter: map[string]string{},
		},
		{
			name:  "rollback bulk set restores all keys",
			setup: map[string]string{"a": "old"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				require.NoError(t, tx.BulkSet(context.Background(), map[string]string{
					"a": "new",
					"b": "added",
				}))
			},
			wantAfter: map[string]string{"a": "old"},
		},
		{
			name:  "rollback compute set action restores old value",
			setup: map[string]string{"a": "old"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				_, _, err := tx.Compute(context.Background(), "a", func(string, bool) (string, cache_dto.ComputeAction) {
					return "new", cache_dto.ComputeActionSet
				})
				require.NoError(t, err)
			},
			wantAfter: map[string]string{"a": "old"},
		},
		{
			name:  "rollback compute delete action restores key",
			setup: map[string]string{"a": "1"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				_, _, err := tx.Compute(context.Background(), "a", func(string, bool) (string, cache_dto.ComputeAction) {
					return "", cache_dto.ComputeActionDelete
				})
				require.NoError(t, err)
			},
			wantAfter: map[string]string{"a": "1"},
		},
		{
			name:  "rollback compute if absent removes created key",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				_, _, err := tx.ComputeIfAbsent(context.Background(), "a", func() string { return "1" })
				require.NoError(t, err)
			},
			wantAfter: map[string]string{},
		},
		{
			name:  "rollback compute if present restores old value",
			setup: map[string]string{"a": "old"},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				_, _, err := tx.ComputeIfPresent(context.Background(), "a", func(string) (string, cache_dto.ComputeAction) {
					return "new", cache_dto.ComputeActionSet
				})
				require.NoError(t, err)
			},
			wantAfter: map[string]string{"a": "old"},
		},
		{
			name:  "rollback compute with TTL restores absence",
			setup: map[string]string{},
			mutate: func(t *testing.T, tx TransactionCache[string, string]) {
				_, _, err := tx.ComputeWithTTL(context.Background(), "a", func(string, bool) cache_dto.ComputeResult[string] {
					return cache_dto.ComputeResult[string]{Value: "1", Action: cache_dto.ComputeActionSet}
				})
				require.NoError(t, err)
			},
			wantAfter: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newTestCache()
			maps.Copy(cache.data, tt.setup)

			tx := newTransactionJournal[string, string](cache)
			tt.mutate(t, tx)

			err := tx.Rollback(context.Background())
			require.NoError(t, err)

			require.Equal(t, tt.wantAfter, cache.data)
		})
	}
}

func TestTransactionJournal_RollbackRestoresTTL(t *testing.T) {
	ctx := context.Background()
	cache := newTestCache()

	futureExpiry := time.Now().Add(10 * time.Minute).UnixNano()
	cache.data["a"] = "original"
	cache.expires["a"] = futureExpiry

	tx := newTransactionJournal[string, string](cache)

	require.NoError(t, tx.Set(ctx, "a", "modified"))

	require.NoError(t, tx.Rollback(ctx))

	require.Equal(t, "original", cache.data["a"])
	require.NotZero(t, cache.expires["a"], "TTL should be restored on rollback")

	require.Greater(t, cache.expires["a"], time.Now().UnixNano(),
		"restored expiry should still be in the future")
}

func TestTransactionJournal_RollbackExpiredTTLInvalidatesKey(t *testing.T) {
	ctx := context.Background()
	cache := newTestCache()

	pastExpiry := time.Now().Add(-1 * time.Second).UnixNano()
	cache.data["a"] = "expired"
	cache.expires["a"] = pastExpiry

	tx := newTransactionJournal[string, string](cache)

	require.NoError(t, tx.Set(ctx, "a", "modified"))

	require.NoError(t, tx.Rollback(ctx))

	_, exists := cache.data["a"]
	require.False(t, exists, "key with elapsed TTL should be invalidated on rollback")
}

func TestTransactionJournal_RollbackUndoesLoaderPopulation(t *testing.T) {
	ctx := context.Background()
	cache := newTestCache()

	loader := cache_dto.LoaderFunc[string, string](func(_ context.Context, key string) (string, error) {
		return "loaded-" + key, nil
	})

	tx := newTransactionJournal[string, string](cache)

	val, err := tx.Get(ctx, "a", loader)
	require.NoError(t, err)
	require.Equal(t, "loaded-a", val)
	require.Equal(t, "loaded-a", cache.data["a"], "loader should have populated the cache")

	require.NoError(t, tx.Rollback(ctx))

	_, exists := cache.data["a"]
	require.False(t, exists, "loader-populated key should be removed on rollback")
}

func TestTransactionJournal_RollbackUndoesBulkLoaderPopulation(t *testing.T) {
	ctx := context.Background()
	cache := newTestCache()
	cache.data["a"] = "existing"

	bulkLoader := cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, keys []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, k := range keys {
			result[k] = "loaded-" + k
		}
		return result, nil
	})

	tx := newTransactionJournal[string, string](cache)

	vals, err := tx.BulkGet(ctx, []string{"a", "b"}, bulkLoader)
	require.NoError(t, err)
	require.Equal(t, "existing", vals["a"], "existing key should not be reloaded")
	require.Equal(t, "loaded-b", vals["b"])

	require.NoError(t, tx.Rollback(ctx))

	require.Equal(t, "existing", cache.data["a"])
	_, exists := cache.data["b"]
	require.False(t, exists, "loader-populated key should be removed on rollback")
}

func TestTransactionJournal_EmptyTransaction(t *testing.T) {
	tests := []struct {
		action func(tx TransactionCache[string, string]) error
		name   string
	}{
		{
			name: "empty commit",
			action: func(tx TransactionCache[string, string]) error {
				return tx.Commit(context.Background())
			},
		},
		{
			name: "empty rollback",
			action: func(tx TransactionCache[string, string]) error {
				return tx.Rollback(context.Background())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newTestCache()
			cache.data["a"] = "1"

			tx := newTransactionJournal[string, string](cache)
			err := tt.action(tx)
			require.NoError(t, err)

			require.Equal(t, map[string]string{"a": "1"}, cache.data)
		})
	}
}

func TestTransactionJournal_ReadsDelegateToInner(t *testing.T) {
	cache := newTestCache()
	cache.data["a"] = "1"
	cache.data["b"] = "2"

	tx := newTransactionJournal[string, string](cache)
	ctx := context.Background()

	t.Run("GetIfPresent returns inner value", func(t *testing.T) {
		v, ok, err := tx.GetIfPresent(ctx, "a")
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, "1", v)
	})

	t.Run("GetIfPresent returns miss for absent key", func(t *testing.T) {
		_, ok, err := tx.GetIfPresent(ctx, "missing")
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("EstimatedSize delegates", func(t *testing.T) {
		require.Equal(t, 2, tx.EstimatedSize())
	})

	t.Run("All iterates inner cache", func(t *testing.T) {
		result := maps.Collect(tx.All())
		require.Equal(t, map[string]string{"a": "1", "b": "2"}, result)
	})
}

func TestTransactionJournal_ReadsReflectMutations(t *testing.T) {
	cache := newTestCache()
	cache.data["a"] = "old"

	tx := newTransactionJournal[string, string](cache)
	ctx := context.Background()

	require.NoError(t, tx.Set(ctx, "a", "new"))
	require.NoError(t, tx.Set(ctx, "b", "added"))

	v, ok, err := tx.GetIfPresent(ctx, "a")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "new", v, "reads should reflect mutations since they are applied immediately")

	v, ok, err = tx.GetIfPresent(ctx, "b")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "added", v)
}

func TestTransactionJournal_CloseDoesNotCloseInner(t *testing.T) {
	cache := newTestCache()
	cache.data["a"] = "1"

	tx := newTransactionJournal[string, string](cache)
	err := tx.Close(context.Background())
	require.NoError(t, err)

	v, ok, _ := cache.GetIfPresent(context.Background(), "a")
	require.True(t, ok)
	require.Equal(t, "1", v)
}

func TestBeginTransaction_UsesJournalFallback(t *testing.T) {
	cache := newTestCache()
	cache.data["a"] = "1"

	tx := BeginTransaction[string, string](context.Background(), cache)
	require.NoError(t, tx.Set(context.Background(), "a", "modified"))

	err := tx.Rollback(context.Background())
	require.NoError(t, err)

	require.Equal(t, "1", cache.data["a"], "BeginTransaction should use journal fallback and support rollback")
}

type testTransactionalCache struct {
	*testCache
	beginCalled bool
}

func (c *testTransactionalCache) BeginTransaction(_ context.Context) TransactionCache[string, string] {
	c.beginCalled = true
	return newTransactionJournal[string, string](c.testCache)
}

func TestBeginTransaction_PrefersNativeTransactional(t *testing.T) {
	inner := newTestCache()
	cache := &testTransactionalCache{testCache: inner}

	tx := BeginTransaction[string, string](context.Background(), cache)
	require.True(t, cache.beginCalled, "BeginTransaction should prefer native Transactional implementation")
	require.NotNil(t, tx)
}

func TestTransactionJournal_DoubleCommitReturnsSentinel(t *testing.T) {
	cache := newTestCache()
	tx := newTransactionJournal[string, string](cache)
	ctx := context.Background()

	require.NoError(t, tx.Commit(ctx))
	err := tx.Commit(ctx)
	require.ErrorIs(t, err, ErrTransactionFinalised)
}

func TestTransactionJournal_DoubleRollbackReturnsSentinel(t *testing.T) {
	cache := newTestCache()
	tx := newTransactionJournal[string, string](cache)
	ctx := context.Background()

	require.NoError(t, tx.Rollback(ctx))
	err := tx.Rollback(ctx)
	require.ErrorIs(t, err, ErrTransactionFinalised)
}

func TestTransactionJournal_MutationAfterCommitReturnsSentinel(t *testing.T) {
	cache := newTestCache()
	tx := newTransactionJournal[string, string](cache)
	ctx := context.Background()

	require.NoError(t, tx.Commit(ctx))
	err := tx.Set(ctx, "a", "1")
	require.ErrorIs(t, err, ErrTransactionFinalised)
}

func TestTransactionJournal_MutationAfterRollbackReturnsSentinel(t *testing.T) {
	cache := newTestCache()
	tx := newTransactionJournal[string, string](cache)
	ctx := context.Background()

	require.NoError(t, tx.Rollback(ctx))
	err := tx.Set(ctx, "a", "1")
	require.ErrorIs(t, err, ErrTransactionFinalised)
}

func TestTransactionJournal_InvalidateByTagsReturnsSentinel(t *testing.T) {
	cache := newTestCache()
	tx := newTransactionJournal[string, string](cache)

	_, err := tx.InvalidateByTags(context.Background(), "tag1")
	require.ErrorIs(t, err, ErrInvalidateByTagsUnsupported)
}

func TestTransactionJournal_InvalidateAllReturnsSentinel(t *testing.T) {
	cache := newTestCache()
	tx := newTransactionJournal[string, string](cache)

	err := tx.InvalidateAll(context.Background())
	require.ErrorIs(t, err, ErrInvalidateAllUnsupported)
}

func TestTransactionJournal_CommitThenRollbackReturnsSentinel(t *testing.T) {
	cache := newTestCache()
	tx := newTransactionJournal[string, string](cache)
	ctx := context.Background()

	require.NoError(t, tx.Set(ctx, "a", "1"))
	require.NoError(t, tx.Commit(ctx))

	err := tx.Rollback(ctx)
	require.ErrorIs(t, err, ErrTransactionFinalised)

	v, ok, _ := cache.GetIfPresent(ctx, "a")
	require.True(t, ok)
	require.Equal(t, "1", v)
}

func TestTransactionJournal_RollbackThenCommitReturnsSentinel(t *testing.T) {
	cache := newTestCache()
	tx := newTransactionJournal[string, string](cache)
	ctx := context.Background()

	require.NoError(t, tx.Set(ctx, "a", "1"))
	require.NoError(t, tx.Rollback(ctx))

	err := tx.Commit(ctx)
	require.ErrorIs(t, err, ErrTransactionFinalised)

	_, ok, _ := cache.GetIfPresent(ctx, "a")
	require.False(t, ok)
}

func TestTransactionJournal_RollbackOfEntryWithoutTTLRestoresNoExpiry(t *testing.T) {
	testCases := []struct {
		write         func(ctx context.Context, transaction *transactionJournal[string, string]) error
		name          string
		expiresAtNano int64
	}{
		{
			name:          "zero expiry undone after Set",
			expiresAtNano: 0,
			write: func(ctx context.Context, transaction *transactionJournal[string, string]) error {
				return transaction.Set(ctx, "a", "modified")
			},
		},
		{
			name:          "largest representable expiry, as Otter reports an entry without a TTL",
			expiresAtNano: math.MaxInt64,
			write: func(ctx context.Context, transaction *transactionJournal[string, string]) error {
				return transaction.SetWithTTL(ctx, "a", "modified", time.Minute)
			},
		},
		{
			name:          "an expiry the transaction gave the entry",
			expiresAtNano: 0,
			write: func(ctx context.Context, transaction *transactionJournal[string, string]) error {
				return transaction.SetExpiresAfter(ctx, "a", time.Minute)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := t.Context()
			cache := newTestCache()
			cache.data["a"] = "original"
			cache.expires["a"] = testCase.expiresAtNano

			transaction := newTransactionJournal[string, string](cache)
			require.NoError(t, testCase.write(ctx, transaction))
			require.NoError(t, transaction.Rollback(ctx))

			require.Equal(t, "original", cache.data["a"])
			require.Equal(t, cache_dto.NoExpiry, cache.ttls["a"],
				"rollback must clear any TTL the transaction set, not leave it in place with a plain Set")
			require.Zero(t, cache.expires["a"])
		})
	}
}

func TestTransactionJournal_RollbackMeasuresTheTTLOnTheProviderClock(t *testing.T) {
	ctx := t.Context()
	cache := newTestCache()
	providerNow := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC)
	cache.snapshotAtNano = providerNow.UnixNano()
	cache.data["a"] = "original"
	cache.expires["a"] = providerNow.Add(10 * time.Minute).UnixNano()

	transaction := newTransactionJournal[string, string](cache)
	require.NoError(t, transaction.Set(ctx, "a", "modified"))
	require.NoError(t, transaction.Rollback(ctx))

	require.Equal(t, "original", cache.data["a"],
		"an entry with ten minutes left on its own clock must not be dropped because the system clock differs")
	assert.Greater(t, cache.ttls["a"], 10*time.Minute-time.Second)
	assert.LessOrEqual(t, cache.ttls["a"], 10*time.Minute)
}

func TestTransactionJournal_WithTransactionClockMeasuresTheTransaction(t *testing.T) {
	ctx := t.Context()
	cache := newTestCache()
	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	cache.data["a"] = "original"
	cache.expires["a"] = mockClock.Now().Add(10 * time.Minute).UnixNano()

	transaction := BeginTransaction[string, string](ctx, cache, WithTransactionClock(mockClock))
	require.NoError(t, transaction.Set(ctx, "a", "modified"))
	mockClock.Advance(3 * time.Minute)
	require.NoError(t, transaction.Rollback(ctx))

	assert.Equal(t, 7*time.Minute, cache.ttls["a"],
		"the time the transaction ran comes off the TTL the entry had left")
}

func TestTransactionJournal_RollbackJoinsEveryRestoreFailure(t *testing.T) {
	ctx := t.Context()
	cache := newTestCache()
	firstFailure := errors.New("first failure")
	secondFailure := errors.New("second failure")
	cache.data["first"] = "original"
	cache.data["second"] = "original"
	cache.data["third"] = "original"
	cache.failures["first"] = firstFailure
	cache.failures["second"] = secondFailure

	transaction := newTransactionJournal[string, string](cache)
	for _, key := range []string{"first", "second", "third"} {
		require.NoError(t, transaction.Set(ctx, key, "modified"))
	}

	err := transaction.Rollback(ctx)

	require.ErrorIs(t, err, firstFailure)
	require.ErrorIs(t, err, secondFailure)
	assert.Contains(t, err.Error(), "first")
	assert.Contains(t, err.Error(), "second")
	assert.Equal(t, "original", cache.data["third"], "one failed restore must not stop the others")
}

func TestTransactionJournal_RollbackErrorsLeaveOutKeys(t *testing.T) {
	ctx := t.Context()
	cache := newTestCache()
	const restoredKey = "restored-user@example.com"
	const addedKey = "added-user@example.com"
	cache.data[restoredKey] = "original"
	cache.failures[restoredKey] = errors.New("restore refused")
	cache.failures[addedKey] = errors.New("removal refused")

	transaction := newTransactionJournal[string, string](cache)
	require.NoError(t, transaction.Set(ctx, restoredKey, "modified"))
	require.NoError(t, transaction.Set(ctx, addedKey, "added"))

	err := transaction.Rollback(ctx)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), restoredKey, "a rollback error must not carry the key, which may hold user data")
	assert.NotContains(t, err.Error(), addedKey, "a rollback error must not carry the key, which may hold user data")
	assert.Contains(t, err.Error(), "journal entry 1 of 2")
	assert.Contains(t, err.Error(), "journal entry 2 of 2")
}

func TestTransactionJournal_EveryMutationAfterCommitReturnsSentinel(t *testing.T) {
	loader := cache_dto.LoaderFunc[string, string](func(context.Context, string) (string, error) { return "loaded", nil })
	bulkLoader := cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, keys []string) (map[string]string, error) {
		return map[string]string{keys[0]: "loaded"}, nil
	})

	testCases := []struct {
		mutate func(ctx context.Context, transaction TransactionCache[string, string]) error
		name   string
	}{
		{name: "SetWithTTL", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			return transaction.SetWithTTL(ctx, "a", "1", time.Minute)
		}},
		{name: "Invalidate", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			return transaction.Invalidate(ctx, "a")
		}},
		{name: "Compute", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			_, _, err := transaction.Compute(ctx, "a", func(string, bool) (string, cache_dto.ComputeAction) {
				return "1", cache_dto.ComputeActionSet
			})
			return err
		}},
		{name: "ComputeIfAbsent", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			_, _, err := transaction.ComputeIfAbsent(ctx, "a", func() string { return "1" })
			return err
		}},
		{name: "ComputeIfPresent", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			_, _, err := transaction.ComputeIfPresent(ctx, "a", func(string) (string, cache_dto.ComputeAction) {
				return "1", cache_dto.ComputeActionSet
			})
			return err
		}},
		{name: "ComputeWithTTL", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			_, _, err := transaction.ComputeWithTTL(ctx, "a", func(string, bool) cache_dto.ComputeResult[string] {
				return cache_dto.ComputeResult[string]{Value: "1", Action: cache_dto.ComputeActionSet}
			})
			return err
		}},
		{name: "BulkSet", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			return transaction.BulkSet(ctx, map[string]string{"a": "1"})
		}},
		{name: "Get", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			_, err := transaction.Get(ctx, "a", loader)
			return err
		}},
		{name: "BulkGet", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			_, err := transaction.BulkGet(ctx, []string{"a"}, bulkLoader)
			return err
		}},
		{name: "SetExpiresAfter", mutate: func(ctx context.Context, transaction TransactionCache[string, string]) error {
			return transaction.SetExpiresAfter(ctx, "a", time.Minute)
		}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := t.Context()
			cache := newTestCache()
			transaction := newTransactionJournal[string, string](cache)
			require.NoError(t, transaction.Commit(ctx))

			require.ErrorIs(t, testCase.mutate(ctx, transaction), ErrTransactionFinalised)
			assert.Empty(t, cache.data, "a finalised transaction must not reach the cache")
		})
	}
}

func TestTransactionJournal_ReadsAndSettingsDelegateToInner(t *testing.T) {
	ctx := t.Context()
	cache := newTestCache()
	cache.data["a"] = "1"
	transaction := newTransactionJournal[string, string](cache)

	assert.Equal(t, []string{"a"}, slices.Collect(transaction.Keys()))
	assert.Equal(t, []string{"1"}, slices.Collect(transaction.Values()))
	entry, found, err := transaction.GetEntry(ctx, "a")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1", entry.Value)
	entry, found, err = transaction.ProbeEntry(ctx, "a")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1", entry.Value)
	assert.Equal(t, cache_dto.Stats{}, transaction.Stats())
	assert.Zero(t, transaction.GetMaximum())
	transaction.SetMaximum(10)
	assert.Zero(t, transaction.WeightedSize())
	require.NoError(t, transaction.SetRefreshableAfter(ctx, "a", time.Minute))
	_, err = transaction.Search(ctx, "query", nil)
	require.ErrorIs(t, err, ErrSearchNotSupported)
	_, err = transaction.Query(ctx, nil)
	require.ErrorIs(t, err, ErrSearchNotSupported)
	assert.False(t, transaction.SupportsSearch())
	assert.Nil(t, transaction.GetSchema())
	transaction.BulkRefresh(ctx, []string{"a"}, nil)
	for range transaction.Refresh(ctx, "a", nil) {
		require.Fail(t, "the test cache closes the refresh channel without a result")
	}
	require.NoError(t, transaction.Rollback(ctx))
}

func TestTransactionJournal_GroupsLogWritesIntoOneBatch(t *testing.T) {
	commitFailed := errors.New("log full")

	testCases := []struct {
		commitErr  error
		finish     func(ctx context.Context, transaction *transactionJournal[string, string]) error
		name       string
		wantBegins int
		wantCommit int
		wantAborts int
		wantErr    bool
	}{
		{
			name: "commit writes the batch",
			finish: func(ctx context.Context, transaction *transactionJournal[string, string]) error {
				return transaction.Commit(ctx)
			},
			wantBegins: 1,
			wantCommit: 1,
		},
		{
			name:      "a failed commit is reported and the transaction can still roll back",
			commitErr: commitFailed,
			finish: func(ctx context.Context, transaction *transactionJournal[string, string]) error {
				return transaction.Commit(ctx)
			},
			wantBegins: 1,
			wantCommit: 1,
			wantErr:    true,
		},
		{
			name: "rollback discards the batch",
			finish: func(ctx context.Context, transaction *transactionJournal[string, string]) error {
				return transaction.Rollback(ctx)
			},
			wantBegins: 1,
			wantAborts: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := t.Context()
			cache := &batchingTestCache{testCache: newTestCache(), commitErr: testCase.commitErr}
			transaction := newTransactionJournal[string, string](cache)
			require.NoError(t, transaction.Set(ctx, "a", "1"))

			err := testCase.finish(ctx, transaction)

			if testCase.wantErr {
				require.ErrorIs(t, err, commitFailed)
				require.NoError(t, transaction.Rollback(ctx), "the transaction stays open after a failed commit")
				assert.Equal(t, 2, cache.begins, "rollback opens a fresh batch to discard")
				assert.Equal(t, 1, cache.aborts)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 1, cache.batched, "the transaction writes through its batch")
			assert.Equal(t, testCase.wantBegins, cache.begins)
			assert.Equal(t, testCase.wantCommit, cache.commits)
			assert.Equal(t, testCase.wantAborts, cache.aborts)
		})
	}
}

type batchingTestCache struct {
	*testCache
	commitErr error
	begins    int
	commits   int
	aborts    int
	batched   int
}

func (c *batchingTestCache) BeginLogBatch() LogBatch[string, string] {
	c.begins++
	return &batchingTestView{testCache: c.testCache, parent: c}
}

type batchingTestView struct {
	*testCache
	parent *batchingTestCache
}

func (v *batchingTestView) Set(ctx context.Context, key string, value string, tags ...string) error {
	v.parent.batched++
	return v.testCache.Set(ctx, key, value, tags...)
}

func (v *batchingTestView) CommitLogBatch(context.Context) error {
	v.parent.commits++
	return v.parent.commitErr
}

func (v *batchingTestView) AbortLogBatch() { v.parent.aborts++ }
