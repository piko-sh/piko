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

package search_dto

// JSONSearchIndex represents the full search index stored as JSON.
// This format is easy to read and helpful for debugging, unlike FlatBuffers.
type JSONSearchIndex struct {
	// CollectionName is the name of the collection used for search indexing.
	CollectionName string `json:"collection_name"`

	// Mode specifies the indexing mode; must be "fast" or "smart".
	Mode string `json:"mode"`

	// Language specifies the language code for text processing (e.g. "english").
	Language string `json:"language"`

	// Terms holds the indexed terms sorted by text for binary search lookup.
	Terms []JSONTerm `json:"terms"`

	// Documents holds metadata for each indexed document.
	Documents []JSONDocMetadata `json:"docs"`

	// PhoneticMap maps phonetic codes to term indices; only present in Smart mode.
	PhoneticMap []JSONPhoneticMapping `json:"phonetic_map,omitempty"`

	// Params holds the build settings for index generation.
	Params JSONIndexParams `json:"params"`

	// Version specifies the format version of the search index.
	Version uint32 `json:"version"`

	// TotalDocuments is the total number of documents in the index.
	TotalDocuments uint32 `json:"total_docs"`

	// AverageFieldLength is the mean length of fields across all documents.
	AverageFieldLength float32 `json:"avg_field_length"`

	// VocabSize is the number of unique terms in the index.
	VocabSize uint32 `json:"vocab_size"`
}

// JSONTerm represents a single term in the vocabulary with its postings.
type JSONTerm struct {
	// Text is the term string, stemmed when using Smart mode.
	Text string `json:"text"`

	// Original is the normalised form of the term (Smart mode).
	Original string `json:"original,omitempty"`

	// Phonetic is the pronunciation guide; only present in Smart mode.
	Phonetic string `json:"phonetic,omitempty"`

	// Postings holds the list of documents where this term appears.
	Postings []JSONPosting `json:"postings"`

	// IDF is the inverse document frequency score for this term.
	IDF float32 `json:"idf"`
}

// JSONPosting represents a single place where a term appears in a document.
type JSONPosting struct {
	// DocumentID is the unique identifier for the document.
	DocumentID uint32 `json:"doc_id"`

	// TermFrequency is how many times the term appears in the document.
	TermFrequency uint16 `json:"term_frequency"`

	// FieldID identifies which field within the document contains the term.
	FieldID uint8 `json:"field_id"`

	// Positions is reserved for future use.
	Positions uint32 `json:"positions"`
}

// JSONDocMetadata holds metadata for a single document in a search index.
type JSONDocMetadata struct {
	// Route is the URL path for this document.
	Route string `json:"route"`

	// DocumentID is the unique identifier for this document.
	DocumentID uint32 `json:"doc_id"`

	// FieldLength is the total length of the field in tokens.
	FieldLength uint32 `json:"field_length"`

	// FieldLengthsPacked stores several field lengths in a single value.
	FieldLengthsPacked uint32 `json:"field_lengths_packed"`
}

// JSONPhoneticMapping maps a phonetic code to term indices in the search index.
type JSONPhoneticMapping struct {
	// Code is the phonetic representation of a term (e.g. "SLT" for "slot").
	Code string `json:"code"`

	// TermIndices holds positions in the Terms array for phonetic lookups.
	TermIndices []uint32 `json:"term_indices"`
}

// JSONIndexParams contains the build-time parameters used to create the index.
type JSONIndexParams struct {
	// FieldWeights holds the weight value for each field: [title, content, excerpt].
	FieldWeights []float32 `json:"field_weights"`

	// BM25K1 is the k1 parameter for BM25 ranking; controls term frequency scaling.
	BM25K1 float32 `json:"bm25_k1"`

	// BM25B is the b parameter for BM25 ranking; it controls length normalisation.
	BM25B float32 `json:"bm25_b"`

	// MinTokenLength is the shortest token length to include in the index.
	MinTokenLength uint16 `json:"min_token_length"`

	// MaxTokenLength is the maximum length of a token in characters.
	MaxTokenLength uint16 `json:"max_token_length"`

	// StopWordsEnabled indicates whether stop word filtering is active.
	StopWordsEnabled bool `json:"stop_words_enabled"`
}
