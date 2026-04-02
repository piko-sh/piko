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

package piko

import (
	"context"

	"piko.sh/piko/wdk/runtime"
)

const (
	// FilterOpEquals is the equality comparison operator for filters.
	FilterOpEquals = runtime.FilterOpEquals

	// FilterOpNotEquals is the filter operation for inequality comparisons.
	FilterOpNotEquals = runtime.FilterOpNotEquals

	// FilterOpGreaterThan is the greater than comparison operator for filters.
	FilterOpGreaterThan = runtime.FilterOpGreaterThan

	// FilterOpGreaterEqual is the greater-than-or-equal filter operation.
	FilterOpGreaterEqual = runtime.FilterOpGreaterEqual

	// FilterOpLessThan is the filter operation for less than comparisons.
	FilterOpLessThan = runtime.FilterOpLessThan

	// FilterOpLessEqual is the filter operator for less than or equal comparison.
	FilterOpLessEqual = runtime.FilterOpLessEqual

	// FilterOpContains is the filter operation for substring matching.
	FilterOpContains = runtime.FilterOpContains

	// FilterOpStartsWith is the filter operation for prefix matching.
	FilterOpStartsWith = runtime.FilterOpStartsWith

	// FilterOpEndsWith is the filter operation for suffix matching.
	FilterOpEndsWith = runtime.FilterOpEndsWith

	// FilterOpIn is the filter operator for membership testing.
	FilterOpIn = runtime.FilterOpIn

	// FilterOpNotIn is the filter operation for excluding matching values.
	FilterOpNotIn = runtime.FilterOpNotIn

	// FilterOpExists is the filter operation that checks for field existence.
	FilterOpExists = runtime.FilterOpExists

	// FilterOpFuzzyMatch is a filter operation for approximate string matching.
	FilterOpFuzzyMatch = runtime.FilterOpFuzzyMatch

	// SortAsc is the ascending sort order.
	SortAsc = runtime.SortAsc

	// SortDesc indicates descending sort order.
	SortDesc = runtime.SortDesc
)

// These types are used for working with collections, filtering, and sorting.

type (
	// Section represents a heading in markdown content (h2-h6).
	// Used for building Table of Contents (flat list).
	Section = runtime.Section

	// SectionNode represents a hierarchical section entry for table of contents.
	// Unlike Section (which is flat), SectionNode contains nested children
	// for building tree-structured navigation.
	SectionNode = runtime.SectionNode

	// SectionTreeOption is a functional option for configuring GetSectionsTree.
	SectionTreeOption = runtime.SectionTreeOption

	// Filter represents a single filtering condition for collection queries.
	Filter = runtime.Filter

	// FilterGroup represents multiple filters combined with AND/OR logic.
	FilterGroup = runtime.FilterGroup

	// FilterOperator defines comparison operations (eq, contains, gt, etc).
	FilterOperator = runtime.FilterOperator

	// SortOrder defines sort direction (ascending or descending).
	SortOrder = runtime.SortOrder

	// SortOption specifies a field and direction for sorting.
	SortOption = runtime.SortOption

	// PaginationOptions specifies offset and limit for pagination.
	PaginationOptions = runtime.PaginationOptions

	// SearchResult represents a search result with relevance scoring.
	SearchResult[T any] = runtime.SearchResult[T]

	// SearchField specifies a field to search with optional weighting.
	SearchField = runtime.SearchField

	// SearchOption is a functional option for configuring search.
	SearchOption = runtime.SearchOption
)

// These types and functions enable hierarchical navigation generation from
// collections. Navigation is automatically derived from frontmatter metadata in
// your content files.

type (
	// NavigationGroups contains multiple named navigation structures. Each group
	// represents a distinct navigation UI such as sidebar, footer, or breadcrumb.
	NavigationGroups = runtime.NavigationGroups

	// NavigationTree represents a hierarchical navigation structure for a specific
	// locale.
	NavigationTree = runtime.NavigationTree

	// NavigationNode represents a single node in the navigation hierarchy.
	// Nodes can be categories (grouping) or content pages (with URLs).
	NavigationNode = runtime.NavigationNode

	// NavigationConfig controls how navigation trees are built.
	NavigationConfig = runtime.NavigationConfig
)

// GetData retrieves the page data from CollectionData and converts it to type
// T. This provides type-safe access to collection data in Render functions.
//
// The function extracts page data from the collection root map and performs
// automatic type conversion using JSON marshalling/unmarshalling, which
// handles:
//   - Field name matching (case-sensitive)
//   - Type coercion (int to float, etc.)
//   - Nested structure conversion
//   - Missing fields (set to zero values)
//
// Takes r (*RequestData) which contains the CollectionData to extract from.
//
// Returns T which is the page data converted to the requested type, or the zero
// value if conversion fails.
func GetData[T any](r *RequestData) T {
	return runtime.GetData[T](r)
}

// GetSections extracts the table of contents sections from collection data.
// Returns a list of headings found in markdown content, useful for building
// a ToC sidebar.
//
// Takes r (*RequestData) which provides the collection data to extract from.
//
// Returns []Section which contains the headings found in the content.
//
// Example usage:
//
//	func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
//	    doc := piko.GetData[Doc](r)
//	    sections := piko.GetSections(r)
//	    return Response{
//	        Title:    doc.Title,
//	        Sections: sections,  // Pass to template for ToC rendering
//	    }, piko.Metadata{}, nil
//	}
func GetSections(r *RequestData) []Section {
	return runtime.GetSections(r)
}

// GetSectionsTree extracts sections from collection data and builds a
// hierarchical tree. Unlike GetSections which returns a flat list, this returns
// nested SectionNode structures suitable for rendering a table of contents with
// proper nesting.
//
// Takes r (*RequestData) which contains the collection data to extract from.
// Takes opts (...SectionTreeOption) which configures level filtering.
//
// Returns []SectionNode which contains top-level sections with nested children.
//
// Example usage:
//
//	func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
//	    // Get hierarchical sections (defaults: h2-h4)
//	    sections := piko.GetSectionsTree(r)
//	    // Or with custom level filtering:
//	    sections := piko.GetSectionsTree(r,
//	        piko.WithMinLevel(2),
//	        piko.WithMaxLevel(3),
//	    )
//	    return Response{Sections: sections}, piko.Metadata{}, nil
//	}
func GetSectionsTree(r *RequestData, opts ...SectionTreeOption) []SectionNode {
	return runtime.GetSectionsTree(r, opts...)
}

// WithMinLevel sets the minimum heading level to include (default: 2).
// Headings below this level are filtered out.
//
// Takes level (int) which specifies the minimum heading level.
//
// Returns SectionTreeOption which configures the section tree filtering.
//
// Example:
// tree := piko.GetSectionsTree(r, piko.WithMinLevel(2)) // Start from h2
func WithMinLevel(level int) SectionTreeOption {
	return runtime.WithMinLevel(level)
}

// WithMaxLevel sets the maximum heading level to include (default: 4).
// Headings above this level are filtered out.
//
// Takes level (int) which specifies the maximum heading level to include.
//
// Returns SectionTreeOption which configures the section tree builder.
//
// Example:
// tree := piko.GetSectionsTree(r, piko.WithMaxLevel(3)) // Only h2 and h3
func WithMaxLevel(level int) SectionTreeOption {
	return runtime.WithMaxLevel(level)
}

// DefaultNavigationConfig returns a NavigationConfig with sensible defaults.
// Use this when you do not need custom configuration.
//
// Returns NavigationConfig which provides ready-to-use navigation settings.
func DefaultNavigationConfig() NavigationConfig {
	return runtime.DefaultNavigationConfig()
}

// GetAllCollectionItems retrieves all items from a static collection.
//
// This function gets all items in a collection without their content ASTs,
// suitable for building navigation, sitemaps, or indexes.
//
// Takes collectionName (string) which specifies the collection to retrieve.
//
// Returns []map[string]any which contains metadata maps for all items.
// Returns error when the collection is not found.
func GetAllCollectionItems(collectionName string) ([]map[string]any, error) {
	return runtime.GetAllCollectionItems(collectionName)
}

// BuildNavigationFromMetadata constructs hierarchical navigation from
// collection metadata.
//
// This function takes metadata maps (from GetAllCollectionItems) and builds
// navigation trees based on the "Navigation" field in each item's metadata.
//
// The navigation metadata should be in this structure:
// metadata["Navigation"] = NavigationMetadata with Groups["sidebar"], etc.
//
// Takes items ([]map[string]interface{}) which is a slice of metadata maps
// from a collection.
// Takes config (NavigationConfig) which specifies navigation building
// options.
//
// Returns *NavigationGroups which contains all named navigation trees.
//
// Example usage in a template:
//
//	func Render(r *piko.RequestData) (Response, piko.Metadata, error) {
//	    // Get all docs metadata (lightweight - no ASTs)
//	    allDocuments, err := piko.GetAllCollectionItems("docs")
//	    if err != nil {
//	        return Response{}, piko.Metadata{}, err
//	    }
//	    // Build navigation
//	    navGroups := piko.BuildNavigationFromMetadata(allDocuments, piko.DefaultNavigationConfig())
//	    // Access navigation for different contexts
//	    sidebar := navGroups.Groups["sidebar"]
//	    footer := navGroups.Groups["footer"]
//	    return Response{
//	        Sidebar: sidebar,
//	        Footer:  footer,
//	    }, piko.Metadata{}, nil
//	}
func BuildNavigationFromMetadata(ctx context.Context, items []map[string]any, config NavigationConfig) *NavigationGroups {
	return runtime.BuildNavigationFromMetadata(ctx, items, config)
}

// SearchCollection performs fuzzy text search on collection data.
// Results are ranked by relevance score (0.0 - 1.0).
//
// This function searches both static (markdown) and dynamic (CMS) collections
// with configurable fuzzy matching.
//
// Takes r (*RequestData) which provides the request context.
// Takes collectionName (string) which identifies the collection to
// search.
// Takes query (string) which is the search term to match against
// collection items.
// Takes opts (...SearchOption) which provides optional search
// configuration such as fields, threshold, and limit.
//
// Returns []SearchResult[T] which contains matched items with relevance
// scores.
// Returns error when the search fails.
//
// Example usage:
//
//	func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
//	    query := r.QueryParams.Get("q")
//	    results := piko.SearchCollection[Post](r, "blog", query,
//	        piko.WithSearchFields(
//	            piko.SearchField{Name: "Title", Weight: 2.0},  // Title weighted 2x
//	            piko.SearchField{Name: "Body", Weight: 1.0},
//	        ),
//	        piko.WithFuzzyThreshold(0.3),
//	        piko.WithSearchLimit(20),
//	    )
//	    return Response{Results: results, Query: query}, piko.Metadata{}, nil
//	}
func SearchCollection[T any](r *RequestData, collectionName string, query string, opts ...SearchOption) ([]SearchResult[T], error) {
	return runtime.SearchCollection[T](r, collectionName, query, opts...)
}

// QuickSearch performs a simple fuzzy search and returns just the matched items
// without scores.
//
// This is a convenience wrapper around SearchCollection for simple use cases.
// It uses default search settings:
//   - Fuzzy threshold: 0.3
//   - Searches all string fields
//   - Returns top 10 results
//
// Example usage:
//
//	func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
//	    query := r.QueryParams.Get("q")
//	    results := piko.QuickSearch[Post](r, "blog", query)
//	    return Response{Results: results}, piko.Metadata{}, nil
//	}
//
// Takes r (*RequestData) which provides the request context.
// Takes collectionName (string) which specifies the collection to search.
// Takes query (string) which is the search term to match.
//
// Returns []T which contains the matched items.
// Returns error when the search fails.
func QuickSearch[T any](r *RequestData, collectionName string, query string) ([]T, error) {
	return runtime.QuickSearch[T](r, collectionName, query)
}

// NewFilter creates a single filter condition.
//
// Takes field (string) which specifies the field name to filter on.
// Takes operator (FilterOperator) which defines the comparison operation.
// Takes value (interface{}) which provides the value to compare against.
//
// Returns Filter which is the constructed filter condition.
//
// Example:
// filter := piko.NewFilter("status", piko.FilterOpEquals, "published")
func NewFilter(field string, operator FilterOperator, value any) Filter {
	return runtime.NewFilter(field, operator, value)
}

// And combines multiple filters with AND logic (all must match).
//
// Takes filters (...Filter) which specifies the filters to combine.
//
// Returns FilterGroup which contains the combined filters.
//
// Example:
// filters := piko.And(
//
//	piko.NewFilter("status", piko.FilterOpEquals, "published"),
//	piko.NewFilter("featured", piko.FilterOpEquals, true),
//
// )
func And(filters ...Filter) FilterGroup {
	return runtime.And(filters...)
}

// Or combines multiple filters with OR logic where at least one must match.
//
// Takes filters (...Filter) which are the conditions to combine.
//
// Returns FilterGroup which contains the combined filters.
//
// Example:
// filters := piko.Or(
//
//	piko.NewFilter("category", piko.FilterOpEquals, "tech"),
//	piko.NewFilter("category", piko.FilterOpEquals, "science"),
//
// )
func Or(filters ...Filter) FilterGroup {
	return runtime.Or(filters...)
}

// NewSortOption creates a sorting option.
//
// Takes field (string) which specifies the field name to sort by.
// Takes order (SortOrder) which sets the sort direction.
//
// Returns SortOption which is the configured sorting option.
//
// Example:
// sort := piko.NewSortOption("publishedAt", piko.SortDesc)
func NewSortOption(field string, order SortOrder) SortOption {
	return runtime.NewSortOption(field, order)
}

// NewPaginationOptions creates pagination parameters.
//
// Takes limit (int) which specifies the maximum number of items to return.
// Takes offset (int) which specifies the number of items to skip.
//
// Returns PaginationOptions which contains the configured pagination settings.
//
// Example:
// pagination := piko.NewPaginationOptions(10, 20)  // Limit 10, Offset 20
func NewPaginationOptions(limit, offset int) PaginationOptions {
	return runtime.NewPaginationOptions(limit, offset)
}

// WithSearchFields specifies which fields to search with their weights.
//
// Takes fields (...SearchField) which defines the searchable fields and their
// relative weights.
//
// Returns SearchOption which configures field-specific search behaviour.
//
// Example:
// results := piko.SearchCollection[Post](r, "blog", "golang",
//
//	piko.WithSearchFields(
//	    piko.SearchField{Name: "Title", Weight: 2.0},
//	    piko.SearchField{Name: "Body", Weight: 1.0},
//	))
func WithSearchFields(fields ...SearchField) SearchOption {
	return runtime.WithSearchFields(fields...)
}

// WithFuzzyThreshold sets the fuzzy matching threshold for search operations.
// Values range from 0.0 to 1.0, with 0.3 as the default.
//
// Takes threshold (float64) which specifies the matching tolerance level.
//
// Returns SearchOption which configures the search with the given threshold.
//
// Example:
// results := piko.SearchCollection[Post](r, "blog", "golang",
//
//	piko.WithFuzzyThreshold(0.5))  // More lenient matching
func WithFuzzyThreshold(threshold float64) SearchOption {
	return runtime.WithFuzzyThreshold(threshold)
}

// WithSearchLimit limits the number of search results.
//
// Takes limit (int) which specifies the maximum number of results to return.
//
// Returns SearchOption which configures the search limit.
//
// Example:
// results := piko.SearchCollection[Post](r, "blog", "golang",
//
//	piko.WithSearchLimit(20))
func WithSearchLimit(limit int) SearchOption {
	return runtime.WithSearchLimit(limit)
}

// WithSearchOffset skips the first N results for pagination.
//
// Takes offset (int) which specifies the number of results to skip.
//
// Returns SearchOption which configures the search to skip the specified
// number of results.
func WithSearchOffset(offset int) SearchOption {
	return runtime.WithSearchOffset(offset)
}

// WithMinScore filters out results below the specified score.
//
// Takes score (float64) which specifies the minimum relevance threshold.
//
// Returns SearchOption which configures the search to exclude low-scoring
// results.
//
// Example:
// results := piko.SearchCollection[Post](r, "blog", "golang",
//
//	piko.WithMinScore(0.5))  // Only highly relevant results
func WithMinScore(score float64) SearchOption {
	return runtime.WithMinScore(score)
}

// WithCaseSensitive enables case-sensitive search.
//
// Takes sensitive (bool) which controls whether matching is case-sensitive.
//
// Returns SearchOption which configures the search behaviour.
//
// Example:
// results := piko.SearchCollection[Post](r, "blog", "GoLang",
//
//	piko.WithCaseSensitive(true))
func WithCaseSensitive(sensitive bool) SearchOption {
	return runtime.WithCaseSensitive(sensitive)
}

// WithSearchMode sets the search mode to use.
//
// Valid values: "fast" (default) or "smart" (with stemming and phonetic
// matching).
//
// Fast mode uses basic tokenisation and exact matching, optimised for speed.
// Smart mode uses stemming, phonetic encoding, and fuzzy matching for better
// recall.
//
// Takes mode (string) which specifies the search mode ("fast" or "smart").
//
// Returns SearchOption which configures the search behaviour.
//
// Example:
// results := piko.SearchCollection[Post](r, "blog", "running",
//
//	piko.WithSearchMode("smart"))  // Matches "run", "runs", "running"
func WithSearchMode(mode string) SearchOption {
	return runtime.WithSearchMode(mode)
}
