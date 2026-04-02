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

package search_adapters

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

// jsonIndexReader implements IndexReaderPort for JSON-based indexes.
// This is primarily for debugging and development; production should use
// FlatBuffers.
type jsonIndexReader struct {
	// index holds the parsed search index data; nil until LoadIndex is called.
	index *search_dto.JSONSearchIndex
}

// LoadIndex parses JSON data and loads it into memory.
//
// Takes data ([]byte) which contains the JSON-encoded search index.
//
// Returns error when data is empty or contains invalid JSON.
func (r *jsonIndexReader) LoadIndex(data []byte) error {
	if len(data) == 0 {
		return errIndexDataEmpty
	}

	r.index = &search_dto.JSONSearchIndex{}
	if err := json.Unmarshal(data, r.index); err != nil {
		return fmt.Errorf("failed to parse JSON index: %w", err)
	}

	return nil
}

// GetTermPostings returns the posting list for a term using binary search.
//
// Takes term (string) which specifies the term to look up in the index.
//
// Returns []search_domain.PostingInfo which contains the posting list entries.
// Returns float64 which is the inverse document frequency for the term.
// Returns error when the index has not been loaded.
func (r *jsonIndexReader) GetTermPostings(term string) ([]search_domain.PostingInfo, float64, error) {
	if r.index == nil {
		return nil, 0, errIndexNotLoaded
	}

	index := sort.Search(len(r.index.Terms), func(i int) bool {
		return r.index.Terms[i].Text >= term
	})

	if index >= len(r.index.Terms) || r.index.Terms[index].Text != term {
		return nil, 0, nil
	}

	termData := r.index.Terms[index]
	postings := make([]search_domain.PostingInfo, len(termData.Postings))

	for i, p := range termData.Postings {
		postings[i] = search_domain.PostingInfo{
			DocumentID:    p.DocumentID,
			TermFrequency: p.TermFrequency,
			FieldID:       p.FieldID,
		}
	}

	return postings, float64(termData.IDF), nil
}

// GetDocMetadata returns metadata for a document by direct index access.
//
// Takes documentID (uint32) which specifies the document to retrieve.
//
// Returns search_domain.DocMetadataInfo which contains the document metadata.
// Returns error when the index is not loaded or documentID is out of range.
func (r *jsonIndexReader) GetDocMetadata(documentID uint32) (search_domain.DocMetadataInfo, error) {
	if r.index == nil {
		return search_domain.DocMetadataInfo{}, errIndexNotLoaded
	}

	if int(documentID) >= len(r.index.Documents) {
		return search_domain.DocMetadataInfo{}, fmt.Errorf("doc ID %d out of range", documentID)
	}

	document := r.index.Documents[documentID]
	return search_domain.DocMetadataInfo{
		DocumentID:         document.DocumentID,
		FieldLength:        document.FieldLength,
		FieldLengthsPacked: document.FieldLengthsPacked,
		Route:              document.Route,
	}, nil
}

// GetCorpusStats returns corpus-wide statistics.
//
// Returns search_domain.CorpusStats which contains the total document count,
// average field length, and vocabulary size. Returns an empty CorpusStats if
// the index is nil.
func (r *jsonIndexReader) GetCorpusStats() search_domain.CorpusStats {
	if r.index == nil {
		return search_domain.CorpusStats{}
	}

	return search_domain.CorpusStats{
		TotalDocuments:     r.index.TotalDocuments,
		AverageFieldLength: r.index.AverageFieldLength,
		VocabSize:          r.index.VocabSize,
	}
}

// GetMode returns the search mode this index was built with.
//
// Returns search_schema_gen.SearchMode which is the mode used to build this index,
// defaulting to SearchModeFast if the index is nil or not in smart mode.
func (r *jsonIndexReader) GetMode() search_schema_gen.SearchMode {
	if r.index == nil {
		return search_schema_gen.SearchModeFast
	}

	if r.index.Mode == "smart" {
		return search_schema_gen.SearchModeSmart
	}
	return search_schema_gen.SearchModeFast
}

// GetLanguage returns the language this index was built with.
//
// Returns string which is the language name, defaulting to "english" if the
// index is nil.
func (r *jsonIndexReader) GetLanguage() string {
	if r.index == nil {
		return "english"
	}
	return r.index.Language
}

// FindPhoneticTerms finds all terms matching a phonetic code (Smart mode only).
//
// Takes phoneticCode (string) which specifies the phonetic code to search for.
//
// Returns []string which contains the matching terms, or nil if none found.
// Returns error when the index is not loaded or not in Smart mode.
func (r *jsonIndexReader) FindPhoneticTerms(phoneticCode string) ([]string, error) {
	if r.index == nil {
		return nil, errIndexNotLoaded
	}

	if r.index.Mode != "smart" {
		return nil, errors.New("phonetic search requires Smart mode index")
	}

	index := sort.Search(len(r.index.PhoneticMap), func(i int) bool {
		return r.index.PhoneticMap[i].Code >= phoneticCode
	})

	if index >= len(r.index.PhoneticMap) || r.index.PhoneticMap[index].Code != phoneticCode {
		return nil, nil
	}

	mapping := r.index.PhoneticMap[index]
	terms := make([]string, 0, len(mapping.TermIndices))

	for _, termIndex := range mapping.TermIndices {
		if int(termIndex) < len(r.index.Terms) {
			terms = append(terms, r.index.Terms[termIndex].Text)
		}
	}

	return terms, nil
}

// GetAllTerms returns all terms in the index.
//
// Returns []string which contains all indexed terms.
// Returns error when the index is not loaded.
func (r *jsonIndexReader) GetAllTerms() ([]string, error) {
	if r.index == nil {
		return nil, errIndexNotLoaded
	}

	terms := make([]string, len(r.index.Terms))
	for i, term := range r.index.Terms {
		terms[i] = term.Text
	}

	return terms, nil
}

// FindTermsWithPrefix returns all terms starting with the given prefix.
//
// Takes prefix (string) which specifies the prefix to match against terms.
//
// Returns []string which contains all matching terms, or nil if none found.
// Returns error when the index has not been loaded.
func (r *jsonIndexReader) FindTermsWithPrefix(prefix string) ([]string, error) {
	if r.index == nil {
		return nil, errIndexNotLoaded
	}

	startIndex := sort.Search(len(r.index.Terms), func(i int) bool {
		return r.index.Terms[i].Text >= prefix
	})

	if startIndex >= len(r.index.Terms) {
		return nil, nil
	}

	var results []string
	for i := startIndex; i < len(r.index.Terms); i++ {
		termText := r.index.Terms[i].Text

		if !strings.HasPrefix(termText, prefix) {
			break
		}

		results = append(results, termText)
	}

	return results, nil
}

// newJSONIndexReader creates a new JSON index reader.
//
// Returns *jsonIndexReader which is ready to read JSON index files.
func newJSONIndexReader() *jsonIndexReader {
	return &jsonIndexReader{}
}
