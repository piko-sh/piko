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

// Package linguistics provides text processing utilities for search and
// natural language processing.
//
// The analysis pipeline covers tokenisation, normalisation, stemming,
// phonetic encoding, stop word filtering, string similarity metrics,
// and fuzzy matching. It supports multiple languages and is designed
// for use with Piko's zero-copy search system.
//
// Stemmers, phonetic encoders, and stop words providers use a factory
// registry pattern. Language-specific implementations live in separate
// sub-packages and self-register via init functions, so a blank import
// is enough to make a language available:
//
//	import _ "piko.sh/piko/wdk/linguistics/linguistics_language_english"
//
//	stemmer := linguistics.CreateStemmer("english")
//	analyser := linguistics.NewAnalyser(config,
//	    linguistics.WithStemmer(stemmer),
//	)
//
// For multi-language applications, create a separate [Analyser] per
// language.
//
// # Thread safety
//
// [Normaliser] is safe for concurrent use. [Analyser], [Tokeniser],
// [Stemmer], and [PhoneticEncoder] instances should not be shared
// between goroutines.
package linguistics
