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
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
	"weak"

	"github.com/maypok86/otter/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/safedisk"
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

	// ExpiresAt is the Unix timestamp in nanoseconds when this entry expires. Zero means the
	// cache's policy decides on restore, and math.MaxInt64 means it never expires.
	ExpiresAt int64
}

var (
	// errInvalidVectorField is returned when a vector field in the search schema has a
	// negative dimension or a distance metric the vector index does not support.
	errInvalidVectorField = errors.New("invalid vector field in search schema")
)

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
	if err := validateVectorFields(options.SearchSchema); err != nil {
		return nil, err
	}

	cache_domain.WarnUnbounded(context.Background(), options)

	adapter := &OtterAdapter[K, V]{
		client:            nil,
		tagIndex:          newTagIndex[K](),
		expiryOverrides:   &expiryOverrides[K]{},
		clock:             options.Clock,
		weigher:           options.Weigher,
		onDeletion:        options.OnDeletion,
		closeErr:          nil,
		namespace:         options.Namespace,
		maxEntryWeight:    options.MaxEntryWeight,
		weightBounded:     options.Weigher != nil && options.MaximumWeight > 0,
		wal:               nil,
		snapshot:          nil,
		fieldExtractor:    nil,
		invertedIndex:     nil,
		sortedIndexes:     nil,
		vectorIndexes:     nil,
		sortedFields:      nil,
		vectorFields:      nil,
		schema:            nil,
		snapshotThreshold: 0,
		maxWALSize:        0,
		checkpointMu:      sync.RWMutex{},
		closeOnce:         sync.Once{},
		logBatchGate:      sync.RWMutex{},
		ttlWarningOnce:    sync.Once{},
		walEnabled:        false,
		expiryEnabled:     options.ExpiryCalculator != nil,
		perEntry:          cache_domain.IsPerEntryExpiry(options.ExpiryCalculator),
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

// validateVectorFields checks the vector fields of schema before any index is built.
//
// An unknown distance metric would otherwise fall back to cosine without a word, and a
// negative dimension would build an index that rejects every vector.
//
// Takes schema (*cache_dto.SearchSchema) which is the schema to check; nil has no vector
// fields.
//
// Returns error which wraps errInvalidVectorField for the first invalid field.
func validateVectorFields(schema *cache_dto.SearchSchema) error {
	if schema == nil {
		return nil
	}
	for _, field := range schema.Fields {
		if field.Type != cache_dto.FieldTypeVector {
			continue
		}
		if field.Dimension < 0 {
			return fmt.Errorf("%w: field %q has negative dimension %d", errInvalidVectorField, field.Name, field.Dimension)
		}
		if !isSupportedDistanceMetric(field.DistanceMetric) {
			return fmt.Errorf("%w: field %q has unsupported distance metric %q", errInvalidVectorField, field.Name, field.DistanceMetric)
		}
	}
	return nil
}

// isSupportedDistanceMetric reports whether the vector index supports metric. An empty
// metric selects cosine.
//
// Takes metric (string) which is the distance metric named in the schema.
//
// Returns bool which is true for an empty metric, cosine, euclidean and dot product.
func isSupportedDistanceMetric(metric string) bool {
	switch vectormaths.Metric(metric) {
	case "", vectormaths.Cosine, vectormaths.Euclidean, vectormaths.DotProduct:
		return true
	default:
		return false
	}
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

	for _, name := range slices.Sorted(maps.Keys(adapter.sortedIndexes)) {
		adapter.sortedFields = append(adapter.sortedFields, sortedField[K]{index: adapter.sortedIndexes[name], name: name})
	}
	for _, name := range slices.Sorted(maps.Keys(adapter.vectorIndexes)) {
		adapter.vectorFields = append(adapter.vectorFields, vectorField[K]{index: adapter.vectorIndexes[name], name: name})
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
	ctx, l := logger_domain.From(ctx, log)

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

	if err := recoverFromPersistence(ctx, adapter); err != nil {
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
// Returns error when the directory cannot be opened, or a file exists but cannot be
// moved.
func quarantinePersistence(walConfig wal_domain.Config) ([]string, error) {
	sandbox, err := safedisk.NewSandbox(walConfig.Dir, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("opening persistence directory %s: %w", walConfig.Dir, err)
	}
	defer func() { _ = sandbox.Close() }()

	suffix := ".corrupt-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	var moved []string
	for _, name := range []string{walConfig.SnapshotFileName, walConfig.WALFileName} {
		path := filepath.Join(walConfig.Dir, name)
		if err := sandbox.Rename(name, name+suffix); err != nil {
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
	nowNano := adapter.nowNano()
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
func recoverFromPersistence[K comparable, V any](ctx context.Context, adapter *OtterAdapter[K, V]) error {
	ctx, l := logger_domain.From(ctx, log)

	nowNano := adapter.nowNano()
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
		if !restoreRecoveredEntry(adapter, key, data, nowNano) {
			continue
		}
		adapter.tagIndex.Add(key, data.Tags)
		adapter.indexDocument(key, data.Value)
	}
}

// restoreRecoveredEntry writes one recovered entry into the cache with the expiry it had.
//
// The remaining TTL is not capped here: otter adds it to a later clock reading than
// nowNano, so overrideDuration caps it against that reading instead.
//
// Takes adapter (*OtterAdapter[K, V]) which is the cache to write to.
// Takes key (K) which identifies the entry.
// Takes data (entryData[V]) which is the recovered entry.
// Takes nowNano (int64) which is the current time in nanoseconds.
//
// Returns bool which is false when the entry has expired since it was logged and was not
// restored.
func restoreRecoveredEntry[K comparable, V any](adapter *OtterAdapter[K, V], key K, data entryData[V], nowNano int64) bool {
	switch data.ExpiresAt {
	case 0:
		adapter.client.Set(key, data.Value)
		return true
	case math.MaxInt64:
		adapter.writeWithExpiry(key, data.Value, cache_dto.NoExpiry)
		return true
	}
	remaining := time.Duration(data.ExpiresAt - nowNano)
	if remaining <= 0 {
		return false
	}
	adapter.writeWithExpiry(key, data.Value, remaining)
	return true
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
		ExpiryCalculator:  buildExpiryCalculator(options.ExpiryCalculator, adapter.expiryOverrides),
		RefreshCalculator: wrapRefreshCalculator(options.RefreshCalculator),
		StatsRecorder:     wrapStatsRecorder(options.StatsRecorder),
		Clock:             wrapClock(options.Clock),
		Logger:            wrapLogger(options.Logger),
	}

	if options.OnDeletion != nil {
		otterOpts.OnDeletion = wrapOnDeletion(options.OnDeletion)
	}
	otterOpts.OnAtomicDeletion = buildDeletionHandler(adapter, wrapOnAtomicDeletion(options.OnAtomicDeletion))

	return otterOpts
}

// buildDeletionHandler creates a handler that updates indexes and invokes the optional
// user handler.
//
// Otter runs it inside the lock that deletes the entry, for every cause, so the indexes
// change in the same step as the cache and a later write of the key is never undone by a
// late handler.
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

// buildExpiryCalculator wraps the configured expiry calculator for otter so that the TTL
// of a write takes precedence over it.
//
// Takes calculator (cache_dto.ExpiryCalculator[K, V]) which is the configured calculator;
// nil means the cache does not expire entries.
// Takes overrides (*expiryOverrides[K]) which holds the TTLs of writes in progress.
//
// Returns otter.ExpiryCalculator[K, V] which is the calculator otter uses, or nil when
// the cache does not expire entries.
func buildExpiryCalculator[K comparable, V any](
	calculator cache_dto.ExpiryCalculator[K, V],
	overrides *expiryOverrides[K],
) otter.ExpiryCalculator[K, V] {
	if calculator == nil {
		return nil
	}
	return newOverridingExpiryCalculator(wrapExpiryCalculator(calculator), overrides)
}
