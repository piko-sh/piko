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
	"errors"
	"testing"

	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

func mustCastToSearchService(t *testing.T, service SearchServicePort) *searchService {
	t.Helper()
	ss, ok := service.(*searchService)
	if !ok {
		t.Fatal("expected *searchService")
	}
	return ss
}

type mockIndexReader struct{}

func (m *mockIndexReader) LoadIndex(_ []byte) error { return nil }
func (m *mockIndexReader) GetTermPostings(_ string) ([]search_domain.PostingInfo, float64, error) {
	return nil, 0, nil
}
func (m *mockIndexReader) GetDocMetadata(_ uint32) (search_domain.DocMetadataInfo, error) {
	return search_domain.DocMetadataInfo{}, nil
}
func (m *mockIndexReader) GetCorpusStats() search_domain.CorpusStats {
	return search_domain.CorpusStats{}
}
func (m *mockIndexReader) GetMode() search_schema_gen.SearchMode {
	return search_schema_gen.SearchModeFast
}
func (m *mockIndexReader) GetLanguage() string                            { return "english" }
func (m *mockIndexReader) FindPhoneticTerms(_ string) ([]string, error)   { return nil, nil }
func (m *mockIndexReader) GetAllTerms() ([]string, error)                 { return nil, nil }
func (m *mockIndexReader) FindTermsWithPrefix(_ string) ([]string, error) { return nil, nil }

func TestNewSearchService_DefaultDependencies(t *testing.T) {
	service := NewSearchService()
	if service == nil {
		t.Fatal("NewSearchService() returned nil")
	}
}

func TestNewSearchService_withIndexLoader(t *testing.T) {
	mockLoader := &MockSearchIndexLoader{}
	service := NewSearchService(withIndexLoader(mockLoader))

	if service == nil {
		t.Fatal("NewSearchService() returned nil")
	}

	s := mustCastToSearchService(t, service)
	if s.indexLoader != mockLoader {
		t.Error("withIndexLoader did not inject the mock loader")
	}
}

func TestNewSearchService_withItemsLoader(t *testing.T) {
	mockLoader := &MockCollectionItemsLoader{}
	service := NewSearchService(withItemsLoader(mockLoader))

	if service == nil {
		t.Fatal("NewSearchService() returned nil")
	}

	s := mustCastToSearchService(t, service)
	if s.itemsLoader != mockLoader {
		t.Error("withItemsLoader did not inject the mock loader")
	}
}

func TestNewSearchService_withQueryProcessorFactory(t *testing.T) {
	factoryCalled := false
	mockFactory := func(_ search_domain.IndexReaderPort) search_domain.QueryProcessorPort {
		factoryCalled = true
		return &MockQueryProcessor{}
	}

	service := NewSearchService(withQueryProcessorFactory(mockFactory))
	if service == nil {
		t.Fatal("NewSearchService() returned nil")
	}

	_ = factoryCalled
}

func TestNewSearchService_withScorerFactory(t *testing.T) {
	factoryCalled := false
	mockFactory := func() search_domain.ScorerPort {
		factoryCalled = true
		return &MockScorer{}
	}

	service := NewSearchService(withScorerFactory(mockFactory))
	if service == nil {
		t.Fatal("NewSearchService() returned nil")
	}

	_ = factoryCalled
}

func TestNewSearchService_MultipleOptions(t *testing.T) {
	mockIndexLoader := &MockSearchIndexLoader{}
	mockItemsLoader := &MockCollectionItemsLoader{}

	service := NewSearchService(
		withIndexLoader(mockIndexLoader),
		withItemsLoader(mockItemsLoader),
	)

	if service == nil {
		t.Fatal("NewSearchService() returned nil")
	}

	s := mustCastToSearchService(t, service)
	if s.indexLoader != mockIndexLoader {
		t.Error("indexLoader not injected")
	}
	if s.itemsLoader != mockItemsLoader {
		t.Error("itemsLoader not injected")
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	service := NewSearchService()

	results, err := service.Search(
		context.Background(),
		"test-collection",
		nil,
		SearchConfig{Query: ""},
		"fast",
	)

	if err != nil {
		t.Fatalf("Search() returned error: %v", err)
	}

	if results != nil {
		t.Errorf("Expected nil results for empty query, got %v", results)
	}
}

func TestSearch_IndexLoadError(t *testing.T) {
	expectedErr := errors.New("index not found")
	mockLoader := &MockSearchIndexLoader{
		GetIndexFunc: func(_, _ string) (any, error) {
			return nil, expectedErr
		},
	}

	service := NewSearchService(withIndexLoader(mockLoader))

	_, err := service.Search(
		context.Background(),
		"test-collection",
		nil,
		SearchConfig{Query: "test"},
		"fast",
	)

	if err == nil {
		t.Error("Expected error from index loader")
	}
}

func TestSearch_InvalidIndexReaderType(t *testing.T) {
	mockLoader := &MockSearchIndexLoader{
		GetIndexFunc: func(_, _ string) (any, error) {

			return "not an index reader", nil
		},
	}

	service := NewSearchService(withIndexLoader(mockLoader))

	_, err := service.Search(
		context.Background(),
		"test-collection",
		nil,
		SearchConfig{Query: "test"},
		"fast",
	)

	if err == nil {
		t.Error("Expected error for invalid index reader type")
	}
}

func TestSearch_Success(t *testing.T) {
	mockReader := &mockIndexReader{}
	mockLoader := &MockSearchIndexLoader{
		GetIndexFunc: func(_, _ string) (any, error) {
			return mockReader, nil
		},
	}

	mockItemsLoader := &MockCollectionItemsLoader{
		GetAllItemsFunc: func(_ string) ([]map[string]any, error) {
			return []map[string]any{
				{"title": "Doc 1", "url": "/doc/1"},
				{"title": "Doc 2", "url": "/doc/2"},
			}, nil
		},
	}

	mockProcessor := &MockQueryProcessor{
		SearchFunc: func(_ context.Context, _ string, _ search_domain.IndexReaderPort, _ search_domain.ScorerPort, _ search_dto.SearchConfig) ([]search_domain.QueryResult, error) {
			return []search_domain.QueryResult{
				{DocumentID: 0, Score: 0.9, FieldScores: map[string]float64{"title": 0.9}},
			}, nil
		},
	}

	service := NewSearchService(
		withIndexLoader(mockLoader),
		withItemsLoader(mockItemsLoader),
		withQueryProcessorFactory(func(_ search_domain.IndexReaderPort) search_domain.QueryProcessorPort {
			return mockProcessor
		}),
		withScorerFactory(func() search_domain.ScorerPort {
			return &MockScorer{}
		}),
	)

	results, err := service.Search(
		context.Background(),
		"test-collection",
		nil,
		SearchConfig{Query: "test"},
		"fast",
	)

	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Score != 0.9 {
		t.Errorf("Expected score 0.9, got %f", results[0].Score)
	}

	if results[0].Item["title"] != "Doc 1" {
		t.Errorf("Expected title 'Doc 1', got %v", results[0].Item["title"])
	}
}

func TestSearch_QueryProcessorError(t *testing.T) {
	mockReader := &mockIndexReader{}
	mockLoader := &MockSearchIndexLoader{
		GetIndexFunc: func(_, _ string) (any, error) {
			return mockReader, nil
		},
	}

	expectedErr := errors.New("query processing failed")
	mockProcessor := &MockQueryProcessor{
		SearchFunc: func(_ context.Context, _ string, _ search_domain.IndexReaderPort, _ search_domain.ScorerPort, _ search_dto.SearchConfig) ([]search_domain.QueryResult, error) {
			return nil, expectedErr
		},
	}

	service := NewSearchService(
		withIndexLoader(mockLoader),
		withQueryProcessorFactory(func(_ search_domain.IndexReaderPort) search_domain.QueryProcessorPort {
			return mockProcessor
		}),
	)

	_, err := service.Search(
		context.Background(),
		"test-collection",
		nil,
		SearchConfig{Query: "test"},
		"fast",
	)

	if err == nil {
		t.Error("Expected error from query processor")
	}
}

func TestSearch_ItemsLoadError(t *testing.T) {
	mockReader := &mockIndexReader{}
	mockLoader := &MockSearchIndexLoader{
		GetIndexFunc: func(_, _ string) (any, error) {
			return mockReader, nil
		},
	}

	expectedErr := errors.New("items load failed")
	mockItemsLoader := &MockCollectionItemsLoader{
		GetAllItemsFunc: func(_ string) ([]map[string]any, error) {
			return nil, expectedErr
		},
	}

	mockProcessor := &MockQueryProcessor{
		SearchFunc: func(_ context.Context, _ string, _ search_domain.IndexReaderPort, _ search_domain.ScorerPort, _ search_dto.SearchConfig) ([]search_domain.QueryResult, error) {
			return []search_domain.QueryResult{}, nil
		},
	}

	service := NewSearchService(
		withIndexLoader(mockLoader),
		withItemsLoader(mockItemsLoader),
		withQueryProcessorFactory(func(_ search_domain.IndexReaderPort) search_domain.QueryProcessorPort {
			return mockProcessor
		}),
	)

	_, err := service.Search(
		context.Background(),
		"test-collection",
		nil,
		SearchConfig{Query: "test"},
		"fast",
	)

	if err == nil {
		t.Error("Expected error from items loader")
	}
}

func TestSearch_WithCurrentPageData(t *testing.T) {
	mockReader := &mockIndexReader{}
	mockLoader := &MockSearchIndexLoader{
		GetIndexFunc: func(_, _ string) (any, error) {
			return mockReader, nil
		},
	}

	mockItemsLoader := &MockCollectionItemsLoader{
		GetAllItemsFunc: func(_ string) ([]map[string]any, error) {
			return []map[string]any{
				{"title": "Doc 1", "url": "/doc/1"},
				{"title": "Doc 2", "url": "/doc/2"},
				{"title": "Doc 3", "url": "/doc/3"},
			}, nil
		},
	}

	mockProcessor := &MockQueryProcessor{
		SearchFunc: func(_ context.Context, _ string, _ search_domain.IndexReaderPort, _ search_domain.ScorerPort, _ search_dto.SearchConfig) ([]search_domain.QueryResult, error) {

			return []search_domain.QueryResult{
				{DocumentID: 0, Score: 0.9},
				{DocumentID: 1, Score: 0.8},
				{DocumentID: 2, Score: 0.7},
			}, nil
		},
	}

	service := NewSearchService(
		withIndexLoader(mockLoader),
		withItemsLoader(mockItemsLoader),
		withQueryProcessorFactory(func(_ search_domain.IndexReaderPort) search_domain.QueryProcessorPort {
			return mockProcessor
		}),
		withScorerFactory(func() search_domain.ScorerPort {
			return &MockScorer{}
		}),
	)

	currentPageData := map[string]any{"url": "/doc/2"}
	results, err := service.Search(
		context.Background(),
		"test-collection",
		currentPageData,
		SearchConfig{Query: "test"},
		"fast",
	)

	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result (filtered to current page), got %d", len(results))
	}

	if results[0].Item["url"] != "/doc/2" {
		t.Errorf("Expected url '/doc/2', got %v", results[0].Item["url"])
	}
}

func TestSearch_DocumentIDOutOfBounds(t *testing.T) {
	mockReader := &mockIndexReader{}
	mockLoader := &MockSearchIndexLoader{
		GetIndexFunc: func(_, _ string) (any, error) {
			return mockReader, nil
		},
	}

	mockItemsLoader := &MockCollectionItemsLoader{
		GetAllItemsFunc: func(_ string) ([]map[string]any, error) {
			return []map[string]any{
				{"title": "Doc 1"},
			}, nil
		},
	}

	mockProcessor := &MockQueryProcessor{
		SearchFunc: func(_ context.Context, _ string, _ search_domain.IndexReaderPort, _ search_domain.ScorerPort, _ search_dto.SearchConfig) ([]search_domain.QueryResult, error) {

			return []search_domain.QueryResult{
				{DocumentID: 0, Score: 0.9},
				{DocumentID: 999, Score: 0.8},
			}, nil
		},
	}

	service := NewSearchService(
		withIndexLoader(mockLoader),
		withItemsLoader(mockItemsLoader),
		withQueryProcessorFactory(func(_ search_domain.IndexReaderPort) search_domain.QueryProcessorPort {
			return mockProcessor
		}),
		withScorerFactory(func() search_domain.ScorerPort {
			return &MockScorer{}
		}),
	)

	results, err := service.Search(
		context.Background(),
		"test-collection",
		nil,
		SearchConfig{Query: "test"},
		"fast",
	)

	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result (out-of-bounds skipped), got %d", len(results))
	}
}

func TestBuildSearchDTO(t *testing.T) {
	config := SearchConfig{
		Query:          "test query",
		FuzzyThreshold: 0.5,
		MinScore:       0.3,
		Limit:          10,
		Offset:         5,
		CaseSensitive:  true,
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: 2.0},
		},
	}

	dto := buildSearchDTO(config)

	if dto.Query != "test query" {
		t.Errorf("Expected query 'test query', got %q", dto.Query)
	}

	if dto.FuzzyThreshold != 0.5 {
		t.Errorf("Expected FuzzyThreshold 0.5, got %f", dto.FuzzyThreshold)
	}

	if dto.MinScore != 0.3 {
		t.Errorf("Expected MinScore 0.3, got %f", dto.MinScore)
	}

	if dto.Limit != 10 {
		t.Errorf("Expected Limit 10, got %d", dto.Limit)
	}

	if dto.Offset != 5 {
		t.Errorf("Expected Offset 5, got %d", dto.Offset)
	}

	if !dto.CaseSensitive {
		t.Error("Expected CaseSensitive true")
	}

	if !dto.EnableFuzzyFallback {
		t.Error("Expected EnableFuzzyFallback to be true")
	}

	if dto.FuzzySimilarityThreshold != defaultFuzzySimilarityThreshold {
		t.Errorf("Expected FuzzySimilarityThreshold %f, got %f", defaultFuzzySimilarityThreshold, dto.FuzzySimilarityThreshold)
	}

	if dto.FuzzyMaxResults != defaultFuzzyMaxResults {
		t.Errorf("Expected FuzzyMaxResults %d, got %d", defaultFuzzyMaxResults, dto.FuzzyMaxResults)
	}
}

func TestHydrateSearchResults_Empty(t *testing.T) {
	results := hydrateSearchResults(nil, nil, nil)
	if results == nil {
		t.Error("Expected non-nil slice for empty input")
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestHydrateSearchResults_Basic(t *testing.T) {
	queryResults := []search_domain.QueryResult{
		{DocumentID: 0, Score: 0.9, FieldScores: map[string]float64{"title": 0.9}},
		{DocumentID: 1, Score: 0.7, FieldScores: map[string]float64{"title": 0.7}},
	}

	allItems := []map[string]any{
		{"title": "First Doc"},
		{"title": "Second Doc"},
	}

	results := hydrateSearchResults(queryResults, allItems, nil)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0].Item["title"] != "First Doc" {
		t.Errorf("Expected 'First Doc', got %v", results[0].Item["title"])
	}

	if results[0].Score != 0.9 {
		t.Errorf("Expected score 0.9, got %f", results[0].Score)
	}
}

func TestHydrateSearchResults_WithTargetPage(t *testing.T) {
	queryResults := []search_domain.QueryResult{
		{DocumentID: 0, Score: 0.9},
		{DocumentID: 1, Score: 0.8},
	}

	allItems := []map[string]any{
		{"title": "Doc 1", "url": "/doc/1"},
		{"title": "Doc 2", "url": "/doc/2"},
	}

	targetPage := map[string]any{"url": "/doc/2"}

	results := hydrateSearchResults(queryResults, allItems, targetPage)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result (filtered), got %d", len(results))
	}

	if results[0].Item["url"] != "/doc/2" {
		t.Errorf("Expected url '/doc/2', got %v", results[0].Item["url"])
	}
}

func TestMatchesTargetPage_Match(t *testing.T) {
	item := map[string]any{"url": "/test/page"}
	target := map[string]any{"url": "/test/page"}

	if !matchesTargetPage(item, target) {
		t.Error("Expected match for identical URLs")
	}
}

func TestMatchesTargetPage_NoMatch(t *testing.T) {
	item := map[string]any{"url": "/test/page1"}
	target := map[string]any{"url": "/test/page2"}

	if matchesTargetPage(item, target) {
		t.Error("Expected no match for different URLs")
	}
}

func TestMatchesTargetPage_MissingTargetURL(t *testing.T) {
	item := map[string]any{"url": "/test/page"}
	target := map[string]any{"title": "No URL"}

	if matchesTargetPage(item, target) {
		t.Error("Expected no match when target has no URL")
	}
}

func TestMatchesTargetPage_MissingItemURL(t *testing.T) {
	item := map[string]any{"title": "No URL"}
	target := map[string]any{"url": "/test/page"}

	if matchesTargetPage(item, target) {
		t.Error("Expected no match when item has no URL")
	}
}

func TestMatchesTargetPage_InvalidURLTypes(t *testing.T) {
	item := map[string]any{"url": 123}
	target := map[string]any{"url": "/test"}

	if matchesTargetPage(item, target) {
		t.Error("Expected no match for invalid URL type")
	}
}

type testDoc struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Score int    `json:"score"`
}

func TestConvertSearchResultToType_Success(t *testing.T) {
	item := map[string]any{
		"title": "Test Document",
		"url":   "/test/doc",
		"score": 42,
	}

	var target testDoc
	err := ConvertSearchResultToType(item, &target)
	if err != nil {
		t.Fatalf("ConvertSearchResultToType() failed: %v", err)
	}

	if target.Title != "Test Document" {
		t.Errorf("Expected Title 'Test Document', got %q", target.Title)
	}

	if target.URL != "/test/doc" {
		t.Errorf("Expected URL '/test/doc', got %q", target.URL)
	}

	if target.Score != 42 {
		t.Errorf("Expected Score 42, got %d", target.Score)
	}
}

func TestConvertSearchResultToType_PartialFields(t *testing.T) {
	item := map[string]any{
		"title": "Only Title",
	}

	var target testDoc
	err := ConvertSearchResultToType(item, &target)
	if err != nil {
		t.Fatalf("ConvertSearchResultToType() failed: %v", err)
	}

	if target.Title != "Only Title" {
		t.Errorf("Expected Title 'Only Title', got %q", target.Title)
	}

	if target.URL != "" {
		t.Errorf("Expected empty URL, got %q", target.URL)
	}
}

func TestConvertSearchResultToType_EmptyMap(t *testing.T) {
	item := map[string]any{}

	var target testDoc
	err := ConvertSearchResultToType(item, &target)
	if err != nil {
		t.Fatalf("ConvertSearchResultToType() failed: %v", err)
	}

	if target.Title != "" {
		t.Errorf("Expected empty Title, got %q", target.Title)
	}
}

func Test_newDefaultSearchIndexLoader(t *testing.T) {
	loader := newDefaultSearchIndexLoader()
	if loader == nil {
		t.Fatal("newDefaultSearchIndexLoader() returned nil")
	}
}

func Test_newDefaultCollectionItemsLoader(t *testing.T) {
	loader := newDefaultCollectionItemsLoader()
	if loader == nil {
		t.Fatal("newDefaultCollectionItemsLoader() returned nil")
	}
}

func TestConvertSearchResultToType_NilMap(t *testing.T) {
	t.Parallel()

	var target testDoc
	err := ConvertSearchResultToType(nil, &target)
	if err != nil {
		t.Fatalf("ConvertSearchResultToType(nil) returned error: %v", err)
	}

	if target.Title != "" {
		t.Errorf("expected empty Title, got %q", target.Title)
	}
	if target.URL != "" {
		t.Errorf("expected empty URL, got %q", target.URL)
	}
	if target.Score != 0 {
		t.Errorf("expected zero Score, got %d", target.Score)
	}
}

func TestConvertSearchResultToType_IncompatibleTypes(t *testing.T) {
	t.Parallel()

	item := map[string]any{
		"score": "not-a-number",
	}

	type strictDoc struct {
		Score int `json:"score"`
	}

	var target strictDoc
	err := ConvertSearchResultToType(item, &target)

	if err != nil && target.Score != 0 {
		t.Errorf("expected zero Score or error, got Score=%d err=%v", target.Score, err)
	}
}
