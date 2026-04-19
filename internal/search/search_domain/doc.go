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

// Package search_domain defines the core search abstractions and business logic.
//
// It defines port interfaces for full-text search and implements index
// building, BM25 ranking, and query processing. It supports both Fast mode
// (basic normalisation) and Smart mode (stemming, phonetic matching, and
// fuzzy search).
//
// # Search modes
//
// The package supports two search modes:
//
// Fast Mode:
//   - Basic text normalisation (lowercase, tokenisation)
//   - Exact term matching with prefix expansion
//   - Minimal memory and CPU overhead
//
// Smart Mode:
//   - Snowball stemming for linguistic normalisation
//   - Phonetic matching for sound-alike terms
//   - Jaro-Winkler fuzzy matching for typo tolerance
//   - Multi-language support (English, Spanish, French, Russian, etc.)
//
// # Integration
//
// Integrates with:
//
//   - linguistics: Text analysis, stemming, and phonetic encoding
//   - search_schema: FlatBuffer schema for zero-copy index serialisation
//   - search_dto: Data transfer objects for search configuration
//   - collection_dto: Content items to be indexed
//
// # Thread safety
//
// IndexBuilder, QueryProcessor, and BM25Scorer instances are safe for
// concurrent use after initialisation. The IndexReaderPort implementations
// provide thread-safe read access to FlatBuffer indexes.
package search_domain
