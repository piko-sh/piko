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

package search_dto

const (
	// defaultFuzzyThreshold is the default threshold for
	// fuzzy matching similarity.
	defaultFuzzyThreshold = 0.3

	// defaultFuzzySimilarityThreshold is the minimum similarity score needed for
	// fuzzy matches.
	defaultFuzzySimilarityThreshold = 0.85

	// defaultFuzzyMaxResults is the largest number of
	// fuzzy search results to return.
	defaultFuzzyMaxResults = 3
)

// SearchConfig holds settings for search operations. It includes query terms,
// field weights, fuzzy matching options, and pagination controls.
type SearchConfig struct {
	// Query is the search text to match against indexed fields.
	Query string

	// Fields specifies which fields to search and their weights.
	// If empty, all string fields are searched with equal weight.
	Fields []SearchField

	// FuzzyThreshold controls how fuzzy the matching is, where 0.0 means exact
	// match only and 1.0 means very loose matching. Default is 0.3.
	FuzzyThreshold float64

	// Limit sets the maximum number of results to return; 0 means no limit.
	Limit int

	// Offset is the number of results to skip for pagination.
	Offset int

	// MinScore filters out results below this score threshold.
	// Default is 0.0 which includes all matches.
	MinScore float64

	// CaseSensitive enables case-sensitive matching when set to true.
	// Default is false, which uses case-insensitive matching.
	CaseSensitive bool

	// EnableFuzzyFallback enables Jaro-Winkler fuzzy matching as a fallback
	// when exact and phonetic matching fail in Smart mode. Default: true.
	EnableFuzzyFallback bool

	// FuzzySimilarityThreshold sets the minimum similarity for
	// fuzzy matches (0.0-1.0), where higher values mean stricter
	// matching with typical values of 0.80-0.90. Default: 0.85.
	FuzzySimilarityThreshold float64

	// FuzzyMaxResults limits the number of fuzzy matches to expand
	// the query with, preventing over-expansion that could return
	// too many irrelevant results. Default: 3.
	FuzzyMaxResults int
}

// SearchField specifies a field to search with an optional weight.
type SearchField struct {
	// Name is the field name to search.
	Name string

	// Weight is the importance multiplier for this field.
	// Default: 1.0
	// Higher values (e.g., 2.0) make matches in this field more important.
	Weight float64
}

// DefaultSearchConfig returns a SearchConfig with sensible default values.
//
// Takes query (string) which specifies the search term to use.
//
// Returns SearchConfig which is ready for use with default settings.
func DefaultSearchConfig(query string) SearchConfig {
	return SearchConfig{
		Query:                    query,
		Fields:                   nil,
		FuzzyThreshold:           defaultFuzzyThreshold,
		Limit:                    0,
		Offset:                   0,
		MinScore:                 0.0,
		CaseSensitive:            false,
		EnableFuzzyFallback:      true,
		FuzzySimilarityThreshold: defaultFuzzySimilarityThreshold,
		FuzzyMaxResults:          defaultFuzzyMaxResults,
	}
}
