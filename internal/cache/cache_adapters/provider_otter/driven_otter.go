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

package provider_otter

import (
	"context"
	"iter"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/maypok86/otter/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/wal/wal_domain"
)

var _ cache_domain.ProviderPort[any, any] = (*OtterAdapter[any, any])(nil)

// TagIndex maps tags to cache keys for the in-memory Otter cache. It keeps a
// reverse index so that all keys with a given tag can be found and removed
// together.
//
// For tokenised full-text search, use InvertedIndex instead.
type TagIndex[K comparable] struct {
	// index maps each tag to the set of keys that have that tag.
	index map[string]map[K]struct{}

	// keyToTags maps keys to their tags for quick reverse lookup.
	keyToTags map[K]map[string]struct{}

	// maxTagsPerKey limits tags per key. Zero means unlimited.
	maxTagsPerKey int

	// mu guards access to the tag index data.
	mu sync.RWMutex
}

// Add links a key with a set of tags.
//
// Takes key (K) which identifies the item to tag.
// Takes tags ([]string) which lists the tags to link with the key.
//
// Removes any existing tags for the key before adding the new ones.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) Add(key K, tags []string) {
	if len(tags) == 0 {
		return
	}
	if ti.maxTagsPerKey > 0 && len(tags) > ti.maxTagsPerKey {
		tags = tags[:ti.maxTagsPerKey]
	}
	ti.mu.Lock()
	defer ti.mu.Unlock()

	ti.removeKeyUnsafe(key)

	tagSet := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if _, ok := ti.index[tag]; !ok {
			ti.index[tag] = make(map[K]struct{})
		}
		ti.index[tag][key] = struct{}{}
		tagSet[tag] = struct{}{}
	}
	ti.keyToTags[key] = tagSet
}

// Invalidate finds all keys associated with the given tags,
// removes them from the index, and returns the list of keys to be
// invalidated from the main cache.
//
// Takes tags ([]string) which contains the tags whose associated
// keys should be invalidated.
//
// Returns []K which contains the keys that were invalidated.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) Invalidate(tags []string) []K {
	if len(tags) == 0 {
		return nil
	}
	ti.mu.Lock()
	defer ti.mu.Unlock()

	keysToInvalidate := make(map[K]struct{})
	for _, tag := range tags {
		if keys, ok := ti.index[tag]; ok {
			for key := range keys {
				keysToInvalidate[key] = struct{}{}
			}
			delete(ti.index, tag)
		}
	}

	if len(keysToInvalidate) == 0 {
		return nil
	}

	result := slices.Collect(maps.Keys(keysToInvalidate))
	for _, key := range result {
		delete(ti.keyToTags, key)
	}
	return result
}

// removeKeyUnsafe removes all tag links for a given key.
//
// Takes key (K) which identifies the entry to remove from the index.
//
// Must be called with a write lock held.
func (ti *TagIndex[K]) removeKeyUnsafe(key K) {
	if tags, ok := ti.keyToTags[key]; ok {
		for tag := range tags {
			if keys, ok := ti.index[tag]; ok {
				delete(keys, key)
				if len(keys) == 0 {
					delete(ti.index, tag)
				}
			}
		}
		delete(ti.keyToTags, key)
	}
}

// RemoveKey removes a single key from the tag index.
//
// Takes key (K) which identifies the entry to remove.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) RemoveKey(key K) {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.removeKeyUnsafe(key)
}

// Get returns all keys associated with the given tag.
//
// Takes tag (string) which is the tag to look up.
//
// Returns map[K]struct{} which is the set of keys with this tag.
// Returns nil if no keys have this tag.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) Get(tag string) map[K]struct{} {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	if keys, ok := ti.index[tag]; ok {
		result := make(map[K]struct{}, len(keys))
		for k := range keys {
			result[k] = struct{}{}
		}
		return result
	}
	return nil
}

// AddSingle links a single key with a single tag, without changing other tags
// the key may have.
//
// Takes tag (string) which is the tag to link.
// Takes key (K) which identifies the item to tag.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) AddSingle(tag string, key K) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if _, ok := ti.index[tag]; !ok {
		ti.index[tag] = make(map[K]struct{})
	}
	ti.index[tag][key] = struct{}{}

	if _, ok := ti.keyToTags[key]; !ok {
		ti.keyToTags[key] = make(map[string]struct{})
	}
	ti.keyToTags[key][tag] = struct{}{}
}

// RemoveSingle removes a single tag from a key, without affecting other tags
// the key may have.
//
// Takes tag (string) which is the tag to disassociate.
// Takes key (K) which identifies the item to untag.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) RemoveSingle(tag string, key K) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if keys, ok := ti.index[tag]; ok {
		delete(keys, key)
		if len(keys) == 0 {
			delete(ti.index, tag)
		}
	}

	if tags, ok := ti.keyToTags[key]; ok {
		delete(tags, tag)
		if len(tags) == 0 {
			delete(ti.keyToTags, key)
		}
	}
}

// Clear removes all tags and keys from the index.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) Clear() {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.index = make(map[string]map[K]struct{})
	ti.keyToTags = make(map[K]map[string]struct{})
}

// GetTags returns the tags associated with a key.
//
// Takes key (K) which identifies the item to look up.
//
// Returns []string which contains the tags for the key, or nil if the key has
// no tags.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) GetTags(key K) []string {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	tagSet, ok := ti.keyToTags[key]
	if !ok || len(tagSet) == 0 {
		return nil
	}

	return slices.Collect(maps.Keys(tagSet))
}

// OtterAdapter is a driven adapter that uses the maypok86/otter/v2 library to
// provide caching. It implements io.Closer.
type OtterAdapter[K comparable, V any] struct {
	// client is the Otter cache instance that stores data.
	client *otter.Cache[K, V]

	// tagIndex maps tags to cache keys for tag-based invalidation.
	tagIndex *TagIndex[K]

	// schema specifies which fields can be searched.
	schema *cache_dto.SearchSchema

	// invertedIndex maps search terms to keys for full-text search.
	invertedIndex *InvertedIndex[K]

	// sortedIndexes holds sorted indexes for fields that support ordering.
	sortedIndexes map[string]*SortedIndex[K]

	// vectorIndexes holds HNSW indexes for vector similarity search.
	vectorIndexes map[string]*VectorIndex[K]

	// fieldExtractor extracts field values from cached items.
	fieldExtractor *FieldExtractor[V]

	// wal is the write-ahead log for persistence. Nil if persistence is disabled.
	wal wal_domain.WAL[K, V]

	// snapshot is the store used to save state. Nil if saving is turned off.
	snapshot wal_domain.SnapshotStore[K, V]

	// walEnabled indicates whether write-ahead log persistence is active.
	walEnabled bool

	// snapshotThreshold is the number of WAL entries before a checkpoint is made.
	// When this limit is reached, a snapshot is saved and the WAL is cleared.
	snapshotThreshold int

	// checkpointMu coordinates checkpoint and write operations. Writes acquire
	// RLock, checkpoint acquires Lock, ensuring the snapshot contains all WAL
	// entries before truncation.
	checkpointMu sync.RWMutex
}

// GetIfPresent returns the value for the given key if present in the cache.
//
// Takes key (K) which is the cache key to look up.
//
// Returns V which is the cached value if found.
// Returns bool which indicates whether the key was present in the cache.
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) GetIfPresent(_ context.Context, key K) (V, bool, error) {
	v, ok := a.client.GetIfPresent(key)
	return v, ok, nil
}

// Get retrieves a value from the cache, loading it via the provided loader
// if not present.
//
// Takes key (K) which identifies the cache entry to retrieve.
// Takes loader (Loader[K, V]) which loads the value if not found in cache.
//
// Returns V which is the cached or newly loaded value.
// Returns error when the loader fails or the cache operation fails.
func (a *OtterAdapter[K, V]) Get(ctx context.Context, key K, loader cache_dto.Loader[K, V]) (V, error) {
	return a.client.Get(ctx, key, loader)
}

// Set stores a value in the cache with optional tags.
//
// Takes ctx (context.Context) which is accepted for interface conformance but
// not checked, as in-memory operations are non-blocking.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error which is always nil for the in-memory adapter.
//
// Safe for concurrent use. Uses a read lock during WAL operations.
func (a *OtterAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	ctx, l := logger_domain.From(ctx, log)

	if a.walEnabled && a.wal != nil {
		a.checkpointMu.RLock()
		entry := wal_domain.Entry[K, V]{
			Operation: wal_domain.OpSet,
			Key:       key,
			Value:     value,
			Tags:      tags,
			Timestamp: time.Now().UnixNano(),
		}
		if err := a.wal.Append(context.WithoutCancel(ctx), entry); err != nil {
			l.Warn("Failed to append to WAL", logger_domain.Error(err))
		}
		a.tagIndex.Add(key, tags)
		a.client.Set(key, value)
		a.indexDocument(key, value)
		a.checkpointMu.RUnlock()
		a.maybeCheckpoint()
		return nil
	}

	a.tagIndex.Add(key, tags)
	a.client.Set(key, value)
	a.indexDocument(key, value)
	return nil
}

// SetWithTTL stores a value with a time-to-live duration.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which specifies how long the entry remains valid.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error when the operation fails.
//
// Safe for concurrent use. Uses a read lock during WAL operations to allow
// concurrent reads while ensuring checkpoint consistency.
func (a *OtterAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	ctx, l := logger_domain.From(ctx, log)

	if a.walEnabled && a.wal != nil {
		a.checkpointMu.RLock()
		expiresAtNs := int64(0)
		if ttl > 0 {
			expiresAtNs = time.Now().Add(ttl).UnixNano()
		}
		entry := wal_domain.Entry[K, V]{
			Operation: wal_domain.OpSet,
			Key:       key,
			Value:     value,
			Tags:      tags,
			ExpiresAt: expiresAtNs,
			Timestamp: time.Now().UnixNano(),
		}
		if err := a.wal.Append(ctx, entry); err != nil {
			l.Warn("Failed to append to WAL", logger_domain.Error(err))
		}
		a.tagIndex.Add(key, tags)
		a.client.Set(key, value)
		a.client.SetExpiresAfter(key, ttl)
		a.indexDocument(key, value)
		a.checkpointMu.RUnlock()
		a.maybeCheckpoint()
		return nil
	}

	a.tagIndex.Add(key, tags)
	a.client.Set(key, value)
	a.client.SetExpiresAfter(key, ttl)
	a.indexDocument(key, value)
	return nil
}

// Invalidate removes a key from the cache.
//
// Takes ctx (context.Context) which is accepted for interface conformance but
// not checked, as in-memory operations are non-blocking.
// Takes key (K) which identifies the cache entry to invalidate.
//
// Returns error which is always nil for the in-memory adapter.
//
// Safe for concurrent use. Uses a read lock during WAL operations
// to allow concurrent invalidations while blocking checkpoints.
func (a *OtterAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	ctx, l := logger_domain.From(ctx, log)

	if a.walEnabled && a.wal != nil {
		a.checkpointMu.RLock()
		entry := wal_domain.Entry[K, V]{
			Operation: wal_domain.OpDelete,
			Key:       key,
			Timestamp: time.Now().UnixNano(),
		}
		if err := a.wal.Append(context.WithoutCancel(ctx), entry); err != nil {
			l.Warn("Failed to append delete to WAL", logger_domain.Error(err))
		}
		a.tagIndex.RemoveKey(key)
		a.removeFromSearchIndex(key)
		a.client.Invalidate(key)
		a.checkpointMu.RUnlock()
		a.maybeCheckpoint()
		return nil
	}

	a.tagIndex.RemoveKey(key)
	a.removeFromSearchIndex(key)
	a.client.Invalidate(key)
	return nil
}

// Compute computes and atomically updates the value in the cache using the
// provided function.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and
// whether it was found, and returns the new value and action to
// perform.
//
// Returns V which is the computed value.
// Returns bool which indicates whether the value exists in the cache.
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) Compute(_ context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	v, ok := a.client.Compute(key, func(oldValue V, found bool) (newValue V, op otter.ComputeOp) {
		newValue, action := computeFunction(oldValue, found)
		return newValue, actionToOp(action)
	})
	return v, ok, nil
}

// ComputeIfAbsent computes and stores a value if the key is not present.
//
// Takes key (K) which is the cache key to look up or store.
// Takes computeFunction (func()) which computes the value if the key is absent.
//
// Returns V which is either the existing value or the newly computed one.
// Returns bool which indicates whether the value was already present.
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) ComputeIfAbsent(_ context.Context, key K, computeFunction func() V) (V, bool, error) {
	v, ok := a.client.ComputeIfAbsent(key, func() (V, bool) {
		return computeFunction(), false
	})
	return v, ok, nil
}

// ComputeIfPresent updates a value only if the key is present.
//
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(...)) which computes the new value
// from the old value.
//
// Returns V which is the computed value, or the zero value if the key was
// absent.
// Returns bool which is true if the key was present and the value was updated.
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) ComputeIfPresent(_ context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	v, ok := a.client.ComputeIfPresent(key, func(oldValue V) (newValue V, op otter.ComputeOp) {
		newValue, action := computeFunction(oldValue)
		return newValue, actionToOp(action)
	})
	return v, ok, nil
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found flag,
// returning a ComputeResult containing the new value, action, and optional TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) ComputeWithTTL(_ context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	var resultTTL time.Duration

	value, present := a.client.Compute(key, func(oldValue V, found bool) (V, otter.ComputeOp) {
		result := computeFunction(oldValue, found)
		resultTTL = result.TTL
		return result.Value, actionToOp(result.Action)
	})

	if present && resultTTL > 0 {
		a.client.SetExpiresAfter(key, resultTTL)
	}

	return value, present, nil
}

// BulkGet retrieves multiple values from the cache, loading any missing ones
// using the bulk loader.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader) which loads values for keys not in the cache.
//
// Returns map[K]V which contains the retrieved or loaded values.
// Returns error when the bulk load operation fails.
func (a *OtterAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) (map[K]V, error) {
	return a.client.BulkGet(ctx, keys, bulkLoader)
}

// BulkSet stores multiple key-value pairs in the cache with optional tags.
// Optimised to batch index updates for better performance.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags to associate with each
// key.
//
// Returns error when the operation fails, though currently always returns nil.
//
// Safe for concurrent use. Uses a read lock during WAL-enabled operations to
// coordinate with checkpointing.
func (a *OtterAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	ctx, l := logger_domain.From(ctx, log)

	if a.walEnabled && a.wal != nil {
		a.checkpointMu.RLock()
		nowNs := time.Now().UnixNano()
		for key, value := range items {
			if ctx.Err() != nil {
				a.checkpointMu.RUnlock()
				return ctx.Err()
			}

			entry := wal_domain.Entry[K, V]{
				Operation: wal_domain.OpSet,
				Key:       key,
				Value:     value,
				Tags:      tags,
				Timestamp: nowNs,
			}
			if err := a.wal.Append(ctx, entry); err != nil {
				l.Warn("Failed to append bulk item to WAL", logger_domain.Error(err))
			}
		}
		for key, value := range items {
			a.tagIndex.Add(key, tags)
			a.client.Set(key, value)
		}
		a.indexDocumentsBatch(items)
		a.checkpointMu.RUnlock()
		a.maybeCheckpoint()
		return nil
	}

	for key, value := range items {
		a.tagIndex.Add(key, tags)
		a.client.Set(key, value)
	}

	a.indexDocumentsBatch(items)

	return nil
}

// InvalidateByTags removes all entries with matching tags from the cache.
//
// Takes ctx (context.Context) which is accepted for interface conformance but
// not checked, as in-memory operations are non-blocking.
// Takes tags (...string) which specifies the tags to match for invalidation.
//
// Returns int which is the number of keys that were invalidated.
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	ctx, l := logger_domain.From(ctx, log)

	keys := a.tagIndex.Invalidate(tags)
	if len(keys) == 0 {
		return 0, nil
	}

	for _, key := range keys {
		a.client.Invalidate(key)
	}

	tagInvalidationsTotal.Add(ctx, 1)
	invalidatedKeysByTagTotal.Add(ctx, int64(len(keys)))
	l.Trace("Invalidated keys by tags",
		logger_domain.Int("tag_count", len(tags)),
		logger_domain.Int("key_count", len(keys)))

	return len(keys), nil
}

// InvalidateAll clears all entries from the cache.
//
// Takes ctx (context.Context) which is accepted for interface conformance but
// not checked, as in-memory operations are non-blocking.
//
// Returns error which is always nil for the in-memory adapter.
//
// Safe for concurrent use. The operation is protected by a read lock during
// WAL operations.
func (a *OtterAdapter[K, V]) InvalidateAll(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	if a.walEnabled && a.wal != nil {
		a.checkpointMu.RLock()
		entry := wal_domain.Entry[K, V]{
			Operation: wal_domain.OpClear,
			Timestamp: time.Now().UnixNano(),
		}
		if err := a.wal.Append(context.WithoutCancel(ctx), entry); err != nil {
			l.Warn("Failed to append clear to WAL", logger_domain.Error(err))
		}
		a.clearAllIndexes()
		a.client.InvalidateAll()
		a.checkpointMu.RUnlock()
		a.maybeCheckpoint()
		return nil
	}

	a.clearAllIndexes()
	a.client.InvalidateAll()
	return nil
}

// BulkRefresh refreshes multiple keys in the background using the bulk loader.
//
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads fresh values for the keys.
func (a *OtterAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) {
	a.client.BulkRefresh(ctx, keys, bulkLoader)
}

// Refresh asynchronously reloads a single key using the provided loader.
//
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which loads the fresh value.
//
// Returns <-chan cache_dto.LoadResult[V] which delivers the
// refresh result.
//
// Safe for concurrent use. Spawns a goroutine that converts
// the otter result channel into a DTO result channel.
func (a *OtterAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V] {
	otterResultChan := a.client.Refresh(ctx, key, loader)

	if otterResultChan == nil {
		ch := make(chan cache_dto.LoadResult[V], 1)
		close(ch)
		return ch
	}

	dtoResultChan := make(chan cache_dto.LoadResult[V], 1)
	go func() {
		defer close(dtoResultChan)
		defer goroutine.RecoverPanic(ctx, "cache.otterRefresh")
		otterResult := <-otterResultChan
		dtoResultChan <- cache_dto.LoadResult[V]{
			Value: otterResult.Value,
			Err:   otterResult.Err,
		}
	}()
	return dtoResultChan
}

// All returns an iterator over all key-value pairs in the cache.
//
// Returns iter.Seq2[K, V] which yields each key-value pair.
func (a *OtterAdapter[K, V]) All() iter.Seq2[K, V] {
	return a.client.All()
}

// Keys returns an iterator over all keys in the cache.
//
// Returns iter.Seq[K] which yields each key in the cache.
func (a *OtterAdapter[K, V]) Keys() iter.Seq[K] {
	return a.client.Keys()
}

// Values returns an iterator over all values in the cache.
//
// Returns iter.Seq[V] which yields each value in the cache.
func (a *OtterAdapter[K, V]) Values() iter.Seq[V] {
	return a.client.Values()
}

// GetEntry returns the full entry for a key, including metadata.
//
// Takes key (K) which identifies the entry to retrieve.
//
// Returns cache_dto.Entry[K, V] which contains the entry with
// metadata.
// Returns bool which indicates whether the key was found.
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) GetEntry(_ context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	entry, ok := a.client.GetEntry(key)
	if !ok {
		return cache_dto.Entry[K, V]{}, false, nil
	}
	return convertEntryToDTO(entry), true, nil
}

// ProbeEntry returns the entry for a key without affecting access statistics.
//
// Takes key (K) which identifies the entry to probe.
//
// Returns cache_dto.Entry[K, V] which contains the entry with
// metadata.
// Returns bool which indicates whether the key was found.
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) ProbeEntry(_ context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	entry, ok := a.client.GetEntryQuietly(key)
	if !ok {
		return cache_dto.Entry[K, V]{}, false, nil
	}
	return convertEntryToDTO(entry), true, nil
}

// EstimatedSize returns the estimated number of entries in the cache.
//
// Returns int which is the estimated count of cached entries.
func (a *OtterAdapter[K, V]) EstimatedSize() int {
	return a.client.EstimatedSize()
}

// Stats returns cache statistics.
//
// Returns cache_dto.Stats which contains counts for hits, misses, and
// evictions.
func (a *OtterAdapter[K, V]) Stats() cache_dto.Stats {
	otterStats := a.client.Stats()

	return cache_dto.Stats{
		Hits:             otterStats.Hits,
		Misses:           otterStats.Misses,
		Evictions:        otterStats.Evictions,
		LoadSuccessCount: 0,
		LoadFailureCount: 0,
		TotalLoadTime:    otterStats.TotalLoadTime,
	}
}

// maybeCheckpoint checks if the WAL entry count has reached the snapshot
// threshold and if so, creates a snapshot and truncates the WAL. This compacts
// the WAL to prevent unbounded growth.
//
// Safe for concurrent use. Acquires a write lock that blocks all writes during
// the checkpoint operation to ensure snapshot consistency.
func (a *OtterAdapter[K, V]) maybeCheckpoint() {
	if !a.walEnabled || a.wal == nil || a.snapshot == nil {
		return
	}

	if a.snapshotThreshold <= 0 {
		return
	}

	if a.wal.EntryCount() < a.snapshotThreshold {
		return
	}

	a.checkpointMu.Lock()
	defer a.checkpointMu.Unlock()

	if a.wal.EntryCount() < a.snapshotThreshold {
		return
	}

	a.performCheckpointLocked()
}

// performCheckpointLocked creates a snapshot of the current cache state and
// truncates the WAL.
//
// Called when the WAL entry count exceeds the snapshot threshold. Caller must
// hold checkpointMu write lock.
func (a *OtterAdapter[K, V]) performCheckpointLocked() {
	ctx := context.Background()
	ctx, l := logger_domain.From(ctx, log)

	entries := a.collectSnapshotEntries()

	if err := a.snapshot.Save(ctx, entries); err != nil {
		l.Warn("Failed to save snapshot during checkpoint", logger_domain.Error(err))
		return
	}

	if err := a.wal.Truncate(ctx); err != nil {
		l.Warn("Failed to truncate WAL after checkpoint", logger_domain.Error(err))
		return
	}

	l.Internal("Checkpoint completed",
		logger_domain.Int("entries_snapshot", len(entries)))
}

// collectSnapshotEntries collects all current cache entries for snapshotting.
//
// Returns []wal_domain.Entry[K, V] which contains all current
// cache entries with their tags and timestamps.
func (a *OtterAdapter[K, V]) collectSnapshotEntries() []wal_domain.Entry[K, V] {
	entries := make([]wal_domain.Entry[K, V], 0, a.client.EstimatedSize())
	nowNano := time.Now().UnixNano()

	for key, value := range a.client.All() {
		entry := wal_domain.Entry[K, V]{
			Operation: wal_domain.OpSet,
			Key:       key,
			Value:     value,
			Timestamp: nowNano,
		}

		if tags := a.tagIndex.GetTags(key); len(tags) > 0 {
			entry.Tags = tags
		}

		entries = append(entries, entry)
	}

	return entries
}

// Close releases all resources held by the cache.
//
// Takes ctx (context.Context) which is accepted for interface conformance but
// not checked, as in-memory operations are non-blocking.
//
// Returns error which is always nil for the in-memory adapter.
//
// Safe for concurrent use. Performs a final checkpoint before closing the WAL
// and snapshot store if WAL is enabled.
func (a *OtterAdapter[K, V]) Close(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	if a.walEnabled {
		a.checkpointMu.Lock()
		if a.wal != nil && a.snapshot != nil && a.wal.EntryCount() > 0 {
			a.performCheckpointLocked()
		}
		a.checkpointMu.Unlock()

		if a.wal != nil {
			if err := a.wal.Close(); err != nil {
				l.Warn("Failed to close WAL", logger_domain.Error(err))
			}
		}
		if a.snapshot != nil {
			if err := a.snapshot.Close(); err != nil {
				l.Warn("Failed to close snapshot store", logger_domain.Error(err))
			}
		}
	}
	a.client.StopAllGoroutines()
	return nil
}

// SetExpiresAfter sets the expiration duration for a key.
//
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies how long until the key
// expires.
//
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) SetExpiresAfter(_ context.Context, key K, expiresAfter time.Duration) error {
	a.client.SetExpiresAfter(key, expiresAfter)
	return nil
}

// GetMaximum returns the maximum size of the cache.
//
// Returns uint64 which is the maximum number of entries the cache can hold.
func (a *OtterAdapter[K, V]) GetMaximum() uint64 {
	return a.client.GetMaximum()
}

// SetMaximum sets the maximum size of the cache.
//
// Takes size (uint64) which specifies the new maximum number of entries.
func (a *OtterAdapter[K, V]) SetMaximum(size uint64) {
	a.client.SetMaximum(size)
}

// WeightedSize returns the weighted size of all entries in the cache.
//
// Returns uint64 which is the total weighted size of cached entries.
func (a *OtterAdapter[K, V]) WeightedSize() uint64 {
	return a.client.WeightedSize()
}

// SetRefreshableAfter sets the duration after which a key becomes refreshable.
//
// Takes key (K) which identifies the cache entry to configure.
// Takes refreshableAfter (time.Duration) which specifies when the key becomes
// eligible for refresh.
//
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) SetRefreshableAfter(_ context.Context, key K, refreshableAfter time.Duration) error {
	a.client.SetRefreshableAfter(key, refreshableAfter)
	return nil
}

// NewTagIndex creates a new empty tag index.
//
// Returns *TagIndex[K] ready for use.
func NewTagIndex[K comparable]() *TagIndex[K] {
	return &TagIndex[K]{
		index:     make(map[string]map[K]struct{}),
		keyToTags: make(map[K]map[string]struct{}),
		mu:        sync.RWMutex{},
	}
}

// newTagIndex is an internal alias for backwards compatibility.
//
// Returns *TagIndex[K] which is a new empty tag index.
func newTagIndex[K comparable]() *TagIndex[K] {
	return NewTagIndex[K]()
}

// actionToOp converts a ComputeAction to its corresponding otter.ComputeOp
// value.
//
// Takes action (cache_dto.ComputeAction) which is the action to convert.
//
// Returns otter.ComputeOp which is the mapped operation. Unknown actions
// return CancelOp by default.
func actionToOp(action cache_dto.ComputeAction) otter.ComputeOp {
	switch action {
	case cache_dto.ComputeActionSet:
		return otter.WriteOp
	case cache_dto.ComputeActionDelete:
		return otter.InvalidateOp
	case cache_dto.ComputeActionNoop:
		return otter.CancelOp
	}
	return otter.CancelOp
}
