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

package db_engine_clickhouse

var (
	// multiSearchCaseAndUTF8Suffixes lists the case-sensitivity / UTF-8 suffix combinations
	// attached to multi-pattern search variants. The catalogue iterates over this list to
	// register every combination without spelling each variant out individually.
	multiSearchCaseAndUTF8Suffixes = []string{
		"",
		"CaseInsensitive",
		"UTF8",
		"CaseInsensitiveUTF8",
	}
)

// registerExtendedStringSearchFunctions covers the long tail of multi-pattern, fuzzy,
// n-gram and token-search helpers that ClickHouse exposes alongside the base hasToken /
// multiSearchAny family.
//
// Splitting by topical group keeps each helper function within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered search functions.
func registerExtendedStringSearchFunctions(b *FunctionCatalogueBuilder) {
	registerMultiSearchPositionAndIndexVariants(b)
	registerMultiMatchAndFuzzyVariants(b)
	registerCountAndExtractHelpers(b)
	registerTokenAndSubsequenceHelpers(b)
	registerNgramDistanceAndSearchVariants(b)
	registerHighlightAndCaseInsensitiveTokenOrNull(b)
}

// registerMultiSearchPositionAndIndexVariants covers multiSearchAllPositions,
// multiSearchFirstIndex, multiSearchFirstPosition and the UTF-8 multi-search hit-flag
// variants across every case-sensitivity / UTF-8 combination.
//
// multiSearchAllPositions returns an array of positions (one per pattern); the index /
// position variants return a single number. The base case-sensitive / case-insensitive
// multiSearchAny live in registerStringSearchFunctions; the UTF-8 hit-flag variants land
// here so the catalogue records every spelling.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered variant functions.
func registerMultiSearchPositionAndIndexVariants(b *FunctionCatalogueBuilder) {
	for _, suffix := range multiSearchCaseAndUTF8Suffixes {
		b.Register("multiSearchAllPositions"+suffix, arrayOf(b.uint64Type), b.textType, arrayOf(b.textType))
		b.Register("multiSearchFirstIndex"+suffix, b.uint64Type, b.textType, arrayOf(b.textType))
		b.Register("multiSearchFirstPosition"+suffix, b.uint64Type, b.textType, arrayOf(b.textType))
	}
	b.Register("multiSearchAnyUTF8", b.boolType, b.textType, arrayOf(b.textType))
	b.Register("multiSearchAnyCaseInsensitiveUTF8", b.boolType, b.textType, arrayOf(b.textType))
}

// registerMultiMatchAndFuzzyVariants covers the regex-based multiMatchAnyIndex helper
// across every case / UTF-8 suffix variant and the fuzzy match family.
//
// The fuzzy variants accept a Levenshtein distance plus the pattern array; their case /
// UTF-8 variants are registered through the same suffix iteration so every spelling
// resolves identically.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered variant functions.
func registerMultiMatchAndFuzzyVariants(b *FunctionCatalogueBuilder) {
	for _, suffix := range multiSearchCaseAndUTF8Suffixes {
		b.Register("multiMatchAnyIndex"+suffix, b.uint64Type, b.textType, arrayOf(b.textType))
		b.Register("multiFuzzyMatchAny"+suffix, b.boolType, b.textType, b.uint64Type, arrayOf(b.textType))
		b.Register("multiFuzzyMatchAnyIndex"+suffix, b.uint64Type, b.textType, b.uint64Type, arrayOf(b.textType))
		b.Register("multiFuzzyMatchAllIndices"+suffix, arrayOf(b.uint64Type), b.textType, b.uint64Type, arrayOf(b.textType))
	}
}

// registerCountAndExtractHelpers covers the case-insensitive count variants plus
// extractGroups and extractAllGroupsHorizontal, which return the captured groups from a
// regex match.
//
// extractGroups returns Array(String) over the first match; extractAllGroupsHorizontal
// returns Array(Array(String)) with one outer element per capture group and one inner
// element per match.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerCountAndExtractHelpers(b *FunctionCatalogueBuilder) {
	b.Register("countMatchesCaseInsensitive", b.uint64Type, b.textType, b.textType)
	b.Register("countSubstringsCaseInsensitive", b.uint64Type, b.textType, b.textType)
	b.Register("countSubstringsCaseInsensitiveUTF8", b.uint64Type, b.textType, b.textType)
	b.Register("extractGroups", arrayOf(b.textType), b.textType, b.textType)
	b.Register("extractAllGroupsHorizontal", arrayOf(arrayOf(b.textType)), b.textType, b.textType)
}

// registerTokenAndSubsequenceHelpers covers hasAllTokens, hasAnyTokens and hasPhrase plus
// the four hasSubsequence variants.
//
// The token helpers accept the haystack and an array of tokens; the subsequence helpers
// accept two strings and test for ordered subsequence membership.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerTokenAndSubsequenceHelpers(b *FunctionCatalogueBuilder) {
	b.Register("hasAllTokens", b.boolType, b.textType, arrayOf(b.textType))
	b.Register("hasAnyTokens", b.boolType, b.textType, arrayOf(b.textType))
	b.Register("hasPhrase", b.boolType, b.textType, b.textType)
	b.Register("hasSubsequence", b.boolType, b.textType, b.textType)
	b.Register("hasSubsequenceCaseInsensitive", b.boolType, b.textType, b.textType)
	b.Register("hasSubsequenceUTF8", b.boolType, b.textType, b.textType)
	b.Register("hasSubsequenceCaseInsensitiveUTF8", b.boolType, b.textType, b.textType)
}

// registerNgramDistanceAndSearchVariants covers ngramDistance, ngramSearch and their
// case-insensitive / UTF-8 variants.
//
// Both functions return a Float32 similarity score in the range [0, 1]; the catalogue
// records the engine-native Float32 width so emitters preserve the narrower numeric type.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered variant functions.
func registerNgramDistanceAndSearchVariants(b *FunctionCatalogueBuilder) {
	for _, suffix := range multiSearchCaseAndUTF8Suffixes {
		b.Register("ngramDistance"+suffix, b.float32Type, b.textType, b.textType)
		b.Register("ngramSearch"+suffix, b.float32Type, b.textType, b.textType)
	}
}

// registerHighlightAndCaseInsensitiveTokenOrNull covers the highlight helper that wraps
// matched needles in marker characters and the hasTokenCaseInsensitiveOrNull helper that
// returns a nullable UInt8.
//
// ClickHouse models the nullable result here so downstream emitters can preserve the SQL
// NULL semantic.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerHighlightAndCaseInsensitiveTokenOrNull(b *FunctionCatalogueBuilder) {
	b.Register("highlight", b.textType, b.textType, arrayOf(b.textType))
	b.Register("hasTokenCaseInsensitiveOrNull", b.uint64Type, b.textType, b.textType)
}
