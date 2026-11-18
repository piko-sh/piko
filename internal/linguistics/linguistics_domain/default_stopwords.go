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
	_ "embed"
	"strings"
)

var (
	// defaultEnglishStopWordsText holds the built-in English stop words.
	//
	//go:embed default_stopwords/english.txt
	defaultEnglishStopWordsText string

	// defaultSpanishStopWordsText holds the built-in Spanish stop words.
	//
	//go:embed default_stopwords/spanish.txt
	defaultSpanishStopWordsText string

	// defaultFrenchStopWordsText holds the built-in French stop words.
	//
	//go:embed default_stopwords/french.txt
	defaultFrenchStopWordsText string

	// defaultRussianStopWordsText holds the built-in Russian stop words.
	//
	//go:embed default_stopwords/russian.txt
	defaultRussianStopWordsText string

	// defaultDutchStopWordsText holds the built-in Dutch stop words.
	//
	//go:embed default_stopwords/dutch.txt
	defaultDutchStopWordsText string

	// defaultGermanStopWordsText holds the built-in German stop words.
	//
	//go:embed default_stopwords/german.txt
	defaultGermanStopWordsText string

	// defaultSwedishStopWordsText holds the built-in Swedish stop words.
	//
	//go:embed default_stopwords/swedish.txt
	defaultSwedishStopWordsText string

	// defaultNorwegianStopWordsText holds the built-in Norwegian stop words.
	//
	//go:embed default_stopwords/norwegian.txt
	defaultNorwegianStopWordsText string

	// defaultHungarianStopWordsText holds the built-in Hungarian stop words.
	//
	//go:embed default_stopwords/hungarian.txt
	defaultHungarianStopWordsText string
)

// defaultEnglishStopWords returns the built-in English stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultEnglishStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultEnglishStopWordsText))
}

// defaultSpanishStopWords returns the built-in Spanish stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultSpanishStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultSpanishStopWordsText))
}

// defaultFrenchStopWords returns the built-in French stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultFrenchStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultFrenchStopWordsText))
}

// defaultRussianStopWords returns the built-in Russian stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultRussianStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultRussianStopWordsText))
}

// defaultDutchStopWords returns the built-in Dutch stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultDutchStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultDutchStopWordsText))
}

// defaultGermanStopWords returns the built-in German stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultGermanStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultGermanStopWordsText))
}

// defaultSwedishStopWords returns the built-in Swedish stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultSwedishStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultSwedishStopWordsText))
}

// defaultNorwegianStopWords returns the built-in Norwegian stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultNorwegianStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultNorwegianStopWordsText))
}

// defaultHungarianStopWords returns the built-in Hungarian stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultHungarianStopWords() map[string]bool {
	return makeStopWordSet(strings.Fields(defaultHungarianStopWordsText))
}
