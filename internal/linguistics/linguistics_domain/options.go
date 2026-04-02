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

// Option sets up an Analyser during creation.
// Use the With* functions to create options.
type Option func(*analyserOptions)

// analyserOptions holds optional configuration for Analyser construction.
// Fields that are nil after applying options will use default implementations.
type analyserOptions struct {
	// stemmer is a custom stemmer to use. If nil, NoOpStemmer is used.
	stemmer StemmerPort

	// tokeniser is a custom tokeniser to use. If nil, the default tokeniser is used.
	tokeniser TokeniserPort

	// phonetic is a custom phonetic encoder to use. If nil, the default
	// PhoneticEncoder is used.
	phonetic PhoneticEncoderPort

	// stopWordsProvider is a custom stop words provider.
	// If nil, the config's stop words are used directly.
	stopWordsProvider StopWordsProviderPort
}

// WithStemmer sets a custom stemmer for word processing.
// If not called, a NoOpStemmer is used that returns words unchanged.
//
// Takes stemmer (StemmerPort) which is the stemmer to use.
//
// Returns Option which sets up the analyser to use the given stemmer.
func WithStemmer(stemmer StemmerPort) Option {
	return func(o *analyserOptions) {
		o.stemmer = stemmer
	}
}

// WithTokeniser sets a custom tokeniser for the analyser.
// If not called, the default Tokeniser is used.
//
// Takes tokeniser (TokeniserPort) which is the tokeniser to use.
//
// Returns Option which configures the analyser to use the given tokeniser.
func WithTokeniser(tokeniser TokeniserPort) Option {
	return func(o *analyserOptions) {
		o.tokeniser = tokeniser
	}
}

// WithPhoneticEncoder sets a custom phonetic encoder for the analyser.
// If not called, the default PhoneticEncoder is used.
//
// Takes encoder (PhoneticEncoderPort) which is the encoder to use.
//
// Returns Option which sets the analyser to use the given encoder.
func WithPhoneticEncoder(encoder PhoneticEncoderPort) Option {
	return func(o *analyserOptions) {
		o.phonetic = encoder
	}
}

// WithStopWordsProvider sets a custom stop words provider.
// When set, the analyser will use this provider to get stop words for the
// configured language instead of using the stop words from the config.
//
// Takes provider (StopWordsProviderPort) which is the provider to use.
//
// Returns Option which configures the analyser to use the given stop words
// provider.
func WithStopWordsProvider(provider StopWordsProviderPort) Option {
	return func(o *analyserOptions) {
		o.stopWordsProvider = provider
	}
}

// WithLanguage configures the analyser with stemmer, phonetic encoder,
// and stop words provider for the specified language.
//
// This is a convenience function that looks up all three components from
// their respective registries. If a component is not registered for the
// language, a no-op implementation is used.
//
// Takes language (string) which is the language code (e.g., "english",
// "french").
//
// Returns Option which configures the analyser with all
// language-specific components.
func WithLanguage(language string) Option {
	return func(o *analyserOptions) {
		o.stemmer = CreateStemmer(language)

		o.phonetic = CreatePhoneticEncoder(language)

		o.stopWordsProvider = CreateStopWordsProvider(language)
	}
}
