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
	"strings"
	"testing"

	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

type mockIndexReader struct {
	err           error
	docs          map[uint32]DocMetadataInfo
	terms         map[string][]PostingInfo
	idfs          map[string]float64
	phoneticTerms map[string][]string
	language      string
	allTerms      []string
	corpusStats   CorpusStats
	mode          search_schema_gen.SearchMode
}

func (m *mockIndexReader) LoadIndex(data []byte) error {
	return nil
}

func (m *mockIndexReader) GetTermPostings(term string) ([]PostingInfo, float64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	postings, ok := m.terms[term]
	if !ok {
		return nil, 0, errors.New("term not found")
	}
	idf := m.idfs[term]
	return postings, idf, nil
}

func (m *mockIndexReader) GetDocMetadata(documentID uint32) (DocMetadataInfo, error) {
	document, ok := m.docs[documentID]
	if !ok {
		return DocMetadataInfo{}, errors.New("document not found")
	}
	return document, nil
}

func (m *mockIndexReader) GetCorpusStats() CorpusStats {
	return m.corpusStats
}

func (m *mockIndexReader) GetMode() search_schema_gen.SearchMode {
	return m.mode
}

func (m *mockIndexReader) GetLanguage() string {
	return m.language
}

func (m *mockIndexReader) FindPhoneticTerms(phoneticCode string) ([]string, error) {
	terms, ok := m.phoneticTerms[phoneticCode]
	if !ok {
		return nil, errors.New("phonetic code not found")
	}
	return terms, nil
}

func (m *mockIndexReader) GetAllTerms() ([]string, error) {
	return m.allTerms, nil
}

func (m *mockIndexReader) FindTermsWithPrefix(prefix string) ([]string, error) {
	var result []string
	for _, term := range m.allTerms {
		if strings.HasPrefix(term, prefix) {
			result = append(result, term)
		}
	}
	return result, nil
}

func TestNewBM25Scorer(t *testing.T) {
	testCases := []struct {
		name   string
		k1     float64
		b      float64
		wantK1 float64
		wantB  float64
	}{
		{
			name:   "valid parameters",
			k1:     1.5,
			b:      0.8,
			wantK1: 1.5,
			wantB:  0.8,
		},
		{
			name:   "zero parameters default to standard values",
			k1:     0,
			b:      0,
			wantK1: 1.2,
			wantB:  0.75,
		},
		{
			name:   "negative parameters default to standard values",
			k1:     -1.0,
			b:      -0.5,
			wantK1: 1.2,
			wantB:  0.75,
		},
		{
			name:   "partial defaults",
			k1:     2.0,
			b:      0,
			wantK1: 2.0,
			wantB:  0.75,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scorer := NewBM25Scorer(tc.k1, tc.b)

			if scorer.k1 != tc.wantK1 {
				t.Errorf("k1: got %.2f, want %.2f", scorer.k1, tc.wantK1)
			}

			if scorer.b != tc.wantB {
				t.Errorf("b: got %.2f, want %.2f", scorer.b, tc.wantB)
			}
		})
	}
}

func TestBM25Scorer_Score(t *testing.T) {
	mockReader := &mockIndexReader{
		docs: map[uint32]DocMetadataInfo{
			0: {
				DocumentID:  0,
				FieldLength: 100,
				Route:       "/doc0",
			},
			1: {
				DocumentID:  1,
				FieldLength: 200,
				Route:       "/doc1",
			},
		},
		terms: map[string][]PostingInfo{
			"test": {
				{
					DocumentID:    0,
					TermFrequency: 5,
					FieldID:       0,
				},
				{
					DocumentID:    1,
					TermFrequency: 2,
					FieldID:       1,
				},
			},
			"query": {
				{
					DocumentID:    0,
					TermFrequency: 3,
					FieldID:       1,
				},
			},
		},
		idfs: map[string]float64{
			"test":  1.5,
			"query": 2.0,
		},
		corpusStats: CorpusStats{
			TotalDocuments:     2,
			AverageFieldLength: 150,
			VocabSize:          10,
		},
		mode:     search_schema_gen.SearchModeFast,
		language: "english",
	}

	testCases := []struct {
		name       string
		queryTerms []string
		config     search_dto.SearchConfig
		wantScore  float64
		documentID uint32
		wantErr    bool
	}{
		{
			name:       "single term match",
			queryTerms: []string{"test"},
			documentID: 0,
			config:     search_dto.DefaultSearchConfig("test"),
			wantScore:  0.0,
			wantErr:    false,
		},
		{
			name:       "multiple terms",
			queryTerms: []string{"test", "query"},
			documentID: 0,
			config:     search_dto.DefaultSearchConfig("test query"),
			wantScore:  0.0,
			wantErr:    false,
		},
		{
			name:       "term not in document",
			queryTerms: []string{"missing"},
			documentID: 0,
			config:     search_dto.DefaultSearchConfig("missing"),
			wantScore:  0.0,
			wantErr:    false,
		},
		{
			name:       "document not found",
			queryTerms: []string{"test"},
			documentID: 999,
			config:     search_dto.DefaultSearchConfig("test"),
			wantScore:  0.0,
			wantErr:    true,
		},
		{
			name:       "duplicate query terms deduplication",
			queryTerms: []string{"test", "test", "test"},
			documentID: 0,
			config:     search_dto.DefaultSearchConfig("test test test"),
			wantScore:  0.0,
			wantErr:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scorer := NewBM25Scorer(1.2, 0.75)
			result, err := scorer.Score(context.Background(), tc.queryTerms, tc.documentID, mockReader, tc.config)

			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tc.name == "single term match" || tc.name == "multiple terms" || tc.name == "duplicate query terms deduplication" {
				if result.Score <= 0 {
					t.Errorf("Expected positive score, got %.4f", result.Score)
				}

				if result.FieldScores == nil {
					t.Error("Expected FieldScores to be non-nil")
				}
			}

			if tc.name == "term not in document" {
				if result.Score != 0 {
					t.Errorf("Expected zero score for missing term, got %.4f", result.Score)
				}
			}
		})
	}
}

func TestBM25Scorer_Score_WithFieldWeights(t *testing.T) {
	mockReader := &mockIndexReader{
		docs: map[uint32]DocMetadataInfo{
			0: {
				DocumentID:  0,
				FieldLength: 100,
				Route:       "/doc0",
			},
		},
		terms: map[string][]PostingInfo{
			"test": {
				{
					DocumentID:    0,
					TermFrequency: 5,
					FieldID:       0,
				},
			},
		},
		idfs: map[string]float64{
			"test": 1.5,
		},
		corpusStats: CorpusStats{
			TotalDocuments:     1,
			AverageFieldLength: 100,
			VocabSize:          5,
		},
		mode:     search_schema_gen.SearchModeFast,
		language: "english",
	}

	scorer := NewBM25Scorer(1.2, 0.75)

	configNoWeight := search_dto.DefaultSearchConfig("test")
	resultNoWeight, err := scorer.Score(context.Background(), []string{"test"}, 0, mockReader, configNoWeight)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	configWithWeight := search_dto.SearchConfig{
		Query: "test",
		Fields: []search_dto.SearchField{
			{
				Name:   "title",
				Weight: 2.0,
			},
		},
	}
	resultWithWeight, err := scorer.Score(context.Background(), []string{"test"}, 0, mockReader, configWithWeight)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resultWithWeight.Score <= resultNoWeight.Score {
		t.Errorf("Expected weighted score (%.4f) to be greater than unweighted score (%.4f)", resultWithWeight.Score, resultNoWeight.Score)
	}

	if len(resultWithWeight.FieldScores) == 0 {
		t.Error("Expected FieldScores to be populated")
	}

	if _, ok := resultWithWeight.FieldScores["title"]; !ok {
		t.Error("Expected 'title' field score to be present")
	}
}

func TestBM25Scorer_Score_ContextCancellation(t *testing.T) {
	mockReader := &mockIndexReader{
		docs: map[uint32]DocMetadataInfo{
			0: {
				DocumentID:  0,
				FieldLength: 100,
				Route:       "/doc0",
			},
		},
		terms: map[string][]PostingInfo{
			"test": {
				{
					DocumentID:    0,
					TermFrequency: 5,
					FieldID:       0,
				},
			},
		},
		idfs: map[string]float64{
			"test": 1.5,
		},
		corpusStats: CorpusStats{
			TotalDocuments:     1,
			AverageFieldLength: 100,
			VocabSize:          5,
		},
	}

	scorer := NewBM25Scorer(1.2, 0.75)
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	config := search_dto.DefaultSearchConfig("test")
	result, err := scorer.Score(ctx, []string{"test"}, 0, mockReader, config)

	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if result.Score != 0 {
		t.Errorf("Expected zero score on error, got %.4f", result.Score)
	}
}

func TestBM25Scorer_getFieldWeight(t *testing.T) {
	scorer := NewBM25Scorer(1.2, 0.75)

	testCases := []struct {
		name       string
		config     search_dto.SearchConfig
		wantWeight float64
		fieldID    uint8
	}{
		{
			name:    "title field with weight",
			fieldID: 0,
			config: search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{
						Name:   "title",
						Weight: 2.5,
					},
				},
			},
			wantWeight: 2.5,
		},
		{
			name:    "content field with default weight",
			fieldID: 1,
			config: search_dto.SearchConfig{
				Fields: []search_dto.SearchField{
					{
						Name:   "content",
						Weight: 0,
					},
				},
			},
			wantWeight: 1.0,
		},
		{
			name:       "field not in config",
			fieldID:    2,
			config:     search_dto.SearchConfig{},
			wantWeight: 1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			weight := scorer.getFieldWeight(tc.fieldID, buildFieldWeightMap(tc.config))
			if weight != tc.wantWeight {
				t.Errorf("getFieldWeight: got %.2f, want %.2f", weight, tc.wantWeight)
			}
		})
	}
}

func TestBM25Scorer_getFieldName(t *testing.T) {
	scorer := NewBM25Scorer(1.2, 0.75)

	testCases := []struct {
		wantName string
		fieldID  uint8
	}{
		{
			fieldID:  0,
			wantName: "title",
		},
		{
			fieldID:  1,
			wantName: "content",
		},
		{
			fieldID:  2,
			wantName: "excerpt",
		},
		{
			fieldID:  99,
			wantName: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.wantName, func(t *testing.T) {
			name := scorer.getFieldName(tc.fieldID)
			if name != tc.wantName {
				t.Errorf("getFieldName(%d): got %s, want %s", tc.fieldID, name, tc.wantName)
			}
		})
	}
}

func TestBM25Scorer_ScoreWithExplanation(t *testing.T) {
	mockReader := &mockIndexReader{
		docs: map[uint32]DocMetadataInfo{
			0: {
				DocumentID:  0,
				FieldLength: 100,
				Route:       "/doc0",
			},
		},
		terms: map[string][]PostingInfo{
			"test": {
				{
					DocumentID:    0,
					TermFrequency: 5,
					FieldID:       0,
				},
			},
			"query": {
				{
					DocumentID:    0,
					TermFrequency: 3,
					FieldID:       1,
				},
			},
		},
		idfs: map[string]float64{
			"test":  1.5,
			"query": 2.0,
		},
		corpusStats: CorpusStats{
			TotalDocuments:     1,
			AverageFieldLength: 100,
			VocabSize:          5,
		},
		mode:     search_schema_gen.SearchModeFast,
		language: "english",
	}

	scorer := NewBM25Scorer(1.2, 0.75)
	config := search_dto.DefaultSearchConfig("test query")

	score, explanation, err := scorer.ScoreWithExplanation(
		context.Background(),
		[]string{"test", "query"},
		0,
		mockReader,
		config,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if score <= 0 {
		t.Errorf("Expected positive score, got %.4f", score)
	}

	if explanation == nil {
		t.Fatal("Expected explanation, got nil")
	}

	if explanation.DocumentID != 0 {
		t.Errorf("Expected DocumentID 0, got %d", explanation.DocumentID)
	}

	if len(explanation.QueryTerms) != 2 {
		t.Errorf("Expected 2 query terms, got %d", len(explanation.QueryTerms))
	}

	if explanation.DocumentLength != 100 {
		t.Errorf("Expected DocumentLength 100, got %d", explanation.DocumentLength)
	}

	if explanation.TotalScore != score {
		t.Errorf("Expected TotalScore to match score: %.4f != %.4f", explanation.TotalScore, score)
	}

	if len(explanation.TermScores) != 2 {
		t.Errorf("Expected 2 term scores, got %d", len(explanation.TermScores))
	}

	for _, termScore := range explanation.TermScores {
		if !termScore.Found {
			t.Errorf("Expected term '%s' to be found", termScore.Term)
		}
	}

	if len(explanation.FieldScores) == 0 {
		t.Error("Expected field scores to be populated")
	}
}

func TestBM25Scorer_ScoreWithExplanation_MissingTerm(t *testing.T) {
	mockReader := &mockIndexReader{
		docs: map[uint32]DocMetadataInfo{
			0: {
				DocumentID:  0,
				FieldLength: 100,
				Route:       "/doc0",
			},
		},
		terms: map[string][]PostingInfo{
			"test": {
				{
					DocumentID:    0,
					TermFrequency: 5,
					FieldID:       0,
				},
			},
			"found": {
				{
					DocumentID:    1,
					TermFrequency: 3,
					FieldID:       0,
				},
			},
		},
		idfs: map[string]float64{
			"test":  1.5,
			"found": 2.0,
		},
		corpusStats: CorpusStats{
			TotalDocuments:     1,
			AverageFieldLength: 100,
			VocabSize:          5,
		},
	}

	scorer := NewBM25Scorer(1.2, 0.75)
	config := search_dto.DefaultSearchConfig("test found")

	_, explanation, err := scorer.ScoreWithExplanation(
		context.Background(),
		[]string{"test", "found"},
		0,
		mockReader,
		config,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if explanation == nil {
		t.Fatal("Expected explanation, got nil")
	}

	if len(explanation.TermScores) != 2 {
		t.Errorf("Expected 2 term scores, got %d", len(explanation.TermScores))
	}

	foundCount := 0
	notFoundCount := 0
	for _, termScore := range explanation.TermScores {
		if termScore.Found {
			foundCount++
		} else {
			notFoundCount++
		}
	}

	if foundCount != 1 {
		t.Errorf("Expected 1 found term, got %d", foundCount)
	}

	if notFoundCount != 1 {
		t.Errorf("Expected 1 not found term, got %d", notFoundCount)
	}
}

func TestScoreExplanation_String(t *testing.T) {
	explanation := &ScoreExplanation{
		DocumentID:            0,
		QueryTerms:            []string{"test", "query"},
		DocumentLength:        100,
		AverageDocumentLength: 150,
		LengthNorm:            1.2,
		TermScores: []TermScoreDetail{
			{
				Term:        "test",
				Found:       true,
				TF:          5,
				IDF:         1.5,
				BaseScore:   7.5,
				FieldID:     0,
				FieldName:   "title",
				FieldWeight: 2.0,
				FinalScore:  15.0,
			},
			{
				Term:  "missing",
				Found: false,
			},
		},
		FieldScores: map[string]float64{
			"title": 15.0,
		},
		TotalScore: 15.0,
	}

	result := explanation.String()

	expectedStrings := []string{
		"Document 0",
		"test",
		"query",
		"Document Length: 100",
		"NOT FOUND",
		"Total Score",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected string to contain '%s', got:\n%s", expected, result)
		}
	}
}

func TestBM25Scorer_Score_FieldScoresPopulated(t *testing.T) {

	mockReader := &mockIndexReader{
		docs: map[uint32]DocMetadataInfo{
			0: {
				DocumentID:  0,
				FieldLength: 100,
				Route:       "/doc0",
			},
		},
		terms: map[string][]PostingInfo{
			"title": {
				{
					DocumentID:    0,
					TermFrequency: 3,
					FieldID:       0,
				},
			},
			"content": {
				{
					DocumentID:    0,
					TermFrequency: 5,
					FieldID:       1,
				},
			},
			"excerpt": {
				{
					DocumentID:    0,
					TermFrequency: 2,
					FieldID:       2,
				},
			},
		},
		idfs: map[string]float64{
			"title":   1.5,
			"content": 1.2,
			"excerpt": 1.8,
		},
		corpusStats: CorpusStats{
			TotalDocuments:     1,
			AverageFieldLength: 100,
			VocabSize:          10,
		},
		mode:     search_schema_gen.SearchModeFast,
		language: "english",
	}

	scorer := NewBM25Scorer(1.2, 0.75)
	config := search_dto.DefaultSearchConfig("title content excerpt")

	result, err := scorer.Score(
		context.Background(),
		[]string{"title", "content", "excerpt"},
		0,
		mockReader,
		config,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Score <= 0 {
		t.Errorf("Expected positive aggregate score, got %.4f", result.Score)
	}

	if result.FieldScores == nil {
		t.Fatal("Expected FieldScores to be non-nil")
	}

	expectedFields := []string{"title", "content", "excerpt"}
	for _, field := range expectedFields {
		score, ok := result.FieldScores[field]
		if !ok {
			t.Errorf("Expected field '%s' to be present in FieldScores", field)
			continue
		}
		if score <= 0 {
			t.Errorf("Expected positive score for field '%s', got %.4f", field, score)
		}
	}

	var fieldScoreSum float64
	for _, score := range result.FieldScores {
		fieldScoreSum += score
	}

	const epsilon = 0.0001
	diff := result.Score - fieldScoreSum
	if diff < -epsilon || diff > epsilon {
		t.Errorf("Sum of field scores (%.4f) should equal total score (%.4f)", fieldScoreSum, result.Score)
	}
}

func TestBM25Scorer_Score_FieldScoresConsistentWithExplanation(t *testing.T) {

	mockReader := &mockIndexReader{
		docs: map[uint32]DocMetadataInfo{
			0: {
				DocumentID:  0,
				FieldLength: 100,
				Route:       "/doc0",
			},
		},
		terms: map[string][]PostingInfo{
			"test": {
				{
					DocumentID:    0,
					TermFrequency: 5,
					FieldID:       0,
				},
			},
			"query": {
				{
					DocumentID:    0,
					TermFrequency: 3,
					FieldID:       1,
				},
			},
		},
		idfs: map[string]float64{
			"test":  1.5,
			"query": 2.0,
		},
		corpusStats: CorpusStats{
			TotalDocuments:     1,
			AverageFieldLength: 100,
			VocabSize:          5,
		},
		mode:     search_schema_gen.SearchModeFast,
		language: "english",
	}

	scorer := NewBM25Scorer(1.2, 0.75)
	config := search_dto.DefaultSearchConfig("test query")
	queryTerms := []string{"test", "query"}

	scoreResult, err := scorer.Score(context.Background(), queryTerms, 0, mockReader, config)
	if err != nil {
		t.Fatalf("Score() error: %v", err)
	}

	explanationScore, explanation, err := scorer.ScoreWithExplanation(context.Background(), queryTerms, 0, mockReader, config)
	if err != nil {
		t.Fatalf("ScoreWithExplanation() error: %v", err)
	}

	const epsilon = 0.0001
	if diff := scoreResult.Score - explanationScore; diff < -epsilon || diff > epsilon {
		t.Errorf("Score mismatch: Score()=%.4f, ScoreWithExplanation()=%.4f", scoreResult.Score, explanationScore)
	}

	for field, scoreFromResult := range scoreResult.FieldScores {
		scoreFromExplanation, ok := explanation.FieldScores[field]
		if !ok {
			t.Errorf("Field '%s' present in Score() but not in ScoreWithExplanation()", field)
			continue
		}
		if diff := scoreFromResult - scoreFromExplanation; diff < -epsilon || diff > epsilon {
			t.Errorf("Field score mismatch for '%s': Score()=%.4f, ScoreWithExplanation()=%.4f",
				field, scoreFromResult, scoreFromExplanation)
		}
	}

	for field := range explanation.FieldScores {
		if _, ok := scoreResult.FieldScores[field]; !ok {
			t.Errorf("Field '%s' present in ScoreWithExplanation() but not in Score()", field)
		}
	}
}
