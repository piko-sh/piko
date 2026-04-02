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
	"testing"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

func TestCreateAnalyserConfigFromSearchConfig(t *testing.T) {
	testCases := []struct {
		name             string
		language         string
		wantLanguage     string
		minTokenLength   int
		maxTokenLength   int
		wantMode         linguistics_domain.AnalysisMode
		mode             search_schema_gen.SearchMode
		stopWordsEnabled bool
		wantStopWords    bool
	}{
		{
			name:             "fast mode with english and stop words",
			mode:             search_schema_gen.SearchModeFast,
			language:         "english",
			minTokenLength:   2,
			maxTokenLength:   50,
			stopWordsEnabled: true,
			wantMode:         linguistics_domain.AnalysisModeFast,
			wantLanguage:     "english",
			wantStopWords:    true,
		},
		{
			name:             "smart mode with spanish and stop words",
			mode:             search_schema_gen.SearchModeSmart,
			language:         "spanish",
			minTokenLength:   3,
			maxTokenLength:   40,
			stopWordsEnabled: true,
			wantMode:         linguistics_domain.AnalysisModeSmart,
			wantLanguage:     "spanish",
			wantStopWords:    true,
		},
		{
			name:             "fast mode with stop words disabled",
			mode:             search_schema_gen.SearchModeFast,
			language:         "english",
			minTokenLength:   2,
			maxTokenLength:   50,
			stopWordsEnabled: false,
			wantMode:         linguistics_domain.AnalysisModeFast,
			wantLanguage:     "english",
			wantStopWords:    false,
		},
		{
			name:             "invalid mode defaults to fast",
			mode:             search_schema_gen.SearchMode(99),
			language:         "english",
			minTokenLength:   2,
			maxTokenLength:   50,
			stopWordsEnabled: true,
			wantMode:         linguistics_domain.AnalysisModeFast,
			wantLanguage:     "english",
			wantStopWords:    true,
		},
		{
			name:             "unknown language passes through (uses NoOpStemmer)",
			mode:             search_schema_gen.SearchModeFast,
			language:         "klingon",
			minTokenLength:   2,
			maxTokenLength:   50,
			stopWordsEnabled: true,
			wantMode:         linguistics_domain.AnalysisModeFast,
			wantLanguage:     "klingon",
			wantStopWords:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createAnalyserConfigFromSearchConfig(
				tc.mode,
				tc.language,
				tc.minTokenLength,
				tc.maxTokenLength,
				tc.stopWordsEnabled,
			)

			if config.Mode != tc.wantMode {
				t.Errorf("Mode: got %v, want %v", config.Mode, tc.wantMode)
			}

			if config.Language != tc.wantLanguage {
				t.Errorf("Language: got %s, want %s", config.Language, tc.wantLanguage)
			}

			if config.MinTokenLength != tc.minTokenLength {
				t.Errorf("MinTokenLength: got %d, want %d", config.MinTokenLength, tc.minTokenLength)
			}

			if config.MaxTokenLength != tc.maxTokenLength {
				t.Errorf("MaxTokenLength: got %d, want %d", config.MaxTokenLength, tc.maxTokenLength)
			}

			if config.PreserveCase != false {
				t.Errorf("PreserveCase: got %v, want false", config.PreserveCase)
			}

			hasStopWords := len(config.StopWords) > 0
			if hasStopWords != tc.wantStopWords {
				t.Errorf("StopWords enabled: got %v, want %v", hasStopWords, tc.wantStopWords)
			}
		})
	}
}

func TestCreateAnalyserConfigFromIndex(t *testing.T) {
	testCases := []struct {
		name         string
		language     string
		wantLanguage string
		mode         search_schema_gen.SearchMode
		wantMode     linguistics_domain.AnalysisMode
	}{
		{
			name:         "fast mode with english",
			mode:         search_schema_gen.SearchModeFast,
			language:     "english",
			wantMode:     linguistics_domain.AnalysisModeFast,
			wantLanguage: "english",
		},
		{
			name:         "smart mode with french",
			mode:         search_schema_gen.SearchModeSmart,
			language:     "french",
			wantMode:     linguistics_domain.AnalysisModeSmart,
			wantLanguage: "french",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createAnalyserConfigFromIndex(tc.mode, tc.language)

			if config.Mode != tc.wantMode {
				t.Errorf("Mode: got %v, want %v", config.Mode, tc.wantMode)
			}

			if config.Language != tc.wantLanguage {
				t.Errorf("Language: got %s, want %s", config.Language, tc.wantLanguage)
			}

			if config.MinTokenLength != 2 {
				t.Errorf("MinTokenLength: got %d, want 2", config.MinTokenLength)
			}

			if config.MaxTokenLength != 50 {
				t.Errorf("MaxTokenLength: got %d, want 50", config.MaxTokenLength)
			}

			if len(config.StopWords) == 0 {
				t.Error("Expected stop words to be enabled")
			}
		})
	}
}

func TestCreateAnalyserConfigFromIndexParams(t *testing.T) {
	testCases := []struct {
		name             string
		language         string
		minTokenLength   int
		maxTokenLength   int
		wantMode         linguistics_domain.AnalysisMode
		mode             search_schema_gen.SearchMode
		stopWordsEnabled bool
	}{
		{
			name:             "fast mode with custom params",
			mode:             search_schema_gen.SearchModeFast,
			language:         "english",
			minTokenLength:   1,
			maxTokenLength:   100,
			stopWordsEnabled: true,
			wantMode:         linguistics_domain.AnalysisModeFast,
		},
		{
			name:             "smart mode with custom params",
			mode:             search_schema_gen.SearchModeSmart,
			language:         "russian",
			minTokenLength:   3,
			maxTokenLength:   60,
			stopWordsEnabled: false,
			wantMode:         linguistics_domain.AnalysisModeSmart,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createAnalyserConfigFromIndexParams(
				tc.mode,
				tc.language,
				tc.minTokenLength,
				tc.maxTokenLength,
				tc.stopWordsEnabled,
			)

			if config.Mode != tc.wantMode {
				t.Errorf("Mode: got %v, want %v", config.Mode, tc.wantMode)
			}

			if config.MinTokenLength != tc.minTokenLength {
				t.Errorf("MinTokenLength: got %d, want %d", config.MinTokenLength, tc.minTokenLength)
			}

			if config.MaxTokenLength != tc.maxTokenLength {
				t.Errorf("MaxTokenLength: got %d, want %d", config.MaxTokenLength, tc.maxTokenLength)
			}

			hasStopWords := len(config.StopWords) > 0
			if hasStopWords != tc.stopWordsEnabled {
				t.Errorf("StopWords enabled: got %v, want %v", hasStopWords, tc.stopWordsEnabled)
			}
		})
	}
}
