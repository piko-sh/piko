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

package linguistics_stemmer_hebrew

const (
	// minStemLength is the minimum rune count required for a candidate
	// stem. Hebrew roots are typically three letters so stripping
	// below this length is rejected.
	minStemLength = 3

	// minWordForPrefix is the minimum rune count required before the
	// stemmer attempts prefix stripping. Words shorter than this are
	// almost never prefixed in practice.
	minWordForPrefix = 4

	// minWordForSuffix is the minimum rune count required before the
	// stemmer attempts suffix stripping.
	minWordForSuffix = 5

	// maxStemIterations bounds how many fixed-point iterations the
	// stemmer performs. Each iteration either shortens the word or
	// reaches a stable state, so the loop always terminates within
	// this number of passes.
	maxStemIterations = 8

	// nikkudLowerStart is the first Unicode code point in the Hebrew
	// cantillation and vowel-point block.
	nikkudLowerStart = '\u0591'

	// nikkudLowerEnd is the last cantillation mark before the maqaf
	// punctuation character at U+05BE.
	nikkudLowerEnd = '\u05BD'

	// nikkudUpperStart is the first vowel-point code after the maqaf.
	nikkudUpperStart = '\u05BF'

	// nikkudUpperEnd is the last vowel-point code in the block.
	nikkudUpperEnd = '\u05C7'
)

// prefixes lists Hebrew prefix particles and their combinations.
//
// The entries are ordered longest-first so that greedy matching
// selects the maximal applicable prefix. The seven single-character
// particles are bet, he, vav, kaf, lamed, mem, and shin; the multi-
// character combinations reflect the grammatical stacking rules of
// Modern Hebrew.
var prefixes = []string{
	"וכשה", "ולכש", "וכשמ", "וכשב", "וכשל",
	"לכש", "כשה", "כשמ", "כשב", "כשל",
	"משה", "בשה",
	"וכה", "ולה", "ומה", "ובה", "ושה",
	"וה", "וב", "וכ", "ול", "ומ", "וש",
	"שה", "שב", "שכ", "של", "שמ",
	"מה", "בה", "כה", "לה", "כש", "מש",
	"ה", "ו", "ב", "כ", "ל", "מ", "ש",
}

// suffixes lists Hebrew inflectional suffixes.
//
// The entries are ordered longest-first so that compound possessive
// endings are matched before their two-letter substrings. Entries
// use the regular (non-final) forms of kaf, mem, nun, pe, and
// tsadi because the stemmer folds final forms to regular forms
// before matching.
var suffixes = []string{
	"יותיהמ", "יותיהנ", "יותיכמ", "יותיכנ",
	"יותינו", "יותייכ", "יותיהו", "ותיהמ", "ותיהנ", "ותיכמ", "ותיכנ", "ותינו", "ותייכ",
	"יותיה", "יותיו", "יותיי", "תיהמ", "תיהנ", "תיכמ", "תיכנ", "תינו", "תייכ",
	"ותיה", "ותיו", "ותיי", "ותיכ",
	"יהמ", "יהנ", "יות", "ותי", "ותו", "ותה", "ותכ", "תיי", "תיה", "תיו", "תיכ",
	"יכמ", "יכנ", "ייכ", "ייה",
	"ימ", "ות", "ית", "ני", "נו", "כמ", "כנ", "המ", "הנ", "תמ", "תנ", "תי",
	"יכ", "יה", "יו", "יי", "תה", "תו", "תכ",
	"ת", "ה",
}
