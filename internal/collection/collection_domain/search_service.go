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

package collection_domain

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/search/search_adapters"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
)

const (
	// defaultFuzzySimilarityThreshold is the minimum Jaro-Winkler similarity score
	// (85%) required for fuzzy matching results.
	defaultFuzzySimilarityThreshold = 0.85

	// defaultFuzzyMaxResults limits fuzzy matching to the top 3 results.
	defaultFuzzyMaxResults = 3
)

var (
	_ SearchServicePort = (*searchService)(nil)

	_ SearchIndexLoaderPort = (*defaultSearchIndexLoader)(nil)

	_ CollectionItemsLoaderPort = (*defaultCollectionItemsLoader)(nil)
)

// SearchResult represents a single search result with its score and field-level
// scores.
type SearchResult struct {
	// Item is the matched document as a raw map for flexible type conversion.
	Item map[string]any

	// FieldScores maps field names to their match scores.
	FieldScores map[string]float64

	// Score is the overall BM25 relevance score (unbounded,
	// higher is more relevant).
	Score float64
}

// SearchConfig holds the settings for searching a collection.
type SearchConfig struct {
	// Query is the search text to match against indexed fields.
	Query string

	// Fields specifies which fields to search and their relative weights.
	Fields []search_dto.SearchField

	// FuzzyThreshold controls fuzzy matching tolerance from 0.0 to 1.0.
	// Lower values are more strict; higher values allow more typos.
	FuzzyThreshold float64

	// MinScore filters out results below this BM25 score
	// threshold; 0 means no filtering.
	MinScore float64

	// Limit sets the maximum number of results to return; 0 means no limit.
	Limit int

	// Offset skips the first N results for pagination; 0 starts from the
	// beginning.
	Offset int

	// CaseSensitive determines whether matching is case-sensitive.
	CaseSensitive bool
}

// searchService implements SearchServicePort with parts that can be swapped.
type searchService struct {
	// indexLoader loads pre-built search indexes for collections.
	indexLoader SearchIndexLoaderPort

	// itemsLoader fetches collection items for adding data to search results.
	itemsLoader CollectionItemsLoaderPort

	// queryFactory creates query processors for search operations.
	queryFactory queryProcessorFactory

	// scorerFactory creates Scorer instances for ranking search results.
	scorerFactory scorerFactory
}

// queryProcessorFactory creates query processors for search indexes.
type queryProcessorFactory func(reader search_domain.IndexReaderPort) search_domain.QueryProcessorPort

// scorerFactory creates scorers for search operations.
type scorerFactory func() search_domain.ScorerPort

// searchServiceOption is a functional option for configuring a searchService.
type searchServiceOption func(*searchService)

// Search implements SearchServicePort.
//
// Takes collectionName (string) which identifies the collection to search.
// Takes currentPageData (map[string]any) which provides context for hydrating
// results.
// Takes config (SearchConfig) which specifies the search query and options.
// Takes searchMode (string) which determines which search index to use.
//
// Returns []SearchResult which contains the matching documents with scores.
// Returns error when the search index cannot be loaded or the query fails.
func (s *searchService) Search(
	ctx context.Context,
	collectionName string,
	currentPageData map[string]any,
	config SearchConfig,
	searchMode string,
) ([]SearchResult, error) {
	if config.Query == "" {
		return nil, nil
	}

	readerAny, err := s.indexLoader.GetIndex(collectionName, searchMode)
	if err != nil {
		return nil, fmt.Errorf("loading search index for collection %q: %w", collectionName, err)
	}

	reader, ok := readerAny.(search_domain.IndexReaderPort)
	if !ok {
		return nil, fmt.Errorf("invalid index reader type for collection %q", collectionName)
	}

	queryProcessor := s.queryFactory(reader)
	scorer := s.scorerFactory()

	searchConfig := buildSearchDTO(config)

	queryResults, err := queryProcessor.Search(ctx, config.Query, reader, scorer, searchConfig)
	if err != nil {
		return nil, fmt.Errorf("executing search query: %w", err)
	}

	allItems, err := s.itemsLoader.GetAllItems(collectionName)
	if err != nil {
		return nil, fmt.Errorf("fetching collection items for hydration: %w", err)
	}

	return hydrateSearchResults(queryResults, allItems, currentPageData), nil
}

// defaultSearchIndexLoader implements SearchIndexLoaderPort using search
// adapters.
type defaultSearchIndexLoader struct{}

// GetIndex retrieves a search index for the given collection and mode.
// Implements SearchIndexLoaderPort.
//
// Takes collectionName (string) which identifies the collection to search.
// Takes searchMode (string) which specifies the search strategy to use.
//
// Returns any which is the search index for the collection.
// Returns error when the index cannot be retrieved.
func (*defaultSearchIndexLoader) GetIndex(collectionName, searchMode string) (any, error) {
	return search_adapters.GetSearchIndex(collectionName, searchMode)
}

// defaultCollectionItemsLoader implements CollectionItemsLoaderPort using
// global functions.
type defaultCollectionItemsLoader struct{}

// GetAllItems retrieves all items from the named collection.
// Implements CollectionItemsLoaderPort.
//
// Takes collectionName (string) which identifies the collection to retrieve.
//
// Returns []map[string]any which contains the items from the collection.
// Returns error when the collection cannot be found or read.
func (*defaultCollectionItemsLoader) GetAllItems(collectionName string) ([]map[string]any, error) {
	return GetStaticCollectionItems(collectionName)
}

// NewSearchService creates a new search service with the given options.
//
// Default implementations are used for any dependencies not explicitly
// provided.
//
// Takes opts (...searchServiceOption) which configures the service behaviour.
//
// Returns SearchServicePort which is the configured search service.
func NewSearchService(opts ...searchServiceOption) SearchServicePort {
	s := &searchService{
		indexLoader:   &defaultSearchIndexLoader{},
		itemsLoader:   &defaultCollectionItemsLoader{},
		queryFactory:  defaultQueryProcessorFactory,
		scorerFactory: defaultScorerFactory,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// ConvertSearchResultToType converts a raw map from a search result into a
// typed struct.
//
// This helper is used by the runtime facade to convert domain-layer search
// results into typed results for the caller.
//
// Takes item (map[string]any) which is the raw map from a search result.
// Takes target (any) which is a pointer to the struct to fill.
//
// Returns error when JSON marshalling or unmarshalling fails.
func ConvertSearchResultToType(item map[string]any, target any) error {
	bytes, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshalling search result: %w", err)
	}

	if err := json.Unmarshal(bytes, target); err != nil {
		return fmt.Errorf("unmarshalling to target type: %w", err)
	}

	return nil
}

// withIndexLoader sets a custom index loader for the search service.
//
// Takes loader (SearchIndexLoaderPort) which provides the index loading
// behaviour.
//
// Returns searchServiceOption which sets up the search service to use the
// given loader.
func withIndexLoader(loader SearchIndexLoaderPort) searchServiceOption {
	return func(s *searchService) {
		s.indexLoader = loader
	}
}

// withItemsLoader sets a custom items loader for the search service.
//
// Takes loader (CollectionItemsLoaderPort) which provides the items loading
// behaviour.
//
// Returns searchServiceOption which configures the service with the given
// loader.
func withItemsLoader(loader CollectionItemsLoaderPort) searchServiceOption {
	return func(s *searchService) {
		s.itemsLoader = loader
	}
}

// withQueryProcessorFactory sets a custom query processor factory.
//
// Takes factory (queryProcessorFactory) which creates query processors for
// search operations.
//
// Returns searchServiceOption which configures the search service to use the
// given factory.
func withQueryProcessorFactory(factory queryProcessorFactory) searchServiceOption {
	return func(s *searchService) {
		s.queryFactory = factory
	}
}

// withScorerFactory sets a custom scorer factory for the search service.
//
// Takes factory (scorerFactory) which provides the scoring logic.
//
// Returns searchServiceOption which configures the search service.
func withScorerFactory(factory scorerFactory) searchServiceOption {
	return func(s *searchService) {
		s.scorerFactory = factory
	}
}

// buildSearchDTO converts a SearchConfig to its DTO form for search operations.
//
// Takes config (SearchConfig) which specifies the search parameters.
//
// Returns search_dto.SearchConfig which is the DTO with fuzzy matching enabled.
func buildSearchDTO(config SearchConfig) search_dto.SearchConfig {
	return search_dto.SearchConfig{
		Query:                    config.Query,
		Fields:                   config.Fields,
		FuzzyThreshold:           config.FuzzyThreshold,
		Limit:                    config.Limit,
		Offset:                   config.Offset,
		MinScore:                 config.MinScore,
		CaseSensitive:            config.CaseSensitive,
		EnableFuzzyFallback:      true,
		FuzzySimilarityThreshold: defaultFuzzySimilarityThreshold,
		FuzzyMaxResults:          defaultFuzzyMaxResults,
	}
}

// defaultQueryProcessorFactory creates a QueryProcessor for the given index
// reader.
//
// Takes reader (search_domain.IndexReaderPort) which provides access to the
// search index.
//
// Returns search_domain.QueryProcessorPort which processes queries against
// the index.
func defaultQueryProcessorFactory(reader search_domain.IndexReaderPort) search_domain.QueryProcessorPort {
	return search_domain.NewQueryProcessorForIndex(reader)
}

// defaultScorerFactory creates a BM25 scorer with default settings.
//
// Returns search_domain.ScorerPort which is a scorer using the default BM25
// settings.
func defaultScorerFactory() search_domain.ScorerPort {
	return search_domain.NewBM25Scorer(0, 0)
}

// newDefaultSearchIndexLoader creates a SearchIndexLoaderPort using the
// default search adapters.
//
// Returns SearchIndexLoaderPort which provides the standard search index
// loading behaviour.
func newDefaultSearchIndexLoader() SearchIndexLoaderPort {
	return &defaultSearchIndexLoader{}
}

// newDefaultCollectionItemsLoader creates a CollectionItemsLoaderPort that
// uses the default global functions for loading.
//
// Returns CollectionItemsLoaderPort which provides collection loading using
// the standard global implementations.
func newDefaultCollectionItemsLoader() CollectionItemsLoaderPort {
	return &defaultCollectionItemsLoader{}
}

// hydrateSearchResults converts query results to SearchResult values by
// looking up the matching document data.
//
// Takes queryResults ([]search_domain.QueryResult) which contains the raw
// search results to hydrate.
// Takes allItems ([]map[string]any) which provides the document data indexed
// by DocumentID.
// Takes targetPageData (map[string]any) which filters results to a single
// page when not nil.
//
// Returns []SearchResult which contains the hydrated results with document
// data attached.
func hydrateSearchResults(
	queryResults []search_domain.QueryResult,
	allItems []map[string]any,
	targetPageData map[string]any,
) []SearchResult {
	results := make([]SearchResult, 0, len(queryResults))

	for _, queryResult := range queryResults {
		if int(queryResult.DocumentID) >= len(allItems) {
			continue
		}

		itemMap := allItems[queryResult.DocumentID]

		if targetPageData != nil {
			if !matchesTargetPage(itemMap, targetPageData) {
				continue
			}
		}

		results = append(results, SearchResult{
			Item:        itemMap,
			FieldScores: queryResult.FieldScores,
			Score:       queryResult.Score,
		})
	}

	return results
}

// matchesTargetPage checks if an item matches the target page by comparing
// their URLs.
//
// Takes itemMap (map[string]any) which contains the item data with its URL.
// Takes targetPageData (map[string]any) which contains the target page data
// with its URL.
//
// Returns bool which is true when both maps have valid URL strings that match.
func matchesTargetPage(itemMap, targetPageData map[string]any) bool {
	targetURL, ok := targetPageData["url"].(string)
	if !ok {
		return false
	}

	itemURL, ok := itemMap["url"].(string)
	if !ok {
		return false
	}

	return targetURL == itemURL
}
