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
	"cmp"
	"context"
	"fmt"
	"slices"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/search/search_dto"
	search_fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

// QueryProcessor implements QueryProcessorPort to run search queries.
// It supports Fast and Smart modes with different query expansion methods.
type QueryProcessor struct {
	// analyser processes query text into tokens for search matching.
	analyser linguistics_domain.AnalyserPort
}

// NewQueryProcessorForIndex creates a query processor configured for a
// specific index. The analyser is created once and reused for all queries.
//
// Takes reader (IndexReaderPort) which provides the index mode and language
// settings.
//
// Returns *QueryProcessor which is ready for concurrent use.
func NewQueryProcessorForIndex(reader IndexReaderPort) *QueryProcessor {
	mode := reader.GetMode()
	language := reader.GetLanguage()

	config := createAnalyserConfigFromIndex(mode, language)

	stemmer := linguistics_domain.CreateStemmer(language)

	return &QueryProcessor{
		analyser: linguistics_domain.NewAnalyser(config, linguistics_domain.WithStemmer(stemmer)),
	}
}

// Search runs a query against an index and returns ranked results.
//
// Takes query (string) which is the search text to match.
// Takes reader (IndexReaderPort) which provides access to the index data.
// Takes scorer (ScorerPort) which calculates relevance scores for matches.
// Takes config (search_dto.SearchConfig) which sets pagination and filters.
//
// Returns []QueryResult which contains the ranked matching documents.
// Returns error when query analysis or document scoring fails.
//
// The search strategy depends on the index mode:
//
// Fast Mode:
//  1. Tokenise query with basic normalisation.
//  2. Look up exact terms in the index.
//  3. Score results with BM25.
//  4. Return the top N results.
//
// Smart Mode:
//  1. Tokenise and stem the query.
//  2. For each term, try these strategies in order:
//     a) Exact match (highest boost).
//     b) Stemmed match (medium boost).
//     c) Phonetic match (lower boost).
//     d) Prefix match (for partial words).
//  3. Score with BM25 plus boost factors.
//  4. Return the top N results.
func (qp *QueryProcessor) Search(
	ctx context.Context,
	query string,
	reader IndexReaderPort,
	scorer ScorerPort,
	config search_dto.SearchConfig,
) ([]QueryResult, error) {
	mode := reader.GetMode()

	if qp.analyser == nil {
		language := reader.GetLanguage()
		analyserConfig := createAnalyserConfigFromIndex(mode, language)

		stemmer := linguistics_domain.CreateStemmer(language)
		qp.analyser = linguistics_domain.NewAnalyser(analyserConfig, linguistics_domain.WithStemmer(stemmer))
	}

	queryTerms, err := qp.analyseQuery(query, mode)
	if err != nil {
		return nil, fmt.Errorf("analysing query: %w", err)
	}

	if len(queryTerms) == 0 {
		return []QueryResult{}, nil
	}

	candidateDocuments, err := qp.findCandidateDocuments(ctx, queryTerms, reader, mode)
	if err != nil {
		return nil, fmt.Errorf("finding candidates: %w", err)
	}

	if len(candidateDocuments) == 0 {
		return []QueryResult{}, nil
	}

	scoredResults, err := qp.scoreDocuments(ctx, candidateDocuments, queryTerms, reader, scorer, config)
	if err != nil {
		return nil, fmt.Errorf("scoring documents: %w", err)
	}

	if config.MinScore > 0 {
		scoredResults = qp.filterByMinScore(scoredResults, config.MinScore)
	}

	slices.SortFunc(scoredResults, func(a, b QueryResult) int {
		return cmp.Compare(b.Score, a.Score)
	})

	scoredResults = qp.applyPagination(scoredResults, config.Limit, config.Offset)

	return scoredResults, nil
}

// SearchWithExplanation executes a search and returns detailed scoring
// explanations for debugging and understanding search results.
//
// Takes query (string) which specifies the search terms.
// Takes reader (IndexReaderPort) which provides access to the search index.
// Takes scorer (*BM25Scorer) which calculates document relevance scores.
// Takes config (search_dto.SearchConfig) which controls search behaviour.
//
// Returns []QueryResult which contains the matched documents.
// Returns []*ScoreExplanation which provides scoring details for each result.
// Returns error when the search or query analysis fails.
func (qp *QueryProcessor) SearchWithExplanation(
	ctx context.Context,
	query string,
	reader IndexReaderPort,
	scorer *BM25Scorer,
	config search_dto.SearchConfig,
) ([]QueryResult, []*ScoreExplanation, error) {
	results, err := qp.Search(ctx, query, reader, scorer, config)
	if err != nil {
		return nil, nil, fmt.Errorf("searching: %w", err)
	}

	if qp.analyser == nil {
		mode := reader.GetMode()
		language := reader.GetLanguage()
		analyserConfig := createAnalyserConfigFromIndex(mode, language)
		qp.analyser = linguistics_domain.NewAnalyser(analyserConfig)
	}

	mode := reader.GetMode()
	queryTerms, err := qp.analyseQuery(query, mode)
	if err != nil {
		return nil, nil, fmt.Errorf("analysing query: %w", err)
	}

	explanations := make([]*ScoreExplanation, len(results))
	for i, result := range results {
		_, explanation, err := scorer.ScoreWithExplanation(
			ctx,
			queryTerms,
			result.DocumentID,
			reader,
			config,
		)
		if err == nil {
			explanations[i] = explanation
		}
	}

	return results, explanations, nil
}

// analyseQuery processes the query string and returns search terms.
//
// Takes query (string) which is the raw search query to process.
// Takes mode (SearchMode) which controls how terms are processed.
//
// Returns []string which contains the processed search terms.
// Returns error when query processing fails.
func (qp *QueryProcessor) analyseQuery(query string, mode search_fb.SearchMode) ([]string, error) {
	tokens := qp.analyser.Analyse(query)

	if len(tokens) == 0 {
		return nil, nil
	}

	if mode == search_fb.SearchModeFast {
		terms := make([]string, len(tokens))
		for i, token := range tokens {
			terms[i] = token.Normalised
		}
		return terms, nil
	}

	terms := make([]string, len(tokens))
	for i, token := range tokens {
		terms[i] = token.Stemmed
	}
	return terms, nil
}

// findCandidateDocuments retrieves all documents that contain any query term.
// This uses the inverted index for O(log n) term lookup.
//
// Takes queryTerms ([]string) which specifies the terms to search for.
// Takes reader (IndexReaderPort) which provides access to the inverted index.
// Takes mode (search_fb.SearchMode) which controls the search behaviour.
//
// Returns map[uint32]bool which contains document IDs matching any query term.
// Returns error when the context is cancelled.
func (qp *QueryProcessor) findCandidateDocuments(
	ctx context.Context,
	queryTerms []string,
	reader IndexReaderPort,
	mode search_fb.SearchMode,
) (map[uint32]bool, error) {
	candidates := make(map[uint32]bool)

	for _, term := range queryTerms {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		qp.findCandidatesForTerm(ctx, term, reader, mode, candidates)
	}

	return candidates, nil
}

// findCandidatesForTerm finds candidate documents for a single search term.
//
// Takes ctx (context.Context) which carries the logger for warning on errors.
// Takes term (string) which is the search term to find candidates for.
// Takes reader (IndexReaderPort) which provides access to the search index.
// Takes mode (search_fb.SearchMode) which sets how to search (smart or fast).
// Takes candidates (map[uint32]bool) which collects matching document IDs.
func (qp *QueryProcessor) findCandidatesForTerm(
	ctx context.Context,
	term string,
	reader IndexReaderPort,
	mode search_fb.SearchMode,
	candidates map[uint32]bool,
) {
	if mode == search_fb.SearchModeSmart {
		if smartError := qp.findCandidatesSmartMode(term, reader, candidates); smartError != nil {
			_, warningLogger := logger_domain.From(ctx, nil)
			warningLogger.Warn("failed to find candidates in smart mode",
				logger_domain.String("term", term),
				logger_domain.Error(smartError))
		}
		return
	}

	qp.findCandidatesFastMode(term, reader, candidates)
}

// findCandidatesFastMode searches for matching documents using fast mode
// strategies: exact match and prefix expansion.
//
// Takes term (string) which is the search term to find matches for.
// Takes reader (IndexReaderPort) which provides access to the search index.
// Takes candidates (map[uint32]bool) which collects matching document IDs.
func (qp *QueryProcessor) findCandidatesFastMode(
	term string,
	reader IndexReaderPort,
	candidates map[uint32]bool,
) {
	postings, _, err := reader.GetTermPostings(term)
	if err == nil {
		addPostingsToCandidates(postings, candidates)
	}

	qp.tryPrefixMatch(term, reader, candidates)
}

// findCandidatesSmartMode uses multiple strategies to find matching documents.
//
// Smart Mode Search Strategies (in order):
//  1. Exact stemmed match - O(log n) index lookup
//  2. Prefix match - O(log n) + expansion, handles partial words
//  3. Phonetic match - O(log n) via phonetic map
//  4. Jaro-Winkler fuzzy match - O(V) vocabulary scan, handles typos
//
// Each strategy is only tried if the previous one fails. This degrades
// gracefully from fast exact matches to slower fuzzy matches.
//
// Takes term (string) which is the search term to find candidates for.
// Takes reader (IndexReaderPort) which provides access to the search index.
// Takes candidates (map[uint32]bool) which accumulates matching document IDs.
//
// Returns error when the search operation fails.
func (qp *QueryProcessor) findCandidatesSmartMode(
	term string,
	reader IndexReaderPort,
	candidates map[uint32]bool,
) error {
	if qp.tryExactMatch(term, reader, candidates) {
		return nil
	}

	if qp.tryPrefixMatch(term, reader, candidates) {
		return nil
	}

	if qp.tryPhoneticMatch(term, reader, candidates) {
		return nil
	}

	qp.tryFuzzyMatch(term, reader, candidates)

	return nil
}

// tryExactMatch looks for documents that contain the given term exactly.
//
// Takes term (string) which is the exact term to search for.
// Takes reader (IndexReaderPort) which provides access to the term index.
// Takes candidates (map[uint32]bool) which collects matching document IDs.
//
// Returns bool which is true if matches were found, false otherwise.
func (*QueryProcessor) tryExactMatch(
	term string,
	reader IndexReaderPort,
	candidates map[uint32]bool,
) bool {
	postings, _, err := reader.GetTermPostings(term)
	if err != nil || len(postings) == 0 {
		return false
	}

	addPostingsToCandidates(postings, candidates)
	return true
}

// tryPhoneticMatch finds documents by matching the phonetic sound of a term.
//
// Takes term (string) which is the search term to encode phonetically.
// Takes reader (IndexReaderPort) which provides access to the search index.
// Takes candidates (map[uint32]bool) which collects matching document IDs.
//
// Returns bool which is true if matches were found, false otherwise.
func (*QueryProcessor) tryPhoneticMatch(
	term string,
	reader IndexReaderPort,
	candidates map[uint32]bool,
) bool {
	phoneticEncoder := linguistics_domain.NewPhoneticEncoder(linguistics_domain.DefaultPhoneticCodeLength)
	phoneticCode := phoneticEncoder.Encode(term)

	if phoneticCode == "" {
		return false
	}

	matchingTerms, err := reader.FindPhoneticTerms(phoneticCode)
	if err != nil || len(matchingTerms) == 0 {
		return false
	}

	for _, matchingTerm := range matchingTerms {
		postings, _, err := reader.GetTermPostings(matchingTerm)
		if err == nil {
			addPostingsToCandidates(postings, candidates)
		}
	}

	return true
}

// tryFuzzyMatch finds documents using fuzzy string matching with the
// Jaro-Winkler algorithm.
//
// Takes term (string) which is the search term to match.
// Takes reader (IndexReaderPort) which provides access to the search index.
// Takes candidates (map[uint32]bool) which collects matching document IDs.
//
// Returns bool which is true if matches were found, false otherwise.
func (qp *QueryProcessor) tryFuzzyMatch(
	term string,
	reader IndexReaderPort,
	candidates map[uint32]bool,
) bool {
	similarTerms, err := qp.findSimilarTermsJaroWinkler(term, reader)
	if err != nil || len(similarTerms) == 0 {
		return false
	}

	for _, similarTerm := range similarTerms {
		postings, _, err := reader.GetTermPostings(similarTerm)
		if err == nil {
			addPostingsToCandidates(postings, candidates)
		}
	}

	return true
}

// tryPrefixMatch attempts to find documents using prefix matching. This
// expands terms like "doc" to "docs", "documentation", and similar variations.
//
// Takes term (string) which specifies the search term to expand.
// Takes reader (IndexReaderPort) which provides access to the term index.
// Takes candidates (map[uint32]bool) which collects matching document IDs.
//
// Returns bool which indicates whether any matches were found.
func (*QueryProcessor) tryPrefixMatch(
	term string,
	reader IndexReaderPort,
	candidates map[uint32]bool,
) bool {
	if len(term) < DefaultMinTokenLength+1 {
		return false
	}

	expandedTerms, err := reader.FindTermsWithPrefix(term)
	if err != nil || len(expandedTerms) == 0 {
		return false
	}

	foundAny := false
	for _, expTerm := range expandedTerms {
		if expTerm == term {
			continue
		}

		postings, _, err := reader.GetTermPostings(expTerm)
		if err == nil && len(postings) > 0 {
			addPostingsToCandidates(postings, candidates)
			foundAny = true
		}
	}

	return foundAny
}

// findSimilarTermsJaroWinkler finds terms in the index vocabulary that are
// similar to the query term using the Jaro-Winkler algorithm.
//
// This is used as a fallback when exact and phonetic matching fail.
// It works well for typos like:
//   - "configurtion" (missing 'a') -> "configuration"
//   - "ocnfiguration" (transposed 'on') -> "configuration"
//
// Takes term (string) which is the query term to find matches for.
// Takes reader (IndexReaderPort) which provides access to the index vocabulary.
//
// Returns []string which contains similar terms from the vocabulary.
// Returns error when the vocabulary cannot be accessed.
func (*QueryProcessor) findSimilarTermsJaroWinkler(
	term string,
	reader IndexReaderPort,
) ([]string, error) {
	vocabulary, err := reader.GetAllTerms()
	if err != nil {
		return nil, fmt.Errorf("retrieving vocabulary for fuzzy matching: %w", err)
	}

	const minSimilarity = 0.85
	const maxResults = 3

	fuzzyMatches := findSimilarTermsJaroWinkler(term, vocabulary, minSimilarity, maxResults)

	similarTerms := make([]string, len(fuzzyMatches))
	for i, match := range fuzzyMatches {
		similarTerms[i] = match.Term
	}

	return similarTerms, nil
}

// scoreDocuments scores all candidate documents using BM25.
//
// Takes candidates (map[uint32]bool) which contains the document IDs to score.
// Takes queryTerms ([]string) which holds the search terms to match against.
// Takes reader (IndexReaderPort) which provides access to the index data.
// Takes scorer (ScorerPort) which calculates document relevance scores.
// Takes config (search_dto.SearchConfig) which specifies scoring parameters.
//
// Returns []QueryResult which contains scored documents sorted by relevance.
// Returns error when the context is cancelled.
func (*QueryProcessor) scoreDocuments(
	ctx context.Context,
	candidates map[uint32]bool,
	queryTerms []string,
	reader IndexReaderPort,
	scorer ScorerPort,
	config search_dto.SearchConfig,
) ([]QueryResult, error) {
	results := make([]QueryResult, 0, len(candidates))

	for documentID := range candidates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		scoreResult, err := scorer.Score(ctx, queryTerms, documentID, reader, config)
		if err != nil {
			continue
		}

		results = append(results, QueryResult{
			DocumentID:  documentID,
			Score:       scoreResult.Score,
			FieldScores: scoreResult.FieldScores,
		})
	}

	return results, nil
}

// filterByMinScore removes results below the minimum score threshold.
//
// Takes results ([]QueryResult) which contains the results to filter.
// Takes minScore (float64) which specifies the minimum score threshold.
//
// Returns []QueryResult which contains only results with scores at or above
// the threshold.
func (*QueryProcessor) filterByMinScore(results []QueryResult, minScore float64) []QueryResult {
	filtered := make([]QueryResult, 0, len(results))
	for _, result := range results {
		if result.Score >= minScore {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// applyPagination applies limit and offset to the results.
//
// Takes results ([]QueryResult) which contains the query results to paginate.
// Takes limit (int) which specifies the maximum number of results to return.
// Takes offset (int) which specifies the number of results to skip.
//
// Returns []QueryResult which contains the paginated subset of results.
func (*QueryProcessor) applyPagination(results []QueryResult, limit, offset int) []QueryResult {
	if offset > 0 {
		if offset >= len(results) {
			return []QueryResult{}
		}
		results = results[offset:]
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// addPostingsToCandidates adds all document IDs from the postings to the
// candidates map.
//
// Takes postings ([]PostingInfo) which contains the posting entries to add.
// Takes candidates (map[uint32]bool) which is the map to fill with document
// IDs.
func addPostingsToCandidates(postings []PostingInfo, candidates map[uint32]bool) {
	for _, posting := range postings {
		candidates[posting.DocumentID] = true
	}
}
