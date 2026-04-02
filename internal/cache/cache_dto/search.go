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

// FieldType defines the type of a searchable field in a cache.
type FieldType int

const (
	// FieldTypeText supports full-text search with tokenization.
	// Values are split into words and indexed for partial matching.
	FieldTypeText FieldType = iota + 1

	// FieldTypeTag supports exact match filtering without tokenization.
	// Values are indexed as-is for precise equality matching.
	FieldTypeTag

	// FieldTypeNumeric supports range queries and numeric sorting.
	// Values must be numeric (int, float, etc.) for comparison operations.
	FieldTypeNumeric

	// FieldTypeGeo supports geographic queries based on coordinates.
	// Values should be lat/long pairs for distance-based operations.
	FieldTypeGeo

	// FieldTypeVector supports vector similarity search using HNSW indexing.
	// Values must be []float32 vectors of the dimension specified in FieldSchema.
	FieldTypeVector
)

const (
	// SortAsc sorts results in ascending order (A-Z, 0-9).
	SortAsc SortOrder = iota

	// SortDesc sorts results in descending order (Z-A, 9-0).
	SortDesc
)

const (
	// FilterOpEq matches when the field equals the value.
	FilterOpEq FilterOp = iota

	// FilterOpNe matches when the field does not equal the value.
	FilterOpNe

	// FilterOpGt matches when the field is greater than the value.
	FilterOpGt

	// FilterOpGe matches when the field is greater than or equal to the value.
	FilterOpGe

	// FilterOpLt matches when the field is less than the value.
	FilterOpLt

	// FilterOpLe matches when the field is less than or equal to the value.
	FilterOpLe

	// FilterOpIn matches when the field value is in the provided set.
	FilterOpIn

	// FilterOpBetween matches when the field is within a range (inclusive).
	FilterOpBetween

	// FilterOpPrefix matches when the field starts with the value (TAG fields).
	FilterOpPrefix
)

// FieldSchema defines a single searchable field in a cached value type.
type FieldSchema struct {
	// Name is the field path in the cached value.
	// Supports dot notation for nested fields (e.g., "address.city").
	Name string

	// DistanceMetric specifies the similarity metric for VECTOR fields --
	// "cosine" (default), "euclidean", or "dot_product" -- and is ignored for
	// non-vector field types.
	DistanceMetric string

	// Type specifies how this field should be searchable.
	Type FieldType

	// Dimension is the vector dimensionality for VECTOR fields. Required when Type
	// is FieldTypeVector; ignored for other field types.
	Dimension int

	// Sortable enables sorting on this field.
	// Has memory cost as the provider maintains sorted structures.
	Sortable bool

	// Weight is the relevance score multiplier for TEXT fields.
	// Higher values increase search ranking impact; default is 1.0.
	Weight float64
}

// TextAnalyseFunc transforms text into index terms for search indexing. The
// same function must be used for indexing and querying; implementations must
// be concurrent-safe, and when nil, providers use default tokenisation.
type TextAnalyseFunc func(text string) []string

// SearchSchema defines which fields of a cached value type are searchable.
// This schema enables providers to build appropriate internal structures
// for efficient search and query operations.
type SearchSchema struct {
	// TextAnalyser is an optional function for linguistic text processing that
	// replaces the default tokenisation. Providers with native search may ignore it.
	TextAnalyser TextAnalyseFunc

	// Language specifies the stemming language for TEXT fields; affects how words
	// are normalised (e.g., "running" becomes "run"). Default is "english".
	Language string

	// Fields lists all searchable fields in the cached value type.
	Fields []FieldSchema

	// StopWords defines words to ignore during text search.
	// If nil, provider defaults are used.
	StopWords []string

	// MaxTagsPerKey limits the number of tags per cache key. Zero means
	// unlimited; excess tags are silently dropped.
	MaxTagsPerKey int

	// MaxInvertedIndexTokens limits unique terms in the inverted index; when the
	// limit is reached, new terms are silently ignored. Zero means unlimited.
	MaxInvertedIndexTokens int

	// MaxVectors limits the maximum number of vectors per vector index, after
	// which Add calls for new keys are silently ignored (updates to existing
	// keys still work) and zero means unlimited.
	MaxVectors int
}

// SortOrder specifies the direction of result sorting.
type SortOrder int

// FilterOp defines filter comparison operations.
type FilterOp int

// Filter represents a single filter condition for structured queries.
type Filter struct {
	// Value is the comparison value for single-value operations.
	Value any

	// Field is the name of the field to filter on.
	Field string

	// Values contains multiple values for In (set membership) and
	// Between (min, max range) operations.
	Values []any

	// Operation is the comparison operation to apply.
	Operation FilterOp
}

// SearchOptions configures a full-text search operation.
type SearchOptions struct {
	// MinScore filters out results below this similarity threshold.
	// If nil, no minimum score filter is applied.
	MinScore *float32

	// SortBy is the field to sort results by.
	// If empty, results are sorted by relevance score.
	SortBy string

	// VectorField specifies which vector field to search.
	// If empty and only one vector field exists in the schema, it is used
	// automatically.
	VectorField string

	// Filters are additional conditions to apply after text matching.
	// All filters must match (AND logic).
	Filters []Filter

	// Fields limits which TEXT fields to search.
	// If nil, all TEXT fields in the schema are searched.
	Fields []string

	// Vector is the query vector for similarity search.
	// When set, the search performs vector similarity matching.
	Vector []float32

	// Limit is the maximum number of results to return.
	// Default is 10, maximum depends on provider.
	Limit int

	// Offset is the starting position for pagination.
	// Use with Limit for paging through results.
	Offset int

	// TopK is the maximum number of nearest neighbours for vector search.
	// If zero, defaults to Limit.
	TopK int

	// SortOrder specifies ascending or descending sort.
	// Default is SortAsc.
	SortOrder SortOrder

	// Highlight enables highlighting of matched terms in results.
	// When true, SearchHit.Highlights contains snippets.
	Highlight bool
}

// QueryOptions configures a structured query operation without full-text search.
type QueryOptions struct {
	// MinScore filters out results below this similarity threshold.
	// If nil, no minimum score filter is applied.
	MinScore *float32

	// SortBy is the field to sort results by.
	// Must be a sortable field in the schema.
	SortBy string

	// VectorField specifies which vector field to search.
	// If empty and only one vector field exists in the schema, it is used
	// automatically.
	VectorField string

	// Filters are the conditions to match.
	// All filters must match (AND logic).
	Filters []Filter

	// Vector is the query vector for similarity search.
	// When set, the query performs vector similarity matching.
	Vector []float32

	// Limit is the maximum number of results to return.
	// Default is 10, maximum depends on provider.
	Limit int

	// Offset is the starting position for pagination.
	// Use with Limit for paging through results.
	Offset int

	// TopK is the maximum number of nearest neighbours for vector search.
	// If zero, defaults to Limit.
	TopK int

	// SortOrder specifies ascending or descending sort.
	// Default is SortAsc.
	SortOrder SortOrder
}

// SearchResult contains the results of a search or query operation.
type SearchResult[K comparable, V any] struct {
	// Items contains the matched cache entries with metadata.
	Items []SearchHit[K, V]

	// Total is the total number of matches in the cache.
	// May exceed len(Items) when pagination is used.
	Total int64

	// Offset is the starting position that was used.
	Offset int

	// Limit is the maximum results that were requested.
	Limit int
}

// Keys returns all keys from the search results.
//
// Returns []K containing just the keys from all hits.
func (r SearchResult[K, V]) Keys() []K {
	keys := make([]K, len(r.Items))
	for i, item := range r.Items {
		keys[i] = item.Key
	}
	return keys
}

// Values returns all values from the search results.
//
// Returns []V containing just the values from all hits.
func (r SearchResult[K, V]) Values() []V {
	values := make([]V, len(r.Items))
	for i, item := range r.Items {
		values[i] = item.Value
	}
	return values
}

// IsEmpty returns true if the result contains no items.
//
// Returns bool which is true when Items is empty.
func (r SearchResult[K, V]) IsEmpty() bool {
	return len(r.Items) == 0
}

// HasMore returns true if there are more results beyond this page.
//
// Returns bool which is true when Total > Offset + len(Items).
func (r SearchResult[K, V]) HasMore() bool {
	return r.Total > int64(r.Offset+len(r.Items))
}

// SearchHit represents a single result from a search or query operation.
type SearchHit[K comparable, V any] struct {
	// Key is the cache key of the matched entry.
	Key K

	// Value is the cached value.
	Value V

	// Highlights maps field names to snippets with matched terms highlighted.
	// Only populated when SearchOptions.Highlight is true.
	Highlights map[string]string

	// Score is the relevance score for full-text search; higher values indicate
	// better matches. For structured queries without text search, this is 0.
	Score float64
}

// TextField creates a FieldSchema for full-text search.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for text search with default weight of 1.0.
func TextField(name string) FieldSchema {
	return FieldSchema{
		Name:   name,
		Type:   FieldTypeText,
		Weight: 1.0,
	}
}

// TagField creates a FieldSchema for exact match filtering.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for tag matching.
func TagField(name string) FieldSchema {
	return FieldSchema{
		Name: name,
		Type: FieldTypeTag,
	}
}

// NumericField creates a FieldSchema for range queries.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for numeric comparisons.
func NumericField(name string) FieldSchema {
	return FieldSchema{
		Name: name,
		Type: FieldTypeNumeric,
	}
}

// SortableNumericField creates a sortable FieldSchema for range queries.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for numeric comparisons and sorting.
func SortableNumericField(name string) FieldSchema {
	return FieldSchema{
		Name:     name,
		Type:     FieldTypeNumeric,
		Sortable: true,
	}
}

// SortableTextField creates a sortable FieldSchema for full-text search.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for text search and sorting.
func SortableTextField(name string) FieldSchema {
	return FieldSchema{
		Name:     name,
		Type:     FieldTypeText,
		Sortable: true,
		Weight:   1.0,
	}
}

// GeoField creates a FieldSchema for geographic queries.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for geo operations.
func GeoField(name string) FieldSchema {
	return FieldSchema{
		Name: name,
		Type: FieldTypeGeo,
	}
}

// VectorField creates a FieldSchema for vector similarity search with the
// default cosine distance metric.
//
// Takes name (string) which is the field path (supports dot notation).
// Takes dimension (int) which is the vector dimensionality.
//
// Returns FieldSchema configured for vector similarity search.
func VectorField(name string, dimension int) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeVector,
		Dimension:      dimension,
		DistanceMetric: "cosine",
	}
}

// VectorFieldWithMetric creates a FieldSchema for vector similarity search
// with a custom distance metric.
//
// Takes name (string) which is the field path (supports dot notation).
// Takes dimension (int) which is the vector dimensionality.
// Takes metric (string) which is the distance metric: "cosine", "euclidean",
// or "dot_product".
//
// Returns FieldSchema configured for vector similarity search.
func VectorFieldWithMetric(name string, dimension int, metric string) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeVector,
		Dimension:      dimension,
		DistanceMetric: metric,
	}
}

// NewSearchSchema creates a SearchSchema with the given fields.
//
// Takes fields (...FieldSchema) which define the searchable fields.
//
// Returns *SearchSchema with english language and default stop words.
func NewSearchSchema(fields ...FieldSchema) *SearchSchema {
	return &SearchSchema{
		Fields:   fields,
		Language: "english",
	}
}

// NewSearchSchemaWithAnalyser creates a SearchSchema with a text analyser for
// linguistic processing of TEXT fields. The analyser replaces the provider's
// default tokenisation, enabling stemming, normalisation, stop words, and
// other NLP features.
//
// Takes analyser (TextAnalyseFunc) which processes text into index terms.
// Takes fields (...FieldSchema) which define the searchable fields.
//
// Returns *SearchSchema with the analyser configured and english as the
// default language.
func NewSearchSchemaWithAnalyser(analyser TextAnalyseFunc, fields ...FieldSchema) *SearchSchema {
	return &SearchSchema{
		Fields:       fields,
		Language:     "english",
		TextAnalyser: analyser,
	}
}

// Eq creates an equality filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the value to match.
//
// Returns Filter configured for equality comparison.
func Eq(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpEq, Value: value}
}

// Ne creates a not-equal filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the value to exclude.
//
// Returns Filter configured for inequality comparison.
func Ne(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpNe, Value: value}
}

// Gt creates a greater-than filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the threshold value.
//
// Returns Filter configured for greater-than comparison.
func Gt(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpGt, Value: value}
}

// Ge creates a greater-than-or-equal filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the threshold value.
//
// Returns Filter configured for greater-than-or-equal comparison.
func Ge(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpGe, Value: value}
}

// Lt creates a less-than filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the threshold value.
//
// Returns Filter configured for less-than comparison.
func Lt(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpLt, Value: value}
}

// Le creates a less-than-or-equal filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the threshold value.
//
// Returns Filter configured for less-than-or-equal comparison.
func Le(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpLe, Value: value}
}

// In creates a set membership filter.
//
// Takes field (string) which is the field name to filter on.
// Takes values (...any) which are the allowed values.
//
// Returns Filter that matches if field value is in the set.
func In(field string, values ...any) Filter {
	return Filter{Field: field, Operation: FilterOpIn, Values: values}
}

// Between creates a range filter (inclusive).
//
// Takes field (string) which is the field name to filter on.
// Takes lower (any) which is the lower bound.
// Takes upper (any) which is the upper bound.
//
// Returns Filter that matches if field is within [lower, upper].
func Between(field string, lower, upper any) Filter {
	return Filter{Field: field, Operation: FilterOpBetween, Values: []any{lower, upper}}
}

// Prefix creates a prefix match filter for TAG fields.
//
// Takes field (string) which is the field name to filter on.
// Takes prefix (string) which is the prefix to match.
//
// Returns Filter that matches if field starts with prefix.
func Prefix(field string, prefix string) Filter {
	return Filter{Field: field, Operation: FilterOpPrefix, Value: prefix}
}
