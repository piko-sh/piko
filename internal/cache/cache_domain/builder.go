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
	"reflect"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultL2MaxFailures is the number of failures allowed before the circuit
	// breaker opens for multi-level caches.
	defaultL2MaxFailures = 5

	// defaultL2OpenTimeoutSeconds is the default timeout in seconds for opening
	// the level 2 cache.
	defaultL2OpenTimeoutSeconds = 30

	// logKeyReason is the log attribute key for explaining why a feature is
	// limited or unavailable.
	logKeyReason = "reason"
)

var (
	transformerBlueprints = make(map[string]TransformerBlueprintFactory)

	transformerBlueprintsMutex sync.RWMutex

	multiLevelAdapterConstructor MultiLevelAdapterConstructor

	multiLevelAdapterMutex sync.RWMutex
)

// MultiLevelAdapterConstructor is a function type that creates a multi-level
// cache adapter from provider ports and configuration.
type MultiLevelAdapterConstructor func(
	ctx context.Context,
	name string,
	l1 any,
	l2 any,
	config any,
) (any, error)

// TransformerBlueprintFactory creates a transformer from its settings.
// The config parameter can be nil, a map[string]any, or a specific struct.
type TransformerBlueprintFactory func(config any) (CacheTransformerPort, error)

// CacheBuilder creates cache instances with a fluent interface. It simplifies
// setup by handling registries, transformers, and encoders.
//
// Usage:
// cache, err := cache_domain.NewCacheBuilder[string, User](service).
//
//	WithProvider("redis").
//	WithMaximumSize(10000).
//	WithCompression().
//	WithEncoder(gobEncoder).
//	Build(ctx)
type CacheBuilder[K comparable, V any] struct {
	// l1ProviderOptions holds settings for the L1 cache provider.
	l1ProviderOptions any

	// cacheLogger logs messages for cache operations.
	cacheLogger cache_dto.Logger

	// service provides cross-reference resolution and symbol lookup.
	service Service

	// l2ProviderOptions holds settings for the level 2 cache provider.
	l2ProviderOptions any

	// expiryCalculator determines when cache entries should expire.
	expiryCalculator cache_dto.ExpiryCalculator[K, V]

	// providerOptions holds settings specific to the chosen provider.
	providerOptions any

	// defaultEncoder is the encoder used when no custom encoder is set.
	defaultEncoder AnyEncoder

	// refreshCalculator decides when cache entries should be refreshed.
	refreshCalculator cache_dto.RefreshCalculator[K, V]

	// statsRecorder records cache hit and miss statistics.
	statsRecorder cache_dto.StatsRecorder

	// clock provides time functions for cache expiry.
	clock cache_dto.Clock

	// weigher calculates the weight of a cache entry; nil uses a default weight of
	// 1.
	weigher func(key K, value V) uint32

	// onAtomicDeletion is called when a single cache entry is removed.
	onAtomicDeletion func(e cache_dto.DeletionEvent[K, V])

	// onDeletion is the callback run when an entry is removed from the cache.
	onDeletion func(e cache_dto.DeletionEvent[K, V])

	// executor runs functions; nil means functions are called directly.
	executor func(operation func())

	// searchSchema defines which fields can be searched when running queries.
	searchSchema *cache_dto.SearchSchema

	// l1ProviderName is the name of the first-level cache provider.
	l1ProviderName string

	// namespace is the prefix for cache keys to prevent key conflicts.
	namespace string

	// factoryBlueprint is the template string used to create cache factory
	// instances.
	factoryBlueprint string

	// l2ProviderName is the name of the level-2 cache provider.
	l2ProviderName string

	// providerName is the name of the cache provider.
	providerName string

	// transformers holds the list of transformer rules to apply.
	transformers []transformerSpec

	// encoders holds the list of encoders used for cache entries.
	encoders []AnyEncoder

	// maximumWeight is the maximum total weight of entries in the cache.
	maximumWeight uint64

	// l2OpenTimeout is the maximum time to wait when opening the L2 cache.
	l2OpenTimeout time.Duration

	// maximumSize is the maximum number of entries in the cache; 0 means no limit.
	maximumSize int

	// initialCapacity is the starting size of the cache; 0 uses the default.
	initialCapacity int

	// l2MaxFailures is the number of failures before the L2 cache is disabled.
	l2MaxFailures int

	// isMultiLevel indicates whether the cache uses more than one storage layer.
	isMultiLevel bool
}

// transformerSpec holds the settings for a transformer to be registered.
type transformerSpec struct {
	// factory creates a transformer from the config; nil means use the default.
	factory func(config any) (CacheTransformerPort, error)

	// config holds settings for this transformer; nil means no options are set.
	config any

	// name identifies the transformer for registration and error messages.
	name string
}

// Provider specifies which cache provider to use (e.g., "otter", "redis",
// "mock"), defaulting to the service's default provider and mutually
// exclusive with MultiLevel.
//
// Takes name (string) which identifies the cache provider to use.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Provider(name string) *CacheBuilder[K, V] {
	b.providerName = name
	return b
}

// Namespace sets the namespace for this cache instance, which becomes the key
// prefix for connection-pooled providers like Redis (e.g., "users:") or is
// used for metrics and logging in in-memory providers like Otter, defaulting
// to "default".
//
// Takes namespace (string) which is the namespace prefix for cache keys.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Namespace(namespace string) *CacheBuilder[K, V] {
	b.namespace = namespace
	return b
}

// FactoryBlueprint specifies a registered factory blueprint to use for creating
// the cache. This enables the creation of fully-typed caches for
// domain-specific types without circular dependencies between the cache hexagon
// and domain hexagons.
//
// Factory blueprints are registered by domain adapter packages via init()
// functions using RegisterProviderFactory(). This pattern allows each domain to
// teach the cache hexagon how to create typed caches for its specific types.
//
// Do not use FactoryBlueprint with Provider or MultiLevel. The factory
// blueprint handles provider selection internally.
//
// Takes name (string) which identifies the registered factory blueprint.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) FactoryBlueprint(name string) *CacheBuilder[K, V] {
	b.factoryBlueprint = name
	return b
}

// MultiLevel configures a multi-level cache with separate L1 and L2 providers.
// This is a high-level convenience method that handles the complex construction
// of a multi-level cache with circuit breaker protection for the L2 layer.
//
// Parameters:
//   - l1Provider: The name of the local/fast cache provider (e.g., "otter")
//   - l2Provider: The name of the remote/distributed cache provider (e.g.,
//     "redis")
//
// Advanced configuration can be done with L1Options and L2Options.
//
// Takes l1Provider (string) which names the local/fast cache provider.
// Takes l2Provider (string) which names the remote/distributed cache
// provider.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) MultiLevel(l1Provider, l2Provider string) *CacheBuilder[K, V] {
	b.isMultiLevel = true
	b.l1ProviderName = l1Provider
	b.l2ProviderName = l2Provider

	if b.l2MaxFailures == 0 {
		b.l2MaxFailures = defaultL2MaxFailures
	}
	if b.l2OpenTimeout == 0 {
		b.l2OpenTimeout = defaultL2OpenTimeoutSeconds * time.Second
	}

	return b
}

// L1Options sets provider-specific configuration for the L1 cache in a
// multi-level setup. This method should only be used after calling MultiLevel.
//
// Takes options (any) which provides provider-specific L1 configuration.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) L1Options(options any) *CacheBuilder[K, V] {
	b.l1ProviderOptions = options
	return b
}

// L2Options sets provider-specific configuration for the L2 cache in a
// multi-level setup. This method should only be used after calling MultiLevel.
//
// Takes options (any) which provides provider-specific L2 configuration.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) L2Options(options any) *CacheBuilder[K, V] {
	b.l2ProviderOptions = options
	return b
}

// L2CircuitBreaker configures the circuit breaker settings for the L2 provider.
// This protects your application from cascading failures if the remote cache
// becomes unavailable.
//
// Parameters:
//   - maxFailures: Number of consecutive failures before opening the circuit
//   - openTimeout: How long to wait before attempting to close the circuit
//
// Takes maxFailures (int) which is the number of consecutive failures
// before opening the circuit.
// Takes openTimeout (time.Duration) which is how long to wait before
// attempting to close the circuit.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) L2CircuitBreaker(maxFailures int, openTimeout time.Duration) *CacheBuilder[K, V] {
	b.l2MaxFailures = maxFailures
	b.l2OpenTimeout = openTimeout
	return b
}

// Options sets provider-specific configuration options via a provider's
// Config struct (e.g., provider_redis.Config) that the provider factory
// type-asserts and applies.
//
// Takes options (any) which is the provider-specific configuration struct.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Options(options any) *CacheBuilder[K, V] {
	b.providerOptions = options
	return b
}

// MaximumSize sets the maximum number of entries the cache may contain. The
// cache will evict entries that are less likely to be used as it approaches
// this limit.
//
// Takes size (int) which is the maximum number of entries allowed.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) MaximumSize(size int) *CacheBuilder[K, V] {
	b.maximumSize = size
	return b
}

// MaximumWeight sets the maximum weight of entries the cache may contain.
// This requires also calling Weigher to define how entries are weighed.
//
// Takes weight (uint64) which is the maximum total weight allowed.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) MaximumWeight(weight uint64) *CacheBuilder[K, V] {
	b.maximumWeight = weight
	return b
}

// InitialCapacity sets the minimum initial capacity for the cache.
//
// Takes capacity (int) which specifies the minimum size for internal data
// structures, avoiding expensive resizing operations.
//
// Returns *CacheBuilder[K, V] for further configuration.
func (b *CacheBuilder[K, V]) InitialCapacity(capacity int) *CacheBuilder[K, V] {
	b.initialCapacity = capacity
	return b
}

// Weigher sets a function to calculate the weight of each cache entry.
// This is required when using MaximumWeight.
//
// Takes weigher (func(key K, value V) uint32) which computes the weight of
// each entry.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Weigher(weigher func(key K, value V) uint32) *CacheBuilder[K, V] {
	b.weigher = weigher
	return b
}

// Transformer adds a cache value transformer by name from globally registered
// blueprints, callable multiple times to chain transformers that execute in
// priority order.
//
// Parameters:
//   - name: The transformer name (e.g., "zstd", "crypto-service")
//   - config: Optional configuration (variadic, can be omitted or a single
//     config value)
//
// Takes name (string) which identifies the registered transformer blueprint.
// Takes transformerConfig (...any) which is optional
// configuration for the transformer.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Transformer(name string, transformerConfig ...any) *CacheBuilder[K, V] {
	var config any
	if len(transformerConfig) > 0 {
		config = transformerConfig[0]
	}

	b.transformers = append(b.transformers, transformerSpec{
		name:    name,
		factory: nil,
		config:  config,
	})
	return b
}

// Compression adds zstd compression to the cache values.
// This is a convenience method that configures the zstd transformer
// automatically.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Compression() *CacheBuilder[K, V] {
	b.transformers = append(b.transformers, transformerSpec{
		name:    "zstd",
		factory: nil,
		config:  nil,
	})
	return b
}

// Encryption adds encryption to cache values by automatically configuring the
// crypto-service transformer with the global crypto service obtained from
// bootstrap at build time.
//
// For explicit crypto service injection (e.g. in tests), use
// EncryptionWithService.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Encryption() *CacheBuilder[K, V] {
	b.transformers = append(b.transformers, transformerSpec{
		name:    "crypto-service",
		factory: nil,
		config:  nil,
	})
	return b
}

// EncryptionWithService adds encryption using an explicit crypto service
// instance. Use it in tests or when a specific crypto service configuration
// is needed.
//
// The service parameter must implement crypto_domain.CryptoServicePort.
//
// Takes service (any) which is the crypto service instance for encryption.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) EncryptionWithService(service any) *CacheBuilder[K, V] {
	b.transformers = append(b.transformers, transformerSpec{
		name:    "crypto-service",
		factory: nil,
		config:  map[string]any{"cryptoService": service},
	})
	return b
}

// Encoder registers a type-specific encoder. Use it to apply efficient binary
// formats (like Gob) for specific types.
//
// For better compile-time type safety, prefer TypedEncoder when possible.
//
// Takes encoder (AnyEncoder) which is the encoder to register.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Encoder(encoder AnyEncoder) *CacheBuilder[K, V] {
	b.encoders = append(b.encoders, encoder)
	return b
}

// TypedEncoder registers a type-specific encoder with compile-time type safety.
// This method verifies that the encoder's type matches the cache's value type
// V, preventing runtime type errors.
//
// This is the recommended method for registering encoders as it catches type
// mismatches at compile time rather than runtime.
//
// Takes encoder (EncoderPort[V]) which is the type-safe encoder to
// register.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) TypedEncoder(encoder EncoderPort[V]) *CacheBuilder[K, V] {
	b.encoders = append(b.encoders, encoder.(AnyEncoder))
	return b
}

// DefaultEncoder sets the fallback encoder for types without specific encoders.
// If not called, JSON encoding is used as the default.
//
// Takes encoder (AnyEncoder) which is the fallback encoder for unmatched
// types.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) DefaultEncoder(encoder AnyEncoder) *CacheBuilder[K, V] {
	b.defaultEncoder = encoder
	return b
}

// ExpiryCalculator configures dynamic expiry times for cache entries.
//
// The calculator determines when entries expire based on their creation,
// update, or access.
//
// Takes calculator (ExpiryCalculator[K, V]) which computes expiry times.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) ExpiryCalculator(calculator cache_dto.ExpiryCalculator[K, V]) *CacheBuilder[K, V] {
	b.expiryCalculator = calculator
	return b
}

// RefreshCalculator configures automatic background refresh for cache entries.
// Stale entries are refreshed asynchronously while still serving the old value.
//
// Takes calculator (RefreshCalculator[K, V]) which determines when and how
// cache entries should be refreshed.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) RefreshCalculator(calculator cache_dto.RefreshCalculator[K, V]) *CacheBuilder[K, V] {
	b.refreshCalculator = calculator
	return b
}

// OnDeletion sets a callback to be invoked when entries are deleted.
// The callback runs asynchronously after the deletion completes.
//
// Takes callback (func(e cache_dto.DeletionEvent[K, V])) which handles
// deletion events.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) OnDeletion(callback func(e cache_dto.DeletionEvent[K, V])) *CacheBuilder[K, V] {
	b.onDeletion = callback
	return b
}

// OnAtomicDeletion sets a callback to be invoked during the atomic deletion
// operation. Use this when the callback must execute as part of the deletion
// transaction.
//
// Takes callback (func(e cache_dto.DeletionEvent[K, V])) which handles
// atomic deletion events.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) OnAtomicDeletion(callback func(e cache_dto.DeletionEvent[K, V])) *CacheBuilder[K, V] {
	b.onAtomicDeletion = callback
	return b
}

// StatsRecorder sets a custom statistics recorder for tracking cache
// performance.
//
// Takes recorder (cache_dto.StatsRecorder) which collects cache performance
// metrics.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) StatsRecorder(recorder cache_dto.StatsRecorder) *CacheBuilder[K, V] {
	b.statsRecorder = recorder
	return b
}

// Executor sets a custom executor for asynchronous tasks (deletion callbacks,
// refreshes). By default, goroutines are used.
//
// Takes executor (func(operation func())) which runs asynchronous cache tasks.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Executor(executor func(operation func())) *CacheBuilder[K, V] {
	b.executor = executor
	return b
}

// Clock sets a custom clock for time-based operations. This is primarily useful
// for testing caches with expiration or refresh policies.
//
// Takes clock (cache_dto.Clock) which provides time functions for expiry
// calculations.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Clock(clock cache_dto.Clock) *CacheBuilder[K, V] {
	b.clock = clock
	return b
}

// Searchable configures the cache with a search schema for
// full-text search and structured query operations, defining
// which fields of cached values are searchable and how they
// should be indexed.
//
// When a search schema is configured, providers that support
// search (Otter, RediSearch) build internal structures for
// efficient searching, Search() and Query() become available,
// and SupportsSearch() returns true. When no schema is
// configured (default), Search() and Query() return
// ErrSearchNotSupported and SupportsSearch() returns false.
//
// Searchable cannot be used with transformers (Compression,
// Encryption) because transformed values are stored as bytes,
// not structured data.
//
// Takes schema (*cache_dto.SearchSchema) which defines the
// searchable fields and their types.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Searchable(schema *cache_dto.SearchSchema) *CacheBuilder[K, V] {
	b.searchSchema = schema
	return b
}

// validateSearchSchema checks that all fields in the search schema have valid
// configuration.
//
// Returns error when any field has an empty name, when a vector field has a
// non-positive dimension, or when a vector field has an empty distance metric.
func (b *CacheBuilder[K, V]) validateSearchSchema() error {
	if b.searchSchema == nil {
		return nil
	}
	for _, f := range b.searchSchema.Fields {
		if f.Name == "" {
			return errors.New("cache: search schema field has empty name")
		}
		if f.Type == cache_dto.FieldTypeVector && f.Dimension <= 0 {
			return fmt.Errorf("cache: vector field %q has non-positive dimension (%d)", f.Name, f.Dimension)
		}
		if f.Type == cache_dto.FieldTypeVector && f.DistanceMetric == "" {
			return fmt.Errorf("cache: vector field %q has empty distance metric", f.Name)
		}
	}
	return nil
}

// Logger sets a custom logger for the cache.
//
// Takes logger (cache_dto.Logger) which handles log output for cache
// operations.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Logger(logger cache_dto.Logger) *CacheBuilder[K, V] {
	b.cacheLogger = logger
	return b
}

// Expiration is a convenience method equivalent to WriteExpiration that sets
// a fixed expiration time for all entries via an ExpiryCalculator.
//
// Takes duration (time.Duration) which is the fixed expiry time for all
// entries.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) Expiration(duration time.Duration) *CacheBuilder[K, V] {
	return b.WriteExpiration(duration)
}

// WriteExpiration sets a fixed TTL applied on creation and updates only
// (not reset on reads), ideal for data with a natural expiration time
// regardless of access patterns.
//
// Takes duration (time.Duration) which is the TTL applied on creation and
// update.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) WriteExpiration(duration time.Duration) *CacheBuilder[K, V] {
	b.expiryCalculator = &writeExpiryCalculator[K, V]{duration: duration}
	return b
}

// AccessExpiration sets a "sliding" TTL that is reset on every access (read,
// write, or update), ideal for session data that should remain cached as long
// as it continues to be used, mimicking Otter's ExpiryAccessing behaviour.
//
// Takes duration (time.Duration) which is the sliding TTL reset on every
// access.
//
// Returns *CacheBuilder[K, V] for method chaining.
func (b *CacheBuilder[K, V]) AccessExpiration(duration time.Duration) *CacheBuilder[K, V] {
	b.expiryCalculator = &accessExpiryCalculator[K, V]{duration: duration}
	return b
}

// writeExpiryCalculator applies a time-to-live only on write operations such
// as create and update. Read operations do not reset the expiry timer.
type writeExpiryCalculator[K comparable, V any] struct {
	// duration is how long until the content expires.
	duration time.Duration
}

// ExpireAfterCreate returns the duration after which a newly created entry
// should expire.
//
// Takes cache_dto.Entry[K, V] which is the newly created cache entry.
//
// Returns time.Duration which is the configured expiry duration.
func (c *writeExpiryCalculator[K, V]) ExpireAfterCreate(cache_dto.Entry[K, V]) time.Duration {
	return c.duration
}

// ExpireAfterUpdate returns the duration after which an entry expires following
// an update.
//
// Takes cache_dto.Entry[K, V] which is the existing cache entry.
// Takes V which is the new value being written.
//
// Returns time.Duration which is the fixed expiry duration for updated entries.
func (c *writeExpiryCalculator[K, V]) ExpireAfterUpdate(cache_dto.Entry[K, V], V) time.Duration {
	return c.duration
}

// ExpireAfterRead returns the expiry duration after a read operation.
//
// Takes Entry[K, V] which is the cache entry that was read.
//
// Returns time.Duration which is -1 to indicate no change to expiry.
func (*writeExpiryCalculator[K, V]) ExpireAfterRead(cache_dto.Entry[K, V]) time.Duration {
	return -1
}

// accessExpiryCalculator applies a time-to-live to all operations.
// Each access resets the expiry timer, giving sliding expiry behaviour.
type accessExpiryCalculator[K comparable, V any] struct {
	// duration is the time period before access expires.
	duration time.Duration
}

// ExpireAfterCreate returns the configured sliding expiry duration for
// a newly created cache entry.
//
// Takes cache_dto.Entry[K, V] which is the newly created entry.
//
// Returns time.Duration which is the configured expiry duration.
func (c *accessExpiryCalculator[K, V]) ExpireAfterCreate(cache_dto.Entry[K, V]) time.Duration {
	return c.duration
}

// ExpireAfterUpdate returns the duration after which an entry expires following
// an update.
//
// Takes cache_dto.Entry[K, V] which is the existing cache entry.
// Takes V which is the new value being set.
//
// Returns time.Duration which is the configured expiry duration.
func (c *accessExpiryCalculator[K, V]) ExpireAfterUpdate(cache_dto.Entry[K, V], V) time.Duration {
	return c.duration
}

// ExpireAfterRead returns the duration after which an entry expires following
// a read.
//
// Takes Entry[K, V] which is the cache entry being accessed.
//
// Returns time.Duration which is the configured expiry duration.
func (c *accessExpiryCalculator[K, V]) ExpireAfterRead(cache_dto.Entry[K, V]) time.Duration {
	return c.duration
}

// Build constructs and returns the configured cache instance.
//
// This method performs all the complex wiring internally:
//   - Creates registries for transformers and encoders
//   - Wraps providers with TransformerWrapper when needed
//   - Configures provider-specific options
//   - Constructs multi-level caches when WithMultiLevel was called
//   - Returns a fully configured, ready-to-use Cache[K, V]
//
// When transformers or custom encoders are used, the returned
// cache is a *TransformerWrapper which implements the core Cache
// methods but may not implement all advanced methods like
// Compute, ComputeIfAbsent, etc.
//
// Returns Cache[K, V] which is the fully configured cache
// instance.
// Returns error when validation, provider creation, or wiring
// fails.
func (b *CacheBuilder[K, V]) Build(ctx context.Context) (Cache[K, V], error) {
	if err := b.validateSearchSchema(); err != nil {
		return nil, fmt.Errorf("validating search schema: %w", err)
	}

	if b.factoryBlueprint != "" {
		return b.buildFromBlueprint(ctx)
	}

	if b.isMultiLevel {
		return b.buildMultiLevelCache(ctx)
	}

	needsWrapper := len(b.transformers) > 0 || len(b.encoders) > 0 || b.defaultEncoder != nil

	if !needsWrapper {
		return b.buildSimpleCache(ctx)
	}

	return b.buildWrappedCache(ctx)
}

// buildFromBlueprint creates a cache using a registered factory blueprint.
// This enables domain-specific type support without circular dependencies.
//
// Returns Cache[K, V] which is the cache created by the factory blueprint.
// Returns error when the blueprint is not found or factory creation fails.
func (b *CacheBuilder[K, V]) buildFromBlueprint(ctx context.Context) (Cache[K, V], error) {
	_, l := logger_domain.From(ctx, log)
	builderLog := l.With(logger_domain.String("component", "cache_builder"))

	factory, exists := GetProviderFactory(b.factoryBlueprint)
	if !exists {
		return nil, fmt.Errorf(
			"factory blueprint '%s' not found - ensure the adapter package is imported",
			b.factoryBlueprint,
		)
	}

	builderLog.Internal("Creating cache using factory blueprint",
		logger_domain.String("blueprint", b.factoryBlueprint),
		logger_domain.String("namespace", b.namespace))

	options := cache_dto.Options[K, V]{
		Provider:          b.providerName,
		Namespace:         b.namespace,
		ProviderSpecific:  b.providerOptions,
		MaximumSize:       b.maximumSize,
		MaximumWeight:     b.maximumWeight,
		InitialCapacity:   b.initialCapacity,
		Weigher:           b.weigher,
		ExpiryCalculator:  b.expiryCalculator,
		OnDeletion:        b.onDeletion,
		OnAtomicDeletion:  b.onAtomicDeletion,
		RefreshCalculator: b.refreshCalculator,
		Executor:          b.executor,
		Clock:             b.clock,
		Logger:            b.cacheLogger,
		StatsRecorder:     b.statsRecorder,
		SearchSchema:      b.searchSchema,
	}

	cacheAny, err := factory(b.service, b.namespace, options)
	if err != nil {
		return nil, fmt.Errorf("factory blueprint '%s' failed: %w", b.factoryBlueprint, err)
	}

	cache, ok := cacheAny.(Cache[K, V])
	if !ok {
		return nil, fmt.Errorf(
			"factory blueprint '%s' returned incorrect type: expected Cache[%T, %T], got %T",
			b.factoryBlueprint, *new(K), *new(V), cacheAny,
		)
	}

	builderLog.Internal("Cache created from factory blueprint",
		logger_domain.String("blueprint", b.factoryBlueprint),
		logger_domain.String("namespace", b.namespace))

	return cache, nil
}

// buildSimpleCache creates a cache without transformation/encoding wrapper.
//
// Returns Cache[K, V] which is the directly created cache instance.
// Returns error when cache creation fails.
func (b *CacheBuilder[K, V]) buildSimpleCache(_ context.Context) (Cache[K, V], error) {
	options := cache_dto.Options[K, V]{
		Provider:          b.providerName,
		Namespace:         b.namespace,
		ProviderSpecific:  b.providerOptions,
		MaximumSize:       b.maximumSize,
		MaximumWeight:     b.maximumWeight,
		InitialCapacity:   b.initialCapacity,
		Weigher:           b.weigher,
		ExpiryCalculator:  b.expiryCalculator,
		OnDeletion:        b.onDeletion,
		OnAtomicDeletion:  b.onAtomicDeletion,
		RefreshCalculator: b.refreshCalculator,
		Executor:          b.executor,
		Clock:             b.clock,
		Logger:            b.cacheLogger,
		StatsRecorder:     b.statsRecorder,
		SearchSchema:      b.searchSchema,
	}

	cache, err := NewCache[K, V](b.service, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return cache, nil
}

// setupTransformerRegistry creates and fills a transformer registry from the
// builder's configuration.
//
// Returns *TransformerRegistry which contains all registered transformers.
// Returns *cache_dto.TransformConfig which holds the transformer settings.
// Returns error when a transformer cannot be created or registered.
func (b *CacheBuilder[K, V]) setupTransformerRegistry(ctx context.Context) (*TransformerRegistry, *cache_dto.TransformConfig, error) {
	if len(b.transformers) == 0 {
		return nil, nil, nil
	}

	registry := NewTransformerRegistry()
	config := &cache_dto.TransformConfig{
		EnabledTransformers: make([]string, 0, len(b.transformers)),
		TransformerOptions:  make(map[string]any),
	}

	for _, spec := range b.transformers {
		transformer, err := b.createTransformer(spec)
		if err != nil {
			return nil, nil, fmt.Errorf("creating transformer %q: %w", spec.name, err)
		}

		if err := registry.Register(ctx, transformer); err != nil {
			return nil, nil, fmt.Errorf("failed to register transformer '%s': %w", spec.name, err)
		}

		config.EnabledTransformers = append(config.EnabledTransformers, spec.name)
		if spec.config != nil {
			config.TransformerOptions[spec.name] = spec.config
		}
	}

	return registry, config, nil
}

// createTransformer creates a transformer from a specification.
//
// Takes spec (transformerSpec) which defines the transformer settings.
//
// Returns CacheTransformerPort which is the created transformer.
// Returns error when the transformer cannot be created.
func (b *CacheBuilder[K, V]) createTransformer(spec transformerSpec) (CacheTransformerPort, error) {
	if spec.factory != nil {
		return spec.factory(spec.config)
	}

	transformer, err := b.resolveConvenienceTransformer(spec.name, spec.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create transformer '%s': %w", spec.name, err)
	}

	return transformer, nil
}

// setupEncoderRegistry creates and populates an encoder registry from
// the builder's configuration.
//
// Returns *EncodingRegistry which contains the registered encoders.
// Returns error when an encoder fails to register.
func (b *CacheBuilder[K, V]) setupEncoderRegistry(ctx context.Context) (*EncodingRegistry, error) {
	if len(b.encoders) == 0 && b.defaultEncoder == nil {
		return nil, nil
	}

	registry := NewEncodingRegistry(b.defaultEncoder)

	for _, encoder := range b.encoders {
		if err := registry.Register(ctx, encoder); err != nil {
			return nil, fmt.Errorf("failed to register encoder: %w", err)
		}
	}

	return registry, nil
}

// createBaseByteCache creates the underlying byte-based cache provider.
//
// Returns ProviderPort[K, []byte] which is the byte-level cache provider.
// Returns error when the base cache cannot be created.
func (b *CacheBuilder[K, V]) createBaseByteCache(_ context.Context) (ProviderPort[K, []byte], error) {
	baseOptions := cache_dto.Options[K, []byte]{
		Provider:          b.providerName,
		Namespace:         b.namespace,
		ProviderSpecific:  b.providerOptions,
		MaximumSize:       b.maximumSize,
		MaximumWeight:     b.maximumWeight,
		InitialCapacity:   b.initialCapacity,
		Weigher:           b.adaptWeigher(),
		ExpiryCalculator:  b.adaptExpiryCalculator(),
		OnDeletion:        b.adaptOnDeletion(),
		OnAtomicDeletion:  b.adaptOnAtomicDeletion(),
		RefreshCalculator: b.adaptRefreshCalculator(),
		Executor:          b.executor,
		Clock:             b.clock,
		Logger:            b.cacheLogger,
		StatsRecorder:     b.statsRecorder,
	}

	baseCache, err := NewCache[K, []byte](b.service, baseOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create base cache: %w", err)
	}

	return baseCache, nil
}

// selectEncoderForWrapper determines the appropriate encoder for the wrapper.
//
// Takes registry (*EncodingRegistry) which provides type-to-encoder
// mappings.
//
// Returns []wrapperOption[K, V] which contains the encoder option for the
// wrapper, or nil if no suitable encoder is found.
func (b *CacheBuilder[K, V]) selectEncoderForWrapper(registry *EncodingRegistry) []wrapperOption[K, V] {
	if registry == nil {
		return nil
	}

	var v V
	anyEncoder, err := registry.GetByType(reflect.TypeOf(v))
	if err == nil {
		if typedEncoder, ok := anyEncoder.(EncoderPort[V]); ok {
			return []wrapperOption[K, V]{withWrapperEncoder[K, V](typedEncoder)}
		}
	}

	if b.defaultEncoder != nil {
		if typedDefault, ok := b.defaultEncoder.(EncoderPort[V]); ok {
			return []wrapperOption[K, V]{withWrapperEncoder[K, V](typedDefault)}
		}
	}

	return nil
}

// buildWrappedCache creates a cache with transformation/encoding wrapper.
//
// Returns *transformerWrapper[K, V] which wraps the base cache with
// encoding and transformation logic.
// Returns error when registry setup, base cache creation, or wrapper
// construction fails.
func (b *CacheBuilder[K, V]) buildWrappedCache(ctx context.Context) (*transformerWrapper[K, V], error) {
	b.warnAboutTransformerLimitations(ctx)

	transformerRegistry, transformConfig, err := b.setupTransformerRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("setting up transformer registry: %w", err)
	}

	encoderRegistry, err := b.setupEncoderRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("setting up encoder registry: %w", err)
	}

	baseCache, err := b.createBaseByteCache(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating base byte cache: %w", err)
	}

	wrapperOpts := b.selectEncoderForWrapper(encoderRegistry)
	wrapper := newTransformerWrapper[K, V](
		baseCache,
		transformerRegistry,
		transformConfig,
		wrapperOpts...,
	)

	return wrapper, nil
}

// buildMultiLevelCache creates a multi-level cache by constructing L1 and L2
// providers and wrapping them with the MultiLevelAdapter.
//
// Returns Cache[K, V] which is the combined multi-level cache instance.
// Returns error when L1/L2 creation or adapter construction fails.
func (b *CacheBuilder[K, V]) buildMultiLevelCache(ctx context.Context) (Cache[K, V], error) {
	l1Provider, l2Provider, err := b.createL1L2Providers(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating L1/L2 providers: %w", err)
	}

	multilevelProvider, err := b.createMultiLevelAdapter(ctx, l1Provider, l2Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-level adapter: %w", err)
	}

	return multilevelProvider, nil
}

// createL1L2Providers creates and returns the L1 and L2 cache providers for
// multi-level caching. Both providers are type-asserted to ProviderPort to
// ensure they implement the required interface.
//
// Returns l1 (ProviderPort[K, V]) which is the L1 (local/fast) provider.
// Returns l2 (ProviderPort[K, V]) which is the L2 (remote/distributed)
// provider.
// Returns err (error) when provider creation or type assertion fails.
func (b *CacheBuilder[K, V]) createL1L2Providers(_ context.Context) (l1 ProviderPort[K, V], l2 ProviderPort[K, V], err error) {
	l1Cache, err := NewCache[K, V](b.service, b.buildL1Options())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create L1 cache provider '%s': %w", b.l1ProviderName, err)
	}

	l2Cache, err := NewCache[K, V](b.service, b.buildL2Options())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create L2 cache provider '%s': %w", b.l2ProviderName, err)
	}

	//nolint:govet // generic assertion valid at runtime
	l1Provider, ok1 := l1Cache.(ProviderPort[K, V])
	l2Provider, ok2 := l2Cache.(ProviderPort[K, V]) //nolint:govet // generic assertion valid at runtime

	if !ok1 || !ok2 {
		return nil, nil, fmt.Errorf("provider does not implement ProviderPort interface (L1=%t, L2=%t)", ok1, ok2)
	}

	return l1Provider, l2Provider, nil
}

// buildL1Options constructs the cache options for the L1 (local/fast) cache
// layer. L1 includes size/weight limits and local capacity settings.
//
// Returns cache_dto.Options[K, V] which contains the L1 configuration.
func (b *CacheBuilder[K, V]) buildL1Options() cache_dto.Options[K, V] {
	return cache_dto.Options[K, V]{
		Provider:          b.l1ProviderName,
		ProviderSpecific:  b.l1ProviderOptions,
		MaximumSize:       b.maximumSize,
		MaximumWeight:     b.maximumWeight,
		InitialCapacity:   b.initialCapacity,
		Weigher:           b.weigher,
		ExpiryCalculator:  b.expiryCalculator,
		OnDeletion:        b.onDeletion,
		OnAtomicDeletion:  b.onAtomicDeletion,
		RefreshCalculator: b.refreshCalculator,
		Executor:          b.executor,
		Clock:             b.clock,
		Logger:            b.cacheLogger,
		StatsRecorder:     b.statsRecorder,
		SearchSchema:      b.searchSchema,
	}
}

// buildL2Options constructs the cache options for the L2 (remote/distributed)
// cache layer. L2 excludes size/weight limits as those are typically managed by
// the remote system.
//
// Returns cache_dto.Options[K, V] which contains the L2 configuration.
func (b *CacheBuilder[K, V]) buildL2Options() cache_dto.Options[K, V] {
	return cache_dto.Options[K, V]{
		Provider:          b.l2ProviderName,
		ProviderSpecific:  b.l2ProviderOptions,
		ExpiryCalculator:  b.expiryCalculator,
		OnDeletion:        b.onDeletion,
		OnAtomicDeletion:  b.onAtomicDeletion,
		RefreshCalculator: b.refreshCalculator,
		Executor:          b.executor,
		Clock:             b.clock,
		Logger:            b.cacheLogger,
		StatsRecorder:     b.statsRecorder,
		SearchSchema:      b.searchSchema,
	}
}

// createMultiLevelAdapter is a helper to construct the multi-level adapter.
// This is separated to allow for easier testing and to handle the import
// dependency.
//
// Takes l1 (ProviderPort[K, V]) which is the L1 (local/fast) cache
// provider.
// Takes l2 (ProviderPort[K, V]) which is the L2 (remote/distributed) cache
// provider.
//
// Returns Cache[K, V] which is the multi-level cache adapter.
// Returns error when the constructor is not registered or adapter creation
// fails.
func (b *CacheBuilder[K, V]) createMultiLevelAdapter(
	ctx context.Context,
	l1 ProviderPort[K, V],
	l2 ProviderPort[K, V],
) (Cache[K, V], error) {
	constructor, ok := getMultiLevelAdapterConstructor()
	if !ok {
		return nil, errors.New(
			"multi-level adapter constructor is not registered - " +
				"ensure provider_multilevel package is imported in your application " +
				"(use blank import: _ \"piko.sh/piko/internal/cache/cache_adapters/provider_multilevel\")",
		)
	}

	config := map[string]any{
		"l1_provider":     b.l1ProviderName,
		"l2_provider":     b.l2ProviderName,
		"l2_max_failures": b.l2MaxFailures,
		"l2_open_timeout": b.l2OpenTimeout,
	}

	result, err := constructor(ctx, "multilevel", l1, l2, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-level adapter: %w", err)
	}

	multilevelAdapter, ok := result.(Cache[K, V])
	if !ok {
		return nil, fmt.Errorf("multi-level adapter constructor returned invalid type: expected Cache[%T, %T]", *new(K), *new(V))
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Multi-level cache created successfully",
		logger_domain.String("l1_provider", b.l1ProviderName),
		logger_domain.String("l2_provider", b.l2ProviderName),
		logger_domain.Int("l2_max_failures", b.l2MaxFailures),
		logger_domain.Duration("l2_open_timeout", b.l2OpenTimeout))

	return multilevelAdapter, nil
}

// warnAboutTransformerLimitations logs warnings when features are set up
// that will not work with TransformerWrapper. This helps users understand the
// limits of transformer-based caches and prevents silent failures.
func (b *CacheBuilder[K, V]) warnAboutTransformerLimitations(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)

	if b.searchSchema != nil {
		l.Warn("Searchable is not supported when transformers are used",
			logger_domain.String(logKeyReason, "search requires structured values, not encoded bytes"),
			logger_domain.String("behaviour", "Search() and Query() will return ErrSearchNotSupported"))
	}

	if b.weigher != nil {
		l.Warn("WithWeigher is not supported when transformers are used and will be ignored",
			logger_domain.String(logKeyReason, "weighing requires decoded values"),
			logger_domain.String("alternative", "weight-based eviction will use byte size if supported by provider"))
	}

	if b.expiryCalculator != nil {
		isBuiltIn := false
		switch b.expiryCalculator.(type) {
		case *writeExpiryCalculator[K, V], *accessExpiryCalculator[K, V]:
			isBuiltIn = true
		}

		if !isBuiltIn {
			l.Warn("Custom ExpiryCalculators have limited support with transformers",
				logger_domain.String(logKeyReason, "calculators require decoded values"),
				logger_domain.String("recommendation", "use WithWriteExpiration or WithAccessExpiration for simple TTL needs"))
		}
	}

	if b.onDeletion != nil {
		l.Warn("WithOnDeletion callbacks are not supported when transformers are used and will be ignored",
			logger_domain.String(logKeyReason, "callbacks require decoded values"))
	}

	if b.onAtomicDeletion != nil {
		l.Warn("WithOnAtomicDeletion callbacks are not supported when transformers are used and will be ignored",
			logger_domain.String(logKeyReason, "callbacks require decoded values"))
	}

	if b.refreshCalculator != nil {
		l.Warn("WithRefreshCalculator is not supported when transformers are used and will be ignored",
			logger_domain.String(logKeyReason, "refresh logic requires decoded values"))
	}

	if b.maximumWeight > 0 && b.weigher == nil {
		l.Warn("WithMaximumWeight is set but no weigher is configured",
			logger_domain.String("behaviour", "provider will use default byte-based weighing if supported"))
	}
}

// resolveConvenienceTransformer creates transformer instances using the global
// blueprint registry.
//
// Takes name (string) which identifies the transformer blueprint to use.
// Takes config (any) which provides configuration for the transformer.
//
// Returns CacheTransformerPort which is the created transformer instance.
// Returns error when the transformer is not registered or creation fails.
func (*CacheBuilder[K, V]) resolveConvenienceTransformer(name string, config any) (CacheTransformerPort, error) {
	factory, exists := getTransformerBlueprint(name)
	if !exists {
		return nil, fmt.Errorf("transformer '%s' is not registered in the blueprint registry", name)
	}

	transformer, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create transformer '%s': %w", name, err)
	}

	return transformer, nil
}

// adaptWeigher returns a weigher function for byte slice values.
//
// Returns func(key K, value []byte) uint32 which is always nil because the
// wrapped cache weighs based on byte size rather than the original typed values.
func (b *CacheBuilder[K, V]) adaptWeigher() func(key K, value []byte) uint32 {
	if b.weigher == nil {
		return nil
	}
	return nil
}

// adaptExpiryCalculator returns nil because expiry calculators cannot
// be adapted for byte-level caches without deserialising the value.
//
// Returns cache_dto.ExpiryCalculator[K, []byte] which is always nil.
func (b *CacheBuilder[K, V]) adaptExpiryCalculator() cache_dto.ExpiryCalculator[K, []byte] {
	if b.expiryCalculator == nil {
		return nil
	}
	return nil
}

// adaptOnDeletion returns a deletion callback adapter.
//
// Returns func(e cache_dto.DeletionEvent[K, []byte]) which is always nil
// because deletion callbacks cannot be adapted without deserialisation.
func (b *CacheBuilder[K, V]) adaptOnDeletion() func(e cache_dto.DeletionEvent[K, []byte]) {
	if b.onDeletion == nil {
		return nil
	}
	return nil
}

// adaptOnAtomicDeletion returns nil because atomic deletion callbacks cannot
// be adapted without deserialisation.
//
// Returns func(e cache_dto.DeletionEvent[K, []byte]) which is always nil for
// this builder.
func (b *CacheBuilder[K, V]) adaptOnAtomicDeletion() func(e cache_dto.DeletionEvent[K, []byte]) {
	if b.onAtomicDeletion == nil {
		return nil
	}
	return nil
}

// adaptRefreshCalculator returns nil because refresh calculators
// cannot be adapted for byte-level caches without deserialising the
// value.
//
// Returns cache_dto.RefreshCalculator[K, []byte] which is always nil.
func (b *CacheBuilder[K, V]) adaptRefreshCalculator() cache_dto.RefreshCalculator[K, []byte] {
	if b.refreshCalculator == nil {
		return nil
	}
	return nil
}

// Clone creates a deep copy of the CacheBuilder, allowing for the creation
// of a cache template that can be modified and used multiple times.
//
// Returns *CacheBuilder[K, V] which is an independent copy of this builder.
func (b *CacheBuilder[K, V]) Clone() *CacheBuilder[K, V] {
	cloned := &CacheBuilder[K, V]{
		service:           b.service,
		providerName:      b.providerName,
		namespace:         b.namespace,
		providerOptions:   b.providerOptions,
		maximumSize:       b.maximumSize,
		maximumWeight:     b.maximumWeight,
		initialCapacity:   b.initialCapacity,
		weigher:           b.weigher,
		expiryCalculator:  b.expiryCalculator,
		onDeletion:        b.onDeletion,
		onAtomicDeletion:  b.onAtomicDeletion,
		refreshCalculator: b.refreshCalculator,
		executor:          b.executor,
		clock:             b.clock,
		cacheLogger:       b.cacheLogger,
		statsRecorder:     b.statsRecorder,
		defaultEncoder:    b.defaultEncoder,
		searchSchema:      b.searchSchema,
	}

	cloned.transformers = make([]transformerSpec, len(b.transformers))
	copy(cloned.transformers, b.transformers)

	cloned.encoders = make([]AnyEncoder, len(b.encoders))
	copy(cloned.encoders, b.encoders)

	return cloned
}

// RegisterMultiLevelAdapterConstructor registers the constructor for creating
// multi-level adapters. This should be called by the provider_multilevel
// package in its init function.
//
// Takes constructor (MultiLevelAdapterConstructor) which provides the factory
// function for creating multi-level cache adapters.
//
// Panics if a constructor has already been registered.
//
// Safe for concurrent use.
func RegisterMultiLevelAdapterConstructor(constructor MultiLevelAdapterConstructor) {
	multiLevelAdapterMutex.Lock()
	defer multiLevelAdapterMutex.Unlock()

	if multiLevelAdapterConstructor != nil {
		panic("multi-level adapter constructor is already registered")
	}
	multiLevelAdapterConstructor = constructor
}

// RegisterTransformerBlueprint registers a factory function for creating
// transformers by name, so the builder can use simple methods like
// WithTransformer("zstd") without needing the caller to import transformer
// packages or build factory functions.
//
// Transformer adapters should call this in their init() function:
//
//	func init() {
//	    cache_domain.RegisterTransformerBlueprint("zstd",
//	        func(config any) (cache_domain.CacheTransformerPort, error) {
//	            config := cache_transformer_zstd.DefaultConfig()
//	            if config != nil {
//	                // Parse config...
//	            }
//	            return cache_transformer_zstd.NewZstdCacheTransformer(config)
//	        })
//	}
//
// Takes name (string) which identifies the transformer for lookup.
// Takes factory (TransformerBlueprintFactory) which creates transformer
// instances.
//
// Panics if a transformer with the given name is already registered.
//
// Safe for concurrent use by multiple goroutines.
func RegisterTransformerBlueprint(name string, factory TransformerBlueprintFactory) {
	transformerBlueprintsMutex.Lock()
	defer transformerBlueprintsMutex.Unlock()

	if _, exists := transformerBlueprints[name]; exists {
		panic(fmt.Sprintf("transformer blueprint '%s' is already registered", name))
	}
	transformerBlueprints[name] = factory
}

// NewCacheBuilder creates a cache builder for the specified key-value types.
//
// Takes service (Service) which provides access to cache providers and
// configuration.
//
// Returns *CacheBuilder[K, V] which is the new builder ready for
// configuration.
func NewCacheBuilder[K comparable, V any](service Service) *CacheBuilder[K, V] {
	return &CacheBuilder[K, V]{
		service:      service,
		transformers: make([]transformerSpec, 0),
		encoders:     make([]AnyEncoder, 0),
	}
}

// getMultiLevelAdapterConstructor gets the registered multi-level adapter
// constructor.
//
// Returns MultiLevelAdapterConstructor which is the registered constructor, or
// nil if none has been registered.
// Returns bool which is true if a constructor has been registered.
//
// Safe for use by multiple goroutines at the same time.
func getMultiLevelAdapterConstructor() (MultiLevelAdapterConstructor, bool) {
	multiLevelAdapterMutex.RLock()
	defer multiLevelAdapterMutex.RUnlock()
	return multiLevelAdapterConstructor, multiLevelAdapterConstructor != nil
}

// getTransformerBlueprint retrieves a registered transformer factory by name.
//
// Takes name (string) which is the unique identifier of the transformer.
//
// Returns TransformerBlueprintFactory which is the factory for creating the
// transformer.
// Returns bool which is true if the transformer was found, false otherwise.
//
// Safe for concurrent use by multiple goroutines.
func getTransformerBlueprint(name string) (TransformerBlueprintFactory, bool) {
	transformerBlueprintsMutex.RLock()
	defer transformerBlueprintsMutex.RUnlock()
	factory, exists := transformerBlueprints[name]
	return factory, exists
}
