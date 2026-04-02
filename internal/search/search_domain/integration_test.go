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
	"fmt"
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/search/search_adapters"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

func TestIntegration_EndToEnd_FastMode(t *testing.T) {
	t.Parallel()

	testDocuments := []collection_dto.ContentItem{
		{
			URL: "/golang-tutorial",
			Metadata: map[string]any{
				"title":       "Introduction to Go Programming",
				"description": "Learn the basics of Go programming language",
			},
			RawContent: "Go is a statically typed, compiled programming language designed at Google.",
		},
		{
			URL: "/python-guide",
			Metadata: map[string]any{
				"title":       "Python for Beginners",
				"description": "Start your journey with Python",
			},
			RawContent: "Python is an interpreted, high-level programming language.",
		},
	}

	builder := search_domain.NewIndexBuilder()
	config := search_domain.DefaultIndexBuildConfig()
	config.AnalysisMode = search_schema_gen.SearchModeFast

	indexData, err := builder.BuildIndex(
		context.Background(),
		"test-collection",
		testDocuments,
		search_schema_gen.SearchModeFast,
		config,
	)
	if err != nil {
		t.Fatalf("Failed to build index: %v", err)
	}

	if len(indexData) == 0 {
		t.Fatal("Index data is empty")
	}

	reader := search_adapters.NewFlatBufferIndexReader()
	if err := reader.LoadIndex(indexData); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	corpusStats := reader.GetCorpusStats()
	if corpusStats.TotalDocuments != uint32(len(testDocuments)) {
		t.Errorf("Expected %d documents, got %d", len(testDocuments), corpusStats.TotalDocuments)
	}

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)
	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
			{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
		},
		Limit: 10,
	}

	results, err := processor.Search(
		context.Background(),
		"programming",
		reader,
		scorer,
		searchConfig,
	)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) < 1 {
		t.Error("Expected at least 1 result for 'programming'")
	}

	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted: result[%d].Score (%f) > result[%d].Score (%f)",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestIntegration_IndexBuilder_MultipleDocuments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		docs             []collection_dto.ContentItem
		validateTerms    []string
		expectedMinVocab uint32
		expectedDocCount uint32
	}{
		{
			name: "single document",
			docs: []collection_dto.ContentItem{
				{
					URL: "/test",
					Metadata: map[string]any{
						"title": "Test Document",
					},
					RawContent: "This is a test document with some content.",
				},
			},
			expectedMinVocab: 3,
			expectedDocCount: 1,
			validateTerms:    []string{"test", "document"},
		},
		{
			name: "multiple documents",
			docs: []collection_dto.ContentItem{
				{
					URL: "/doc1",
					Metadata: map[string]any{
						"title": "First Document",
					},
					RawContent: "Content of the first document.",
				},
				{
					URL: "/doc2",
					Metadata: map[string]any{
						"title": "Second Document",
					},
					RawContent: "Content of the second document.",
				},
			},
			expectedMinVocab: 3,
			expectedDocCount: 2,
			validateTerms:    []string{"first", "second", "document"},
		},
		{
			name: "empty documents",
			docs: []collection_dto.ContentItem{
				{
					URL:        "/empty",
					Metadata:   map[string]any{},
					RawContent: "",
				},
			},
			expectedMinVocab: 0,
			expectedDocCount: 1,
			validateTerms:    []string{},
		},
	}

	builder := search_domain.NewIndexBuilder()
	config := search_domain.DefaultIndexBuildConfig()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			indexData, err := builder.BuildIndex(
				context.Background(),
				"test-collection",
				tc.docs,
				search_schema_gen.SearchModeFast,
				config,
			)
			if err != nil {
				t.Fatalf("BuildIndex failed: %v", err)
			}

			reader := search_adapters.NewFlatBufferIndexReader()
			if err := reader.LoadIndex(indexData); err != nil {
				t.Fatalf("Failed to load index: %v", err)
			}

			stats := reader.GetCorpusStats()
			if stats.TotalDocuments != tc.expectedDocCount {
				t.Errorf("Expected %d documents, got %d", tc.expectedDocCount, stats.TotalDocuments)
			}

			for _, term := range tc.validateTerms {
				postings, _, err := reader.GetTermPostings(term)
				if err != nil || len(postings) == 0 {
					t.Errorf("Expected term '%s' to be in index", term)
				}
			}
		})
	}
}

func TestIntegration_QueryProcessor_Search(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{
		{
			URL: "/go-tutorial",
			Metadata: map[string]any{
				"title": "Go Programming Tutorial",
			},
			RawContent: "Learn Go programming with examples.",
		},
		{
			URL: "/python-basics",
			Metadata: map[string]any{
				"title": "Python Basics",
			},
			RawContent: "Python programming fundamentals.",
		},
	}

	builder := search_domain.NewIndexBuilder()
	config := search_domain.DefaultIndexBuildConfig()

	indexData, err := builder.BuildIndex(
		context.Background(),
		"test",
		docs,
		search_schema_gen.SearchModeFast,
		config,
	)
	if err != nil {
		t.Fatalf("Failed to build index: %v", err)
	}

	reader := search_adapters.NewFlatBufferIndexReader()
	if err := reader.LoadIndex(indexData); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	testCases := []struct {
		name       string
		query      string
		minResults int
		maxResults int
	}{
		{
			name:       "single term",
			query:      "programming",
			minResults: 1,
			maxResults: 2,
		},
		{
			name:       "no results",
			query:      "quantum",
			minResults: 0,
			maxResults: 0,
		},
		{
			name:       "empty query",
			query:      "",
			minResults: 0,
			maxResults: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			searchConfig := search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
					{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
				},
				Limit: 10,
			}

			results, err := processor.Search(
				context.Background(),
				tc.query,
				reader,
				scorer,
				searchConfig,
			)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(results) < tc.minResults {
				t.Errorf("Expected at least %d results, got %d", tc.minResults, len(results))
			}
			if len(results) > tc.maxResults {
				t.Errorf("Expected at most %d results, got %d", tc.maxResults, len(results))
			}
		})
	}
}

func TestIntegration_ContextCancellation(t *testing.T) {
	t.Parallel()

	docs := make([]collection_dto.ContentItem, 100)
	for i := range 100 {
		docs[i] = collection_dto.ContentItem{
			URL:        "/doc",
			Metadata:   map[string]any{"title": "Test"},
			RawContent: "Content",
		}
	}

	builder := search_domain.NewIndexBuilder()
	config := search_domain.DefaultIndexBuildConfig()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := builder.BuildIndex(ctx, "test", docs, search_schema_gen.SearchModeFast, config)
	if err == nil {
		t.Error("Expected error from cancelled context, got nil")
	}
}
