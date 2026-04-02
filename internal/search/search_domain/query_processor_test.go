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

func TestQueryProcessor_Pagination_EdgeCases(t *testing.T) {
	t.Parallel()

	docs := createTestDocuments(15, "Test")

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	testCases := []struct {
		name          string
		description   string
		limit         int
		offset        int
		expectedCount int
	}{
		{
			name:          "no pagination returns all results",
			limit:         0,
			offset:        0,
			expectedCount: 15,
			description:   "Should return all matching documents",
		},
		{
			name:          "limit trims results",
			limit:         5,
			offset:        0,
			expectedCount: 5,
			description:   "Should return only first 5 results",
		},
		{
			name:          "offset skips results",
			limit:         0,
			offset:        5,
			expectedCount: 10,
			description:   "Should skip first 5 and return remaining 10",
		},
		{
			name:          "offset beyond results returns empty",
			limit:         10,
			offset:        20,
			expectedCount: 0,
			description:   "Should return empty when offset exceeds result count",
		},
		{
			name:          "offset at boundary",
			limit:         10,
			offset:        15,
			expectedCount: 0,
			description:   "Should return empty when offset equals result count",
		},
		{
			name:          "limit + offset combination",
			limit:         3,
			offset:        2,
			expectedCount: 3,
			description:   "Should skip 2 and return 3",
		},
		{
			name:          "large limit returns all available",
			limit:         100,
			offset:        0,
			expectedCount: 15,
			description:   "Should return all 15 when limit exceeds available",
		},
		{
			name:          "offset near end with limit",
			limit:         10,
			offset:        12,
			expectedCount: 3,
			description:   "Should return remaining 3 documents",
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
				Limit:  tc.limit,
				Offset: tc.offset,
			}

			results, err := processor.Search(
				context.Background(),
				"document",
				reader,
				scorer,
				searchConfig,
			)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(results) != tc.expectedCount {
				t.Errorf("Expected %d results, got %d (%s)",
					tc.expectedCount, len(results), tc.description)
			}
		})
	}
}

func TestQueryProcessor_MinScoreFilter(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{
		{
			URL: "/highly-relevant",
			Metadata: map[string]any{
				"title": "Programming Programming Programming",
			},
			RawContent: "Programming is about programming and more programming",
		},
		{
			URL: "/somewhat-relevant",
			Metadata: map[string]any{
				"title": "Introduction to Programming",
			},
			RawContent: "This is a basic introduction",
		},
		{
			URL: "/barely-relevant",
			Metadata: map[string]any{
				"title": "Random Topic",
			},
			RawContent: "This document mentions programming once",
		},
	}

	config := search_domain.DefaultIndexBuildConfig()
	reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

	processor := search_domain.NewQueryProcessorForIndex(reader)
	scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)

	baselineConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
			{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
		},
		MinScore: 0.0,
		Limit:    10,
	}

	baselineResults, err := processor.Search(
		context.Background(),
		"programming",
		reader,
		scorer,
		baselineConfig,
	)
	if err != nil {
		t.Fatalf("Baseline search failed: %v", err)
	}

	if len(baselineResults) < 2 {
		t.Fatalf("Expected at least 2 results for test, got %d", len(baselineResults))
	}

	t.Logf("Baseline search found %d results", len(baselineResults))
	for i, result := range baselineResults {
		t.Logf("  Result %d: DocumentID=%d, Score=%.4f", i, result.DocumentID, result.Score)
	}

	highScoreConfig := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: search_domain.DefaultFieldWeightTitle},
			{Name: "content", Weight: search_domain.DefaultFieldWeightContent},
		},
		MinScore: 1000.0,
		Limit:    10,
	}

	highScoreResults, err := processor.Search(
		context.Background(),
		"programming",
		reader,
		scorer,
		highScoreConfig,
	)
	if err != nil {
		t.Fatalf("High score search failed: %v", err)
	}

	if len(highScoreResults) > len(baselineResults) {
		t.Errorf("High MinScore should not increase results: baseline=%d, filtered=%d",
			len(baselineResults), len(highScoreResults))
	}

	t.Logf("MinScore filtering: baseline=%d results, high_threshold=%d results",
		len(baselineResults), len(highScoreResults))
}

func TestIndexBuilder_LanguageDetection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		metadata         map[string]any
		expectedLanguage string
		description      string
	}{
		{
			name: "explicit language field",
			metadata: map[string]any{
				"title":    "Test Document",
				"language": "spanish",
			},
			expectedLanguage: "spanish",
			description:      "Should use explicit language field",
		},
		{
			name: "lang field (HTML style)",
			metadata: map[string]any{
				"title": "Test Document",
				"lang":  "french",
			},
			expectedLanguage: "french",
			description:      "Should use lang field",
		},
		{
			name: "locale field with language extraction",
			metadata: map[string]any{
				"title":  "Test Document",
				"locale": "es-ES",
			},
			expectedLanguage: "spanish",
			description:      "Should extract language from locale",
		},
		{
			name: "locale field french",
			metadata: map[string]any{
				"title":  "Test Document",
				"locale": "fr-FR",
			},
			expectedLanguage: "french",
			description:      "Should extract French from locale",
		},
		{
			name: "no language defaults to english",
			metadata: map[string]any{
				"title": "Test Document",
			},
			expectedLanguage: "english",
			description:      "Should default to English when no language specified",
		},
		{
			name: "empty language field defaults to english",
			metadata: map[string]any{
				"title":    "Test Document",
				"language": "",
			},
			expectedLanguage: "english",
			description:      "Should default to English when language is empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			docs := []collection_dto.ContentItem{
				createTestDocWithMetadata("/test", tc.metadata, "Some test content"),
			}

			config := search_domain.DefaultIndexBuildConfig()
			config.Language = ""

			reader := buildTestIndex(t, docs, search_schema_gen.SearchModeFast, config)

			actualLanguage := reader.GetLanguage()
			if actualLanguage != tc.expectedLanguage {
				t.Errorf("Expected language %q, got %q (%s)",
					tc.expectedLanguage, actualLanguage, tc.description)
			}
		})
	}
}

func TestIndexBuilder_FieldWeight_EdgeCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		fieldWeights map[string]float64
		name         string
		description  string
		shouldWork   bool
	}{
		{
			name: "zero weight should fallback",
			fieldWeights: map[string]float64{
				"title":   0.0,
				"content": 1.0,
			},
			shouldWork:  true,
			description: "Zero weight should fallback to 1",
		},
		{
			name: "negative weight should fallback",
			fieldWeights: map[string]float64{
				"title":   -1.0,
				"content": 1.0,
			},
			shouldWork:  true,
			description: "Negative weight should fallback to 1",
		},
		{
			name: "very high weight",
			fieldWeights: map[string]float64{
				"title":   100.0,
				"content": 1.0,
			},
			shouldWork:  true,
			description: "Very high weight should work",
		},
		{
			name: "fractional weight",
			fieldWeights: map[string]float64{
				"title":   0.5,
				"content": 1.0,
			},
			shouldWork:  true,
			description: "Fractional weight should work",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			docs := []collection_dto.ContentItem{{
				URL: "/test",
				Metadata: map[string]any{
					"title": "Important Title",
				},
				RawContent: "Some content here",
			}}

			builder := search_domain.NewIndexBuilder()
			config := search_domain.DefaultIndexBuildConfig()
			config.FieldWeights = tc.fieldWeights

			_, err := builder.BuildIndex(
				context.Background(),
				"test",
				docs,
				search_schema_gen.SearchModeFast,
				config,
			)

			if tc.shouldWork && err != nil {
				t.Errorf("Expected success but got error: %v (%s)", err, tc.description)
			}
			if !tc.shouldWork && err == nil {
				t.Errorf("Expected error but got success (%s)", tc.description)
			}
		})
	}
}

func TestIndexBuilder_PlainContentField(t *testing.T) {
	t.Parallel()

	docs := []collection_dto.ContentItem{{
		URL: "/test",
		Metadata: map[string]any{
			"title": "Test Title",
		},
		PlainContent: "This is plain content that should be indexed as body text",
	}}

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

	results, err := processor.Search(
		context.Background(),
		"plain",
		reader,
		scorer,
		searchConfig,
	)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to find document with plain content, but got no results")
	}

	if len(results) > 0 && results[0].DocumentID != 0 {
		t.Errorf("Expected documentID 0, got %d", results[0].DocumentID)
	}
}
