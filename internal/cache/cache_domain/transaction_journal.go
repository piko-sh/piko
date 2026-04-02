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
	"fmt"
	"iter"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
)

// Compile-time check that transactionJournal implements TransactionCache.
var _ TransactionCache[string, any] = (*transactionJournal[string, any])(nil)

// journalEntry records the pre-mutation state of a single cache key so
// the mutation can be undone on Rollback.
//
// Tags are not captured because the cache interface does not expose
// per-entry tag retrieval. This is acceptable because DAL consumers
// manage their own tag indexes separately from the cache.
type journalEntry[K comparable, V any] struct {
	// key is the cache key that was mutated.
	key K

	// oldValue holds the value that existed before the mutation.
	oldValue V

	// existed is true when the key was present before the mutation.
	// When false, rollback removes the key rather than restoring a value.
	existed bool

	// expiresAtNano is the Unix timestamp in nanoseconds when the
	// entry was due to expire. Zero means no TTL was set, and on
	// rollback the remaining TTL is computed from this value.
	expiresAtNano int64
}

// transactionJournal wraps any ProviderPort and adds undo-journal-based
// rollback. Mutations are applied immediately to the inner cache but
// recorded so they can be reversed.
//
// Synchronous loader methods (Get, BulkGet) snapshot keys before
// delegating, so loader-triggered population is rolled back correctly.
// Asynchronous methods (Refresh, BulkRefresh) cannot be journalled
// because the cache write happens after the method returns - callers
// should avoid these within transactions if rollback correctness is
// required.
//
// This type is NOT safe for concurrent use. It is designed to be used
// within a single goroutine that holds an external lock (e.g. the DAL
// mutex).
type transactionJournal[K comparable, V any] struct {
	// inner is the underlying cache that receives all operations.
	inner ProviderPort[K, V]

	// snapshotted tracks which keys already have a journal entry so that
	// only the first mutation per key is recorded.
	snapshotted map[K]struct{}

	// journal records pre-mutation state for each key, in mutation order.
	journal []journalEntry[K, V]

	// finalised is true after Commit or Rollback has been called.
	// Subsequent mutations or double-finalise attempts return an error.
	finalised bool
}

// newTransactionJournal creates a transaction journal wrapping the
// given cache.
//
// Takes inner (ProviderPort[K, V]) which is the underlying cache
// that receives all operations.
//
// Returns *transactionJournal[K, V] which wraps inner with undo
// journalling.
func newTransactionJournal[K comparable, V any](inner ProviderPort[K, V]) *transactionJournal[K, V] {
	return &transactionJournal[K, V]{
		inner:       inner,
		snapshotted: make(map[K]struct{}),
	}
}

// snapshotKey records the current state of key before a mutation.
// Only the first mutation per key is recorded; subsequent mutations
// to the same key are skipped because rollback only needs the
// original value.
//
// Takes key (K) which identifies the cache entry to snapshot.
//
// Returns error when the context is cancelled or the probe fails.
func (t *transactionJournal[K, V]) snapshotKey(ctx context.Context, key K) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.finalised {
		return ErrTransactionFinalised
	}
	if _, already := t.snapshotted[key]; already {
		return nil
	}

	entry, existed, err := t.inner.ProbeEntry(ctx, key)
	if err != nil {
		return fmt.Errorf("snapshotting key before mutation: %w", err)
	}
	t.snapshotted[key] = struct{}{}
	t.journal = append(t.journal, journalEntry[K, V]{
		key:           key,
		oldValue:      entry.Value,
		existed:       existed,
		expiresAtNano: entry.ExpiresAtNano,
	})
	return nil
}

// Commit discards the undo journal, making all mutations permanent.
//
// Returns error when the transaction has already been finalised.
func (t *transactionJournal[K, V]) Commit(_ context.Context) error {
	if t.finalised {
		return ErrTransactionFinalised
	}
	t.finalised = true
	t.journal = nil
	t.snapshotted = nil
	return nil
}

// Rollback replays the undo journal, restoring every mutated key
// to its pre-transaction state. Journal entries are replayed in
// reverse order so that the first mutation's original value is
// restored last (though with the snapshotted-once design each key
// appears at most once).
//
// When the original entry had a TTL, the remaining duration is
// computed from the snapshotted expiry time and restored via
// SetWithTTL. If the TTL has already elapsed the entry is not
// restored (it would have expired anyway).
//
// All entries are attempted even when individual restores fail;
// errors are collected and returned via errors.Join so that a
// single failure does not leave remaining entries unreplayed.
//
// Returns error when the transaction was already finalised or
// any restore operation fails.
func (t *transactionJournal[K, V]) Rollback(ctx context.Context) error {
	if t.finalised {
		return ErrTransactionFinalised
	}
	t.finalised = true

	nowNano := time.Now().UnixNano()
	var errs []error
	for i := len(t.journal) - 1; i >= 0; i-- {
		entry := t.journal[i]
		if entry.existed {
			if err := t.restoreEntry(ctx, entry, nowNano); err != nil {
				errs = append(errs, fmt.Errorf("restoring key during rollback: %w", err))
			}
		} else {
			if err := t.inner.Invalidate(ctx, entry.key); err != nil {
				errs = append(errs, fmt.Errorf("invalidating key during rollback: %w", err))
			}
		}
	}
	t.journal = nil
	t.snapshotted = nil
	return errors.Join(errs...)
}

// restoreEntry puts back a single journal entry, computing the
// remaining TTL if one was set and invalidating the key when
// the TTL has already elapsed.
//
// Takes entry (journalEntry[K, V]) which holds the pre-mutation
// state to restore.
// Takes nowNano (int64) which is the current Unix time in
// nanoseconds, used to compute remaining TTL.
//
// Returns error when the underlying cache operation fails.
func (t *transactionJournal[K, V]) restoreEntry(ctx context.Context, entry journalEntry[K, V], nowNano int64) error {
	if entry.expiresAtNano == 0 {
		return t.inner.Set(ctx, entry.key, entry.oldValue)
	}
	remaining := time.Duration(entry.expiresAtNano - nowNano)
	if remaining <= 0 {
		return t.inner.Invalidate(ctx, entry.key)
	}
	return t.inner.SetWithTTL(ctx, entry.key, entry.oldValue, remaining)
}

// Set stores a value, snapshotting the key's prior state for
// rollback.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional group names for
// invalidation.
//
// Returns error when snapshotting or storing fails.
func (t *transactionJournal[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	if err := t.snapshotKey(ctx, key); err != nil {
		return err
	}
	return t.inner.Set(ctx, key, value, tags...)
}

// SetWithTTL stores a value with TTL, snapshotting the key's
// prior state.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which sets when this entry expires.
// Takes tags (...string) which enable group-based invalidation.
//
// Returns error when snapshotting or storing fails.
func (t *transactionJournal[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	if err := t.snapshotKey(ctx, key); err != nil {
		return err
	}
	return t.inner.SetWithTTL(ctx, key, value, ttl, tags...)
}

// Invalidate removes a key, snapshotting the key's prior state
// for rollback.
//
// Takes key (K) which identifies the entry to remove.
//
// Returns error when snapshotting or removal fails.
func (t *transactionJournal[K, V]) Invalidate(ctx context.Context, key K) error {
	if err := t.snapshotKey(ctx, key); err != nil {
		return err
	}
	return t.inner.Invalidate(ctx, key)
}

// Compute atomically updates a key, snapshotting the key's prior
// state.
//
// Takes key (K) which identifies the cache entry to update.
// Takes fn (func) which receives the current value and presence
// flag, returning the new value and action.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when snapshotting or computation fails.
func (t *transactionJournal[K, V]) Compute(ctx context.Context, key K, fn func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	if err := t.snapshotKey(ctx, key); err != nil {
		var zero V
		return zero, false, err
	}
	return t.inner.Compute(ctx, key, fn)
}

// ComputeIfAbsent computes a value if absent, snapshotting the
// key's prior state.
//
// Takes key (K) which identifies the entry to look up or create.
// Takes fn (func() V) which generates the value if the key is
// absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which indicates whether computation occurred.
// Returns error when snapshotting or computation fails.
func (t *transactionJournal[K, V]) ComputeIfAbsent(ctx context.Context, key K, fn func() V) (V, bool, error) {
	if err := t.snapshotKey(ctx, key); err != nil {
		var zero V
		return zero, false, err
	}
	return t.inner.ComputeIfAbsent(ctx, key, fn)
}

// ComputeIfPresent updates a key if present, snapshotting the
// key's prior state.
//
// Takes key (K) which identifies the entry to update.
// Takes fn (func) which computes the new value from the
// existing value.
//
// Returns V which is the new value if the key was present.
// Returns bool which indicates whether the key was present.
// Returns error when snapshotting or computation fails.
func (t *transactionJournal[K, V]) ComputeIfPresent(ctx context.Context, key K, fn func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	if err := t.snapshotKey(ctx, key); err != nil {
		var zero V
		return zero, false, err
	}
	return t.inner.ComputeIfPresent(ctx, key, fn)
}

// ComputeWithTTL atomically computes a value with TTL,
// snapshotting the key's prior state.
//
// Takes key (K) which identifies the cache entry to update.
// Takes fn (func) which receives the current value and presence
// flag, returning a ComputeResult with value, action, and TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when snapshotting or computation fails.
func (t *transactionJournal[K, V]) ComputeWithTTL(ctx context.Context, key K, fn func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	if err := t.snapshotKey(ctx, key); err != nil {
		var zero V
		return zero, false, err
	}
	return t.inner.ComputeWithTTL(ctx, key, fn)
}

// BulkSet stores multiple values, snapshotting each key's prior
// state.
//
// Takes items (map[K]V) which contains the key-value pairs to
// store.
// Takes tags (...string) which are optional tags to associate
// with all items.
//
// Returns error when snapshotting or storing fails.
func (t *transactionJournal[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	for key := range items {
		if err := t.snapshotKey(ctx, key); err != nil {
			return err
		}
	}
	return t.inner.BulkSet(ctx, items, tags...)
}

// InvalidateByTags is not supported within a transaction because
// the Cache interface does not expose tag-to-key resolution,
// making it impossible to journal individual key changes for
// rollback.
//
// Returns int which is always zero.
// Returns error which is always ErrInvalidateByTagsUnsupported.
func (*transactionJournal[K, V]) InvalidateByTags(_ context.Context, _ ...string) (int, error) {
	return 0, ErrInvalidateByTagsUnsupported
}

// InvalidateAll is not supported within a transaction because
// bulk invalidation cannot be efficiently journalled at the key
// level.
//
// Returns error which is always ErrInvalidateAllUnsupported.
func (*transactionJournal[K, V]) InvalidateAll(_ context.Context) error {
	return ErrInvalidateAllUnsupported
}

// GetIfPresent returns the value for a key if it is present and
// not expired, delegating directly to the inner cache.
//
// Takes key (K) which identifies the cached item to retrieve.
//
// Returns V which is the cached value.
// Returns bool which is true if the key was found.
// Returns error when the operation fails.
func (t *transactionJournal[K, V]) GetIfPresent(ctx context.Context, key K) (V, bool, error) {
	return t.inner.GetIfPresent(ctx, key)
}

// Get returns the value for key, using the loader to populate the
// cache on a miss. The key is snapshotted before delegating so
// that loader-triggered population can be undone on rollback.
//
// Takes key (K) which identifies the cached value.
// Takes loader (Loader[K, V]) which fetches the value if not
// in cache.
//
// Returns V which is the cached or newly loaded value.
// Returns error when snapshotting or loading fails.
func (t *transactionJournal[K, V]) Get(ctx context.Context, key K, loader cache_dto.Loader[K, V]) (V, error) {
	if err := t.snapshotKey(ctx, key); err != nil {
		var zero V
		return zero, err
	}
	return t.inner.Get(ctx, key, loader)
}

// BulkGet returns values for all keys, using the bulk loader to
// populate missing entries. All keys are snapshotted before
// delegating so that loader-triggered population can be undone
// on rollback.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader[K, V]) which loads values for
// missing keys.
//
// Returns map[K]V which contains all requested key-value pairs.
// Returns error when snapshotting or bulk loading fails.
func (t *transactionJournal[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) (map[K]V, error) {
	for _, key := range keys {
		if err := t.snapshotKey(ctx, key); err != nil {
			return nil, err
		}
	}
	return t.inner.BulkGet(ctx, keys, bulkLoader)
}

// BulkRefresh triggers asynchronous refresh for the given
// keys, bypassing the undo journal because the actual cache
// writes happen after this method returns.
//
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader[K, V]) which fetches new values
// in bulk.
func (t *transactionJournal[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) {
	t.inner.BulkRefresh(ctx, keys, bulkLoader)
}

// Refresh triggers an asynchronous refresh for a single key.
// Because the actual cache write happens asynchronously, it
// cannot be reliably journalled; callers should avoid Refresh
// within transactions if rollback correctness is required.
//
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which fetches the new value.
//
// Returns <-chan LoadResult[V] which receives the result upon
// completion.
func (t *transactionJournal[K, V]) Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V] {
	return t.inner.Refresh(ctx, key, loader)
}

// All returns an iterator over all key-value pairs in the cache,
// delegating to the inner cache.
//
// Returns iter.Seq2[K, V] which yields each key and its value.
func (t *transactionJournal[K, V]) All() iter.Seq2[K, V] {
	return t.inner.All()
}

// Keys returns an iterator over all keys in the cache,
// delegating to the inner cache.
//
// Returns iter.Seq[K] which yields each key.
func (t *transactionJournal[K, V]) Keys() iter.Seq[K] {
	return t.inner.Keys()
}

// Values returns an iterator over all values in the cache,
// delegating to the inner cache.
//
// Returns iter.Seq[V] which yields each value.
func (t *transactionJournal[K, V]) Values() iter.Seq[V] {
	return t.inner.Values()
}

// GetEntry returns a snapshot of the entry including metadata,
// delegating to the inner cache.
//
// Takes key (K) which identifies the entry to retrieve.
//
// Returns Entry[K, V] which contains the entry data and metadata.
// Returns bool which indicates whether the entry was found.
// Returns error when the operation fails.
func (t *transactionJournal[K, V]) GetEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	return t.inner.GetEntry(ctx, key)
}

// ProbeEntry returns a snapshot of the entry without resetting
// its access timer, delegating to the inner cache.
//
// Takes key (K) which identifies the entry to probe.
//
// Returns Entry[K, V] which is the entry snapshot.
// Returns bool which indicates whether the entry exists.
// Returns error when the operation fails.
func (t *transactionJournal[K, V]) ProbeEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	return t.inner.ProbeEntry(ctx, key)
}

// EstimatedSize returns the approximate number of entries in the
// cache, delegating to the inner cache.
//
// Returns int which is the approximate entry count.
func (t *transactionJournal[K, V]) EstimatedSize() int {
	return t.inner.EstimatedSize()
}

// Stats returns a snapshot of the cache's performance statistics,
// delegating to the inner cache.
//
// Returns Stats which contains hit/miss counts and other
// performance metrics.
func (t *transactionJournal[K, V]) Stats() cache_dto.Stats {
	return t.inner.Stats()
}

// GetMaximum returns the current maximum capacity of the cache,
// delegating to the inner cache.
//
// Returns uint64 which is the maximum number of items.
func (t *transactionJournal[K, V]) GetMaximum() uint64 {
	return t.inner.GetMaximum()
}

// SetMaximum changes the maximum capacity of the cache,
// delegating to the inner cache.
//
// Takes size (uint64) which is the new maximum number of items.
func (t *transactionJournal[K, V]) SetMaximum(size uint64) {
	t.inner.SetMaximum(size)
}

// WeightedSize returns the total weight of all items in the
// cache, delegating to the inner cache.
//
// Returns uint64 which is the sum of all item weights.
func (t *transactionJournal[K, V]) WeightedSize() uint64 {
	return t.inner.WeightedSize()
}

// SetExpiresAfter sets or changes the expiry time for a key,
// delegating to the inner cache.
//
// Takes key (K) which identifies the item to update.
// Takes expiresAfter (time.Duration) which specifies how long
// until the key expires.
//
// Returns error when the operation fails.
func (t *transactionJournal[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	return t.inner.SetExpiresAfter(ctx, key, expiresAfter)
}

// SetRefreshableAfter sets or overrides the refresh time for a
// key, delegating to the inner cache.
//
// Takes key (K) which identifies the cache entry to update.
// Takes refreshableAfter (time.Duration) which specifies the
// new refresh time.
//
// Returns error when the operation fails.
func (t *transactionJournal[K, V]) SetRefreshableAfter(ctx context.Context, key K, refreshableAfter time.Duration) error {
	return t.inner.SetRefreshableAfter(ctx, key, refreshableAfter)
}

// Search performs full-text search across indexed fields,
// delegating to the inner cache.
//
// Takes query (string) which is the search query text.
// Takes opts (*SearchOptions) which configures pagination,
// sorting, and filters.
//
// Returns SearchResult[K, V] which contains the matched
// entries.
// Returns error when the search fails.
func (t *transactionJournal[K, V]) Search(ctx context.Context, query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	return t.inner.Search(ctx, query, opts)
}

// Query performs structured filtering and pagination without
// full-text search, delegating to the inner cache.
//
// Takes opts (*QueryOptions) which specifies filters, sorting,
// and pagination.
//
// Returns SearchResult[K, V] which contains the matched
// entries.
// Returns error when the query fails.
func (t *transactionJournal[K, V]) Query(ctx context.Context, opts *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error) {
	return t.inner.Query(ctx, opts)
}

// SupportsSearch returns true if the inner cache supports search
// and query operations.
//
// Returns bool which is true when search is available.
func (t *transactionJournal[K, V]) SupportsSearch() bool {
	return t.inner.SupportsSearch()
}

// GetSchema returns the search schema for the inner cache, or
// nil if search is not configured.
//
// Returns *SearchSchema which describes the indexed fields,
// or nil when search is not configured.
func (t *transactionJournal[K, V]) GetSchema() *cache_dto.SearchSchema {
	return t.inner.GetSchema()
}

// Close is a no-op on the transaction journal since the
// transaction wrapper does not own the underlying cache.
//
// Returns error which is always nil.
func (*transactionJournal[K, V]) Close(_ context.Context) error {
	return nil
}
