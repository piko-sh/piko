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

// AnalyserPort defines the interface for the text analysis pipeline.
// It handles tokenisation, normalisation, stemming, and phonetic encoding.
type AnalyserPort interface {
	// Analyse performs a full text analysis based on the set mode.
	//
	// Takes text (string) which is the input to analyse.
	//
	// Returns []Token which contains tokens with their normalised, stemmed,
	// or phonetic forms filled in.
	Analyse(text string) []Token

	// AnalyseToStrings returns the normalised token strings for quick processing.
	//
	// Takes text (string) which is the input to analyse.
	//
	// Returns []string which contains the normalised tokens.
	AnalyseToStrings(text string) []string

	// AnalyseToStemmed returns the stemmed forms of tokens from the given text.
	// This method is only available in Smart mode.
	//
	// Takes text (string) which is the input to analyse.
	//
	// Returns []string which contains the stemmed token forms.
	AnalyseToStemmed(text string) []string

	// AnalyseToPhonetic returns the phonetic codes for each token in the text.
	// This method only works in Smart mode.
	//
	// Takes text (string) which is the input to analyse.
	//
	// Returns []string which contains the phonetic code for each token.
	AnalyseToPhonetic(text string) []string

	// GetStemmer returns the underlying stemmer for direct use.
	GetStemmer() StemmerPort

	// GetPhoneticEncoder returns the phonetic encoder for direct use.
	//
	// Returns PhoneticEncoderPort which provides access to the phonetic encoder.
	GetPhoneticEncoder() PhoneticEncoderPort

	// GetMode returns the current analysis mode.
	GetMode() AnalysisMode
}

// TokeniserPort defines the contract for text tokenisation.
// It splits text into tokens with position and offset tracking.
type TokeniserPort interface {
	// Tokenise splits the input text into tokens.
	//
	// Takes text (string) which is the input to split.
	//
	// Returns []Token which contains the tokens with their positions and offsets.
	Tokenise(text string) []Token

	// TokeniseToStrings is a convenience method that returns just the normalised
	// token strings. Useful when position information is not needed.
	//
	// Takes text (string) which is the input to tokenise.
	//
	// Returns []string which contains the normalised tokens.
	TokeniseToStrings(text string) []string
}

// NormaliserPort defines the interface for text normalisation.
// It handles Unicode normalisation, diacritic removal, and case folding.
type NormaliserPort interface {
	// Normalise applies all normalisation changes to the input text.
	//
	// Takes text (string) which is the text to normalise.
	//
	// Returns string which is the normalised text ready for indexing and search.
	Normalise(text string) string

	// NormaliseRune normalises a single rune for character-level operations.
	//
	// Takes r (rune) which is the character to normalise.
	//
	// Returns rune which is the normalised form of the input character.
	NormaliseRune(r rune) rune
}

// StemmerPort defines the contract for multi-language word stemming.
// It reduces words to their root form using the Snowball algorithm.
type StemmerPort interface {
	// Stem reduces a word to its root form using the Snowball algorithm. The input
	// word should already be normalised (lowercase, with diacritics removed).
	//
	// Takes word (string) which is the word to stem.
	//
	// Returns string which is the stemmed word, or the original if stemming fails.
	Stem(word string) string

	// GetLanguage returns the language used by this stemmer.
	GetLanguage() string
}

// PhoneticEncoderPort defines the contract for phonetic encoding.
// It converts words to phonetic codes for "sounds-like" matching.
type PhoneticEncoderPort interface {
	// Encode returns the main phonetic code for a word.
	// The input should be in lowercase with no accents.
	//
	// Takes word (string) which is the word to encode.
	//
	// Returns string which is the phonetic code, usually four characters.
	Encode(word string) string

	// GetLanguage returns the language this encoder supports.
	//
	// Returns string which is the language code (e.g., "english", "french").
	GetLanguage() string
}

// StopWordsProviderPort defines the contract for stop words providers.
// It provides language-specific stop word lists for text analysis.
type StopWordsProviderPort interface {
	// GetStopWords returns the stop words for a given language.
	//
	// Takes language (string) which is the language code (e.g., "english").
	//
	// Returns map[string]bool which contains stop words as keys for fast lookup.
	// Returns an empty map if the language is not supported.
	GetStopWords(language string) map[string]bool

	// SupportedLanguages returns the list of languages this provider supports.
	//
	// Returns []string which contains the language codes.
	SupportedLanguages() []string
}

var (
	_ AnalyserPort = (*Analyser)(nil)

	_ TokeniserPort = (*Tokeniser)(nil)

	_ NormaliserPort = (*Normaliser)(nil)

	_ PhoneticEncoderPort = (*PhoneticEncoder)(nil)
)
