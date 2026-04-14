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

package cache_domain

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"sync"
	"unicode"

	"piko.sh/piko/internal/cache/cache_dto"
)

const (
	// initialTokenBufferCapacity is the starting size for tokenisation buffers.
	// This size handles typical query lengths without needing to grow the buffer.
	initialTokenBufferCapacity = 32

	// bm25K1 is the BM25 term frequency saturation parameter. Higher values
	// make term frequency more important.
	bm25K1 = 1.2

	// bm25B is the BM25 document length normalisation parameter. Values range
	// from 0 (no normalisation) to 1 (full normalisation), with 0.75 being standard.
	bm25B = 0.75
)

// TermMatch specifies how multi-term queries are matched.
type TermMatch int

const (
	// TermMatchAll requires documents to contain ALL query terms (AND logic).
	// This is the default and is consistent with [InvertedIndex.Search].
	TermMatchAll TermMatch = iota

	// TermMatchAny is a term matching mode that matches documents containing any
	// query term using OR logic. Documents matching more terms score higher
	// through BM25 IDF accumulation, making this the preferred mode for hybrid
	// search where text scoring is one of several ranking inputs.
	TermMatchAny
)

// tokeniseBuffers holds reusable buffers for tokenisation to reduce memory
// allocations.
type tokeniseBuffers struct {
	// words holds the extracted words for reuse.
	words []string

	// seen tracks words already added to result to prevent duplicates.
	seen map[string]struct{}

	// result holds unique tokens found during tokenisation.
	result []string
}

// tokenisePool pools tokenisation buffers to eliminate per-query allocations.
var tokenisePool = sync.Pool{
	New: func() any {
		return &tokeniseBuffers{
			words:  make([]string, 0, initialTokenBufferCapacity),
			seen:   make(map[string]struct{}, initialTokenBufferCapacity),
			result: make([]string, 0, initialTokenBufferCapacity),
		}
	},
}

// ScoredResult holds a key and its BM25 relevance score from a scored search.
type ScoredResult[K comparable] struct {
	// Key is the cache key of the matched entry.
	Key K

	// Score is the BM25 relevance score; higher values indicate better matches.
	Score float64
}

// InvertedIndex provides fast full-text search by mapping terms to keys.
// It tokenises text into terms and maintains a reverse mapping from terms
// to the set of keys containing those terms.
//
// Thread-safe for concurrent read/write access.
type InvertedIndex[K comparable] struct {
	// index maps terms to the set of keys that contain each term.
	index map[string]map[K]struct{}

	// keyTerms maps keys to their indexed terms for fast removal.
	keyTerms map[K]map[string]struct{}

	// docLengths stores the number of terms in each indexed document, used for
	// BM25 document length normalisation.
	docLengths map[K]int

	// analyseFunc replaces the default tokenise when set, enabling linguistic
	// processing (stemming, normalisation, stop words, etc.).
	analyseFunc cache_dto.TextAnalyseFunc

	// maxTokens limits the total unique terms in the vocabulary. Zero means unlimited.
	maxTokens int

	// totalDocuments is the number of documents currently indexed.
	totalDocuments int

	// totalTerms is the sum of all document lengths, used to compute average
	// document length for BM25.
	totalTerms int64

	// mu guards concurrent access to the index.
	mu sync.RWMutex
}

// SetAnalyseFunction configures the text analysis function used for
// both indexing and querying. Must be called before adding documents.
//
// Takes analyseFunction (cache_dto.TextAnalyseFunc) which transforms
// text into index terms.
func (idx *InvertedIndex[K]) SetAnalyseFunction(analyseFunction cache_dto.TextAnalyseFunc) {
	idx.analyseFunc = analyseFunction
}

// SetMaxTokens sets the maximum number of unique terms in the vocabulary.
// Zero means unlimited.
//
// Takes maxTokens (int) which specifies the vocabulary limit.
func (idx *InvertedIndex[K]) SetMaxTokens(maxTokens int) {
	idx.maxTokens = maxTokens
}

// Add indexes text content for a key. If the key already exists,
// it removes old terms first before adding new ones.
//
// Takes key (K) which identifies the document.
// Takes texts ([]string) which are the text fields to tokenise and index.
//
// Safe for concurrent use.
func (idx *InvertedIndex[K]) Add(key K, texts []string) {
	terms := idx.tokeniseAll(texts)
	if len(terms) == 0 {
		return
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.AddUnsafe(key, texts)
}

// AddUnsafe indexes text without holding the lock. The caller must hold the
// write lock.
//
// Takes key (K) which identifies the entry to index.
// Takes texts ([]string) which contains the text content to tokenise.
func (idx *InvertedIndex[K]) AddUnsafe(key K, texts []string) {
	terms := idx.tokeniseAll(texts)
	if len(terms) == 0 {
		return
	}

	idx.RemoveKeyUnsafe(key)

	termSet := make(map[string]struct{}, len(terms))
	for _, term := range terms {
		if _, exists := idx.index[term]; !exists {
			if idx.maxTokens > 0 && len(idx.index) >= idx.maxTokens {
				continue
			}
			idx.index[term] = make(map[K]struct{})
		}
		idx.index[term][key] = struct{}{}
		termSet[term] = struct{}{}
	}
	idx.keyTerms[key] = termSet

	idx.docLengths[key] = len(terms)
	idx.totalDocuments++
	idx.totalTerms += int64(len(terms))
}

// Remove removes a key from the index.
//
// Takes key (K) which identifies the document to remove.
//
// Safe for concurrent use.
func (idx *InvertedIndex[K]) Remove(key K) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.RemoveKeyUnsafe(key)
}

// RemoveKeyUnsafe removes a key from the index without holding the lock.
// The caller must hold the write lock.
//
// Takes key (K) which is the key to remove.
func (idx *InvertedIndex[K]) RemoveKeyUnsafe(key K) {
	terms, ok := idx.keyTerms[key]
	if !ok {
		return
	}

	for term := range terms {
		if keys, exists := idx.index[term]; exists {
			delete(keys, key)
			if len(keys) == 0 {
				delete(idx.index, term)
			}
		}
	}

	if dl, exists := idx.docLengths[key]; exists {
		idx.totalDocuments--
		idx.totalTerms -= int64(dl)
		delete(idx.docLengths, key)
	}

	delete(idx.keyTerms, key)
}

// Lock acquires the write lock on the inverted index.
//
// Concurrency: acquires the write lock.
func (idx *InvertedIndex[K]) Lock() {
	idx.mu.Lock()
}

// Unlock releases the write lock on the inverted index.
//
// Concurrency: releases the write lock.
func (idx *InvertedIndex[K]) Unlock() {
	idx.mu.Unlock()
}

// Search finds all keys whose indexed text contains all query terms.
// Returns keys in no particular order.
//
// Takes query (string) which is the search query to tokenise and match.
//
// Returns []K containing keys that match all query terms.
//
// Safe for concurrent use.
func (idx *InvertedIndex[K]) Search(query string) []K {
	terms := idx.tokenise(query)
	if len(terms) == 0 {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(terms) == 1 {
		return idx.searchSingleTerm(terms[0])
	}

	return idx.searchMultiTerm(terms)
}

// searchSingleTerm handles the search for a single term.
// The caller must hold the read lock.
//
// Takes term (string) which specifies the term to look up.
//
// Returns []K which contains the keys linked to the term, or nil if not found.
func (idx *InvertedIndex[K]) searchSingleTerm(term string) []K {
	termKeys, ok := idx.index[term]
	if !ok || len(termKeys) == 0 {
		return nil
	}

	result := make([]K, 0, len(termKeys))
	for k := range termKeys {
		result = append(result, k)
	}
	return result
}

// searchMultiTerm finds keys that match all the given search terms.
//
// Takes terms ([]string) which contains the search terms to intersect.
//
// Returns []K which contains keys present in all term sets, or nil if no
// matches exist.
//
// Caller must hold read lock.
func (idx *InvertedIndex[K]) searchMultiTerm(terms []string) []K {
	smallestTerm, smallestKeys := idx.findSmallestTermSet(terms)
	if smallestKeys == nil {
		return nil
	}

	candidates := make(map[K]struct{}, len(smallestKeys))
	for k := range smallestKeys {
		candidates[k] = struct{}{}
	}

	if !idx.intersectTerms(terms, smallestTerm, candidates) {
		return nil
	}

	return idx.mapToSlice(candidates)
}

// findSmallestTermSet finds the term with the fewest matching keys.
// Caller must hold read lock.
//
// Takes terms ([]string) which contains the terms to evaluate.
//
// Returns string which is the term with the smallest key set.
// Returns map[K]struct{} which is the key set for that term, or nil if any
// term has no matches.
func (idx *InvertedIndex[K]) findSmallestTermSet(terms []string) (string, map[K]struct{}) {
	var smallestTerm string
	var smallestKeys map[K]struct{}
	smallestSize := int(^uint(0) >> 1)

	for _, term := range terms {
		termKeys, ok := idx.index[term]
		if !ok || len(termKeys) == 0 {
			return "", nil
		}
		if len(termKeys) < smallestSize {
			smallestTerm = term
			smallestKeys = termKeys
			smallestSize = len(termKeys)
		}
	}

	return smallestTerm, smallestKeys
}

// intersectTerms filters candidate keys by keeping only those that appear in
// all term sets.
//
// Takes terms ([]string) which contains the search terms to check.
// Takes skipTerm (string) which specifies a term to exclude from filtering.
// Takes candidates (map[K]struct{}) which holds the keys to filter in place.
//
// Returns bool which is false if no candidates remain after filtering.
//
// Caller must hold read lock.
func (idx *InvertedIndex[K]) intersectTerms(terms []string, skipTerm string, candidates map[K]struct{}) bool {
	for _, term := range terms {
		if term == skipTerm {
			continue
		}

		termKeys, ok := idx.index[term]
		if !ok {
			return false
		}

		for k := range candidates {
			if _, exists := termKeys[k]; !exists {
				delete(candidates, k)
			}
		}

		if len(candidates) == 0 {
			return false
		}
	}

	return true
}

// mapToSlice converts a map set to a slice.
// Caller must hold read lock or have exclusive access to the map.
//
// Takes m (map[K]struct{}) which is the set to convert.
//
// Returns []K which contains all keys from the map.
func (*InvertedIndex[K]) mapToSlice(m map[K]struct{}) []K {
	result := make([]K, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// Clear removes all entries from the index.
//
// Safe for concurrent use.
func (idx *InvertedIndex[K]) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.index = make(map[string]map[K]struct{})
	idx.keyTerms = make(map[K]map[string]struct{})
	idx.docLengths = make(map[K]int)
	idx.totalDocuments = 0
	idx.totalTerms = 0
}

// ScoredSearchOption configures behaviour of [InvertedIndex.SearchScored].
type ScoredSearchOption func(*scoredSearchConfig)

// scoredSearchConfig holds resolved options for a scored search.
type scoredSearchConfig struct {
	// match specifies whether all or any search terms must match.
	match TermMatch
}

// SearchScored performs BM25-scored search across indexed TEXT fields,
// returning only documents matching ALL query terms (AND logic) by default
// unless [WithTermMatch]([TermMatchAny]) is passed for OR semantics.
//
// Results are sorted by BM25 relevance score descending.
//
// Takes query (string) which is the search query to tokenise and match.
// Takes opts (...ScoredSearchOption) which configure matching behaviour.
//
// Returns []ScoredResult[K] sorted by score descending, or nil if no matches.
//
// Safe for concurrent use.
func (idx *InvertedIndex[K]) SearchScored(query string, opts ...ScoredSearchOption) []ScoredResult[K] {
	config := &scoredSearchConfig{match: TermMatchAll}
	for _, opt := range opts {
		opt(config)
	}

	terms := idx.tokenise(query)
	if len(terms) == 0 {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.totalDocuments == 0 {
		return nil
	}

	if config.match == TermMatchAll {
		return idx.searchScoredAll(terms)
	}
	return idx.searchScoredAny(terms)
}

// searchScoredAll scores documents matching ALL query terms (AND logic).
// Caller must hold the read lock.
//
// Takes terms ([]string) which contains the query terms to match against.
//
// Returns []ScoredResult[K] which contains the matching documents sorted by
// BM25 score in descending order, or nil if no documents match all terms.
func (idx *InvertedIndex[K]) searchScoredAll(terms []string) []ScoredResult[K] {
	candidates := idx.findCandidatesUnsafe(terms)
	if len(candidates) == 0 {
		return nil
	}

	avgdl := float64(idx.totalTerms) / float64(idx.totalDocuments)
	scores := make(map[K]float64, len(candidates))

	for _, term := range terms {
		termKeys, ok := idx.index[term]
		if !ok {
			continue
		}

		df := float64(len(termKeys))
		n := float64(idx.totalDocuments)
		idf := math.Log(1.0 + (n-df+0.5)/(df+0.5))

		for key := range candidates {
			if _, inTerm := termKeys[key]; !inTerm {
				continue
			}
			dl := float64(idx.docLengths[key])
			tf := 1.0
			denom := tf + bm25K1*(1.0-bm25B+bm25B*dl/avgdl)
			scores[key] += idf * (tf * (bm25K1 + 1.0)) / denom
		}
	}

	return idx.collectAndSort(scores)
}

// searchScoredAny scores documents matching ANY query term (OR logic) where
// documents matching more terms naturally score higher through BM25 IDF
// accumulation (caller must hold the read lock).
//
// Takes terms ([]string) which contains the query terms to match against.
//
// Returns []ScoredResult[K] which contains the matching documents sorted by
// BM25 score in descending order, or nil if no documents match any term.
func (idx *InvertedIndex[K]) searchScoredAny(terms []string) []ScoredResult[K] {
	avgdl := float64(idx.totalTerms) / float64(idx.totalDocuments)
	scores := make(map[K]float64)

	for _, term := range terms {
		termKeys, ok := idx.index[term]
		if !ok {
			continue
		}

		df := float64(len(termKeys))
		n := float64(idx.totalDocuments)
		idf := math.Log(1.0 + (n-df+0.5)/(df+0.5))

		for key := range termKeys {
			dl := float64(idx.docLengths[key])
			tf := 1.0
			denom := tf + bm25K1*(1.0-bm25B+bm25B*dl/avgdl)
			scores[key] += idf * (tf * (bm25K1 + 1.0)) / denom
		}
	}

	return idx.collectAndSort(scores)
}

// collectAndSort converts a score map into a sorted slice of ScoredResult.
// Returns nil if scores is empty.
//
// Takes scores (map[K]float64) which maps document keys to their BM25
// relevance scores.
//
// Returns []ScoredResult[K] which contains the results sorted by score in
// descending order, or nil if scores is empty.
func (*InvertedIndex[K]) collectAndSort(scores map[K]float64) []ScoredResult[K] {
	if len(scores) == 0 {
		return nil
	}

	results := make([]ScoredResult[K], 0, len(scores))
	for key, score := range scores {
		results = append(results, ScoredResult[K]{Key: key, Score: score})
	}

	slices.SortFunc(results, func(a, b ScoredResult[K]) int {
		return cmp.Compare(b.Score, a.Score)
	})

	return results
}

// findCandidatesUnsafe returns the set of keys that match all query terms.
// Caller must hold the read lock.
//
// Takes terms ([]string) which contains the query terms to match against.
//
// Returns map[K]struct{} which contains keys present in all term sets, or nil
// if no matches exist.
func (idx *InvertedIndex[K]) findCandidatesUnsafe(terms []string) map[K]struct{} {
	if len(terms) == 1 {
		return idx.copySingleTermKeys(terms[0])
	}

	smallestTerm, smallestKeys := idx.findSmallestTermSet(terms)
	if smallestKeys == nil {
		return nil
	}

	candidates := make(map[K]struct{}, len(smallestKeys))
	for k := range smallestKeys {
		candidates[k] = struct{}{}
	}

	if !idx.intersectTerms(terms, smallestTerm, candidates) {
		return nil
	}

	return candidates
}

// copySingleTermKeys returns a copy of the key set for a single term, or nil
// if the term is not indexed.
// Caller must hold the read lock.
//
// Takes term (string) which is the term to look up.
//
// Returns map[K]struct{} which contains a copy of the keys for the term.
func (idx *InvertedIndex[K]) copySingleTermKeys(term string) map[K]struct{} {
	termKeys, ok := idx.index[term]
	if !ok || len(termKeys) == 0 {
		return nil
	}
	result := make(map[K]struct{}, len(termKeys))
	for k := range termKeys {
		result[k] = struct{}{}
	}
	return result
}

// tokeniseAll processes multiple text strings and returns unique terms.
//
// Takes texts ([]string) which contains the strings to process.
//
// Returns []string which contains the unique terms from all inputs.
func (idx *InvertedIndex[K]) tokeniseAll(texts []string) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, text := range texts {
		for _, term := range idx.tokenise(text) {
			if _, ok := seen[term]; !ok {
				seen[term] = struct{}{}
				result = append(result, term)
			}
		}
	}
	return result
}

// tokenise splits text into lowercase terms, removing punctuation. When an
// analyseFunc is configured, it delegates to the linguistic analyser instead.
//
// Takes text (string) which is the input to split into terms.
//
// Returns []string which contains unique terms of two or more characters.
// Uses pooled buffers to avoid memory allocation when using the default path.
func (idx *InvertedIndex[K]) tokenise(text string) []string {
	if idx.analyseFunc != nil {
		return idx.analyseFunc(text)
	}

	return idx.tokeniseDefault(text)
}

// tokeniseDefault is the built-in tokeniser that splits text into lowercase
// terms, removing punctuation.
//
// Takes text (string) which is the input to split into terms.
//
// Returns []string which contains unique terms of two or more characters.
func (*InvertedIndex[K]) tokeniseDefault(text string) []string {
	buffers, ok := tokenisePool.Get().(*tokeniseBuffers)
	if !ok {
		buffers = &tokeniseBuffers{
			words:  make([]string, 0, initialTokenBufferCapacity),
			seen:   make(map[string]struct{}, initialTokenBufferCapacity),
			result: make([]string, 0, initialTokenBufferCapacity),
		}
	}
	defer func() {
		buffers.words = buffers.words[:0]
		for k := range buffers.seen {
			delete(buffers.seen, k)
		}
		buffers.result = buffers.result[:0]
		tokenisePool.Put(buffers)
	}()

	text = strings.ToLower(text)

	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		if _, ok := buffers.seen[word]; !ok {
			buffers.seen[word] = struct{}{}
			buffers.result = append(buffers.result, word)
		}
	}

	result := make([]string, len(buffers.result))
	copy(result, buffers.result)
	return result
}

// NewInvertedIndex creates a new empty inverted index.
//
// Returns *InvertedIndex[K] which is an empty index ready for use.
func NewInvertedIndex[K comparable]() *InvertedIndex[K] {
	return &InvertedIndex[K]{
		index:      make(map[string]map[K]struct{}),
		keyTerms:   make(map[K]map[string]struct{}),
		docLengths: make(map[K]int),
	}
}

// WithTermMatch sets the term matching strategy for a scored search.
//
// Takes match ([TermMatch]) which controls AND vs OR semantics.
//
// Returns [ScoredSearchOption] to pass to [InvertedIndex.SearchScored].
func WithTermMatch(match TermMatch) ScoredSearchOption {
	return func(c *scoredSearchConfig) {
		c.match = match
	}
}
