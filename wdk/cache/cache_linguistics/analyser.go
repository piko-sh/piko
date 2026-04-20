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

package cache_linguistics

import (
	"runtime"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// defaultPoolSize is the number of pre-allocated analysers in the pool. It
// matches the number of CPUs to allow one analyser per goroutine without
// contention.
var defaultPoolSize = runtime.NumCPU()

// NewTextAnalyser creates a [cache_dto.TextAnalyseFunc] from a linguistics
// configuration. The returned function uses an internal analyser pool for
// concurrent safety and low allocation overhead.
//
// In [linguistics_domain.AnalysisModeSmart] mode, tokens are stemmed to their
// root form (e.g. "running" -> "run"). In other modes, normalised tokens are
// returned (lowercase, diacritics removed, stop words filtered).
//
// Takes config (linguistics_domain.AnalyserConfig) which specifies language,
// mode, stop words, and token length limits.
// Takes opts (...linguistics_domain.Option) which configure custom stemmers,
// phonetic encoders, or stop words providers.
//
// Returns cache_dto.TextAnalyseFunc which is safe for concurrent use.
func NewTextAnalyser(config linguistics_domain.AnalyserConfig, opts ...linguistics_domain.Option) cache_dto.TextAnalyseFunc {
	pool := linguistics_domain.NewAnalyserPool(config, defaultPoolSize, opts...)

	return func(text string) []string {
		analyser := pool.Get()
		defer pool.Put(analyser)

		if config.Mode == linguistics_domain.AnalysisModeSmart {
			return analyser.AnalyseToStemmed(text)
		}
		return analyser.AnalyseToStrings(text)
	}
}

// NewEnglishTextAnalyser creates a text analyser configured for English
// with Smart mode (stemming and phonetic encoding).
//
// This is a convenience function equivalent to:
// NewTextAnalyser(smartConfig,
//
//	linguistics_domain.WithLanguage("english"))
//
// Requires importing the English language adapters:
//
//	_ "piko.sh/piko/wdk/linguistics/linguistics_language_english"
//
// Returns cache_dto.TextAnalyseFunc which is safe for concurrent
// use.
func NewEnglishTextAnalyser() cache_dto.TextAnalyseFunc {
	config := linguistics_domain.DefaultConfigForLanguage(linguistics_domain.LanguageEnglish)
	config.Mode = linguistics_domain.AnalysisModeSmart

	return NewTextAnalyser(config, linguistics_domain.WithLanguage(linguistics_domain.LanguageEnglish))
}

// NewTextAnalyserForLanguage creates a text analyser for the
// specified language.
//
// The analyser uses Smart mode with language-specific stemming and
// phonetic encoding. Language adapters must be imported for these
// features to work:
//
//	_ "piko.sh/piko/wdk/linguistics/linguistics_language_french"
//
// Takes language (string) which is the language code (e.g.
// "english", "french", "german", "spanish", "dutch", "russian",
// "swedish", "norwegian", "hungarian", "hebrew").
//
// Returns cache_dto.TextAnalyseFunc which is safe for concurrent
// use.
func NewTextAnalyserForLanguage(language string) cache_dto.TextAnalyseFunc {
	config := linguistics_domain.DefaultConfigForLanguage(language)
	config.Mode = linguistics_domain.AnalysisModeSmart

	return NewTextAnalyser(config, linguistics_domain.WithLanguage(language))
}
