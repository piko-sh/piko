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
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// fieldKey is the structured log field for cache keys.
	fieldKey = "key"

	// fmtVerb is the format verb for printing cache key values.
	fmtVerb = "%v"
)

// MultiLevelAdapter implements ProviderPort by orchestrating an L1 and L2 cache. It
// provides resilience against L2 failures using a circuit breaker.
type MultiLevelAdapter[K comparable, V any] struct {
	// l1Provider is the primary cache provider for fast access.
	l1Provider cache_domain.ProviderPort[K, V]

	// l2Provider is the second-level cache provider in the hierarchy.
	l2Provider cache_domain.ProviderPort[K, V]

	// l2Circuit is the circuit breaker for L2 cache operations.
	l2Circuit *gobreaker.CircuitBreaker[any]

	// backgroundSemaphore bounds the number of concurrent background workers
	// (back-population, async write-back). A nil semaphore means unbounded.
	backgroundSemaphore chan struct{}

	// name is the identifier for this adapter level.
	name string

	// closeOnce guards single execution of Close so the underlying providers are only closed
	// once.
	closeOnce sync.Once
}

// computeOutcome holds what a level reports after a compute.
type computeOutcome[V any] struct {
	// value is the value present after the compute.
	value V

	// present reports whether a value is present after the compute.
	present bool
}

// levelEntry is an entry together with the level it was read from.
type levelEntry[K comparable, V any] struct {
	// Entry is the entry as the level reported it.
	cache_dto.Entry[K, V]

	// fromL2 records that the entry came from L2 after an L1 miss.
	fromL2 bool
}

var (
	_ cache_domain.ProviderPort[any, any] = (*MultiLevelAdapter[any, any])(nil)
)

// GetIfPresent returns the cached value if present in L1 or L2, with automatic
// back-population from L2 to L1.
//
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

	cacheValue, ok := m.tryL2Get(ctx, key)
	return cacheValue, ok, nil
}

// Get retrieves a value from L1 cache, L2 cache, or loads it using the provided loader
// with write-back to both cache levels.
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

// tryL2Get attempts to retrieve a value from L2 cache through the circuit breaker. On
// success, it back-populates L1 with the retrieved value and the TTL L2 reported when it
// read the entry.
//
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns V which is the cached value, or the zero value if not found.
// Returns bool which indicates whether the value was found in L2 cache.
func (m *MultiLevelAdapter[K, V]) tryL2Get(ctx context.Context, key K) (V, bool) {
	var entry cache_dto.Entry[K, V]
	_, l2Err := m.l2Circuit.Execute(func() (any, error) {
		found, ok, err := m.l2Provider.GetEntry(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("L2 GetEntry during tryL2Get: %w", err)
		}
		if !ok {
			return nil, cache_dto.ErrNotFound
		}
		entry = found
		return nil, nil
	})

	if l2Err != nil {
		if !errors.Is(l2Err, gobreaker.ErrOpenState) {
			l2ErrorsTotal.Add(ctx, 1)
		}
		totalMissesTotal.Add(ctx, 1)
		return *new(V), false
	}

	m.backfillL1(ctx, entry)
	l2HitsTotal.Add(ctx, 1)
	backPopulations.Add(ctx, 1)
	return entry.Value, true
}

// backfillL1 copies an entry read from L2 into L1.
//
// It uses the TTL at L2's snapshot. Time spent before the L1 write can make the copy
// outlive the L2 entry. An entry with no time left at the snapshot is not copied. An
// entry without expiry is written as Set writes it.
//
// Takes entry (cache_dto.Entry[K, V]) which is the entry as L2 reported it.
func (m *MultiLevelAdapter[K, V]) backfillL1(ctx context.Context, entry cache_dto.Entry[K, V]) {
	ctx, l := logger_domain.From(ctx, log)

	var err error
	if entry.ExpiresAtNano == 0 || entry.ExpiresAtNano == math.MaxInt64 {
		err = m.l1Provider.Set(ctx, entry.Key, entry.Value)
	} else {
		remaining := remainingTTL(entry)
		if remaining <= 0 {
			return
		}
		err = m.l1Provider.SetWithTTL(ctx, entry.Key, entry.Value, remaining)
	}
	if err != nil {
		l.Warn("Failed to back-populate L1 from L2",
			logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, entry.Key)),
			logger_domain.Error(err))
	}
}

// createWriteBackLoader wraps a loader to add asynchronous L2 write-back after loading.
//
// Takes loader (Loader[K, V]) which provides the original load function.
//
// Returns cache_dto.LoaderFunc[K, V] which wraps the loader with L2 write-back behaviour.
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

// asyncWriteToL2 asynchronously writes a value to L2 cache through the circuit breaker.
//
// The write keeps the trace values of ctx but not its cancellation or pooled request
// carrier.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
func (m *MultiLevelAdapter[K, V]) asyncWriteToL2(ctx context.Context, key K, value V) {
	m.spawnBackgroundWorker(ctx, "cache.multilevelAsyncWriteToL2", func(backgroundCtx context.Context) {
		_, err := m.l2Circuit.Execute(func() (any, error) {
			return nil, m.l2Provider.Set(backgroundCtx, key, value)
		})
		if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
			l2ErrorsTotal.Add(backgroundCtx, 1)
			_, l := logger_domain.From(backgroundCtx, log)
			l.Warn("Failed to write back to L2 after load",
				logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
				logger_domain.Error(err))
		}
	})
}

// spawnBackgroundWorker launches fn in a goroutine bounded by the adapter's
// backgroundSemaphore. Acquisition of the semaphore happens synchronously on the caller,
// so a saturated pool back-pressures the calling path rather than piling up unbounded
// goroutines.
//
// Takes component (string) which identifies the worker for panic logging.
// Takes fn (func(context.Context)) which is the work to execute on the background
// goroutine, given the detached context.
func (m *MultiLevelAdapter[K, V]) spawnBackgroundWorker(ctx context.Context, component string, fn func(context.Context)) {
	backgroundCtx := daemon_dto.DetachRequestContext(ctx)
	if m.backgroundSemaphore != nil {
		m.backgroundSemaphore <- struct{}{}
	}
	go func() {
		defer goroutine.RecoverPanic(backgroundCtx, component)
		if m.backgroundSemaphore != nil {
			defer func() { <-m.backgroundSemaphore }()
		}
		fn(backgroundCtx)
	}()
}

// Set attempts an L2 write and stores the value in L1 even if L2 is unavailable.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional labels for cache invalidation.
//
// Returns error when the L1 write fails. An L2 failure is logged.
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

// SetWithTTL writes the value with a TTL to L1 and attempts the same write in L2.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which specifies how long the entry remains valid.
// Takes tags (...string) which provides optional labels for cache invalidation.
//
// Returns error when the L1 write fails, or when L1 succeeds but the L2 write failed.
func (m *MultiLevelAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	var l2Err error
	_, err := m.l2Circuit.Execute(func() (any, error) {
		l2Err = m.l2Provider.SetWithTTL(ctx, key, value, ttl, tags...)
		return nil, l2Err
	})

	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
	}

	if l1Err := m.l1Provider.SetWithTTL(ctx, key, value, ttl, tags...); l1Err != nil {
		return fmt.Errorf("L1 SetWithTTL failed: %w", l1Err)
	}

	if l2Err != nil {
		return fmt.Errorf("L2 SetWithTTL failed (L1 succeeded): %w", l2Err)
	}

	return nil
}

// Invalidate removes the key from L1 and attempts to remove it from L2.
//
// Takes key (K) which specifies the cache key to remove.
//
// Returns error when the L1 invalidation fails. An L2 failure is recorded separately.
func (m *MultiLevelAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	_, err := m.l2Circuit.Execute(func() (any, error) {
		return nil, m.l2Provider.Invalidate(ctx, key)
	})
	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
	}

	return m.l1Provider.Invalidate(ctx, key)
}

// Compute atomically computes a new value for a key.
//
// L2 is the source of truth, so the compute runs there and L1 then drops its copy; the
// next read fills L1 from L2 with the TTL L2 has left. When L2 cannot be reached the
// compute runs on L1 alone. The compute function may run more than once, because a remote
// L2 retries it when the key changes underneath, so it must not have side effects.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(V, bool) (V, cache_dto.ComputeAction)) which computes a new
// value and action from the old value and presence flag.
//
// Returns V which is the resulting value after the computation.
// Returns bool which indicates whether a value exists after the operation.
// Returns error when the compute fails on the level that ran it.
func (m *MultiLevelAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return m.computeOnLevels(ctx, key, "Compute", func(level cache_domain.ProviderPort[K, V]) (V, bool, error) {
		return level.Compute(ctx, key, computeFunction)
	})
}

// ComputeIfAbsent atomically computes and stores a value if the key is absent.
//
// L2 is the source of truth, as for Compute, and L1 falls back to running it alone when
// L2 cannot be reached. The compute function may run more than once on a remote L2, so it
// must not have side effects.
//
// Takes key (K) which identifies the cache entry to look up or create.
// Takes computeFunction (func() V) which computes the value if the key is absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which reports whether a value is present afterwards.
// Returns error when the compute fails on the level that ran it.
func (m *MultiLevelAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	return m.computeOnLevels(ctx, key, "ComputeIfAbsent", func(level cache_domain.ProviderPort[K, V]) (V, bool, error) {
		return level.ComputeIfAbsent(ctx, key, computeFunction)
	})
}

// ComputeIfPresent atomically updates an existing value.
//
// L2 is the source of truth, as for Compute, and L1 falls back to running it alone when
// L2 cannot be reached. The compute function may run more than once on a remote L2, so it
// must not have side effects.
//
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(V) (V, cache_dto.ComputeAction)) which computes the new
// value and action from the old value.
//
// Returns V which is the computed value, or the zero value if the key was not present.
// Returns bool which indicates whether the key was present and updated.
// Returns error when the compute fails on the level that ran it.
func (m *MultiLevelAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return m.computeOnLevels(ctx, key, "ComputeIfPresent", func(level cache_domain.ProviderPort[K, V]) (V, bool, error) {
		return level.ComputeIfPresent(ctx, key, computeFunction)
	})
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// L2 is the source of truth, so the compute runs there and L1 then drops its copy.
// Mirroring a result could apply the wrong TTL or reorder concurrent computes. The next
// read backfills L1 from L2. If L2 fails, the compute runs on L1. A remote L2 may retry
// the compute function, so it must not have side effects.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(V, bool) cache_dto.ComputeResult[V]) which computes the new
// value, action, and optional TTL from the old value and presence flag.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the compute fails on the level that ran it.
func (m *MultiLevelAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	return m.computeOnLevels(ctx, key, "ComputeWithTTL", func(level cache_domain.ProviderPort[K, V]) (V, bool, error) {
		return level.ComputeWithTTL(ctx, key, computeFunction)
	})
}

// computeOnLevels runs a compute on L2 and drops the L1 copy of the key afterwards, or
// runs it on L1 alone when L2 cannot be reached.
//
// Once L2 has committed the compute its result is returned even when L1 cannot drop its
// copy, because a caller that retried on that error would apply the compute twice. The L1
// failure is logged instead.
//
// Takes key (K) which identifies the entry.
// Takes operation (string) which names the compute in log output.
// Takes compute (func(cache_domain.ProviderPort[K, V]) (V, bool, error)) which runs the
// compute on the given level.
//
// Returns V which is the value present afterwards.
// Returns bool which reports whether a value is present afterwards.
// Returns error when the compute fails on the level that ran it.
func (m *MultiLevelAdapter[K, V]) computeOnLevels(
	ctx context.Context,
	key K,
	operation string,
	compute func(level cache_domain.ProviderPort[K, V]) (V, bool, error),
) (V, bool, error) {
	ctx, l := logger_domain.From(ctx, log)

	var computed computeOutcome[V]
	_, err := m.l2Circuit.Execute(func() (any, error) {
		value, present, l2Err := compute(m.l2Provider)
		computed = computeOutcome[V]{value: value, present: present}
		return nil, l2Err
	})
	if err != nil {
		if !errors.Is(err, gobreaker.ErrOpenState) {
			l2ErrorsTotal.Add(ctx, 1)
			l.Warn("L2 compute failed, computing on L1 alone",
				logger_domain.String("operation", operation),
				logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
				logger_domain.Error(err))
		}
		return compute(m.l1Provider)
	}

	if invalidateErr := m.l1Provider.Invalidate(ctx, key); invalidateErr != nil {
		l.Warn("Failed to drop the L1 copy after an L2 compute; L1 may serve the old value until it expires",
			logger_domain.String("operation", operation),
			logger_domain.String(fieldKey, fmt.Sprintf(fmtVerb, key)),
			logger_domain.Error(invalidateErr))
	}
	return computed.value, computed.present, nil
}

// BulkGet retrieves multiple values from L1/L2 with fallback to the bulk loader.
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

// bulkGetFromL1 retrieves values from L1 cache, adding hits to results and returning
// misses.
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

// bulkGetFromL2 retrieves values from L2 cache through the circuit breaker. Adds hits to
// results, back-populates L1, and returns remaining misses.
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
// Returns map[K]cache_dto.Entry[K, V] which contains the entries found in the L2 cache,
// with the expiry L2 reports for each.
// Returns error when the circuit breaker rejects the request or an L2 read fails.
func (m *MultiLevelAdapter[K, V]) fetchFromL2(ctx context.Context, keys []K) (map[K]cache_dto.Entry[K, V], error) {
	l2Hits := make(map[K]cache_dto.Entry[K, V])
	_, l2Err := m.l2Circuit.Execute(func() (any, error) {
		for _, key := range keys {
			entry, ok, err := m.l2Provider.GetEntry(ctx, key)
			if err != nil {
				return nil, fmt.Errorf("L2 GetEntry during fetchFromL2: %w", err)
			}
			if ok {
				l2Hits[key] = entry
			}
		}
		return nil, nil
	})

	if l2Err != nil {
		return nil, l2Err
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

// processL2Hits adds L2 hits to results and back-populates L1 asynchronously, using the
// TTL reported at each entry's L2 snapshot.
//
// Takes l2Hits (map[K]cache_dto.Entry[K, V]) which contains the cache hits from the L2
// provider.
// Takes results (map[K]V) which is the destination map for collected results.
func (m *MultiLevelAdapter[K, V]) processL2Hits(ctx context.Context, l2Hits map[K]cache_dto.Entry[K, V], results map[K]V) {
	if len(l2Hits) == 0 {
		return
	}

	for key, entry := range l2Hits {
		results[key] = entry.Value
		l2HitsTotal.Add(ctx, 1)
		backPopulations.Add(ctx, 1)
	}

	m.spawnBackgroundWorker(ctx, "cache.multilevelBackPopulateL1", func(backgroundCtx context.Context) {
		for _, entry := range l2Hits {
			m.backfillL1(backgroundCtx, entry)
		}
	})
}

// collectRemainingMisses returns keys that were not found in L2 hits.
//
// Takes keys ([]K) which is the list of keys to check.
// Takes l2Hits (map[K]cache_dto.Entry[K, V]) which contains the keys found in L2 cache.
//
// Returns []K which contains the keys not present in l2Hits.
func (*MultiLevelAdapter[K, V]) collectRemainingMisses(keys []K, l2Hits map[K]cache_dto.Entry[K, V]) []K {
	misses := make([]K, 0, len(keys))
	for _, key := range keys {
		if _, found := l2Hits[key]; !found {
			misses = append(misses, key)
		}
	}
	return misses
}

// loadAndStoreValues loads missing keys using the bulk loader and stores the results in
// both caches.
//
// Takes keys ([]K) which specifies the cache keys to load.
// Takes results (map[K]V) which receives the loaded values.
// Takes bulkLoader (BulkLoader) which provides the data source for cache misses.
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
func (m *MultiLevelAdapter[K, V]) storeLoadedValues(ctx context.Context, values map[K]V) {
	for k, v := range values {
		_ = m.l1Provider.Set(ctx, k, v)
	}

	vals := values

	m.spawnBackgroundWorker(ctx, "cache.multilevelStoreToL2", func(backgroundCtx context.Context) {
		_, err := m.l2Circuit.Execute(func() (any, error) {
			for k, v := range vals {
				if setErr := m.l2Provider.Set(backgroundCtx, k, v); setErr != nil {
					return nil, setErr
				}
			}
			return nil, nil
		})
		if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
			l2ErrorsTotal.Add(backgroundCtx, 1)
			_, l := logger_domain.From(backgroundCtx, log)
			l.Warn("Failed to write loaded values to L2 cache",
				logger_domain.Int("value_count", len(vals)),
				logger_domain.Error(err))
		}
	})
}

// BulkSet writes multiple entries to L1 and attempts the same write in L2.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional cache tags for the entries.
//
// Returns error when the L1 write fails, or when L1 succeeds but the L2 write failed.
func (m *MultiLevelAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	if len(items) == 0 {
		return nil
	}

	var l2Err error
	_, err := m.l2Circuit.Execute(func() (any, error) {
		l2Err = m.l2Provider.BulkSet(ctx, items, tags...)
		return nil, l2Err
	})

	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
	}

	if l1Err := m.l1Provider.BulkSet(ctx, items, tags...); l1Err != nil {
		return fmt.Errorf("L1 BulkSet failed: %w", l1Err)
	}

	if l2Err != nil {
		return fmt.Errorf("L2 BulkSet failed (L1 succeeded): %w", l2Err)
	}

	return nil
}

// InvalidateByTags removes matching entries from L1 and attempts to remove them from L2.
//
// Takes tags (...string) which specifies the cache tags to match for removal.
//
// Returns int which is the count of entries invalidated from the L1 cache.
// Returns error when the L1 invalidation fails. An L2 failure is recorded separately.
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

// InvalidateAll clears L1 and attempts to clear L2.
//
// Returns error when the L1 clear fails. An L2 failure is logged.
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
// Returns iter.Seq2[K, V] which yields each key-value pair from L1.
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
// Returns error which joins the L1 and L2 close errors on the first call. Later calls
// return nil without closing either provider again.
func (m *MultiLevelAdapter[K, V]) Close(ctx context.Context) error {
	var closeErr error
	m.closeOnce.Do(func() {
		l1Err := m.l1Provider.Close(ctx)
		l2Err := m.l2Provider.Close(ctx)
		closeErr = errors.Join(l1Err, l2Err)
	})
	return closeErr
}

// BulkRefresh asynchronously refreshes multiple keys using the bulk loader via L1.
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
// Returns <-chan cache_dto.LoadResult[V] which delivers the refresh result.
func (m *MultiLevelAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V] {
	return m.l1Provider.Refresh(ctx, key, loader)
}

// GetEntry returns the full entry metadata for a key from L1, or from L2 on an L1 miss,
// back-populating L1 as any read does.
//
// Takes key (K) which identifies the entry to retrieve.
//
// Returns cache_dto.Entry[K, V] which contains the entry with metadata.
// Returns bool which indicates whether the key was found.
// Returns error when L1 fails or L2 fails while its circuit is closed.
func (m *MultiLevelAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	entry, found, err := m.entryFromLevels(ctx, func(level cache_domain.ProviderPort[K, V]) (cache_dto.Entry[K, V], bool, error) {
		return level.GetEntry(ctx, key)
	})
	if err == nil && found && entry.fromL2 {
		m.backfillL1(ctx, entry.Entry)
	}
	return entry.Entry, found, err
}

// ProbeEntry returns entry metadata from L1, or from L2 on an L1 miss, without affecting
// statistics or back-populating L1.
//
// Takes key (K) which identifies the entry to probe.
//
// Returns cache_dto.Entry[K, V] which contains the entry with metadata.
// Returns bool which indicates whether the key was found.
// Returns error when L1 fails or L2 fails while its circuit is closed.
func (m *MultiLevelAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	entry, found, err := m.entryFromLevels(ctx, func(level cache_domain.ProviderPort[K, V]) (cache_dto.Entry[K, V], bool, error) {
		return level.ProbeEntry(ctx, key)
	})
	return entry.Entry, found, err
}

// entryFromLevels reads an entry from L1 and, on a miss, from L2, since computes leave L2
// holding entries that L1 has dropped.
//
// Takes read (func(cache_domain.ProviderPort[K, V]) (cache_dto.Entry[K, V], bool, error))
// which reads the entry from the given level.
//
// Returns levelEntry[K, V] which is the entry and the level it came from.
// Returns bool which indicates whether the key was found.
// Returns error when L1 fails or L2 fails while its circuit is closed; an open circuit
// reports a miss.
func (m *MultiLevelAdapter[K, V]) entryFromLevels(
	ctx context.Context,
	read func(level cache_domain.ProviderPort[K, V]) (cache_dto.Entry[K, V], bool, error),
) (levelEntry[K, V], bool, error) {
	entry, found, err := read(m.l1Provider)
	if err != nil || found {
		return levelEntry[K, V]{Entry: entry, fromL2: false}, found, err
	}

	var l2Entry cache_dto.Entry[K, V]
	var l2Found bool
	_, err = m.l2Circuit.Execute(func() (any, error) {
		var l2Err error
		l2Entry, l2Found, l2Err = read(m.l2Provider)
		return nil, l2Err
	})
	if errors.Is(err, gobreaker.ErrOpenState) {
		return levelEntry[K, V]{}, false, nil
	}
	if err != nil {
		l2ErrorsTotal.Add(ctx, 1)
		return levelEntry[K, V]{}, false, fmt.Errorf("reading entry from L2: %w", err)
	}
	return levelEntry[K, V]{Entry: l2Entry, fromL2: true}, l2Found, nil
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

// SetExpiresAfter attempts to set the key's expiry in L2 and then updates L1.
//
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the new expiration time.
// cache_dto.NoExpiry removes the TTL. Other nonpositive values do nothing.
//
// Returns error when the L1 update fails, or when L1 succeeds but the L2 update failed.
func (m *MultiLevelAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	if expiresAfter <= 0 {
		return nil
	}

	var l2Err error
	_, err := m.l2Circuit.Execute(func() (any, error) {
		l2Err = m.l2Provider.SetExpiresAfter(ctx, key, expiresAfter)
		return nil, l2Err
	})
	if err != nil && !errors.Is(err, gobreaker.ErrOpenState) {
		l2ErrorsTotal.Add(ctx, 1)
	}

	if l1Err := m.l1Provider.SetExpiresAfter(ctx, key, expiresAfter); l1Err != nil {
		return fmt.Errorf("L1 SetExpiresAfter failed: %w", l1Err)
	}
	if l2Err != nil {
		return fmt.Errorf("L2 SetExpiresAfter failed (L1 succeeded): %w", l2Err)
	}
	return nil
}

// SetRefreshableAfter manually sets the refresh time for a key in L1.
//
// Takes key (K) which identifies the cache entry to update.
// Takes refreshableAfter (time.Duration) which specifies when the entry becomes eligible
// for background refresh.
//
// Returns error when the operation fails.
func (m *MultiLevelAdapter[K, V]) SetRefreshableAfter(ctx context.Context, key K, refreshableAfter time.Duration) error {
	return m.l1Provider.SetRefreshableAfter(ctx, key, refreshableAfter)
}

// Search performs full-text search across indexed fields. Delegates to the L1 provider.
//
// Takes query (string) which is the search query.
// Takes opts (*cache_dto.SearchOptions) which configures the search.
//
// Returns cache_dto.SearchResult[K, V] which contains the search results from the L1
// provider.
// Returns error when the search operation fails.
func (m *MultiLevelAdapter[K, V]) Search(ctx context.Context, query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	return m.l1Provider.Search(ctx, query, opts)
}

// Query performs structured filtering, sorting, and pagination. Delegates to the L1
// provider.
//
// Takes opts (*cache_dto.QueryOptions) which configures the query.
//
// Returns cache_dto.SearchResult[K, V] which contains the query results from the L1
// provider.
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

// AdapterOption configures optional behaviour of a MultiLevelAdapter at construction
// time.
type AdapterOption[K comparable, V any] func(*MultiLevelAdapter[K, V])

// WithBackgroundConcurrency limits how many background workers (back-population and async
// write-back) may run at once. A non-positive limit disables the bound entirely
// (unbounded concurrency, retained for backward compatibility).
//
// Takes limit (int) which is the maximum number of in-flight background goroutines.
//
// Returns AdapterOption[K, V] which configures the adapter.
func WithBackgroundConcurrency[K comparable, V any](limit int) AdapterOption[K, V] {
	return func(m *MultiLevelAdapter[K, V]) {
		if limit <= 0 {
			m.backgroundSemaphore = nil
			return
		}
		m.backgroundSemaphore = make(chan struct{}, limit)
	}
}

// NewMultiLevelAdapter creates a new multi-level cache provider.
//
// Takes name (string) which identifies this adapter for logging.
// Takes l1 (ProviderPort[K, V]) which is the fast local cache.
// Takes l2 (ProviderPort[K, V]) which is the remote backing cache.
// Takes cbConfig (Config) which configures the L2 circuit breaker.
// Takes opts (...AdapterOption[K, V]) which configures optional behaviour such as
// background worker concurrency.
//
// Returns *MultiLevelAdapter[K, V] which orchestrates L1 and L2 caches.
//
// By default the adapter bounds background goroutines to runtime.NumCPU(). Pass
// WithBackgroundConcurrency to override.
func NewMultiLevelAdapter[K comparable, V any](
	ctx context.Context,
	name string,
	l1 cache_domain.ProviderPort[K, V],
	l2 cache_domain.ProviderPort[K, V],
	cbConfig Config,
	opts ...AdapterOption[K, V],
) *MultiLevelAdapter[K, V] {
	adapter := &MultiLevelAdapter[K, V]{
		l1Provider:          l1,
		l2Provider:          l2,
		l2Circuit:           newCircuitBreaker(ctx, name, cbConfig.MaxConsecutiveFailures, cbConfig.OpenStateTimeout),
		backgroundSemaphore: make(chan struct{}, runtime.NumCPU()),
		name:                name,
		closeOnce:           sync.Once{},
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

// remainingTTL finds the time between an entry's reported expiry and L2 snapshot. The
// result does not account for time spent before the L1 write, but it does not require the
// L2 and local clocks to agree.
//
// Takes entry (cache_dto.Entry[K, V]) which carries the expiry L2 reported.
//
// Returns time.Duration which is the TTL at the L2 snapshot, or zero or negative if the
// entry had expired by then.
func remainingTTL[K comparable, V any](entry cache_dto.Entry[K, V]) time.Duration {
	anchorNano := entry.SnapshotAtNano
	if anchorNano <= 0 {
		anchorNano = time.Now().UnixNano()
	}
	return time.Duration(entry.ExpiresAtNano - anchorNano)
}
