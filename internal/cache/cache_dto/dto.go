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

package cache_dto

import (
	"context"
	"errors"
	"time"
)

const (
	// CauseInvalidation means the entry was removed by the user.
	CauseInvalidation DeletionCause = 1 + iota

	// CauseReplacement means the entry's value was replaced by the user.
	CauseReplacement

	// CauseOverflow means the entry was evicted due to size constraints.
	CauseOverflow

	// CauseExpiration means the entry's expiration timestamp has passed.
	CauseExpiration
)

const (
	// ComputeActionSet means the computed value should be stored in the cache.
	ComputeActionSet ComputeAction = iota

	// ComputeActionDelete indicates the entry should be removed from the cache.
	ComputeActionDelete

	// ComputeActionNoop indicates no action should be taken (value unchanged).
	ComputeActionNoop
)

const (
	// TransformerCompression is a transformer type that compresses cached values.
	TransformerCompression TransformerType = "compression"

	// TransformerEncryption marks a transformer that encrypts cached values.
	TransformerEncryption TransformerType = "encryption"

	// TransformerCustom is a transformer type for user-defined transformers.
	TransformerCustom TransformerType = "custom"
)

// DeletionCause represents the reason why an entry was removed from the cache.
type DeletionCause int

// DeletionEvent holds the details passed to deletion handlers when an entry
// is removed from the cache.
type DeletionEvent[K comparable, V any] struct {
	// Key is the identifier for the deleted item.
	Key K

	// Value is the data linked to this deletion event.
	Value V

	// Cause specifies why the deletion event happened.
	Cause DeletionCause
}

// WasEvicted reports whether the deletion was automatic rather than manual.
//
// Returns bool which is true when the cause is not invalidation or replacement.
func (de DeletionEvent[K, V]) WasEvicted() bool {
	return de.Cause != CauseInvalidation && de.Cause != CauseReplacement
}

// ComputeAction represents the action to be taken after a compute function
// executes.
type ComputeAction int

// ComputeResult holds the result of a compute operation with optional TTL
// override. A zero TTL means use the cache's default expiration policy.
type ComputeResult[V any] struct {
	// Value is the computed value to store (when Action is ComputeActionSet).
	Value V

	// Action specifies whether to set, delete, or do nothing.
	Action ComputeAction

	// TTL specifies a custom time-to-live for this entry; zero means use default.
	// Only meaningful when Action is ComputeActionSet.
	TTL time.Duration
}

// ErrNotFound is returned by a Loader when a value is missing from the data
// source.
var ErrNotFound = errors.New("key not found")

// Entry is a fixed snapshot of a key-value pair in the cache.
type Entry[K comparable, V any] struct {
	// Key is the map key for this entry.
	Key K

	// Value holds the data stored in this entry.
	Value V

	// Weight indicates how important this entry is for sorting or selection.
	Weight uint32

	// ExpiresAtNano is the Unix timestamp in nanoseconds when this entry expires.
	ExpiresAtNano int64

	// RefreshableAtNano is the Unix timestamp in nanoseconds when this entry
	// becomes eligible for refresh.
	RefreshableAtNano int64

	// SnapshotAtNano is the Unix timestamp in nanoseconds when the snapshot was
	// taken.
	SnapshotAtNano int64
}

// Loader computes or fetches values for use in filling the cache.
type Loader[K comparable, V any] interface {
	// Load retrieves the value associated with the given key.
	//
	// Takes key (K) which identifies the value to retrieve.
	//
	// Returns V which is the value associated with the key.
	// Returns error when the key is not found or retrieval fails.
	Load(ctx context.Context, key K) (V, error)

	// Reload refreshes the value for the given key.
	//
	// Takes key (K) which identifies the entry to reload.
	// Takes oldValue (V) which is the current cached value.
	//
	// Returns V which is the newly loaded value.
	// Returns error when the reload operation fails.
	Reload(ctx context.Context, key K, oldValue V) (V, error)
}

// LoaderFunc is an adapter that allows ordinary functions to be used as
// loaders.
type LoaderFunc[K comparable, V any] func(ctx context.Context, key K) (V, error)

// Load implements the Loader interface by calling the underlying function.
//
// Takes key (K) which specifies the key to load.
//
// Returns V which is the loaded value.
// Returns error when the underlying function fails.
func (lf LoaderFunc[K, V]) Load(ctx context.Context, key K) (V, error) {
	return lf(ctx, key)
}

// Reload implements the Loader interface by calling the underlying function,
// ignoring the old value.
//
// Takes key (K) which identifies the entry to reload.
//
// Returns V which is the reloaded value.
// Returns error when the underlying function fails.
func (lf LoaderFunc[K, V]) Reload(ctx context.Context, key K, _ V) (V, error) {
	return lf(ctx, key)
}

// BulkLoader gets or computes values for many keys at once.
type BulkLoader[K comparable, V any] interface {
	// BulkLoad retrieves multiple values by their keys in a single operation.
	//
	// Takes keys ([]K) which contains the keys to look up.
	//
	// Returns map[K]V which contains the found key-value pairs.
	// Returns error when the bulk load operation fails.
	BulkLoad(ctx context.Context, keys []K) (map[K]V, error)

	// BulkReload reloads multiple cache entries in a single batch operation.
	//
	// Takes keys ([]K) which contains the cache keys to reload.
	// Takes oldValues ([]V) which contains the previous values for those keys.
	//
	// Returns map[K]V which contains the reloaded values keyed by their cache key.
	// Returns error when the bulk reload operation fails.
	BulkReload(ctx context.Context, keys []K, oldValues []V) (map[K]V, error)
}

// BulkLoaderFunc is an adapter that allows ordinary functions to be used as
// bulk loaders.
type BulkLoaderFunc[K comparable, V any] func(ctx context.Context, keys []K) (map[K]V, error)

// BulkLoad implements the BulkLoader interface by calling the underlying
// function.
//
// Takes keys ([]K) which specifies the keys to load values for.
//
// Returns map[K]V which contains the loaded values keyed by their input keys.
// Returns error when the underlying function fails.
func (blf BulkLoaderFunc[K, V]) BulkLoad(ctx context.Context, keys []K) (map[K]V, error) {
	return blf(ctx, keys)
}

// BulkReload implements the BulkLoader interface by calling the underlying
// function, ignoring old values.
//
// Takes keys ([]K) which contains the cache keys to reload.
//
// Returns map[K]V which contains the reloaded values keyed by their cache
// key.
// Returns error when the underlying function fails.
func (blf BulkLoaderFunc[K, V]) BulkReload(ctx context.Context, keys []K, _ []V) (map[K]V, error) {
	return blf(ctx, keys)
}

// LoadResult holds the outcome of an asynchronous load or refresh operation.
// It is sent over a channel to signal when the operation is complete.
type LoadResult[V any] struct {
	// Value holds the data that was loaded successfully.
	Value V

	// Err holds any error from loading; nil means success.
	Err error
}

// ExpiryCalculator sets when cache entries expire after being created, read,
// or updated.
type ExpiryCalculator[K comparable, V any] interface {
	// ExpireAfterCreate returns the duration after which a newly created entry
	// should expire.
	//
	// Takes entry (Entry[K, V]) which is the cache entry being created.
	//
	// Returns time.Duration which is the time until the entry expires.
	ExpireAfterCreate(entry Entry[K, V]) time.Duration

	// ExpireAfterUpdate returns the duration after which an entry should expire
	// following an update.
	//
	// Takes entry (Entry[K, V]) which is the cache entry that was updated.
	// Takes oldValue (V) which is the previous value before the update.
	//
	// Returns time.Duration which specifies how long until the entry expires.
	ExpireAfterUpdate(entry Entry[K, V], oldValue V) time.Duration

	// ExpireAfterRead returns the duration after which an entry should expire
	// following a read operation.
	//
	// Takes entry (Entry[K, V]) which is the cache entry that was read.
	//
	// Returns time.Duration which specifies how long until the entry expires.
	ExpireAfterRead(entry Entry[K, V]) time.Duration
}

// RefreshCalculator calculates when cache entries should be asynchronously
// refreshed.
type RefreshCalculator[K comparable, V any] interface {
	// RefreshAfterCreate returns the duration after which a newly created entry
	// should be refreshed.
	//
	// Takes entry (Entry[K, V]) which is the cache entry that was just created.
	//
	// Returns time.Duration which specifies how long to wait before refreshing.
	RefreshAfterCreate(entry Entry[K, V]) time.Duration

	// RefreshAfterUpdate calculates the refresh duration after a value update.
	//
	// Takes entry (Entry[K, V]) which is the cache entry being updated.
	// Takes oldValue (V) which is the previous value before the update.
	//
	// Returns time.Duration which specifies how long until the entry should be
	// refreshed.
	RefreshAfterUpdate(entry Entry[K, V], oldValue V) time.Duration

	// RefreshAfterReload determines the refresh interval after reloading an entry.
	//
	// Takes entry (Entry[K, V]) which is the cache entry being refreshed.
	// Takes oldValue (V) which is the previous value before reload.
	//
	// Returns time.Duration which is the time to wait before the next refresh.
	RefreshAfterReload(entry Entry[K, V], oldValue V) time.Duration

	// RefreshAfterReloadFailure returns the duration to wait before retrying
	// after a failed reload.
	//
	// Takes entry (Entry[K, V]) which is the cache entry that failed to reload.
	// Takes err (error) which is the error that caused the reload to fail.
	//
	// Returns time.Duration which is the delay before the next reload attempt.
	RefreshAfterReloadFailure(entry Entry[K, V], err error) time.Duration
}

// Clock provides a way to get the current time. Inject a mock
// implementation to test time-dependent code deterministically.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// Logger defines a logging interface for cache operations.
// Implementations can use any logging framework.
type Logger interface {
	// Error logs an error message with optional key-value pairs.
	//
	// Takes message (string) which is the error message to log.
	// Takes keysAndValues (...any) which are optional pairs for structured data.
	Error(message string, keysAndValues ...any)

	// Warn logs a warning message with optional key-value pairs.
	//
	// Takes message (string) which is the warning message to log.
	// Takes keysAndValues (...any) which provides extra context as key-value
	// pairs.
	Warn(message string, keysAndValues ...any)

	// Info logs a message at the informational level.
	//
	// Takes message (string) which is the message to log.
	// Takes keysAndValues (...any) which are optional key-value pairs to include.
	Info(message string, keysAndValues ...any)

	// Debug logs a message at debug level with optional key-value pairs.
	//
	// Takes message (string) which is the message to log.
	// Takes keysAndValues (...any) which are alternating keys and values.
	Debug(message string, keysAndValues ...any)
}

// TransformerType identifies the kind of cache value transformer.
type TransformerType string

// TransformConfig configures value transformations for cache operations.
// Transformers run in priority order on Set and are automatically reversed
// on Get, with metadata embedded to enable reversal even if config changes.
type TransformConfig struct {
	// TransformerOptions maps transformer names to their settings.
	TransformerOptions map[string]any

	// EnabledTransformers is the ordered list of transformer names to apply.
	// The actual execution order is determined by each transformer's priority.
	EnabledTransformers []string
}

// Options holds the configuration for creating a new cache instance.
type Options[K comparable, V any] struct {
	// ExpiryCalculator calculates when items expire; nil uses default expiry.
	ExpiryCalculator ExpiryCalculator[K, V]

	// Logger handles structured logging output; nil uses the default logger.
	Logger Logger

	// ProviderSpecific holds settings for a particular provider. The provider
	// factory must type-assert this to its own Config type, for example
	// provider_redis.Config{Address: "localhost:6379"}.
	ProviderSpecific any

	// Clock provides time functions; if nil, the real system clock is used.
	Clock Clock

	// RefreshCalculator computes new values when cache entries need updating.
	RefreshCalculator RefreshCalculator[K, V]

	// StatsRecorder records statistics about check runs.
	StatsRecorder StatsRecorder

	// Executor runs functions; if nil, functions run in the current goroutine.
	Executor func(operation func())

	// Weigher calculates the weight of a cache entry; nil uses a default of 1.
	Weigher func(key K, value V) uint32

	// OnDeletion is called when an entry is removed from the cache.
	OnDeletion func(e DeletionEvent[K, V])

	// OnAtomicDeletion is a callback called when an entry is removed from the
	// cache.
	OnAtomicDeletion func(e DeletionEvent[K, V])

	// TransformConfig specifies how to change cached values, such as compression
	// or encryption. If nil, values are stored without changes.
	TransformConfig *TransformConfig

	// SearchSchema defines which fields are searchable for query operations.
	// If nil, Search() and Query() return ErrSearchNotSupported.
	SearchSchema *SearchSchema

	// Provider specifies which provider to use (e.g., "otter", "redis").
	// If empty, the service's default provider is used.
	Provider string

	// Namespace is the logical identifier for this cache instance, used as a key
	// prefix for Redis or for metrics in Otter; defaults to "default" if empty.
	Namespace string

	// InitialCapacity is the starting size for internal buffers; 0 uses a default.
	InitialCapacity int

	// MaximumWeight is the maximum total weight allowed; 0 means no limit.
	MaximumWeight uint64

	// MaximumSize is the largest allowed size in bytes; 0 means no limit.
	MaximumSize int
}

// DefaultTransformConfig returns a TransformConfig with no transformations
// enabled.
//
// Returns TransformConfig which is ready for use with empty transformer lists.
func DefaultTransformConfig() TransformConfig {
	return TransformConfig{
		EnabledTransformers: []string{},
		TransformerOptions:  make(map[string]any),
	}
}
