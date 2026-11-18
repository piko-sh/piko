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
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/wal/wal_domain"
)

var (
	_ cache_domain.LogBatcher[string, any] = (*OtterAdapter[string, any])(nil)

	_ cache_domain.LogBatch[string, any] = (*batchedAdapter[string, any])(nil)
)

// logBatch collects the log entries of one transaction until it commits.
type logBatch[K comparable, V any] struct {
	// entries holds the collected entries in the order they were written.
	entries []wal_domain.Entry[K, V]

	// mu guards entries and ended.
	mu sync.Mutex

	// ended records that the batch was committed or aborted, after which it takes no more
	// entries.
	ended bool
}

// add appends entries to the batch while it is still open.
//
// Takes entries (...wal_domain.Entry[K, V]) which are the entries to collect.
//
// Returns bool which is false when the batch has ended, in which case the caller must log
// the entries itself.
//
// Safe for concurrent use.
func (b *logBatch[K, V]) add(entries ...wal_domain.Entry[K, V]) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ended {
		return false
	}
	b.entries = append(b.entries, entries...)
	return true
}

// end closes the batch and hands back what it collected.
//
// Returns []wal_domain.Entry[K, V] which holds the collected entries.
// Returns bool which is false when the batch had already ended.
//
// Safe for concurrent use.
func (b *logBatch[K, V]) end() ([]wal_domain.Entry[K, V], bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ended {
		return nil, false
	}
	b.ended = true
	entries := b.entries
	b.entries = nil
	return entries, true
}

// batchedAdapter is a view of the cache whose writes collect in one log batch. Writes
// made on the cache itself, by any goroutine, never join the batch.
//
// Reads and the operations a transaction does not allow pass straight to the cache.
type batchedAdapter[K comparable, V any] struct {
	*OtterAdapter[K, V]

	// batch collects the writes made through this view; nil when the cache keeps no log.
	batch *logBatch[K, V]
}

// BeginLogBatch starts a batch so a transaction reaches the log as one record.
//
// Returns cache_domain.LogBatch[K, V] which is the view to write through. When the cache
// keeps no log its writes go straight to the cache and ending it does nothing.
//
// Concurrency: The batch holds the batch gate for reading until it commits or aborts.
// Checkpoint and close wait for it to end. Separate batches do not wait for each other.
func (a *OtterAdapter[K, V]) BeginLogBatch() cache_domain.LogBatch[K, V] {
	view := &batchedAdapter[K, V]{OtterAdapter: a, batch: nil}
	if !a.logsWrites() {
		return view
	}
	a.logBatchGate.RLock()
	view.batch = &logBatch[K, V]{entries: nil, mu: sync.Mutex{}, ended: false}
	return view
}

// CommitLogBatch writes the collected entries to the log as one record and ends the
// batch. A single entry is written as itself; a batch too large for one record is written
// entry by entry, which loses atomicity but never the entries.
//
// Returns error when the log cannot be written.
//
// Concurrency: releases the batch gate taken by BeginLogBatch.
func (b *batchedAdapter[K, V]) CommitLogBatch(ctx context.Context) error {
	if b.batch == nil {
		return nil
	}
	entries, open := b.batch.end()
	if !open {
		return nil
	}

	b.checkpointMu.RLock()
	err := b.appendEntries(ctx, entries)
	b.checkpointMu.RUnlock()
	b.logBatchGate.RUnlock()
	if err != nil {
		return err
	}
	b.maybeCheckpoint()
	return nil
}

// AbortLogBatch discards the collected entries and ends the batch.
//
// Concurrency: releases the batch gate taken by BeginLogBatch.
func (b *batchedAdapter[K, V]) AbortLogBatch() {
	if b.batch == nil {
		return
	}
	if _, open := b.batch.end(); open {
		b.logBatchGate.RUnlock()
	}
}

// Set stores a value as part of the batch.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error when the write cannot be logged.
func (b *batchedAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	return b.set(ctx, b.batch, key, value, tags)
}

// SetWithTTL stores a value with a TTL as part of the batch.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which specifies how long the entry remains valid.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error when the write cannot be logged.
func (b *batchedAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	return b.setWithTTL(ctx, b.batch, key, value, ttl, tags)
}

// Invalidate removes a key as part of the batch.
//
// Takes key (K) which identifies the cache entry to remove.
//
// Returns error when the removal cannot be logged.
func (b *batchedAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	return b.invalidate(ctx, b.batch, key)
}

// Compute runs a compute as part of the batch.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(V, bool) (V, cache_dto.ComputeAction)) which computes the
// new value and action.
//
// Returns V which is the computed value.
// Returns bool which indicates whether the value exists in the cache.
// Returns error when the write cannot be logged.
func (b *batchedAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return b.compute(ctx, b.batch, key, computeFunction)
}

// ComputeIfAbsent stores a computed value, when the key is absent, as part of the batch.
//
// Takes key (K) which is the cache key to look up or store.
// Takes computeFunction (func()) which computes the value if the key is absent.
//
// Returns V which is either the existing value or the newly computed one.
// Returns bool which indicates whether a value is present afterwards.
// Returns error when the write cannot be logged.
func (b *batchedAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	return b.computeIfAbsent(ctx, b.batch, key, computeFunction)
}

// ComputeIfPresent updates a present value as part of the batch.
//
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(V) (V, cache_dto.ComputeAction)) which computes the new
// value and action from the old value.
//
// Returns V which is the computed value, or the zero value if the key was absent.
// Returns bool which is true if the key was present and the value was updated.
// Returns error when the write cannot be logged.
func (b *batchedAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return b.computeIfPresent(ctx, b.batch, key, computeFunction)
}

// ComputeWithTTL runs a compute with TTL control as part of the batch.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(V, bool) cache_dto.ComputeResult[V]) which computes the new
// value, action and TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the write cannot be logged.
func (b *batchedAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	return b.computeWithTTL(ctx, b.batch, key, computeFunction)
}

// BulkSet stores several values as part of the batch.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which are optional labels for every key.
//
// Returns error when a write cannot be logged.
func (b *batchedAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	return b.bulkSet(ctx, b.batch, items, tags)
}

// SetExpiresAfter changes the expiry of an existing key as part of the batch.
//
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies how long until the key expires.
//
// Returns error when the write cannot be logged.
func (b *batchedAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	return b.setExpiresAfter(ctx, b.batch, key, expiresAfter)
}

// Close does nothing, since the view does not own the cache.
//
// Returns error which is always nil.
func (*batchedAdapter[K, V]) Close(_ context.Context) error {
	return nil
}

// logEntries records entries in batch, or writes them to the log as one record when batch
// is nil or has ended.
//
// Must be called with checkpointMu held for reading.
//
// Takes batch (*logBatch[K, V]) which is the batch the write belongs to; nil when none.
// Takes entries (...wal_domain.Entry[K, V]) which are the entries to record.
//
// Returns error when the log cannot be written.
func (a *OtterAdapter[K, V]) logEntries(ctx context.Context, batch *logBatch[K, V], entries ...wal_domain.Entry[K, V]) error {
	if batch != nil && batch.add(entries...) {
		return nil
	}
	return a.appendEntries(ctx, entries)
}

// appendEntries writes one log record when possible, falling back to separate records
// when the batch exceeds the limit.
//
// Must be called with checkpointMu held for reading.
//
// Takes entries ([]wal_domain.Entry[K, V]) which are the entries to write.
//
// Returns error when the log cannot be written.
func (a *OtterAdapter[K, V]) appendEntries(ctx context.Context, entries []wal_domain.Entry[K, V]) error {
	ctx, l := logger_domain.From(context.WithoutCancel(ctx), log)
	switch len(entries) {
	case 0:
		return nil
	case 1:
		return a.wal.Append(ctx, entries[0])
	}

	err := a.wal.Append(ctx, wal_domain.Entry[K, V]{
		Operation: wal_domain.OpBatch,
		Batch:     entries,
		Timestamp: a.nowNano(),
		Key:       *new(K),
		Value:     *new(V),
		Tags:      nil,
		ExpiresAt: 0,
	})
	if !errors.Is(err, wal_domain.ErrEntryTooLarge) {
		return err
	}

	l.Warn("Transaction too large for one log record; writing its entries one by one",
		logger_domain.Int("entries", len(entries)))
	for i, entry := range entries {
		if err := a.wal.Append(ctx, entry); err != nil {
			return fmt.Errorf("writing entry %d of an oversized transaction: %w", i, err)
		}
	}
	return nil
}
