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

// Package linguistics_domain defines the text analysis pipeline for
// Piko's search system, covering tokenisation, normalisation, stemming,
// phonetic encoding, stop words, and fuzzy matching with multi-language
// support and typo tolerance. Port interfaces ([AnalyserPort],
// [TokeniserPort], [NormaliserPort], [StemmerPort],
// [PhoneticEncoderPort], [StopWordsProviderPort]) allow pluggable
// language-specific adapters registered via factory registries.
//
// # Components
//
// The package includes six major components:
//
//   - Tokenisation: Split text into words with position and byte offset tracking
//   - Normalisation: Unicode normalisation, diacritic removal, and case folding
//   - Stemming: Multi-language word stemming via pluggable adapters (9 languages)
//   - Phonetic Encoding: Pluggable encoders including Double Metaphone and French
//   - Stop Words: Language-specific stop word filtering via pluggable providers
//   - String Metrics: Levenshtein distance, Jaro, and Jaro-Winkler similarity
//   - Fuzzy Matching: High-level fuzzy text matching with configurable thresholds
//
// # Adapter registry pattern
//
// Stemmers, phonetic encoders, and stop words providers use a factory registry
// pattern. Import the desired adapters to register them:
//
//	import (
//	    _ "piko.sh/piko/internal/linguistics/linguistics_adapters/driven_stemmer_snowball"
//	    _ "piko.sh/piko/internal/linguistics/linguistics_adapters/driven_phonetic_metaphone"
//	    _ "piko.sh/piko/internal/linguistics/linguistics_adapters/driven_phonetic_french"
//	    _ "piko.sh/piko/internal/linguistics/linguistics_adapters/driven_stopwords_builtin"
//	)
//
// Then create components via the registry:
//
//	stemmer := linguistics.CreateStemmer("snowball", "english")
//	encoder := linguistics.CreatePhoneticEncoder("metaphone", "english")
//	provider := linguistics.CreateStopWordsProvider("builtin")
//
// Or inject them directly:
//
//	analyser := linguistics.NewAnalyser(config,
//	    linguistics.WithStemmer(stemmer),
//	    linguistics.WithPhoneticEncoder(encoder),
//	    linguistics.WithStopWordsProvider(provider),
//	)
//
// # Usage
//
// For most use cases, create an Analyser with a configuration:
//
//	config := linguistics.DefaultConfig()
//	analyser := linguistics.NewAnalyser(config)
//	tokens := analyser.Analyse("The quick brown fox")
//
// For fuzzy matching without full analysis:
//
//	matched, score := linguistics.FuzzyMatch("configuration", "config", 0.5, false)
//
// For string similarity comparisons:
//
//	similarity := linguistics.JaroWinkler("MARTHA", "MARHTA", 0.7, 4)
//
// # Supported languages
//
// The package supports nine languages: English, Spanish, French, German, Dutch,
// Russian, Swedish, Norwegian, and Hungarian. Each language has its own stemmer,
// phonetic rules, and stop words available via the respective adapters.
//
// # Multi-language applications
//
// For applications supporting multiple languages, create separate analysers:
//
//	englishAnalyser := linguistics.NewAnalyser(
//	    linguistics.DefaultConfigForLanguage("english"),
//	    linguistics.WithStemmer(englishStemmer),
//	    linguistics.WithPhoneticEncoder(englishPhonetic),
//	)
//	frenchAnalyser := linguistics.NewAnalyser(
//	    linguistics.DefaultConfigForLanguage("french"),
//	    linguistics.WithStemmer(frenchStemmer),
//	    linguistics.WithPhoneticEncoder(frenchPhonetic),
//	)
//
// # Dependency injection
//
// All major components define port interfaces ([AnalyserPort], [TokeniserPort],
// [NormaliserPort], [StemmerPort], [PhoneticEncoderPort], [StopWordsProviderPort])
// for dependency injection and test mocking. Use the functional options
// in production and the mock adapters in tests:
//
//	analyser := linguistics.NewAnalyser(config,
//	    linguistics.WithStemmer(mockStemmer),
//	    linguistics.WithPhoneticEncoder(mockEncoder),
//	    linguistics.WithStopWordsProvider(mockProvider),
//	)
//
// # Thread safety
//
// [Normaliser] is safe for concurrent use. [Analyser], [Tokeniser], [Stemmer],
// and [PhoneticEncoder] instances should not be shared between goroutines.
// For concurrent text processing across multiple goroutines, create separate
// analyser instances per goroutine or use appropriate synchronisation.
package linguistics_domain
