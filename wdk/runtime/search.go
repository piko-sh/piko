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

//go:build !(js && wasm)

package runtime

import (
	"context"
	"reflect"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/logger"
)

const (
	// maxHydratedSearchResults caps the reflect.Value entries the
	// //piko:link hydration path is willing to build. It guards against
	// a misconfigured backend returning a pathologically long result
	// set.
	maxHydratedSearchResults = 10_000

	// maxHydratedSearchItemBytes caps the JSON size of a single
	// SearchResult item during hydration. Real items are a few KB;
	// the cap exists so a single oversized payload cannot OOM the
	// interpreted render path.
	maxHydratedSearchItemBytes = 1 << 20

	// logAttrTargetType is the attribute key used across
	// hydrateSearchResultsReflect's diagnostic logs; pulled into a
	// constant so the log shape stays consistent.
	logAttrTargetType = "target_type"
)

// SearchCollection performs fuzzy text search on collection data.
// Returns results ranked by relevance score.
//
// This is a facade function that delegates to the bootstrap SearchService. It
// handles:
//  1. Extracting current page data from r.CollectionData (if available)
//  2. Converting search options to domain configuration
//  3. Converting domain search results to typed results
//
// The function automatically detects which mode to use:
//   - Single-page mode: When r.CollectionData is populated (called from
//     collection page)
//   - Full-collection mode: When r.CollectionData is nil (called from search
//     page)
//
// Takes r (*templater_dto.RequestData) which provides the request context.
// Takes collectionName (string) which identifies the collection to search.
// Takes query (string) which is the search text.
// Takes opts (...SearchOption) which provides optional search configuration.
//
// Returns []SearchResult[T] which contains the ranked search results.
// Returns error when the search fails.
//
//piko:link SearchCollectionLink
func SearchCollection[T any](
	r *templater_dto.RequestData,
	collectionName string,
	query string,
	opts ...SearchOption,
) ([]SearchResult[T], error) {
	if query == "" {
		return nil, nil
	}

	searchConfig := applySearchOptions(opts...)

	currentPageData, err := extractCurrentPageData(r)
	if err != nil {
		return nil, err
	}

	domainResults, err := executeCollectionSearch(r.Context(), collectionName, query, currentPageData, searchConfig)
	if err != nil {
		return nil, err
	}

	return convertSearchResults[T](domainResults), nil
}

// QuickSearch performs a simple search returning just the items (no scores).
// Uses default settings: fuzzy threshold 0.3, searches all fields, top 10
// results.
//
// Takes r (*templater_dto.RequestData) which provides the request context.
// Takes collectionName (string) which specifies the collection to search.
// Takes query (string) which contains the search text.
//
// Returns []T which contains the matching items without scores.
// Returns error when the search fails.
//
//piko:link QuickSearchLink
func QuickSearch[T any](r *templater_dto.RequestData, collectionName string, query string) ([]T, error) {
	results, err := SearchCollection[T](r, collectionName, query,
		WithSearchLimit(defaultQuickSearchLimit),
	)
	if err != nil {
		return nil, err
	}

	items := make([]T, len(results))
	for i, result := range results {
		items[i] = result.Item
	}

	return items, nil
}

// extractCurrentPageData extracts the current page data from the request if
// available.
//
// Takes r (*templater_dto.RequestData) which contains the collection data to
// extract from.
//
// Returns map[string]any which contains the page data from the collection.
// Returns error when the collection data is not a map or page data is missing.
func extractCurrentPageData(r *templater_dto.RequestData) (map[string]any, error) {
	if r.CollectionData() == nil {
		return nil, nil
	}

	rootMap, ok := r.CollectionData().(map[string]any)
	if !ok {
		return nil, ErrCollectionDataNotMap
	}

	pageData, exists := rootMap["page"]
	if !exists {
		return nil, ErrNoPageData
	}

	pageMap, ok := pageData.(map[string]any)
	if !ok {
		return nil, ErrPageDataNotMap
	}

	return pageMap, nil
}

// executeCollectionSearch performs the search using the domain search service.
//
// Takes collectionName (string) which identifies the collection to search.
// Takes query (string) which specifies the search query text.
// Takes currentPageData (map[string]any) which provides page context for the
// search.
// Takes searchConfig (searchConfig) which contains search options such as fields,
// limits,
// and thresholds.
//
// Returns []collection_domain.SearchResult which contains the matching results.
// Returns error when the search service cannot be obtained or the search fails.
func executeCollectionSearch(ctx context.Context, collectionName, query string, currentPageData map[string]any, searchConfig searchConfig) ([]collection_domain.SearchResult, error) {
	domainConfig := collection_domain.SearchConfig{
		Query:          query,
		Fields:         convertSearchFields(searchConfig.fields),
		FuzzyThreshold: searchConfig.fuzzyThreshold,
		MinScore:       searchConfig.minScore,
		Limit:          searchConfig.limit,
		Offset:         searchConfig.offset,
		CaseSensitive:  searchConfig.caseSensitive,
	}

	searchService, err := bootstrap.GetSearchService()
	if err != nil {
		return nil, err
	}

	return searchService.Search(ctx, collectionName, currentPageData, domainConfig, searchConfig.searchMode)
}

// SearchCollectionLink is the //piko:link sibling for SearchCollection.
// It mirrors SearchCollection's non-generic logic and builds the
// returned slice via reflect so the interpreter can dispatch
// SearchCollection[T] with a user-defined T that has no compiled
// instantiation.
//
// Takes tType (reflect.Type) which is the instantiated type the user
// wrote inside the brackets.
// Remaining parameters mirror SearchCollection's non-type-parameter
// signature.
//
// Returns a reflect.Value wrapping []SearchResult[T] plus any search
// error. An empty query returns a zero-length slice without error.
func SearchCollectionLink(
	tType reflect.Type,
	r *templater_dto.RequestData,
	collectionName string,
	query string,
	opts ...SearchOption,
) (reflect.Value, error) {
	emptySlice := makeEmptySearchResultSlice(tType)
	if query == "" {
		return emptySlice, nil
	}
	searchConfig := applySearchOptions(opts...)

	currentPageData, err := extractCurrentPageData(r)
	if err != nil {
		return emptySlice, err
	}

	domainResults, err := executeCollectionSearch(r.Context(), collectionName, query, currentPageData, searchConfig)
	if err != nil {
		return emptySlice, err
	}

	return hydrateSearchResultsReflect(r.Context(), domainResults, tType), nil
}

// QuickSearchLink is the //piko:link sibling for QuickSearch. It
// reuses SearchCollectionLink internally and projects the Item field
// out of each SearchResult[T] entry to return a []T.
//
// Takes tType (reflect.Type) which is the instantiated type.
// Remaining parameters mirror QuickSearch's non-type-parameter
// signature.
//
// Returns a reflect.Value wrapping []T plus any search error.
func QuickSearchLink(
	tType reflect.Type,
	r *templater_dto.RequestData,
	collectionName string,
	query string,
) (reflect.Value, error) {
	searchResults, err := SearchCollectionLink(tType, r, collectionName, query,
		WithSearchLimit(defaultQuickSearchLimit),
	)
	if err != nil {
		return reflect.MakeSlice(reflect.SliceOf(tType), 0, 0), err
	}
	items := reflect.MakeSlice(reflect.SliceOf(tType), 0, searchResults.Len())
	for i := range searchResults.Len() {
		entry := searchResults.Index(i)
		itemField := entry.FieldByName("Item")
		if itemField.IsValid() {
			items = reflect.Append(items, itemField)
		}
	}
	return items, nil
}

// searchResultReflectType synthesises the reflect.Type for
// SearchResult[T] matching the interpreter's own structural
// representation. The interpreter's converter skips its sentinel field
// for linked generic types, so this plain reflect.StructOf shape is
// canonical.
//
// Takes tType (reflect.Type) which is the instantiated type argument.
//
// Returns the reflect.Type for SearchResult[T].
func searchResultReflectType(tType reflect.Type) reflect.Type {
	return reflect.StructOf([]reflect.StructField{
		{Name: "FieldScores", Type: reflect.TypeFor[map[string]float64]()},
		{Name: "Item", Type: tType},
		{Name: "Score", Type: reflect.TypeFor[float64]()},
	})
}

// makeEmptySearchResultSlice produces a zero-length
// []SearchResult[T] reflect.Value for error paths and empty queries.
//
// Takes tType (reflect.Type) which is the instantiated type argument.
//
// Returns a zero-length slice of the synthesised SearchResult[T] type.
func makeEmptySearchResultSlice(tType reflect.Type) reflect.Value {
	return reflect.MakeSlice(reflect.SliceOf(searchResultReflectType(tType)), 0, 0)
}

// hydrateSearchResultsReflect converts domain search results into a
// reflect.Value of type []SearchResult[T] via JSON round-trip.
//
// Truncates to maxHydratedSearchResults to prevent a runaway backend
// from forcing an unbounded reflect allocation in the interpreted
// hot path.
//
// Takes domainResults ([]collection_domain.SearchResult) which are the
// raw hits from the search service.
// Takes tType (reflect.Type) which is the instantiated type argument.
//
// Returns a reflect.Value wrapping []SearchResult[T].
func hydrateSearchResultsReflect(ctx context.Context, domainResults []collection_domain.SearchResult, tType reflect.Type) reflect.Value {
	_, l := logger.From(ctx, log)
	if len(domainResults) > maxHydratedSearchResults {
		l.Warn("Truncating search results at hard cap",
			logger.String(logAttrTargetType, tType.String()),
			logger.Int("returned", len(domainResults)),
			logger.Int("cap", maxHydratedSearchResults))
		domainResults = domainResults[:maxHydratedSearchResults]
	}
	searchResultType := searchResultReflectType(tType)
	slice := reflect.MakeSlice(reflect.SliceOf(searchResultType), 0, len(domainResults))
	for index, domainResult := range domainResults {
		itemPtr := reflect.New(tType)
		rawBytes, err := json.Marshal(domainResult.Item)
		if err != nil {
			l.Warn("Marshalling search item failed; skipping",
				logger.String(logAttrTargetType, tType.String()),
				logger.Int("index", index),
				logger.Error(err))
			continue
		}
		if len(rawBytes) > maxHydratedSearchItemBytes {
			l.Warn("Search item exceeds size limit; skipping",
				logger.String(logAttrTargetType, tType.String()),
				logger.Int("index", index),
				logger.Int("encoded_bytes", len(rawBytes)),
				logger.Int("limit_bytes", maxHydratedSearchItemBytes))
			continue
		}
		if err := json.Unmarshal(rawBytes, itemPtr.Interface()); err != nil {
			l.Warn("Unmarshalling search item failed; skipping",
				logger.String(logAttrTargetType, tType.String()),
				logger.Int("index", index),
				logger.Error(err))
			continue
		}
		entry := reflect.New(searchResultType).Elem()
		entry.FieldByName("FieldScores").Set(reflect.ValueOf(domainResult.FieldScores))
		entry.FieldByName("Item").Set(itemPtr.Elem())
		entry.FieldByName("Score").Set(reflect.ValueOf(domainResult.Score))
		slice = reflect.Append(slice, entry)
	}
	return slice
}

// convertSearchResults converts domain search results to typed facade results.
//
// Takes domainResults ([]collection_domain.SearchResult) which contains the
// domain-level search results to convert.
//
// Returns []SearchResult[T] which contains the typed facade results.
func convertSearchResults[T any](domainResults []collection_domain.SearchResult) []SearchResult[T] {
	results := make([]SearchResult[T], 0, len(domainResults))
	for _, domainResult := range domainResults {
		var item T
		if err := collection_domain.ConvertSearchResultToType(domainResult.Item, &item); err != nil {
			continue
		}

		results = append(results, SearchResult[T]{
			Item:        item,
			Score:       domainResult.Score,
			FieldScores: domainResult.FieldScores,
		})
	}
	return results
}

// convertSearchFields converts runtime SearchField values to search_dto
// SearchField values.
//
// Takes fields ([]SearchField) which specifies the fields to convert.
//
// Returns []search_dto.SearchField which contains the converted fields.
func convertSearchFields(fields []SearchField) []search_dto.SearchField {
	result := make([]search_dto.SearchField, len(fields))
	for i, f := range fields {
		result[i] = search_dto.SearchField{
			Name:   f.Name,
			Weight: f.Weight,
		}
	}
	return result
}
