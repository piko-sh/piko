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

package annotator_domain

// Provides suggestion helpers for generating helpful error messages with spelling corrections and similar term recommendations.
// Uses fuzzy matching and canonical term mapping to suggest corrections when users reference undefined variables or properties.

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// commonSynonymGroups is a curated list of common synonyms for actions in our
	// domain. The first word in each group is considered the canonical or
	// preferred term.
	commonSynonymGroups = [][]string{
		{"Create", "Add", "New", "Insert", "Make", "Build"},
		{"Delete", "Remove", "Erase", "Destroy", "Drop", "Purge"},
		{"Update", "Modify", "Change", "Edit", "Set"},
		{"Save", "Persist", "Store", "Write"},
		{"Find", "Get", "Retrieve", "Query", "Select", "Fetch"},
		{"List", "GetAll", "FindAll", "FetchAll", "SelectAll"},
		{"First", "Top", "Initial", "Head"},
		{"Last", "Final", "End"},
		{"Enable", "Activate", "Start", "TurnOn", "Resume"},
		{"Disable", "Deactivate", "Stop", "TurnOff", "Pause"},
		{"Toggle"},
		{"Publish", "Release", "Deploy"},
		{"Archive", "Hide", "Unpublish"},
		{"Reset", "Clear", "Wipe"},
		{"Validate", "Check", "Verify", "Assert"},
		{"Exists", "Has", "Contains", "Includes"},
		{"Count", "Total", "Size", "Length"},
		{"Process", "Execute", "Run", "Perform", "Handle"},
		{"Send", "Dispatch", "Emit", "Fire", "Trigger"},
		{"Schedule", "Queue", "Enqueue"},
		{"Subscribe", "Watch", "Observe", "Listen"},
		{"Unsubscribe", "Unwatch", "StopListening"},
		{"Login", "SignIn", "Authenticate"},
		{"Logout", "SignOut"},
		{"Register", "SignUp"},
		{"Approve", "Accept", "Authorise"},
		{"Reject", "Deny", "Decline"},
		{"Format", "Render", "Display", "Print"},
		{"Parse", "Decode"},
		{"Encode", "Stringify"},
		{"Merge", "Combine", "Join"},
		{"Split", "Separate"},
	}

	// synonymMap maps any synonym to the canonical term for its group.
	// It is a pre-processed map for fast lookups, initialised once by init.
	synonymMap map[string]string
)

// getCanonicalTerm returns the preferred term for a given word if it is a
// known synonym. If the word is not a known synonym, it returns the original
// word unchanged.
//
// Takes word (string) which is the term to look up in the synonym map.
//
// Returns string which is the preferred term if found, or the original word
// if not.
func getCanonicalTerm(word string) string {
	canonical, ok := synonymMap[strings.ToLower(word)]
	if ok {
		return canonical
	}
	return word
}

// findClosestMatch finds the best matching candidate for an input string.
//
// It checks for matches in this order:
//  1. A synonym match (e.g. input "RemoveUser" matches "DeleteUser").
//  2. The closest match based on edit distance for typos.
//
// Takes input (string) which is the text to find a match for.
// Takes candidates ([]string) which contains the possible matches.
//
// Returns string which is the best match, or empty if none is found.
func findClosestMatch(input string, candidates []string) string {
	if input == "" || len(candidates) == 0 {
		return ""
	}

	if synonymMatch := findSynonymMatch(input, candidates); synonymMatch != "" {
		return synonymMatch
	}

	return findClosestTypo(input, candidates)
}

// findClosestTypo finds the closest spelling match for a misspelt word.
//
// It uses Levenshtein distance to compare the input against a list of valid
// words. A match is only returned if it falls within a threshold based on
// the input length.
//
// Takes input (string) which is the misspelt word to correct.
// Takes candidates ([]string) which contains valid words to match against.
//
// Returns string which is the closest match, or empty if none is close enough.
func findClosestTypo(input string, candidates []string) string {
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
		distance := levenshteinDistance(inputLower, strings.ToLower(candidate))
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

// levenshteinDistance counts the fewest single-character edits needed to turn
// one string into another. It is used to find typos and suggest similar words.
//
// Takes source (string) which is the starting string.
// Takes target (string) which is the string to match against.
//
// Returns int which is the number of edits needed.
func levenshteinDistance(source, target string) int {
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

// findSynonymMatch checks if the input term has a known synonym that matches
// any of the candidates.
//
// Takes input (string) which is the term to check for synonyms.
// Takes candidates ([]string) which contains possible standard matches.
//
// Returns string which is the matching candidate, or an empty string if no
// match is found.
func findSynonymMatch(input string, candidates []string) string {
	inputAction, inputSubject := splitActionAndSubject(input)
	if inputAction == "" {
		return ""
	}

	canonicalAction := getCanonicalTerm(inputAction)
	if strings.EqualFold(canonicalAction, inputAction) {
		return ""
	}

	for _, candidate := range candidates {
		candidateAction, candidateSubject := splitActionAndSubject(candidate)
		if strings.EqualFold(getCanonicalTerm(candidateAction), canonicalAction) && strings.EqualFold(candidateSubject, inputSubject) {
			return candidate
		}
	}

	return ""
}

// splitActionAndSubject splits a CamelCase name into its first word and the
// rest.
//
// It handles names like "RemoveUser" or "URLShortener", finding where the
// action word ends and the subject starts.
//
// Takes identifier (string) which is the CamelCase name to split.
//
// Returns action (string) which is the first word (e.g. "Remove" or "URL").
// Returns subject (string) which is the rest (e.g. "User" or "Shortener").
func splitActionAndSubject(identifier string) (action, subject string) {
	if identifier == "" {
		return "", ""
	}

	runes := []rune(identifier)
	if len(runes) == 0 {
		return "", ""
	}

	var splitIndex int
	for i := 1; i < len(runes); i++ {
		if unicode.IsLower(runes[i-1]) && unicode.IsUpper(runes[i]) {
			splitIndex = i
			break
		}
		if i+1 < len(runes) && unicode.IsUpper(runes[i-1]) && unicode.IsUpper(runes[i]) && unicode.IsLower(runes[i+1]) {
			splitIndex = i
			break
		}
	}

	if splitIndex == 0 {
		return identifier, ""
	}

	action = string(runes[:splitIndex])
	subject = string(runes[splitIndex:])
	return action, subject
}

func init() {
	synonymMap = make(map[string]string)
	for _, group := range commonSynonymGroups {
		if len(group) > 0 {
			canonical := group[0]
			for _, synonym := range group {
				synonymMap[strings.ToLower(synonym)] = canonical
			}
		}
	}
}
