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

package provider_otter

import (
	"context"
	"slices"
	"sync"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

const (
	// keyWithValuePoolMaxCap is the maximum capacity for pooled slices. Slices
	// larger than this won't be returned to the pool to prevent memory bloat.
	keyWithValuePoolMaxCap = 10000

	// initialKeyWithValueCapacity is the starting capacity for pooled keyWithValue
	// slices. Sized to handle typical sort operations without reallocation.
	initialKeyWithValueCapacity = 128
)

// keyWithValue holds a key and its extracted field value for sorting.
type keyWithValue[K comparable] struct {
	// key is the map key for this entry.
	key K

	// value holds the parsed value for this key.
	value any
}

// keyWithValuePool reuses keyWithValue slices to reduce allocation pressure
// during search result sorting.
var keyWithValuePool = sync.Pool{
	New: func() any {
		return new(make([]keyWithValue[any], 0, 128))
	},
}

// indexDocument adds a value to the search indexes if search is enabled.
//
// Takes key (K) which identifies the document in the indexes.
// Takes value (V) which is the document to extract fields from for indexing.
func (a *OtterAdapter[K, V]) indexDocument(key K, value V) {
	if a.schema == nil || a.fieldExtractor == nil {
		return
	}

	a.indexTextFields(key, value)
	a.indexSortedFields(key, value)
	a.indexVectorFields(key, value)
}

// indexTextFields adds a value's text fields to the inverted index.
//
// Takes key (K) which identifies the entry in the index.
// Takes value (V) which contains the text fields to extract and index.
func (a *OtterAdapter[K, V]) indexTextFields(key K, value V) {
	if a.invertedIndex == nil {
		return
	}
	texts := a.fieldExtractor.ExtractTextFields(value)
	if len(texts) > 0 {
		a.invertedIndex.Add(key, texts)
	}
}

// indexSortedFields adds a value's sortable fields to the sorted indexes.
//
// Takes key (K) which identifies the entry being indexed.
// Takes value (V) which contains the fields to extract and index.
func (a *OtterAdapter[K, V]) indexSortedFields(key K, value V) {
	if a.sortedIndexes == nil {
		return
	}
	for fieldName := range a.sortedIndexes {
		if sortableValue, ok := a.fieldExtractor.ExtractSortableValue(value, fieldName); ok {
			a.sortedIndexes[fieldName].Add(key, sortableValue)
		}
	}
}

// indexVectorFields adds a value's vector fields to the vector indexes.
//
// Takes key (K) which identifies the entry being indexed.
// Takes value (V) which contains the vector fields to extract and index.
func (a *OtterAdapter[K, V]) indexVectorFields(key K, value V) {
	if a.vectorIndexes == nil {
		return
	}
	for fieldName, index := range a.vectorIndexes {
		if vec, ok := a.fieldExtractor.ExtractVectorValue(value, fieldName); ok {
			index.Add(key, vec)
		}
	}
}

// removeFromSearchIndex removes a key from all search indexes.
//
// Takes key (K) which is the key to remove from the indexes.
func (a *OtterAdapter[K, V]) removeFromSearchIndex(key K) {
	if a.invertedIndex != nil {
		a.invertedIndex.Remove(key)
	}
	if a.sortedIndexes != nil {
		for _, index := range a.sortedIndexes {
			index.Remove(key)
		}
	}
	if a.vectorIndexes != nil {
		for _, index := range a.vectorIndexes {
			index.Remove(key)
		}
	}
}

// indexDocumentsBatch indexes multiple documents with batched lock access.
//
// Takes items (map[K]V) which contains the key-value pairs to index.
func (a *OtterAdapter[K, V]) indexDocumentsBatch(items map[K]V) {
	if a.schema == nil || a.fieldExtractor == nil {
		return
	}

	a.batchIndexTextFields(items)
	a.batchIndexSortableFields(items)
	a.batchIndexVectorFields(items)
}

// batchIndexTextFields indexes text fields for the inverted index with a
// single lock.
//
// Takes items (map[K]V) which contains the key-value pairs to index.
//
// Safe for concurrent use. Acquires the inverted index lock for the entire
// batch operation.
func (a *OtterAdapter[K, V]) batchIndexTextFields(items map[K]V) {
	if a.invertedIndex == nil {
		return
	}

	a.invertedIndex.Lock()
	defer a.invertedIndex.Unlock()

	for key, value := range items {
		texts := a.fieldExtractor.ExtractTextFields(value)
		if len(texts) > 0 {
			a.invertedIndex.AddUnsafe(key, texts)
		}
	}
}

// batchIndexSortableFields indexes sortable fields with per-field locking.
//
// Takes items (map[K]V) which contains the key-value pairs to index.
//
// Safe for concurrent use. Each sorted index is locked individually during
// updates to allow concurrent indexing of different fields.
func (a *OtterAdapter[K, V]) batchIndexSortableFields(items map[K]V) {
	if a.sortedIndexes == nil {
		return
	}

	for fieldName, index := range a.sortedIndexes {
		index.Lock()
		for key, value := range items {
			if sortableValue, ok := a.fieldExtractor.ExtractSortableValue(value, fieldName); ok {
				index.AddUnsafe(key, sortableValue)
			}
		}
		index.Unlock()
	}
}

// batchIndexVectorFields indexes vector fields for all items.
//
// Takes items (map[K]V) which contains the key-value pairs to index.
//
// The HNSW graph handles its own internal locking, so no external
// synchronisation is needed here.
func (a *OtterAdapter[K, V]) batchIndexVectorFields(items map[K]V) {
	if a.vectorIndexes == nil {
		return
	}

	for fieldName, index := range a.vectorIndexes {
		for key, value := range items {
			if vec, ok := a.fieldExtractor.ExtractVectorValue(value, fieldName); ok {
				index.Add(key, vec)
			}
		}
	}
}

// clearAllIndexes resets all secondary indexes (tags, inverted, sorted,
// vector).
func (a *OtterAdapter[K, V]) clearAllIndexes() {
	a.tagIndex.Clear()
	if a.invertedIndex != nil {
		a.invertedIndex.Clear()
	}
	if a.sortedIndexes != nil {
		for _, index := range a.sortedIndexes {
			index.Clear()
		}
	}
	if a.vectorIndexes != nil {
		for _, index := range a.vectorIndexes {
			index.Clear()
		}
	}
}

// Search performs full-text search across indexed TEXT fields.
// When a text analyser is configured, results are scored using
// BM25 relevance ranking.
//
// Takes query (string) which is the search query to tokenise and
// match.
// Takes opts (*cache_dto.SearchOptions) which configures
// pagination, sorting, and filters.
//
// Returns cache_dto.SearchResult[K, V] which contains matched
// entries with metadata.
// Returns error when no schema is configured
// (ErrSearchNotSupported).
func (a *OtterAdapter[K, V]) Search(_ context.Context, query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache_dto.SearchResult[K, V]{}, cache_domain.ErrSearchNotSupported
	}

	if len(opts.Vector) > 0 && a.vectorIndexes != nil {
		return a.vectorSearch(query, opts)
	}

	var candidateKeys []K
	var keyScores map[K]float64
	var keysAreTrusted bool

	if a.invertedIndex != nil && query != "" {
		scored := a.invertedIndex.SearchScored(query)
		if len(scored) > 0 {
			candidateKeys = make([]K, len(scored))
			keyScores = make(map[K]float64, len(scored))
			for i, s := range scored {
				candidateKeys[i] = s.Key
				keyScores[s.Key] = s.Score
			}
			keysAreTrusted = true
		}
	}

	if candidateKeys == nil {
		if query != "" {
			return cache_dto.SearchResult[K, V]{
				Items:  nil,
				Total:  0,
				Offset: opts.Offset,
				Limit:  opts.Limit,
			}, nil
		}
		candidateKeys = a.getAllKeys()
		keysAreTrusted = false
	}

	if len(candidateKeys) == 0 {
		return cache_dto.SearchResult[K, V]{
			Items:  nil,
			Total:  0,
			Offset: opts.Offset,
			Limit:  opts.Limit,
		}, nil
	}

	filteredKeys := a.applyFiltersWithTrust(candidateKeys, opts.Filters, keysAreTrusted)

	sortedKeys := a.sortKeys(filteredKeys, opts.SortBy, opts.SortOrder)

	return a.buildSearchResultWithScores(sortedKeys, keyScores, opts.Offset, opts.Limit)
}

// Query performs structured filtering, sorting, and pagination
// without full-text search.
//
// Takes opts (*cache_dto.QueryOptions) which specifies filters,
// sorting, and pagination.
//
// Returns cache_dto.SearchResult[K, V] which contains matched
// entries.
// Returns error when no schema is configured
// (ErrSearchNotSupported).
func (a *OtterAdapter[K, V]) Query(_ context.Context, opts *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache_dto.SearchResult[K, V]{}, cache_domain.ErrSearchNotSupported
	}

	if len(opts.Vector) > 0 && a.vectorIndexes != nil {
		return a.vectorQuery(opts)
	}

	candidateKeys := a.getAllKeys()

	if len(candidateKeys) == 0 {
		return cache_dto.SearchResult[K, V]{
			Items:  nil,
			Total:  0,
			Offset: opts.Offset,
			Limit:  opts.Limit,
		}, nil
	}

	filteredKeys := a.applyFiltersWithTrust(candidateKeys, opts.Filters, false)

	sortedKeys := a.sortKeys(filteredKeys, opts.SortBy, opts.SortOrder)

	return a.buildSearchResult(sortedKeys, opts.Offset, opts.Limit)
}

// SupportsSearch returns true if a search schema is configured.
//
// Returns bool which is true when search operations are available.
func (a *OtterAdapter[K, V]) SupportsSearch() bool {
	return a.schema != nil
}

// GetSchema returns the search schema for this cache.
//
// Returns *cache_dto.SearchSchema which describes searchable fields, or nil.
func (a *OtterAdapter[K, V]) GetSchema() *cache_dto.SearchSchema {
	return a.schema
}

// getAllKeys returns all keys stored in the cache.
//
// Returns []K which contains all keys present in the cache.
func (a *OtterAdapter[K, V]) getAllKeys() []K {
	keys := make([]K, 0, a.client.EstimatedSize())
	for k := range a.client.Keys() {
		keys = append(keys, k)
	}
	return keys
}

// applyFiltersWithTrust filters keys based on the provided filter conditions.
//
// When trustKeys is true, skips existence validation as keys from InvertedIndex
// are trusted. Optimises range filters by using SortedIndex B-tree range
// queries when available.
//
// Takes keys ([]K) which specifies the keys to filter.
// Takes filters ([]cache_dto.Filter) which provides the filter conditions.
// Takes trustKeys (bool) which indicates whether to skip existence validation.
//
// Returns []K which contains the keys that match all filter conditions.
func (a *OtterAdapter[K, V]) applyFiltersWithTrust(keys []K, filters []cache_dto.Filter, trustKeys bool) []K {
	if len(filters) == 0 || a.fieldExtractor == nil {
		return keys
	}

	if len(filters) == 1 && len(keys) == a.client.EstimatedSize() {
		if optimisedKeys := a.tryRangeQueryFilter(filters[0]); optimisedKeys != nil {
			return optimisedKeys
		}
	}

	result := make([]K, 0, len(keys))
	for _, key := range keys {
		value, ok := a.client.GetIfPresent(key)
		if !trustKeys && !ok {
			continue
		}

		if a.matchesAllFilters(value, filters) {
			result = append(result, key)
		}
	}
	return result
}

// tryRangeQueryFilter attempts to use B-tree range queries for efficient
// filtering.
//
// Takes filter (cache_dto.Filter) which specifies the field, operation, and
// value(s) to filter by.
//
// Returns []K which contains the matching keys, or nil if the optimisation
// cannot be applied.
func (a *OtterAdapter[K, V]) tryRangeQueryFilter(filter cache_dto.Filter) []K {
	if a.sortedIndexes == nil {
		return nil
	}

	index, ok := a.sortedIndexes[filter.Field]
	if !ok {
		return nil
	}

	switch filter.Operation {
	case cache_dto.FilterOpGt:
		return index.KeysGreaterThan(filter.Value, true)
	case cache_dto.FilterOpGe:
		return index.KeysGreaterThanOrEqual(filter.Value, true)
	case cache_dto.FilterOpLt:
		return index.KeysLessThan(filter.Value, true)
	case cache_dto.FilterOpLe:
		return index.KeysLessThanOrEqual(filter.Value, true)
	case cache_dto.FilterOpBetween:
		if len(filter.Values) == 2 {
			return index.KeysBetween(filter.Values[0], filter.Values[1], true)
		}
		return nil
	default:
		return nil
	}
}

// matchesAllFilters checks if a value matches all filter conditions.
//
// Takes value (V) which is the value to check against the filters.
// Takes filters ([]cache_dto.Filter) which contains the filter conditions.
//
// Returns bool which is true if the value matches all filters.
func (a *OtterAdapter[K, V]) matchesAllFilters(value V, filters []cache_dto.Filter) bool {
	return cache_domain.MatchesAllFilters(a.fieldExtractor, value, filters)
}

// sortKeys sorts keys by the specified field and order.
//
// Takes keys ([]K) which contains the keys to sort.
// Takes sortBy (string) which specifies the field name to sort by.
// Takes sortOrder (cache_dto.SortOrder) which specifies ascending or
// descending.
//
// Returns []K which contains the sorted keys, or the original keys if sortBy
// is empty or keys is empty.
func (a *OtterAdapter[K, V]) sortKeys(keys []K, sortBy string, sortOrder cache_dto.SortOrder) []K {
	if sortBy == "" || len(keys) == 0 {
		return keys
	}

	ascending := sortOrder == cache_dto.SortAsc

	if a.sortedIndexes != nil {
		if sortedIndex, ok := a.sortedIndexes[sortBy]; ok {
			return sortedIndex.KeysFilteredSlice(keys, ascending)
		}
	}

	return a.sortKeysByField(keys, sortBy, ascending)
}

// sortKeysByField sorts keys by extracting field values and comparing.
// Uses a sync.Pool to reduce allocations.
//
// Takes keys ([]K) which contains the keys to sort.
// Takes fieldName (string) which specifies the field to extract for comparison.
// Takes ascending (bool) which determines the sort direction.
//
// Returns []K which contains the sorted keys.
func (a *OtterAdapter[K, V]) sortKeysByField(keys []K, fieldName string, ascending bool) []K {
	if a.fieldExtractor == nil {
		return keys
	}

	itemsPtr, ok := keyWithValuePool.Get().(*[]keyWithValue[any])
	if !ok {
		itemsPtr = new(make([]keyWithValue[any], 0, initialKeyWithValueCapacity))
	}
	items := (*itemsPtr)[:0]

	for _, key := range keys {
		value, ok := a.client.GetIfPresent(key)
		if !ok {
			continue
		}
		fieldValue, _ := a.fieldExtractor.ExtractAny(value, fieldName)
		items = append(items, keyWithValue[any]{key: key, value: fieldValue})
	}

	slices.SortFunc(items, func(itemA, itemB keyWithValue[any]) int {
		comparison := cache_domain.CompareNumeric(itemA.value, itemB.value)
		if !ascending {
			comparison = -comparison
		}
		return comparison
	})

	result := make([]K, 0, len(items))
	for _, item := range items {
		if key, ok := item.key.(K); ok {
			result = append(result, key)
		}
	}

	if cap(items) <= keyWithValuePoolMaxCap {
		*itemsPtr = items[:0]
		keyWithValuePool.Put(itemsPtr)
	}

	return result
}

// buildSearchResult creates a SearchResult with pagination
// applied and flat scoring (1.0 for all hits).
//
// Takes keys ([]K) which contains the matched keys in display
// order.
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns cache_dto.SearchResult[K, V] which contains the
// paginated results with flat scores.
// Returns error which is always nil.
func (a *OtterAdapter[K, V]) buildSearchResult(keys []K, offset, limit int) (cache_dto.SearchResult[K, V], error) {
	return a.buildSearchResultWithScores(keys, nil, offset, limit)
}

// buildSearchResultWithScores creates a SearchResult with pagination applied.
// When scores is non-nil, each hit uses the score from the map; otherwise a
// flat score of 1.0 is used.
//
// Takes keys ([]K) which contains the matched keys in display order.
// Takes scores (map[K]float64) which maps keys to relevance scores (may be
// nil).
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns SearchResult with scored items.
func (a *OtterAdapter[K, V]) buildSearchResultWithScores(keys []K, scores map[K]float64, offset, limit int) (cache_dto.SearchResult[K, V], error) {
	total := int64(len(keys))
	keys, limit = cache_domain.ApplyPagination(keys, offset, limit)

	items := make([]cache_dto.SearchHit[K, V], 0, len(keys))
	for _, key := range keys {
		value, ok := a.client.GetIfPresent(key)
		if !ok {
			continue
		}

		score := 1.0
		if scores != nil {
			if s, exists := scores[key]; exists {
				score = s
			}
		}

		items = append(items, cache_dto.SearchHit[K, V]{
			Key:   key,
			Value: value,
			Score: score,
		})
	}

	return cache_dto.SearchResult[K, V]{
		Items:  items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

// vectorSearch performs vector similarity search with optional text fusion.
// When a text query is provided, results are combined using Reciprocal Rank
// Fusion (RRF) to merge semantic and lexical relevance signals.
//
// Takes query (string) which is an optional text query for hybrid search.
// Takes opts (*cache_dto.SearchOptions) which provides the vector, filters,
// and pagination parameters.
//
// Returns SearchResult with items sorted by score (highest first).
func (a *OtterAdapter[K, V]) vectorSearch(query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	vectorField := a.resolveVectorField(opts.VectorField)
	index, ok := a.vectorIndexes[vectorField]
	if !ok {
		return cache_dto.SearchResult[K, V]{
			Offset: opts.Offset,
			Limit:  opts.Limit,
		}, nil
	}

	topK := opts.TopK
	if topK <= 0 {
		topK = opts.Limit
	}
	if topK <= 0 {
		topK = cache_domain.DefaultSearchLimit
	}

	hits := index.Search(opts.Vector, topK, opts.MinScore)

	if a.invertedIndex != nil && query != "" {
		textScored := a.invertedIndex.SearchScored(query, cache_domain.WithTermMatch(cache_domain.TermMatchAny))
		if len(textScored) > 0 {
			return a.rrfFusion(hits, textScored, opts.Filters, opts.Offset, opts.Limit)
		}
	}

	return a.buildVectorSearchResult(hits, nil, opts.Filters, opts.Offset, opts.Limit)
}

// vectorQuery performs vector similarity search with filters but no text
// search.
//
// Takes opts (*cache_dto.QueryOptions) which provides the vector, filters,
// and pagination parameters.
//
// Returns SearchResult with items sorted by similarity score (highest first).
func (a *OtterAdapter[K, V]) vectorQuery(opts *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error) {
	vectorField := a.resolveVectorField(opts.VectorField)
	index, ok := a.vectorIndexes[vectorField]
	if !ok {
		return cache_dto.SearchResult[K, V]{
			Offset: opts.Offset,
			Limit:  opts.Limit,
		}, nil
	}

	topK := opts.TopK
	if topK <= 0 {
		topK = opts.Limit
	}
	if topK <= 0 {
		topK = cache_domain.DefaultSearchLimit
	}

	hits := index.Search(opts.Vector, topK, opts.MinScore)

	return a.buildVectorSearchResult(hits, nil, opts.Filters, opts.Offset, opts.Limit)
}

// resolveVectorField returns the explicit vector field name if provided, or
// defaults to the first vector field in the schema.
//
// Takes explicit (string) which is the caller-specified field name (may be
// empty).
//
// Returns string which is the resolved vector field name.
func (a *OtterAdapter[K, V]) resolveVectorField(explicit string) string {
	if explicit != "" {
		return explicit
	}

	for name := range a.vectorIndexes {
		return name
	}
	return ""
}

// buildVectorSearchResult constructs a SearchResult from vector hits, applying
// optional text key intersection, metadata filters, and pagination.
//
// Takes hits ([]cache_domain.VectorHit[K]) which are the vector search results sorted by
// score descending.
// Takes textKeys (map[K]struct{}) which limits results to text-matched keys.
// Nil means no text filtering.
// Takes filters ([]cache_dto.Filter) which are additional metadata filters.
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns SearchResult with scored items.
func (a *OtterAdapter[K, V]) buildVectorSearchResult(
	hits []cache_domain.VectorHit[K],
	textKeys map[K]struct{},
	filters []cache_dto.Filter,
	offset, limit int,
) (cache_dto.SearchResult[K, V], error) {
	items := make([]cache_dto.SearchHit[K, V], 0, len(hits))

	for _, hit := range hits {
		if textKeys != nil {
			if _, ok := textKeys[hit.Key]; !ok {
				continue
			}
		}

		value, ok := a.client.GetIfPresent(hit.Key)
		if !ok {
			continue
		}

		if len(filters) > 0 && a.fieldExtractor != nil && !a.matchesAllFilters(value, filters) {
			continue
		}

		items = append(items, cache_dto.SearchHit[K, V]{
			Key:   hit.Key,
			Value: value,
			Score: float64(hit.Score),
		})
	}

	total := int64(len(items))
	items, limit = cache_domain.ApplyPagination(items, offset, limit)

	return cache_dto.SearchResult[K, V]{
		Items:  items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}
