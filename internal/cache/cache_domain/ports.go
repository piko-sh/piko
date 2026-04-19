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
	"iter"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
)

// Cache is the interface for interacting with a cache instance.
// Its API mirrors maypok86/otter/v2's Cache API.
type Cache[K comparable, V any] interface {
	// GetIfPresent returns the value for a key if it is present
	// and not expired.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the cached item to retrieve.
	//
	// Returns V which is the cached value.
	// Returns bool which is true if the key was found and not expired.
	// Returns error when the operation fails (e.g. network error
	// for distributed caches).
	GetIfPresent(ctx context.Context, key K) (V, bool, error)

	// Get returns the value for a key, calling the loader if the key is missing.
	// Provides built-in protection against cache stampede (thundering
	// herd).
	//
	// Takes key (K) which identifies the cached value.
	// Takes loader (Loader[K, V]) which fetches the value if not in cache.
	//
	// Returns V which is the cached or newly loaded value.
	// Returns error when the loader fails.
	Get(ctx context.Context, key K, loader cache_dto.Loader[K, V]) (V, error)

	// Set stores a value with the given key, replacing any existing entry.
	// It accepts optional tags for group-based invalidation.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the value.
	// Takes value (V) which is the data to store.
	// Takes tags (...string) which are optional group names for invalidation.
	//
	// Returns error when the operation fails.
	Set(ctx context.Context, key K, value V, tags ...string) error

	// SetWithTTL stores a value with a custom expiry time for this entry.
	//
	// This overrides the cache's default expiration policy. The operation is
	// atomic and performs better than calling Set followed by SetExpiresAfter
	// for remote caches.
	//
	// Takes key (K) which identifies the cached entry.
	// Takes value (V) which is the data to store.
	// Takes ttl (time.Duration) which sets when this entry expires.
	// Takes tags (...string) which enable group-based invalidation.
	//
	// Returns error when the cache operation fails.
	SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error

	// Invalidate removes an entry from the cache.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the entry to remove.
	//
	// Returns error when the operation fails.
	Invalidate(ctx context.Context, key K) error

	// Computes a new value atomically for the given key based on the
	// current value. The compute function executes while holding a lock on the key.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the cache entry to update.
	// Takes computeFunction (func) which receives the current value and presence flag,
	// returning the new value and an action (Set, Delete, or Noop).
	//
	// Returns V which is the resulting value after the operation.
	// Returns bool which indicates whether a value is now present.
	// Returns error when the operation fails.
	Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error)

	// ComputeIfAbsent atomically computes and stores a value if the key is absent.
	// The compute function is only called if the key is not present.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the entry to look up or create.
	// Takes computeFunction (func() V) which generates the value if the key is absent.
	//
	// Returns V which is the existing value if present, or the newly computed value.
	// Returns bool which indicates whether computation occurred.
	// Returns error when the operation fails.
	ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error)

	// ComputeIfPresent atomically computes a new value for the key if it is
	// present. The compute function is only called if the key exists.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the entry to update.
	// Takes computeFunction (func(oldValue V) (newValue V,
	// action ComputeAction)) which computes the new value from
	// the existing value.
	//
	// Returns V which is the new value if the key was present.
	// Returns bool which indicates whether the key was present.
	// Returns error when the operation fails.
	ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error)

	// ComputeWithTTL atomically computes a new value for the given key with
	// per-call TTL control. The compute function executes while holding a lock
	// on the key and can specify a custom TTL for the entry.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the cache entry to update.
	// Takes computeFunction (func) which receives the current value and presence flag,
	// returning a ComputeResult containing the new value, action, and optional TTL.
	//
	// Returns V which is the resulting value after the operation.
	// Returns bool which indicates whether a value is now present.
	// Returns error when the operation fails.
	ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error)

	// BulkGet retrieves multiple keys, using the bulk loader for any misses.
	//
	// Takes keys ([]K) which specifies the cache keys to retrieve.
	// Takes bulkLoader (BulkLoader[K, V]) which loads values for missing keys.
	//
	// Returns map[K]V which contains all requested key-value pairs.
	// Returns error when retrieval or bulk loading fails.
	BulkGet(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) (map[K]V, error)

	// BulkSet stores multiple key-value pairs in a single operation.
	//
	// For remote caches (e.g., Redis), this uses a pipeline or MSET for optimal
	// performance. For in-memory caches, this typically loops over items. All
	// items share the same tags.
	//
	// Takes items (map[K]V) which contains the key-value pairs to store.
	// Takes tags (...string) which are optional tags to associate with all items.
	//
	// Returns error when the bulk storage operation fails.
	BulkSet(ctx context.Context, items map[K]V, tags ...string) error

	// InvalidateByTags removes all entries associated with any of the given tags.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes tags (...string) which specifies the tags whose entries should be
	// removed.
	//
	// Returns int which is the number of keys that were invalidated. The
	// operation's atomicity depends on the underlying provider.
	// Returns error when the operation fails.
	InvalidateByTags(ctx context.Context, tags ...string) (int, error)

	// InvalidateAll removes all entries from the cache.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	//
	// Returns error when the operation fails.
	InvalidateAll(ctx context.Context) error

	// BulkRefresh reloads values for multiple keys using a bulk loader.
	//
	// The old values are served until the refresh completes. This is a
	// fire-and-forget operation.
	//
	// Takes keys ([]K) which specifies the cache keys to refresh.
	// Takes bulkLoader (BulkLoader[K, V]) which fetches the new values in bulk.
	BulkRefresh(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V])

	// Refresh asynchronously reloads the value for a key.
	//
	// The old value is served until the refresh completes.
	//
	// Takes key (K) which identifies the cache entry to refresh.
	// Takes loader (cache_dto.Loader[K, V]) which fetches the new value.
	//
	// Returns <-chan cache_dto.LoadResult[V] which receives the result upon
	// completion.
	Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V]

	// All returns an iterator over all key-value pairs in the cache.
	// The snapshot may not include changes made while iterating.
	//
	// Returns iter.Seq2[K, V] which yields each key and its value.
	All() iter.Seq2[K, V]

	// Keys returns an iterator over all keys in the cache.
	//
	// Returns iter.Seq[K] which yields each key stored in the cache.
	Keys() iter.Seq[K]

	// Values returns an iterator over all values in the cache.
	//
	// Returns iter.Seq[V] which yields each value stored in the cache.
	Values() iter.Seq[V]

	// GetEntry returns a snapshot of the entry, including metadata like weight
	// and expiration. This operation resets the entry's access timer for TTL
	// calculations.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the entry to retrieve.
	//
	// Returns Entry[K, V] which contains the entry data and metadata.
	// Returns bool which indicates whether the entry was found.
	// Returns error when the operation fails.
	GetEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error)

	// ProbeEntry returns a snapshot of the entry without changing its access
	// patterns. This does not reset the entry's access timer, making it useful for
	// monitoring.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the entry to probe.
	//
	// Returns cache_dto.Entry[K, V] which is the entry snapshot.
	// Returns bool which indicates whether the entry exists.
	// Returns error when the operation fails.
	ProbeEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error)

	// EstimatedSize returns the approximate number of entries in the cache.
	EstimatedSize() int

	// Stats returns a snapshot of the cache's performance statistics.
	Stats() cache_dto.Stats

	// Close releases any resources used by the cache, such as background
	// goroutines.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	//
	// Returns error when resources cannot be released cleanly.
	Close(ctx context.Context) error

	// GetMaximum returns the current maximum capacity (size or weight) of the cache.
	GetMaximum() uint64

	// SetMaximum changes the maximum capacity of the cache.
	// If the new capacity is smaller, this may cause items to be removed at once.
	//
	// Takes size (uint64) which is the new maximum number of items.
	SetMaximum(size uint64)

	// WeightedSize returns the total weight of all items in the cache. If the
	// cache does not use weights, this may return the same value as EstimatedSize.
	//
	// Returns uint64 which is the current total weight.
	WeightedSize() uint64

	// SetExpiresAfter sets or changes the expiry time for a key that already
	// exists.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the item to update.
	// Takes expiresAfter (time.Duration) which specifies how long until the key
	// expires.
	//
	// Returns error when the operation fails.
	SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error

	// SetRefreshableAfter manually sets or overrides the refresh time for an
	// existing key.
	//
	// When the context is already cancelled or has exceeded its deadline, returns
	// the context's error without performing any work.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes key (K) which identifies the cache entry to update.
	// Takes refreshableAfter (time.Duration) which specifies the new refresh time.
	//
	// Returns error when the operation fails.
	SetRefreshableAfter(ctx context.Context, key K, refreshableAfter time.Duration) error

	// Search performs full-text search across indexed TEXT fields.
	// This enables finding cached entries by content rather than just by key.
	//
	// The query string is tokenised and matched against TEXT fields defined
	// in the SearchSchema. Results are ranked by relevance score.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes query (string) which is the search query text.
	// Takes opts (*SearchOptions) which configures pagination, sorting, and filters.
	//
	// Returns SearchResult containing matched entries with metadata.
	// Returns ErrSearchNotSupported if the provider doesn't support search.
	Search(ctx context.Context, query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error)

	// Query performs structured filtering, sorting, and pagination without
	// full-text search. This is for finding entries by exact field values
	// or ranges.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes opts (*QueryOptions) which specifies filters, sorting, and pagination.
	//
	// Returns SearchResult containing matched entries.
	// Returns ErrSearchNotSupported if the provider doesn't support query.
	Query(ctx context.Context, opts *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error)

	// SupportsSearch returns true if this cache supports search and query
	// operations. Use this to check capability before calling Search or Query.
	//
	// Returns bool which is true when search operations are available.
	SupportsSearch() bool

	// GetSchema returns the search schema for this cache, or nil if search
	// is not configured. The schema describes which fields are searchable.
	//
	// Returns *SearchSchema which describes searchable fields, or nil.
	GetSchema() *cache_dto.SearchSchema
}

// ProviderPort defines the driven port for cache adapters in a hexagonal
// architecture. Each concrete implementation (Otter, Redis, etc.) must satisfy
// the contract defined here.
type ProviderPort[K comparable, V any] interface {
	Cache[K, V]
}

// Transactional is an optional interface for cache providers that support
// multi-operation rollback. Providers that implement this can participate
// in DAL-level transactions with proper rollback semantics.
//
// Providers that do not implement Transactional can still participate in
// transactions via the generic journal-based fallback provided by
// BeginTransaction.
type Transactional[K comparable, V any] interface {
	// BeginTransaction returns a transactional view of this cache.
	//
	// All mutations through the returned TransactionCache are
	// journalled. The caller MUST call either Commit or Rollback
	// on the returned cache.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	//
	// Returns TransactionCache[K, V] which wraps the cache with
	// undo journalling.
	BeginTransaction(ctx context.Context) TransactionCache[K, V]
}

// TransactionCache wraps a cache with undo journalling. Reads pass through
// to the underlying cache; writes are applied immediately but journalled so
// they can be undone on Rollback.
type TransactionCache[K comparable, V any] interface {
	ProviderPort[K, V]

	// Commit discards the undo journal, making all mutations permanent.
	//
	// Returns error when the commit operation fails.
	Commit(ctx context.Context) error

	// Rollback replays the undo journal in reverse, restoring every
	// mutated key to its pre-transaction state.
	//
	// Returns error when a rollback operation fails.
	Rollback(ctx context.Context) error
}

// Provider is the non-generic interface for cache providers that manage
// resources. It implements io.Closer and manages connections, pools, and other
// shared resources.
//
// Architecture:
//   - Provider = Resource manager (e.g., ONE Redis connection pool)
//   - Namespace = Type-specific cache instance (e.g., "users" Cache[string, User])
//
// This design allows:
//   - Resource sharing: ONE Redis client serves many typed caches
//   - No type conflicts: "redis" provider can serve [string, User], etc.
//   - Clear semantics: Provider manages resources, Namespace manages data
//
// CreateNamespaceTyped is a non-generic method that uses type erasure
// (any). Use the standalone CreateNamespace[K, V]() function for type-safe
// access.
type Provider interface {
	// CreateNamespaceTyped creates a namespace with type-erased options.
	// Internal method - use CreateNamespace[K, V]() instead for type safety.
	CreateNamespaceTyped(namespace string, options any) (any, error)

	// Close releases all resources managed by this provider, including
	// connections and goroutines.
	//
	// Returns error when resources cannot be released cleanly.
	Close() error

	// Name returns the provider's identifier (e.g., "redis", "otter",
	// "redis-cluster").
	Name() string
}

// Service manages cache providers and creates configured cache instances.
// It implements io.Closer and cache.Service.
//
// CreateNamespace is not part of Service because Go interfaces
// cannot have generic methods. Use the standalone NewCache function or
// CacheBuilder instead.
type Service interface {
	// RegisterProvider adds a new cache provider implementation to the service.
	//
	// Takes ctx (context.Context) which carries logging context.
	// Takes name (string) which identifies the provider in the registry.
	// Takes provider (any) which must be a Provider instance.
	RegisterProvider(ctx context.Context, name string, provider any) error

	// GetProvider retrieves a registered Provider by name.
	//
	// Takes name (string) which identifies the provider to retrieve.
	//
	// Returns Provider which is the registered provider.
	// Returns error when the provider is not found.
	GetProvider(name string) (Provider, error)

	// GetProviders returns a sorted list of all registered provider names.
	GetProviders() []string

	// SetDefaultProvider sets the default cache provider to use when none is
	// specified.
	//
	// Takes ctx (context.Context) which carries logging context.
	// Takes name (string) which is the provider name to set as default.
	//
	// Returns error when the provider name is not registered.
	SetDefaultProvider(ctx context.Context, name string) error

	// GetDefaultProvider returns the name of the current default provider.
	//
	// Returns string which is the default provider name.
	GetDefaultProvider() string

	// Close shuts down all registered providers and releases their resources.
	//
	// Takes ctx (context.Context) which carries logging context.
	//
	// Returns error when any provider fails to close.
	Close(ctx context.Context) error
}
