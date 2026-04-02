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

package search_domain_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

type mockScorerWithError struct {
	err error
}

func (m *mockScorerWithError) Score(
	_ context.Context,
	_ []string,
	_ uint32,
	_ search_domain.IndexReaderPort,
	_ search_dto.SearchConfig,
) (search_domain.ScoreResult, error) {
	if m.err != nil {
		return search_domain.ScoreResult{}, m.err
	}
	return search_domain.ScoreResult{
		Score:       1.0,
		FieldScores: make(map[string]float64),
	}, nil
}

func TestQueryProcessor_ContextCancellation_InSearch(t *testing.T) {
	t.Parallel()

	docs := createTestDocuments(50, "Test")

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
			{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
		},
		Limit: 100,
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), 1*time.Nanosecond, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	_, err := processor.Search(
		ctx,
		"test",
		reader,
		scorer,
		searchConfig,
	)

	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context cancellation error, got: %v", err)
		}
		t.Logf("Context cancellation handled correctly: %v", err)
	} else {
		t.Log("Search completed before context cancellation (acceptable)")
	}
}

func TestQueryProcessor_ContextCancellation_Immediate(t *testing.T) {
	t.Parallel()

	docs := createTestDocuments(10, "Test")

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
		},
		Limit: 10,
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := processor.Search(
		ctx,
		"test",
		reader,
		scorer,
		searchConfig,
	)

	if err != nil {
		t.Logf("Cancelled context handled: %v", err)
	}
}

func TestQueryProcessor_ScoringErrors(t *testing.T) {
	t.Parallel()

	docs := createTestDocuments(5, "Test")

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)

	mockScorer := &mockScorerWithError{
		err: errors.New("scoring failed"),
	}

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
		},
		Limit: 10,
	}

	results, err := processor.Search(
		context.Background(),
		"test",
		reader,
		mockScorer,
		searchConfig,
	)

	if err != nil {
		t.Errorf("Search should not fail when scoring errors occur, got: %v", err)
	}

	if len(results) > 0 {
		t.Errorf("Expected no results when all scoring fails, got %d", len(results))
	}

	t.Log("Scoring errors handled gracefully - documents skipped")
}

func TestQueryProcessor_EmptyQuery(t *testing.T) {
	t.Parallel()

	docs := createTestDocuments(5, "Test")

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
		},
		Limit: 10,
	}

	testCases := []struct {
		name  string
		query string
	}{
		{
			name:  "empty string",
			query: "",
		},
		{
			name:  "whitespace only",
			query: "   ",
		},
		{
			name:  "special characters only",
			query: "!@#$%",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := processor.Search(
				context.Background(),
				tc.query,
				reader,
				scorer,
				searchConfig,
			)

			if err != nil {
				t.Errorf("Should handle empty query gracefully, got error: %v", err)
			}

			if len(results) != 0 {
				t.Errorf("Expected no results for empty query, got %d", len(results))
			}
		})
	}
}

func TestQueryProcessor_NoResults(t *testing.T) {
	t.Parallel()

	docs := createTestDocuments(5, "Programming")

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
		},
		Limit: 10,
	}

	results, err := processor.Search(
		context.Background(),
		"nonexistentterm",
		reader,
		scorer,
		searchConfig,
	)

	if err != nil {
		t.Errorf("Should handle no results gracefully, got error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected no results, got %d", len(results))
	}
}

func TestIndexBuilder_EmptyDocuments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		description string
		docs        []collection_dto.ContentItem
		shouldError bool
	}{
		{
			name:        "empty document list",
			docs:        []collection_dto.ContentItem{},
			shouldError: false,
			description: "Should handle empty document list",
		},
		{
			name: "document with no content",
			docs: []collection_dto.ContentItem{
				{
					URL:        "/empty",
					Metadata:   map[string]any{},
					RawContent: "",
				},
			},
			shouldError: false,
			description: "Should handle document with no extractable content",
		},
		{
			name: "document with only whitespace",
			docs: []collection_dto.ContentItem{
				{
					URL: "/whitespace",
					Metadata: map[string]any{
						"title": "   ",
					},
					RawContent: "   ",
				},
			},
			shouldError: false,
			description: "Should handle document with only whitespace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder := search_domain.NewIndexBuilder()
			config := search_domain.DefaultIndexBuildConfig()

			indexData, err := builder.BuildIndex(
				context.Background(),
				"test",
				tc.docs,
				search_schema_gen.SearchModeFast,
				config,
			)

			if tc.shouldError && err == nil {
				t.Errorf("Expected error for %s, got none", tc.description)
			}

			if !tc.shouldError && err != nil {
				t.Errorf("Expected no error for %s, got: %v", tc.description, err)
			}

			if !tc.shouldError && err == nil {
				if len(indexData) == 0 {
					t.Errorf("Expected index data for %s", tc.description)
				}
				t.Logf("%s: index size = %d bytes", tc.description, len(indexData))
			}
		})
	}
}

func TestBM25Scorer_ExtremeValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		k1   float64
		b    float64
	}{
		{
			name: "zero k1",
			k1:   0.0,
			b:    0.75,
		},
		{
			name: "zero b",
			k1:   1.2,
			b:    0.0,
		},
		{
			name: "very high k1",
			k1:   100.0,
			b:    0.75,
		},
		{
			name: "k1 = 1",
			k1:   1.0,
			b:    1.0,
		},
	}

	docs := createTestDocuments(3, "Test")
	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
		},
		Limit: 10,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scorer := search_domain.NewBM25Scorer(tc.k1, tc.b)

			results, err := processor.Search(
				context.Background(),
				"test",
				reader,
				scorer,
				searchConfig,
			)

			if err != nil {
				t.Errorf("Search with %s failed: %v", tc.name, err)
			}

			if len(results) == 0 {
				t.Errorf("Expected results with %s", tc.name)
			}

			t.Logf("%s: found %d results with scores", tc.name, len(results))
		})
	}
}

func TestSearchWithExplanation(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{
		{
			URL: "/doc1",
			Metadata: map[string]any{
				"title": "Programming Guide",
			},
			RawContent: "Learn programming fundamentals",
		},
		{
			URL: "/doc2",
			Metadata: map[string]any{
				"title": "Coding Basics",
			},
			RawContent: "Basic coding concepts",
		},
	}

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
			{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
		},
		Limit: 10,
	}

	results, explanations, err := processor.SearchWithExplanation(
		context.Background(),
		"programming",
		reader,
		scorer,
		searchConfig,
	)

	if err != nil {
		t.Fatalf("SearchWithExplanation failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	if len(results) != len(explanations) {
		t.Errorf("Expected %d explanations, got %d", len(results), len(explanations))
	}

	for i, explanation := range explanations {
		if explanation == nil {
			t.Errorf("Explanation %d is nil", i)
			continue
		}

		if explanation.DocumentID != results[i].DocumentID {
			t.Errorf("Explanation documentID %d doesn't match result documentID %d",
				explanation.DocumentID, results[i].DocumentID)
		}

		if explanation.TotalScore != results[i].Score {
			t.Errorf("Explanation score %.4f doesn't match result score %.4f",
				explanation.TotalScore, results[i].Score)
		}

		explanationString := explanation.String()
		if explanationString == "" {
			t.Error("Explanation string is empty")
		}

		t.Logf("Explanation %d:\n%s", i, explanationString)
	}
}

func TestSearchWithExplanation_NoResults(t *testing.T) {
	t.Parallel()

	docs := createTestDocuments(2, "Test")
	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
		},
		Limit: 10,
	}

	results, explanations, err := processor.SearchWithExplanation(
		context.Background(),
		"nonexistent",
		reader,
		scorer,
		searchConfig,
	)

	if err != nil {
		t.Fatalf("SearchWithExplanation failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected no results, got %d", len(results))
	}

	if len(explanations) != 0 {
		t.Errorf("Expected no explanations, got %d", len(explanations))
	}
}

func TestIntegration_LargeIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large index test in short mode")
	}

	t.Parallel()

	const numDocuments = 1000
	docs := make([]collection_dto.ContentItem, numDocuments)
	for i := range numDocuments {
		docs[i] = collection_dto.ContentItem{
			URL: fmt.Sprintf("/doc%d", i),
			Metadata: map[string]any{
				"title": fmt.Sprintf("Document %d about topic %d", i, i%10),
			},
			RawContent: fmt.Sprintf("This is content for document number %d with keywords topic%d", i, i%10),
		}
	}

	config := search_domain.DefaultIndexBuildConfig()

	start := time.Now()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)
	buildTime := time.Since(start)

	t.Logf("Built index of %d documents in %v", numDocuments, buildTime)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
			{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
		},
		Limit: 20,
	}

	start = time.Now()
	results, err := processor.Search(
		context.Background(),
		"document topic",
		reader,
		scorer,
		searchConfig,
	)
	searchTime := time.Since(start)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	t.Logf("Searched %d documents in %v, found %d results",
		numDocuments, searchTime, len(results))

	if len(results) == 0 {
		t.Error("Expected to find results in large index")
	}

	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted: result[%d].Score (%f) > result[%d].Score (%f)",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}
