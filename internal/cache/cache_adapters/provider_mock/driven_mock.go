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

package provider_mock

import (
	"context"
	"iter"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/wdk/clock"
)

var _ cache_domain.ProviderPort[any, any] = (*MockAdapter[any, any])(nil)

// mockEntry holds an item in the mock cache's memory, including metadata.
type mockEntry[V any] struct {
	// value holds the cached data for this entry.
	value V

	// expiresAt is when this cache entry becomes invalid.
	expiresAt time.Time

	// tags holds the set of tags associated with this entry.
	tags map[string]struct{}
}

// MockAdapter is a thread-safe, in-memory mock implementation of the
// ProviderPort interface. It implements io.Closer and is designed for testing,
// providing call recording, state simulation, and error injection.
type MockAdapter[K comparable, V any] struct {
	// clock provides time operations for testing.
	clock clock.Clock

	// errToReturn is the error to return from mock method calls.
	errToReturn error

	// setExpiresAfterCalls records the duration passed to each SetExpiresAfter
	// call, keyed by the argument used.
	setExpiresAfterCalls map[any]time.Duration

	// storage maps keys to their mock entries for test assertions.
	storage map[any]*mockEntry[V]

	// tagIndex maps tag names to sets of tagged values.
	tagIndex map[string]map[any]struct{}

	// entryToReturn is the entry to return from Get calls; nil means cache miss.
	entryToReturn *cache_dto.Entry[K, V]

	// bulkGetFunc mocks the BulkGet method.
	bulkGetFunc func(ctx context.Context, keys []K) (map[K]V, error)

	// setRefreshableAfterCalls maps tokens to durations after which they become
	// refreshable.
	setRefreshableAfterCalls map[any]time.Duration

	// setCalls records each call to Set with its key, value, and tags.
	setCalls []struct {
		Key   K
		Value V
		Tags  []string
	}

	// setMaximumCalls records the maximum call limits set for each method.
	setMaximumCalls []uint64

	// getCalls records the keys passed to Get method calls for test verification.
	getCalls []K

	// refreshCalls records the keys passed to Refresh for test verification.
	refreshCalls []K

	// bulkRefreshCalls records keys from each BulkRefresh call for test verification.
	bulkRefreshCalls [][]K

	// getEntryCalls records the keys passed to GetEntry calls.
	getEntryCalls []K

	// probeEntryCalls records the keys passed to ProbeEntry for test verification.
	probeEntryCalls []K

	// invalidateByTagsCalls records the tags passed to InvalidateByTags calls.
	invalidateByTagsCalls []string

	// getIfPresentCalls records the keys passed to GetIfPresent for verification.
	getIfPresentCalls []K

	// invalidateCalls records keys passed to Invalidate for test verification.
	invalidateCalls []K

	// invalidateAllCount tracks the number of times InvalidateAll was called.
	invalidateAllCount int

	// maximumToReturn is the maximum number of results to return; 0 means no limit.
	maximumToReturn uint64

	// weightedSizeToReturn is the value returned by WeightedSize calls.
	weightedSizeToReturn uint64

	// closeCount tracks the number of times Close has been called.
	closeCount int

	// mu guards concurrent access to the mock's state.
	mu sync.RWMutex
}

// MockAdapterOption is a functional option for setting up a MockAdapter.
type MockAdapterOption[K comparable, V any] func(*MockAdapter[K, V])

// GetIfPresent returns the value for the given key if present in the cache.
//
// Takes key (K) which specifies the cache key to look up.
//
// Returns V which is the cached value if found.
// Returns bool which indicates whether the key was present in the cache.
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) GetIfPresent(_ context.Context, key K) (V, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getIfPresentCalls = append(m.getIfPresentCalls, key)

	if m.errToReturn != nil {
		return *new(V), false, m.errToReturn
	}

	entry, ok := m.storage[key]
	if !ok {
		return *new(V), false, nil
	}

	if !entry.expiresAt.IsZero() && m.clock.Now().After(entry.expiresAt) {
		delete(m.storage, key)
		return *new(V), false, nil
	}

	return entry.value, true, nil
}

// Get returns the value for the given key, loading it if not present.
//
// Takes key (K) which identifies the cache entry to retrieve.
// Takes loader (Loader) which loads the value if the key is not in the cache.
//
// Returns V which is the cached or newly loaded value.
// Returns error when the configured error is set or the loader fails.
//
// Safe for concurrent use; access is protected by a mutex.
func (m *MockAdapter[K, V]) Get(ctx context.Context, key K, loader cache_dto.Loader[K, V]) (V, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getCalls = append(m.getCalls, key)

	if m.errToReturn != nil {
		return *new(V), m.errToReturn
	}

	if entry, ok := m.storage[key]; ok {
		if entry.expiresAt.IsZero() || m.clock.Now().Before(entry.expiresAt) {
			return entry.value, nil
		}
	}

	loadedVal, err := loader.Load(ctx, key)
	if err != nil {
		return *new(V), err
	}

	m.storage[key] = &mockEntry[V]{value: loadedVal, expiresAt: time.Time{}, tags: nil}
	return loadedVal, nil
}

// Set stores a value in the cache with optional tags.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) Set(_ context.Context, key K, value V, tags ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.setCalls = append(m.setCalls, struct {
		Key   K
		Value V
		Tags  []string
	}{Key: key, Value: value, Tags: tags})

	if m.errToReturn != nil {
		return m.errToReturn
	}

	entry := &mockEntry[V]{value: value, expiresAt: time.Time{}, tags: nil}
	m.storage[key] = entry

	if len(tags) > 0 {
		tagSet := make(map[string]struct{}, len(tags))
		for _, tag := range tags {
			if _, ok := m.tagIndex[tag]; !ok {
				m.tagIndex[tag] = make(map[any]struct{})
			}
			m.tagIndex[tag][key] = struct{}{}
			tagSet[tag] = struct{}{}
		}
		entry.tags = tagSet
	}

	return nil
}

// SetWithTTL stores a value with a time-to-live duration.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which specifies how long the entry remains valid.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error when the operation fails.
func (m *MockAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	if err := m.Set(ctx, key, value, tags...); err != nil {
		return err
	}
	return m.SetExpiresAfter(ctx, key, ttl)
}

// BulkSet stores multiple values in the cache with optional tags.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags for cache grouping.
//
// Returns error when the operation fails.
func (m *MockAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	for key, value := range items {
		if err := m.Set(ctx, key, value, tags...); err != nil {
			return err
		}
	}
	return nil
}

// Invalidate removes a key from the cache.
//
// Takes key (K) which specifies the cache key to remove.
//
// Returns error when the operation fails.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) Invalidate(_ context.Context, key K) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.invalidateCalls = append(m.invalidateCalls, key)

	entry, ok := m.storage[key]
	if !ok {
		return nil
	}

	for tag := range entry.tags {
		if keys, ok := m.tagIndex[tag]; ok {
			delete(keys, key)
			if len(keys) == 0 {
				delete(m.tagIndex, tag)
			}
		}
	}

	delete(m.storage, key)

	return nil
}

// Compute computes a new value atomically based on the current value.
//
// Takes key (K) which identifies the cache entry to compute.
// Takes computeFunction (func(...)) which receives the current value
// and whether it exists, and returns the new value with an action to
// perform.
//
// Returns V which is the resulting value after the computation.
// Returns bool which indicates whether a value exists for the key.
// Returns error when the operation fails.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) Compute(_ context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errToReturn != nil {
		return *new(V), false, m.errToReturn
	}

	entry, found := m.storage[key]
	var oldValue V
	if found {
		oldValue = entry.value
	}

	newValue, action := computeFunction(oldValue, found)

	switch action {
	case cache_dto.ComputeActionSet:
		m.storage[key] = &mockEntry[V]{value: newValue, expiresAt: time.Time{}, tags: nil}
		return newValue, true, nil
	case cache_dto.ComputeActionDelete:
		if found {
			delete(m.storage, key)
		}
		return *new(V), false, nil
	case cache_dto.ComputeActionNoop:
		if found {
			return oldValue, true, nil
		}
		return *new(V), false, nil
	default:
		return *new(V), false, nil
	}
}

// ComputeIfAbsent computes and stores a value if the key is not present.
//
// Takes key (K) which is the cache key to look up or store.
// Takes computeFunction (func() V) which computes the value if the key is absent.
//
// Returns V which is either the existing value or the newly computed value.
// Returns bool which is true if computation occurred, false if value existed.
// Returns error when the operation fails.
//
// Safe for concurrent use; guards access with a mutex.
func (m *MockAdapter[K, V]) ComputeIfAbsent(_ context.Context, key K, computeFunction func() V) (V, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errToReturn != nil {
		return *new(V), false, m.errToReturn
	}

	entry, found := m.storage[key]
	if found {
		return entry.value, false, nil
	}

	newValue := computeFunction()
	m.storage[key] = &mockEntry[V]{value: newValue, expiresAt: time.Time{}, tags: nil}
	return newValue, true, nil
}

// ComputeIfPresent updates a value only if the key is present.
//
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(...)) which computes the new value
// from the old value.
//
// Returns V which is the resulting value after computation, or zero value if
// key not found or deleted.
// Returns bool which indicates whether the key exists with a value after the
// operation.
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) ComputeIfPresent(_ context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errToReturn != nil {
		return *new(V), false, m.errToReturn
	}

	entry, found := m.storage[key]
	if !found {
		return *new(V), false, nil
	}

	newValue, action := computeFunction(entry.value)

	switch action {
	case cache_dto.ComputeActionSet:
		m.storage[key] = &mockEntry[V]{value: newValue, expiresAt: time.Time{}, tags: nil}
		return newValue, true, nil
	case cache_dto.ComputeActionDelete:
		delete(m.storage, key)
		return *new(V), false, nil
	default:
		return entry.value, true, nil
	}
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found flag,
// returning a ComputeResult containing the new value, action, and optional TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) ComputeWithTTL(_ context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errToReturn != nil {
		return *new(V), false, m.errToReturn
	}

	entry, found := m.storage[key]
	var oldValue V
	if found {
		oldValue = entry.value
	}

	result := computeFunction(oldValue, found)

	switch result.Action {
	case cache_dto.ComputeActionSet:
		newEntry := &mockEntry[V]{value: result.Value, expiresAt: time.Time{}, tags: nil}
		if result.TTL > 0 {
			newEntry.expiresAt = m.clock.Now().Add(result.TTL)
		}
		m.storage[key] = newEntry
		return result.Value, true, nil
	case cache_dto.ComputeActionDelete:
		if found {
			delete(m.storage, key)
		}
		return *new(V), false, nil
	case cache_dto.ComputeActionNoop:
		if found {
			return oldValue, true, nil
		}
		return *new(V), false, nil
	default:
		return *new(V), false, nil
	}
}

// BulkGet retrieves multiple values, loading missing ones with the bulk loader.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader[K, V]) which loads values for keys not in cache.
//
// Returns map[K]V which contains the retrieved and loaded values.
// Returns error when the bulk loader fails to load missing keys.
func (m *MockAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) (map[K]V, error) {
	if m.bulkGetFunc != nil {
		return m.bulkGetFunc(ctx, keys)
	}

	results := make(map[K]V, len(keys))
	var misses []K
	for _, key := range keys {
		value, ok, err := m.GetIfPresent(ctx, key)
		if err != nil {
			return results, err
		}
		if ok {
			results[key] = value
		} else {
			misses = append(misses, key)
		}
	}
	if len(misses) > 0 {
		loaded, err := bulkLoader.BulkLoad(ctx, misses)
		if err != nil {
			return results, err
		}
		for k, v := range loaded {
			if err := m.Set(ctx, k, v); err != nil {
				return results, err
			}
			results[k] = v
		}
	}
	return results, nil
}

// InvalidateByTags removes all entries with the given tags.
//
// Takes tags (...string) which specifies the tags whose entries should be
// removed.
//
// Returns int which is the number of entries that were invalidated.
// Returns error when the operation fails.
//
// Safe for concurrent use; protects internal state with a mutex.
func (m *MockAdapter[K, V]) InvalidateByTags(_ context.Context, tags ...string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.invalidateByTagsCalls = append(m.invalidateByTagsCalls, tags...)

	keysToInvalidate := make(map[any]struct{})
	for _, tag := range tags {
		if keys, ok := m.tagIndex[tag]; ok {
			for key := range keys {
				keysToInvalidate[key] = struct{}{}
			}
			delete(m.tagIndex, tag)
		}
	}

	for keyAny := range keysToInvalidate {
		delete(m.storage, keyAny)
	}

	return len(keysToInvalidate), nil
}

// InvalidateAll removes all entries from the cache.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) InvalidateAll(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.invalidateAllCount++
	m.storage = make(map[any]*mockEntry[V])
	m.tagIndex = make(map[string]map[any]struct{})

	return nil
}

// BulkRefresh asynchronously refreshes multiple keys.
//
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads fresh values for the keys.
//
// Safe for concurrent use. Spawns a goroutine that calls
// bulkLoader.BulkLoad to refresh the values in the background.
func (m *MockAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) {
	m.mu.Lock()
	m.bulkRefreshCalls = append(m.bulkRefreshCalls, keys)
	m.mu.Unlock()
	go func() {
		_, _ = bulkLoader.BulkLoad(ctx, keys)
	}()
}

// Refresh asynchronously refreshes a single key.
//
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which loads the fresh value.
//
// Returns <-chan cache_dto.LoadResult[V] which delivers the
// refresh result.
//
// Safe for concurrent use. Spawns a goroutine that calls
// loader.Load to refresh the value in the background.
func (m *MockAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V] {
	m.mu.Lock()
	m.refreshCalls = append(m.refreshCalls, key)
	m.mu.Unlock()

	resultChan := make(chan cache_dto.LoadResult[V], 1)
	go func() {
		value, err := loader.Load(ctx, key)
		resultChan <- cache_dto.LoadResult[V]{Value: value, Err: err}
		close(resultChan)
	}()
	return resultChan
}

// All returns an iterator over all key-value pairs.
//
// Returns iter.Seq2[K, V] which yields each key-value pair in the
// cache.
//
// Safe for concurrent use. Takes a snapshot of the storage under
// a read lock and iterates over the copy.
func (m *MockAdapter[K, V]) All() iter.Seq2[K, V] {
	m.mu.RLock()
	snapshot := make(map[K]V)
	for k, v := range m.storage {
		snapshot[k.(K)] = v.value
	}
	m.mu.RUnlock()

	return func(yield func(K, V) bool) {
		for k, v := range snapshot {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Keys returns an iterator over all keys.
//
// Returns iter.Seq[K] which yields each key in the cache.
func (m *MockAdapter[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range m.All() {
			if !yield(k) {
				return
			}
		}
	}
}

// Values returns an iterator over all values.
//
// Returns iter.Seq[V] which yields each value in the cache.
func (m *MockAdapter[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range m.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// GetEntry returns the full entry for a key, including metadata.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to retrieve.
//
// Returns cache_dto.Entry[K, V] which contains the entry with
// metadata.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	m.mu.Lock()
	m.getEntryCalls = append(m.getEntryCalls, key)
	m.mu.Unlock()

	if m.entryToReturn != nil {
		return *m.entryToReturn, true, nil
	}
	value, ok, err := m.GetIfPresent(ctx, key)
	if err != nil {
		return cache_dto.Entry[K, V]{}, false, err
	}
	if !ok {
		return cache_dto.Entry[K, V]{}, false, nil
	}
	return cache_dto.Entry[K, V]{
		Key:               key,
		Value:             value,
		Weight:            0,
		ExpiresAtNano:     0,
		RefreshableAtNano: 0,
		SnapshotAtNano:    m.clock.Now().UnixNano(),
	}, true, nil
}

// ProbeEntry returns the entry for a key without affecting access statistics.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to probe.
//
// Returns cache_dto.Entry[K, V] which contains the entry with
// metadata.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	m.mu.Lock()
	m.probeEntryCalls = append(m.probeEntryCalls, key)
	m.mu.Unlock()
	return m.GetEntry(ctx, key)
}

// EstimatedSize returns the approximate number of entries in the cache.
//
// Returns int which is the current count of stored entries.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) EstimatedSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.storage)
}

// Stats returns cache statistics.
//
// Returns cache_dto.Stats which contains mock statistics based on recorded
// method calls.
func (m *MockAdapter[K, V]) Stats() cache_dto.Stats {
	return cache_dto.Stats{
		Hits:             uint64(len(m.getIfPresentCalls)),
		Misses:           0,
		Evictions:        0,
		LoadSuccessCount: uint64(len(m.getCalls)),
		LoadFailureCount: 0,
		TotalLoadTime:    0,
	}
}

// Close releases resources held by the cache.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the operation fails.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) Close(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCount++
	return nil
}

// SetExpiresAfter sets the expiration duration for a key.
//
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies when the entry expires.
//
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) SetExpiresAfter(_ context.Context, key K, expiresAfter time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setExpiresAfterCalls[key] = expiresAfter
	if entry, ok := m.storage[key]; ok {
		entry.expiresAt = m.clock.Now().Add(expiresAfter)
	}
	return nil
}

// GetMaximum returns the maximum size of the cache.
//
// Returns uint64 which is the configured maximum cache capacity.
//
// Safe for concurrent use; protected by a read lock.
func (m *MockAdapter[K, V]) GetMaximum() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.maximumToReturn
}

// SetMaximum sets the maximum size of the cache.
//
// Takes size (uint64) which specifies the new maximum cache size.
//
// Safe for concurrent use; protected by mutex.
func (m *MockAdapter[K, V]) SetMaximum(size uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setMaximumCalls = append(m.setMaximumCalls, size)
	m.maximumToReturn = size
}

// WeightedSize returns the weighted size of all entries in the cache.
//
// Returns uint64 which is the total weighted size, or the storage count if no
// weighted size has been configured.
//
// Safe for concurrent use; protected by a read lock.
func (m *MockAdapter[K, V]) WeightedSize() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.weightedSizeToReturn > 0 {
		return m.weightedSizeToReturn
	}
	return uint64(len(m.storage))
}

// SetRefreshableAfter sets the duration after which a key becomes refreshable.
//
// Takes key (K) which identifies the cache entry to configure.
// Takes refreshableAfter (time.Duration) which specifies when the key becomes
// eligible for refresh.
//
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockAdapter[K, V]) SetRefreshableAfter(_ context.Context, key K, refreshableAfter time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setRefreshableAfterCalls[key] = refreshableAfter
	return nil
}

// Reset clears all recorded calls, stored data, and configured errors.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storage = make(map[any]*mockEntry[V])
	m.tagIndex = make(map[string]map[any]struct{})
	m.getCalls = nil
	m.getIfPresentCalls = nil
	m.setCalls = nil
	m.invalidateCalls = nil
	m.invalidateAllCount = 0
	m.invalidateByTagsCalls = nil
	m.closeCount = 0
	m.errToReturn = nil
	m.bulkGetFunc = nil
}

// SetError configures a generic error to be returned by methods that support
// it.
//
// Takes err (error) which is the error value to return from subsequent calls.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errToReturn = err
}

// SetBulkGetFunc allows tests to provide a custom implementation for BulkGet.
//
// Takes f (func) which is the custom function to handle bulk key retrieval.
//
// Safe for concurrent use; the function is stored while holding the mutex.
func (m *MockAdapter[K, V]) SetBulkGetFunc(f func(ctx context.Context, keys []K) (map[K]V, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bulkGetFunc = f
}

// GetGetCalls returns a copy of all keys passed to Get.
//
// Returns []K which contains a copy of the recorded Get call arguments.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) GetGetCalls() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]K, len(m.getCalls))
	copy(calls, m.getCalls)
	return calls
}

// GetSetCalls returns a copy of all parameters passed to Set.
//
// Returns []struct{Key, Value, Tags} which contains all recorded Set calls.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) GetSetCalls() []struct {
	Key   K
	Value V
	Tags  []string
} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]struct {
		Key   K
		Value V
		Tags  []string
	}, len(m.setCalls))
	copy(calls, m.setCalls)
	return calls
}

// GetInvalidateCalls returns a copy of all keys passed to Invalidate.
//
// Returns []K which contains a copy of all keys that were invalidated.
//
// Safe for concurrent use; holds a read lock while copying.
func (m *MockAdapter[K, V]) GetInvalidateCalls() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]K, len(m.invalidateCalls))
	copy(calls, m.invalidateCalls)
	return calls
}

// GetInvalidateAllCount returns how many times InvalidateAll was called.
//
// Returns int which is the count of InvalidateAll invocations.
//
// Safe for concurrent use.
func (m *MockAdapter[K, V]) GetInvalidateAllCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.invalidateAllCount
}

// Search returns ErrSearchNotSupported by default.
//
// Returns cache_dto.SearchResult[K, V] which is always empty.
// Returns error which is always ErrSearchNotSupported.
func (*MockAdapter[K, V]) Search(_ context.Context, _ string, _ *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	return cache_dto.SearchResult[K, V]{}, cache_domain.ErrSearchNotSupported
}

// Query returns ErrSearchNotSupported by default.
//
// Returns cache_dto.SearchResult[K, V] which is always empty.
// Returns error which is always ErrSearchNotSupported.
func (*MockAdapter[K, V]) Query(_ context.Context, _ *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error) {
	return cache_dto.SearchResult[K, V]{}, cache_domain.ErrSearchNotSupported
}

// SupportsSearch returns false by default.
//
// Returns bool which indicates whether the adapter supports search.
func (*MockAdapter[K, V]) SupportsSearch() bool {
	return false
}

// GetSchema returns nil by default.
//
// Returns *cache_dto.SearchSchema which is always nil for mock adapters.
func (*MockAdapter[K, V]) GetSchema() *cache_dto.SearchSchema {
	return nil
}

// MockProviderFactory is the factory function for creating the mock adapter.
// It accepts typed Options and returns a properly configured MockAdapter.
//
// Returns cache_domain.ProviderPort[K, V] which is the mock cache
// adapter.
// Returns error which is always nil for this mock factory.
func MockProviderFactory[K comparable, V any](_ cache_dto.Options[K, V]) (cache_domain.ProviderPort[K, V], error) {
	return NewMockAdapter[K, V](), nil
}

// WithMockClock sets a custom clock for time operations.
// This is primarily used for testing to make time-based logic deterministic.
//
// Takes c (clock.Clock) which provides deterministic time for
// testing.
//
// Returns MockAdapterOption[K, V] which configures the clock on
// the adapter.
func WithMockClock[K comparable, V any](c clock.Clock) MockAdapterOption[K, V] {
	return func(m *MockAdapter[K, V]) {
		m.clock = c
	}
}

// NewMockAdapter creates a new, initialised mock cache adapter.
//
// Takes opts (...MockAdapterOption[K, V]) which are optional
// configuration functions for the adapter.
//
// Returns *MockAdapter[K, V] which is the initialised mock adapter.
func NewMockAdapter[K comparable, V any](opts ...MockAdapterOption[K, V]) *MockAdapter[K, V] {
	m := &MockAdapter[K, V]{
		clock:                    clock.RealClock(),
		errToReturn:              nil,
		setExpiresAfterCalls:     make(map[any]time.Duration),
		storage:                  make(map[any]*mockEntry[V]),
		tagIndex:                 make(map[string]map[any]struct{}),
		entryToReturn:            nil,
		bulkGetFunc:              nil,
		setRefreshableAfterCalls: make(map[any]time.Duration),
		setCalls:                 nil,
		setMaximumCalls:          nil,
		getCalls:                 nil,
		refreshCalls:             nil,
		bulkRefreshCalls:         nil,
		getEntryCalls:            nil,
		probeEntryCalls:          nil,
		invalidateByTagsCalls:    nil,
		getIfPresentCalls:        nil,
		invalidateCalls:          nil,
		invalidateAllCount:       0,
		maximumToReturn:          0,
		weightedSizeToReturn:     0,
		closeCount:               0,
		mu:                       sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}
