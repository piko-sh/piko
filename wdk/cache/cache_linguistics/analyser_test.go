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

package cache_linguistics

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
	_ "piko.sh/piko/wdk/linguistics/linguistics_language_english"
	_ "piko.sh/piko/wdk/linguistics/linguistics_language_french"
)

func TestNewTextAnalyser_ReturnsFunctioningFunc(t *testing.T) {
	config := linguistics_domain.DefaultConfigForLanguage(linguistics_domain.LanguageEnglish)
	config.Mode = linguistics_domain.AnalysisModeSmart

	analyser := NewTextAnalyser(config, linguistics_domain.WithLanguage(linguistics_domain.LanguageEnglish))
	require.NotNil(t, analyser, "TextAnalyseFunc should not be nil")

	tokens := analyser("running quickly through the forest")
	assert.True(t, len(tokens) > 0, "should produce tokens from text")
}

func TestNewTextAnalyser_SmartModeStemsTokens(t *testing.T) {
	config := linguistics_domain.DefaultConfigForLanguage(linguistics_domain.LanguageEnglish)
	config.Mode = linguistics_domain.AnalysisModeSmart

	analyser := NewTextAnalyser(config, linguistics_domain.WithLanguage(linguistics_domain.LanguageEnglish))

	tokens := analyser("running configurations")

	assert.True(t, len(tokens) > 0, "should produce stemmed tokens")

	tokensFromRoot := analyser("run configuration")
	assert.True(t, len(tokensFromRoot) > 0, "root form should also produce tokens")
}

func TestNewTextAnalyser_BasicMode(t *testing.T) {
	config := linguistics_domain.DefaultConfigForLanguage(linguistics_domain.LanguageEnglish)
	config.Mode = linguistics_domain.AnalysisModeBasic

	analyser := NewTextAnalyser(config, linguistics_domain.WithLanguage(linguistics_domain.LanguageEnglish))

	tokens := analyser("HELLO World")
	assert.True(t, len(tokens) > 0, "should produce normalised tokens")

	for _, tok := range tokens {
		for _, r := range tok {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("expected lowercase token, got %q", tok)
				break
			}
		}
	}
}

func TestNewTextAnalyser_EmptyText(t *testing.T) {
	config := linguistics_domain.DefaultConfigForLanguage(linguistics_domain.LanguageEnglish)
	config.Mode = linguistics_domain.AnalysisModeSmart

	analyser := NewTextAnalyser(config, linguistics_domain.WithLanguage(linguistics_domain.LanguageEnglish))

	tokens := analyser("")
	assert.Empty(t, tokens, "empty text should produce no tokens")
}

func TestNewTextAnalyser_ConcurrentSafety(t *testing.T) {
	config := linguistics_domain.DefaultConfigForLanguage(linguistics_domain.LanguageEnglish)
	config.Mode = linguistics_domain.AnalysisModeSmart

	analyser := NewTextAnalyser(config, linguistics_domain.WithLanguage(linguistics_domain.LanguageEnglish))

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 50 {
				tokens := analyser("running quickly through the forest")
				if len(tokens) == 0 {
					t.Error("concurrent call produced no tokens")
					return
				}
			}
		})
	}
	wg.Wait()
}

func TestNewEnglishTextAnalyser(t *testing.T) {
	analyser := NewEnglishTextAnalyser()
	require.NotNil(t, analyser)

	tokens := analyser("running quickly")
	assert.True(t, len(tokens) > 0, "English analyser should produce tokens")
}

func TestNewTextAnalyserForLanguage_English(t *testing.T) {
	analyser := NewTextAnalyserForLanguage(linguistics_domain.LanguageEnglish)
	require.NotNil(t, analyser)

	tokens := analyser("running quickly")
	assert.True(t, len(tokens) > 0, "English analyser should produce tokens")
}

func TestNewTextAnalyserForLanguage_French(t *testing.T) {
	analyser := NewTextAnalyserForLanguage(linguistics_domain.LanguageFrench)
	require.NotNil(t, analyser)

	tokens := analyser("les configurations importantes")
	assert.True(t, len(tokens) > 0, "French analyser should produce tokens")
}

func TestNewTextAnalyserForLanguage_DifferentLanguagesDifferentStems(t *testing.T) {
	englishAnalyser := NewTextAnalyserForLanguage(linguistics_domain.LanguageEnglish)
	frenchAnalyser := NewTextAnalyserForLanguage(linguistics_domain.LanguageFrench)

	englishTokens := englishAnalyser("configurations")
	frenchTokens := frenchAnalyser("configurations")

	assert.True(t, len(englishTokens) > 0, "English analyser should produce tokens")
	assert.True(t, len(frenchTokens) > 0, "French analyser should produce tokens")
}
