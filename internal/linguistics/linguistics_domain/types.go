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

const (
	// DefaultMinTokenLength is the smallest token length in characters.
	DefaultMinTokenLength = 2

	// DefaultMaxTokenLength is the longest token length to process.
	DefaultMaxTokenLength = 50

	// DefaultPhoneticCodeLength is the standard length of codes that the phonetic encoder
	// produces.
	DefaultPhoneticCodeLength = 4
)

// Token represents a single analysed token from text with its various forms and position.
type Token struct {
	// Original is the token as it appeared in the source text.
	Original string

	// Normalised is the lowercase form with accents and special marks removed.
	Normalised string

	// Stemmed is the root form of the word (e.g., "running" becomes "run"). Only populated
	// when using a stemmer.
	Stemmed string

	// Phonetic is the Double Metaphone encoding of the token. Only set when phonetic
	// encoding is enabled.
	Phonetic string

	// Position is the offset of this token in the original text, measured in tokens rather
	// than bytes.
	Position int

	// ByteOffset is the starting byte position in the original text.
	ByteOffset int

	// ByteLength is the length in bytes of the original token.
	ByteLength int
}

// AnalysisMode specifies which text analysis methods to use.
type AnalysisMode int

const (
	// AnalysisModeBasic performs only tokenisation and normalisation. This is the fastest
	// option, suitable for exact and prefix matching.
	AnalysisModeBasic AnalysisMode = iota

	// AnalysisModeFast performs tokenisation, normalisation, and stop word filtering. Good
	// balance of performance and quality for most use cases.
	AnalysisModeFast

	// AnalysisModeSmart performs full linguistic analysis including stemming and phonetics.
	// Best for fuzzy matching and handling misspellings.
	AnalysisModeSmart
)

// AnalyserConfig configures the text analysis pipeline. Fields ordered for optimal memory
// alignment (larger types first).
type AnalyserConfig struct {
	// StopWords is a set of words to filter out (e.g., "the", "and", "is"). Only used in
	// Fast and Smart modes.
	StopWords map[string]bool

	// Language specifies the language for stemming, defaulting to "english".
	Language string

	// MinTokenLength filters out tokens shorter than this value; 0 means no minimum.
	MinTokenLength int

	// MaxTokenLength sets the maximum length for tokens; 0 means no limit.
	MaxTokenLength int

	// Mode specifies which analysis techniques to apply during text processing.
	Mode AnalysisMode

	// PreserveCase prevents lowercasing when true; useful for code identifiers.
	PreserveCase bool
}

// DefaultConfig returns a default configuration for general text analysis.
//
// Returns AnalyserConfig which provides sensible defaults for English text.
func DefaultConfig() AnalyserConfig {
	return DefaultConfigForLanguage(LanguageEnglish)
}

// DefaultConfigForLanguage returns a default configuration for a given language. The stop
// words are set based on the language code.
//
// Takes language (string) which specifies the language code for the analyser.
//
// Returns AnalyserConfig which contains the default settings for the given language.
func DefaultConfigForLanguage(language string) AnalyserConfig {
	language = ValidateLanguage(language)

	return AnalyserConfig{
		Mode:           AnalysisModeFast,
		Language:       language,
		StopWords:      DefaultStopWords(language),
		MinTokenLength: DefaultMinTokenLength,
		MaxTokenLength: DefaultMaxTokenLength,
		PreserveCase:   false,
	}
}

// DefaultStopWords returns the default stop words for the specified language. It first
// tries the builtin stop words registry, then falls back to hardcoded stop words for
// backward compatibility.
//
// Takes language (string) which specifies the language code (e.g. "english", "spanish",
// "french", "german").
//
// Returns map[string]bool which contains the stop words as keys with true values for
// quick lookup.
func DefaultStopWords(language string) map[string]bool {
	provider := CreateStopWordsProvider("builtin")
	stopWords := provider.GetStopWords(language)
	if len(stopWords) > 0 {
		return stopWords
	}

	return defaultStopWordsFallback(language)
}

// defaultStopWordsFallback provides hardcoded stop words for backward compatibility when
// no stop words adapter is imported.
//
// Takes language (string) which specifies the language for stop words.
//
// Returns map[string]bool which contains the stop words for the given language,
// defaulting to English if the language is not supported.
func defaultStopWordsFallback(language string) map[string]bool {
	switch language {
	case "spanish":
		return defaultSpanishStopWords()
	case "french":
		return defaultFrenchStopWords()
	case "german":
		return defaultGermanStopWords()
	case "dutch":
		return defaultDutchStopWords()
	case "russian":
		return defaultRussianStopWords()
	case "swedish":
		return defaultSwedishStopWords()
	case "norwegian":
		return defaultNorwegianStopWords()
	case "hungarian":
		return defaultHungarianStopWords()
	default:
		return defaultEnglishStopWords()
	}
}

// makeStopWordSet converts a slice of stop words into a set for fast lookup.
//
// Takes stopWords ([]string) which contains the words to include in the set.
//
// Returns map[string]bool which maps each stop word to true for quick lookups.
func makeStopWordSet(stopWords []string) map[string]bool {
	set := make(map[string]bool, len(stopWords))
	for _, word := range stopWords {
		set[word] = true
	}
	return set
}
