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

package search

import (
	"errors"

	search_fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

// errFlatBufferParseFailed is returned when a FlatBuffer payload cannot be
// decoded.
var errFlatBufferParseFailed = errors.New("failed to parse FlatBuffer payload")

// SearchIndex is a JSON-serialisable representation of a compiled search index.
type SearchIndex struct {
	// Params holds the BM25 tuning parameters.
	Params *IndexParams `json:"params,omitempty"`

	// Mode is the search mode: "Fast" or "Smart".
	Mode string `json:"mode"`

	// CollectionName is the name of the indexed collection.
	CollectionName string `json:"collection_name"`

	// Language is the language code used for indexing.
	Language string `json:"language"`

	// Terms holds all indexed terms with their postings.
	Terms []Term `json:"terms"`

	// Documents holds per-document metadata for scoring.
	Documents []DocMetadata `json:"docs"`

	// Version is the schema version number.
	Version uint32 `json:"version"`

	// TotalDocuments is the total number of indexed documents.
	TotalDocuments uint32 `json:"total_docs"`

	// AverageFieldLength is the average field length for BM25 normalisation.
	AverageFieldLength float32 `json:"avg_field_length"`

	// VocabSize is the number of unique terms in the index.
	VocabSize uint32 `json:"vocab_size"`
}

// IndexParams holds BM25 and tokenisation configuration.
type IndexParams struct {
	// FieldWeights specifies the relative importance of each field in search ranking.
	FieldWeights []float32 `json:"field_weights,omitempty"`

	// BM25K1 is the term frequency saturation parameter for BM25 ranking.
	BM25K1 float32 `json:"bm25_k1"`

	// BM25B is the BM25 document length normalisation parameter; typically 0.75.
	BM25B float32 `json:"bm25_b"`

	// MinTokenLength is the minimum token length for indexing; shorter tokens
	// are ignored.
	MinTokenLength uint16 `json:"min_token_length"`

	// MaxTokenLength is the maximum token length; longer tokens are ignored.
	MaxTokenLength uint16 `json:"max_token_length"`

	// StopWordsEnabled indicates whether common words are filtered from the index.
	StopWordsEnabled bool `json:"stop_words_enabled"`
}

// Term is an indexed term with its inverted postings list.
type Term struct {
	// Text is the indexed form of the term.
	Text string `json:"text"`

	// Original is the pre-stemmed form (Smart mode only).
	Original string `json:"original,omitempty"`

	// Phonetic is the Double Metaphone code (Smart mode only).
	Phonetic string `json:"phonetic,omitempty"`

	// Postings lists which documents contain this term.
	Postings []Posting `json:"postings"`

	// IDF is the pre-computed inverse document frequency.
	IDF float32 `json:"idf"`
}

// Posting records a single term occurrence in a document.
type Posting struct {
	// DocumentID identifies the document.
	DocumentID uint32 `json:"doc_id"`

	// TermFrequency is the count of this term in the document.
	TermFrequency uint16 `json:"term_frequency"`

	// FieldID indicates the field: 0=title, 1=content, 2=excerpt.
	FieldID uint8 `json:"field_id"`
}

// DocMetadata holds per-document scoring information.
type DocMetadata struct {
	// Route is the URL path for mapping results back to documents.
	Route string `json:"route"`

	// DocumentID identifies the document.
	DocumentID uint32 `json:"doc_id"`

	// FieldLength is the total token count across all fields.
	FieldLength uint32 `json:"field_length"`

	// FieldLengthsPacked packs per-field lengths into a single uint32.
	FieldLengthsPacked uint32 `json:"field_lengths_packed"`
}

// ConvertSearchIndex parses a raw FlatBuffer search index payload into a
// JSON-serialisable struct.
//
// Takes payload ([]byte) which is the raw FlatBuffer data after stripping the
// version header (use Unpack first).
//
// Returns *SearchIndex which contains the full index data.
// Returns error when the payload cannot be parsed.
func ConvertSearchIndex(payload []byte) (*SearchIndex, error) {
	fb := search_fb.GetRootAsSearchIndex(payload, 0)
	if fb == nil {
		return nil, errFlatBufferParseFailed
	}

	return &SearchIndex{
		Mode:               fb.Mode().String(),
		CollectionName:     string(fb.CollectionName()),
		Language:           string(fb.Language()),
		Version:            fb.Version(),
		TotalDocuments:     fb.TotalDocuments(),
		AverageFieldLength: fb.AverageFieldLength(),
		VocabSize:          fb.VocabularySize(),
		Params:             convertParams(fb),
		Terms:              convertTerms(fb),
		Documents:          convertDocuments(fb),
	}, nil
}

// convertParams extracts BM25 parameters from the FlatBuffer.
//
// Takes fb (*search_fb.SearchIndex) which contains the serialised index data.
//
// Returns *IndexParams which holds the extracted BM25 and tokenisation
// settings, or nil if the FlatBuffer contains no parameters.
func convertParams(fb *search_fb.SearchIndex) *IndexParams {
	var paramsFB search_fb.IndexParams
	params := fb.Params(&paramsFB)
	if params == nil {
		return nil
	}

	weights := convertFieldWeights(params)

	return &IndexParams{
		BM25K1:           params.Bm25K1(),
		BM25B:            params.Bm25B(),
		MinTokenLength:   params.MinTokenLength(),
		MaxTokenLength:   params.MaxTokenLength(),
		StopWordsEnabled: params.StopWordsEnabled(),
		FieldWeights:     weights,
	}
}

// convertFieldWeights extracts the field weight vector.
//
// Takes fb (*search_fb.IndexParams) which contains the index parameters.
//
// Returns []float32 which contains the field weights, or nil if none exist.
func convertFieldWeights(fb *search_fb.IndexParams) []float32 {
	length := fb.FieldWeightsLength()
	if length == 0 {
		return nil
	}
	weights := make([]float32, length)
	for i := range length {
		weights[i] = fb.FieldWeights(i)
	}
	return weights
}

// convertTerms extracts all terms from the FlatBuffer.
//
// Takes fb (*search_fb.SearchIndex) which contains the serialised search index.
//
// Returns []Term which contains the converted terms, or nil if there are none.
func convertTerms(fb *search_fb.SearchIndex) []Term {
	length := fb.TermsLength()
	if length == 0 {
		return nil
	}
	terms := make([]Term, length)
	var item search_fb.Term
	for i := range length {
		if fb.Terms(&item, i) {
			terms[i] = convertTerm(&item)
		}
	}
	return terms
}

// convertTerm converts a single FlatBuffer term to the native Term type.
//
// Takes fb (*search_fb.Term) which is the FlatBuffer term to convert.
//
// Returns Term which contains the converted text, phonetic data, and postings.
func convertTerm(fb *search_fb.Term) Term {
	return Term{
		Text:     string(fb.Text()),
		Original: string(fb.Original()),
		Phonetic: string(fb.Phonetic()),
		IDF:      fb.InverseDocumentFrequency(),
		Postings: convertPostings(fb),
	}
}

// convertPostings extracts postings from a FlatBuffers term.
//
// Takes fb (*search_fb.Term) which is the FlatBuffers term to extract from.
//
// Returns []Posting which contains the extracted posting data, or nil if empty.
func convertPostings(fb *search_fb.Term) []Posting {
	length := fb.PostingsLength()
	if length == 0 {
		return nil
	}
	postings := make([]Posting, length)
	var item search_fb.Posting
	for i := range length {
		if fb.Postings(&item, i) {
			postings[i] = Posting{
				DocumentID:    item.DocumentId(),
				TermFrequency: item.TermFrequency(),
				FieldID:       item.FieldId(),
			}
		}
	}
	return postings
}

// convertDocuments extracts document metadata from the FlatBuffer.
//
// Takes fb (*search_fb.SearchIndex) which contains the serialised search index.
//
// Returns []DocMetadata which holds the extracted document metadata, or nil
// if no documents exist.
func convertDocuments(fb *search_fb.SearchIndex) []DocMetadata {
	length := fb.DocumentsLength()
	if length == 0 {
		return nil
	}
	docs := make([]DocMetadata, length)
	var item search_fb.DocumentMetadata
	for i := range length {
		if fb.Documents(&item, i) {
			docs[i] = DocMetadata{
				DocumentID:         item.DocumentId(),
				FieldLength:        item.FieldLength(),
				FieldLengthsPacked: item.FieldLengthsPacked(),
				Route:              string(item.Route()),
			}
		}
	}
	return docs
}
