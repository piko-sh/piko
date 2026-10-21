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
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"weak"

	"github.com/maypok86/otter/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/vectormaths"
)

const (
	// readOnlyLoadAttempts limits retries when an owner checkpoint interrupts a read-only
	// load.
	readOnlyLoadAttempts = 3
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

// OtterProviderFactory is the factory function for creating Otter adapters that accepts
// typed Options and returns a properly configured OtterAdapter with cache_dto-to-otter
// type conversions handled by adapter wrappers.
//
// Takes options (cache_dto.Options[K, V]) which provides the cache configuration
// including size limits, expiry, and search schema.
//
// Returns cache_domain.ProviderPort[K, V] which is the configured Otter cache adapter.
// Returns error when the otter cache instance cannot be created.
func OtterProviderFactory[K comparable, V any](options cache_dto.Options[K, V]) (cache_domain.ProviderPort[K, V], error) {
	if err := cache_domain.ValidateOptions(options); err != nil {
		return nil, err
	}

	cache_domain.WarnUnbounded(context.Background(), options)

	adapter := &OtterAdapter[K, V]{
		client:            nil,
		tagIndex:          newTagIndex[K](),
		weigher:           options.Weigher,
		onDeletion:        options.OnDeletion,
		closeErr:          nil,
		maxEntryWeight:    options.MaxEntryWeight,
		weightBounded:     options.Weigher != nil && options.MaximumWeight > 0,
		wal:               nil,
		snapshot:          nil,
		fieldExtractor:    nil,
		invertedIndex:     nil,
		sortedIndexes:     nil,
		vectorIndexes:     nil,
		schema:            nil,
		snapshotThreshold: 0,
		maxWALSize:        0,
		checkpointMu:      sync.RWMutex{},
		closeOnce:         sync.Once{},
		openBatch:         atomic.Pointer[logBatch[K, V]]{},
		logBatchGate:      sync.Mutex{},
		walEnabled:        false,
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

// configureSearchSchema sets up the tag, inverted, sorted, and vector indexes on the
// adapter from the provided search schema.
//
// Takes adapter (*OtterAdapter[K, V]) which is the adapter to configure.
// Takes schema (*cache_dto.SearchSchema) which defines the index configuration.
func configureSearchSchema[K comparable, V any](adapter *OtterAdapter[K, V], schema *cache_dto.SearchSchema) {
	if schema.MaxTagsPerKey > 0 {
		adapter.tagIndex.maxTagsPerKey = schema.MaxTagsPerKey
	}

	adapter.schema = schema
	adapter.fieldExtractor = NewFieldExtractor[V](schema)
	adapter.invertedIndex = NewInvertedIndex[K]()
	if schema.MaxInvertedIndexTokens > 0 {
		adapter.invertedIndex.maxTokens = schema.MaxInvertedIndexTokens
	}
	if schema.TextAnalyser != nil {
		adapter.invertedIndex.SetAnalyseFunction(schema.TextAnalyser)
	}

	adapter.sortedIndexes = make(map[string]*SortedIndex[K])
	adapter.vectorIndexes = make(map[string]*VectorIndex[K])
	configureFieldIndexes(adapter, schema)
}

// configureFieldIndexes creates sorted and vector indexes for each field in the schema.
//
// Takes adapter (*OtterAdapter[K, V]) which receives the index instances.
// Takes schema (*cache_dto.SearchSchema) which defines the fields and their properties.
func configureFieldIndexes[K comparable, V any](adapter *OtterAdapter[K, V], schema *cache_dto.SearchSchema) {
	for _, field := range schema.Fields {
		if field.Sortable {
			adapter.sortedIndexes[field.Name] = NewSortedIndex[K]()
		}
		if field.Type == cache_dto.FieldTypeVector {
			metric := vectormaths.Cosine
			if field.DistanceMetric != "" {
				metric = vectormaths.Metric(field.DistanceMetric)
			}
			vi := NewVectorIndex[K](field.Dimension, metric)
			if schema.MaxVectors > 0 {
				vi.maxVectors = schema.MaxVectors
			}
			adapter.vectorIndexes[field.Name] = vi
		}
	}
}

// initialisePersistence initialises WAL and snapshot store if persistence is configured.
//
// Takes options (Options) which provides cache configuration including persistence
// settings.
// Takes adapter (*OtterAdapter) which is the cache adapter to configure.
//
// Returns error when persistence configuration is invalid, WAL creation fails, snapshot
// store creation fails, or recovery from persistence fails.
func initialisePersistence[K comparable, V any](ctx context.Context, options cache_dto.Options[K, V], adapter *OtterAdapter[K, V]) error {
	persistConfig, ok := options.ProviderSpecific.(PersistenceConfig[K, V])
	if !ok || !persistConfig.Enabled {
		return nil
	}

	if err := persistConfig.Validate(); err != nil {
		return fmt.Errorf("invalid persistence config: %w", err)
	}

	codec := driven_disk.NewBinaryCodec(persistConfig.KeyCodec, persistConfig.ValueCodec)

	walConfig := persistConfig.WALConfig.WithDefaults()

	if persistConfig.ReadOnly {
		return loadReadOnly(ctx, walConfig, codec, adapter)
	}

	err := openPersistence(ctx, walConfig, codec, adapter)
	if err != nil && persistConfig.QuarantineCorrupt {
		moved, quarantineErr := quarantinePersistence(walConfig)
		if quarantineErr != nil {
			return errors.Join(err, quarantineErr)
		}
		_, l := logger_domain.From(ctx, log)
		l.Error("Persisted cache state is unreadable; moved it aside and starting empty",
			logger_domain.String("dir", walConfig.Dir),
			logger_domain.Strings("moved", moved),
			logger_domain.Error(err))
		persistenceQuarantinedTotal.Add(ctx, 1)
		adapter.clearAllIndexes()
		adapter.client.InvalidateAll()
		err = openPersistence(ctx, walConfig, codec, adapter)
	}
	if err != nil {
		return err
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Persistence initialised",
		logger_domain.String("dir", persistConfig.WALConfig.Dir),
		logger_domain.Bool("compression", persistConfig.WALConfig.EnableCompression))

	return nil
}

// openPersistence opens the write-ahead log and snapshot under walConfig and recovers the
// cache from them.
//
// Takes walConfig (wal_domain.Config) which locates the directory.
// Takes codec (*driven_disk.BinaryCodec[K, V]) which encodes and decodes entries.
// Takes adapter (*OtterAdapter[K, V]) which receives the recovered entries and keeps the
// log and snapshot.
//
// Returns error when the log or snapshot cannot be opened or recovered; nothing stays
// open then.
func openPersistence[K comparable, V any](
	ctx context.Context,
	walConfig wal_domain.Config,
	codec *driven_disk.BinaryCodec[K, V],
	adapter *OtterAdapter[K, V],
) error {
	wal, err := driven_disk.NewDiskWAL(ctx, walConfig, codec)
	if err != nil {
		return fmt.Errorf("creating WAL: %w", err)
	}

	snapshot, err := driven_disk.NewDiskSnapshot(ctx, walConfig, codec)
	if err != nil {
		_ = wal.Close()
		return fmt.Errorf("creating snapshot store: %w", err)
	}

	adapter.wal = wal
	adapter.snapshot = snapshot
	adapter.walEnabled = true
	adapter.snapshotThreshold = walConfig.SnapshotThreshold
	adapter.maxWALSize = walConfig.MaxWALSize

	if err := recoverFromPersistence(adapter); err != nil {
		_ = wal.Close()
		_ = snapshot.Close()
		adapter.wal = nil
		adapter.snapshot = nil
		adapter.walEnabled = false
		return fmt.Errorf("recovering from persistence: %w", err)
	}
	return nil
}

// quarantinePersistence renames a cache's snapshot and write-ahead log aside, keeping
// them for inspection, so the cache can start afresh in the same directory.
//
// Takes walConfig (wal_domain.Config) which locates the files.
//
// Returns []string which names the files moved aside.
// Returns error when a file exists but cannot be moved.
func quarantinePersistence(walConfig wal_domain.Config) ([]string, error) {
	suffix := ".corrupt-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	var moved []string
	for _, name := range []string{walConfig.SnapshotFileName, walConfig.WALFileName} {
		path := filepath.Join(walConfig.Dir, name)
		if err := os.Rename(path, path+suffix); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return moved, fmt.Errorf("moving unreadable %s aside: %w", path, err)
		}
		moved = append(moved, path+suffix)
	}
	return moved, nil
}

// loadReadOnly loads a snapshot and complete log entries into an in-memory cache without
// modifying persistence files.
//
// Takes walConfig (wal_domain.Config) which locates the directory.
// Takes codec (*driven_disk.BinaryCodec[K, V]) which decodes its entries.
// Takes adapter (*OtterAdapter[K, V]) which receives the entries.
//
// Returns error when the snapshot or log cannot be read.
func loadReadOnly[K comparable, V any](
	ctx context.Context,
	walConfig wal_domain.Config,
	codec *driven_disk.BinaryCodec[K, V],
	adapter *OtterAdapter[K, V],
) error {
	nowNano := time.Now().UnixNano()
	var (
		state                       map[K]entryData[V]
		snapshotEntries, walEntries int
		err                         error
	)
	for range readOnlyLoadAttempts {
		before := snapshotIdentity(walConfig)
		state, snapshotEntries, walEntries, err = readPersistedState(ctx, walConfig, codec, nowNano)
		if err != nil {
			return err
		}
		if snapshotIdentity(walConfig) == before {
			break
		}

	}
	populateCacheFromState(adapter, state, nowNano)

	_, l := logger_domain.From(ctx, log)
	l.Internal("Cache loaded read-only from persistence",
		logger_domain.String("dir", walConfig.Dir),
		logger_domain.Int("snapshot_entries", snapshotEntries),
		logger_domain.Int("wal_entries", walEntries),
		logger_domain.Int("final_keys", len(state)))
	return nil
}

// readPersistedState reads the snapshot and then the write-ahead log into one state map.
//
// Takes walConfig (wal_domain.Config) which locates the directory.
// Takes codec (*driven_disk.BinaryCodec[K, V]) which decodes its entries.
// Takes nowNano (int64) which drops entries already expired.
//
// Returns state (map[K]entryData[V]) which is the merged state.
// Returns snapshotEntries (int) which is the number of snapshot entries read.
// Returns walEntries (int) which is the number of log entries read.
// Returns error when the snapshot or log cannot be read.
func readPersistedState[K comparable, V any](
	ctx context.Context,
	walConfig wal_domain.Config,
	codec *driven_disk.BinaryCodec[K, V],
	nowNano int64,
) (state map[K]entryData[V], snapshotEntries int, walEntries int, err error) {
	state = make(map[K]entryData[V])
	snapshot, err := driven_disk.NewDiskSnapshot(ctx, walConfig, codec)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("opening snapshot read-only: %w", err)
	}
	snapshotEntries, err = loadSnapshotEntries(ctx, snapshot, state, nowNano)
	_ = snapshot.Close()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("reading snapshot: %w", err)
	}

	for entry, err := range driven_disk.ReadEntries(ctx, walConfig, codec) {
		if err != nil {
			return nil, 0, 0, fmt.Errorf("reading write-ahead log: %w", err)
		}
		applyEntryToState(state, entry, nowNano)
		walEntries++
	}
	return state, snapshotEntries, walEntries, nil
}

// snapshotFileIdentity identifies one version of a snapshot file.
type snapshotFileIdentity struct {
	// modified is the file's modification time in nanoseconds.
	modified int64

	// size is the file's length in bytes.
	size int64
}

// snapshotIdentity returns the snapshot file's current identity, or the zero identity
// when it does not exist. The owner replaces the file whole on every checkpoint, so a
// change in either field means a checkpoint ran.
//
// Takes walConfig (wal_domain.Config) which locates the snapshot.
//
// Returns snapshotFileIdentity which identifies the file's current version.
func snapshotIdentity(walConfig wal_domain.Config) snapshotFileIdentity {
	info, err := os.Stat(filepath.Join(walConfig.Dir, walConfig.SnapshotFileName))
	if err != nil {
		return snapshotFileIdentity{}
	}
	return snapshotFileIdentity{modified: info.ModTime().UnixNano(), size: info.Size()}
}

// recoverFromPersistence loads data from snapshot and WAL using streaming APIs, replaying
// operations into the cache without loading all entries into memory.
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
// Takes nowNano (int64) which is the current time in nanoseconds for expiry calculations.
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
// Takes nowNano (int64) which is the current time in nanoseconds for expiry checks.
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

	case wal_domain.OpBatch:

	}
}

// buildOtterOptions constructs otter.Options from cache_dto.Options, wiring up the tag
// index cleanup handlers and converting interface types.
//
// Takes options (cache_dto.Options[K, V]) which provides the cache configuration to
// convert.
// Takes adapter (*OtterAdapter[K, V]) which provides access to the tag and search indexes
// for deletion handlers.
//
// Returns *otter.Options[K, V] which contains the converted otter configuration.
func buildOtterOptions[K comparable, V any](options cache_dto.Options[K, V], adapter *OtterAdapter[K, V]) *otter.Options[K, V] {
	otterOpts := &otter.Options[K, V]{
		MaximumSize:       options.MaximumEntries,
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

	evicts := options.MaximumEntries > 0 || options.MaximumWeight > 0 || options.ExpiryCalculator != nil
	if evicts || options.OnDeletion != nil {
		otterOpts.OnDeletion = buildDeletionHandler(adapter, wrapOnDeletion(options.OnDeletion))
	}
	otterOpts.OnAtomicDeletion = buildDeletionHandler(adapter, wrapOnAtomicDeletion(options.OnAtomicDeletion))

	return otterOpts
}

// buildDeletionHandler creates a handler that updates indexes and invokes the optional
// user handler.
//
// It holds the adapter weakly so the cache cleanup can run once the adapter becomes
// unreachable.
//
// Takes adapter (*OtterAdapter[K, V]) which provides access to the indexes.
// Takes userHandler (func(otter.DeletionEvent[K, V])) which is an optional callback to
// run after cleanup.
//
// Returns func(otter.DeletionEvent[K, V]) which handles deletion events by removing the
// key from all indexes and calling the user handler if provided.
func buildDeletionHandler[K comparable, V any](adapter *OtterAdapter[K, V], userHandler func(otter.DeletionEvent[K, V])) func(otter.DeletionEvent[K, V]) {
	ref := weak.Make(adapter)
	return func(e otter.DeletionEvent[K, V]) {
		if e.Cause != otter.CauseReplacement {
			if live := ref.Value(); live != nil {
				live.tagIndex.RemoveKey(e.Key)
				live.removeFromSearchIndex(e.Key)
			}
		}
		if userHandler != nil {
			userHandler(e)
		}
	}
}
