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

package collection

import (
	"context"
	"go/ast"

	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

// CollectionProvider is the primary interface for data source adapters.
//
// Every data source (CMS, database, API, etc.) must implement CollectionProvider
// to integrate with the Piko collection system.
//
// The interface supports three provider types:
//   - Static: All data fetched at build time
//   - Dynamic: All data fetched at runtime
//   - Hybrid: Build-time snapshot with runtime revalidation (ISR)
type CollectionProvider = collection_domain.CollectionProvider

// RuntimeProvider defines the interface for providers that operate at runtime.
//
// Dynamic and Hybrid providers must implement RuntimeProvider AND register it
// with the runtime when the application starts.
type RuntimeProvider = collection_domain.RuntimeProvider

// ProviderType indicates how a provider fetches and handles data.
type ProviderType = collection_domain.ProviderType

const (
	// ProviderTypeStatic indicates a provider whose data is fetched at build time.
	// All content is embedded in the binary, resulting in zero runtime overhead.
	ProviderTypeStatic = collection_domain.ProviderTypeStatic

	// ProviderTypeDynamic indicates all data is fetched at runtime.
	// Ideal for: E-commerce products, user-generated content, real-time data.
	ProviderTypeDynamic = collection_domain.ProviderTypeDynamic

	// ProviderTypeHybrid indicates data is fetched at both build and runtime.
	// This implements the Incremental Static Regeneration (ISR) pattern.
	ProviderTypeHybrid = collection_domain.ProviderTypeHybrid

	// FilterOpEquals is the equals operator for exact value matching.
	FilterOpEquals = collection_dto.FilterOpEquals

	// FilterOpNotEquals is the not-equals comparison operator.
	FilterOpNotEquals = collection_dto.FilterOpNotEquals

	// FilterOpGreaterThan checks if a field value is greater than the filter value.
	FilterOpGreaterThan = collection_dto.FilterOpGreaterThan

	// FilterOpGreaterEqual checks if a field is greater than or equal to a value.
	FilterOpGreaterEqual = collection_dto.FilterOpGreaterEqual

	// FilterOpLessThan matches when the field value is smaller than the filter value.
	FilterOpLessThan = collection_dto.FilterOpLessThan

	// FilterOpLessEqual checks if a field is less than or equal to a value.
	FilterOpLessEqual = collection_dto.FilterOpLessEqual

	// FilterOpContains checks if a field value contains the given text.
	FilterOpContains = collection_dto.FilterOpContains

	// FilterOpStartsWith matches when a field value begins with a given prefix.
	FilterOpStartsWith = collection_dto.FilterOpStartsWith

	// FilterOpEndsWith matches when a field ends with the given suffix.
	FilterOpEndsWith = collection_dto.FilterOpEndsWith

	// FilterOpIn checks if a field value exists in the given array.
	FilterOpIn = collection_dto.FilterOpIn

	// FilterOpNotIn checks if a field value is not in the given list.
	FilterOpNotIn = collection_dto.FilterOpNotIn

	// FilterOpExists checks whether a field exists in the metadata.
	FilterOpExists = collection_dto.FilterOpExists

	// FilterOpFuzzyMatch is a filter operator for finding close text matches.
	FilterOpFuzzyMatch = collection_dto.FilterOpFuzzyMatch

	// SortAsc is the sort order for ascending results.
	SortAsc = collection_dto.SortAsc

	// SortDesc sorts results from highest to lowest.
	SortDesc = collection_dto.SortDesc

	// SortRandom shuffles results in random order.
	SortRandom = collection_dto.SortRandom

	// ETagSourceModtimeHash uses file modification times to compute the ETag.
	ETagSourceModtimeHash = collection_dto.ETagSourceModtimeHash

	// ETagSourceContentHash computes the ETag from the actual file content.
	ETagSourceContentHash = collection_dto.ETagSourceContentHash

	// ETagSourceProviderETag uses the ETag from the provider.
	ETagSourceProviderETag = collection_dto.ETagSourceProviderETag
)

// ContentItem represents a single piece of content from any provider.
//
// This is the universal representation of content that works across all
// data sources - markdown files, CMS entries, database records, etc.
type ContentItem = collection_dto.ContentItem

// CollectionInfo describes a collection offered by a provider.
//
// This is returned by DiscoverCollections() to inform the framework about
// available collections and their schemas.
type CollectionInfo = collection_dto.CollectionInfo

// ProviderConfig contains configuration for a collection provider.
//
// This is passed to providers during initialisation and collection discovery.
type ProviderConfig = collection_dto.ProviderConfig

// FetchOptions contains options for fetching collection data.
//
// These options are parsed from the user's GetCollection() call arguments
// and passed to providers during data fetching.
type FetchOptions = collection_dto.FetchOptions

// CacheConfig specifies caching behaviour for data fetching.
type CacheConfig = collection_dto.CacheConfig

// HybridConfig contains per-collection configuration for hybrid mode (ISR).
//
// This controls the revalidation behaviour for hybrid providers.
type HybridConfig = collection_dto.HybridConfig

// RuntimeFetcherCode represents generated Go code for runtime data fetching.
//
// This is the output of GenerateRuntimeFetcher() for dynamic providers.
type RuntimeFetcherCode = collection_dto.RuntimeFetcherCode

// RetryConfig specifies retry behaviour for failed fetches.
type RetryConfig = collection_dto.RetryConfig

// FieldProjection specifies which fields to include/exclude from responses.
// This reduces payload size by omitting fields the client doesn't need.
type FieldProjection = collection_dto.FieldProjection

// Filter represents a single condition used to query collections.
type Filter = collection_dto.Filter

// FilterGroup combines multiple filters with AND/OR logic.
type FilterGroup = collection_dto.FilterGroup

// FilterOperator defines a comparison type used to match filter values.
type FilterOperator = collection_dto.FilterOperator

// SortOption specifies how to sort collection items.
type SortOption = collection_dto.SortOption

// SortOrder defines the direction for sorting results.
type SortOrder = collection_dto.SortOrder

// PaginationOptions specifies how to divide results into pages.
type PaginationOptions = collection_dto.PaginationOptions

// PaginationMeta holds details about a paginated set of results.
type PaginationMeta = collection_dto.PaginationMeta

// HybridRevalidationResult is returned by background revalidation operations.
type HybridRevalidationResult = collection_dto.HybridRevalidationResult

// SimpleProvider is an interface for providers that only need basic operations.
//
// Use it for providers that don't need to implement the full
// CollectionProvider interface with AST-based code generation.
type SimpleProvider interface {
	// Name returns the unique name for this provider.
	Name() string

	// Type returns how this provider's data should be handled.
	Type() ProviderType

	// DiscoverCollections scans the provider's data source and returns
	// information about available collections.
	DiscoverCollections(ctx context.Context, config ProviderConfig) ([]CollectionInfo, error)

	// FetchContent retrieves all content from a collection.
	FetchContent(ctx context.Context, collectionName string, opts *FetchOptions) ([]ContentItem, error)

	// ComputeETag computes a content fingerprint for staleness detection.
	// Returns empty string if not supported.
	ComputeETag(ctx context.Context, collectionName string) (string, error)

	// ValidateETag checks if the current content matches an expected ETag.
	ValidateETag(ctx context.Context, collectionName string, expectedETag string) (currentETag string, changed bool, err error)
}

// SimpleProviderAdapter wraps a SimpleProvider to implement CollectionProvider.
//
// This adapter makes it easy to create providers without implementing
// AST-based code generation methods.
type SimpleProviderAdapter struct {
	// provider is the underlying SimpleProvider being adapted.
	provider SimpleProvider
}

// NewSimpleProviderAdapter creates an adapter that wraps a SimpleProvider.
//
// Takes provider (SimpleProvider) which is the provider to wrap.
//
// Returns *SimpleProviderAdapter which wraps the provider for use with the
// standard provider interface.
func NewSimpleProviderAdapter(provider SimpleProvider) *SimpleProviderAdapter {
	return &SimpleProviderAdapter{provider: provider}
}

// Name returns the unique name for this provider.
//
// Returns string which is the provider's unique identifier.
func (a *SimpleProviderAdapter) Name() string {
	return a.provider.Name()
}

// Type returns how this provider's data should be handled.
//
// Returns ProviderType which indicates the provider classification.
func (a *SimpleProviderAdapter) Type() ProviderType {
	return a.provider.Type()
}

// DiscoverCollections scans the provider's data source.
//
// Takes config (ProviderConfig) which specifies the provider settings.
//
// Returns []CollectionInfo which contains the discovered collections.
// Returns error when the scan fails.
func (a *SimpleProviderAdapter) DiscoverCollections(ctx context.Context, config ProviderConfig) ([]CollectionInfo, error) {
	return a.provider.DiscoverCollections(ctx, config)
}

// ValidateTargetType checks if a user's target struct is compatible.
// The simple adapter accepts all types.
//
// Returns error when the target type is incompatible; always nil for this
// adapter.
func (*SimpleProviderAdapter) ValidateTargetType(_ ast.Expr) error {
	return nil
}

// FetchStaticContent retrieves all content from a collection at build time.
//
// Takes collectionName (string) which specifies the collection to fetch.
//
// Returns []ContentItem which contains all items from the collection.
// Returns error when the collection cannot be fetched.
func (a *SimpleProviderAdapter) FetchStaticContent(ctx context.Context, collectionName string, _ collection_dto.ContentSource) ([]ContentItem, error) {
	return a.provider.FetchContent(ctx, collectionName, nil)
}

// GenerateRuntimeFetcher generates Go code for fetching data at runtime.
// The simple adapter returns an error as it does not support code generation.
//
// Returns *RuntimeFetcherCode which is always nil for this adapter.
// Returns error when called, as code generation is not supported.
func (*SimpleProviderAdapter) GenerateRuntimeFetcher(
	_ context.Context,
	_ string,
	_ ast.Expr,
	_ FetchOptions,
) (*RuntimeFetcherCode, error) {
	return nil, ErrCodeGenerationNotSupported
}

// ComputeETag computes a content fingerprint for hybrid mode.
//
// Takes collectionName (string) which identifies the collection to fingerprint.
//
// Returns string which is the computed ETag value.
// Returns error when the fingerprint cannot be computed.
func (a *SimpleProviderAdapter) ComputeETag(ctx context.Context, collectionName string, _ collection_dto.ContentSource) (string, error) {
	return a.provider.ComputeETag(ctx, collectionName)
}

// ValidateETag checks if the current content matches an expected ETag.
//
// Takes collectionName (string) which identifies the collection to
// validate.
// Takes expectedETag (string) which is the ETag value to compare
// against the current content.
//
// Returns currentETag (string) which is the current ETag of the
// collection content.
// Returns changed (bool) which is true if the content has changed
// since the expected ETag was generated.
// Returns err (error) when the validation cannot be performed.
func (a *SimpleProviderAdapter) ValidateETag(ctx context.Context, collectionName string, expectedETag string, _ collection_dto.ContentSource) (currentETag string, changed bool, err error) {
	return a.provider.ValidateETag(ctx, collectionName, expectedETag)
}

// GenerateRevalidator generates Go code for runtime ETag validation.
// The simple adapter returns an error as it doesn't support code generation.
//
// Returns *RuntimeFetcherCode which is always nil for this adapter.
// Returns error when called, as code generation is not supported.
func (*SimpleProviderAdapter) GenerateRevalidator(
	_ context.Context,
	_ string,
	_ ast.Expr,
	_ HybridConfig,
) (*RuntimeFetcherCode, error) {
	return nil, ErrCodeGenerationNotSupported
}

// ApplyFilterGroup checks if a content item matches a filter group.
//
// When the group is nil or has no filters, returns true.
//
// Takes item (*ContentItem) which is the content item to check.
// Takes group (*FilterGroup) which defines the filters to apply.
//
// Returns bool which is true if the item matches the filter group.
func ApplyFilterGroup(item *ContentItem, group *FilterGroup) bool {
	return collection_dto.ApplyFilterGroup(item, group)
}

// SortItems sorts a slice of content items based on the given sort options.
//
// This sorts the slice in place.
//
// Takes items ([]*ContentItem) which is the slice to sort.
// Takes sortOptions ([]SortOption) which specifies the sorting criteria.
func SortItems(items []*ContentItem, sortOptions []SortOption) {
	collection_dto.SortItems(items, sortOptions)
}

// PaginateItems applies pagination to a slice of items.
//
// When pagination is nil, returns the original slice unchanged.
//
// Takes items ([]*ContentItem) which is the slice to paginate.
// Takes pagination (*PaginationOptions) which specifies offset and limit.
//
// Returns []*ContentItem which is the paginated subset of items.
func PaginateItems(items []*ContentItem, pagination *PaginationOptions) []*ContentItem {
	return collection_dto.PaginateItems(items, pagination)
}

// CalculatePaginationMeta calculates pagination metadata.
//
// Takes total (int) which is the total number of items.
// Takes pagination (*PaginationOptions) which specifies the page size and
// offset.
//
// Returns *PaginationMeta which contains the calculated pagination details.
func CalculatePaginationMeta(total int, pagination *PaginationOptions) *PaginationMeta {
	return collection_dto.CalculatePaginationMeta(total, pagination)
}

// DefaultHybridConfig returns the default settings for hybrid caching.
//
// Returns HybridConfig which contains the default hybrid caching settings.
func DefaultHybridConfig() HybridConfig {
	return collection_dto.DefaultHybridConfig()
}

// DefaultRetryConfig returns a sensible default retry configuration.
//
// Returns *RetryConfig which contains reasonable defaults for retry behaviour.
func DefaultRetryConfig() *RetryConfig {
	return collection_dto.DefaultRetryConfig()
}
