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
	"piko.sh/piko/internal/linguistics/linguistics_domain"
	search_fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

// createAnalyserConfigFromSearchConfig creates a linguistics.AnalyserConfig
// from search build parameters.
//
// This is the single source of truth for creating analyser configurations.
// Used by IndexBuilder during index construction.
//
// Ensures that the same analysis pipeline is used consistently across:
//   - Index building (build time)
//   - Query processing (runtime)
//
// Takes mode (search_fb.SearchMode) which specifies the search mode to map to
// an analysis mode; defaults to Fast mode for invalid or unrecognised modes.
// Takes language (string) which specifies the language for stop words and
// validation; will be normalised if invalid.
// Takes minTokenLength (int) which sets the minimum token length to keep.
// Takes maxTokenLength (int) which sets the maximum token length to keep.
// Takes stopWordsEnabled (bool) which controls whether language-specific stop
// words are loaded.
//
// Returns linguistics_domain.AnalyserConfig which is the configured analyser
// ready for use in indexing or query processing.
func createAnalyserConfigFromSearchConfig(
	mode search_fb.SearchMode,
	language string,
	minTokenLength int,
	maxTokenLength int,
	stopWordsEnabled bool,
) linguistics_domain.AnalyserConfig {
	var analysisMode linguistics_domain.AnalysisMode

	switch mode {
	case search_fb.SearchModeSmart:
		analysisMode = linguistics_domain.AnalysisModeSmart
	default:
		analysisMode = linguistics_domain.AnalysisModeFast
	}

	language = linguistics_domain.ValidateLanguage(language)

	var stopWords map[string]bool
	if stopWordsEnabled {
		stopWords = linguistics_domain.DefaultStopWords(language)
	}

	return linguistics_domain.AnalyserConfig{
		Mode:           analysisMode,
		Language:       language,
		StopWords:      stopWords,
		MinTokenLength: minTokenLength,
		MaxTokenLength: maxTokenLength,
		PreserveCase:   false,
	}
}

// createAnalyserConfigFromIndex creates a linguistics.AnalyserConfig by reading
// parameters from a search index.
//
// This means runtime query analysis uses the exact same configuration as
// build-time indexing, guaranteeing consistency. Used by QueryProcessor to
// configure the analyser for query processing.
//
// Takes mode (search_fb.SearchMode) which specifies the search mode.
// Takes language (string) which specifies the language for analysis.
//
// Returns linguistics_domain.AnalyserConfig which is ready for query
// processing.
func createAnalyserConfigFromIndex(mode search_fb.SearchMode, language string) linguistics_domain.AnalyserConfig {
	const defaultMinTokenLength = 2
	const defaultMaxTokenLength = 50

	return createAnalyserConfigFromSearchConfig(
		mode,
		language,
		defaultMinTokenLength,
		defaultMaxTokenLength,
		true,
	)
}

// createAnalyserConfigFromIndexParams creates a linguistics.AnalyserConfig
// from index parameters stored in the FlatBuffer.
//
// This provides perfect consistency by reading all parameters from the index.
// Use this when IndexParams are fully populated and accessible.
//
// Takes mode (search_fb.SearchMode) which specifies the search mode to use.
// Takes language (string) which specifies the language for text analysis.
// Takes minTokenLength (int) which sets the minimum token length to index.
// Takes maxTokenLength (int) which sets the maximum token length to index.
// Takes stopWordsEnabled (bool) which controls whether stop words are filtered.
//
// Returns linguistics_domain.AnalyserConfig which is configured for the given
// params.
func createAnalyserConfigFromIndexParams(
	mode search_fb.SearchMode,
	language string,
	minTokenLength int,
	maxTokenLength int,
	stopWordsEnabled bool,
) linguistics_domain.AnalyserConfig {
	return createAnalyserConfigFromSearchConfig(
		mode,
		language,
		minTokenLength,
		maxTokenLength,
		stopWordsEnabled,
	)
}
