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
	"fmt"
	"strings"

	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/wdk/safeconv"
)

// BM25Scorer implements the ScorerPort interface using the BM25 (Best
// Matching 25) ranking algorithm. BM25 is a probabilistic information
// retrieval function used by search engines to estimate the relevance of
// documents to a given search query.
//
// The algorithm is based on the probabilistic retrieval framework developed
// by Robertson and Sparck Jones in the 1970s and refined over time.
//
// References:
//   - Robertson, S. & Zaragoza, H. (2009). "The Probabilistic Relevance
//     Framework: BM25 and Beyond"
//   - https://en.wikipedia.org/wiki/Okapi_BM25
type BM25Scorer struct {
	// k1 controls term frequency saturation; typical value is 1.2.
	k1 float64

	// b controls length normalisation; typical value is 0.75.
	b float64
}

// NewBM25Scorer creates a new BM25 scorer with the given parameters.
// If parameters are <= 0, defaults are used (k1=1.2, b=0.75).
//
// Takes k1 (float64) which controls term frequency saturation.
// Takes b (float64) which controls document length normalisation.
//
// Returns *BM25Scorer which is configured with the specified or default values.
func NewBM25Scorer(k1, b float64) *BM25Scorer {
	if k1 <= 0 {
		k1 = BM25DefaultK1
	}
	if b <= 0 {
		b = BM25DefaultB
	}

	return &BM25Scorer{
		k1: k1,
		b:  b,
	}
}

// Score calculates the BM25 relevance score for a document given query terms.
//
// BM25 Formula:
//
//	score(D, Q) = SUM IDF(qi) * (f(qi, D) * (k1 + 1)) / (f(qi, D) + k1 * (1 - b + b * (|D| / avgdl)))
//
// Where:
//   - D: document being scored
//   - Q: query (set of terms)
//   - qi: i-th query term
//   - f(qi, D): frequency of qi in document D (term frequency)
//   - |D|: length of document D (in words)
//   - avgdl: average document length in the collection
//   - k1: controls non-linear term frequency normalisation (saturation)
//   - b: controls to what degree document length normalises tf values
//   - IDF(qi): inverse document frequency of query term qi
//
// Takes queryTerms ([]string) which contains the search terms to score against.
// Takes documentID (uint32) which identifies the document to score.
// Takes reader (IndexReaderPort) which provides access to the search index.
// Takes config (search_dto.SearchConfig) which specifies search behaviour.
//
// Returns ScoreResult which contains the calculated relevance score.
// Returns error when the document length normalisation cannot be calculated.
func (s *BM25Scorer) Score(
	ctx context.Context,
	queryTerms []string,
	documentID uint32,
	reader IndexReaderPort,
	config search_dto.SearchConfig,
) (ScoreResult, error) {
	lengthNorm, err := s.calculateLengthNorm(documentID, reader)
	if err != nil {
		return ScoreResult{}, err
	}

	uniqueTerms := deduplicateTerms(queryTerms)

	return s.scoreTerms(ctx, uniqueTerms, documentID, lengthNorm, reader, config)
}

// ScoreWithExplanation calculates the BM25 score and returns a detailed
// explanation of why a document was ranked the way it was.
//
// Takes queryTerms ([]string) which specifies the search terms to score.
// Takes documentID (uint32) which identifies the document to score.
// Takes reader (IndexReaderPort) which provides access to index data.
// Takes config (search_dto.SearchConfig) which controls scoring behaviour.
//
// Returns float64 which is the calculated BM25 score.
// Returns *ScoreExplanation which provides detailed breakdown of the score.
// Returns error when document metadata cannot be retrieved.
func (s *BM25Scorer) ScoreWithExplanation(
	ctx context.Context,
	queryTerms []string,
	documentID uint32,
	reader IndexReaderPort,
	config search_dto.SearchConfig,
) (float64, *ScoreExplanation, error) {
	docMeta, err := reader.GetDocMetadata(documentID)
	if err != nil {
		return 0, nil, fmt.Errorf("getting doc metadata: %w", err)
	}

	corpusStats := reader.GetCorpusStats()
	docLength := float64(docMeta.FieldLength)
	avgDocumentLength := float64(corpusStats.AverageFieldLength)
	lengthNorm := s.k1 * (1 - s.b + s.b*(docLength/avgDocumentLength))

	explanation := &ScoreExplanation{
		DocumentID:            documentID,
		QueryTerms:            queryTerms,
		DocumentLength:        docMeta.FieldLength,
		AverageDocumentLength: corpusStats.AverageFieldLength,
		LengthNorm:            lengthNorm,
		TermScores:            make([]TermScoreDetail, 0),
		FieldScores:           make(map[string]float64),
		TotalScore:            0.0,
	}

	totalScore, err := s.scoreTermsWithExplanation(ctx, queryTerms, documentID, lengthNorm, reader, config, explanation)
	if err != nil {
		return 0, nil, err
	}

	explanation.TotalScore = totalScore

	return totalScore, explanation, nil
}

// ScoreExplanation provides a breakdown of how a BM25 score was calculated.
// It implements fmt.Stringer.
type ScoreExplanation struct {
	// FieldScores maps each field name to its BM25 score.
	FieldScores map[string]float64

	// QueryTerms holds the search terms from the original query.
	QueryTerms []string

	// TermScores holds the scoring details for each query term.
	TermScores []TermScoreDetail

	// LengthNorm is the document length normalisation factor.
	LengthNorm float64

	// TotalScore is the final relevance score after combining all term scores.
	TotalScore float64

	// AverageDocumentLength is the mean document length across
	// all documents in the corpus.
	AverageDocumentLength float32

	// DocumentID is the unique identifier of the document being scored.
	DocumentID uint32

	// DocumentLength is the number of terms in the document.
	DocumentLength uint32
}

// String returns a human-readable explanation of the score.
//
// Returns string which contains the formatted BM25 score breakdown,
// including term contributions, field scores, and the total score.
func (e *ScoreExplanation) String() string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "BM25 Score Explanation for Document %d\n", e.DocumentID)
	_, _ = fmt.Fprintf(&builder, "Query: %v\n", e.QueryTerms)
	_, _ = fmt.Fprintf(&builder, "Document Length: %d (avg: %.2f, norm: %.4f)\n\n",
		e.DocumentLength, e.AverageDocumentLength, e.LengthNorm)

	builder.WriteString("Term Contributions:\n")
	for _, term := range e.TermScores {
		if !term.Found {
			_, _ = fmt.Fprintf(&builder, "  '%s': NOT FOUND\n", term.Term)
			continue
		}
		_, _ = fmt.Fprintf(&builder, "  '%s': TF=%d, IDF=%.4f, Field=%s (weight=%.2f), Score=%.4f\n",
			term.Term, term.TF, term.IDF, term.FieldName, term.FieldWeight, term.FinalScore)
	}

	builder.WriteString("\nField Scores:\n")
	for field, score := range e.FieldScores {
		_, _ = fmt.Fprintf(&builder, "  %s: %.4f\n", field, score)
	}

	_, _ = fmt.Fprintf(&builder, "\nTotal Score: %.4f\n", e.TotalScore)

	return builder.String()
}

// TermScoreDetail explains the score contribution of a single query term.
type TermScoreDetail struct {
	// Term is the search term being scored.
	Term string

	// FieldName is the name of the field where this term was found.
	FieldName string

	// IDF is the inverse document frequency for the term.
	IDF float64

	// BaseScore is the BM25 term score (IDF x TF saturation
	// with length normalisation), before field weighting.
	BaseScore float64

	// FieldWeight is the multiplier applied to the field type.
	FieldWeight float64

	// FinalScore is the term's score after field weighting is applied.
	FinalScore float64

	// TF is the number of times this term appears in the document.
	TF uint16

	// FieldID identifies the field within the term that was scored.
	FieldID uint8

	// Found indicates whether this term was found in the document.
	Found bool
}

// calculateLengthNorm computes the BM25 length normalisation factor using the
// formula: k1 * (1 - b + b * (|D| / avgdl)).
//
// Takes documentID (uint32) which identifies the document to
// calculate the norm for.
// Takes reader (IndexReaderPort) which provides access to document metadata and
// corpus statistics.
//
// Returns float64 which is the computed length normalisation factor.
// Returns error when the document metadata cannot be retrieved.
func (s *BM25Scorer) calculateLengthNorm(documentID uint32, reader IndexReaderPort) (float64, error) {
	docMeta, err := reader.GetDocMetadata(documentID)
	if err != nil {
		return 0, fmt.Errorf("getting doc metadata: %w", err)
	}

	corpusStats := reader.GetCorpusStats()
	docLength := float64(docMeta.FieldLength)
	avgDocumentLength := float64(corpusStats.AverageFieldLength)

	return s.k1 * (1 - s.b + s.b*(docLength/avgDocumentLength)), nil
}

// scoreTerms calculates the total BM25 score across all query terms.
//
// Takes uniqueTerms (map[string]bool) which contains the unique terms to score.
// Takes documentID (uint32) which identifies the document to score.
// Takes lengthNorm (float64) which provides the length normalisation factor.
// Takes reader (IndexReaderPort) which provides access to the index.
// Takes config (search_dto.SearchConfig) which specifies the search settings.
//
// Returns ScoreResult which contains both aggregate score and per-field
// breakdown.
// Returns error when the context is cancelled.
func (s *BM25Scorer) scoreTerms(
	ctx context.Context,
	uniqueTerms map[string]bool,
	documentID uint32,
	lengthNorm float64,
	reader IndexReaderPort,
	config search_dto.SearchConfig,
) (ScoreResult, error) {
	result := ScoreResult{
		FieldScores: make(map[string]float64),
		Score:       0,
	}

	fieldWeights := buildFieldWeightMap(config)

	for term := range uniqueTerms {
		select {
		case <-ctx.Done():
			return ScoreResult{}, ctx.Err()
		default:
		}

		termResult := s.scoreTermInDocument(term, documentID, lengthNorm, reader, fieldWeights)
		result.Score += termResult.score

		if termResult.fieldName != "" {
			result.FieldScores[termResult.fieldName] += termResult.score
		}
	}

	return result, nil
}

// scoreTermsWithExplanation scores each term and populates the explanation
// struct.
//
// Takes queryTerms ([]string) which contains the search terms to score.
// Takes documentID (uint32) which identifies the document being scored.
// Takes lengthNorm (float64) which provides the document length normalisation.
// Takes reader (IndexReaderPort) which provides access to the search index.
// Takes config (search_dto.SearchConfig) which specifies the search settings.
// Takes explanation (*ScoreExplanation) which is populated with term details.
//
// Returns float64 which is the total score for all terms.
// Returns error when the context is cancelled.
func (s *BM25Scorer) scoreTermsWithExplanation(
	ctx context.Context,
	queryTerms []string,
	documentID uint32,
	lengthNorm float64,
	reader IndexReaderPort,
	config search_dto.SearchConfig,
	explanation *ScoreExplanation,
) (float64, error) {
	uniqueTerms := deduplicateTerms(queryTerms)
	fieldWeights := buildFieldWeightMap(config)

	var totalScore float64

	for term := range uniqueTerms {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		termDetail, termScore := s.scoreTermWithDetail(term, documentID, lengthNorm, reader, fieldWeights)
		explanation.TermScores = append(explanation.TermScores, termDetail)

		if termDetail.Found {
			totalScore += termScore
			explanation.FieldScores[termDetail.FieldName] += termScore
		}
	}

	return totalScore, nil
}

// scoreTermWithDetail calculates the BM25 score for a term and returns
// detailed scoring information.
//
// Takes term (string) which is the search term to score.
// Takes documentID (uint32) which identifies the document being scored.
// Takes lengthNorm (float64) which is the length normalisation factor.
// Takes reader (IndexReaderPort) which provides access to term postings.
// Takes fieldWeights (map[string]float64) which specifies field weights.
//
// Returns TermScoreDetail which contains the scoring breakdown.
// Returns float64 which is the final weighted term score.
func (s *BM25Scorer) scoreTermWithDetail(
	term string,
	documentID uint32,
	lengthNorm float64,
	reader IndexReaderPort,
	fieldWeights map[string]float64,
) (TermScoreDetail, float64) {
	postings, idf, err := reader.GetTermPostings(term)
	if err != nil {
		return createNotFoundTermDetail(term), 0
	}

	tf, fieldID, found := findTermFrequency(postings, documentID)
	if !found {
		return createNotFoundTermDetail(term), 0
	}

	numerator := tf * (s.k1 + 1)
	denominator := tf + lengthNorm
	baseScore := idf * (numerator / denominator)

	fieldWeight := s.getFieldWeight(fieldID, fieldWeights)
	fieldName := s.getFieldName(fieldID)
	termScore := baseScore * fieldWeight

	return TermScoreDetail{
		Term:        term,
		Found:       true,
		TF:          safeconv.IntToUint16(int(tf)),
		IDF:         idf,
		BaseScore:   baseScore,
		FieldID:     fieldID,
		FieldName:   fieldName,
		FieldWeight: fieldWeight,
		FinalScore:  termScore,
	}, termScore
}

// scoreTermInDocument calculates the BM25 score for a single term in a
// document.
//
// Takes term (string) which is the search term to score.
// Takes documentID (uint32) which identifies the document to score against.
// Takes lengthNorm (float64) which is the pre-computed length normalisation.
// Takes reader (IndexReaderPort) which provides access to term postings.
// Takes fieldWeights (map[string]float64) which specifies field weighting.
//
// Returns termScoreResult which contains the score and the field name.
func (s *BM25Scorer) scoreTermInDocument(
	term string,
	documentID uint32,
	lengthNorm float64,
	reader IndexReaderPort,
	fieldWeights map[string]float64,
) termScoreResult {
	postings, idf, err := reader.GetTermPostings(term)
	if err != nil || len(postings) == 0 {
		return termScoreResult{}
	}

	tf, fieldID, found := findTermFrequency(postings, documentID)
	if !found {
		return termScoreResult{}
	}

	numerator := tf * (s.k1 + 1)
	denominator := tf + lengthNorm
	baseScore := idf * (numerator / denominator)

	fieldWeight := s.getFieldWeight(fieldID, fieldWeights)
	fieldName := s.getFieldName(fieldID)

	return termScoreResult{
		score:     baseScore * fieldWeight,
		fieldName: fieldName,
	}
}

// buildFieldWeightMap creates a lookup map from field name to weight from the
// search configuration, for O(1) weight lookups during scoring.
//
// Takes config (search_dto.SearchConfig) which holds the field weight
// settings.
//
// Returns map[string]float64 mapping field names to their configured weights.
func buildFieldWeightMap(config search_dto.SearchConfig) map[string]float64 {
	weights := make(map[string]float64, len(config.Fields))
	for _, field := range config.Fields {
		if field.Weight > 0 {
			weights[field.Name] = field.Weight
		}
	}
	return weights
}

// getFieldWeight returns the weight multiplier for a field.
//
// Takes fieldID (uint8) which identifies the field to look up.
// Takes fieldWeights (map[string]float64) which maps field names to their
// configured weights.
//
// Returns float64 which is the set weight for the field, or
// DefaultFieldWeightContent if not set.
func (s *BM25Scorer) getFieldWeight(fieldID uint8, fieldWeights map[string]float64) float64 {
	fieldName := s.getFieldName(fieldID)
	if weight, ok := fieldWeights[fieldName]; ok {
		return weight
	}
	return DefaultFieldWeightContent
}

// getFieldName maps a field ID to its name.
//
// Takes fieldID (uint8) which identifies the field to look up.
//
// Returns string which is the field name, or "unknown" if the ID is not
// found.
func (*BM25Scorer) getFieldName(fieldID uint8) string {
	switch fieldID {
	case fieldIDTitle:
		return "title"
	case fieldIDContent:
		return "content"
	case fieldIDExcerpt:
		return "excerpt"
	default:
		return "unknown"
	}
}

// termScoreResult holds the score and field name for a single search term.
type termScoreResult struct {
	// fieldName is the name of the field that contributed this score.
	fieldName string

	// score is the BM25 relevance score for this term.
	score float64
}

// deduplicateTerms removes duplicate terms from a query.
//
// Takes queryTerms ([]string) which contains the search terms to deduplicate.
//
// Returns map[string]bool which contains the unique terms as keys.
func deduplicateTerms(queryTerms []string) map[string]bool {
	uniqueTerms := make(map[string]bool)
	for _, term := range queryTerms {
		uniqueTerms[term] = true
	}
	return uniqueTerms
}

// findTermFrequency searches for a document in the postings list and returns
// its term frequency and field ID.
//
// Takes postings ([]PostingInfo) which contains the term occurrence data.
// Takes documentID (uint32) which identifies the document to find.
//
// Returns float64 which is the term frequency for the document.
// Returns uint8 which is the field ID where the term was found.
// Returns bool which is true if the document was found, false otherwise.
func findTermFrequency(postings []PostingInfo, documentID uint32) (float64, uint8, bool) {
	for _, posting := range postings {
		if posting.DocumentID == documentID {
			return float64(posting.TermFrequency), posting.FieldID, true
		}
	}
	return 0, 0, false
}

// createNotFoundTermDetail creates a TermScoreDetail for a term that was not
// found in the index.
//
// Takes term (string) which is the search term that was not found.
//
// Returns TermScoreDetail which has zero values for all score fields and Found
// set to false.
func createNotFoundTermDetail(term string) TermScoreDetail {
	return TermScoreDetail{
		Term:        term,
		Found:       false,
		TF:          0,
		IDF:         0.0,
		BaseScore:   0.0,
		FieldID:     0,
		FieldName:   "",
		FieldWeight: 0.0,
		FinalScore:  0.0,
	}
}
