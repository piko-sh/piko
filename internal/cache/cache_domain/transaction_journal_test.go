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
	"iter"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
)

type testCache struct {
	data    map[string]string
	expires map[string]int64
}

func newTestCache() *testCache {
	return &testCache{
		data:    make(map[string]string),
		expires: make(map[string]int64),
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
	c.data[key] = value
	c.expires[key] = time.Now().Add(ttl).UnixNano()
	return nil
}

func (c *testCache) Invalidate(_ context.Context, key string) error {
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
	return cache_dto.Entry[string, string]{Key: key, Value: v, ExpiresAtNano: c.expires[key]}, true, nil
}

func (c *testCache) ProbeEntry(ctx context.Context, key string) (cache_dto.Entry[string, string], bool, error) {
	return c.GetEntry(ctx, key)
}

func (c *testCache) EstimatedSize() int                                                 { return len(c.data) }
func (c *testCache) Stats() cache_dto.Stats                                             { return cache_dto.Stats{} }
func (c *testCache) Close(_ context.Context) error                                      { return nil }
func (c *testCache) GetMaximum() uint64                                                 { return 0 }
func (c *testCache) SetMaximum(_ uint64)                                                {}
func (c *testCache) WeightedSize() uint64                                               { return 0 }
func (c *testCache) SetExpiresAfter(_ context.Context, _ string, _ time.Duration) error { return nil }
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
		name      string
		setup     map[string]string
		mutate    func(t *testing.T, tx TransactionCache[string, string])
		wantAfter map[string]string
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
		name      string
		setup     map[string]string
		mutate    func(t *testing.T, tx TransactionCache[string, string])
		wantAfter map[string]string
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

func TestTransactionJournal_RollbackNoTTLUsesPlainSet(t *testing.T) {
	ctx := context.Background()
	cache := newTestCache()

	cache.data["a"] = "original"

	tx := newTransactionJournal[string, string](cache)

	require.NoError(t, tx.Set(ctx, "a", "modified"))

	require.NoError(t, tx.Rollback(ctx))

	require.Equal(t, "original", cache.data["a"])
	require.Zero(t, cache.expires["a"], "no TTL should be set when original had none")
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
		name   string
		action func(tx TransactionCache[string, string]) error
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
