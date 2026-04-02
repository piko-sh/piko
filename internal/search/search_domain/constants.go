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

const (
	// BM25DefaultK1 is the default term frequency saturation parameter for BM25
	// scoring. It controls non-linear term frequency normalisation, with a typical
	// range of 1.2 to 2.0.
	BM25DefaultK1 = 1.2

	// BM25DefaultB is the default length normalisation parameter for BM25 scoring.
	// It controls the degree to which document length normalises term frequency
	// values, with a typical range of 0.5 to 0.8.
	BM25DefaultB = 0.75

	// DefaultFieldWeightTitle is the default weight multiplier for title fields.
	// Titles are given higher importance in search results.
	DefaultFieldWeightTitle = 2.5

	// DefaultFieldWeightContent is the default weight multiplier for content
	// fields. Content is the baseline weight.
	DefaultFieldWeightContent = 1.0

	// DefaultFieldWeightExcerpt is the default weight multiplier for excerpt
	// fields. Excerpts have moderate importance.
	DefaultFieldWeightExcerpt = 1.5

	// DefaultMinTokenLength is the shortest token length that will be indexed.
	DefaultMinTokenLength = 2

	// DefaultMaxTokenLength is the longest token length to include in the index.
	DefaultMaxTokenLength = 50

	// fuzzyMatchHighSimilarity is the similarity score above which a fuzzy match
	// is considered high confidence.
	fuzzyMatchHighSimilarity = 0.98

	// fuzzyMatchLowSimilarity is the minimum threshold for acceptable fuzzy
	// matches.
	fuzzyMatchLowSimilarity = 0.80

	// LanguageEnglish is the default language used for search indexing.
	LanguageEnglish = "english"

	// numSearchableFields is the number of searchable fields (title, content,
	// excerpt).
	numSearchableFields = 3

	// fieldIDTitle is the field identifier for the title field.
	fieldIDTitle uint8 = 0

	// fieldIDContent is the field identifier for the main body content.
	fieldIDContent uint8 = 1

	// fieldIDExcerpt is the field ID for excerpt or summary text.
	fieldIDExcerpt uint8 = 2

	// frontmatterDelimiterLength is the length of the "---" YAML frontmatter
	// marker.
	frontmatterDelimiterLength = 3

	// htmlCommentEndLength is the length of the HTML comment end marker "-->".
	htmlCommentEndLength = 3

	// minHorizontalRuleLength is the minimum length for horizontal rules
	// such as ---, ***, or ___.
	minHorizontalRuleLength = 3

	// newlineChar is the newline character used to split and join lines.
	newlineChar = "\n"
)

// fieldText represents a segment of text tagged with its source field.
// It tracks which field each token came from during index building, which
// allows for field-aware BM25 scoring.
type fieldText struct {
	// text holds the content to be tokenised and indexed.
	text string

	// fieldID identifies the source field (fieldIDTitle, fieldIDContent,
	// fieldIDExcerpt).
	fieldID uint8
}
