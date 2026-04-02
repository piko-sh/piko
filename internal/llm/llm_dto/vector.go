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

package llm_dto

// SimilarityMetric specifies how to measure the distance between vectors in a
// similarity search.
type SimilarityMetric string

const (
	// SimilarityCosine uses cosine similarity for vector comparison. Values range
	// from -1 to 1, where 1 means the vectors point in the same direction.
	SimilarityCosine SimilarityMetric = "cosine"

	// SimilarityEuclidean uses Euclidean distance to compare vectors. The range is
	// [0, +inf) where 0 means the vectors are the same.
	SimilarityEuclidean SimilarityMetric = "euclidean"

	// SimilarityDotProduct is a similarity metric that uses dot product for vector
	// comparison. Higher values show greater similarity, with a range from negative
	// infinity to positive infinity.
	SimilarityDotProduct SimilarityMetric = "dot_product"
)

// VectorDocument represents a document stored in the vector store.
type VectorDocument struct {
	// Metadata holds key-value pairs linked to the document.
	Metadata map[string]any

	// ID is the unique identifier for this document.
	ID string

	// Content is the original text of the document.
	Content string

	// Vector is the embedding vector for this document.
	Vector []float32
}

// VectorSearchRequest holds the settings for a vector similarity search.
type VectorSearchRequest struct {
	// MinScore filters out results below this similarity threshold.
	// If nil, no minimum score filter is applied.
	MinScore *float32

	// Filter specifies metadata criteria that documents must match.
	Filter map[string]any

	// Namespace is the collection or index to search within.
	Namespace string

	// TextQuery is an optional text query for hybrid search. When set with
	// Vector, combines text and vector results using RRF; requires TEXT
	// fields in the cache search schema.
	TextQuery string

	// Vector is the query embedding used to find similar documents.
	Vector []float32

	// TopK is the maximum number of results to return.
	TopK int

	// IncludeVectors determines whether to return vectors in results.
	IncludeVectors bool

	// IncludeMetadata determines whether to return metadata in results.
	IncludeMetadata bool
}

// VectorSearchResult represents a single result from a vector similarity search.
type VectorSearchResult struct {
	// Metadata holds document metadata; only present if IncludeMetadata was true.
	Metadata map[string]any

	// ID is the unique identifier of the matching document.
	ID string

	// Content is the original text of the document.
	Content string

	// Vector is the embedding vector; included when IncludeVectors is true.
	Vector []float32

	// Score is the similarity score for this result.
	Score float32
}

// VectorSearchResponse contains the results of a vector similarity search.
type VectorSearchResponse struct {
	// Results holds the matching documents sorted by similarity.
	Results []VectorSearchResult

	// TotalCount is the total number of documents that matched before the TopK
	// limit was applied.
	TotalCount int
}

// FirstResult returns the first search result, or nil if there are no results.
// This is a convenience method for single-result queries.
//
// Returns *VectorSearchResult which is the first result, or nil if empty.
func (r *VectorSearchResponse) FirstResult() *VectorSearchResult {
	if len(r.Results) == 0 {
		return nil
	}
	return &r.Results[0]
}

// HasResults reports whether the search returned any results.
//
// Returns bool which is true if at least one result was found.
func (r *VectorSearchResponse) HasResults() bool {
	return len(r.Results) > 0
}
