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
	"unicode/utf8"
)

// ClosestName finds the candidate closest in spelling to a misspelt name, ignoring case.
//
// It uses Levenshtein distance to compare the input against a list of valid words. A
// match is only returned if it falls within a threshold based on the input length.
//
// Takes input (string) which is the misspelt word to correct.
// Takes candidates ([]string) which contains valid words to match against.
//
// Returns string which is the closest match, or empty if none is close enough.
func ClosestName(input string, candidates []string) string {
	inputLower := strings.ToLower(input)
	bestMatch := ""

	const similarityThresholdDivisor = 3
	const minimumDistanceThreshold = 2

	threshold := (utf8.RuneCountInString(input) / similarityThresholdDivisor) + 1
	if threshold < minimumDistanceThreshold && utf8.RuneCountInString(input) > 2 {
		threshold = minimumDistanceThreshold
	}

	bestDistance := threshold + 1

	for _, candidate := range candidates {
		distance := runeLevenshteinDistance(inputLower, strings.ToLower(candidate))
		if distance < bestDistance {
			bestDistance = distance
			bestMatch = candidate
		}
	}

	if bestDistance <= threshold {
		return bestMatch
	}

	return ""
}

// runeLevenshteinDistance counts the fewest single-character edits needed to turn one
// string into another. It is used to find typos and suggest similar words.
//
// Takes source (string) which is the starting string.
// Takes target (string) which is the string to match against.
//
// Returns int which is the number of edits needed.
func runeLevenshteinDistance(source, target string) int {
	sourceRunes := []rune(source)
	targetRunes := []rune(target)
	sourceLen := len(sourceRunes)
	targetLen := len(targetRunes)

	if sourceLen == 0 {
		return targetLen
	}
	if targetLen == 0 {
		return sourceLen
	}

	previousRow := make([]int, targetLen+1)
	currentRow := make([]int, targetLen+1)

	for targetIndex := 0; targetIndex <= targetLen; targetIndex++ {
		previousRow[targetIndex] = targetIndex
	}

	for sourceIndex := range sourceLen {
		currentRow[0] = sourceIndex + 1
		for targetIndex := range targetLen {
			cost := 0
			if sourceRunes[sourceIndex] != targetRunes[targetIndex] {
				cost = 1
			}
			deletionCost := previousRow[targetIndex+1] + 1
			insertionCost := currentRow[targetIndex] + 1
			substitutionCost := previousRow[targetIndex] + cost
			currentRow[targetIndex+1] = min(deletionCost, insertionCost, substitutionCost)
		}
		copy(previousRow, currentRow)
	}
	return currentRow[targetLen]
}
