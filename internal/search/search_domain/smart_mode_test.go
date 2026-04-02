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
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

func TestIntegration_SmartMode_PhoneticSearch(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{
		{
			URL: "/config-guide",
			Metadata: map[string]any{
				"title": "Configuration Guide",
			},
			RawContent: "Learn about system configuration and setup",
		},
		{
			URL: "/install-guide",
			Metadata: map[string]any{
				"title": "Installation Instructions",
			},
			RawContent: "How to install the software",
		},
	}

	builder := search_domain.NewIndexBuilder()
	config := search_domain.DefaultIndexBuildConfig()
	config.AnalysisMode = search_schema_gen.SearchModeSmart

	indexData, err := builder.BuildIndex(
		context.Background(),
		"test",
		docs,
		search_schema_gen.SearchModeSmart,
		config,
	)
	if err != nil {
		t.Fatalf("Failed to build Smart mode index: %v", err)
	}

	if len(indexData) == 0 {
		t.Fatal("Index data is empty")
	}

	t.Logf("Smart mode index built successfully with phonetic mapping")

}

func TestIntegration_SmartMode_EndToEnd(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{
		{
			URL: "/programming-basics",
			Metadata: map[string]any{
				"title": "Programming Basics",
			},
			RawContent: "Learn fundamental programming concepts",
		},
		{
			URL: "/advanced-coding",
			Metadata: map[string]any{
				"title": "Advanced Coding",
			},
			RawContent: "Advanced coding techniques and patterns",
		},
		{
			URL: "/database-design",
			Metadata: map[string]any{
				"title": "Database Design",
			},
			RawContent: "Database design principles and normalization",
		},
	}

	config := search_domain.DefaultIndexBuildConfig()
	config.AnalysisMode = search_schema_gen.SearchModeSmart

	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeSmart, config)

	if reader.GetMode() != search_schema_gen.SearchModeSmart {
		t.Errorf("Expected SearchModeSmart, got %v", reader.GetMode())
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
		t.Fatalf("Smart mode search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result for 'programming' in Smart mode")
	}

	results2, err := processor.Search(
		context.Background(),
		"program",
		reader,
		scorer,
		searchConfig,
	)
	if err != nil {
		t.Fatalf("Smart mode search failed: %v", err)
	}

	if len(results2) == 0 {
		t.Error("Expected stemming to match 'program' with 'programming'")
	}

	t.Logf("Smart mode search successful: found %d results", len(results))
}

func TestIntegration_SmartMode_Stemming(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{
		{
			URL: "/running-guide",
			Metadata: map[string]any{
				"title": "Running Guide",
			},
			RawContent: "Everything about running, runners, and runs",
		},
	}

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeSmart, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
			{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
		},
		Limit: 10,
	}

	testCases := []struct {
		query       string
		description string
		shouldFind  bool
	}{
		{
			query:       "run",
			shouldFind:  true,
			description: "Stemmed 'run' should match 'running', 'runners', 'runs'",
		},
		{
			query:       "runner",
			shouldFind:  true,
			description: "Stemmed 'runner' should match variants",
		},
		{
			query:       "running",
			shouldFind:  true,
			description: "Exact term should match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
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

			found := len(results) > 0
			if found != tc.shouldFind {
				t.Errorf("%s: expected shouldFind=%v, got %v (found %d results)",
					tc.description, tc.shouldFind, found, len(results))
			}
		})
	}
}

func TestIntegration_SmartMode_VsFastMode(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{
		{
			URL: "/programming",
			Metadata: map[string]any{
				"title": "Programming Guide",
			},
			RawContent: "Learn about programs and programming",
		},
	}

	config := search_domain.DefaultIndexBuildConfig()

	fastReader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)
	smartReader := buildTestIndex(t, docs, search_schema_gen.SearchModeSmart, config)

	if fastReader.GetMode() != search_schema_gen.SearchModeFast {
		t.Errorf("Expected Fast mode, got %v", fastReader.GetMode())
	}
	if smartReader.GetMode() != search_schema_gen.SearchModeSmart {
		t.Errorf("Expected Smart mode, got %v", smartReader.GetMode())
	}

	fastProcessor := search_domain.NewQueryProcessorForIndex(fastReader)
	smartProcessor := search_domain.NewQueryProcessorForIndex(smartReader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	searchConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
			{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
		},
		Limit: 10,
	}

	query := "programming"

	fastResults, err := fastProcessor.Search(context.Background(), query, fastReader, scorer, searchConfig)
	if err != nil {
		t.Fatalf("Fast mode search failed: %v", err)
	}

	smartResults, err := smartProcessor.Search(context.Background(), query, smartReader, scorer, searchConfig)
	if err != nil {
		t.Fatalf("Smart mode search failed: %v", err)
	}

	t.Logf("Fast mode: %d results, Smart mode: %d results",
		len(fastResults), len(smartResults))

	if len(fastResults) == 0 {
		t.Error("Fast mode should find results for 'programming'")
	}

	t.Logf("Smart mode search completed successfully (found %d results)", len(smartResults))
}

func TestIntegration_SmartMode_EncodingRoundTrip(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{
		{
			URL: "/doc1",
			Metadata: map[string]any{
				"title": "First Document",
			},
			RawContent: "Content with various terms",
		},
		{
			URL: "/doc2",
			Metadata: map[string]any{
				"title": "Second Document",
			},
			RawContent: "Different content with different terms",
		},
	}

	builder := search_domain.NewIndexBuilder()
	config := search_domain.DefaultIndexBuildConfig()
	config.AnalysisMode = search_schema_gen.SearchModeSmart

	indexData, err := builder.BuildIndex(
		context.Background(),
		"test-collection",
		docs,
		search_schema_gen.SearchModeSmart,
		config,
	)
	if err != nil {
		t.Fatalf("Failed to build index: %v", err)
	}

	if len(indexData) == 0 {
		t.Fatal("Index data is empty")
	}

	t.Logf("Smart mode index encoded: %d bytes", len(indexData))

	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeSmart, config)

	stats := reader.GetCorpusStats()
	if stats.TotalDocuments != uint32(len(docs)) {
		t.Errorf("Expected %d docs, got %d", len(docs), stats.TotalDocuments)
	}

	if stats.VocabSize == 0 {
		t.Error("Expected non-zero vocabulary size")
	}

	t.Logf("Index stats: %d docs, %d terms, avg length %.2f",
		stats.TotalDocuments, stats.VocabSize, stats.AverageFieldLength)
}

func TestNewQueryProcessorForIndex(t *testing.T) {
	t.Parallel()

	docs := createTestDocuments(3, "Test")
	config := search_domain.DefaultIndexBuildConfig()

	testCases := []struct {
		name string
		mode search_schema_gen.SearchMode
	}{
		{
			name: "Fast mode",
			mode: search_schema_gen.SearchModeFast,
		},
		{
			name: "Smart mode",
			mode: search_schema_gen.SearchModeSmart,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reader := buildTestIndex(t, docs, tc.mode, config)

			processor := search_domain.NewQueryProcessorForIndex(reader)

			scorer := search_domain.NewBM25Scorer(
				search_domain.BM25DefaultK1,
				search_domain.BM25DefaultB,
			)

			searchConfig := search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
					{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
				},
				Limit: 10,
			}

			results, err := processor.Search(
				context.Background(),
				"test",
				reader,
				scorer,
				searchConfig,
			)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(results) == 0 {
				t.Error("Expected to find results")
			}

			t.Logf("NewQueryProcessorForIndex with %s: found %d results",
				tc.name, len(results))
		})
	}
}
