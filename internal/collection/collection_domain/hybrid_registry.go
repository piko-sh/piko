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

package collection_domain

import (
	"context"
	"fmt"
	"slices"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	// logKeyHybridProvider is the log field key for the hybrid provider name.
	logKeyHybridProvider = "provider"

	// logKeyHybridCollection is the log key for the hybrid collection name.
	logKeyHybridCollection = "collection"

	// defaultHybridCacheMaxEntries is the maximum number of hybrid collections
	// that can be cached. This is a generous default; most applications have
	// far fewer hybrid collections.
	defaultHybridCacheMaxEntries = 10_000
)

// HybridCacheValue holds the runtime state for a single hybrid collection.
// Exported so the bootstrap layer can reference the cache type parameter.
type HybridCacheValue struct {
	// LastRevalidated is when the cache entry was last checked for freshness.
	LastRevalidated time.Time

	// CurrentBlob holds the current FlatBuffer blob, which may differ from
	// the embedded version after revalidation.
	CurrentBlob []byte

	// SnapshotBlob is the original blob saved at build time.
	SnapshotBlob []byte

	// CurrentETag is the ETag for the current cached content.
	CurrentETag string

	// SnapshotETag is the ETag value set when the cache entry was built.
	SnapshotETag string

	// ProviderName identifies the provider that owns this collection.
	ProviderName string

	// CollectionName identifies the collection for cache operations.
	CollectionName string

	// Config holds the revalidation settings for this cache entry.
	Config collection_dto.HybridConfig
}

var (
	// defaultHybridClock is the package-level clock used by global functions. It
	// defaults to RealClock but can be overridden for testing via setHybridClock.
	defaultHybridClock clock.Clock = clock.RealClock()

	_ HybridRegistryPort = (*defaultHybridRegistry)(nil)

	// hybridCache stores all hybrid collection entries, initialised eagerly so
	// that generated init() functions can call RegisterHybridSnapshot before
	// the application bootstrap runs.
	hybridCache cache_domain.Cache[string, HybridCacheValue]

	// defaultHybridRegistryInstance is the package-level default hybrid registry.
	// Used by global functions for backward compatibility.
	defaultHybridRegistryInstance = newDefaultHybridRegistry()
)

func init() {
	c, err := newDefaultHybridCache("hybrid-collections-startup")
	if err != nil {
		panic("failed to initialise hybrid cache: " + err.Error())
	}
	hybridCache = c
}

// newDefaultHybridCache builds a fresh hybrid cache instance using the
// otter provider factory.
//
// Takes namespace (string) which identifies the cache instance for telemetry.
//
// Returns cache_domain.Cache[string, HybridCacheValue] which is the
// constructed cache.
// Returns error which wraps any factory failure.
func newDefaultHybridCache(namespace string) (cache_domain.Cache[string, HybridCacheValue], error) {
	c, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, HybridCacheValue]{
		Namespace:   namespace,
		MaximumSize: defaultHybridCacheMaxEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("creating otter hybrid cache for %q: %w", namespace, err)
	}
	return c, nil
}

// InitHybridCache replaces the startup hybrid cache with a fully configured
// instance from the bootstrap layer. Existing entries registered during init()
// are migrated to the new cache.
//
// Takes c (cache_domain.Cache) which is the new cache instance.
func InitHybridCache(c cache_domain.Cache[string, HybridCacheValue]) {
	if c == nil {
		return
	}
	old := hybridCache
	hybridCache = c

	if old != nil {
		ctx := context.Background()
		for key, val := range old.All() { //nolint:gocritic // iterator yields values
			_ = c.Set(ctx, key, val)
		}
		_ = old.Close(ctx)
	}
}

// defaultHybridRegistry implements HybridRegistryPort with injectable
// dependencies. It wraps the package-level cache while allowing
// dependencies to be injected for testing.
//
// Thread-safety: All operations are safe for concurrent use.
type defaultHybridRegistry struct {
	// runtimeRegistry provides access to providers set up at runtime.
	runtimeRegistry RuntimeProviderRegistryPort

	// encoder encodes collections; nil falls back to staticCollectionRegistry.
	encoder CollectionEncoderPort

	// clock provides time operations for cache expiry checks.
	clock clock.Clock
}

// hybridRegistryOption is a functional option for configuring
// defaultHybridRegistry.
type hybridRegistryOption func(*defaultHybridRegistry)

// Register stores a hybrid snapshot for the given provider and collection.
// Implements HybridRegistryPort.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which specifies the collection to register.
// Takes blob ([]byte) which contains the snapshot data.
// Takes etag (string) which provides the version identifier for caching.
// Takes config (HybridConfig) which specifies the hybrid behaviour settings.
func (*defaultHybridRegistry) Register(ctx context.Context, providerName, collectionName string, blob []byte, etag string, config collection_dto.HybridConfig) {
	RegisterHybridSnapshot(ctx, providerName, collectionName, blob, etag, config)
}

// GetBlob implements HybridRegistryPort.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes providerName (string) which identifies the provider to query.
// Takes collectionName (string) which specifies the collection to retrieve.
//
// Returns blob ([]byte) which contains the retrieved data.
// Returns needsRevalidation (bool) which indicates whether the data should be
// refreshed.
func (*defaultHybridRegistry) GetBlob(ctx context.Context, providerName, collectionName string) (blob []byte, needsRevalidation bool) {
	return GetHybridBlob(ctx, providerName, collectionName)
}

// GetETag returns the ETag for the specified provider and collection.
// Implements HybridRegistryPort.
//
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which identifies the collection within the
// provider.
//
// Returns string which is the computed ETag value.
func (*defaultHybridRegistry) GetETag(providerName, collectionName string) string {
	return GetHybridETag(providerName, collectionName)
}

// Has reports whether a hybrid collection exists for the given provider and
// collection. Implements HybridRegistryPort.
//
// Takes providerName (string) which identifies the provider to check.
// Takes collectionName (string) which identifies the collection to check.
//
// Returns bool which is true if the hybrid collection exists.
func (*defaultHybridRegistry) Has(providerName, collectionName string) bool {
	return HasHybridCollection(providerName, collectionName)
}

// List returns the names of all registered hybrid collections.
//
// Implements HybridRegistryPort.
//
// Returns []string which contains the names of all registered hybrid
// collections.
func (*defaultHybridRegistry) List() []string {
	return ListHybridCollections()
}

// TriggerRevalidation implements HybridRegistryPort. It triggers background
// revalidation via the cache hexagon's Refresh mechanism, which handles
// deduplication of concurrent revalidation requests.
//
// Takes providerName (string) which identifies the provider to check again.
// Takes collectionName (string) which specifies the collection to check again.
func (h *defaultHybridRegistry) TriggerRevalidation(ctx context.Context, providerName, collectionName string) {
	key := makeHybridKey(providerName, collectionName)

	if _, ok, _ := hybridCache.GetIfPresent(ctx, key); !ok {
		return
	}

	encoder := h.encoder
	if encoder == nil {
		encoder = staticCollectionRegistry.encoder
	}

	loader := &hybridLoader{
		runtimeRegistry: h.runtimeRegistry,
		encoder:         encoder,
		clock:           h.clock,
	}

	_ = hybridCache.Refresh(ctx, key, loader)
}

// hybridCapableProvider defines the interface for providers that support
// hybrid mode.
type hybridCapableProvider interface {
	// ValidateETag checks whether the expected ETag matches the current state.
	ValidateETag(ctx context.Context, collectionName, expectedETag string) (string, bool, error)

	// FetchStaticContent retrieves all static content items from a collection.
	FetchStaticContent(ctx context.Context, collectionName string) ([]collection_dto.ContentItem, error)
}

// hybridError represents an error that occurs during hybrid revalidation.
type hybridError struct {
	// message holds the human-readable error description.
	message string
}

// Error implements the error interface for hybridError.
//
// Returns string which is the error message.
func (e *hybridError) Error() string {
	return e.message
}

// hybridLoader implements cache_dto.Loader for hybrid collection revalidation.
type hybridLoader struct {
	// runtimeRegistry provides access to providers set up at runtime.
	runtimeRegistry RuntimeProviderRegistryPort

	// encoder serialises collection items into a FlatBuffer blob.
	encoder CollectionEncoderPort

	// clock provides time operations for revalidation timestamps.
	clock clock.Clock
}

// Load is not used because hybrid entries are pre-registered via
// RegisterHybridSnapshot, never loaded on demand.
//
// Returns HybridCacheValue which is always zero-valued.
// Returns error which is always non-nil.
func (*hybridLoader) Load(_ context.Context, _ string) (HybridCacheValue, error) {
	return HybridCacheValue{}, &hybridError{message: "hybrid entries are pre-registered, not loaded on demand"}
}

// Reload performs the revalidation by checking the ETag against the provider,
// fetching fresh content if changed, and returning the updated value.
//
// Takes oldValue (HybridCacheValue) which is the current cached state to
// revalidate against.
//
// Returns HybridCacheValue which is the updated cache entry.
// Returns error which is nil on success.
func (l *hybridLoader) Reload(ctx context.Context, _ string, oldValue HybridCacheValue) (HybridCacheValue, error) {
	result := revalidateCollection(ctx, oldValue, l.runtimeRegistry, l.clock)
	return applyRevalidationResult(ctx, oldValue, result, l.encoder)
}

// RegisterHybridSnapshot registers a build-time snapshot for runtime use.
// Called from generated init() functions to register the embedded FlatBuffer
// blob and its ETag for hybrid mode operation.
//
// Takes ctx (context.Context) which carries logging context.
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which specifies the collection to register.
// Takes blob ([]byte) which contains the FlatBuffer snapshot data.
// Takes etag (string) which provides the version identifier for caching.
// Takes config (HybridConfig) which specifies the revalidation settings.
func RegisterHybridSnapshot(
	ctx context.Context,
	providerName, collectionName string,
	blob []byte,
	etag string,
	config collection_dto.HybridConfig,
) {
	key := makeHybridKey(providerName, collectionName)

	val := HybridCacheValue{
		LastRevalidated: defaultHybridClock.Now(),
		CurrentETag:     etag,
		SnapshotETag:    etag,
		ProviderName:    providerName,
		CollectionName:  collectionName,
		CurrentBlob:     blob,
		SnapshotBlob:    blob,
		Config:          config,
	}

	_ = hybridCache.Set(ctx, key, val)

	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered hybrid snapshot",
		logger_domain.String(logKeyHybridProvider, providerName),
		logger_domain.String(logKeyHybridCollection, collectionName),
		logger_domain.String("etag", etag),
		logger_domain.Int("blob_size", len(blob)))
}

// GetHybridBlob returns the current FlatBuffer blob and whether revalidation
// is needed.
//
// Takes ctx (context.Context) which carries logging context.
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which identifies the collection.
//
// Returns blob ([]byte) which contains the cached FlatBuffer data.
// Returns needsRevalidation (bool) which is true when the TTL has elapsed.
func GetHybridBlob(ctx context.Context, providerName, collectionName string) (blob []byte, needsRevalidation bool) {
	key := makeHybridKey(providerName, collectionName)

	val, ok, _ := hybridCache.GetIfPresent(ctx, key)
	if !ok {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Hybrid collection not registered",
			logger_domain.String(logKeyHybridProvider, providerName),
			logger_domain.String(logKeyHybridCollection, collectionName))
		return nil, false
	}

	return val.CurrentBlob, shouldRevalidate(val, defaultHybridClock)
}

// TriggerHybridRevalidation triggers background revalidation for a hybrid
// collection, returning immediately while concurrent calls are deduplicated.
//
// Takes ctx (context.Context) which carries logging context.
// Takes providerName (string) which identifies the provider to revalidate.
// Takes collectionName (string) which specifies the collection to revalidate.
func TriggerHybridRevalidation(ctx context.Context, providerName, collectionName string) {
	defaultHybridRegistryInstance.TriggerRevalidation(ctx, providerName, collectionName)
}

// GetHybridETag returns the current ETag for a hybrid collection.
//
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which identifies the collection.
//
// Returns string which is the current ETag, or empty if not registered.
func GetHybridETag(providerName, collectionName string) string {
	key := makeHybridKey(providerName, collectionName)

	val, ok, _ := hybridCache.GetIfPresent(context.Background(), key)
	if !ok {
		return ""
	}

	return val.CurrentETag
}

// HasHybridCollection reports whether a hybrid collection is registered.
//
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which identifies the collection.
//
// Returns bool which is true if the hybrid collection exists.
func HasHybridCollection(providerName, collectionName string) bool {
	key := makeHybridKey(providerName, collectionName)
	_, ok, _ := hybridCache.GetIfPresent(context.Background(), key)
	return ok
}

// ListHybridCollections returns all registered hybrid collection keys.
//
// Returns []string which contains the cache keys of all registered entries.
func ListHybridCollections() []string {
	return slices.Collect(hybridCache.Keys())
}

// ResetHybridRegistry clears the hybrid registry for test isolation by closing
// the current cache and creating a fresh one.
//
// On failure the previous cache instance is retained and the error is logged
// but not surfaced; callers that need to react to the failure should use
// TryResetHybridRegistry.
func ResetHybridRegistry() {
	if err := TryResetHybridRegistry(); err != nil {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("Failed to reset hybrid cache", logger_domain.Error(err))
	}
}

// TryResetHybridRegistry clears the hybrid registry for test isolation by
// closing the current cache and creating a fresh one. It is the error-aware
// sibling of ResetHybridRegistry.
//
// Returns error which wraps the underlying provider failure when the
// replacement cache cannot be created. The previous cache instance is
// retained on failure.
func TryResetHybridRegistry() error {
	ctx := context.Background()

	c, err := newDefaultHybridCache("hybrid-collections-test")
	if err != nil {
		return fmt.Errorf("resetting hybrid cache: %w", err)
	}

	_ = hybridCache.Close(ctx)
	hybridCache = c
	return nil
}

// withHybridRuntimeRegistry sets a custom runtime provider registry.
//
// Takes registry (RuntimeProviderRegistryPort) which replaces the default.
//
// Returns hybridRegistryOption which applies the override.
func withHybridRuntimeRegistry(registry RuntimeProviderRegistryPort) hybridRegistryOption {
	return func(h *defaultHybridRegistry) {
		h.runtimeRegistry = registry
	}
}

// withHybridEncoder sets a custom collection encoder.
//
// Takes encoder (CollectionEncoderPort) which replaces the default.
//
// Returns hybridRegistryOption which applies the override.
func withHybridEncoder(encoder CollectionEncoderPort) hybridRegistryOption {
	return func(h *defaultHybridRegistry) {
		h.encoder = encoder
	}
}

// withHybridClock sets a custom clock for time operations.
//
// Takes c (clock.Clock) which replaces the default clock.
//
// Returns hybridRegistryOption which applies the override.
func withHybridClock(c clock.Clock) hybridRegistryOption {
	return func(h *defaultHybridRegistry) {
		h.clock = c
	}
}

// newDefaultHybridRegistry creates a HybridRegistryPort with the given options.
//
// Takes opts (hybridRegistryOption) which configure the registry.
//
// Returns HybridRegistryPort which is the constructed registry.
func newDefaultHybridRegistry(opts ...hybridRegistryOption) HybridRegistryPort {
	h := &defaultHybridRegistry{
		runtimeRegistry: NewDefaultRuntimeProviderRegistry(),
		encoder:         nil,
		clock:           defaultHybridClock,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// makeHybridKey builds a registry key for a provider and collection pair.
//
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which identifies the collection.
//
// Returns string which is the colon-separated cache key.
func makeHybridKey(providerName, collectionName string) string {
	return providerName + ":" + collectionName
}

// shouldRevalidate checks if a hybrid cache value needs to be checked again
// based on its TTL configuration.
//
// Takes val (HybridCacheValue) which is the cached entry to inspect.
// Takes c (clock.Clock) which provides the current time.
//
// Returns bool which is true when the TTL has elapsed.
func shouldRevalidate(val HybridCacheValue, c clock.Clock) bool {
	if val.Config.RevalidationTTL <= 0 {
		return true
	}

	return c.Now().Sub(val.LastRevalidated) > val.Config.RevalidationTTL
}

// revalidateCollection performs the actual revalidation work by validating the
// ETag against the provider and fetching new content if changed.
//
// Takes ctx (context.Context) which carries logging context.
// Takes val (HybridCacheValue) which is the current cached state.
// Takes runtimeRegistry (RuntimeProviderRegistryPort) which provides access
// to runtime providers.
// Takes c (clock.Clock) which provides the current time for timestamps.
//
// Returns *HybridRevalidationResult which describes what changed.
func revalidateCollection(
	ctx context.Context,
	val HybridCacheValue,
	runtimeRegistry RuntimeProviderRegistryPort,
	c clock.Clock,
) *collection_dto.HybridRevalidationResult {
	result := &collection_dto.HybridRevalidationResult{
		ETagChanged:   false,
		RevalidatedAt: c.Now(),
	}

	hybridProvider, err := getHybridCapableProviderFromRegistry(val.ProviderName, runtimeRegistry)
	if err != nil {
		result.Error = err
		return result
	}

	return validateAndFetchIfChanged(ctx, hybridProvider, val, result)
}

// getHybridCapableProviderFromRegistry retrieves a provider and asserts it
// implements the hybridCapableProvider interface.
//
// Takes providerName (string) which identifies the provider to look up.
// Takes runtimeRegistry (RuntimeProviderRegistryPort) which holds runtime
// providers.
//
// Returns hybridCapableProvider which is the resolved provider.
// Returns error which is non-nil if the provider is missing or not hybrid
// capable.
func getHybridCapableProviderFromRegistry(
	providerName string,
	runtimeRegistry RuntimeProviderRegistryPort,
) (hybridCapableProvider, error) {
	provider, err := runtimeRegistry.Get(providerName)
	if err != nil {
		return nil, errProviderNotFound(providerName)
	}

	hybridProvider, ok := provider.(hybridCapableProvider)
	if !ok {
		return nil, errProviderNotHybridCapable(providerName)
	}

	return hybridProvider, nil
}

// validateAndFetchIfChanged checks the ETag and fetches new content if it has
// changed.
//
// Takes ctx (context.Context) which carries logging context.
// Takes provider (hybridCapableProvider) which is the source to validate
// against.
// Takes val (HybridCacheValue) which holds the current ETag and metadata.
// Takes result (*HybridRevalidationResult) which accumulates the outcome.
//
// Returns *HybridRevalidationResult which describes what changed.
func validateAndFetchIfChanged(
	ctx context.Context,
	provider hybridCapableProvider,
	val HybridCacheValue,
	result *collection_dto.HybridRevalidationResult,
) *collection_dto.HybridRevalidationResult {
	ctx, l := logger_domain.From(ctx, log)
	newETag, changed, err := provider.ValidateETag(ctx, val.CollectionName, val.CurrentETag)
	if err != nil {
		l.Warn("ETag validation failed",
			logger_domain.String(logKeyHybridProvider, val.ProviderName),
			logger_domain.String(logKeyHybridCollection, val.CollectionName),
			logger_domain.Error(err))
		result.Error = err
		return result
	}

	result.NewETag = newETag
	result.ETagChanged = changed

	if !changed {
		l.Trace("Content unchanged, ETag still valid",
			logger_domain.String(logKeyHybridProvider, val.ProviderName),
			logger_domain.String(logKeyHybridCollection, val.CollectionName),
			logger_domain.String("etag", newETag))
		return result
	}

	return fetchFreshContent(ctx, provider, val, newETag, result)
}

// fetchFreshContent gets new content after an ETag change is found.
//
// Takes ctx (context.Context) which carries logging context.
// Takes provider (hybridCapableProvider) which supplies the fresh items.
// Takes val (HybridCacheValue) which holds the current metadata for logging.
// Takes newETag (string) which is the updated ETag from the provider.
// Takes result (*HybridRevalidationResult) which accumulates the outcome.
//
// Returns *HybridRevalidationResult which contains the fetched items.
func fetchFreshContent(
	ctx context.Context,
	provider hybridCapableProvider,
	val HybridCacheValue,
	newETag string,
	result *collection_dto.HybridRevalidationResult,
) *collection_dto.HybridRevalidationResult {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Content changed, fetching fresh data",
		logger_domain.String(logKeyHybridProvider, val.ProviderName),
		logger_domain.String(logKeyHybridCollection, val.CollectionName),
		logger_domain.String("old_etag", val.CurrentETag),
		logger_domain.String("new_etag", newETag))

	items, err := provider.FetchStaticContent(ctx, val.CollectionName)
	if err != nil {
		l.Warn("Failed to fetch fresh content",
			logger_domain.String(logKeyHybridProvider, val.ProviderName),
			logger_domain.String(logKeyHybridCollection, val.CollectionName),
			logger_domain.Error(err))
		result.Error = err
		return result
	}

	result.NewItems = items
	return result
}

// applyRevalidationResult creates an updated HybridCacheValue from the
// revalidation outcome, always updating LastRevalidated to prevent rapid
// retries.
//
// Takes ctx (context.Context) which carries logging context.
// Takes oldValue (HybridCacheValue) which is the previous cached state.
// Takes result (*HybridRevalidationResult) which describes the revalidation
// outcome.
// Takes encoder (CollectionEncoderPort) which serialises updated items.
//
// Returns HybridCacheValue which is the updated cache entry.
// Returns error which is nil on success.
func applyRevalidationResult(
	ctx context.Context,
	oldValue HybridCacheValue,
	result *collection_dto.HybridRevalidationResult,
	encoder CollectionEncoderPort,
) (HybridCacheValue, error) {
	_, l := logger_domain.From(ctx, log)

	newValue := oldValue
	newValue.LastRevalidated = result.RevalidatedAt

	if result.Error != nil {
		if oldValue.Config.StaleIfError {
			l.Internal("Serving stale content due to revalidation error",
				logger_domain.String(logKeyHybridProvider, oldValue.ProviderName),
				logger_domain.String(logKeyHybridCollection, oldValue.CollectionName),
				logger_domain.Error(result.Error))
		}

		return newValue, nil
	}

	if !result.ETagChanged || len(result.NewItems) == 0 {
		return newValue, nil
	}

	blob, err := encodeItemsToBlob(result.NewItems, encoder)
	if err != nil {
		l.Warn("Failed to encode updated content",
			logger_domain.String(logKeyHybridProvider, oldValue.ProviderName),
			logger_domain.String(logKeyHybridCollection, oldValue.CollectionName),
			logger_domain.Error(err))
		return newValue, nil
	}

	newValue.CurrentBlob = blob
	newValue.CurrentETag = result.NewETag

	l.Internal("Updated hybrid cache with fresh content",
		logger_domain.String(logKeyHybridProvider, oldValue.ProviderName),
		logger_domain.String(logKeyHybridCollection, oldValue.CollectionName),
		logger_domain.String("new_etag", result.NewETag),
		logger_domain.Int("blob_size", len(blob)),
		logger_domain.Int("item_count", len(result.NewItems)))

	return newValue, nil
}

// encodeItemsToBlob encodes content items using the provided encoder.
//
// Takes items ([]ContentItem) which are the content items to serialise.
// Takes encoder (CollectionEncoderPort) which performs the serialisation.
//
// Returns []byte which is the encoded FlatBuffer blob.
// Returns error which is non-nil if encoding fails.
func encodeItemsToBlob(items []collection_dto.ContentItem, encoder CollectionEncoderPort) ([]byte, error) {
	return encoder.EncodeCollection(items)
}

// errProviderNotFound returns an error for a missing hybrid provider.
//
// Takes name (string) which identifies the provider that was not found.
//
// Returns error which describes the missing provider.
func errProviderNotFound(name string) error {
	return &hybridError{message: "hybrid provider not found: " + name}
}

// errProviderNotHybridCapable returns an error for a provider that does not
// support hybrid mode.
//
// Takes name (string) which identifies the incapable provider.
//
// Returns error which describes the unsupported operation.
func errProviderNotHybridCapable(name string) error {
	return &hybridError{message: "provider does not support hybrid mode: " + name}
}

// setHybridClock sets the package-level clock for testing.
//
// Takes c (clock.Clock) which replaces the default clock.
func setHybridClock(c clock.Clock) {
	defaultHybridClock = c
}
