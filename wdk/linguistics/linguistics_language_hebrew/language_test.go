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

package linguistics_language_hebrew_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
	_ "piko.sh/piko/wdk/linguistics/linguistics_language_hebrew"
)

const hebrewLanguageCode = "hebrew"

func TestLanguageBundle_RegistersStemmer(t *testing.T) {
	t.Parallel()

	stemmer := linguistics_domain.CreateStemmer(hebrewLanguageCode)
	require.NotNil(t, stemmer)
	assert.Equal(t, hebrewLanguageCode, stemmer.GetLanguage())
	assert.Equal(t, "בנק", stemmer.Stem("הבנק"))
}

func TestLanguageBundle_RegistersPhoneticEncoder(t *testing.T) {
	t.Parallel()

	encoder := linguistics_domain.CreatePhoneticEncoder(hebrewLanguageCode)
	require.NotNil(t, encoder)
	assert.Equal(t, hebrewLanguageCode, encoder.GetLanguage())
	assert.NotEmpty(t, encoder.Encode("שלום"))
}

func TestLanguageBundle_RegistersStopWords(t *testing.T) {
	t.Parallel()

	provider := linguistics_domain.CreateStopWordsProvider(hebrewLanguageCode)
	require.NotNil(t, provider)
	words := provider.GetStopWords(hebrewLanguageCode)
	assert.NotEmpty(t, words)
	assert.True(t, words["אני"])
}

func TestLanguageBundle_AnalyserIntegration(t *testing.T) {
	t.Parallel()

	config := linguistics_domain.DefaultConfigForLanguage(hebrewLanguageCode)
	config.Mode = linguistics_domain.AnalysisModeSmart
	analyser := linguistics_domain.NewAnalyser(
		config,
		linguistics_domain.WithLanguage(hebrewLanguageCode),
	)

	tokens := analyser.Analyse("שלום עולם")
	require.NotEmpty(t, tokens)
	for _, token := range tokens {
		assert.NotEmpty(t, token.Stemmed, "smart mode should populate stemmed for %q", token.Original)
	}
}
