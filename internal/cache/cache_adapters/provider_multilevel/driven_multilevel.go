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

package provider_multilevel

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// fieldKey is the structured log field for cache keys.
	fieldKey = "key"

	// fmtVerb is the format verb for printing cache key values.
	fmtVerb = "%v"
)

// MultiLevelAdapter implements ProviderPort by orchestrating an L1 and L2
// cache. It provides resilience against L2 failures using a circuit breaker.
type MultiLevelAdapter[K comparable, V any] struct {
	// l1Provider is the primary cache provider for fast access.
	l1Provider cache_domain.ProviderPort[K, V]

	// l2Provider is the second-level cache provider in the hierarchy.
	l2Provider cache_domain.ProviderPort[K, V]

	// l2Circuit is the circuit breaker for L2 cache operations.
	l2Circuit *gobreaker.CircuitBreaker[any]

	// name is the identifier for this adapter level.
	name string
}

var _ cache_domain.ProviderPort[any, any] = (*MultiLevelAdapter[any, any])(nil)

// GetIfPresent returns the cached value if present in L1 or L2, with
// automatic back-population from L2 to L1.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the value to retrieve from the cache.
//
// Returns V which is the cached value if found.
// Returns bool which indicates whether the key was present in the cache.
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) GetIfPresent(ctx context.Context, key K) (V, bool, error) {
	ctx, l := logger_domain.From(ctx, log)

	if cacheValue, ok, err := m.l1Provider.GetIfPresent(ctx, key); err != nil {
		l.Warn("L1 GetIfPresent failed, falling through to L2",
			logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
			logger_domain.Error(err))
	} else if ok {
		l1HitsTotal.Add(ctx, 1)
		return cacheValue, true, nil
	}

	result, err := m.l2Circuit.Execute(func() (any, error) {
		cacheValue, ok, l2Err := m.l2Provider.GetIfPresent(ctx, key)
		if l2Err != nil {
			return nil, l2Err
		}
		if !ok {
			return nil, cache_dto.ErrNotFound
		}
		return cacheValue, nil
	})

	if err == nil {
		cacheValue, ok := result.(V)
		if !ok {
			return *new(V), false, nil
		}

		_ = m.l1Provider.Set(ctx, key, cacheValue)
		l2HitsTotal.Add(ctx, 1)
		backPopulations.Add(ctx, 1)
		return cacheValue, true, nil
	}

	if !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
	}
	totalMissesTotal.Add(ctx, 1)
	return *new(V), false, nil
}

// Get retrieves a value from L1 cache, L2 cache, or loads it using the
// provided loader with write-back to both cache levels.
//
// Takes key (K) which identifies the cached value to retrieve.
// Takes loader (Loader[K, V]) which loads the value if not found in any cache.
//
// Returns V which is the cached or newly loaded value.
// Returns error when the loader fails to retrieve the value.
func (m *MultiLevelAdapter[K, V]) Get(ctx context.Context, key K, loader cache_dto.Loader[K, V]) (V, error) {
	ctx, l := logger_domain.From(ctx, log)

	if cacheValue, ok, err := m.l1Provider.GetIfPresent(ctx, key); err != nil {
		l.Warn("L1 GetIfPresent failed during Get, falling through to L2",
			logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
			logger_domain.Error(err))
	} else if ok {
		l1HitsTotal.Add(ctx, 1)
		return cacheValue, nil
	}

	if cacheValue, ok := m.tryL2Get(ctx, key); ok {
		return cacheValue, nil
	}

	wrappedLoader := m.createWriteBackLoader(loader)
	return m.l1Provider.Get(ctx, key, wrappedLoader)
}

// tryL2Get attempts to retrieve a value from L2 cache through the circuit
// breaker. On success, it back-populates the L1 cache with the retrieved value.
//
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns V which is the cached value, or the zero value if not found.
// Returns bool which indicates whether the value was found in L2 cache.
func (m *MultiLevelAdapter[K, V]) tryL2Get(ctx context.Context, key K) (V, bool) {
	l2Res, l2Err := m.l2Circuit.Execute(func() (any, error) {
		cacheValue, ok, err := m.l2Provider.GetIfPresent(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("L2 GetIfPresent during tryL2Get: %w", err)
		}
		if !ok {
			return nil, cache_dto.ErrNotFound
		}
		return cacheValue, nil
	})

	if l2Err != nil {
		if !errors.Is(l2Err, gobreaker.ErrOpenState) {
			l2ErrorsTotal.Add(ctx, 1)
		}
		totalMissesTotal.Add(ctx, 1)
		return *new(V), false
	}

	cacheValue, ok := l2Res.(V)
	if !ok {
		return *new(V), false
	}

	_ = m.l1Provider.Set(ctx, key, cacheValue)
	l2HitsTotal.Add(ctx, 1)
	backPopulations.Add(ctx, 1)
	return cacheValue, true
}

// createWriteBackLoader wraps a loader to add asynchronous L2 write-back after
// loading.
//
// Takes loader (Loader[K, V]) which provides the original load
// function.
//
// Returns cache_dto.LoaderFunc[K, V] which wraps the loader with
// L2 write-back behaviour.
func (m *MultiLevelAdapter[K, V]) createWriteBackLoader(loader cache_dto.Loader[K, V]) cache_dto.LoaderFunc[K, V] {
	return func(ctx context.Context, k K) (V, error) {
		loadedVal, loadErr := loader.Load(ctx, k)
		if loadErr != nil {
			return *new(V), loadErr
		}

		m.asyncWriteToL2(ctx, k, loadedVal)
		return loadedVal, nil
	}
}

// asyncWriteToL2 asynchronously writes a value to L2 cache through the
// circuit breaker.
//
// Takes ctx (context.Context) whose values (e.g. trace spans) are preserved
// but whose cancellation and deadline are stripped via context.WithoutCancel.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
//
// Concurrent use is safe. Spawns a goroutine that completes independently.
// The write may fail silently if the circuit breaker is open.
func (m *MultiLevelAdapter[K, V]) asyncWriteToL2(ctx context.Context, key K, value V) {
	ctx, l := logger_domain.From(ctx, log)

	detachedCtx := context.WithoutCancel(ctx)

	go func() {
		defer goroutine.RecoverPanic(detachedCtx, "cache.multilevelAsyncWriteToL2")
		_, err := m.l2Circuit.Execute(func() (any, error) {
			return nil, m.l2Provider.Set(detachedCtx, key, value)
		})
		if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
			l2ErrorsTotal.Add(detachedCtx, 1)
			l.Warn("Failed to write back to L2 after load",
				logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
				logger_domain.Error(err))
		}
	}()
}

// Set stores the value in both L1 and L2 caches using write-through policy.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional labels for cache invalidation.
//
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	ctx, l := logger_domain.From(ctx, log)

	_, err := m.l2Circuit.Execute(func() (any, error) {
		return nil, m.l2Provider.Set(ctx, key, value, tags...)
	})

	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
		l.Error("Failed to write to L2 cache, value not set in L1 for consistency",
			logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
			logger_domain.Error(err))
	}

	return m.l1Provider.Set(ctx, key, value, tags...)
}

// SetWithTTL stores the value with a TTL in both L1 and L2 caches.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which specifies how long the entry remains valid.
// Takes tags (...string) which provides optional labels for cache invalidation.
//
// Returns error when the L1 cache write fails, or when L2 fails after L1
// succeeds.
func (m *MultiLevelAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	ctx, l := logger_domain.From(ctx, log)

	var l2Err error
	_, err := m.l2Circuit.Execute(func() (any, error) {
		l2Err = m.l2Provider.SetWithTTL(ctx, key, value, ttl, tags...)
		return nil, l2Err
	})

	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
		l.Error("Failed to SetWithTTL to L2 cache",
			logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
			logger_domain.Error(err))
	}

	if l1Err := m.l1Provider.SetWithTTL(ctx, key, value, ttl, tags...); l1Err != nil {
		return fmt.Errorf("L1 SetWithTTL failed: %w", l1Err)
	}

	if l2Err != nil {
		return fmt.Errorf("L2 SetWithTTL failed (L1 succeeded): %w", l2Err)
	}

	return nil
}

// Invalidate removes the key from both L1 and L2 caches.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which specifies the cache key to remove.
//
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	_, err := m.l2Circuit.Execute(func() (any, error) {
		return nil, m.l2Provider.Invalidate(ctx, key)
	})
	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
	}

	return m.l1Provider.Invalidate(ctx, key)
}

// Compute computes and atomically updates the value in L1 cache using the
// provided function.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and
// whether it was found, and returns the new value with an action to
// perform.
//
// Returns V which is the resulting value after the computation.
// Returns bool which indicates whether a value exists after the operation.
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return m.l1Provider.Compute(ctx, key, computeFunction)
}

// ComputeIfAbsent atomically computes and stores a value in L1 if absent.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to look up or create.
// Takes computeFunction (func() V) which computes the value if the key is absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which is true if the value was already present.
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	return m.l1Provider.ComputeIfAbsent(ctx, key, computeFunction)
}

// ComputeIfPresent atomically updates an existing value in L1 cache.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(...)) which computes the new value
// from the old value.
//
// Returns V which is the computed value, or the zero value if the key was not
// present.
// Returns bool which indicates whether the key was present and updated.
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return m.l1Provider.ComputeIfPresent(ctx, key, computeFunction)
}

// ComputeWithTTL atomically computes a new value with per-call TTL control in
// L1 cache.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found flag,
// returning a ComputeResult containing the new value, action, and optional TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	return m.l1Provider.ComputeWithTTL(ctx, key, computeFunction)
}

// BulkGet retrieves multiple values from L1/L2 with fallback to the bulk
// loader.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader) which loads values for keys not found in cache.
//
// Returns map[K]V which contains the retrieved values keyed by their cache key.
// Returns error when the bulk loader fails to load missing values.
func (m *MultiLevelAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) (map[K]V, error) {
	if len(keys) == 0 {
		return make(map[K]V), nil
	}

	results := make(map[K]V, len(keys))

	l1Misses := m.bulkGetFromL1(ctx, keys, results)
	if len(l1Misses) == 0 {
		return results, nil
	}

	l2Misses := m.bulkGetFromL2(ctx, l1Misses, results)
	if len(l2Misses) == 0 {
		return results, nil
	}

	if err := m.loadAndStoreValues(ctx, l2Misses, results, bulkLoader); err != nil {
		return results, err
	}

	return results, nil
}

// bulkGetFromL1 retrieves values from L1 cache, adding hits to results and
// returning misses.
//
// Takes keys ([]K) which specifies the cache keys to look up.
// Takes results (map[K]V) which receives the found key-value pairs.
//
// Returns []K which contains the keys that were not found in L1 cache.
func (m *MultiLevelAdapter[K, V]) bulkGetFromL1(ctx context.Context, keys []K, results map[K]V) []K {
	ctx, l := logger_domain.From(ctx, log)

	misses := make([]K, 0, len(keys))
	for _, key := range keys {
		cacheValue, ok, err := m.l1Provider.GetIfPresent(ctx, key)
		if err != nil {
			l.Warn("L1 GetIfPresent failed during BulkGet",
				logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
				logger_domain.Error(err))
			misses = append(misses, key)
		} else if ok {
			results[key] = cacheValue
			l1HitsTotal.Add(ctx, 1)
		} else {
			misses = append(misses, key)
		}
	}
	return misses
}

// bulkGetFromL2 retrieves values from L2 cache through the circuit breaker.
// Adds hits to results, back-populates L1, and returns remaining misses.
//
// Takes keys ([]K) which specifies the cache keys to look up.
// Takes results (map[K]V) which receives the values found in L2.
//
// Returns []K which contains the keys that were not found in L2.
func (m *MultiLevelAdapter[K, V]) bulkGetFromL2(ctx context.Context, keys []K, results map[K]V) []K {
	l2Hits, l2Err := m.fetchFromL2(ctx, keys)

	if l2Err != nil {
		m.handleL2Error(ctx, l2Err, len(keys))
		return keys
	}

	m.processL2Hits(ctx, l2Hits, results)

	return m.collectRemainingMisses(keys, l2Hits)
}

// fetchFromL2 retrieves values from L2 cache through the circuit breaker.
//
// Takes keys ([]K) which specifies the cache keys to look up.
//
// Returns map[K]V which contains the values found in the L2 cache.
// Returns error when the circuit breaker rejects the request.
func (m *MultiLevelAdapter[K, V]) fetchFromL2(ctx context.Context, keys []K) (map[K]V, error) {
	l2Result, l2Err := m.l2Circuit.Execute(func() (any, error) {
		l2Values := make(map[K]V)
		for _, key := range keys {
			cacheValue, ok, err := m.l2Provider.GetIfPresent(ctx, key)
			if err != nil {
				return nil, fmt.Errorf("L2 GetIfPresent during fetchFromL2: %w", err)
			}
			if ok {
				l2Values[key] = cacheValue
			}
		}
		return l2Values, nil
	})

	if l2Err != nil {
		return nil, l2Err
	}

	l2Hits, ok := l2Result.(map[K]V)
	if !ok {
		return make(map[K]V), nil
	}
	return l2Hits, nil
}

// handleL2Error records metrics for L2 errors during BulkGet.
//
// Takes err (error) which is the error encountered from the L2 cache.
// Takes missCount (int) which is the number of L1 cache misses to log.
func (*MultiLevelAdapter[K, V]) handleL2Error(ctx context.Context, err error, missCount int) {
	ctx, l := logger_domain.From(ctx, log)

	if !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
		l.Warn("L2 cache unavailable during BulkGet",
			logger_domain.Int("l1_miss_count", missCount),
			logger_domain.Error(err))
	}
}

// processL2Hits adds L2 hits to results and back-populates L1 asynchronously.
//
// Takes l2Hits (map[K]V) which contains the cache hits from the L2 provider.
// Takes results (map[K]V) which is the destination map for collected results.
//
// Safe for concurrent use. Spawns a goroutine to back-populate L1 cache
// without blocking the caller.
func (m *MultiLevelAdapter[K, V]) processL2Hits(ctx context.Context, l2Hits map[K]V, results map[K]V) {
	if len(l2Hits) == 0 {
		return
	}

	for key, cacheValue := range l2Hits {
		results[key] = cacheValue
		l2HitsTotal.Add(ctx, 1)
		backPopulations.Add(ctx, 1)
	}

	detachedCtx := context.WithoutCancel(ctx)

	go func(hits map[K]V) {
		defer goroutine.RecoverPanic(detachedCtx, "cache.multilevelBackPopulateL1")
		for k, v := range hits {
			_ = m.l1Provider.Set(detachedCtx, k, v)
		}
	}(l2Hits)
}

// collectRemainingMisses returns keys that were not found in L2 hits.
//
// Takes keys ([]K) which is the list of keys to check.
// Takes l2Hits (map[K]V) which contains the keys found in L2 cache.
//
// Returns []K which contains the keys not present in l2Hits.
func (*MultiLevelAdapter[K, V]) collectRemainingMisses(keys []K, l2Hits map[K]V) []K {
	misses := make([]K, 0, len(keys))
	for _, key := range keys {
		if _, found := l2Hits[key]; !found {
			misses = append(misses, key)
		}
	}
	return misses
}

// loadAndStoreValues loads missing keys using the bulk loader and stores the
// results in both caches.
//
// Takes keys ([]K) which specifies the cache keys to load.
// Takes results (map[K]V) which receives the loaded values.
// Takes bulkLoader (BulkLoader) which provides the data source for cache
// misses.
//
// Returns error when the bulk loader fails to load the requested keys.
func (m *MultiLevelAdapter[K, V]) loadAndStoreValues(
	ctx context.Context,
	keys []K,
	results map[K]V,
	bulkLoader cache_dto.BulkLoader[K, V],
) error {
	totalMissesTotal.Add(ctx, int64(len(keys)))

	loaded, err := bulkLoader.BulkLoad(ctx, keys)
	if err != nil {
		return fmt.Errorf("bulk loader failed: %w", err)
	}

	maps.Copy(results, loaded)

	if len(loaded) > 0 {
		m.storeLoadedValues(ctx, loaded)
	}

	return nil
}

// storeLoadedValues stores values in L1 immediately and L2 asynchronously.
//
// Takes values (map[K]V) which contains the key-value pairs to store.
//
// Safe for concurrent use. Spawns a goroutine to write values to L2 via
// the circuit breaker. L1 and L2 providers handle their own
// synchronisation.
func (m *MultiLevelAdapter[K, V]) storeLoadedValues(ctx context.Context, values map[K]V) {
	ctx, l := logger_domain.From(ctx, log)

	for k, v := range values {
		_ = m.l1Provider.Set(ctx, k, v)
	}

	detachedCtx := context.WithoutCancel(ctx)

	go func(vals map[K]V) {
		defer goroutine.RecoverPanic(detachedCtx, "cache.multilevelStoreToL2")
		_, err := m.l2Circuit.Execute(func() (any, error) {
			for k, v := range vals {
				if setErr := m.l2Provider.Set(detachedCtx, k, v); setErr != nil {
					return nil, setErr
				}
			}
			return nil, nil
		})
		if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
			l2ErrorsTotal.Add(detachedCtx, 1)
			l.Warn("Failed to write loaded values to L2 cache",
				logger_domain.Int("value_count", len(vals)),
				logger_domain.Error(err))
		}
	}(values)
}

// BulkSet stores multiple key-value pairs in both L1 and L2 caches.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional cache tags for the entries.
//
// Returns error when the L1 cache fails, or when L2 fails but L1 succeeds.
func (m *MultiLevelAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	if len(items) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)

	var l2Err error
	_, err := m.l2Circuit.Execute(func() (any, error) {
		l2Err = m.l2Provider.BulkSet(ctx, items, tags...)
		return nil, l2Err
	})

	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
		l.Error("Failed to BulkSet to L2 cache",
			logger_domain.Int("item_count", len(items)),
			logger_domain.Error(err))
	}

	if l1Err := m.l1Provider.BulkSet(ctx, items, tags...); l1Err != nil {
		return fmt.Errorf("L1 BulkSet failed: %w", l1Err)
	}

	if l2Err != nil {
		return fmt.Errorf("L2 BulkSet failed (L1 succeeded): %w", l2Err)
	}

	return nil
}

// InvalidateByTags removes all entries with matching tags from both caches.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes tags (...string) which specifies the cache tags to match for removal.
//
// Returns int which is the count of entries invalidated from the L1 cache.
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	_, err := m.l2Circuit.Execute(func() (any, error) {
		_, l2Err := m.l2Provider.InvalidateByTags(ctx, tags...)
		return nil, l2Err
	})
	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
	}

	invalidatedCount, l1Err := m.l1Provider.InvalidateByTags(ctx, tags...)
	return invalidatedCount, l1Err
}

// InvalidateAll clears all entries from both L1 and L2 caches.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) InvalidateAll(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	_, err := m.l2Circuit.Execute(func() (any, error) {
		return nil, m.l2Provider.InvalidateAll(ctx)
	})
	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
		l.Warn("Failed to InvalidateAll on L2 cache", logger_domain.Error(err))
	}

	return m.l1Provider.InvalidateAll(ctx)
}

// All returns an iterator over all key-value pairs in the L1 cache.
//
// Returns iter.Seq2[K, V] which yields each key-value pair from
// L1.
func (m *MultiLevelAdapter[K, V]) All() iter.Seq2[K, V] {
	return m.l1Provider.All()
}

// Keys returns an iterator over all keys in the L1 cache.
//
// Returns iter.Seq[K] which yields each key from L1.
func (m *MultiLevelAdapter[K, V]) Keys() iter.Seq[K] {
	return m.l1Provider.Keys()
}

// Values returns an iterator over all values in the L1 cache.
//
// Returns iter.Seq[V] which yields each value from L1.
func (m *MultiLevelAdapter[K, V]) Values() iter.Seq[V] {
	return m.l1Provider.Values()
}

// EstimatedSize returns the approximate number of entries in L1 cache.
//
// Returns int which is the estimated entry count.
func (m *MultiLevelAdapter[K, V]) EstimatedSize() int {
	return m.l1Provider.EstimatedSize()
}

// Stats returns cache statistics from the L1 cache.
//
// Returns cache_dto.Stats which contains the current cache statistics.
func (m *MultiLevelAdapter[K, V]) Stats() cache_dto.Stats {
	return m.l1Provider.Stats()
}

// Close releases resources held by both L1 and L2 caches.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when resources cannot be released cleanly.
func (m *MultiLevelAdapter[K, V]) Close(ctx context.Context) error {
	l1Err := m.l1Provider.Close(ctx)
	l2Err := m.l2Provider.Close(ctx)
	return errors.Join(l1Err, l2Err)
}

// BulkRefresh asynchronously refreshes multiple keys using the bulk loader
// via L1.
//
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which provides the function to load values.
func (m *MultiLevelAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) {
	m.l1Provider.BulkRefresh(ctx, keys, bulkLoader)
}

// Refresh asynchronously reloads a single key via L1.
//
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which loads the fresh value.
//
// Returns <-chan cache_dto.LoadResult[V] which delivers the
// refresh result.
func (m *MultiLevelAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V] {
	return m.l1Provider.Refresh(ctx, key, loader)
}

// GetEntry returns the full entry metadata for a key from L1.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to retrieve.
//
// Returns cache_dto.Entry[K, V] which contains the entry with
// metadata.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	return m.l1Provider.GetEntry(ctx, key)
}

// ProbeEntry returns entry metadata without affecting statistics.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to probe.
//
// Returns cache_dto.Entry[K, V] which contains the entry with
// metadata.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	return m.l1Provider.ProbeEntry(ctx, key)
}

// GetMaximum returns the maximum capacity of the L1 cache.
//
// Returns uint64 which is the maximum number of entries the L1 cache can hold.
func (m *MultiLevelAdapter[K, V]) GetMaximum() uint64 {
	return m.l1Provider.GetMaximum()
}

// SetMaximum sets the maximum capacity for L1 cache only.
//
// Takes size (uint64) which specifies the maximum number of items to store.
func (m *MultiLevelAdapter[K, V]) SetMaximum(size uint64) {
	m.l1Provider.SetMaximum(size)
}

// WeightedSize returns the weighted size of entries in L1 cache.
//
// Returns uint64 which is the total weighted size of cached entries.
func (m *MultiLevelAdapter[K, V]) WeightedSize() uint64 {
	return m.l1Provider.WeightedSize()
}

// SetExpiresAfter manually sets the expiration time for a key in L1.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the new expiration time.
//
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	return m.l1Provider.SetExpiresAfter(ctx, key, expiresAfter)
}

// SetRefreshableAfter manually sets the refresh time for a key in L1.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes refreshableAfter (time.Duration) which specifies when the entry
// becomes eligible for background refresh.
//
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) SetRefreshableAfter(ctx context.Context, key K, refreshableAfter time.Duration) error {
	return m.l1Provider.SetRefreshableAfter(ctx, key, refreshableAfter)
}

// Search performs full-text search across indexed fields.
// Delegates to the L1 provider.
//
// Takes ctx (context.Context) for cancellation.
// Takes query (string) which is the search query.
// Takes opts (*cache_dto.SearchOptions) which configures the
// search.
//
// Returns cache_dto.SearchResult[K, V] which contains the
// search results from the L1 provider.
// Returns error when the search operation fails.
func (m *MultiLevelAdapter[K, V]) Search(ctx context.Context, query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	return m.l1Provider.Search(ctx, query, opts)
}

// Query performs structured filtering, sorting, and pagination.
// Delegates to the L1 provider.
//
// Takes ctx (context.Context) for cancellation.
// Takes opts (*cache_dto.QueryOptions) which configures the
// query.
//
// Returns cache_dto.SearchResult[K, V] which contains the
// query results from the L1 provider.
// Returns error when the query operation fails.
func (m *MultiLevelAdapter[K, V]) Query(ctx context.Context, opts *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error) {
	return m.l1Provider.Query(ctx, opts)
}

// SupportsSearch returns whether the L1 provider supports search operations.
//
// Returns bool from L1 provider's SupportsSearch.
func (m *MultiLevelAdapter[K, V]) SupportsSearch() bool {
	return m.l1Provider.SupportsSearch()
}

// GetSchema returns the search schema from the L1 provider.
//
// Returns *cache_dto.SearchSchema from L1 provider.
func (m *MultiLevelAdapter[K, V]) GetSchema() *cache_dto.SearchSchema {
	return m.l1Provider.GetSchema()
}

// NewMultiLevelAdapter creates a new multi-level cache provider.
//
// Takes ctx (context.Context) which carries logging context for circuit
// breaker initialisation.
// Takes name (string) which identifies this adapter for logging.
// Takes l1 (ProviderPort[K, V]) which is the fast local cache.
// Takes l2 (ProviderPort[K, V]) which is the remote backing
// cache.
// Takes cbConfig (Config) which configures the L2 circuit breaker.
//
// Returns *MultiLevelAdapter[K, V] which orchestrates L1 and L2
// caches.
func NewMultiLevelAdapter[K comparable, V any](
	ctx context.Context,
	name string,
	l1 cache_domain.ProviderPort[K, V],
	l2 cache_domain.ProviderPort[K, V],
	cbConfig Config,
) *MultiLevelAdapter[K, V] {
	return &MultiLevelAdapter[K, V]{
		l1Provider: l1,
		l2Provider: l2,
		l2Circuit:  newCircuitBreaker(ctx, name, cbConfig.MaxConsecutiveFailures, cbConfig.OpenStateTimeout),
		name:       name,
	}
}
