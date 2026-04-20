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

package linguistics_stopwords_hebrew

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

func TestNew(t *testing.T) {
	t.Parallel()

	provider, err := New()
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestFactory(t *testing.T) {
	t.Parallel()

	provider, err := Factory()
	require.NoError(t, err)
	require.NotNil(t, provider)
	var _ linguistics_domain.StopWordsProviderPort = provider
}

func TestProvider_GetStopWords(t *testing.T) {
	t.Parallel()

	provider := &Provider{}
	words := provider.GetStopWords(Language)
	assert.NotEmpty(t, words)
	assert.True(t, words["אני"], "expected pronoun אני to be a stop word")
	assert.True(t, words["של"], "expected preposition של to be a stop word")
	assert.True(t, words["לא"], "expected negation לא to be a stop word")
	assert.False(t, words["שלום"], "shalom should not be a stop word")
}

func TestProvider_GetStopWords_IgnoresLanguageArg(t *testing.T) {
	t.Parallel()

	provider := &Provider{}
	withHebrew := provider.GetStopWords("hebrew")
	withEmpty := provider.GetStopWords("")
	withOther := provider.GetStopWords("klingon")
	assert.Equal(t, len(withHebrew), len(withEmpty))
	assert.Equal(t, len(withHebrew), len(withOther))
}

func TestProvider_SupportedLanguages(t *testing.T) {
	t.Parallel()

	provider := &Provider{}
	supported := provider.SupportedLanguages()
	assert.Equal(t, []string{Language}, supported)
	assert.Equal(t, []string{"hebrew"}, supported)
}

func TestProvider_RegisteredViaInit(t *testing.T) {
	t.Parallel()

	port := linguistics_domain.CreateStopWordsProvider(Language)
	require.NotNil(t, port)
	assert.Contains(t, port.SupportedLanguages(), Language)
}

func TestStopWords_NoPrefixedForms(t *testing.T) {
	t.Parallel()

	prefixedSamples := []string{"ואני", "שלא", "בכל", "ולא"}
	for _, word := range prefixedSamples {
		assert.False(t, stopWords[word], "prefixed form %q should not appear in base stop word list", word)
	}
}
