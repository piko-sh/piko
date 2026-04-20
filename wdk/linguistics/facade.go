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

package linguistics

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

const (
	// LanguageEnglish is the language code for English.
	LanguageEnglish = linguistics_domain.LanguageEnglish

	// LanguageDutch is the language code for Dutch.
	LanguageDutch = linguistics_domain.LanguageDutch

	// LanguageGerman is the language code for German.
	LanguageGerman = linguistics_domain.LanguageGerman

	// LanguageSpanish is the language code for Spanish.
	LanguageSpanish = linguistics_domain.LanguageSpanish

	// LanguageFrench is the language code for French.
	LanguageFrench = linguistics_domain.LanguageFrench

	// LanguageRussian is the language code for Russian.
	LanguageRussian = linguistics_domain.LanguageRussian

	// LanguageSwedish is the language code for Swedish.
	LanguageSwedish = linguistics_domain.LanguageSwedish

	// LanguageNorwegian is the language code for Norwegian.
	LanguageNorwegian = linguistics_domain.LanguageNorwegian

	// LanguageHungarian is the language code for Hungarian.
	LanguageHungarian = linguistics_domain.LanguageHungarian

	// LanguageHebrew is the language code for Hebrew.
	LanguageHebrew = linguistics_domain.LanguageHebrew

	// AnalysisModeBasic is the basic analysis mode that performs simple checks.
	AnalysisModeBasic = linguistics_domain.AnalysisModeBasic

	// AnalysisModeFast enables quick analysis that skips slower checks.
	AnalysisModeFast = linguistics_domain.AnalysisModeFast

	// AnalysisModeSmart selects analysis mode based on file characteristics.
	AnalysisModeSmart = linguistics_domain.AnalysisModeSmart

	// DefaultMinTokenLength is the default minimum token length for text analysis.
	DefaultMinTokenLength = linguistics_domain.DefaultMinTokenLength

	// DefaultMaxTokenLength is the default maximum length for tokens in text
	// processing.
	DefaultMaxTokenLength = linguistics_domain.DefaultMaxTokenLength

	// DefaultPhoneticCodeLength is the default length for generated phonetic codes.
	DefaultPhoneticCodeLength = linguistics_domain.DefaultPhoneticCodeLength
)

// Type aliases for core types.
type (
	// Token represents a single analysed token from text.
	Token = linguistics_domain.Token

	// AnalysisMode specifies which text analysis methods to use.
	AnalysisMode = linguistics_domain.AnalysisMode

	// AnalyserConfig configures the text analysis pipeline.
	AnalyserConfig = linguistics_domain.AnalyserConfig

	// Option configures an Analyser during construction.
	Option = linguistics_domain.Option
)

// Type aliases for port interfaces.
type (
	// AnalyserPort defines the interface for the text analysis pipeline.
	AnalyserPort = linguistics_domain.AnalyserPort

	// TokeniserPort defines the contract for text tokenisation.
	TokeniserPort = linguistics_domain.TokeniserPort

	// NormaliserPort defines the interface for text normalisation.
	NormaliserPort = linguistics_domain.NormaliserPort

	// StemmerPort defines the contract for multi-language word stemming.
	StemmerPort = linguistics_domain.StemmerPort

	// PhoneticEncoderPort defines the contract for phonetic encoding.
	PhoneticEncoderPort = linguistics_domain.PhoneticEncoderPort

	// StopWordsProviderPort defines the contract for stop words providers.
	StopWordsProviderPort = linguistics_domain.StopWordsProviderPort
)

// Type aliases for concrete implementations.
type (
	// Analyser orchestrates the complete text analysis pipeline.
	Analyser = linguistics_domain.Analyser

	// Tokeniser splits text into tokens with position tracking.
	Tokeniser = linguistics_domain.Tokeniser

	// Normaliser handles Unicode normalisation and case folding.
	Normaliser = linguistics_domain.Normaliser

	// PhoneticEncoder implements Double Metaphone phonetic encoding.
	PhoneticEncoder = linguistics_domain.PhoneticEncoder
)

// DefaultConfig returns a default configuration for general text analysis.
//
// Returns AnalyserConfig which provides sensible defaults for text analysis.
func DefaultConfig() AnalyserConfig {
	return linguistics_domain.DefaultConfig()
}

// DefaultConfigForLanguage returns a default configuration for a given language.
//
// Takes language (string) which specifies the language to configure.
//
// Returns AnalyserConfig which contains the default settings for the language.
func DefaultConfigForLanguage(language string) AnalyserConfig {
	return linguistics_domain.DefaultConfigForLanguage(language)
}

// ValidateLanguage normalises a language string.
//
// Takes language (string) which specifies the language to validate.
//
// Returns string which is the normalised language value.
func ValidateLanguage(language string) string {
	return linguistics_domain.ValidateLanguage(language)
}

// SupportedLanguages returns a list of commonly supported languages.
//
// Returns []string which contains the language codes that are supported.
func SupportedLanguages() []string {
	return linguistics_domain.SupportedLanguages()
}

// NewAnalyser creates a new text analyser with the given configuration.
//
// Takes config (AnalyserConfig) which specifies the analyser settings.
// Takes opts (...Option) which provides optional behaviour controls.
//
// Returns *Analyser which is the configured text analyser ready for use.
func NewAnalyser(config AnalyserConfig, opts ...Option) *Analyser {
	return linguistics_domain.NewAnalyser(config, opts...)
}

// NewTokeniser creates a new tokeniser with the given configuration.
//
// Takes config (AnalyserConfig) which specifies the tokenisation settings.
//
// Returns *Tokeniser which is the configured tokeniser ready for use.
func NewTokeniser(config AnalyserConfig) *Tokeniser {
	return linguistics_domain.NewTokeniser(config)
}

// NewNormaliser creates a new text normaliser.
//
// Takes preserveCase (bool) which controls whether letter case is preserved
// during normalisation.
//
// Returns *Normaliser which is the configured normaliser ready for use.
func NewNormaliser(preserveCase bool) *Normaliser {
	return linguistics_domain.NewNormaliser(preserveCase)
}

// NewPhoneticEncoder creates a new Double Metaphone encoder.
//
// Takes maxLength (int) which specifies the maximum length of the encoded
// output.
//
// Returns *PhoneticEncoder which is the configured encoder ready for use.
func NewPhoneticEncoder(maxLength int) *PhoneticEncoder {
	return linguistics_domain.NewPhoneticEncoder(maxLength)
}

// WithStemmer sets a custom stemmer implementation.
//
// Takes stemmer (StemmerPort) which provides the stemming algorithm to use.
//
// Returns Option which configures the linguistics service with the stemmer.
func WithStemmer(stemmer StemmerPort) Option {
	return linguistics_domain.WithStemmer(stemmer)
}

// WithTokeniser sets a custom tokeniser implementation.
//
// Takes tokeniser (TokeniserPort) which provides the tokenisation logic.
//
// Returns Option which configures the tokeniser setting.
func WithTokeniser(tokeniser TokeniserPort) Option {
	return linguistics_domain.WithTokeniser(tokeniser)
}

// WithPhoneticEncoder sets a custom phonetic encoder implementation.
//
// Takes encoder (PhoneticEncoderPort) which provides the phonetic encoding
// logic.
//
// Returns Option which configures the linguistics service to use the given
// encoder.
func WithPhoneticEncoder(encoder PhoneticEncoderPort) Option {
	return linguistics_domain.WithPhoneticEncoder(encoder)
}

// WithStopWordsProvider sets a custom stop words provider.
//
// Takes provider (StopWordsProviderPort) which supplies the stop words to use.
//
// Returns Option which configures the linguistics service.
func WithStopWordsProvider(provider StopWordsProviderPort) Option {
	return linguistics_domain.WithStopWordsProvider(provider)
}

// WithLanguage configures the analyser with stemmer, phonetic encoder, and
// stop words provider for the specified language. This is a convenience
// function that looks up all three components from their respective registries.
//
// Takes language (string) which specifies the language to configure.
//
// Returns Option which applies the language configuration to an analyser.
//
// Example usage:
//
//	import _ "piko.sh/piko/wdk/linguistics/linguistics_language_english"
//
//	config := linguistics.DefaultConfigForLanguage("english")
//	analyser := linguistics.NewAnalyser(config, linguistics.WithLanguage("english"))
func WithLanguage(language string) Option {
	return linguistics_domain.WithLanguage(language)
}

// RegisterStemmerFactory registers a stemmer factory for a language.
//
// Takes language (string) which identifies the language for the stemmer.
// Takes factory (func) which creates a new StemmerPort instance when called.
func RegisterStemmerFactory(language string, factory func() (StemmerPort, error)) {
	linguistics_domain.RegisterStemmerFactory(language, factory)
}

// RegisterPhoneticEncoderFactory registers a phonetic encoder factory for a
// language.
//
// Takes language (string) which specifies the language code for the encoder.
// Takes factory (func) which creates a new phonetic encoder when called.
func RegisterPhoneticEncoderFactory(language string, factory func() (PhoneticEncoderPort, error)) {
	linguistics_domain.RegisterPhoneticEncoderFactory(language, factory)
}

// RegisterStopWordsProviderFactory registers a stop words provider factory.
//
// Takes name (string) which identifies the provider factory.
// Takes factory (func() (StopWordsProviderPort, error)) which creates new
// provider instances.
func RegisterStopWordsProviderFactory(name string, factory func() (StopWordsProviderPort, error)) {
	linguistics_domain.RegisterStopWordsProviderFactory(name, factory)
}

// CreateStemmer creates a stemmer for the given language using the registry.
//
// Takes language (string) which specifies the language for stemming.
//
// Returns StemmerPort which provides stemming operations for the language.
func CreateStemmer(language string) StemmerPort {
	return linguistics_domain.CreateStemmer(language)
}

// CreatePhoneticEncoder creates a phonetic encoder for the given language.
//
// Takes language (string) which specifies the language code for encoding.
//
// Returns PhoneticEncoderPort which provides phonetic encoding for the
// specified language.
func CreatePhoneticEncoder(language string) PhoneticEncoderPort {
	return linguistics_domain.CreatePhoneticEncoder(language)
}

// CreateStopWordsProvider creates a stop words provider with the given name.
//
// Takes name (string) which specifies the stop words provider to create.
//
// Returns StopWordsProviderPort which is the configured stop words provider.
func CreateStopWordsProvider(name string) StopWordsProviderPort {
	return linguistics_domain.CreateStopWordsProvider(name)
}

// Jaro returns the Jaro similarity between two strings.
//
// Takes a (string) which is the first string to compare.
// Takes b (string) which is the second string to compare.
//
// Returns float64 which is the similarity score between 0 and 1.
func Jaro(a, b string) float64 {
	return linguistics_domain.Jaro(a, b)
}

// JaroWinkler returns the Jaro-Winkler similarity between two strings.
//
// Takes a (string) which is the first string to compare.
// Takes b (string) which is the second string to compare.
// Takes boostThreshold (float64) which sets the minimum similarity score
// required before the prefix bonus is applied.
// Takes prefixSize (int) which limits how many prefix characters are
// considered for the bonus.
//
// Returns float64 which is the similarity score between 0.0 and 1.0.
func JaroWinkler(a, b string, boostThreshold float64, prefixSize int) float64 {
	return linguistics_domain.JaroWinkler(a, b, boostThreshold, prefixSize)
}

// FuzzyMatch performs fuzzy string matching with configurable threshold.
//
// Takes text (string) which is the text to search within.
// Takes pattern (string) which is the pattern to match against.
// Takes threshold (float64) which sets the minimum similarity score required.
// Takes caseSensitive (bool) which controls whether matching is case-sensitive.
//
// Returns bool which indicates whether the match meets the threshold.
// Returns float64 which is the similarity score between 0 and 1.
func FuzzyMatch(text, pattern string, threshold float64, caseSensitive bool) (bool, float64) {
	return linguistics_domain.FuzzyMatch(text, pattern, threshold, caseSensitive)
}

// SoundexEncode encodes a word using the Soundex phonetic algorithm.
//
// Takes word (string) which is the word to encode.
//
// Returns string which is the four-character Soundex code.
func SoundexEncode(word string) string {
	return linguistics_domain.SoundexEncode(word)
}

// DefaultStopWords returns the default stop words for a language.
//
// Takes language (string) which specifies the language code.
//
// Returns map[string]bool which contains the stop words as keys.
func DefaultStopWords(language string) map[string]bool {
	return linguistics_domain.DefaultStopWords(language)
}

// NewNoOpStemmer creates a no-op stemmer that returns words unchanged.
//
// Takes language (string) which specifies the language code for the stemmer.
//
// Returns StemmerPort which provides a stemmer that returns words as-is.
func NewNoOpStemmer(language string) StemmerPort {
	return linguistics_domain.NewNoOpStemmer(language)
}

// NewNoOpPhoneticEncoder creates a no-op phonetic encoder.
//
// Takes language (string) which specifies the language for the encoder.
//
// Returns PhoneticEncoderPort which is a no-op encoder that performs no
// phonetic transformations.
func NewNoOpPhoneticEncoder(language string) PhoneticEncoderPort {
	return linguistics_domain.NewNoOpPhoneticEncoder(language)
}

// NewNoOpStopWordsProvider creates a no-op stop words provider.
//
// Returns StopWordsProviderPort which provides no stop words filtering.
func NewNoOpStopWordsProvider() StopWordsProviderPort {
	return linguistics_domain.NewNoOpStopWordsProvider()
}

// RegisteredStemmerFactories returns the list of registered stemmer languages.
//
// Returns []string which contains the names of all available stemmer languages.
func RegisteredStemmerFactories() []string {
	return linguistics_domain.RegisteredStemmerFactories()
}

// RegisteredPhoneticEncoderFactories returns the list of registered phonetic
// encoder languages.
//
// Returns []string which contains the names of all available phonetic encoder
// languages.
func RegisteredPhoneticEncoderFactories() []string {
	return linguistics_domain.RegisteredPhoneticEncoderFactories()
}

// RegisteredStopWordsProviderFactories returns the list of registered stop
// words provider names.
//
// Returns []string which contains the names of all registered providers.
func RegisteredStopWordsProviderFactories() []string {
	return linguistics_domain.RegisteredStopWordsProviderFactories()
}
