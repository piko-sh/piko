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
	"fmt"
	"slices"
	"sync"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

const (
	// inlineSearchFields is the number of sortable or vector fields stored without
	// allocation.
	inlineSearchFields = 2

	// defaultSearchLimit is the default number of results to return when no limit is
	// specified.
	defaultSearchLimit = 10

	// keyWithValuePoolMaxCap is the maximum capacity for pooled slices.
	keyWithValuePoolMaxCap = 10000

	// initialKeyWithValueCapacity is the starting capacity for pooled keyWithValue slices.
	initialKeyWithValueCapacity = 128
)

// searchEntry holds the values to commit to the search indexes under the bucket lock.
type searchEntry struct {
	// terms holds the value's analysed text terms, each once.
	terms []string

	// sortable holds the value of each sortable field, in the order of the adapter's
	// sortedFields.
	sortable []sortableValue

	// vectors holds the vector of each vector field, in the order of the adapter's
	// vectorFields. An empty vector means the value has none.
	vectors [][]float32

	// sortableBuffer backs sortable when the schema has few sortable fields, so the entry
	// needs no allocation of its own for them.
	sortableBuffer [inlineSearchFields]sortableValue

	// vectorBuffer backs vectors when the schema has few vector fields, so the entry needs
	// no allocation of its own for them.
	vectorBuffer [inlineSearchFields][]float32
}

// sortableValue is the value of one sortable field.
type sortableValue struct {
	// value is the field's value, valid when present.
	value any

	// present reports whether the value has the field.
	present bool
}

// sortedField pairs a sortable field with its index, so a write walks a slice rather than
// a map.
type sortedField[K comparable] struct {
	// index is the field's sorted index.
	index *SortedIndex[K]

	// name is the field's name in the schema.
	name string
}

// vectorField pairs a vector field with its index, so a write walks a slice rather than a
// map.
type vectorField[K comparable] struct {
	// index is the field's vector index.
	index *VectorIndex[K]

	// name is the field's name in the schema.
	name string
}

// keyWithValue holds a key and its extracted field value for sorting.
type keyWithValue[K comparable] struct {
	// key is the map key for this entry.
	key K

	// value holds the parsed value for this key.
	value any
}

var (
	// keyWithValuePool reuses keyWithValue slices to reduce allocation pressure during
	// search result sorting.
	keyWithValuePool = sync.Pool{
		New: func() any {
			return new(make([]keyWithValue[any], 0, 128))
		},
	}
)

// indexDocument brings the search indexes in line with a value stored for key.
//
// Takes key (K) which identifies the document in the indexes.
// Takes value (V) which is the document to extract fields from for indexing.
func (a *OtterAdapter[K, V]) indexDocument(key K, value V) {
	if entry := a.prepareSearchEntry(value); entry != nil {
		a.commitSearchEntry(key, entry)
	}
}

// prepareSearchEntry builds the search entry of a write whose value is known before
// otter's bucket lock is taken.
//
// Takes value (V) which is the value the write stores.
//
// Returns *searchEntry which is the value's entry, or nil when the cache keeps no search
// indexes, so a cache without them carries nothing extra on its writes.
func (a *OtterAdapter[K, V]) prepareSearchEntry(value V) *searchEntry {
	if !a.indexesSearch() {
		return nil
	}
	entry := new(searchEntry)
	a.fillSearchEntry(entry, value)
	return entry
}

// indexesSearch reports whether the cache keeps search indexes.
//
// Returns bool which is true when a search schema is configured.
func (a *OtterAdapter[K, V]) indexesSearch() bool {
	return a.schema != nil && a.fieldExtractor != nil
}

// fillSearchEntry derives text terms and field values before taking the bucket lock.
//
// Takes entry (*searchEntry) which is an empty entry to fill. Its slices may point into
// its own buffers, so it must not be copied once filled.
// Takes value (V) which is the document to extract fields from.
//
// Must only be called when the cache keeps search indexes.
func (a *OtterAdapter[K, V]) fillSearchEntry(entry *searchEntry, value V) {
	if a.invertedIndex != nil && len(a.fieldExtractor.textFields) > 0 {
		entry.terms = a.invertedIndex.tokeniseAll(a.fieldExtractor.ExtractTextFields(value))
	}
	if count := len(a.sortedFields); count > 0 {
		entry.sortable = entry.sortableBuffer[:0]
		if count > inlineSearchFields {
			entry.sortable = make([]sortableValue, 0, count)
		}
		for _, field := range a.sortedFields {
			sortable, present := a.fieldExtractor.ExtractSortableValue(value, field.name)
			entry.sortable = append(entry.sortable, sortableValue{value: sortable, present: present})
		}
	}
	if count := len(a.vectorFields); count > 0 {
		entry.vectors = entry.vectorBuffer[:0]
		if count > inlineSearchFields {
			entry.vectors = make([][]float32, 0, count)
		}
		for _, field := range a.vectorFields {
			vector, _ := a.fieldExtractor.ExtractVectorValue(value, field.name)
			entry.vectors = append(entry.vectors, vector)
		}
	}
}

// commitSearchEntry updates the search indexes for key and entry.
//
// Takes key (K) which identifies the document.
// Takes entry (*searchEntry) which is what the stored value contributes.
//
// Must only be called when the cache keeps search indexes.
func (a *OtterAdapter[K, V]) commitSearchEntry(key K, entry *searchEntry) {
	if a.invertedIndex != nil && len(a.fieldExtractor.textFields) > 0 {
		a.invertedIndex.replaceTerms(key, entry.terms)
	}
	for position, field := range a.sortedFields {
		if sortable := entry.sortable[position]; sortable.present {
			field.index.Add(key, sortable.value)
		} else {
			field.index.Remove(key)
		}
	}
	for position, field := range a.vectorFields {
		field.index.Replace(key, entry.vectors[position])
	}
}

// removeFromSearchIndex removes a key from all search indexes.
//
// Takes key (K) which is the key to remove from the indexes.
func (a *OtterAdapter[K, V]) removeFromSearchIndex(key K) {
	if a.invertedIndex != nil {
		a.invertedIndex.Remove(key)
	}
	for _, field := range a.sortedFields {
		field.index.Remove(key)
	}
	for _, field := range a.vectorFields {
		field.index.Remove(key)
	}
}

// clearAllIndexes resets all secondary indexes (tags, inverted, sorted, vector).
func (a *OtterAdapter[K, V]) clearAllIndexes() {
	a.tagIndex.Clear()
	if a.invertedIndex != nil {
		a.invertedIndex.Clear()
	}
	for _, field := range a.sortedFields {
		field.index.Clear()
	}
	for _, field := range a.vectorFields {
		field.index.Clear()
	}
}

// Search performs full-text search across indexed TEXT fields.
//
// Takes query (string) which is the search query to tokenise and match.
// Takes opts (*cache_dto.SearchOptions) which configures pagination, sorting, and
// filters.
//
// Returns cache_dto.SearchResult[K, V] which contains matched entries with metadata.
// Returns error when no schema is configured (ErrSearchNotSupported), or when a vector
// search is given a query vector the index cannot compare (ErrInvalidQueryVector).
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
	var candidatesAreWholeCache bool

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
				Items:      nil,
				Total:      0,
				Offset:     opts.Offset,
				Limit:      opts.Limit,
				Candidates: 0,
			}, nil
		}
		candidateKeys = a.getAllKeys()
		keysAreTrusted = false
		candidatesAreWholeCache = true
	}

	if len(candidateKeys) == 0 {
		return cache_dto.SearchResult[K, V]{
			Items:      nil,
			Total:      0,
			Offset:     opts.Offset,
			Limit:      opts.Limit,
			Candidates: 0,
		}, nil
	}

	filteredKeys := a.applyFilters(candidateKeys, opts.Filters, keysAreTrusted, candidatesAreWholeCache)

	sortedKeys := a.sortKeys(filteredKeys, opts.SortBy, opts.SortOrder)

	return a.buildSearchResultWithScores(sortedKeys, keyScores, opts.Offset, opts.Limit)
}

// Query performs structured filtering, sorting, and pagination without full-text search.
//
// Takes opts (*cache_dto.QueryOptions) which specifies filters, sorting, and pagination.
//
// Returns cache_dto.SearchResult[K, V] which contains matched entries.
// Returns error when no schema is configured (ErrSearchNotSupported), or when a vector
// query is given a query vector the index cannot compare (ErrInvalidQueryVector).
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
			Items:      nil,
			Total:      0,
			Offset:     opts.Offset,
			Limit:      opts.Limit,
			Candidates: 0,
		}, nil
	}

	filteredKeys := a.applyFilters(candidateKeys, opts.Filters, false, true)

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

// applyFilters narrows a candidate set, using an index when one can serve a filter.
//
// Takes keys ([]K) which specifies the candidate keys.
// Takes filters ([]cache_dto.Filter) which provides the filter conditions.
// Takes trustKeys (bool) which indicates whether to skip existence validation.
// Takes candidatesAreWholeCache (bool) which reports that keys covers every live entry,
// so an index lookup answers the same question as scanning them.
//
// Returns []K which contains the keys matching every filter.
func (a *OtterAdapter[K, V]) applyFilters(
	keys []K,
	filters []cache_dto.Filter,
	trustKeys bool,
	candidatesAreWholeCache bool,
) []K {
	if len(filters) == 0 || a.fieldExtractor == nil {
		return keys
	}

	remaining := filters
	if candidatesAreWholeCache {
		for i, filter := range filters {
			optimisedKeys := a.tryRangeQueryFilter(filter)
			if optimisedKeys == nil {
				continue
			}

			keys = optimisedKeys
			trustKeys = false
			remaining = make([]cache_dto.Filter, 0, len(filters)-1)
			remaining = append(remaining, filters[:i]...)
			remaining = append(remaining, filters[i+1:]...)

			break
		}
	}

	if len(remaining) == 0 {
		return keys
	}

	result := make([]K, 0, len(keys))
	for _, key := range keys {
		value, ok := a.client.GetIfPresent(key)
		if !ok && !trustKeys {
			continue
		}

		if a.matchesAllFilters(value, remaining) {
			result = append(result, key)
		}
	}
	return result
}

// tryRangeQueryFilter attempts to use B-tree range queries for efficient filtering.
//
// Takes filter (cache_dto.Filter) which specifies the field, operation, and value(s) to
// filter by.
//
// Returns []K which contains the matching keys, or nil if the optimisation cannot be
// applied.
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
	case cache_dto.FilterOpEq:
		return index.KeysBetween(filter.Value, filter.Value, true)
	case cache_dto.FilterOpIn:
		return unionOfEqualityProbes(index, filter.Values)
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
	for _, filter := range filters {
		if !a.matchesFilter(value, filter) {
			return false
		}
	}
	return true
}

// matchesFilter checks if a value matches a single filter condition.
//
// Takes value (V) which is the value to check against the filter.
// Takes filter (cache_dto.Filter) which specifies the filter condition to apply.
//
// Returns bool which is true if the value matches the filter condition.
func (a *OtterAdapter[K, V]) matchesFilter(value V, filter cache_dto.Filter) bool {
	if matched, ok := a.fieldExtractor.compareFieldDirect(value, filter.Field, filter.Operation, filter.Value, filter.Values); ok {
		return matched
	}

	fieldVal, ok := a.fieldExtractor.extractAny(value, filter.Field)
	if !ok {
		return false
	}

	switch filter.Operation {
	case cache_dto.FilterOpEq:
		return a.compareEqual(fieldVal, filter.Value)
	case cache_dto.FilterOpNe:
		return !a.compareEqual(fieldVal, filter.Value)
	case cache_dto.FilterOpGt:
		return a.compareNumeric(fieldVal, filter.Value) > 0
	case cache_dto.FilterOpGe:
		return a.compareNumeric(fieldVal, filter.Value) >= 0
	case cache_dto.FilterOpLt:
		return a.compareNumeric(fieldVal, filter.Value) < 0
	case cache_dto.FilterOpLe:
		return a.compareNumeric(fieldVal, filter.Value) <= 0
	case cache_dto.FilterOpIn:
		return a.matchesIn(fieldVal, filter.Values)
	case cache_dto.FilterOpBetween:
		if len(filter.Values) != 2 {
			return false
		}
		cmpMin := a.compareNumeric(fieldVal, filter.Values[0])
		cmpMax := a.compareNumeric(fieldVal, filter.Values[1])
		return cmpMin >= 0 && cmpMax <= 0
	case cache_dto.FilterOpPrefix:
		return a.matchesPrefix(fieldVal, filter.Value)
	default:
		return false
	}
}

// compareEqual checks if two values are equal.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns bool which is true if the values are equal by direct comparison or by their
// string form.
func (*OtterAdapter[K, V]) compareEqual(a, b any) bool {
	if a == b {
		return true
	}
	return toString(a) == toString(b)
}

// compareNumeric compares two values as numbers.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns int which is -1, 0, or 1 showing whether a is less than, equal to, or greater
// than b. Falls back to string comparison if values cannot be converted to numbers.
func (*OtterAdapter[K, V]) compareNumeric(a, b any) int {
	aNum, aOk := toFloat64(a)
	bNum, bOk := toFloat64(b)
	if aOk && bOk {
		switch {
		case aNum < bNum:
			return -1
		case aNum > bNum:
			return 1
		default:
			return 0
		}
	}
	aString := toString(a)
	bString := toString(b)
	switch {
	case aString < bString:
		return -1
	case aString > bString:
		return 1
	default:
		return 0
	}
}

// matchesIn checks if fieldVal matches any value in the set.
//
// Takes fieldVal (any) which is the value to search for.
// Takes values ([]any) which is the set of values to match against.
//
// Returns bool which is true if fieldVal equals any value in the set.
func (a *OtterAdapter[K, V]) matchesIn(fieldVal any, values []any) bool {
	for _, v := range values {
		if a.compareEqual(fieldVal, v) {
			return true
		}
	}
	return false
}

// matchesPrefix checks if fieldVal starts with the prefix.
//
// Takes fieldVal (any) which is the value to check.
// Takes prefix (any) which is the prefix to match against.
//
// Returns bool which is true if fieldVal starts with prefix.
func (*OtterAdapter[K, V]) matchesPrefix(fieldVal, prefix any) bool {
	fieldString := toString(fieldVal)
	prefixString := toString(prefix)
	return len(fieldString) >= len(prefixString) && fieldString[:len(prefixString)] == prefixString
}

// sortKeys sorts keys by the specified field and order.
//
// Takes keys ([]K) which contains the keys to sort.
// Takes sortBy (string) which specifies the field name to sort by.
// Takes sortOrder (cache_dto.SortOrder) which specifies ascending or descending.
//
// Returns []K which contains the sorted keys, or the original keys if sortBy is empty or
// keys is empty.
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
		fieldValue, _ := a.fieldExtractor.extractAny(value, fieldName)
		items = append(items, keyWithValue[any]{key: key, value: fieldValue})
	}

	slices.SortFunc(items, func(itemA, itemB keyWithValue[any]) int {
		cmp := a.compareNumeric(itemA.value, itemB.value)
		if !ascending {
			cmp = -cmp
		}
		return cmp
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

// buildSearchResult creates a SearchResult with pagination applied and flat scoring (1.0
// for all hits).
//
// Takes keys ([]K) which contains the matched keys in display order.
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns cache_dto.SearchResult[K, V] which contains the paginated results with flat
// scores.
// Returns error which is always nil.
func (a *OtterAdapter[K, V]) buildSearchResult(keys []K, offset, limit int) (cache_dto.SearchResult[K, V], error) {
	return a.buildSearchResultWithScores(keys, nil, offset, limit)
}

// buildSearchResultWithScores creates a SearchResult with pagination applied.
//
// Takes keys ([]K) which contains the matched keys in display order.
// Takes scores (map[K]float64) which maps keys to relevance scores (may be nil).
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns cache_dto.SearchResult[K, V] which contains the live hits and pagination
// metadata.
// Returns error which is nil.
func (a *OtterAdapter[K, V]) buildSearchResultWithScores(keys []K, scores map[K]float64, offset, limit int) (cache_dto.SearchResult[K, V], error) {
	total := int64(len(keys))
	keys, limit = applyPagination(keys, offset, limit)

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
			Key:        key,
			Value:      value,
			Score:      score,
			Highlights: nil,
		})
	}

	return cache_dto.SearchResult[K, V]{
		Items:      items,
		Total:      total,
		Offset:     offset,
		Limit:      limit,
		Candidates: 0,
	}, nil
}

// vectorSearch performs vector similarity search with optional text fusion.
//
// Takes query (string) which is an optional text query for hybrid search.
// Takes opts (*cache_dto.SearchOptions) which provides the vector, filters, and
// pagination parameters.
//
// Returns cache_dto.SearchResult[K, V] which holds hits in descending score order.
// Returns error which wraps cache_domain.ErrInvalidQueryVector when the query vector
// cannot be compared with the indexed vectors.
func (a *OtterAdapter[K, V]) vectorSearch(query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	vectorField := a.resolveVectorField(opts.VectorField)
	index, ok := a.vectorIndexes[vectorField]
	if !ok {
		return cache_dto.SearchResult[K, V]{
			Offset:     opts.Offset,
			Limit:      opts.Limit,
			Items:      nil,
			Total:      0,
			Candidates: 0,
		}, nil
	}

	topK, pageSize := opts.VectorWindow(defaultSearchLimit)

	hits, err := index.Search(opts.Vector, topK, opts.MinScore)
	if err != nil {
		return cache_dto.SearchResult[K, V]{}, fmt.Errorf("searching vector field %q: %w", vectorField, err)
	}

	if a.invertedIndex != nil && query != "" {
		textScored := a.invertedIndex.SearchScored(query, WithTermMatch(TermMatchAny))
		if len(textScored) > 0 {
			return a.rrfFusion(hits, textScored, opts.Filters, opts.Offset, pageSize)
		}
	}

	return a.buildVectorSearchResult(hits, opts.Filters, opts.Offset, pageSize)
}

// vectorQuery performs vector similarity search with filters but no text search.
//
// Takes opts (*cache_dto.QueryOptions) which provides the vector, filters, and pagination
// parameters.
//
// Returns cache_dto.SearchResult[K, V] which holds hits in descending score order.
// Returns error which wraps cache_domain.ErrInvalidQueryVector when the query vector
// cannot be compared with the indexed vectors.
func (a *OtterAdapter[K, V]) vectorQuery(opts *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error) {
	vectorField := a.resolveVectorField(opts.VectorField)
	index, ok := a.vectorIndexes[vectorField]
	if !ok {
		return cache_dto.SearchResult[K, V]{
			Offset:     opts.Offset,
			Limit:      opts.Limit,
			Items:      nil,
			Total:      0,
			Candidates: 0,
		}, nil
	}

	topK, pageSize := opts.VectorWindow(defaultSearchLimit)

	hits, err := index.Search(opts.Vector, topK, opts.MinScore)
	if err != nil {
		return cache_dto.SearchResult[K, V]{}, fmt.Errorf("querying vector field %q: %w", vectorField, err)
	}

	return a.buildVectorSearchResult(hits, opts.Filters, opts.Offset, pageSize)
}

// resolveVectorField returns the explicit vector field name if provided, or defaults to
// the first vector field by name, so the default is the same on every call.
//
// Takes explicit (string) which is the caller-specified field name (may be empty).
//
// Returns string which is the resolved vector field name.
func (a *OtterAdapter[K, V]) resolveVectorField(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if len(a.vectorFields) == 0 {
		return ""
	}
	return a.vectorFields[0].name
}

// buildVectorSearchResult selects a page of live vector hits that match the filters.
//
// Takes hits ([]VectorHit[K]) which contains the hits in descending similarity order.
// Takes filters ([]cache_dto.Filter) which are the metadata conditions each hit must
// satisfy.
// Takes offset (int) which is the number of eligible hits to skip.
// Takes limit (int) which is the page size, with NoLimit selecting every hit.
//
// Returns cache_dto.SearchResult[K, V] which contains the page and total eligible hit
// count.
// Returns error which is nil.
func (a *OtterAdapter[K, V]) buildVectorSearchResult(hits []VectorHit[K], filters []cache_dto.Filter, offset, limit int) (cache_dto.SearchResult[K, V], error) {
	return a.rankedPage(len(hits), len(hits), func(i int) (K, float64) {
		return hits[i].Key, float64(hits[i].Score)
	}, filters, offset, limit), nil
}

// rankedPage counts eligible hits and retains the requested page.
//
// Takes count (int) which is the number of ranked hits.
// Takes candidates (int) which is the number of KNN candidates the vector index returned,
// reported as the result's candidate count.
// Takes ranked (func(int) (K, float64)) which supplies each hit in display order.
// Takes filters ([]cache_dto.Filter) which are the metadata conditions each hit must
// satisfy.
// Takes offset (int) which is the number of eligible hits to skip.
// Takes limit (int) which is the page size, with NoLimit selecting every hit.
//
// Returns cache_dto.SearchResult[K, V] which contains live matching hits and their total
// count.
func (a *OtterAdapter[K, V]) rankedPage(
	count, candidates int,
	ranked func(int) (K, float64),
	filters []cache_dto.Filter,
	offset, limit int,
) cache_dto.SearchResult[K, V] {
	switch {
	case limit == cache_dto.NoLimit:
		limit = count
	case limit <= 0:
		limit = defaultSearchLimit
	}
	items := make([]cache_dto.SearchHit[K, V], 0, min(count, limit))
	total := 0
	for i := range count {
		key, score := ranked(i)
		value, ok := a.client.GetIfPresent(key)
		if !ok || (len(filters) > 0 && a.fieldExtractor != nil && !a.matchesAllFilters(value, filters)) {
			continue
		}
		total++
		if total > offset && len(items) < limit {
			items = append(items, cache_dto.SearchHit[K, V]{Key: key, Value: value, Score: score, Highlights: nil})
		}
	}
	return cache_dto.SearchResult[K, V]{
		Items:      items,
		Total:      int64(total),
		Offset:     offset,
		Limit:      limit,
		Candidates: candidates,
	}
}

// applyPagination applies offset and limit to a slice, returning the paginated slice and
// the resolved limit.
//
// Takes items ([]T) which is the slice to paginate.
// Takes offset (int) which specifies the number of items to skip.
// Takes limit (int) which specifies the maximum number of items to return.
//
// Returns []T which is the paginated slice.
// Returns int which is the resolved limit after applying constraints.
func applyPagination[T any](items []T, offset, limit int) ([]T, int) {
	if offset > 0 {
		if offset >= len(items) {
			items = nil
		} else {
			items = items[offset:]
		}
	}

	if limit == cache_dto.NoLimit {
		return items, len(items)
	}

	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > len(items) {
		limit = len(items)
	}
	items = items[:limit]

	return items, limit
}

// unionOfEqualityProbes collects the keys matching any of the given values.
//
// Takes index (*SortedIndex[K]) which serves the probes.
// Takes values ([]any) which are the values to match.
//
// Returns []K which contains the deduplicated union, or nil when no values were given.
func unionOfEqualityProbes[K comparable](index *SortedIndex[K], values []any) []K {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[K]struct{})
	union := make([]K, 0)

	for _, value := range values {
		for _, key := range index.KeysBetween(value, value, true) {
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			union = append(union, key)
		}
	}

	return union
}
