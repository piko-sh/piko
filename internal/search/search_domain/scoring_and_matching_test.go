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

package search_domain

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"piko.sh/piko/internal/search/search_dto"
)

func TestScoreTermInDocument_ErrorCases(t *testing.T) {
	t.Parallel()

	scorer := NewBM25Scorer(BM25DefaultK1, BM25DefaultB)

	config := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: DefaultFieldWeightTitle},
			{Name: "content", Weight: DefaultFieldWeightContent},
		},
	}

	testCases := []struct {
		reader            IndexReaderPort
		name              string
		term              string
		expectedFieldName string
		description       string
		expectedScore     float64
		documentID        uint32
	}{
		{
			name: "term not found",
			reader: &mockIndexReader{
				terms: map[string][]PostingInfo{},
				idfs:  map[string]float64{},
			},
			term:              "nonexistent",
			documentID:        0,
			expectedScore:     0.0,
			expectedFieldName: "",
			description:       "Should return 0 when term is not in index",
		},
		{
			name: "error getting term postings",
			reader: &mockIndexReader{
				err: errors.New("index read error"),
			},
			term:              "test",
			documentID:        0,
			expectedScore:     0.0,
			expectedFieldName: "",
			description:       "Should return 0 on error",
		},
		{
			name: "term exists but not in document",
			reader: &mockIndexReader{
				terms: map[string][]PostingInfo{
					"test": {
						{DocumentID: 1, TermFrequency: 5, FieldID: 0},
						{DocumentID: 2, TermFrequency: 3, FieldID: 1},
					},
				},
				idfs: map[string]float64{
					"test": 1.5,
				},
			},
			term:              "test",
			documentID:        99,
			expectedScore:     0.0,
			expectedFieldName: "",
			description:       "Should return 0 when term not found in specific document",
		},
		{
			name: "empty postings list",
			reader: &mockIndexReader{
				terms: map[string][]PostingInfo{
					"test": {},
				},
				idfs: map[string]float64{
					"test": 0.0,
				},
			},
			term:              "test",
			documentID:        0,
			expectedScore:     0.0,
			expectedFieldName: "",
			description:       "Should return 0 for empty postings list",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := scorer.scoreTermInDocument(tc.term, tc.documentID, 1.0, tc.reader, buildFieldWeightMap(config))

			if result.score != tc.expectedScore {
				t.Errorf("Expected score %f, got %f", tc.expectedScore, result.score)
			}

			if result.fieldName != tc.expectedFieldName {
				t.Errorf("Expected fieldName %q, got %q", tc.expectedFieldName, result.fieldName)
			}
		})
	}
}

func TestFindTermFrequency(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		description   string
		postings      []PostingInfo
		expectedTF    float64
		documentID    uint32
		expectedFID   uint8
		expectedFound bool
	}{
		{
			name: "term found in document",
			postings: []PostingInfo{
				{DocumentID: 0, TermFrequency: 5, FieldID: 0},
				{DocumentID: 1, TermFrequency: 3, FieldID: 1},
			},
			documentID:    0,
			expectedTF:    5.0,
			expectedFID:   0,
			expectedFound: true,
			description:   "Should find term in first posting",
		},
		{
			name: "term found in middle",
			postings: []PostingInfo{
				{DocumentID: 0, TermFrequency: 2, FieldID: 0},
				{DocumentID: 1, TermFrequency: 7, FieldID: 1},
				{DocumentID: 2, TermFrequency: 4, FieldID: 0},
			},
			documentID:    1,
			expectedTF:    7.0,
			expectedFID:   1,
			expectedFound: true,
			description:   "Should find term in middle posting",
		},
		{
			name: "term not found",
			postings: []PostingInfo{
				{DocumentID: 0, TermFrequency: 2, FieldID: 0},
				{DocumentID: 1, TermFrequency: 3, FieldID: 1},
			},
			documentID:    99,
			expectedTF:    0.0,
			expectedFID:   0,
			expectedFound: false,
			description:   "Should return not found for missing documentID",
		},
		{
			name:          "empty postings",
			postings:      []PostingInfo{},
			documentID:    0,
			expectedTF:    0.0,
			expectedFID:   0,
			expectedFound: false,
			description:   "Should return not found for empty postings",
		},
		{
			name: "term at end of list",
			postings: []PostingInfo{
				{DocumentID: 0, TermFrequency: 2, FieldID: 0},
				{DocumentID: 1, TermFrequency: 3, FieldID: 1},
				{DocumentID: 2, TermFrequency: 9, FieldID: 2},
			},
			documentID:    2,
			expectedTF:    9.0,
			expectedFID:   2,
			expectedFound: true,
			description:   "Should find term at end of postings",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tf, fid, found := findTermFrequency(tc.postings, tc.documentID)

			if found != tc.expectedFound {
				t.Errorf("Expected found=%v, got found=%v", tc.expectedFound, found)
			}

			if tf != tc.expectedTF {
				t.Errorf("Expected TF=%f, got TF=%f", tc.expectedTF, tf)
			}

			if fid != tc.expectedFID {
				t.Errorf("Expected FieldID=%d, got FieldID=%d", tc.expectedFID, fid)
			}
		})
	}
}

func TestFindBestFuzzyMatch_AllBranches(t *testing.T) {
	t.Parallel()

	vocabulary := []string{
		"configuration",
		"configure",
		"configured",
		"container",
		"content",
	}

	testCases := []struct {
		name           string
		queryTerm      string
		expectTerm     string
		description    string
		vocabulary     []string
		maxDistance    int
		expectMinScore float64
	}{
		{
			name:           "empty query term",
			queryTerm:      "",
			vocabulary:     vocabulary,
			maxDistance:    2,
			expectTerm:     "",
			expectMinScore: 0.0,
			description:    "Should return empty for empty query",
		},
		{
			name:           "empty vocabulary",
			queryTerm:      "test",
			vocabulary:     []string{},
			maxDistance:    2,
			expectTerm:     "",
			expectMinScore: 0.0,
			description:    "Should return empty for empty vocabulary",
		},
		{
			name:           "exact distance 1 match",
			queryTerm:      "configurtion",
			vocabulary:     vocabulary,
			maxDistance:    2,
			expectTerm:     "configuration",
			expectMinScore: 0.90,
			description:    "Should use Ukkonen for distance 1 match",
		},
		{
			name:           "distance 2 falls through to Jaro-Winkler",
			queryTerm:      "confgrtion",
			vocabulary:     vocabulary,
			maxDistance:    3,
			expectTerm:     "configuration",
			expectMinScore: 0.70,
			description:    "Should fall through to Jaro-Winkler for distance > 1",
		},
		{
			name:           "no matches found",
			queryTerm:      "xyz",
			vocabulary:     vocabulary,
			maxDistance:    1,
			expectTerm:     "",
			expectMinScore: 0.0,
			description:    "Should return empty when no match found",
		},
		{
			name:           "very different term uses Jaro-Winkler",
			queryTerm:      "konfig",
			vocabulary:     vocabulary,
			maxDistance:    2,
			expectTerm:     "",
			expectMinScore: 0.0,
			description:    "Should return empty for very different terms",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			term, score := findBestFuzzyMatch(tc.queryTerm, tc.vocabulary, tc.maxDistance)

			if term != tc.expectTerm {
				t.Errorf("Expected term %q, got %q", tc.expectTerm, term)
			}

			if score < tc.expectMinScore {
				t.Errorf("Expected score >= %f, got %f", tc.expectMinScore, score)
			}

			t.Logf("Query %q → %q (score: %.3f)", tc.queryTerm, term, score)
		})
	}
}

func TestGetFieldWeight_AllFields(t *testing.T) {
	t.Parallel()

	scorer := NewBM25Scorer(BM25DefaultK1, BM25DefaultB)

	testCases := []struct {
		name           string
		description    string
		config         search_dto.SearchConfig
		expectedWeight float64
		fieldID        uint8
	}{
		{
			name:    "title field (ID 0)",
			fieldID: 0,
			config: search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{Name: "title", Weight: 3.0},
					{Name: "content", Weight: 1.0},
					{Name: "excerpt", Weight: 2.0},
				},
			},
			expectedWeight: 3.0,
			description:    "Should use configured weight for title",
		},
		{
			name:    "content field (ID 1)",
			fieldID: 1,
			config: search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{Name: "title", Weight: 3.0},
					{Name: "content", Weight: 1.5},
					{Name: "excerpt", Weight: 2.0},
				},
			},
			expectedWeight: 1.5,
			description:    "Should use configured weight for content",
		},
		{
			name:    "excerpt field (ID 2)",
			fieldID: 2,
			config: search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{Name: "title", Weight: 3.0},
					{Name: "content", Weight: 1.0},
					{Name: "excerpt", Weight: 2.5},
				},
			},
			expectedWeight: 2.5,
			description:    "Should use configured weight for excerpt",
		},
		{
			name:    "unknown field ID",
			fieldID: 99,
			config: search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{Name: "title", Weight: 3.0},
				},
			},
			expectedWeight: DefaultFieldWeightContent,
			description:    "Should use default weight for unknown field",
		},
		{
			name:    "field not in config",
			fieldID: 0,
			config: search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{Name: "content", Weight: 1.0},
				},
			},
			expectedWeight: DefaultFieldWeightContent,
			description:    "Should use default when field not in config",
		},
		{
			name:    "zero weight falls back to default",
			fieldID: 0,
			config: search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{Name: "title", Weight: 0.0},
				},
			},
			expectedWeight: DefaultFieldWeightContent,
			description:    "Should use default for zero weight",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			weight := scorer.getFieldWeight(tc.fieldID, buildFieldWeightMap(tc.config))

			if weight != tc.expectedWeight {
				t.Errorf("Expected weight %f, got %f", tc.expectedWeight, weight)
			}
		})
	}
}

func TestGetFieldName_AllFields(t *testing.T) {
	t.Parallel()

	scorer := NewBM25Scorer(BM25DefaultK1, BM25DefaultB)

	testCases := []struct {
		expectedName string
		fieldID      uint8
	}{
		{fieldID: 0, expectedName: "title"},
		{fieldID: 1, expectedName: "content"},
		{fieldID: 2, expectedName: "excerpt"},
		{fieldID: 3, expectedName: "unknown"},
		{fieldID: 99, expectedName: "unknown"},
		{fieldID: 255, expectedName: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedName, func(t *testing.T) {
			t.Parallel()

			name := scorer.getFieldName(tc.fieldID)

			if name != tc.expectedName {
				t.Errorf("Expected field name %q for ID %d, got %q",
					tc.expectedName, tc.fieldID, name)
			}
		})
	}
}

func TestDeduplicateTerms(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		description    string
		queryTerms     []string
		expectedUnique int
	}{
		{
			name:           "no duplicates",
			queryTerms:     []string{"go", "programming", "tutorial"},
			expectedUnique: 3,
			description:    "Should preserve all unique terms",
		},
		{
			name:           "with duplicates",
			queryTerms:     []string{"go", "go", "programming", "go", "tutorial"},
			expectedUnique: 3,
			description:    "Should deduplicate repeated terms",
		},
		{
			name:           "all same term",
			queryTerms:     []string{"go", "go", "go"},
			expectedUnique: 1,
			description:    "Should reduce to single term",
		},
		{
			name:           "empty list",
			queryTerms:     []string{},
			expectedUnique: 0,
			description:    "Should handle empty list",
		},
		{
			name:           "single term",
			queryTerms:     []string{"go"},
			expectedUnique: 1,
			description:    "Should handle single term",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uniqueTerms := deduplicateTerms(tc.queryTerms)

			if len(uniqueTerms) != tc.expectedUnique {
				t.Errorf("Expected %d unique terms, got %d",
					tc.expectedUnique, len(uniqueTerms))
			}

			for _, term := range tc.queryTerms {
				if !uniqueTerms[term] {
					t.Errorf("Expected term %q to be in result", term)
				}
			}
		})
	}
}

func TestCalculateLengthNorm_ErrorHandling(t *testing.T) {
	t.Parallel()

	scorer := NewBM25Scorer(BM25DefaultK1, BM25DefaultB)

	testCases := []struct {
		reader      IndexReaderPort
		name        string
		description string
		documentID  uint32
		expectError bool
	}{
		{
			name: "document not found",
			reader: &mockIndexReader{
				docs: map[uint32]DocMetadataInfo{},
			},
			documentID:  99,
			expectError: true,
			description: "Should return error when document not found",
		},
		{
			name: "successful calculation",
			reader: &mockIndexReader{
				docs: map[uint32]DocMetadataInfo{
					0: {DocumentID: 0, FieldLength: 100},
				},
				corpusStats: CorpusStats{
					TotalDocuments:     10,
					AverageFieldLength: 50.0,
					VocabSize:          1000,
				},
			},
			documentID:  0,
			expectError: false,
			description: "Should calculate norm successfully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := scorer.calculateLengthNorm(tc.documentID, tc.reader)

			if tc.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestScoreTerms_ContextCancellation(t *testing.T) {
	t.Parallel()

	scorer := NewBM25Scorer(BM25DefaultK1, BM25DefaultB)

	reader := &mockIndexReader{
		terms: map[string][]PostingInfo{
			"test": {{DocumentID: 0, TermFrequency: 5, FieldID: 0}},
		},
		idfs: map[string]float64{
			"test": 1.0,
		},
		docs: map[uint32]DocMetadataInfo{
			0: {DocumentID: 0, FieldLength: 100},
		},
		corpusStats: CorpusStats{
			TotalDocuments:     1,
			AverageFieldLength: 100.0,
			VocabSize:          1,
		},
	}

	config := search_dto.SearchConfig{
		Fields: []search_dto.SearchField{
			{Name: "title", Weight: 1.0},
		},
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	uniqueTerms := map[string]bool{"test": true}

	_, err := scorer.scoreTerms(ctx, uniqueTerms, 0, 1.0, reader, config)

	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
