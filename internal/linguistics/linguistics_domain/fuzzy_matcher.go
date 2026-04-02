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

package linguistics_domain

import (
	"strings"
)

const (
	// scoreExactSubstring is the lowest score given for an exact substring match.
	scoreExactSubstring = 0.5

	// scorePrefixMatch is the score given when the text begins with the pattern.
	scorePrefixMatch = 0.9

	// scoreWordExactMatch is the score given when a word matches exactly.
	scoreWordExactMatch = 0.85

	// scoreWordPrefixMatch is the score given when a pattern word matches the
	// beginning of a text word.
	scoreWordPrefixMatch = 0.75

	// scoreWordContainsMatch is the score for a word substring match.
	scoreWordContainsMatch = 0.65
)

// FuzzyMatch performs fuzzy string matching using a hybrid algorithm.
//
// The function tries multiple strategies in order of speed and accuracy,
// returning as soon as a match is found above threshold.
//
// Strategy progression:
//  1. Exact match (instant, score 1.0)
//  2. Substring/prefix match (fast, score 0.5-1.0)
//  3. Word-level match (moderate, score 0.65-0.85)
//  4. Levenshtein distance (slow, score based on edit
//     distance)
//
// When pattern is empty, returns true with score 1.0.
//
// When text is empty, returns false with score 0.0.
//
// Takes text (string) which is the text to search in.
// Takes pattern (string) which is the search query.
// Takes threshold (float64) which is the minimum score to consider a match
// (0.0 to 1.0).
// Takes caseSensitive (bool) which controls whether to perform case-sensitive
// matching.
//
// Returns matched (bool) which is true if score meets or exceeds threshold.
// Returns score (float64) which is the relevance score from 0.0 to 1.0.
func FuzzyMatch(text, pattern string, threshold float64, caseSensitive bool) (matched bool, score float64) {
	if pattern == "" {
		return true, 1.0
	}
	if text == "" {
		return false, 0.0
	}

	searchText, searchPattern := normaliseForMatching(text, pattern, caseSensitive)

	if score, matched := tryFastMatch(searchText, searchPattern); matched {
		return score >= threshold, score
	}

	if score, matched := tryWordLevelMatch(searchText, searchPattern); matched {
		return score >= threshold, score
	}

	score = calculateLevenshteinScore(searchText, searchPattern)
	return score >= threshold, score
}

// normaliseForMatching prepares text and pattern strings for comparison.
//
// Takes text (string) which is the input string to prepare.
// Takes pattern (string) which is the pattern string to prepare.
// Takes caseSensitive (bool) which controls whether letter case is kept.
//
// Returns normalisedText (string) which is the prepared input string.
// Returns normalisedPattern (string) which is the prepared pattern string.
func normaliseForMatching(text, pattern string, caseSensitive bool) (normalisedText string, normalisedPattern string) {
	if caseSensitive {
		return text, pattern
	}
	return strings.ToLower(text), strings.ToLower(pattern)
}

// tryFastMatch attempts exact, substring, and prefix matching.
//
// Takes text (string) which is the input to match against.
// Takes pattern (string) which is the pattern to search for.
//
// Returns score (float64) which shows match quality from 0.0 to 1.0.
// Returns matched (bool) which is true if any fast method matched.
func tryFastMatch(text, pattern string) (score float64, matched bool) {
	if text == pattern {
		return 1.0, true
	}

	if strings.Contains(text, pattern) {
		score := float64(len(pattern)) / float64(len(text))
		if score < scoreExactSubstring {
			score = scoreExactSubstring
		}
		return score, true
	}

	if strings.HasPrefix(text, pattern) {
		return scorePrefixMatch, true
	}

	return 0.0, false
}

// tryWordLevelMatch checks whether words in the pattern appear in the text.
//
// Takes text (string) which is the text to search within.
// Takes pattern (string) which contains the words to find.
//
// Returns float64 which is the match score, or zero if no words match.
// Returns bool which is true if any words match.
func tryWordLevelMatch(text, pattern string) (float64, bool) {
	textWords := splitIntoWords(text)
	patternWords := splitIntoWords(pattern)

	if len(patternWords) == 0 {
		return 0.0, false
	}

	maxScore := scoreWordMatches(textWords, patternWords)

	if maxScore > 0 {
		return maxScore, true
	}

	return 0.0, false
}

// scoreWordMatches finds the best match score between text and pattern words.
//
// Takes textWords ([]string) which contains the words to search within.
// Takes patternWords ([]string) which contains the words to find.
//
// Returns float64 which is the highest match score found. An exact match
// returns straight away with the best score. Prefix matches and substring
// matches score lower.
func scoreWordMatches(textWords, patternWords []string) float64 {
	maxScore := 0.0

	for _, pWord := range patternWords {
		for _, tWord := range textWords {
			if tWord == pWord {
				return scoreWordExactMatch
			}

			if strings.HasPrefix(tWord, pWord) && maxScore < scoreWordPrefixMatch {
				maxScore = scoreWordPrefixMatch
			}

			if strings.Contains(tWord, pWord) && maxScore < scoreWordContainsMatch {
				maxScore = scoreWordContainsMatch
			}
		}
	}

	return maxScore
}

// calculateLevenshteinScore computes a similarity score using edit distance.
//
// Takes text (string) which is the text to compare.
// Takes pattern (string) which is the pattern to match against.
//
// Returns float64 which is a score from 0.0 (no match) to 1.0 (exact match).
func calculateLevenshteinScore(text, pattern string) float64 {
	distance := levenshteinDistance(text, pattern)
	maxLen := max(len(text), len(pattern))

	if maxLen == 0 {
		return 0.0
	}

	return 1.0 - (float64(distance) / float64(maxLen))
}

// levenshteinDistance calculates a weighted edit distance
// between two strings using the Wagner-Fischer algorithm with
// a substitution cost of 2 (insert and delete cost 1).
//
// Takes s1 (string) which is the first string to compare.
// Takes s2 (string) which is the second string to compare.
//
// Returns int which is the weighted edit distance.
func levenshteinDistance(s1, s2 string) int {
	return WagnerFischer(s1, s2, 1, 1, 2)
}

// splitIntoWords splits text into words by removing punctuation.
//
// Takes text (string) which is the input text to split.
//
// Returns []string which contains the words found in the text.
func splitIntoWords(text string) []string {
	var words []string
	var currentWord strings.Builder

	for _, r := range text {
		if isWordChar(r) {
			_, _ = currentWord.WriteRune(r)
		} else if currentWord.Len() > 0 {
			words = append(words, currentWord.String())
			currentWord.Reset()
		}
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}
