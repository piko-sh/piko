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

// Analyser combines tokenisation, normalisation, stemming, and phonetic encoding
// into a single pipeline for text analysis.
//
// Fields use interfaces to enable dependency injection and test mocking.
type Analyser struct {
	// tokeniser splits text into tokens for analysis.
	tokeniser TokeniserPort

	// stemmer finds the root form of words for better matching.
	stemmer StemmerPort

	// phonetic produces sound-based codes from normalised tokens.
	phonetic PhoneticEncoderPort

	// config holds settings that control how the analyser processes text.
	config AnalyserConfig
}

// NewAnalyser creates a new text analyser with the given settings. Use options
// to configure custom implementations.
//
// Without options, stemming is a no-op (words pass through unchanged). This
// allows the core linguistics package to have no hard dependencies on external
// stemming libraries.
//
// Takes config (AnalyserConfig) which specifies the language and analysis
// settings.
// Takes opts (...Option) which are optional configuration functions.
//
// Returns *Analyser which is ready to tokenise, stem, and encode text.
func NewAnalyser(config AnalyserConfig, opts ...Option) *Analyser {
	language := ValidateLanguage(config.Language)
	if language != config.Language {
		config.Language = language
	}

	options := &analyserOptions{}
	for _, opt := range opts {
		opt(options)
	}

	tokeniser := options.tokeniser
	if tokeniser == nil {
		tokeniser = NewTokeniser(config)
	}

	stemmer := options.stemmer
	if stemmer == nil {
		stemmer = NewNoOpStemmer(config.Language)
	}

	phonetic := options.phonetic
	if phonetic == nil {
		phonetic = NewPhoneticEncoder(DefaultPhoneticCodeLength)
	}

	return NewAnalyserWithDeps(tokeniser, stemmer, phonetic, config)
}

// NewAnalyserWithDeps creates a new analyser with injected dependencies.
// This constructor enables dependency injection for testing and custom
// implementations.
//
// Use this in tests to inject mocks:
// analyser := NewAnalyserWithDeps(
//
//	mockTokeniser,
//	mockStemmer,
//	mockPhonetic,
//	config,
//
// )
// For production code, use NewAnalyser which uses real implementations.
//
// Takes tokeniser (TokeniserPort) which splits text into tokens.
// Takes stemmer (StemmerPort) which reduces words to their root form.
// Takes phonetic (PhoneticEncoderPort) which encodes words phonetically.
// Takes config (AnalyserConfig) which specifies analyser behaviour.
//
// Returns *Analyser which is ready for text analysis.
func NewAnalyserWithDeps(
	tokeniser TokeniserPort,
	stemmer StemmerPort,
	phonetic PhoneticEncoderPort,
	config AnalyserConfig,
) *Analyser {
	return &Analyser{
		tokeniser: tokeniser,
		stemmer:   stemmer,
		phonetic:  phonetic,
		config:    config,
	}
}

// Analyse performs complete text analysis based on the configured mode.
//
// Takes text (string) which is the input to analyse.
//
// Returns []Token which contains tokens with their normalised, stemmed, and/or
// phonetic forms populated depending on the analysis mode.
func (a *Analyser) Analyse(text string) []Token {
	tokens := a.tokeniser.Tokenise(text)

	if a.config.Mode == AnalysisModeSmart {
		for i := range tokens {
			tokens[i].Stemmed = a.stemmer.Stem(tokens[i].Normalised)

			tokens[i].Phonetic = a.phonetic.Encode(tokens[i].Normalised)
		}
	}

	return tokens
}

// AnalyseToStrings returns the normalised token strings for quick processing.
//
// Takes text (string) which is the input to analyse.
//
// Returns []string which contains the normalised form of each token.
func (a *Analyser) AnalyseToStrings(text string) []string {
	tokens := a.Analyse(text)
	result := make([]string, len(tokens))
	for i, token := range tokens {
		result[i] = token.Normalised
	}
	return result
}

// AnalyseToStemmed returns the stemmed forms of tokens (Smart mode only).
// Falls back to normalised tokens if not in Smart mode.
//
// Takes text (string) which is the input text to analyse.
//
// Returns []string which contains the stemmed form of each token.
func (a *Analyser) AnalyseToStemmed(text string) []string {
	if a.config.Mode != AnalysisModeSmart {
		return a.AnalyseToStrings(text)
	}

	tokens := a.Analyse(text)
	result := make([]string, len(tokens))
	for i, token := range tokens {
		result[i] = token.Stemmed
	}
	return result
}

// AnalyseToPhonetic returns the phonetic codes of tokens (Smart mode only).
//
// Takes text (string) which is the input to analyse for phonetic codes.
//
// Returns []string which contains the phonetic codes, or nil if not in Smart
// mode.
func (a *Analyser) AnalyseToPhonetic(text string) []string {
	if a.config.Mode != AnalysisModeSmart {
		return nil
	}

	tokens := a.Analyse(text)
	result := make([]string, len(tokens))
	for i, token := range tokens {
		result[i] = token.Phonetic
	}
	return result
}

// GetStemmer returns the underlying stemmer for direct use.
//
// Returns StemmerPort which provides the stemmer instance.
func (a *Analyser) GetStemmer() StemmerPort {
	return a.stemmer
}

// GetPhoneticEncoder returns the underlying phonetic encoder for direct use.
//
// Returns PhoneticEncoderPort which provides phonetic encoding operations.
func (a *Analyser) GetPhoneticEncoder() PhoneticEncoderPort {
	return a.phonetic
}

// GetMode returns the current analysis mode.
//
// Returns AnalysisMode which indicates how the analyser processes input.
func (a *Analyser) GetMode() AnalysisMode {
	return a.config.Mode
}

// AnalyserPool provides a pool of analysers for concurrent use when
// processing many documents in parallel during indexing.
//
// Fields ordered for optimal memory alignment.
type AnalyserPool struct {
	// opts holds options for creating new analysers when the pool is empty.
	opts []Option

	// analysers is a buffered channel that holds pooled Analyser instances.
	analysers chan *Analyser

	// config holds settings for creating new analysers when the pool is empty.
	config AnalyserConfig
}

// NewAnalyserPool creates a pool of analysers with the given configuration.
//
// Takes config (AnalyserConfig) which specifies the analyser settings.
// Takes poolSize (int) which sets the number of analysers in the pool.
// If poolSize is zero or negative, it defaults to one.
// Takes opts (...Option) which are optional configuration functions passed to
// each analyser in the pool.
//
// Returns *AnalyserPool which is a ready-to-use pool of analysers.
func NewAnalyserPool(config AnalyserConfig, poolSize int, opts ...Option) *AnalyserPool {
	if poolSize <= 0 {
		poolSize = 1
	}

	pool := &AnalyserPool{
		config:    config,
		opts:      opts,
		analysers: make(chan *Analyser, poolSize),
	}

	for range poolSize {
		pool.analysers <- NewAnalyser(config, opts...)
	}

	return pool
}

// Get retrieves an analyser from the pool.
//
// Returns *Analyser which is a ready-to-use analyser. If the pool is empty,
// a new analyser is created using the pool's configuration and options.
func (p *AnalyserPool) Get() *Analyser {
	select {
	case analyser := <-p.analysers:
		return analyser
	default:
		return NewAnalyser(p.config, p.opts...)
	}
}

// Put returns an analyser to the pool for reuse.
//
// Takes analyser (*Analyser) which is the analyser to return to the pool.
func (p *AnalyserPool) Put(analyser *Analyser) {
	select {
	case p.analysers <- analyser:
	default:
	}
}
