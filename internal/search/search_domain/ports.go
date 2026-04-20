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

package search_domain

import (
	"context"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/search/search_dto"
	search_fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

// IndexBuilderPort defines the contract for building search indexes from
// collection items. Implementations generate FlatBuffer-encoded inverted
// indexes for either Fast or Smart mode.
type IndexBuilderPort interface {
	// BuildIndex analyses collection items and generates a search index.
	// The mode determines the level of linguistic analysis applied.
	//
	// Takes collectionName (string) which identifies the collection to index.
	// Takes items ([]collection_dto.ContentItem) which contains the content to
	// analyse.
	// Takes mode (search_fb.SearchMode) which sets the linguistic analysis level.
	// Takes config (IndexBuildConfig) which provides build settings.
	//
	// Returns []byte which is the serialised FlatBuffer binary ready for disk.
	// Returns error when indexing fails.
	BuildIndex(
		ctx context.Context,
		collectionName string,
		items []collection_dto.ContentItem,
		mode search_fb.SearchMode,
		config IndexBuildConfig,
	) ([]byte, error)
}

// IndexReaderPort defines the contract for reading and querying search indexes.
// Implementations provide zero-copy access to FlatBuffer-encoded index data.
type IndexReaderPort interface {
	// LoadIndex sets up the reader with a FlatBuffer binary blob.
	// The blob should come from //go:embed or a similar zero-copy source.
	//
	// Takes data ([]byte) which is the FlatBuffer binary data to load.
	//
	// Returns error when the data cannot be parsed.
	LoadIndex(data []byte) error

	// GetTermPostings returns the posting list for a given term.
	//
	// Takes term (string) which is the word to look up in the index.
	//
	// Returns []PostingInfo which contains the documents where the term appears.
	// Returns float64 which is the inverse document frequency of the term.
	// Returns error when the lookup fails. Returns nil for the posting list if
	// the term is not found in the index.
	GetTermPostings(term string) ([]PostingInfo, float64, error)

	// GetDocMetadata returns scoring metadata for a specific document.
	//
	// Takes documentID (uint32) which identifies the document.
	//
	// Returns DocMetadataInfo which contains the document's scoring metadata.
	// Returns error when the document cannot be found or read.
	GetDocMetadata(documentID uint32) (DocMetadataInfo, error)

	// GetCorpusStats returns global statistics needed for BM25 scoring.
	GetCorpusStats() CorpusStats

	// GetMode returns the search mode this index was built with.
	GetMode() search_fb.SearchMode

	// GetLanguage returns the language this index was built with.
	GetLanguage() string

	// FindPhoneticTerms returns all terms that match a phonetic code.
	// Only works with Smart mode indexes.
	//
	// Takes phoneticCode (string) which is the phonetic code to search for.
	//
	// Returns []string which contains all matching terms.
	// Returns error when the search fails.
	FindPhoneticTerms(phoneticCode string) ([]string, error)

	// GetAllTerms returns all terms in the index vocabulary. Used for fuzzy
	// matching fallback strategies.
	//
	// Returns []string which contains all terms in the vocabulary.
	// Returns error when retrieval fails.
	//
	// This is NOT zero-copy as it builds a slice of strings.
	GetAllTerms() ([]string, error)

	// FindTermsWithPrefix returns all terms in the index that start with the
	// given prefix.
	//
	// Uses binary search for efficient O(log n) lookup. Used for prefix matching
	// to expand partial queries (e.g., "doc" -> ["docs", "documentation"]).
	//
	// Takes prefix (string) which is the term prefix to search for.
	//
	// Returns []string which contains all matching terms.
	// Returns error when the search operation fails.
	FindTermsWithPrefix(prefix string) ([]string, error)
}

// ScoreResult contains the complete result of scoring a document against a
// query. This enables both aggregate relevance ranking and per-field score
// breakdown.
type ScoreResult struct {
	// FieldScores holds the score contribution from each field, keyed by field
	// name (e.g., "title", "content", "excerpt"). Useful for understanding why a
	// document ranked where it did.
	FieldScores map[string]float64

	// Score is the total BM25 relevance score for the document. It is the sum of
	// all field scores and is used to rank search results.
	Score float64
}

// ScorerPort defines the contract for scoring documents against a query.
type ScorerPort interface {
	// Score calculates how relevant a document is for the given query terms.
	//
	// Takes queryTerms ([]string) which contains the words to search for.
	// Takes documentID (uint32) which identifies the document to score.
	// Takes reader (IndexReaderPort) which provides access to the search index.
	// Takes config (search_dto.SearchConfig) which controls how scoring works.
	//
	// Returns ScoreResult which contains the total score and a breakdown by field.
	// Returns error when the score cannot be calculated.
	Score(
		ctx context.Context,
		queryTerms []string,
		documentID uint32,
		reader IndexReaderPort,
		config search_dto.SearchConfig,
	) (ScoreResult, error)
}

// QueryProcessorPort defines the contract for executing search queries.
type QueryProcessorPort interface {
	// Search runs a query against an index and returns ranked results.
	// The results contain document IDs that can be used to fetch full documents
	// from the collection.
	//
	// Takes query (string) which is the search text to match.
	// Takes reader (IndexReaderPort) which provides access to the search index.
	// Takes scorer (ScorerPort) which ranks the matching results.
	// Takes config (search_dto.SearchConfig) which sets search options.
	//
	// Returns []QueryResult which contains the ranked document matches.
	// Returns error when the search fails.
	Search(
		ctx context.Context,
		query string,
		reader IndexReaderPort,
		scorer ScorerPort,
		config search_dto.SearchConfig,
	) ([]QueryResult, error)
}

// IndexBuildConfig contains parameters for index construction.
type IndexBuildConfig struct {
	// FieldWeights maps field names to their weight multipliers for scoring.
	FieldWeights map[string]float64

	// Language specifies the language for stemming, defaulting
	// to "english" with support for "spanish", "french",
	// "russian", "swedish", "norwegian", "hungarian", and
	// "hebrew".
	Language string

	// Format specifies the output encoding. Supported values are "flatbuffers"
	// (default, for production) and "json" (for debugging).
	Format string

	// SearchableMetadataFields specifies which metadata fields to index for
	// search. The index builder checks both lowercase and capitalised variants of
	// each field name.
	//
	// Supported field types:
	//   - String fields: "title", "description", "author"
	//   - Array fields: "tags", "keywords", "categories" (joined with spaces)
	//   - Field names are case-insensitive: "tags" matches both "tags" and "Tags"
	//
	// The RawContent field (the markdown body) is always indexed regardless of
	// this setting.
	SearchableMetadataFields []string

	// BM25K1 controls how fast term frequency gains shrink; default is 1.2.
	BM25K1 float64

	// BM25B is the length normalisation factor for BM25 scoring; default is 0.75.
	BM25B float64

	// MinTokenLength is the shortest token length to index; default is 2.
	MinTokenLength int

	// MaxTokenLength is the maximum length of tokens to index; default is 50.
	MaxTokenLength int

	// AnalysisMode specifies how text is processed during indexing.
	AnalysisMode search_fb.SearchMode

	// StopWordsEnabled controls whether to filter out stop words; default is true.
	StopWordsEnabled bool
}

// PostingInfo represents a decoded posting entry from the search index.
type PostingInfo struct {
	// DocumentID is the unique document identifier used to find and score documents.
	DocumentID uint32

	// TermFrequency is the number of times the term appears in the document.
	TermFrequency uint16

	// FieldID identifies which field this posting is from (0=title, 1=content).
	FieldID uint8
}

// DocMetadataInfo holds metadata about a document in the search index.
type DocMetadataInfo struct {
	// FieldLengths maps field names to their length in characters.
	FieldLengths map[string]uint32

	// Route is the path used to fetch the collection item.
	Route string

	// DocumentID is the unique identifier for the document.
	DocumentID uint32

	// FieldLength is the number of tokens in the document field.
	FieldLength uint32

	// FieldLengthsPacked stores field lengths in a compact binary format.
	FieldLengthsPacked uint32
}

// CorpusStats holds statistics about the full document collection for search
// scoring.
type CorpusStats struct {
	// TotalDocuments is the total number of documents in the corpus.
	TotalDocuments uint32

	// AverageFieldLength is the average document length across the corpus.
	AverageFieldLength float32

	// VocabSize is the total number of unique terms in the corpus.
	VocabSize uint32
}

// QueryResult holds a search result before it is hydrated with full data.
type QueryResult struct {
	// FieldScores maps each field name to its individual score.
	FieldScores map[string]float64

	// Score is the BM25 relevance score; higher values indicate better matches.
	Score float64

	// DocumentID is the unique document identifier used to look up document metadata.
	DocumentID uint32
}

// TermStats holds statistics about a term in the search index.
type TermStats struct {
	// Term is the text being analysed.
	Term string

	// DocumentCount is the number of documents that contain this term.
	DocumentCount uint32

	// TotalOccurrences is the total number of times this term appears.
	TotalOccurrences uint64

	// IDF holds the pre-computed inverse document frequency.
	IDF float64
}

// DefaultIndexBuildConfig returns default settings for building a search index.
//
// Returns IndexBuildConfig which contains settings for English language.
func DefaultIndexBuildConfig() IndexBuildConfig {
	return DefaultIndexBuildConfigForLanguage(LanguageEnglish)
}

// DefaultIndexBuildConfigForLanguage returns default index settings for a
// specific language.
//
// Takes language (string) which specifies the language used for text analysis.
//
// Returns IndexBuildConfig which contains the default settings for the given
// language.
func DefaultIndexBuildConfigForLanguage(language string) IndexBuildConfig {
	return IndexBuildConfig{
		BM25K1:           BM25DefaultK1,
		BM25B:            BM25DefaultB,
		Language:         language,
		MinTokenLength:   DefaultMinTokenLength,
		MaxTokenLength:   DefaultMaxTokenLength,
		StopWordsEnabled: true,
		FieldWeights: map[string]float64{
			"title":   DefaultFieldWeightTitle,
			"content": DefaultFieldWeightContent,
			"excerpt": DefaultFieldWeightExcerpt,
		},
		AnalysisMode: search_fb.SearchModeFast,
		Format:       "flatbuffers",
		SearchableMetadataFields: []string{
			"title",
			"description",
			"tags",
			"keywords",
			"category",
			"section",
		},
	}
}
