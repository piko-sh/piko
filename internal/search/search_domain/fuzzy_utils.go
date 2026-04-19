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
	"slices"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// fuzzyMatchResult represents a fuzzy match result with its similarity score.
type fuzzyMatchResult struct {
	// Term is the matched term from the vocabulary.
	Term string

	// Similarity is the match score from 0.0 (no match) to 1.0 (exact match).
	Similarity float64

	// Distance is the edit distance from the query term.
	Distance int
}

// findSimilarTermsJaroWinkler finds vocabulary terms similar to the query term
// using the Jaro-Winkler algorithm.
//
// Jaro-Winkler is designed for detecting typos in short strings (like names,
// search terms). It handles:
//   - Missing characters: "configurtion" -> "configuration"
//   - Transposed characters: "ocnfiguration" -> "configuration"
//   - Character substitutions: "konfig" -> "config"
//
// The algorithm gives bonus scores to strings that match at the beginning
// (prefix boost), well suited for search-as-you-type and autocomplete
// scenarios.
//
// Takes queryTerm (string) which is the search term (possibly with typos).
// Takes vocabulary ([]string) which is the list of valid terms from the index.
// Takes minSimilarity (float64) which is the minimum similarity threshold
// (0.0-1.0, typically 0.85).
// Takes maxResults (int) which is the maximum number of results to return
// (0 = no limit).
//
// Returns []fuzzyMatchResult which contains matches sorted by similarity
// (descending).
//
// Performance:
//   - O(V) where V = vocabulary size
//   - ~500ns per term comparison
//   - For 10,000 term vocabulary: ~5ms
func findSimilarTermsJaroWinkler(
	queryTerm string,
	vocabulary []string,
	minSimilarity float64,
	maxResults int,
) []fuzzyMatchResult {
	if queryTerm == "" || len(vocabulary) == 0 {
		return nil
	}

	results := make([]fuzzyMatchResult, 0, max(maxResults, 0))

	const boostThreshold = 0.7
	const prefixSize = 4

	for _, vocabTerm := range vocabulary {
		if vocabTerm == queryTerm {
			continue
		}

		similarity := linguistics_domain.JaroWinkler(queryTerm, vocabTerm, boostThreshold, prefixSize)

		if similarity >= minSimilarity {
			results = append(results, fuzzyMatchResult{
				Term:       vocabTerm,
				Similarity: similarity,
				Distance:   0,
			})

			if similarity > fuzzyMatchHighSimilarity && maxResults > 0 && len(results) >= maxResults {
				break
			}
		}
	}

	slices.SortFunc(results, func(a, b fuzzyMatchResult) int {
		return cmp.Compare(b.Similarity, a.Similarity)
	})

	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

// findTermsWithinEditDistance finds vocabulary terms within a specified edit
// distance using the Wagner-Fischer algorithm.
//
// Wagner-Fischer calculates Levenshtein edit distance using an optimised
// two-row approach instead of a full matrix, reducing memory usage to O(n).
//
// Use cases include strict typo correction (maxDistance=1 or 2), "did you
// mean?" suggestions, and query expansion for fuzzy search.
//
// Takes queryTerm (string) which is the search term, possibly with typos.
// Takes vocabulary ([]string) which is the list of valid terms from the index.
// Takes maxDistance (int) which is the maximum edit distance (typically 1 or 2).
//
// Returns []fuzzyMatchResult which contains matches sorted by distance in
// ascending order (closest first).
func findTermsWithinEditDistance(
	queryTerm string,
	vocabulary []string,
	maxDistance int,
) []fuzzyMatchResult {
	if queryTerm == "" || len(vocabulary) == 0 || maxDistance < 0 {
		return nil
	}

	results := make([]fuzzyMatchResult, 0)

	const insertionCost = 1
	const deletionCost = 1
	const substitutionCost = 2

	for _, vocabTerm := range vocabulary {
		if vocabTerm == queryTerm {
			continue
		}

		distance := linguistics_domain.WagnerFischer(queryTerm, vocabTerm, insertionCost, deletionCost, substitutionCost)

		if distance <= maxDistance {
			maxLen := max(len(queryTerm), len(vocabTerm))
			similarity := 1.0 - (float64(distance) / float64(maxLen))

			results = append(results, fuzzyMatchResult{
				Term:       vocabTerm,
				Similarity: similarity,
				Distance:   distance,
			})
		}
	}

	slices.SortFunc(results, func(a, b fuzzyMatchResult) int {
		return cmp.Compare(a.Distance, b.Distance)
	})

	return results
}

// findBestFuzzyMatch finds the best fuzzy match for a search term.
//
// Uses two methods to find matches:
//   - Edit distance for close matches (good for small spelling errors)
//   - Jaro-Winkler for similarity scoring (good for typos and swapped letters)
//
// Picks the result from whichever method gives the better match. When
// queryTerm is empty or vocabulary is empty, an empty string with a score of
// zero is produced.
//
// Takes queryTerm (string) which is the term to search for.
// Takes vocabulary ([]string) which is the list of valid terms to match against.
// Takes maxDistance (int) which is the largest edit distance allowed.
//
// Returns string which is the best matching term, or empty if none found.
// Returns float64 which is the similarity score of the match.
func findBestFuzzyMatch(
	queryTerm string,
	vocabulary []string,
	maxDistance int,
) (string, float64) {
	if queryTerm == "" || len(vocabulary) == 0 {
		return "", 0.0
	}

	distanceMatches := findTermsWithinEditDistance(queryTerm, vocabulary, maxDistance)

	if len(distanceMatches) > 0 && distanceMatches[0].Distance <= 1 {
		return distanceMatches[0].Term, distanceMatches[0].Similarity
	}

	similarityMatches := findSimilarTermsJaroWinkler(queryTerm, vocabulary, fuzzyMatchLowSimilarity, 1)

	if len(similarityMatches) > 0 {
		return similarityMatches[0].Term, similarityMatches[0].Similarity
	}

	return "", 0.0
}

// rankTermsBySimilarity ranks vocabulary terms by similarity to a query term.
//
// Uses Jaro-Winkler for ranking, which is better for short strings and handles
// prefix matching, transpositions, and common typos.
//
// Useful for:
//   - Autocomplete suggestions
//   - "Did you mean?" features
//   - Query expansion
//
// Takes queryTerm (string) which is the partial or complete search term.
// Takes vocabulary ([]string) which is the list of candidate terms.
// Takes limit (int) which is the maximum number of results to return.
//
// Returns []fuzzyMatchResult which contains the top N terms sorted by
// similarity, best first.
func rankTermsBySimilarity(
	queryTerm string,
	vocabulary []string,
	limit int,
) []fuzzyMatchResult {
	return findSimilarTermsJaroWinkler(queryTerm, vocabulary, 0.0, limit)
}
