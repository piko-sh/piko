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

package cache

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

// DeletionCause represents the reason an entry was removed from the cache.
type DeletionCause = cache_dto.DeletionCause

const (
	// CauseInvalidation means the entry was deleted by the user.
	CauseInvalidation = cache_dto.CauseInvalidation

	// CauseReplacement means the user replaced the entry's value.
	CauseReplacement = cache_dto.CauseReplacement

	// CauseOverflow means the entry was evicted due to size limits.
	CauseOverflow = cache_dto.CauseOverflow

	// CauseExpiration means the entry's expiration time has passed.
	CauseExpiration = cache_dto.CauseExpiration

	// ComputeActionSet indicates the computed value should be stored in the cache.
	ComputeActionSet = cache_dto.ComputeActionSet

	// ComputeActionDelete indicates the entry should be removed from the cache.
	ComputeActionDelete = cache_dto.ComputeActionDelete

	// ComputeActionNoop indicates that no action should be taken.
	ComputeActionNoop = cache_dto.ComputeActionNoop

	// TransformerCompression applies compression to cached values.
	TransformerCompression = cache_dto.TransformerCompression

	// TransformerEncryption applies encryption to cached values.
	TransformerEncryption = cache_dto.TransformerEncryption

	// TransformerCustom is for custom transformers defined by users.
	TransformerCustom = cache_dto.TransformerCustom

	// FieldTypeText supports full-text search with tokenization.
	FieldTypeText = cache_dto.FieldTypeText

	// FieldTypeTag supports exact match filtering without tokenization.
	FieldTypeTag = cache_dto.FieldTypeTag

	// FieldTypeNumeric supports range queries and numeric sorting.
	FieldTypeNumeric = cache_dto.FieldTypeNumeric

	// FieldTypeGeo supports geographic queries based on coordinates.
	FieldTypeGeo = cache_dto.FieldTypeGeo

	// FieldTypeVector supports vector similarity search using HNSW indexing.
	FieldTypeVector = cache_dto.FieldTypeVector

	// SortAsc sorts results in ascending order (A-Z, 0-9).
	SortAsc = cache_dto.SortAsc

	// SortDesc sorts results in descending order (Z-A, 9-0).
	SortDesc = cache_dto.SortDesc

	// FilterOpEq matches when the field equals the value.
	FilterOpEq = cache_dto.FilterOpEq

	// FilterOpNe matches when the field does not equal the value.
	FilterOpNe = cache_dto.FilterOpNe

	// FilterOpGt matches when the field is greater than the value.
	FilterOpGt = cache_dto.FilterOpGt

	// FilterOpGe matches when the field is greater than or equal to the value.
	FilterOpGe = cache_dto.FilterOpGe

	// FilterOpLt matches when the field is less than the value.
	FilterOpLt = cache_dto.FilterOpLt

	// FilterOpLe matches when the field is less than or equal to the value.
	FilterOpLe = cache_dto.FilterOpLe

	// FilterOpIn matches when the field value is in the provided set.
	FilterOpIn = cache_dto.FilterOpIn

	// FilterOpBetween matches when the field is within a range (inclusive).
	FilterOpBetween = cache_dto.FilterOpBetween

	// FilterOpPrefix matches when the field starts with the value (TAG fields).
	FilterOpPrefix = cache_dto.FilterOpPrefix
)

// ComputeAction represents the action to take after a compute function runs.
type ComputeAction = cache_dto.ComputeAction

// TransformerType identifies the kind of cache value transformer.
type TransformerType = cache_dto.TransformerType

// FieldType defines the type of a searchable field.
type FieldType = cache_dto.FieldType

// SortOrder specifies the direction of result sorting.
type SortOrder = cache_dto.SortOrder

// FilterOp defines filter comparison operations.
type FilterOp = cache_dto.FilterOp

// Cache is the interface for interacting with a cache instance.
// Its API mirrors maypok86/otter/v2's Cache API.
type Cache[K comparable, V any] = cache_domain.Cache[K, V]

// Service manages cache providers and creates configured cache instances.
type Service = cache_domain.Service

// Provider is the non-generic interface for cache providers that manage
// resources. A Provider manages connections, pools, and other shared
// resources, and creates type-specific namespaced cache instances on demand.
//
// This is the recommended way to create cache providers. See Provider
// documentation for the architectural benefits of the namespace pattern.
type Provider = cache_domain.Provider

// ProviderPort is the driven port interface that all cache providers must
// implement. Implement it to create custom cache providers.
type ProviderPort[K comparable, V any] = cache_domain.ProviderPort[K, V]

// EncoderPort defines how to encode and decode values of type V.
// Implement it to create custom encoders for cache changes.
type EncoderPort[V any] = cache_domain.EncoderPort[V]

// AnyEncoder is a non-generic interface that all typed EncoderPorts implement,
// so that different encoders can be stored in a single registry.
type AnyEncoder = cache_domain.AnyEncoder

// EncodingRegistry provides a thread-safe registry for type-specific encoders.
// Use this when configuring cache providers that need to serialise or
// deserialise values.
type EncodingRegistry = cache_domain.EncodingRegistry

// TransformerPort defines the interface for cache value transformers.
// Transformers can apply compression, encryption, or custom transformations.
type TransformerPort = cache_domain.CacheTransformerPort

// Builder builds caches with custom settings.
// Use NewCacheBuilder to create a builder.
type Builder[K comparable, V any] = cache_domain.CacheBuilder[K, V]

// Options holds the settings for creating a new cache instance.
type Options[K comparable, V any] = cache_dto.Options[K, V]

// Entry is an immutable snapshot of a key-value pair in the cache, including
// metadata.
type Entry[K comparable, V any] = cache_dto.Entry[K, V]

// DeletionEvent is an event passed to deletion handlers when entries are
// removed.
type DeletionEvent[K comparable, V any] = cache_dto.DeletionEvent[K, V]

// Loader computes or fetches values for the cache when a key is not found.
type Loader[K comparable, V any] = cache_dto.Loader[K, V]

// LoaderFunc is an adapter to allow ordinary functions to be used as loaders.
type LoaderFunc[K comparable, V any] = cache_dto.LoaderFunc[K, V]

// BulkLoader computes or fetches values for many keys at once.
type BulkLoader[K comparable, V any] = cache_dto.BulkLoader[K, V]

// BulkLoaderFunc is an adapter that allows ordinary functions to be used as
// bulk loaders. It is an alias for cache_dto.BulkLoaderFunc.
type BulkLoaderFunc[K comparable, V any] = cache_dto.BulkLoaderFunc[K, V]

// LoadResult holds the outcome of an asynchronous load or refresh operation.
type LoadResult[V any] = cache_dto.LoadResult[V]

// ComputeResult holds the result of a compute operation with optional TTL
// override. A zero TTL means use the cache's default expiration policy.
type ComputeResult[V any] = cache_dto.ComputeResult[V]

// ExpiryCalculator calculates when cache entries should expire.
type ExpiryCalculator[K comparable, V any] = cache_dto.ExpiryCalculator[K, V]

// RefreshCalculator calculates when cache entries should be asynchronously
// refreshed.
type RefreshCalculator[K comparable, V any] = cache_dto.RefreshCalculator[K, V]

// Clock is an interface for getting the current time, letting time-based
// features be tested with a mock clock.
type Clock = cache_dto.Clock

// Logger is an interface for logging cache events and errors.
type Logger = cache_dto.Logger

// TransformConfig sets up value changes for cache Set and Get actions.
type TransformConfig = cache_dto.TransformConfig

// Stats represents a snapshot of cache statistics at a point in time.
type Stats = cache_dto.Stats

// StatsRecorder is an interface for recording cache statistics.
type StatsRecorder = cache_dto.StatsRecorder

var (
	// ErrNotFound is returned by a Loader when a value is missing from the data
	// source.
	ErrNotFound = cache_dto.ErrNotFound

	// ErrSearchNotSupported is returned when a provider does not support search
	// operations. Check the provider documentation for search capabilities.
	ErrSearchNotSupported = cache_domain.ErrSearchNotSupported
)

// TextAnalyseFunc transforms text into a slice of index terms, handling
// tokenisation, normalisation, stemming, and stop word removal. When nil,
// providers use their default tokenisation.
type TextAnalyseFunc = cache_dto.TextAnalyseFunc

// SearchSchema defines which fields of a cached value type are searchable.
// This schema enables providers to build appropriate internal structures
// for efficient search and query operations.
type SearchSchema = cache_dto.SearchSchema

// FieldSchema defines a single searchable field in a cached value type.
type FieldSchema = cache_dto.FieldSchema

// SearchOptions configures a full-text search operation.
type SearchOptions = cache_dto.SearchOptions

// QueryOptions configures a structured query operation without full-text
// search.
type QueryOptions = cache_dto.QueryOptions

// SearchResult contains the results of a search or query operation.
type SearchResult[K comparable, V any] = cache_dto.SearchResult[K, V]

// SearchHit represents a single result from a search or query operation.
type SearchHit[K comparable, V any] = cache_dto.SearchHit[K, V]

// Filter represents a single filter condition for structured queries.
type Filter = cache_dto.Filter

// ProviderFactoryBlueprint is a function that creates a typed cache instance.
// It receives a Service, namespace, and type-erased options, and returns
// a typed cache (as any for storage).
//
// This pattern enables domain-specific cache types to be created without
// circular dependencies. External packages can register their own factory
// blueprints via init() functions.
type ProviderFactoryBlueprint = cache_domain.ProviderFactoryBlueprint

// NewSearchSchema creates a SearchSchema with the given fields.
//
// Takes fields (...FieldSchema) which defines the schema fields to include.
//
// Returns *SearchSchema which is the configured search schema.
func NewSearchSchema(fields ...FieldSchema) *SearchSchema {
	return cache_dto.NewSearchSchema(fields...)
}

// NewSearchSchemaWithAnalyser creates a SearchSchema with a text analyser for
// linguistic processing of TEXT fields. The analyser replaces the provider's
// default tokenisation, enabling stemming, normalisation, stop words, and
// other NLP features.
//
// Takes analyser (TextAnalyseFunc) which processes text into index terms.
// Takes fields (...FieldSchema) which define the searchable fields.
//
// Returns *SearchSchema with the analyser configured.
func NewSearchSchemaWithAnalyser(analyser TextAnalyseFunc, fields ...FieldSchema) *SearchSchema {
	return cache_dto.NewSearchSchemaWithAnalyser(analyser, fields...)
}

// TextField creates a FieldSchema for full-text search.
//
// Takes name (string) which specifies the field name to index.
//
// Returns FieldSchema which defines the text field configuration.
func TextField(name string) FieldSchema {
	return cache_dto.TextField(name)
}

// TagField creates a FieldSchema for exact match filtering.
//
// Takes name (string) which specifies the tag field name to filter on.
//
// Returns FieldSchema which is configured for exact match filtering.
func TagField(name string) FieldSchema {
	return cache_dto.TagField(name)
}

// NumericField creates a FieldSchema for range queries.
//
// Takes name (string) which specifies the field name for numeric indexing.
//
// Returns FieldSchema which defines the numeric field configuration.
func NumericField(name string) FieldSchema {
	return cache_dto.NumericField(name)
}

// SortableNumericField creates a sortable FieldSchema for range queries.
//
// Takes name (string) which specifies the field name.
//
// Returns FieldSchema which is configured for sortable numeric operations.
func SortableNumericField(name string) FieldSchema {
	return cache_dto.SortableNumericField(name)
}

// SortableTextField creates a sortable FieldSchema for full-text search.
//
// Takes name (string) which specifies the field name to make sortable.
//
// Returns FieldSchema which is configured for sortable full-text search.
func SortableTextField(name string) FieldSchema {
	return cache_dto.SortableTextField(name)
}

// GeoField creates a FieldSchema for geographic queries.
//
// Takes name (string) which specifies the field name for geographic data.
//
// Returns FieldSchema which defines the schema for geographic queries.
func GeoField(name string) FieldSchema {
	return cache_dto.GeoField(name)
}

// VectorField creates a FieldSchema for vector similarity search with the
// default cosine distance metric.
//
// Takes name (string) which specifies the field name.
// Takes dimension (int) which sets the vector dimension size.
//
// Returns FieldSchema which is configured for vector similarity search.
func VectorField(name string, dimension int) FieldSchema {
	return cache_dto.VectorField(name, dimension)
}

// VectorFieldWithMetric creates a FieldSchema for vector similarity search
// with a custom distance metric.
//
// Takes name (string) which specifies the field name.
// Takes dimension (int) which sets the vector dimension size.
// Takes metric (string) which defines the distance metric to use.
//
// Returns FieldSchema which is configured for vector similarity search.
func VectorFieldWithMetric(name string, dimension int, metric string) FieldSchema {
	return cache_dto.VectorFieldWithMetric(name, dimension, metric)
}

// Eq creates an equality filter.
//
// Takes field (string) which specifies the field name to match against.
// Takes value (any) which specifies the value to compare for equality.
//
// Returns Filter which matches records where the field equals the value.
func Eq(field string, value any) Filter {
	return cache_dto.Eq(field, value)
}

// Ne creates a not-equal filter for the specified field and value.
//
// Takes field (string) which specifies the field name to compare.
// Takes value (any) which specifies the value that must not match.
//
// Returns Filter which excludes records where the field equals the value.
func Ne(field string, value any) Filter {
	return cache_dto.Ne(field, value)
}

// Gt creates a greater-than filter for the specified field.
//
// Takes field (string) which specifies the field name to compare.
// Takes value (any) which specifies the value to compare against.
//
// Returns Filter which matches records where the field exceeds the value.
func Gt(field string, value any) Filter {
	return cache_dto.Gt(field, value)
}

// Ge creates a greater-than-or-equal filter.
//
// Takes field (string) which specifies the field name to compare.
// Takes value (any) which specifies the value to compare against.
//
// Returns Filter which matches records where the field is greater than or
// equal to the value.
func Ge(field string, value any) Filter {
	return cache_dto.Ge(field, value)
}

// Lt creates a less-than filter for the specified field.
//
// Takes field (string) which is the name of the field to compare.
// Takes value (any) which is the value to compare against.
//
// Returns Filter which matches records where field is less than value.
func Lt(field string, value any) Filter {
	return cache_dto.Lt(field, value)
}

// Le creates a less-than-or-equal filter for the specified field.
//
// Takes field (string) which specifies the field name to filter on.
// Takes value (any) which specifies the upper bound value to compare against.
//
// Returns Filter which can be used to query records where field <= value.
func Le(field string, value any) Filter {
	return cache_dto.Le(field, value)
}

// In creates a set membership filter.
//
// Takes field (string) which specifies the field name to match against.
// Takes values (...any) which provides the set of values to check membership.
//
// Returns Filter which matches records where the field value is in the set.
func In(field string, values ...any) Filter {
	return cache_dto.In(field, values...)
}

// Between creates a range filter (inclusive).
//
// Takes field (string) which specifies the field name to filter on.
// Takes minVal (any) which specifies the lower bound of the range.
// Takes maxVal (any) which specifies the upper bound of the range.
//
// Returns Filter which matches records where the field value falls within the
// specified range, including both boundary values.
func Between(field string, minVal, maxVal any) Filter {
	return cache_dto.Between(field, minVal, maxVal)
}

// Prefix creates a prefix match filter for TAG fields.
//
// Takes field (string) which specifies the TAG field name to match against.
// Takes prefix (string) which specifies the prefix value to match.
//
// Returns Filter which matches entries where the field starts with the prefix.
func Prefix(field string, prefix string) Filter {
	return cache_dto.Prefix(field, prefix)
}

// NewService creates a new cache service with the specified default provider.
// The default provider is used when creating caches that do not explicitly
// specify a provider.
//
// Takes defaultProvider (string) which specifies the cache provider to use by
// default.
//
// Returns Service which is the configured cache service ready for use.
//
// Example:
//
//	service := cache.NewService("otter")
func NewService(defaultProvider string) Service {
	return cache_domain.NewService(defaultProvider)
}

// CreateNamespace creates a new cache instance using the namespace pattern.
// This is the RECOMMENDED way to create caches with the new Provider interface.
//
// Takes service (Service) which is the cache service managing providers.
// Takes providerName (string) which identifies the registered provider
// to use.
// Takes namespace (string) which is the logical namespace for the cache
// (e.g. "users", "products").
// Takes options (Options[K, V]) which configures the cache behaviour.
//
// Returns Cache[K, V] which is the configured cache instance.
// Returns error when the provider is not registered or the cache cannot
// be created.
//
// Example:
//
//	opts := cache.Options[string, User]{MaximumSize: 10000}
//	userCache, err := cache.CreateNamespace[string, User](service, "redis", "users", opts)
func CreateNamespace[K comparable, V any](ctx context.Context, service Service, providerName, namespace string, options Options[K, V]) (Cache[K, V], error) {
	return cache_domain.CreateNamespace[K, V](ctx, service, providerName, namespace, options)
}

// NewCache creates a cache instance directly from options without using the
// builder. Use it when you have pre-constructed Options.
//
// Takes service (Service) which manages providers and creates caches.
// Takes options (Options[K, V]) which specifies the full cache
// configuration.
//
// Returns Cache[K, V] which is the configured cache instance.
// Returns error when the cache cannot be created from the given options.
//
// Example:
//
//	opts := cache.Options[string, string]{
//	    Provider: "otter",
//	    Namespace: "users",
//	    MaximumSize: 1000,
//	}
//	myCache, err := cache.NewCache(service, opts)
func NewCache[K comparable, V any](service Service, options Options[K, V]) (Cache[K, V], error) {
	return cache_domain.NewCache[K, V](service, options)
}

// NewCacheBuilder creates a cache builder.
//
// Takes service (Service) which manages cache providers and creation.
//
// Returns *Builder[K, V] which provides a fluent interface for configuring a
// cache.
// Returns error when service is nil.
//
// Example:
//
//	builder, err := cache.NewCacheBuilder[string, []byte](service)
//	if err != nil {
//	    return err
//	}
//	myCache, err := builder.
//	    WithProvider("redis").
//	    WithNamespace("products").
//	    WithMaximumSize(10000).
//	    WithCompression().
//	    Build(ctx)
func NewCacheBuilder[K comparable, V any](service Service) (*Builder[K, V], error) {
	if service == nil {
		return nil, errors.New("cache: service must not be nil")
	}
	return cache_domain.NewCacheBuilder[K, V](service), nil
}

// NewEncodingRegistry creates a new encoding registry with an optional
// default encoder. The default encoder is used as a fallback when no
// type-specific encoder is registered.
//
// Takes defaultEncoder (AnyEncoder) which provides the fallback encoder for
// unregistered types, or nil for no default.
//
// Returns *EncodingRegistry which is the configured registry ready for use.
func NewEncodingRegistry(defaultEncoder AnyEncoder) *EncodingRegistry {
	return cache_domain.NewEncodingRegistry(defaultEncoder)
}

// GetDefaultService returns the global cache service instance from the Piko
// framework.
//
// Returns Service which is the configured cache service.
// Returns error when the framework is not initialised or the service
// cannot be created.
//
// Example:
//
//	service, err := cache.GetDefaultService()
//	if err != nil {
//	    return err
//	}
//	myCache, err := cache.NewCacheBuilder[string, string](service).
//	    WithMaximumSize(1000).
//	    Build(ctx)
func GetDefaultService() (Service, error) {
	service, err := bootstrap.GetCacheService()
	if err != nil {
		return nil, fmt.Errorf("cache: get default service: %w", err)
	}
	return service, nil
}

// NewCacheFromDefault creates a cache using the global service from the Piko
// framework. It's a convenience wrapper around GetDefaultService() and
// NewCache().
//
// Takes options (Options[K, V]) which specifies the cache configuration.
//
// Returns Cache[K, V] which is the configured cache instance.
// Returns error when the framework is not initialised or the cache cannot be
// created.
//
// Example:
//
//	opts := cache.Options[string, string]{MaximumSize: 1000}
//	myCache, err := cache.NewCacheFromDefault(opts)
func NewCacheFromDefault[K comparable, V any](options Options[K, V]) (Cache[K, V], error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf("cache: get default service: %w", err)
	}
	return NewCache[K, V](service, options)
}

// NewCacheBuilderFromDefault creates a cache builder using the global service
// from the Piko framework. It's a convenience wrapper around
// GetDefaultService() and NewCacheBuilder().
//
// Returns *Builder[K, V] which is the configured builder ready for use.
// Returns error when the framework is not initialised.
//
// Example:
//
//	builder, err := cache.NewCacheBuilderFromDefault[string, string]()
//	if err != nil {
//	    return err
//	}
//	myCache, err := builder.
//	    WithMaximumSize(1000).
//	    Build(ctx)
func NewCacheBuilderFromDefault[K comparable, V any]() (*Builder[K, V], error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, err
	}
	return NewCacheBuilder[K, V](service)
}

// RegisterProviderFactory registers a named factory function that creates typed
// cache instances. Factories registered here can be retrieved by name when
// building caches.
//
// Takes name (string) which identifies the factory for later retrieval.
// Takes factory (ProviderFactoryBlueprint) which creates typed cache instances.
//
// Example:
//
//	func init() {
//	    cache.RegisterProviderFactory("my-domain-cache",
//	        func(service cache.Service, namespace string, options any) (any, error) {
//	            opts, ok := options.(cache.Options[string, *MyType])
//	            if !ok {
//	                return nil, errors.New("invalid options type")
//	            }
//	            return cache_provider_otter.OtterProviderFactory[string, *MyType](opts)
//	        },
//	    )
//	}
func RegisterProviderFactory(name string, factory ProviderFactoryBlueprint) {
	cache_domain.RegisterProviderFactory(name, factory)
}
