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

import (
	"strings"
	"unicode/utf8"
)

// stripNikkud removes Hebrew cantillation marks and vowel points from
// the input. Modern Hebrew text usually omits these but the stemmer
// normalises defensively so that pointed input produces the same stem
// as unpointed input.
//
// Takes word (string) which may contain Hebrew diacritical marks.
//
// Returns string with all nikkud removed.
func stripNikkud(word string) string {
	if word == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(word))
	for _, character := range word {
		if isNikkud(character) {
			continue
		}
		_, _ = builder.WriteRune(character)
	}
	return builder.String()
}

// normaliseFinalForms folds Hebrew final-form letters to their
// regular-form equivalents.
//
// The five Hebrew letters kaf, mem, nun, pe, and tsadi each take a
// distinct glyph when they appear at the end of a word. Folding
// them to the regular form lets mid-word and end-of-word
// occurrences of the same consonant compare equal.
//
// Takes word (string) which may contain final-form letters.
//
// Returns string which is the input with all final-form letters
// replaced by their regular-form equivalents.
func normaliseFinalForms(word string) string {
	if word == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(word))
	for _, character := range word {
		_, _ = builder.WriteRune(foldFinalForm(character))
	}
	return builder.String()
}

// foldFinalForm returns the regular form of a Hebrew final-form
// letter, or the input unchanged when the rune is not a final form.
//
// Takes character (rune) which is the code point to fold.
//
// Returns rune which is the folded value.
func foldFinalForm(character rune) rune {
	switch character {
	case '\u05DA':
		return '\u05DB'
	case '\u05DD':
		return '\u05DE'
	case '\u05DF':
		return '\u05E0'
	case '\u05E3':
		return '\u05E4'
	case '\u05E5':
		return '\u05E6'
	default:
		return character
	}
}

// isNikkud reports whether a rune is a Hebrew vowel point or
// cantillation mark. The maqaf at U+05BE is excluded because it is
// punctuation rather than a diacritic.
//
// Takes character (rune) which is the code point to classify.
//
// Returns bool which is true when the rune is a nikkud mark.
func isNikkud(character rune) bool {
	if character >= nikkudLowerStart && character <= nikkudLowerEnd {
		return true
	}
	if character >= nikkudUpperStart && character <= nikkudUpperEnd {
		return true
	}
	return false
}

// tryPrefixStrip removes the longest applicable prefix from a word.
//
// The prefix table is maintained in longest-first order so the
// first match is also the longest. The match respects
// minWordForPrefix and minStemLength so that short words and overly
// aggressive strips are rejected. Single-character prefix strips
// that would leave a three-rune stem flagged by unsafeSingleCharStrip
// are skipped because such stems are typically internally-vocalised
// verb conjugations rather than valid roots.
//
// Takes word (string) which is the Hebrew word to examine.
//
// Returns string which is the word with the prefix removed, or the
// original when no prefix applies.
// Returns bool which is true when a prefix was stripped.
func tryPrefixStrip(word string) (string, bool) {
	if utf8.RuneCountInString(word) < minWordForPrefix {
		return word, false
	}

	for _, prefix := range prefixes {
		if !strings.HasPrefix(word, prefix) {
			continue
		}
		remainder := word[len(prefix):]
		if utf8.RuneCountInString(remainder) < minStemLength {
			continue
		}
		if unsafeSingleCharStrip(prefix, remainder) {
			continue
		}
		return remainder, true
	}
	return word, false
}

// unsafeSingleCharStrip reports whether a single-character prefix
// strip should be skipped to protect a conjugated verb form.
//
// The heuristic rejects two shapes: a three-rune remainder bounded
// by Hebrew vowel letters on both ends, and a three-rune remainder
// whose first rune is a vowel letter when the prefix is one of the
// content clitics (bet, kaf, lamed, mem). Both patterns are typical
// of pa'al past plural forms that would otherwise be mis-stripped.
//
// Takes prefix (string) which is the candidate prefix to strip.
// Takes remainder (string) which is the stem that would result.
//
// Returns bool which is true when the strip should be skipped.
func unsafeSingleCharStrip(prefix, remainder string) bool {
	if utf8.RuneCountInString(prefix) != 1 {
		return false
	}
	if utf8.RuneCountInString(remainder) != minStemLength {
		return false
	}
	runes := []rune(remainder)
	first := runes[0]
	last := runes[len(runes)-1]
	if isHebrewVowelLetter(first) && isHebrewVowelLetter(last) {
		return true
	}
	if isHebrewVowelLetter(first) && isContentPrefix(prefix) {
		return true
	}
	return false
}

// isContentPrefix reports whether a prefix particle is one of the
// four content clitics whose mis-stripping tends to destroy verb
// forms. The definite article and coordinator are excluded because
// they appear more often as true clitics on nouns.
//
// Takes prefix (string) which is the candidate prefix.
//
// Returns bool which is true for bet, kaf, lamed, and mem.
func isContentPrefix(prefix string) bool {
	switch prefix {
	case "\u05D1", "\u05DB", "\u05DC", "\u05DE":
		return true
	default:
		return false
	}
}

// isHebrewVowelLetter reports whether a rune is one of the four
// Hebrew letters most often used as matres lectionis.
//
// Takes character (rune) which is the code point to classify.
//
// Returns bool which is true for alef, he, vav, and yod.
func isHebrewVowelLetter(character rune) bool {
	switch character {
	case '\u05D0', '\u05D4', '\u05D5', '\u05D9':
		return true
	default:
		return false
	}
}

// trySuffixStrip removes the longest applicable suffix from a word.
//
// The suffix table is maintained in longest-first order so the
// first match is also the longest. Length guards mirror those in
// tryPrefixStrip.
//
// Takes word (string) which is the Hebrew word to examine.
//
// Returns string which is the word with the suffix removed, or the
// original when no suffix applies.
// Returns bool which is true when a suffix was stripped.
func trySuffixStrip(word string) (string, bool) {
	if utf8.RuneCountInString(word) < minWordForSuffix {
		return word, false
	}

	for _, suffix := range suffixes {
		if !strings.HasSuffix(word, suffix) {
			continue
		}
		remainder := word[:len(word)-len(suffix)]
		if utf8.RuneCountInString(remainder) < minStemLength {
			continue
		}
		return remainder, true
	}
	return word, false
}

// stem runs the Hebrew stemming pipeline to a fixed point.
//
// The input is stripped of nikkud and folded to regular letter
// forms, then iterated through the irregulars lookup and the
// prefix-suffix stripping pass until no further shortening is
// possible or maxStemIterations is reached. The shortest candidate
// at each pass wins; when candidates tie on length suffix-only is
// preferred over prefix-only, and either is preferred over the
// unchanged input.
//
// Takes word (string) which is the Hebrew word to stem.
//
// Returns string which is the stem.
func stem(word string) string {
	cleaned := normaliseFinalForms(stripNikkud(word))
	if cleaned == "" {
		return cleaned
	}

	current := cleaned
	for range maxStemIterations {
		if base, ok := lookupIrregular(current); ok {
			return base
		}
		next := stemOnePass(current)
		if next == current {
			return current
		}
		current = next
	}
	return current
}

// stemOnePass runs the prefix and suffix strippers once over the
// input and returns the shortest valid candidate. When no strategy
// produces a shorter form, the input is returned unchanged so that
// the caller can detect a fixed point.
//
// Takes word (string) which is the nikkud-stripped and final-form-
// folded surface form.
//
// Returns string which is the shortest candidate stem for this pass.
func stemOnePass(word string) string {
	suffixResult, suffixOK := trySuffixStrip(word)
	prefixResult, prefixOK := tryPrefixStrip(word)

	best := word
	bestLength := utf8.RuneCountInString(word)
	consider := func(candidate string) {
		candidateLength := utf8.RuneCountInString(candidate)
		if candidateLength < bestLength {
			best = candidate
			bestLength = candidateLength
		}
	}

	if suffixOK {
		consider(suffixResult)
	}
	if prefixOK {
		consider(prefixResult)
	}
	if suffixOK && prefixOK {
		if combined, combinedOK := tryPrefixStrip(suffixResult); combinedOK {
			consider(combined)
		}
	}
	return best
}
