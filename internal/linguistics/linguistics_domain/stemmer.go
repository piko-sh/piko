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

import "strings"

const (
	// LanguageEnglish is the default language for text analysis.
	LanguageEnglish = "english"

	// LanguageDutch is the language code for Dutch text processing.
	LanguageDutch = "dutch"

	// LanguageGerman is the language code for German.
	LanguageGerman = "german"

	// LanguageSpanish is the language code for Spanish.
	LanguageSpanish = "spanish"

	// LanguageFrench is the language code for French.
	LanguageFrench = "french"

	// LanguageRussian is the language code for Russian.
	LanguageRussian = "russian"

	// LanguageSwedish is the language code for Swedish.
	LanguageSwedish = "swedish"

	// LanguageNorwegian is the language code for Norwegian.
	LanguageNorwegian = "norwegian"

	// LanguageHungarian is the language code for Hungarian.
	LanguageHungarian = "hungarian"
)

// ValidateLanguage normalises a language string by lowercasing and trimming
// whitespace. If the result is empty, returns LanguageEnglish as the default.
//
// Takes language (string) which is the language code to validate.
//
// Returns string which is the normalised language code.
func ValidateLanguage(language string) string {
	normalised := strings.ToLower(strings.TrimSpace(language))
	if normalised == "" {
		return LanguageEnglish
	}
	return normalised
}

// SupportedLanguages returns a list of commonly supported language names for
// backwards compatibility. Actual language support depends on which stemmer
// adapter is used.
//
// Returns []string which contains the language names.
func SupportedLanguages() []string {
	return []string{
		LanguageEnglish,
		LanguageSpanish,
		LanguageFrench,
		LanguageGerman,
		LanguageDutch,
		LanguageRussian,
		LanguageSwedish,
		LanguageNorwegian,
		LanguageHungarian,
	}
}
