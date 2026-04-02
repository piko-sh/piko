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
	"bytes"
	"errors"
	"fmt"
	"sort"

	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_schema"
	search_fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
	"piko.sh/piko/wdk/safeconv"
)

var (
	// errIndexNotLoaded is the error returned when the index is nil.
	errIndexNotLoaded = errors.New("index not loaded")

	// errIndexDataEmpty is returned when index data has zero length.
	errIndexDataEmpty = errors.New("index data is empty")

	// errSearchSchemaVersionMismatch indicates the search index was serialised with
	// a different schema version. This typically occurs when upgrading Piko and
	// requires index regeneration.
	errSearchSchemaVersionMismatch = fbs.ErrSchemaVersionMismatch
)

// FlatBufferIndexReader implements IndexReaderPort for zero-copy access to
// FlatBuffer indexes. All data access is direct from the embedded binary blob
// with no allocations.
type FlatBufferIndexReader struct {
	// index is the root accessor for the FlatBuffer search index.
	index *search_fb.SearchIndex

	// data holds the raw FlatBuffer bytes loaded at startup.
	data []byte
}

// NewFlatBufferIndexReader creates a new index reader for FlatBuffer data.
//
// Returns *FlatBufferIndexReader which is ready to read FlatBuffer indices.
func NewFlatBufferIndexReader() *FlatBufferIndexReader {
	return &FlatBufferIndexReader{}
}

// LoadIndex initialises the reader with FlatBuffer data.
//
// Takes data ([]byte) which contains the serialised FlatBuffer index.
//
// Returns error when the data is empty or was built with a different
// schema version (ErrSearchSchemaVersionMismatch).
//
// SAFETY: Returned data from methods like GetDocMetadata, GetLanguage,
// etc. contains strings that reference 'data' directly via mem.String.
// Go's GC keeps 'data' alive through these string references. The caller
// must not modify 'data' while the reader is in use.
func (r *FlatBufferIndexReader) LoadIndex(data []byte) error {
	if len(data) == 0 {
		return errIndexDataEmpty
	}

	payload, err := search_schema.Unpack(data)
	if err != nil {
		return fmt.Errorf("search index schema version mismatch (regenerate index): %w", errSearchSchemaVersionMismatch)
	}

	r.data = data
	r.index = search_fb.GetRootAsSearchIndex(payload, 0)

	return nil
}

// GetTermPostings returns the posting list for a term using binary search.
// This is the critical path for search performance - must be zero-copy.
//
// Takes term (string) which is the search term to look up.
//
// Returns []search_domain.PostingInfo which contains the posting list entries.
// Returns float64 which is the IDF (inverse document frequency) score.
// Returns error when the index is not loaded or term access fails.
func (r *FlatBufferIndexReader) GetTermPostings(term string) ([]search_domain.PostingInfo, float64, error) {
	if r.index == nil {
		return nil, 0, errIndexNotLoaded
	}

	termIndex := r.binarySearchTerms(term)
	if termIndex < 0 {
		return nil, 0, nil
	}

	var termObj search_fb.Term
	if !r.index.Terms(&termObj, termIndex) {
		return nil, 0, fmt.Errorf("failed to access term at index %d", termIndex)
	}

	idf := float64(termObj.InverseDocumentFrequency())

	postingsCount := termObj.PostingsLength()
	postings := make([]search_domain.PostingInfo, postingsCount)

	for i := range postingsCount {
		var posting search_fb.Posting
		if !termObj.Postings(&posting, i) {
			return nil, 0, fmt.Errorf("failed to access posting %d", i)
		}

		postings[i] = search_domain.PostingInfo{
			DocumentID:    posting.DocumentId(),
			TermFrequency: posting.TermFrequency(),
			FieldID:       posting.FieldId(),
		}
	}

	return postings, idf, nil
}

// GetDocMetadata returns metadata for a document by direct index access.
//
// Takes documentID (uint32) which is the document identifier to look up.
//
// Returns search_domain.DocMetadataInfo which contains the document metadata.
// Returns error when the index is not loaded or the document ID is out of
// range.
func (r *FlatBufferIndexReader) GetDocMetadata(documentID uint32) (search_domain.DocMetadataInfo, error) {
	if r.index == nil {
		return search_domain.DocMetadataInfo{}, errIndexNotLoaded
	}

	if int(documentID) >= r.index.DocumentsLength() {
		return search_domain.DocMetadataInfo{}, fmt.Errorf("doc ID %d out of range", documentID)
	}

	var docMeta search_fb.DocumentMetadata
	if !r.index.Documents(&docMeta, int(documentID)) {
		return search_domain.DocMetadataInfo{}, fmt.Errorf("failed to access doc metadata %d", documentID)
	}

	return search_domain.DocMetadataInfo{
		DocumentID:         docMeta.DocumentId(),
		FieldLength:        docMeta.FieldLength(),
		FieldLengthsPacked: docMeta.FieldLengthsPacked(),
		Route:              mem.String(docMeta.Route()),
	}, nil
}

// GetCorpusStats returns corpus-wide statistics.
//
// Returns search_domain.CorpusStats which contains document count, average
// field length, and vocabulary size. Returns an empty CorpusStats when the
// index is nil.
func (r *FlatBufferIndexReader) GetCorpusStats() search_domain.CorpusStats {
	if r.index == nil {
		return search_domain.CorpusStats{}
	}

	return search_domain.CorpusStats{
		TotalDocuments:     r.index.TotalDocuments(),
		AverageFieldLength: r.index.AverageFieldLength(),
		VocabSize:          r.index.VocabularySize(),
	}
}

// GetMode returns the search mode this index was built with.
//
// Returns search_fb.SearchMode which indicates the mode used when building
// the index.
func (r *FlatBufferIndexReader) GetMode() search_fb.SearchMode {
	if r.index == nil {
		return search_fb.SearchModeFast
	}
	return r.index.Mode()
}

// GetLanguage returns the language this index was built with.
//
// Returns string which is the language name, defaulting to "english" if the
// index is nil.
func (r *FlatBufferIndexReader) GetLanguage() string {
	if r.index == nil {
		return "english"
	}
	return mem.String(r.index.Language())
}

// FindPhoneticTerms finds all terms matching a phonetic code (Smart mode only).
//
// Takes phoneticCode (string) which specifies the phonetic code to search for.
//
// Returns []string which contains all terms matching the phonetic code.
// Returns error when the index is not loaded or is not in Smart mode.
func (r *FlatBufferIndexReader) FindPhoneticTerms(phoneticCode string) ([]string, error) {
	if r.index == nil {
		return nil, errIndexNotLoaded
	}

	if r.index.Mode() != search_fb.SearchModeSmart {
		return nil, errors.New("phonetic search requires Smart mode index")
	}

	mappingIndex := r.binarySearchPhoneticMap(phoneticCode)
	if mappingIndex < 0 {
		return nil, nil
	}

	var mapping search_fb.PhoneticMapping
	if !r.index.PhoneticMap(&mapping, mappingIndex) {
		return nil, fmt.Errorf("failed to access phonetic mapping at index %d", mappingIndex)
	}

	indicesCount := mapping.TermIndicesLength()
	terms := make([]string, 0, indicesCount)

	for i := range indicesCount {
		termIndex := int(mapping.TermIndices(i))

		var termObj search_fb.Term
		if !r.index.Terms(&termObj, termIndex) {
			continue
		}

		terms = append(terms, mem.String(termObj.Text()))
	}

	return terms, nil
}

// GetAllTerms returns all terms in the index for debugging or analysis.
// This is not zero-copy as it builds a slice of strings.
//
// Returns []string which contains all terms in the index.
// Returns error when the index is not loaded or a term cannot be accessed.
func (r *FlatBufferIndexReader) GetAllTerms() ([]string, error) {
	if r.index == nil {
		return nil, errIndexNotLoaded
	}

	termsCount := r.index.TermsLength()
	terms := make([]string, termsCount)

	for i := range termsCount {
		var term search_fb.Term
		if !r.index.Terms(&term, i) {
			return nil, fmt.Errorf("failed to access term at index %d", i)
		}
		terms[i] = mem.String(term.Text())
	}

	return terms, nil
}

// GetTermStats returns statistics about a specific term for analysis and
// debugging.
//
// Takes term (string) which specifies the term to look up in the index.
//
// Returns *search_domain.TermStats which contains the term's document count,
// total occurrences, and IDF score.
// Returns error when the index is not loaded or the term is not found.
func (r *FlatBufferIndexReader) GetTermStats(term string) (*search_domain.TermStats, error) {
	if r.index == nil {
		return nil, errIndexNotLoaded
	}

	termIndex := r.binarySearchTerms(term)
	if termIndex < 0 {
		return nil, fmt.Errorf("term not found: %s", term)
	}

	var termObj search_fb.Term
	if !r.index.Terms(&termObj, termIndex) {
		return nil, errors.New("failed to access term")
	}

	var totalOccurrences uint64
	postingsCount := termObj.PostingsLength()

	for i := range postingsCount {
		var posting search_fb.Posting
		if termObj.Postings(&posting, i) {
			totalOccurrences += uint64(posting.TermFrequency())
		}
	}

	return &search_domain.TermStats{
		Term:             mem.String(termObj.Text()),
		DocumentCount:    safeconv.IntToUint32(postingsCount),
		TotalOccurrences: totalOccurrences,
		IDF:              float64(termObj.InverseDocumentFrequency()),
	}, nil
}

// FindTermsWithPrefix returns all terms starting with the given prefix.
// Useful for autocomplete functionality.
//
// Takes prefix (string) which specifies the prefix to match against terms.
//
// Returns []string which contains all matching terms in sorted order.
// Returns error when the index is not loaded.
func (r *FlatBufferIndexReader) FindTermsWithPrefix(prefix string) ([]string, error) {
	if r.index == nil {
		return nil, errIndexNotLoaded
	}

	termsCount := r.index.TermsLength()
	startIndex := sort.Search(termsCount, func(i int) bool {
		var term search_fb.Term
		if !r.index.Terms(&term, i) {
			return false
		}
		return mem.String(term.Text()) >= prefix
	})

	if startIndex >= termsCount {
		return nil, nil
	}

	var results []string

	for i := startIndex; i < termsCount; i++ {
		var term search_fb.Term
		if !r.index.Terms(&term, i) {
			break
		}

		termText := mem.String(term.Text())

		if len(termText) < len(prefix) || termText[:len(prefix)] != prefix {
			break
		}

		results = append(results, termText)
	}

	return results, nil
}

// GetIndexMetadata returns metadata about the index for debugging.
//
// Returns map[string]any which contains index statistics and parameters, or a
// minimal map indicating the index is not loaded if the index is nil.
func (r *FlatBufferIndexReader) GetIndexMetadata() map[string]any {
	if r.index == nil {
		return map[string]any{
			"loaded": false,
		}
	}

	var params search_fb.IndexParams
	r.index.Params(&params)

	return map[string]any{
		"loaded":           true,
		"mode":             r.GetMode(),
		"language":         r.GetLanguage(),
		"collection_name":  mem.String(r.index.CollectionName()),
		"total_docs":       r.index.TotalDocuments(),
		"vocab_size":       r.index.VocabularySize(),
		"avg_field_length": r.index.AverageFieldLength(),
		"version":          r.index.Version(),
		"bm25_k1":          params.Bm25K1(),
		"bm25_b":           params.Bm25B(),
		"data_size_bytes":  len(r.data),
	}
}

// binarySearchTerms searches for a term in the sorted terms vector.
//
// Takes searchTerm (string) which specifies the term to find.
//
// Returns int which is the index of the term, or -1 if not found.
func (r *FlatBufferIndexReader) binarySearchTerms(searchTerm string) int {
	termsCount := r.index.TermsLength()

	left, right := 0, termsCount-1

	for left <= right {
		mid := left + (right-left)/2

		var term search_fb.Term
		if !r.index.Terms(&term, mid) {
			return -1
		}

		termBytes := term.Text()
		cmp := bytes.Compare(termBytes, []byte(searchTerm))
		if cmp == 0 {
			return mid
		} else if cmp < 0 {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return -1
}

// binarySearchPhoneticMap searches for a phonetic code in the sorted map.
//
// Takes code (string) which is the phonetic code to find.
//
// Returns int which is the index of the matching entry, or -1 if not found.
func (r *FlatBufferIndexReader) binarySearchPhoneticMap(code string) int {
	if r.index.PhoneticMapLength() == 0 {
		return -1
	}

	left, right := 0, r.index.PhoneticMapLength()-1

	for left <= right {
		mid := left + (right-left)/2

		var mapping search_fb.PhoneticMapping
		if !r.index.PhoneticMap(&mapping, mid) {
			return -1
		}

		mappingBytes := mapping.Code()
		cmp := bytes.Compare(mappingBytes, []byte(code))

		switch {
		case cmp == 0:
			return mid
		case cmp < 0:
			left = mid + 1
		default:
			right = mid - 1
		}
	}

	return -1
}
