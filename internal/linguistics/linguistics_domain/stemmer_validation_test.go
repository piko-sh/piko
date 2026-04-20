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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateLanguage_Lowercase(t *testing.T) {
	assert.Equal(t, "english", ValidateLanguage("ENGLISH"))
	assert.Equal(t, "french", ValidateLanguage("French"))
	assert.Equal(t, "german", ValidateLanguage("GERMAN"))
}

func TestValidateLanguage_TrimWhitespace(t *testing.T) {
	assert.Equal(t, "french", ValidateLanguage("  french  "))
	assert.Equal(t, "english", ValidateLanguage("\tenglish\t"))
}

func TestValidateLanguage_EmptyDefaultsToEnglish(t *testing.T) {
	assert.Equal(t, LanguageEnglish, ValidateLanguage(""))
}

func TestValidateLanguage_WhitespaceOnlyDefaultsToEnglish(t *testing.T) {
	assert.Equal(t, LanguageEnglish, ValidateLanguage("   "))
	assert.Equal(t, LanguageEnglish, ValidateLanguage("\t\n"))
}

func TestValidateLanguage_PreservesValidInput(t *testing.T) {
	assert.Equal(t, "dutch", ValidateLanguage("dutch"))
	assert.Equal(t, "swedish", ValidateLanguage("swedish"))
}

func TestSupportedLanguages_ContainsExpected(t *testing.T) {
	langs := SupportedLanguages()

	expected := []string{
		LanguageEnglish, LanguageSpanish, LanguageFrench,
		LanguageGerman, LanguageDutch, LanguageRussian,
		LanguageSwedish, LanguageNorwegian, LanguageHungarian,
		LanguageHebrew,
	}
	for _, lang := range expected {
		assert.Contains(t, langs, lang, "should contain %s", lang)
	}
}

func TestSupportedLanguages_NoDuplicates(t *testing.T) {
	languages := SupportedLanguages()
	seen := make(map[string]struct{}, len(languages))
	for _, language := range languages {
		_, duplicate := seen[language]
		assert.False(t, duplicate, "language %q appears more than once", language)
		seen[language] = struct{}{}
	}
	assert.GreaterOrEqual(t, len(languages), 10)
}
