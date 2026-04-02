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

const (
	// jaroWinklerPrefixBonus is the weight given to matching prefixes in the
	// Jaro-Winkler similarity calculation.
	jaroWinklerPrefixBonus = 0.1

	// jaroDivisor is the value used to calculate the average in Jaro similarity.
	jaroDivisor = 3.0

	// jaroTranspositionDivisor halves transposition counts, which are counted twice.
	jaroTranspositionDivisor = 2.0
)

// WagnerFischer calculates the Levenshtein edit distance between two strings
// using the Wagner-Fischer algorithm.
//
// The Levenshtein distance is the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to transform one string
// into another.
//
// This implementation uses a space-optimised version that maintains only two
// rows of the distance matrix instead of the full O(m*n) matrix, reducing
// memory usage to O(n).
//
// Takes a (string) which is the first string to compare.
// Takes b (string) which is the second string to compare.
// Takes insertCost (int) which is the cost of inserting a character.
// Takes deleteCost (int) which is the cost of deleting a character.
// Takes substituteCost (int) which is the cost of substituting a character.
//
// Returns int which is the minimum edit distance between the two strings.
//
// Time Complexity: O(m*n) where m, n are string lengths.
// Space Complexity: O(n) - only two rows needed.
//
// Reference: Wagner & Fischer (1974) "The String-to-String Correction Problem"
func WagnerFischer(a, b string, insertCost, deleteCost, substituteCost int) int {
	if a == b {
		return 0
	}

	if len(a) == 0 {
		return len(b) * insertCost
	}
	if len(b) == 0 {
		return len(a) * deleteCost
	}

	row1 := make([]int, len(b)+1)
	row2 := make([]int, len(b)+1)

	for i := 1; i <= len(b); i++ {
		row1[i] = i * insertCost
	}

	for i := 1; i <= len(a); i++ {
		row2[0] = i * deleteCost

		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				row2[j] = row1[j-1]
			} else {
				insertion := row2[j-1] + insertCost
				deletion := row1[j] + deleteCost
				substitution := row1[j-1] + substituteCost

				row2[j] = min(insertion, deletion, substitution)
			}
		}

		row1, row2 = row2, row1
	}

	return row1[len(b)]
}

// Jaro calculates the Jaro similarity between two strings.
//
// The Jaro similarity is a measure of similarity between two strings,
// with a focus on detecting typos and transpositions. It returns a value
// between 0.0 (completely different) and 1.0 (identical).
//
// The algorithm considers:
//   - Matching characters (within a search window)
//   - Transpositions (characters that appear in different order)
//
// Jaro similarity is defined as:
// (m/|a| + m/|b| + (m-t)/m) / 3
// Where:
//   - m = number of matching characters
//   - t = number of transpositions
//   - |a|, |b| = lengths of the strings
//
// When both strings are empty, returns 1.0 (identical).
//
// When one string is empty, returns 0.0 (completely different).
//
// Takes a (string) which is the first string to compare.
// Takes b (string) which is the second string to compare.
//
// Returns float64 which is the similarity score between 0.0 and 1.0.
//
// Time Complexity: O(m*n)
// Space Complexity: O(m + n)
//
// Reference: Jaro (1989) "Advances in Record-Linkage Methodology"
//
// See JaroWinkler for a variant that gives more weight to prefix matches.
func Jaro(a, b string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	if a == b {
		return 1.0
	}

	matchesA, matchesB, matchCount := findJaroMatches(a, b)

	if matchCount == 0 {
		return 0.0
	}

	transpositions := countJaroTranspositions(a, b, matchesA, matchesB)

	return calculateJaroSimilarity(matchCount, transpositions, len(a), len(b))
}

// JaroWinkler calculates the Jaro-Winkler similarity between two strings.
//
// Jaro-Winkler is an extension of the Jaro algorithm that gives higher scores
// to strings that match at the beginning (prefix matching). This makes it
// effective for detecting typos in names, search queries, and
// short strings.
//
// The algorithm:
//  1. Calculates base Jaro similarity.
//  2. If similarity > boostThreshold, applies a prefix bonus.
//  3. Bonus = 0.1 x prefixLength x (1 - jaroSimilarity).
//
// This prefix boost makes Jaro-Winkler ideal for:
//   - Search-as-you-type
//   - Name matching with typos
//   - Query correction
//
// Takes a (string) which is the first string to compare.
// Takes b (string) which is the second string to compare.
// Takes boostThreshold (float64) which is the minimum Jaro score required to
// apply the prefix boost (typically 0.7).
// Takes prefixSize (int) which is the maximum prefix length to consider
// (typically 4).
//
// Returns float64 which is the similarity score between 0.0 and 1.0.
//
// Reference: Winkler (1990) "String Comparator Metrics and Enhanced Decision
// Rules".
func JaroWinkler(a, b string, boostThreshold float64, prefixSize int) float64 {
	jaroSimilarity := Jaro(a, b)

	if jaroSimilarity <= boostThreshold {
		return jaroSimilarity
	}

	maxPrefix := min(len(a), min(prefixSize, len(b)))
	prefixMatch := 0

	for i := range maxPrefix {
		if a[i] != b[i] {
			break
		}
		prefixMatch++
	}

	prefixBonus := jaroWinklerPrefixBonus * float64(prefixMatch) * (1.0 - jaroSimilarity)

	return jaroSimilarity + prefixBonus
}

// findJaroMatches finds matching characters within a match window.
//
// Takes a (string) which is the first string to compare.
// Takes b (string) which is the second string to compare.
//
// Returns matchesA ([]bool) which shows which positions matched in the first
// string.
// Returns matchesB ([]bool) which shows which positions matched in the second
// string.
// Returns matchCount (int) which is the number of matching characters found.
func findJaroMatches(a, b string) (matchesA []bool, matchesB []bool, matchCount int) {
	matchWindow := max(max(len(a), len(b))/2-1, 0)

	matchesA = make([]bool, len(a))
	matchesB = make([]bool, len(b))
	matchCount = 0

	for i := range len(a) {
		start := max(0, i-matchWindow)
		end := min(len(b)-1, i+matchWindow)

		for j := start; j <= end; j++ {
			if matchesB[j] || a[i] != b[j] {
				continue
			}

			matchesA[i] = true
			matchesB[j] = true
			matchCount++
			break
		}
	}

	return matchesA, matchesB, matchCount
}

// countJaroTranspositions counts how many matched characters appear in a
// different order between two strings. A transposition occurs when a character
// matches but sits at a different position.
//
// Takes a (string) which is the first string being compared.
// Takes b (string) which is the second string being compared.
// Takes matchesA ([]bool) which marks matched character positions in a.
// Takes matchesB ([]bool) which marks matched character positions in b.
//
// Returns int which is the number of transpositions found.
func countJaroTranspositions(a, b string, matchesA, matchesB []bool) int {
	transpositions := 0
	k := 0

	for i := range len(a) {
		if !matchesA[i] {
			continue
		}

		for !matchesB[k] {
			k++
		}

		if a[i] != b[k] {
			transpositions++
		}

		k++
	}

	return transpositions
}

// calculateJaroSimilarity computes the final Jaro similarity score using the
// formula: (m/|a| + m/|b| + (m-t/2)/m) / 3.
//
// Takes matches (int) which is the number of matching characters.
// Takes transpositions (int) which is the number of transposed characters.
// Takes lenA (int) which is the length of the first string.
// Takes lenB (int) which is the length of the second string.
//
// Returns float64 which is the Jaro similarity score between 0 and 1.
func calculateJaroSimilarity(matches, transpositions, lenA, lenB int) float64 {
	m := float64(matches)
	t := float64(transpositions) / jaroTranspositionDivisor
	la := float64(lenA)
	lb := float64(lenB)

	return (m/la + m/lb + (m-t)/m) / jaroDivisor
}
