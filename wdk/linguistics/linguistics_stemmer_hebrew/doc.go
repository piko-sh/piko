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

// Package linguistics_stemmer_hebrew provides Hebrew word stemming
// using a rule-based pipeline backed by a small irregular-form map.
//
// The stemmer applies the following stages in order:
//
//  1. strip nikkud (cantillation marks and vowel points)
//  2. fold final-form letters to their regular-form equivalents
//  3. consult an irregular-form map for attested Hebrew surface
//     forms that resist productive affix stripping; on a hit the
//     stored base form is returned immediately
//  4. iterate prefix, suffix, and combined prefix-plus-suffix
//     stripping to a fixed point, selecting the shortest valid
//     candidate at each pass
//
// The irregulars map carries roughly 140 entries covering broken
// plurals, suppletive verb forms, and weak-root inflections that
// cannot be recovered by affix rules alone. Each entry is an
// independently attested Hebrew linguistic fact.
//
// Hebrew morphology is both concatenative (prefixed particles,
// plural and possessive suffixes) and non-concatenative (root-and-
// pattern verb paradigms). This stemmer handles the concatenative
// portion well and leaves non-concatenative forms untouched; it is
// therefore suitable for search indexing and cache-key normalisation
// where symmetric tokenisation matters more than complete
// morphological analysis.
//
// Verb conjugation prefixes (alef, yod, nun, tav) are deliberately
// omitted from the prefix table to avoid destroying common roots
// that begin with those letters. Single-character prefix stripping
// is additionally suppressed in two cases: when the resulting
// three-rune stem is bounded by vowel letters on both ends, and
// when the prefix is one of the content clitics (bet, kaf, lamed,
// mem) and the stem begins with a vowel letter. Both shapes
// typically mark internally-vocalised verb forms rather than true
// clitic usage.
//
// It self-registers via an init function so that a blank import is
// sufficient to make Hebrew stemming available through the registry.
//
// [Stemmer] is stateless and safe for concurrent use.
package linguistics_stemmer_hebrew
