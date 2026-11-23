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
	// NoLimit asks a search or query to return every match rather than a page of them.
	NoLimit = -1
)

const (
	// FieldTypeText supports full-text search with tokenization.
	FieldTypeText FieldType = iota + 1

	// FieldTypeTag supports exact match filtering without tokenization.
	FieldTypeTag

	// FieldTypeNumeric supports range queries and numeric sorting.
	FieldTypeNumeric

	// FieldTypeGeo supports geographic queries based on coordinates.
	FieldTypeGeo

	// FieldTypeVector supports vector similarity search using HNSW indexing.
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
	Name string

	// DistanceMetric is the VECTOR metric, defaulting to "cosine".
	DistanceMetric string

	// Type specifies how the field should be searchable.
	Type FieldType

	// Dimension is the required vector length for FieldTypeVector.
	Dimension int

	// Sortable enables sorting on the field.
	Sortable bool

	// Weight is the TEXT relevance multiplier, defaulting to 1.0.
	Weight float64
}

// TextAnalyseFunc transforms text into terms for indexing and queries.
//
// Implementations must support concurrent calls. A nil function uses default
// tokenisation.
type TextAnalyseFunc func(text string) []string

// SearchSchema defines which fields of a cached value type are searchable.
type SearchSchema struct {
	// TextAnalyser replaces default tokenisation when supplied.
	TextAnalyser TextAnalyseFunc

	// Language is the TEXT stemming language, defaulting to "english".
	Language string

	// Fields lists all searchable fields in the cached value type.
	Fields []FieldSchema

	// StopWords defines words to ignore during text search.
	StopWords []string

	// MaxTagsPerKey caps tags per key, with zero allowing unlimited tags and excess tags
	// dropped.
	MaxTagsPerKey int

	// MaxInvertedIndexTokens caps unique terms, with zero allowing unlimited terms.
	MaxInvertedIndexTokens int

	// MaxVectors caps new vectors without limiting updates, with zero disabling the cap.
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

	// Values contains multiple values for In (set membership) and Between (min, max range)
	// operations.
	Values []any

	// Operation is the comparison operation to apply.
	Operation FilterOp
}

// SearchOptions configures a full-text search operation.
//
// A search with both a text query and a vector is hybrid. The in-memory provider ranks
// the union of the text and vector matches with Reciprocal Rank Fusion, while Redis and
// Valkey rank only the vector neighbours that also match the text query.
type SearchOptions struct {
	// MinScore is the optional minimum similarity score.
	MinScore *float32

	// SortBy is the field to sort results by.
	SortBy string

	// VectorField selects a vector field, defaulting to the only VECTOR field when
	// unambiguous.
	VectorField string

	// Filters are additional conditions to apply after text matching.
	Filters []Filter

	// Fields selects TEXT fields, with nil searching all TEXT fields.
	Fields []string

	// Vector is the query vector for similarity search.
	Vector []float32

	// Limit is the maximum number of results to return.
	Limit int

	// Offset is the starting position for pagination.
	Offset int

	// TopK is the KNN candidate count, defaulting to Limit when zero.
	TopK int

	// SortOrder is the sort direction, defaulting to SortAsc.
	SortOrder SortOrder

	// Highlight enables highlighting of matched terms in results.
	Highlight bool
}

// VectorWindow resolves the KNN candidate count and returned page size.
//
// Takes defaultSize (int) which is the page size when Limit is neither positive nor
// NoLimit.
//
// Returns candidates (int) which is a positive TopK, or else enough candidates to fill
// the requested page after Offset.
// Returns pageSize (int) which is the page size, at most candidates.
func (o *SearchOptions) VectorWindow(defaultSize int) (candidates, pageSize int) {
	return resolveVectorWindow(o.TopK, o.Limit, o.Offset, defaultSize)
}

// QueryOptions configures a structured query operation without full-text search.
type QueryOptions struct {
	// MinScore is the optional minimum similarity score.
	MinScore *float32

	// SortBy is the field to sort results by.
	SortBy string

	// VectorField selects a vector field, defaulting to the only VECTOR field when
	// unambiguous.
	VectorField string

	// Filters are the conditions to match.
	Filters []Filter

	// Vector is the query vector for similarity search.
	Vector []float32

	// Limit is the maximum number of results to return.
	Limit int

	// Offset is the starting position for pagination.
	Offset int

	// TopK is the KNN candidate count, defaulting to Limit when zero.
	TopK int

	// SortOrder is the sort direction, defaulting to SortAsc.
	SortOrder SortOrder
}

// VectorWindow resolves the KNN candidate count and returned page size.
//
// Takes defaultSize (int) which is the page size when Limit is neither positive nor
// NoLimit.
//
// Returns candidates (int) which is a positive TopK, or else enough candidates to fill
// the requested page after Offset.
// Returns pageSize (int) which is the page size, at most candidates.
func (o *QueryOptions) VectorWindow(defaultSize int) (candidates, pageSize int) {
	return resolveVectorWindow(o.TopK, o.Limit, o.Offset, defaultSize)
}

// SearchResult contains the results of a search or query operation.
type SearchResult[K comparable, V any] struct {
	// Items contains the matched cache entries with metadata.
	Items []SearchHit[K, V]

	// Total is the total number of matches in the cache.
	//
	// For a vector search it counts the ranked candidates, so it never exceeds the candidate
	// count. In-memory providers count only candidates that meet MinScore and the filters.
	// Redis and Valkey apply MinScore to the returned page after the server has ranked the
	// candidates, so their total also counts candidates below MinScore and a page can hold
	// fewer than Limit items.
	Total int64

	// Offset is the starting position that was used.
	Offset int

	// Limit is the page size that was applied after resolving defaults.
	Limit int

	// Candidates counts ranked vector candidates, or zero for searches without a vector.
	//
	// The in-memory provider counts candidates at or above MinScore. Redis and Valkey infer
	// the cutoff from the returned page, so the count can include candidates below MinScore
	// outside that page.
	Candidates int
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
	Highlights map[string]string

	// Score is the relevance score, or zero for structured queries without text search.
	Score float64
}

// TextField creates a FieldSchema for full-text search.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for text search with default weight of 1.0.
func TextField(name string) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeText,
		Weight:         1.0,
		DistanceMetric: "",
		Dimension:      0,
		Sortable:       false,
	}
}

// TagField creates a FieldSchema for exact match filtering.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for tag matching.
func TagField(name string) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeTag,
		DistanceMetric: "",
		Dimension:      0,
		Sortable:       false,
		Weight:         0,
	}
}

// NumericField creates a FieldSchema for range queries.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for numeric comparisons.
func NumericField(name string) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeNumeric,
		DistanceMetric: "",
		Dimension:      0,
		Sortable:       false,
		Weight:         0,
	}
}

// SortableNumericField creates a sortable FieldSchema for range queries.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for numeric comparisons and sorting.
func SortableNumericField(name string) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeNumeric,
		Sortable:       true,
		DistanceMetric: "",
		Dimension:      0,
		Weight:         0,
	}
}

// SortableTextField creates a sortable FieldSchema for full-text search.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for text search and sorting.
func SortableTextField(name string) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeText,
		Sortable:       true,
		Weight:         1.0,
		DistanceMetric: "",
		Dimension:      0,
	}
}

// GeoField creates a FieldSchema for geographic queries.
//
// Takes name (string) which is the field path (supports dot notation).
//
// Returns FieldSchema configured for geo operations.
func GeoField(name string) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeGeo,
		DistanceMetric: "",
		Dimension:      0,
		Sortable:       false,
		Weight:         0,
	}
}

// VectorField creates a FieldSchema for vector similarity search with the default cosine
// distance metric.
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
		Sortable:       false,
		Weight:         0,
	}
}

// VectorFieldWithMetric creates a FieldSchema for vector similarity search with a custom
// distance metric.
//
// Takes name (string) which is the field path (supports dot notation).
// Takes dimension (int) which is the vector dimensionality.
// Takes metric (string) which is the distance metric, one of "cosine", "euclidean", or
// "dot_product".
//
// Returns FieldSchema configured for vector similarity search.
func VectorFieldWithMetric(name string, dimension int, metric string) FieldSchema {
	return FieldSchema{
		Name:           name,
		Type:           FieldTypeVector,
		Dimension:      dimension,
		DistanceMetric: metric,
		Sortable:       false,
		Weight:         0,
	}
}

// NewSearchSchema creates a SearchSchema with the given fields.
//
// Takes fields (...FieldSchema) which define the searchable fields.
//
// Returns *SearchSchema with english language and default stop words.
func NewSearchSchema(fields ...FieldSchema) *SearchSchema {
	return &SearchSchema{
		Fields:                 fields,
		Language:               "english",
		TextAnalyser:           nil,
		StopWords:              nil,
		MaxTagsPerKey:          0,
		MaxInvertedIndexTokens: 0,
		MaxVectors:             0,
	}
}

// NewSearchSchemaWithAnalyser creates a SearchSchema with a text analyser for linguistic
// processing of TEXT fields.
//
// Takes analyser (TextAnalyseFunc) which processes text into index terms.
// Takes fields (...FieldSchema) which define the searchable fields.
//
// Returns *SearchSchema with the analyser configured and english as the default language.
func NewSearchSchemaWithAnalyser(analyser TextAnalyseFunc, fields ...FieldSchema) *SearchSchema {
	return &SearchSchema{
		Fields:                 fields,
		Language:               "english",
		TextAnalyser:           analyser,
		StopWords:              nil,
		MaxTagsPerKey:          0,
		MaxInvertedIndexTokens: 0,
		MaxVectors:             0,
	}
}

// Eq creates an equality filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the value to match.
//
// Returns Filter configured for equality comparison.
func Eq(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpEq, Value: value, Values: nil}
}

// Ne creates a not-equal filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the value to exclude.
//
// Returns Filter configured for inequality comparison.
func Ne(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpNe, Value: value, Values: nil}
}

// Gt creates a greater-than filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the threshold value.
//
// Returns Filter configured for greater-than comparison.
func Gt(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpGt, Value: value, Values: nil}
}

// Ge creates a greater-than-or-equal filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the threshold value.
//
// Returns Filter configured for greater-than-or-equal comparison.
func Ge(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpGe, Value: value, Values: nil}
}

// Lt creates a less-than filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the threshold value.
//
// Returns Filter configured for less-than comparison.
func Lt(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpLt, Value: value, Values: nil}
}

// Le creates a less-than-or-equal filter.
//
// Takes field (string) which is the field name to filter on.
// Takes value (any) which is the threshold value.
//
// Returns Filter configured for less-than-or-equal comparison.
func Le(field string, value any) Filter {
	return Filter{Field: field, Operation: FilterOpLe, Value: value, Values: nil}
}

// In creates a set membership filter.
//
// Takes field (string) which is the field name to filter on.
// Takes values (...any) which are the allowed values.
//
// Returns Filter that matches if field value is in the set.
func In(field string, values ...any) Filter {
	return Filter{Field: field, Operation: FilterOpIn, Values: values, Value: nil}
}

// Between creates a range filter (inclusive).
//
// Takes field (string) which is the field name to filter on.
// Takes lower (any) which is the lower bound.
// Takes upper (any) which is the upper bound.
//
// Returns Filter that matches if field is within [lower, upper].
func Between(field string, lower, upper any) Filter {
	return Filter{Field: field, Operation: FilterOpBetween, Values: []any{lower, upper}, Value: nil}
}

// Prefix creates a prefix match filter for TAG fields.
//
// Takes field (string) which is the field name to filter on.
// Takes prefix (string) which is the prefix to match.
//
// Returns Filter that matches if field starts with prefix.
func Prefix(field string, prefix string) Filter {
	return Filter{Field: field, Operation: FilterOpPrefix, Value: prefix, Values: nil}
}

// resolveVectorWindow resolves the KNN candidate count and page size.
//
// Takes topK (int) which is an explicit candidate count, or zero.
// Takes limit (int) which is the page size, zero for defaultSize, or NoLimit for every
// ranked candidate.
// Takes offset (int) which is the number of ranked results the page skips.
// Takes defaultSize (int) which is the page size when limit is neither positive nor
// NoLimit.
//
// Returns candidates (int) which is topK when positive, otherwise offset plus the page
// size, so a later page is not empty because too few candidates were ranked.
// Returns pageSize (int) which is the page size, at most candidates.
func resolveVectorWindow(topK, limit, offset, defaultSize int) (candidates, pageSize int) {
	pageSize = limit
	if pageSize <= 0 {
		pageSize = defaultSize
	}
	candidates = topK
	if candidates <= 0 {
		candidates = max(offset, 0) + pageSize
	}
	if limit == NoLimit {
		pageSize = candidates
	}
	return candidates, min(pageSize, candidates)
}
