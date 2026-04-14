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
	"time"

	"github.com/maypok86/otter/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/vectormaths"
	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
)

// entryData holds the final state of a cache entry after replay.
type entryData[V any] struct {
	// Value is the stored value for this cache entry.
	Value V

	// Tags contains the tag strings associated with this entry.
	Tags []string

	// ExpiresAt is the Unix timestamp when this entry expires; 0 means no expiry.
	ExpiresAt int64
}

// OtterProviderFactory is the factory function for creating Otter adapters
// that accepts typed Options and returns a properly configured OtterAdapter
// with cache_dto-to-otter type conversions handled by adapter wrappers.
//
// Takes options (cache_dto.Options[K, V]) which provides the cache
// configuration including size limits, expiry, and search schema.
//
// Returns cache_domain.ProviderPort[K, V] which is the configured
// Otter cache adapter.
// Returns error when the otter cache instance cannot be created.
func OtterProviderFactory[K comparable, V any](options cache_dto.Options[K, V]) (cache_domain.ProviderPort[K, V], error) {
	adapter := &OtterAdapter[K, V]{
		client:   nil,
		tagIndex: newTagIndex[K](),
	}

	if options.SearchSchema != nil {
		configureSearchSchema(adapter, options.SearchSchema)
	}

	otterOpts := buildOtterOptions(options, adapter)

	client, err := otter.New(otterOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create otter cache instance: %w", err)
	}

	adapter.client = client

	if err := initialisePersistence(context.Background(), options, adapter); err != nil {
		client.StopAllGoroutines()
		return nil, fmt.Errorf("failed to initialise persistence: %w", err)
	}

	return adapter, nil
}

// configureSearchSchema sets up the tag, inverted, sorted, and vector indexes
// on the adapter from the provided search schema.
//
// Takes adapter (*OtterAdapter[K, V]) which is the adapter to configure.
// Takes schema (*cache_dto.SearchSchema) which defines the index configuration.
func configureSearchSchema[K comparable, V any](adapter *OtterAdapter[K, V], schema *cache_dto.SearchSchema) {
	if schema.MaxTagsPerKey > 0 {
		adapter.tagIndex.maxTagsPerKey = schema.MaxTagsPerKey
	}

	adapter.schema = schema
	adapter.fieldExtractor = cache_domain.NewFieldExtractor[V](schema)
	adapter.invertedIndex = cache_domain.NewInvertedIndex[K]()
	if schema.MaxInvertedIndexTokens > 0 {
		adapter.invertedIndex.SetMaxTokens(schema.MaxInvertedIndexTokens)
	}
	if schema.TextAnalyser != nil {
		adapter.invertedIndex.SetAnalyseFunction(schema.TextAnalyser)
	}

	adapter.sortedIndexes = make(map[string]*cache_domain.SortedIndex[K])
	adapter.vectorIndexes = make(map[string]*cache_domain.VectorIndex[K])
	configureFieldIndexes(adapter, schema)
}

// configureFieldIndexes creates sorted and vector indexes for each field in the
// schema.
//
// Takes adapter (*OtterAdapter[K, V]) which receives the index instances.
// Takes schema (*cache_dto.SearchSchema) which defines the fields and their
// properties.
func configureFieldIndexes[K comparable, V any](adapter *OtterAdapter[K, V], schema *cache_dto.SearchSchema) {
	for _, field := range schema.Fields {
		if field.Sortable {
			adapter.sortedIndexes[field.Name] = cache_domain.NewSortedIndex[K]()
		}
		if field.Type == cache_dto.FieldTypeVector {
			metric := vectormaths.Cosine
			if field.DistanceMetric != "" {
				metric = vectormaths.Metric(field.DistanceMetric)
			}
			vi := cache_domain.NewVectorIndex[K](field.Dimension, metric)
			if schema.MaxVectors > 0 {
				vi.SetMaxVectors(schema.MaxVectors)
			}
			adapter.vectorIndexes[field.Name] = vi
		}
	}
}

// initialisePersistence initialises WAL and snapshot store if persistence is
// configured.
//
// Takes options (Options) which provides cache configuration including
// persistence settings.
// Takes adapter (*OtterAdapter) which is the cache adapter to configure.
//
// Returns error when persistence configuration is invalid, WAL creation fails,
// snapshot store creation fails, or recovery from persistence fails.
func initialisePersistence[K comparable, V any](ctx context.Context, options cache_dto.Options[K, V], adapter *OtterAdapter[K, V]) error {
	persistConfig, ok := options.ProviderSpecific.(PersistenceConfig[K, V])
	if !ok || !persistConfig.Enabled {
		return nil
	}

	if err := persistConfig.Validate(); err != nil {
		return fmt.Errorf("invalid persistence config: %w", err)
	}

	codec := driven_disk.NewBinaryCodec(persistConfig.KeyCodec, persistConfig.ValueCodec)

	wal, err := driven_disk.NewDiskWAL(ctx, persistConfig.WALConfig, codec)
	if err != nil {
		return fmt.Errorf("creating WAL: %w", err)
	}

	snapshot, err := driven_disk.NewDiskSnapshot(ctx, persistConfig.WALConfig, codec)
	if err != nil {
		_ = wal.Close()
		return fmt.Errorf("creating snapshot store: %w", err)
	}

	adapter.wal = wal
	adapter.snapshot = snapshot
	adapter.walEnabled = true
	adapter.snapshotThreshold = persistConfig.WALConfig.SnapshotThreshold

	if err := recoverFromPersistence(adapter); err != nil {
		_ = wal.Close()
		_ = snapshot.Close()
		return fmt.Errorf("recovering from persistence: %w", err)
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Persistence initialised",
		logger_domain.String("dir", persistConfig.WALConfig.Dir),
		logger_domain.Bool("compression", persistConfig.WALConfig.EnableCompression))

	return nil
}

// recoverFromPersistence loads data from snapshot and WAL using streaming APIs,
// replaying operations into the cache without loading all entries into memory.
//
// Takes adapter (*OtterAdapter) which provides the cache and persistence layer.
//
// Returns error when loading snapshot or WAL entries fails.
func recoverFromPersistence[K comparable, V any](adapter *OtterAdapter[K, V]) error {
	ctx := context.Background()
	ctx, l := logger_domain.From(ctx, log)

	nowNano := time.Now().UnixNano()
	state := make(map[K]entryData[V])

	snapshotEntries, err := loadSnapshotEntries(ctx, adapter.snapshot, state, nowNano)
	if err != nil {
		return fmt.Errorf("recovering snapshot entries: %w", err)
	}

	walEntries, err := loadWALEntries(ctx, adapter.wal, state, nowNano)
	if err != nil {
		return fmt.Errorf("recovering WAL entries: %w", err)
	}

	populateCacheFromState(adapter, state, nowNano)

	l.Internal("Cache recovered from persistence",
		logger_domain.Int("snapshot_entries", snapshotEntries),
		logger_domain.Int("wal_entries", walEntries),
		logger_domain.Int("final_keys", len(state)))

	return nil
}

// loadSnapshotEntries streams entries from snapshot into state.
//
// Takes snapshot (wal_domain.SnapshotStore) which provides the stored entries.
// Takes state (map[K]entryData) which receives the loaded entries.
// Takes nowNano (int64) which specifies the current time in nanoseconds.
//
// Returns int which is the number of entries loaded.
// Returns error when the snapshot cannot be read.
func loadSnapshotEntries[K comparable, V any](
	ctx context.Context,
	snapshot wal_domain.SnapshotStore[K, V],
	state map[K]entryData[V],
	nowNano int64,
) (int, error) {
	if !snapshot.Exists() {
		return 0, nil
	}

	var count int
	for entry, err := range snapshot.Load(ctx) {
		if err != nil {
			if errors.Is(err, wal_domain.ErrSnapshotNotFound) {
				break
			}
			return count, fmt.Errorf("loading snapshot: %w", err)
		}
		applyEntryToState(state, entry, nowNano)
		count++
	}
	return count, nil
}

// loadWALEntries streams entries from WAL into state.
//
// Takes wal (WAL[K, V]) which provides the write-ahead log to recover from.
// Takes state (map[K]entryData[V]) which receives the recovered entries.
// Takes nowNano (int64) which specifies the current time in nanoseconds.
//
// Returns int which is the number of entries loaded.
// Returns error when WAL recovery fails.
func loadWALEntries[K comparable, V any](
	ctx context.Context,
	wal wal_domain.WAL[K, V],
	state map[K]entryData[V],
	nowNano int64,
) (int, error) {
	var count int
	for entry, err := range wal.Recover(ctx) {
		if err != nil {
			return count, fmt.Errorf("recovering WAL: %w", err)
		}
		applyEntryToState(state, entry, nowNano)
		count++
	}
	return count, nil
}

// populateCacheFromState populates the cache from the recovered state.
//
// Takes adapter (*OtterAdapter[K, V]) which is the cache adapter to populate.
// Takes state (map[K]entryData[V]) which contains the recovered entries.
// Takes nowNano (int64) which is the current time in nanoseconds for expiry
// calculations.
func populateCacheFromState[K comparable, V any](adapter *OtterAdapter[K, V], state map[K]entryData[V], nowNano int64) {
	for key, data := range state {
		adapter.tagIndex.Add(key, data.Tags)
		adapter.client.Set(key, data.Value)
		adapter.indexDocument(key, data.Value)

		if data.ExpiresAt > 0 {
			remaining := time.Duration(data.ExpiresAt - nowNano)
			if remaining > 0 {
				adapter.client.SetExpiresAfter(key, remaining)
			}
		}
	}
}

// applyEntryToState applies a single WAL entry to the state map.
//
// Takes state (map[K]entryData[V]) which is the current state to modify.
// Takes entry (wal_domain.Entry[K, V]) which is the WAL entry to apply.
// Takes nowNano (int64) which is the current time in nanoseconds for expiry
// checks.
func applyEntryToState[K comparable, V any](state map[K]entryData[V], entry wal_domain.Entry[K, V], nowNano int64) {
	switch entry.Operation {
	case wal_domain.OpSet:
		if entry.ExpiresAt > 0 && entry.ExpiresAt < nowNano {
			return
		}
		state[entry.Key] = entryData[V]{
			Value:     entry.Value,
			Tags:      entry.Tags,
			ExpiresAt: entry.ExpiresAt,
		}

	case wal_domain.OpDelete:
		delete(state, entry.Key)

	case wal_domain.OpClear:
		clear(state)
	}
}

// buildOtterOptions constructs otter.Options from cache_dto.Options, wiring up
// the tag index cleanup handlers and converting interface types.
//
// Takes options (cache_dto.Options[K, V]) which provides the cache
// configuration to convert.
// Takes adapter (*OtterAdapter[K, V]) which provides access to
// the tag and search indexes for deletion handlers.
//
// Returns *otter.Options[K, V] which contains the converted otter
// configuration.
func buildOtterOptions[K comparable, V any](options cache_dto.Options[K, V], adapter *OtterAdapter[K, V]) *otter.Options[K, V] {
	otterOpts := &otter.Options[K, V]{
		MaximumSize:       options.MaximumSize,
		MaximumWeight:     options.MaximumWeight,
		InitialCapacity:   options.InitialCapacity,
		Weigher:           options.Weigher,
		Executor:          options.Executor,
		ExpiryCalculator:  wrapExpiryCalculator(options.ExpiryCalculator),
		RefreshCalculator: wrapRefreshCalculator(options.RefreshCalculator),
		StatsRecorder:     wrapStatsRecorder(options.StatsRecorder),
		Clock:             wrapClock(options.Clock),
		Logger:            wrapLogger(options.Logger),
	}

	otterOpts.OnDeletion = buildDeletionHandler(adapter, wrapOnDeletion(options.OnDeletion))
	otterOpts.OnAtomicDeletion = buildDeletionHandler(adapter, wrapOnAtomicDeletion(options.OnAtomicDeletion))

	return otterOpts
}

// buildDeletionHandler creates a handler for deletion events that removes
// entries from the tag index and search indexes, then calls any user-provided
// handler.
//
// Takes adapter (*OtterAdapter[K, V]) which provides access to the indexes.
// Takes userHandler (func(otter.DeletionEvent[K, V])) which is an optional
// callback to run after cleanup.
//
// Returns func(otter.DeletionEvent[K, V]) which handles deletion events by
// removing the key from all indexes and calling the user handler if provided.
func buildDeletionHandler[K comparable, V any](adapter *OtterAdapter[K, V], userHandler func(otter.DeletionEvent[K, V])) func(otter.DeletionEvent[K, V]) {
	return func(e otter.DeletionEvent[K, V]) {
		if e.Cause != otter.CauseReplacement {
			adapter.tagIndex.RemoveKey(e.Key)
			adapter.removeFromSearchIndex(e.Key)
		}
		if userHandler != nil {
			userHandler(e)
		}
	}
}
