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
	"iter"
	"math"
	"sync"
	"time"

	"github.com/maypok86/otter/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// invalidateChunkSize bounds how many keys one tag invalidation removes between checks
	// of the context and the checkpoint trigger.
	invalidateChunkSize = 256
)

var (
	_ cache_domain.ProviderPort[any, any] = (*OtterAdapter[any, any])(nil)

	// errPreparedWithoutDeferredLog reports an entry encoded ahead for a log that cannot
	// queue it, which only a log swapped out from under a write could cause.
	errPreparedWithoutDeferredLog = errors.New("cache log does not support deferred writes")
)

// OtterAdapter stores cache entries with Otter and optionally persists them.
//
// Cache reads, writes, and invalidations can run concurrently while the adapter is open.
type OtterAdapter[K comparable, V any] struct {
	// wal is the write-ahead log for persistence. Nil if persistence is disabled.
	wal wal_domain.WAL[K, V]

	// snapshot is the store used to save state. Nil if saving is turned off.
	snapshot wal_domain.SnapshotStore[K, V]

	// fieldExtractor extracts field values from cached items.
	fieldExtractor *FieldExtractor[V]

	// invertedIndex maps search terms to keys for full-text search.
	invertedIndex *InvertedIndex[K]

	// sortedIndexes holds sorted indexes for fields that support ordering.
	sortedIndexes map[string]*SortedIndex[K]

	// vectorIndexes holds HNSW indexes for vector similarity search.
	vectorIndexes map[string]*VectorIndex[K]

	// client is the Otter cache instance that stores data.
	client *otter.Cache[K, V]

	// schema specifies which fields can be searched.
	schema *cache_dto.SearchSchema

	// tagIndex maps tags to cache keys for tag-based invalidation.
	tagIndex *TagIndex[K]

	// expiryOverrides passes the TTL of a write to the expiry calculator running inside that
	// write. It is a separate allocation so the expiry calculator, which otter keeps, does
	// not hold the adapter alive.
	expiryOverrides *expiryOverrides[K]

	// clock is the time source the cache was built with; nil means the system clock.
	clock cache_dto.Clock

	// weigher measures a loaded value against maxEntryWeight; nil when no ceiling is set.
	weigher func(K, V) uint32

	// onDeletion reports entries leaving the cache, and values refused admission by the
	// per-entry ceiling. It may be nil.
	onDeletion func(cache_dto.DeletionEvent[K, V])

	// closeErr holds the result of the first Close so later calls report the same outcome.
	closeErr error

	// namespace names the cache in log output.
	namespace string

	// sortedFields lists the sorted indexes by field name, for writes to walk.
	sortedFields []sortedField[K]

	// vectorFields lists the vector indexes by field name, for writes to walk.
	vectorFields []vectorField[K]

	// snapshotThreshold is the number of WAL entries before a checkpoint is made. When this
	// limit is reached, a snapshot is saved and the WAL is cleared.
	snapshotThreshold int

	// maxWALSize is the WAL size in bytes before a checkpoint is made. An entry count alone
	// cannot bound a log of variable-size values, so a cache of large entries would grow the
	// WAL far past any stated limit before the count trigger fired.
	maxWALSize int64

	// checkpointMu coordinates checkpoint and write operations. Writes acquire RLock,
	// checkpoint acquires Lock, ensuring the snapshot contains all WAL entries before
	// truncation.
	checkpointMu sync.RWMutex

	// closeOnce guards single execution of Close to prevent double-checkpoint and
	// double-close on the WAL or snapshot store.
	closeOnce sync.Once

	// logBatchGate is held for reading from BeginLogBatch until the batch ends, and for
	// writing by a checkpoint or close, so neither ever splits a batch.
	logBatchGate sync.RWMutex

	// ttlWarningOnce limits the warning about a TTL this cache cannot honour to one per
	// cache.
	ttlWarningOnce sync.Once

	// maxEntryWeight is the largest weight a single entry may have; 0 means no ceiling.
	maxEntryWeight uint32

	// weightBounded records that the cache was configured with a weigher and a weight bound,
	// so its WeightedSize is a byte total rather than an entry count.
	weightBounded bool

	// walEnabled indicates whether write-ahead log persistence is active.
	walEnabled bool

	// expiryEnabled records that the cache was built with an expiry calculator. Otter
	// ignores per-entry TTLs on a cache built without one.
	expiryEnabled bool

	// perEntry records that the cache was built with PerEntryExpiration, under which a write
	// without a TTL keeps the entry's current expiry.
	perEntry bool
}

// computeRequest describes one write made through computeWithExpiry.
type computeRequest[K comparable, V any] struct {
	// value is the value a plain write stores, used when remap is nil.
	value V

	// remap computes a new value, operation and TTL from the current value.
	//
	// It runs under otter's bucket lock and must not call back into the cache. Nil makes the
	// request a plain write of value with ttl.
	remap func(oldValue V, found bool) (V, otter.ComputeOp, time.Duration)

	// batch is the log batch the write joins; nil logs the write on its own.
	batch *logBatch[K, V]

	// tags are the tags to give the entry when it is written, each once and cut to the
	// per-key limit; empty keeps its tags.
	tags []string

	// search is what value contributes to the search indexes, built before the lock; nil
	// when it was not built. It describes value only, so it is committed only when remap is
	// nil and the write stores value; any other write builds its entry from the value it
	// applies.
	search *searchEntry

	// prepared is the log entry of a write whose value is known before the lock, encoded
	// ahead; it is not ready when the entry is encoded under the lock.
	prepared preparedEntry[V]

	// ttl is the TTL a plain write passes to the compute, used when remap is nil.
	ttl time.Duration

	// keepsValue records that a write stores the value it found, so the search indexes need
	// no update.
	keepsValue bool

	// logsMissingRemoval records that a removal is logged even when the key is absent, so a
	// write the log still holds cannot return on replay.
	logsMissingRemoval bool
}

// preparedEntry is a log entry encoded before the bucket lock is taken, with what its
// encoding assumed about the entry's state.
type preparedEntry[V any] struct {
	// encoded is the encoded entry; it is queued or released exactly once.
	encoded wal_domain.EncodedEntry

	// tags are the tags the entry was encoded with.
	tags []string

	// expiresAt is the expiry the entry was encoded with.
	expiresAt int64

	// checksExpiry records that expiresAt was read from the entry's current state, so it
	// must still hold under the lock.
	checksExpiry bool

	// checksTags records that tags were read from the tag index, so they must still hold
	// under the lock.
	checksTags bool

	// ready records that the entry was encoded and may be queued.
	ready bool

	// used records that the entry was queued.
	used bool
}

// computeState is what one compute call shares with the function otter runs under its
// lock. Keeping it in one allocation, rather than in several captured variables, keeps
// the write path to as few allocations as it can.
type computeState[K comparable, V any] struct {
	// override is the TTL handed to the expiry calculator, discarded once the compute
	// returns; nil when the write carries no TTL.
	override *expiryOverride

	// outcome records what the compute did.
	outcome computeOutcome[V]

	// request describes the write.
	request computeRequest[K, V]
}

// resolvedTTL is the TTL a cache applies for a requested TTL.
type resolvedTTL struct {
	// duration is the TTL to apply, valid when applies.
	duration time.Duration

	// applies reports whether the write carries a TTL.
	applies bool

	// unsupported reports that a TTL was requested on a cache built without expiry support,
	// so it was ignored.
	unsupported bool
}

// computedWrite describes what one compute applies, for the log.
type computedWrite[V any] struct {
	// value is the value being written, valid when op is otter.WriteOp.
	value V

	// tags are the tags the write gives the entry; empty logs the tags it already has.
	tags []string

	// duration is the TTL of the write, valid when hasTTL.
	duration time.Duration

	// op is the operation being applied.
	op otter.ComputeOp

	// found reports whether the entry existed.
	found bool

	// hasTTL reports whether the write carries a TTL.
	hasTTL bool
}

// computeOutcome records what happened inside one compute call.
type computeOutcome[V any] struct {
	// value is the value present after the compute.
	value V

	// err is set when the write could not be logged; the cache is then unchanged.
	err error

	// wait completes the logging of the write; it is never nil.
	wait func() error

	// op is the operation the compute applied.
	op otter.ComputeOp

	// present reports whether a value is present after the compute.
	present bool

	// ran records that otter called the compute function at all.
	ran bool

	// ttlIgnored records that the write asked for a TTL the cache cannot honour, so the
	// warning is logged once the lock is released.
	ttlIgnored bool
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

// Get retrieves a value from the cache, loading it via the provided loader if not
// present.
//
// Takes key (K) which identifies the cache entry to retrieve.
// Takes loader (Loader[K, V]) which loads the value if not found in cache.
//
// Returns V which is the cached or newly loaded value.
// Returns error when the loader fails or the cache operation fails.
func (a *OtterAdapter[K, V]) Get(ctx context.Context, key K, loader cache_dto.Loader[K, V]) (V, error) {
	return unwrapLoadOutcome(a.client.Get(ctx, key, a.wrapLoader(loader)))
}

// Set stores a value under key with optional tags.
//
// Writes that update the log or indexes count as a hit or miss in cache statistics.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error when logging fails. A failed log wait can leave the write in memory.
func (a *OtterAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	return a.set(ctx, nil, key, value, tags)
}

// set stores a value, joining batch when one is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags ([]string) which are optional labels for grouping entries.
//
// Returns error when the write cannot be logged.
func (a *OtterAdapter[K, V]) set(ctx context.Context, batch *logBatch[K, V], key K, value V, tags []string) error {
	if a.writesDirectly(tags) {
		a.client.Set(key, value)
		return nil
	}
	_, _, err := a.computeWithExpiry(ctx, key, a.plainWrite(batch, key, value, 0, resolvedTTL{}, tags))
	return err
}

// writesDirectly reports whether a plain write can skip the locked compute, which it can
// only when there is no log to order and no index to keep in step.
//
// Takes tags ([]string) which are the tags of the write.
//
// Returns bool which is true when the write can go straight to otter.
func (a *OtterAdapter[K, V]) writesDirectly(tags []string) bool {
	return !a.logsWrites() && len(tags) == 0 && a.schema == nil
}

// plainWrite prepares a write whose value does not depend on the current one. It prepares
// the log entry, tags and search values before taking otter's bucket lock.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes key (K) which identifies the entry.
// Takes value (V) which is the value to store.
// Takes ttl (time.Duration) which is the requested TTL, passed to the compute.
// Takes resolved (resolvedTTL) which is the TTL the cache applies.
// Takes tags ([]string) which are the tags of the write.
//
// Returns computeRequest[K, V] which describes the write.
func (a *OtterAdapter[K, V]) plainWrite(
	batch *logBatch[K, V],
	key K,
	value V,
	ttl time.Duration,
	resolved resolvedTTL,
	tags []string,
) computeRequest[K, V] {
	tags = a.tagIndex.prepareTags(tags)
	return computeRequest[K, V]{
		value:              value,
		remap:              nil,
		batch:              batch,
		tags:               tags,
		prepared:           a.prepareEntry(batch, key, value, resolved, tags),
		search:             a.prepareSearchEntry(value),
		ttl:                ttl,
		keepsValue:         false,
		logsMissingRemoval: false,
	}
}

// prepareEntry encodes the log entry of a plain write before the bucket lock is taken, so
// the lock is not held while the value is encoded.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; a batch is
// encoded as a whole when it commits, so nothing is prepared for one.
// Takes key (K) which identifies the entry.
// Takes value (V) which is the value to store.
// Takes resolved (resolvedTTL) which is the TTL the cache applies.
// Takes tags ([]string) which are the tags of the write.
//
// Returns preparedEntry[V] which holds the encoded entry; it is not ready when nothing
// could be prepared, in which case the entry is encoded under the lock.
func (a *OtterAdapter[K, V]) prepareEntry(batch *logBatch[K, V], key K, value V, resolved resolvedTTL, tags []string) preparedEntry[V] {
	if batch != nil || !a.logsWrites() {
		return preparedEntry[V]{}
	}
	deferred, ok := a.wal.(wal_domain.DeferredAppender[K, V])
	if !ok {
		return preparedEntry[V]{}
	}

	loggedTags := tags
	if len(tags) == 0 {
		loggedTags = a.tagIndex.GetTags(key)
	}
	expiresAt := a.loggedExpiry(key, resolved.duration, resolved.applies)
	encoded, err := deferred.Encode(wal_domain.Entry[K, V]{
		Operation: wal_domain.OpSet,
		Key:       key,
		Value:     value,
		Tags:      loggedTags,
		Timestamp: a.nowNano(),
		ExpiresAt: expiresAt,
		Batch:     nil,
	})
	if err != nil {
		return preparedEntry[V]{}
	}
	return preparedEntry[V]{
		encoded:      encoded,
		tags:         loggedTags,
		expiresAt:    expiresAt,
		checksExpiry: !resolved.applies,
		checksTags:   len(tags) == 0,
		ready:        true,
		used:         false,
	}
}

// SetDeferred stores and indexes a value, then queues its write-ahead log entry without
// waiting for durability.
//
// Entries retain queue order, so waiting for the last entry covers preceding writes.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns func() error which waits until the entry is durable and reports a failure to
// write it; it is a no-op when the cache keeps no log.
// Returns error when the entry cannot be queued; the cache is then left unchanged.
func (a *OtterAdapter[K, V]) SetDeferred(ctx context.Context, key K, value V, tags ...string) (func() error, error) {
	if _, ok := a.wal.(wal_domain.DeferredAppender[K, V]); !a.logsWrites() || !ok {
		return noWait, a.Set(ctx, key, value, tags...)
	}

	outcome := a.runCompute(ctx, key, a.plainWrite(nil, key, value, 0, resolvedTTL{}, tags), nil)
	a.signalLog()
	if outcome.err != nil {
		return noWait, fmt.Errorf("logging write: %w", outcome.err)
	}
	a.maybeCheckpoint()
	return outcome.wait, nil
}

// noWait is the wait of a write that is already complete.
//
// Returns error which is always nil.
func noWait() error { return nil }

// SetWithTTL stores a value with a time-to-live duration.
//
// The value and its TTL are written in one locked step, so a concurrent write can never
// leave the value without the TTL. Like any compute, the write counts as one hit or miss
// in the cache statistics.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which specifies how long the entry remains valid; zero or
// negative writes the value as Set does, and cache_dto.NoExpiry removes any expiry.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error when logging fails. A failed log wait can leave the write in memory.
func (a *OtterAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	return a.setWithTTL(ctx, nil, key, value, ttl, tags)
}

// setWithTTL stores a value with a TTL, joining batch when one is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which specifies how long the entry remains valid.
// Takes tags ([]string) which are optional labels for grouping entries.
//
// Returns error when the write cannot be logged.
func (a *OtterAdapter[K, V]) setWithTTL(ctx context.Context, batch *logBatch[K, V], key K, value V, ttl time.Duration, tags []string) error {
	resolved := a.resolveTTL(ttl)
	if resolved.unsupported {
		a.warnTTLUnsupported(ctx)
	}
	if !resolved.applies {
		return a.set(ctx, batch, key, value, tags)
	}

	_, _, err := a.computeWithExpiry(ctx, key, a.plainWrite(batch, key, value, ttl, resolved, tags))
	return err
}

// resolveTTL turns a requested TTL into the one the cache applies. It never logs, so it
// is safe to call under otter's bucket lock.
//
// Takes ttl (time.Duration) which is the requested TTL.
//
// The TTL is not capped here. A cap only holds for the clock reading it was worked out
// against, and the clock moves on before the TTL is added to it, so loggedExpiry and
// overrideDuration each cap the TTL against the very reading they add it to.
//
// Returns resolvedTTL which holds cache_dto.NoExpiry or ttl, and whether any TTL applies:
// none does when ttl is zero or negative, or when the cache was built without expiry
// support.
func (a *OtterAdapter[K, V]) resolveTTL(ttl time.Duration) resolvedTTL {
	switch {
	case ttl == cache_dto.NoExpiry:
		return resolvedTTL{duration: cache_dto.NoExpiry, applies: a.expiryEnabled, unsupported: false}
	case ttl <= 0:
		return resolvedTTL{duration: 0, applies: false, unsupported: false}
	case !a.expiryEnabled:
		return resolvedTTL{duration: 0, applies: false, unsupported: true}
	}
	return resolvedTTL{duration: ttl, applies: true, unsupported: false}
}

// warnTTLUnsupported logs once per cache when a TTL is requested without expiry support.
func (a *OtterAdapter[K, V]) warnTTLUnsupported(ctx context.Context) {
	a.ttlWarningOnce.Do(func() {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Cache ignores per-entry TTLs because it was built without expiry support",
			logger_domain.String(logKeyNamespace, a.namespace),
			logger_domain.String("remedy", "build the cache with PerEntryExpiration or another expiration option"))
	})
}

// computeWithExpiry applies a write and its expiry, indexes and log entry in order.
//
// It waits for log durability after the cache write and checkpoints when due.
//
// Takes key (K) which identifies the entry.
// Takes request (computeRequest[K, V]) which describes the write.
//
// Returns V which is the value present afterwards.
// Returns bool which reports whether a value is present afterwards.
// Returns error when the write cannot be logged, in which case the cache is unchanged, or
// when waiting for the log fails, in which case the cache holds the write but it may not
// survive a restart.
func (a *OtterAdapter[K, V]) computeWithExpiry(ctx context.Context, key K, request computeRequest[K, V]) (V, bool, error) {
	return a.finishCompute(ctx, a.runCompute(ctx, key, request, nil))
}

// runCompute calls otter through invoke with a compute function that logs the write,
// updates the indexes and records its TTL, holding the checkpoint read lock when the
// cache keeps a log.
//
// Takes key (K) which identifies the entry.
// Takes request (computeRequest[K, V]) which describes the write.
// Takes invoke (func(func(V, bool) (V, otter.ComputeOp)) (V, bool)) which calls the otter
// compute method with the locked function. Nil calls otter's Compute.
//
// Returns computeOutcome[V] which records what the compute did.
//
// Concurrency: The compute callback runs under otter's bucket lock and must not call back
// into the cache.
func (a *OtterAdapter[K, V]) runCompute(
	ctx context.Context,
	key K,
	request computeRequest[K, V],
	invoke func(locked func(V, bool) (V, otter.ComputeOp)) (V, bool),
) computeOutcome[V] {
	state := &computeState[K, V]{
		request:  request,
		override: nil,
		outcome: computeOutcome[V]{
			value:      *new(V),
			err:        nil,
			wait:       noWait,
			op:         otter.CancelOp,
			present:    false,
			ran:        false,
			ttlIgnored: false,
		},
	}
	logging := a.logsWrites()
	if logging {
		a.checkpointMu.RLock()
		defer a.checkpointMu.RUnlock()
	}
	defer a.releaseComputeState(key, state)

	locked := func(oldValue V, found bool) (V, otter.ComputeOp) {
		return a.lockedCompute(ctx, key, state, logging, oldValue, found)
	}
	if invoke == nil {
		state.outcome.value, state.outcome.present = a.client.Compute(key, locked)
	} else {
		state.outcome.value, state.outcome.present = invoke(locked)
	}
	return state.outcome
}

// lockedCompute is the body of a compute, run by otter under its bucket lock: it works
// out the new value, logs the write when the cache keeps a log, updates the indexes and
// hands any TTL to the expiry calculator of the same write.
//
// Takes key (K) which identifies the entry.
// Takes state (*computeState[K, V]) which holds the request and receives the outcome.
// Takes logging (bool) which reports whether the cache keeps a log.
// Takes oldValue (V) which is the current value.
// Takes found (bool) which reports whether the entry exists.
//
// Returns V which is the value otter stores.
// Returns otter.ComputeOp which is the operation otter applies.
func (a *OtterAdapter[K, V]) lockedCompute(ctx context.Context, key K, state *computeState[K, V], logging bool, oldValue V, found bool) (V, otter.ComputeOp) {
	request := &state.request
	state.outcome.ran = true
	newValue, op, ttl := request.apply(oldValue, found)
	resolved := a.resolveTTL(ttl)
	state.outcome.ttlIgnored = resolved.unsupported
	if logging {
		wait, err := a.logComputeLocked(ctx, key, request, computedWrite[V]{
			value:    newValue,
			tags:     request.tags,
			duration: resolved.duration,
			op:       op,
			found:    found,
			hasTTL:   resolved.applies,
		})
		if err != nil {
			state.outcome.err = err
			state.outcome.op = otter.CancelOp
			return oldValue, otter.CancelOp
		}
		state.outcome.wait = wait
	}
	state.outcome.op = op
	a.indexComputeLocked(key, newValue, op, found, request)
	if op == otter.WriteOp && resolved.applies {
		state.override = a.expiryOverrides.store(key, resolved.duration)
	}
	return newValue, op
}

// releaseComputeState releases what a compute held once otter has returned: the prepared
// log entry when it was never queued, and the TTL override.
//
// Takes key (K) which identifies the entry.
// Takes state (*computeState[K, V]) which holds what the compute held.
func (a *OtterAdapter[K, V]) releaseComputeState(key K, state *computeState[K, V]) {
	state.request.prepared.releaseUnused()
	if state.override != nil {
		a.expiryOverrides.discard(key, state.override)
	}
}

// indexComputeLocked updates tag and search indexes under otter's bucket lock.
//
// This keeps the indexes aligned with the stored value during concurrent writes.
//
// Must be called inside the otter compute callback. The indexes only take their own locks
// and never call back into the cache, so this cannot deadlock.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is the value the compute stores.
// Takes op (otter.ComputeOp) which is the operation being applied.
// Takes found (bool) which reports whether the entry existed.
// Takes request (*computeRequest[K, V]) which describes the write.
func (a *OtterAdapter[K, V]) indexComputeLocked(key K, value V, op otter.ComputeOp, found bool, request *computeRequest[K, V]) {
	switch {
	case op == otter.WriteOp:
		a.tagIndex.replace(key, request.tags)
		if request.keepsValue || !a.indexesSearch() {
			return
		}
		if request.remap == nil && request.search != nil {
			a.commitSearchEntry(key, request.search)
			return
		}
		a.indexDocument(key, value)
	case op == otter.InvalidateOp && !found:
		a.tagIndex.RemoveKey(key)
		a.removeFromSearchIndex(key)
	}
}

// finishCompute completes a compute outside otter's lock: it logs any warning the compute
// raised, waits for the log, and checkpoints when due.
//
// Takes outcome (computeOutcome[V]) which records what the compute did.
//
// Returns V which is the value present afterwards.
// Returns bool which reports whether a value is present afterwards.
// Returns error when the write could not be logged or the wait for the log failed.
func (a *OtterAdapter[K, V]) finishCompute(ctx context.Context, outcome computeOutcome[V]) (V, bool, error) {
	if outcome.ttlIgnored {
		a.warnTTLUnsupported(ctx)
	}
	if outcome.err != nil {
		return outcome.value, outcome.present, fmt.Errorf("logging write: %w", outcome.err)
	}
	if err := outcome.wait(); err != nil {
		return outcome.value, outcome.present,
			fmt.Errorf("waiting for the logged write, which the cache holds but may not keep after a restart: %w", err)
	}
	a.maybeCheckpoint()
	return outcome.value, outcome.present, nil
}

// logComputeLocked records the result of a compute in the log.
//
// Must be called inside the otter compute callback with checkpointMu held for reading.
//
// Takes key (K) which identifies the entry.
// Takes request (*computeRequest[K, V]) which carries the batch and any prepared entry.
// Takes write (computedWrite[V]) which describes what the compute applies.
//
// Returns func() error which waits for the entry to be written.
// Returns error when the entry cannot be logged.
func (a *OtterAdapter[K, V]) logComputeLocked(ctx context.Context, key K, request *computeRequest[K, V], write computedWrite[V]) (func() error, error) {
	switch {
	case write.op == otter.WriteOp:
		tags := write.tags
		if len(tags) == 0 {
			tags = a.tagIndex.GetTags(key)
		}
		expiresAt := a.loggedExpiry(key, write.duration, write.hasTTL)
		if prepared := &request.prepared; prepared.matches(expiresAt, tags) {
			return a.enqueuePreparedLocked(ctx, prepared)
		}
		return a.enqueueLocked(ctx, request.batch, wal_domain.Entry[K, V]{
			Operation: wal_domain.OpSet,
			Key:       key,
			Value:     write.value,
			Tags:      tags,
			Timestamp: a.nowNano(),
			ExpiresAt: expiresAt,
			Batch:     nil,
		})
	case write.op == otter.InvalidateOp && (write.found || request.logsMissingRemoval):
		entry := wal_domain.Entry[K, V]{}
		entry.Operation = wal_domain.OpDelete
		entry.Key = key
		entry.Timestamp = a.nowNano()
		return a.enqueueLocked(ctx, request.batch, entry)
	}
	return noWait, nil
}

// enqueuePreparedLocked queues an entry encoded before the lock was taken.
//
// Must be called with checkpointMu held for reading, and only when the log supports
// deferred writes.
//
// Takes prepared (*preparedEntry[V]) which holds the encoded entry.
//
// Returns func() error which waits for the entry to be written; it is never nil.
// Returns error when the entry cannot be queued, including when the log's queue stays
// full for its queue timeout.
func (a *OtterAdapter[K, V]) enqueuePreparedLocked(ctx context.Context, prepared *preparedEntry[V]) (func() error, error) {
	deferred, ok := a.wal.(wal_domain.DeferredAppender[K, V])
	if !ok {
		return noWait, errPreparedWithoutDeferredLog
	}
	prepared.used = true
	wait, err := deferred.QueueEncoded(detachContext(ctx), prepared.encoded)
	if err != nil {
		return noWait, err
	}
	return wait, nil
}

// enqueueLocked adds entry to batch, queues it on a log that supports deferred writes, or
// else writes it at once.
//
// A deferred log only queues the entry on a channel, waiting a bounded time when the
// channel is full, and its writer never touches the cache, so queueing under otter's
// bucket lock cannot deadlock, does no file I/O and cannot block for long. A log without
// deferred writes is written under the lock.
//
// Must be called with checkpointMu held for reading.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes entry (wal_domain.Entry[K, V]) which is the entry to log.
//
// Returns func() error which waits for the entry to be written; it is never nil.
// Returns error when the entry cannot be logged.
func (a *OtterAdapter[K, V]) enqueueLocked(ctx context.Context, batch *logBatch[K, V], entry wal_domain.Entry[K, V]) (func() error, error) {
	if batch != nil && batch.add(entry) {
		return noWait, nil
	}
	if deferred, ok := a.wal.(wal_domain.DeferredAppender[K, V]); ok {
		encoded, err := deferred.Encode(entry)
		if err != nil {
			return noWait, err
		}
		wait, err := deferred.QueueEncoded(detachContext(ctx), encoded)
		if err != nil {
			return noWait, err
		}
		return wait, nil
	}
	return noWait, a.appendEntries(ctx, []wal_domain.Entry[K, V]{entry})
}

// loggedExpiry returns the expiry to record in the log for a write of key.
//
// The log holds zero when the cache's policy decides the expiry on replay, math.MaxInt64
// when the entry never expires, and otherwise the absolute expiry in Unix nanoseconds on
// the cache's clock. The record format is unchanged, so an older reader sees
// math.MaxInt64 as an expiry a century away.
//
// The TTL is capped against the clock reading used here. A TTL near math.MaxInt64 would
// otherwise wrap the logged expiry into the past and drop the entry on replay.
//
// Takes key (K) which identifies the entry being written.
// Takes duration (time.Duration) which is the TTL of the write, valid when hasTTL.
// Takes hasTTL (bool) which reports whether the write carries a TTL.
//
// Returns int64 which is the expiry to log.
func (a *OtterAdapter[K, V]) loggedExpiry(key K, duration time.Duration, hasTTL bool) int64 {
	if hasTTL {
		if duration == cache_dto.NoExpiry {
			return math.MaxInt64
		}
		now := a.nowNano()
		return now + int64(capTTL(duration, now))
	}
	if a.perEntry {
		if entry, ok := a.client.GetEntryQuietly(key); ok {
			return entry.ExpiresAtNano
		}
	}
	return 0
}

// signalLog wakes the log's writer for entries queued under the bucket lock. Writes that
// wait for their entry need not call it, since the wait signals first; a write that hands
// its wait to the caller, or queues several entries, signals once its locks are released.
func (a *OtterAdapter[K, V]) signalLog() {
	if deferred, ok := a.wal.(wal_domain.DeferredAppender[K, V]); ok {
		deferred.Signal()
	}
}

// logsWrites reports whether the cache records its writes in a write-ahead log.
//
// Returns bool which is true when persistence is active.
func (a *OtterAdapter[K, V]) logsWrites() bool {
	return a.walEnabled && a.wal != nil
}

// nowNano returns the current time on the cache's clock.
//
// Returns int64 which is the current time in Unix nanoseconds.
func (a *OtterAdapter[K, V]) nowNano() int64 {
	if a.clock != nil {
		return a.clock.Now().UnixNano()
	}
	return time.Now().UnixNano()
}

// writeWithExpiry stores value with a TTL without logging it, for restoring entries from
// persistence.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is the value to store.
// Takes duration (time.Duration) which is the TTL, or cache_dto.NoExpiry.
func (a *OtterAdapter[K, V]) writeWithExpiry(key K, value V, duration time.Duration) {
	if !a.expiryEnabled {
		a.client.Set(key, value)
		return
	}
	var override *expiryOverride
	a.client.Compute(key, func(V, bool) (V, otter.ComputeOp) {
		override = a.expiryOverrides.store(key, duration)
		return value, otter.WriteOp
	})
	if override != nil {
		a.expiryOverrides.discard(key, override)
	}
}

// Invalidate removes a key from the cache.
//
// When the cache keeps a log the removal goes through the locked compute, so it reaches
// the log in the order it was applied relative to writes of the same key.
//
// Takes key (K) which identifies the cache entry to invalidate.
//
// Returns error when the removal cannot be logged; the cache is then left unchanged.
func (a *OtterAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	return a.invalidate(ctx, nil, key)
}

// invalidate removes a key, joining batch when one is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the removal joins; nil when none.
// Takes key (K) which identifies the cache entry to remove.
//
// Returns error when the removal cannot be logged.
func (a *OtterAdapter[K, V]) invalidate(ctx context.Context, batch *logBatch[K, V], key K) error {
	if !a.logsWrites() {
		a.tagIndex.RemoveKey(key)
		a.removeFromSearchIndex(key)
		a.client.Invalidate(key)
		return nil
	}

	_, _, err := a.computeWithExpiry(ctx, key, computeRequest[K, V]{
		value: *new(V),
		remap: func(oldValue V, _ bool) (V, otter.ComputeOp, time.Duration) {
			return oldValue, otter.InvalidateOp, 0
		},
		batch:              batch,
		prepared:           preparedEntry[V]{},
		search:             nil,
		ttl:                0,
		tags:               nil,
		keepsValue:         false,
		logsMissingRemoval: true,
	})
	if err != nil {
		return fmt.Errorf("removing key: %w", err)
	}
	return nil
}

// Compute computes and atomically updates the value in the cache using the provided
// function.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(V, bool) (V, cache_dto.ComputeAction)) which computes the
// new value and action from the old value and presence flag. It runs under a key lock and
// must not call back into the cache.
//
// Returns V which is the computed value.
// Returns bool which indicates whether the value exists in the cache.
// Returns error when logging fails. A failed log wait can leave the write in memory.
func (a *OtterAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return a.compute(ctx, nil, key, computeFunction)
}

// compute runs a compute, joining batch when one is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(V, bool) (V, cache_dto.ComputeAction)) which computes the
// new value and action.
//
// Returns V which is the computed value.
// Returns bool which indicates whether the value exists in the cache.
// Returns error when the write cannot be logged.
func (a *OtterAdapter[K, V]) compute(ctx context.Context, batch *logBatch[K, V], key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return a.computeWithExpiry(ctx, key, computeRequest[K, V]{
		value: *new(V),
		remap: func(oldValue V, found bool) (V, otter.ComputeOp, time.Duration) {
			newValue, action := computeFunction(oldValue, found)
			return newValue, actionToOp(action), 0
		},
		batch:              batch,
		prepared:           preparedEntry[V]{},
		search:             nil,
		ttl:                0,
		tags:               nil,
		keepsValue:         false,
		logsMissingRemoval: false,
	})
}

// ComputeIfAbsent computes and stores a value if the key is not present.
//
// Takes key (K) which is the cache key to look up or store.
// Takes computeFunction (func()) which computes the value if the key is absent. It runs
// under a lock on the key and must not call back into the cache.
//
// Returns V which is either the existing value or the newly computed one.
// Returns bool which indicates whether a value is present afterwards.
// Returns error when logging fails. A failed log wait can leave the write in memory.
func (a *OtterAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	return a.computeIfAbsent(ctx, nil, key, computeFunction)
}

// computeIfAbsent stores a computed value when the key is absent, joining batch when one
// is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes key (K) which is the cache key to look up or store.
// Takes computeFunction (func()) which computes the value if the key is absent.
//
// Returns V which is either the existing value or the newly computed one.
// Returns bool which indicates whether a value is present afterwards.
// Returns error when the write cannot be logged.
func (a *OtterAdapter[K, V]) computeIfAbsent(ctx context.Context, batch *logBatch[K, V], key K, computeFunction func() V) (V, bool, error) {
	request := computeRequest[K, V]{
		value: *new(V),
		remap: func(oldValue V, found bool) (V, otter.ComputeOp, time.Duration) {
			if found {
				return oldValue, otter.CancelOp, 0
			}
			return computeFunction(), otter.WriteOp, 0
		},
		batch:              batch,
		prepared:           preparedEntry[V]{},
		search:             nil,
		ttl:                0,
		tags:               nil,
		keepsValue:         false,
		logsMissingRemoval: false,
	}
	outcome := a.runCompute(ctx, key, request, func(locked func(V, bool) (V, otter.ComputeOp)) (V, bool) {
		return a.client.ComputeIfAbsent(key, func() (V, bool) {
			var zero V
			value, op := locked(zero, false)
			return value, op != otter.WriteOp
		})
	})
	return a.finishCompute(ctx, outcome)
}

// ComputeIfPresent updates a value only if the key is present.
//
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(V) (V, cache_dto.ComputeAction)) which computes the new
// value and action from the old value. It runs under a key lock and must not call back
// into the cache.
//
// Returns V which is the computed value, or the zero value if the key was absent.
// Returns bool which is true if the key was present and the value was updated.
// Returns error when logging fails. A failed log wait can leave the write in memory.
func (a *OtterAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	return a.computeIfPresent(ctx, nil, key, computeFunction)
}

// computeIfPresent updates a present value, joining batch when one is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(V) (V, cache_dto.ComputeAction)) which computes the new
// value and action from the old value.
//
// Returns V which is the computed value, or the zero value if the key was absent.
// Returns bool which is true if the key was present and the value was updated.
// Returns error when the write cannot be logged.
func (a *OtterAdapter[K, V]) computeIfPresent(ctx context.Context, batch *logBatch[K, V], key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	request := computeRequest[K, V]{
		value: *new(V),
		remap: func(oldValue V, _ bool) (V, otter.ComputeOp, time.Duration) {
			newValue, action := computeFunction(oldValue)
			return newValue, actionToOp(action), 0
		},
		batch:              batch,
		prepared:           preparedEntry[V]{},
		search:             nil,
		ttl:                0,
		tags:               nil,
		keepsValue:         false,
		logsMissingRemoval: false,
	}
	outcome := a.runCompute(ctx, key, request, func(locked func(V, bool) (V, otter.ComputeOp)) (V, bool) {
		return a.client.ComputeIfPresent(key, func(oldValue V) (V, otter.ComputeOp) {
			return locked(oldValue, true)
		})
	})
	return a.finishCompute(ctx, outcome)
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// The TTL is applied by the same locked write that stores the value, and only when the
// action is ComputeActionSet. A zero TTL leaves the entry's expiry to the cache's expiry
// calculator; under PerEntryExpiration that keeps the current expiry of an existing
// entry. cache_dto.NoExpiry removes any expiry.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(V, bool) cache_dto.ComputeResult[V]) which computes the new
// value, action and optional TTL. It runs under a key lock and must not call back into
// the cache.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when logging fails. A failed log wait can leave the write in memory.
func (a *OtterAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	return a.computeWithTTL(ctx, nil, key, computeFunction)
}

// computeWithTTL runs a compute with TTL control, joining batch when one is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(V, bool) cache_dto.ComputeResult[V]) which computes the new
// value, action and TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the write cannot be logged.
func (a *OtterAdapter[K, V]) computeWithTTL(ctx context.Context, batch *logBatch[K, V], key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	return a.computeWithExpiry(ctx, key, computeRequest[K, V]{
		value: *new(V),
		remap: func(oldValue V, found bool) (V, otter.ComputeOp, time.Duration) {
			result := computeFunction(oldValue, found)
			return result.Value, actionToOp(result.Action), result.TTL
		},
		batch:              batch,
		prepared:           preparedEntry[V]{},
		search:             nil,
		ttl:                0,
		tags:               nil,
		keepsValue:         false,
		logsMissingRemoval: false,
	})
}

// BulkGet retrieves multiple values from the cache, loading any missing ones using the
// bulk loader.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader) which loads values for keys not in the cache.
//
// Returns map[K]V which contains the retrieved or loaded values.
// Returns error when the bulk load operation fails.
func (a *OtterAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) (map[K]V, error) {
	return a.client.BulkGet(ctx, keys, a.wrapBulkLoader(bulkLoader))
}

// BulkSet stores multiple key-value pairs in the cache with optional tags.
//
// When the cache keeps a log, or has tags or search indexes to keep, each key is written
// through the locked compute as Set does, so the log and the indexes follow the order the
// writes were applied. The keys are not written as one unit: when a write cannot be
// logged, the keys written before it stay.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags to associate with each key.
//
// Returns error when the context ends or a write cannot be logged.
func (a *OtterAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	return a.bulkSet(ctx, nil, items, tags)
}

// bulkSet stores several values, joining batch when one is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the writes join; nil when none.
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags ([]string) which are optional labels for every key.
//
// Returns error when the context ends or a write cannot be logged.
func (a *OtterAdapter[K, V]) bulkSet(ctx context.Context, batch *logBatch[K, V], items map[K]V, tags []string) error {
	if a.writesDirectly(tags) {
		for key, value := range items {
			a.client.Set(key, value)
		}
		return nil
	}

	wait, err := a.queueBulk(ctx, batch, items, tags)
	if err != nil {
		a.signalLog()
		return err
	}
	if err := wait(); err != nil {
		return fmt.Errorf("waiting for the logged bulk write, which the cache holds but may not keep after a restart: %w", err)
	}
	a.maybeCheckpoint()
	return nil
}

// queueBulk writes each item through the locked compute, queueing its log entry without
// waking the log's writer, so the whole bulk write reaches the writer at once.
//
// Takes batch (*logBatch[K, V]) which is the log batch the writes join; nil when none.
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags ([]string) which are optional labels for every key.
//
// Returns func() error which waits for the last entry, and so for every entry before it.
// Returns error when the context ends or a write cannot be logged; the entries queued
// before it stay queued and the caller must signal the log.
func (a *OtterAdapter[K, V]) queueBulk(ctx context.Context, batch *logBatch[K, V], items map[K]V, tags []string) (func() error, error) {
	wait := noWait
	for key, value := range items {
		if err := ctx.Err(); err != nil {
			return noWait, fmt.Errorf("bulk write: %w", err)
		}
		outcome := a.runCompute(ctx, key, a.plainWrite(batch, key, value, 0, resolvedTTL{}, tags), nil)
		if outcome.err != nil {
			return noWait, fmt.Errorf("logging bulk write: %w", outcome.err)
		}
		wait = outcome.wait
	}
	return wait, nil
}

// InvalidateByTags removes all entries with matching tags from the cache.
//
// When the run stops early, because the context ended or a removal could not be logged,
// the keys it did not remove keep their tags, so a later call can still find them.
//
// Takes tags (...string) which specifies the tags to match for invalidation.
//
// Returns int which is the number of keys that were invalidated.
// Returns error when the context ends or a removal cannot be logged.
func (a *OtterAdapter[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("invalidating tagged keys: %w", err)
	}

	keys, previousTags := a.tagIndex.invalidate(tags, a.logsWrites())
	if len(keys) == 0 {
		return 0, nil
	}

	removed, err := a.invalidateKeys(ctx, keys)
	if removed < len(keys) {
		for _, key := range keys[removed:] {
			a.tagIndex.restoreIfUntagged(key, previousTags[key])
		}
	}

	tagInvalidationsTotal.Add(ctx, 1)
	invalidatedKeysByTagTotal.Add(ctx, int64(removed))
	l.Trace("Invalidated keys by tags",
		logger_domain.Int("tag_count", len(tags)),
		logger_domain.Int("key_count", removed))

	return removed, err
}

// invalidateKeys removes and logs keys, checking the context and the checkpoint trigger
// between chunks so a tag matching many keys cannot run on unchecked.
//
// Takes keys ([]K) which identifies the entries to remove.
//
// Returns int which is how many keys were removed before the context ended.
// Returns error which is the context's error when the run was cut short, or the log's
// when a removal could not be logged.
func (a *OtterAdapter[K, V]) invalidateKeys(ctx context.Context, keys []K) (int, error) {
	if !a.logsWrites() {
		for _, key := range keys {
			a.tagIndex.RemoveKey(key)
			a.removeFromSearchIndex(key)
			a.client.Invalidate(key)
		}

		return len(keys), nil
	}

	removed := 0
	for start := 0; start < len(keys); start += invalidateChunkSize {
		if err := ctx.Err(); err != nil {
			return removed, fmt.Errorf("invalidating tagged keys: %w", err)
		}

		end := min(start+invalidateChunkSize, len(keys))
		for _, key := range keys[start:end] {
			if err := a.invalidate(ctx, nil, key); err != nil {
				return removed, fmt.Errorf("invalidating tagged keys: %w", err)
			}
			removed++
		}
	}

	return removed, nil
}

// InvalidateAll clears all entries from the cache.
//
// Returns error which is always nil for the in-memory adapter.
//
// Safe for concurrent use. The operation is protected by a read lock during WAL
// operations.
func (a *OtterAdapter[K, V]) InvalidateAll(ctx context.Context) error {
	if a.walEnabled && a.wal != nil {
		a.checkpointMu.RLock()
		entry := wal_domain.Entry[K, V]{}
		entry.Operation = wal_domain.OpClear
		entry.Timestamp = a.nowNano()
		if err := a.logEntries(ctx, nil, entry); err != nil {
			a.checkpointMu.RUnlock()
			return fmt.Errorf("logging clear: %w", err)
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
// The context is detached through daemon_dto.DetachRequestContext before the loader runs,
// because the refresh can outlive the calling request.
//
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads fresh values for the keys.
func (a *OtterAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) {
	a.client.BulkRefresh(daemon_dto.DetachRequestContext(ctx), keys, a.wrapBulkLoader(bulkLoader))
}

// Refresh asynchronously reloads a single key using the provided loader.
//
// The context is detached through daemon_dto.DetachRequestContext before the loader runs,
// because the refresh can outlive the calling request.
//
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which loads the fresh value.
//
// Returns <-chan cache_dto.LoadResult[V] which delivers the refresh result.
//
// Safe for concurrent use. Spawns a goroutine that converts the otter result channel into
// a DTO result channel.
func (a *OtterAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V] {
	ctx = daemon_dto.DetachRequestContext(ctx)
	otterResultChan := a.client.Refresh(ctx, key, a.wrapLoader(loader))

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
		value, err := unwrapLoadOutcome(otterResult.Value, otterResult.Err)
		dtoResultChan <- cache_dto.LoadResult[V]{
			Value: value,
			Err:   err,
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
// Returns cache_dto.Entry[K, V] which contains the entry with metadata.
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
// Returns cache_dto.Entry[K, V] which contains the entry with metadata.
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
// Returns cache_dto.Stats which contains counts for hits, misses, and evictions.
func (a *OtterAdapter[K, V]) Stats() cache_dto.Stats {
	otterStats := a.client.Stats()

	return cache_dto.Stats{
		Hits:             otterStats.Hits,
		Misses:           otterStats.Misses,
		Evictions:        otterStats.Evictions,
		LoadSuccessCount: otterStats.LoadSuccesses,
		LoadFailureCount: otterStats.LoadFailures,
		TotalLoadTime:    otterStats.TotalLoadTime,
	}
}

// maybeCheckpoint checks if the WAL entry count has reached the snapshot threshold and if
// so, creates a snapshot and truncates the WAL. This compacts the WAL to prevent
// unbounded growth.
//
// Safe for concurrent use. Skips the checkpoint while a log batch is open, since the
// cache then holds writes the log does not yet have, or while another checkpoint runs.
// Acquires a write lock that blocks all writes during the checkpoint operation to ensure
// snapshot consistency.
func (a *OtterAdapter[K, V]) maybeCheckpoint() {
	if !a.walEnabled || a.wal == nil || a.snapshot == nil {
		return
	}
	if !a.checkpointDue() {
		return
	}
	if !a.logBatchGate.TryLock() {
		return
	}
	defer a.logBatchGate.Unlock()

	a.checkpointMu.Lock()
	defer a.checkpointMu.Unlock()

	if !a.checkpointDue() {
		return
	}

	_ = a.performCheckpointLocked()
}

// checkpointDue reports whether the WAL has passed either compaction trigger.
//
// Returns bool which is true when a checkpoint should be taken.
func (a *OtterAdapter[K, V]) checkpointDue() bool {
	if a.snapshotThreshold > 0 && a.wal.EntryCount() >= a.snapshotThreshold {
		return true
	}

	return a.maxWALSize > 0 && a.wal.Size() >= a.maxWALSize
}

// Checkpoint unconditionally writes a snapshot of the current cache state and truncates
// the WAL without closing the adapter.
//
// Returns error when the snapshot save or WAL truncate fails, so an explicit flush
// (unlike the best-effort threshold checkpoint) can fail loudly. Returns nil when
// persistence is disabled (nothing to flush).
//
// Concurrency: safe for concurrent use. Acquires checkpointMu so the snapshot captures
// all WAL entries before truncation.
func (a *OtterAdapter[K, V]) Checkpoint(_ context.Context) error {
	if !a.walEnabled || a.wal == nil || a.snapshot == nil {
		return nil
	}
	a.logBatchGate.Lock()
	defer a.logBatchGate.Unlock()
	a.checkpointMu.Lock()
	defer a.checkpointMu.Unlock()
	return a.performCheckpointLocked()
}

// performCheckpointLocked creates a snapshot of the current cache state and truncates the
// WAL.
//
// Called when the WAL entry count exceeds the snapshot threshold, on close, and on an
// explicit Checkpoint. Caller must hold checkpointMu write lock.
//
// Returns error when the snapshot save or WAL truncate fails. The error is also logged so
// best-effort callers (threshold checkpoint, close) can ignore it.
func (a *OtterAdapter[K, V]) performCheckpointLocked() error {
	ctx := context.Background()
	ctx, l := logger_domain.From(ctx, log)

	entries := a.collectSnapshotEntries()

	if err := a.snapshot.Save(ctx, entries); err != nil {
		l.Warn("Failed to save snapshot during checkpoint", logger_domain.Error(err))
		return err
	}

	if err := a.wal.Truncate(ctx); err != nil {
		l.Warn("Failed to truncate WAL after checkpoint", logger_domain.Error(err))
		return err
	}

	l.Internal("Checkpoint completed",
		logger_domain.Int("entries_snapshot", len(entries)))
	return nil
}

// SnapshotEntries returns the cache's current entries as a snapshot holds them, for
// writing a snapshot somewhere other than the cache's own directory.
//
// Returns []wal_domain.Entry[K, V] which holds every live entry with its tags and expiry.
func (a *OtterAdapter[K, V]) SnapshotEntries() []wal_domain.Entry[K, V] {
	return a.collectSnapshotEntries()
}

// collectSnapshotEntries collects all current cache entries for snapshotting.
//
// Each entry keeps its absolute expiry, so a restart restores the TTL it had left rather
// than applying the cache's policy afresh. A cache without expiry support records zero.
//
// Returns []wal_domain.Entry[K, V] which contains all current cache entries with their
// tags, expiries and timestamps.
func (a *OtterAdapter[K, V]) collectSnapshotEntries() []wal_domain.Entry[K, V] {
	entries := make([]wal_domain.Entry[K, V], 0, a.client.EstimatedSize())
	nowNano := a.nowNano()

	for key := range a.client.Keys() {
		cached, ok := a.client.GetEntryQuietly(key)
		if !ok {
			continue
		}
		entry := wal_domain.Entry[K, V]{
			Operation: wal_domain.OpSet,
			Key:       key,
			Value:     cached.Value,
			Timestamp: nowNano,
			Tags:      nil,
			ExpiresAt: 0,
			Batch:     nil,
		}
		if a.expiryEnabled {
			entry.ExpiresAt = cached.ExpiresAtNano
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
// closeOnce guarantees that the final checkpoint and underlying store closes happen
// exactly once; subsequent invocations return the captured error without re-running the
// close path.
//
// Returns error which contains any joined errors from closing the WAL and snapshot store;
// nil when the cache is purely in-memory.
func (a *OtterAdapter[K, V]) Close(_ context.Context) error {
	a.closeOnce.Do(func() {
		if a.walEnabled {
			a.closeErr = a.closePersistenceLocked()
		}
		a.client.StopAllGoroutines()
	})
	return a.closeErr
}

// closePersistenceLocked finalises the WAL by checkpointing any pending entries, closing
// the WAL and snapshot store, and joining any errors from the two closes.
//
// Returns error which is the joined set of WAL/snapshot close errors, or nil when both
// close cleanly.
//
// Concurrency: acquires checkpointMu while flushing pending WAL entries before closing
// the WAL and snapshot stores.
func (a *OtterAdapter[K, V]) closePersistenceLocked() error {
	a.logBatchGate.Lock()
	defer a.logBatchGate.Unlock()
	a.checkpointMu.Lock()
	if a.wal != nil && a.snapshot != nil && a.wal.EntryCount() > 0 {
		_ = a.performCheckpointLocked()
	}
	a.checkpointMu.Unlock()

	var walErr, snapshotErr error
	if a.wal != nil {
		if err := a.wal.Close(); err != nil {
			walErr = fmt.Errorf("closing WAL: %w", err)
		}
	}
	if a.snapshot != nil {
		if err := a.snapshot.Close(); err != nil {
			snapshotErr = fmt.Errorf("closing snapshot store: %w", err)
		}
	}
	return errors.Join(walErr, snapshotErr)
}

// SetExpiresAfter changes an existing key's expiry without creating the key.
//
// The update behaves as a write. It cancels an in-progress refresh, treats the entry as
// recently written for eviction, reports cache_dto.CauseReplacement to deletion handlers,
// reruns the weigher and counts a hit or miss in cache statistics.
//
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies how long until the key expires; zero
// or negative does nothing, and cache_dto.NoExpiry removes any expiry.
//
// Returns error when logging fails. A failed log wait can leave the write in memory.
func (a *OtterAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	return a.setExpiresAfter(ctx, nil, key, expiresAfter)
}

// setExpiresAfter changes the expiry of an existing key, joining batch when one is given.
//
// Takes batch (*logBatch[K, V]) which is the log batch the write joins; nil when none.
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies how long until the key expires.
//
// Returns error when the write cannot be logged.
func (a *OtterAdapter[K, V]) setExpiresAfter(ctx context.Context, batch *logBatch[K, V], key K, expiresAfter time.Duration) error {
	resolved := a.resolveTTL(expiresAfter)
	if resolved.unsupported {
		a.warnTTLUnsupported(ctx)
	}
	if !resolved.applies {
		return nil
	}
	_, _, err := a.computeWithExpiry(ctx, key, computeRequest[K, V]{
		value: *new(V),
		remap: func(oldValue V, found bool) (V, otter.ComputeOp, time.Duration) {
			if !found {
				return oldValue, otter.CancelOp, 0
			}
			return oldValue, otter.WriteOp, expiresAfter
		},
		batch:              batch,
		prepared:           preparedEntry[V]{},
		search:             nil,
		ttl:                0,
		tags:               nil,
		keepsValue:         true,
		logsMissingRemoval: false,
	})
	return err
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

// IsWeightBounded reports whether this cache is bounded by weight rather than entry
// count.
//
// Returns bool which is true when the cache was configured with a weigher and a weight
// bound.
func (a *OtterAdapter[K, V]) IsWeightBounded() bool {
	return a.weightBounded
}

// SetRefreshableAfter sets the duration after which a key becomes refreshable.
//
// Takes key (K) which identifies the cache entry to configure.
// Takes refreshableAfter (time.Duration) which specifies when the key becomes eligible
// for refresh.
//
// Returns error which is always nil for the in-memory adapter.
func (a *OtterAdapter[K, V]) SetRefreshableAfter(_ context.Context, key K, refreshableAfter time.Duration) error {
	a.client.SetRefreshableAfter(key, refreshableAfter)
	return nil
}

// actionToOp converts a ComputeAction to its corresponding otter.ComputeOp value.
//
// Takes action (cache_dto.ComputeAction) which is the action to convert.
//
// Returns otter.ComputeOp which is the mapped operation. Unknown actions return CancelOp
// by default.
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

// matches reports whether a prepared entry still describes the write under the lock, so
// it can be queued as it is.
//
// Takes expiresAt (int64) which is the expiry the write logs under the lock.
// Takes tags ([]string) which are the tags the write logs under the lock.
//
// Returns bool which is false for an entry that is not ready or when the state it was
// encoded from has changed since.
func (p *preparedEntry[V]) matches(expiresAt int64, tags []string) bool {
	if !p.ready {
		return false
	}
	if p.checksExpiry && p.expiresAt != expiresAt {
		return false
	}
	return !p.checksTags || sameTags(p.tags, tags)
}

// releaseUnused releases the encoded entry when it was made but never queued.
func (p *preparedEntry[V]) releaseUnused() {
	if p.ready && !p.used {
		p.encoded.Release()
	}
}

// apply works out what the request writes, given the current value.
//
// Takes oldValue (V) which is the current value.
// Takes found (bool) which reports whether the entry exists.
//
// Returns V which is the value to store.
// Returns otter.ComputeOp which is the operation to apply.
// Returns time.Duration which is the requested TTL.
func (r *computeRequest[K, V]) apply(oldValue V, found bool) (V, otter.ComputeOp, time.Duration) {
	if r.remap == nil {
		return r.value, otter.WriteOp, r.ttl
	}
	return r.remap(oldValue, found)
}

// capTTL limits a TTL so that adding it to now cannot overflow, which would wrap the
// expiry to a time in the past and expire the entry at once.
//
// The cap only holds for nowNano, so callers pass the same clock reading they then add
// the TTL to.
//
// Takes ttl (time.Duration) which is the requested TTL.
// Takes nowNano (int64) which is the current time in Unix nanoseconds.
//
// Returns time.Duration which is ttl, or the longest TTL that stays representable.
func capTTL(ttl time.Duration, nowNano int64) time.Duration {
	if nowNano <= 0 {
		return ttl
	}
	return min(ttl, time.Duration(math.MaxInt64-nowNano-1))
}

// sameTags reports whether two tag lists hold the same tags, in any order.
//
// Takes left ([]string) which is the first list.
// Takes right ([]string) which is the second list.
//
// Returns bool which is true when both hold the same set of tags.
func sameTags(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	set := make(map[string]struct{}, len(left))
	for _, tag := range left {
		set[tag] = struct{}{}
	}
	for _, tag := range right {
		if _, ok := set[tag]; !ok {
			return false
		}
	}
	return true
}

// detachContext keeps ctx values but removes cancellation so a queued log write can
// finish.
//
// Returns context.Context which keeps the values of ctx and is never cancelled.
func detachContext(ctx context.Context) context.Context {
	if ctx.Done() == nil {
		return ctx
	}
	return context.WithoutCancel(ctx)
}
