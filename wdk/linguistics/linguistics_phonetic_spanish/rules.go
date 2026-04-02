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

package linguistics_phonetic_spanish

const (
	// PhoneticK is the phonetic symbol for the hard C, K, or QU sound.
	PhoneticK = "K"

	// PhoneticS represents the S, Z, or soft C sound (as in "centre").
	PhoneticS = "S"

	// PhoneticJ is the guttural sound made by J, G before E or I, and X.
	PhoneticJ = "J"

	// PhoneticC represents the CH sound in the phonetic alphabet.
	PhoneticC = "C"

	// PhoneticNJ represents the Ñ sound (a palatalized N).
	PhoneticNJ = "NJ"

	// PhoneticB is the letter for the B/V sound, which are the same in Spanish.
	PhoneticB = "B"

	// PhoneticR is the phonetic symbol for the R or RR sound.
	PhoneticR = "R"

	// PhoneticG is the letter G pronounced with a hard sound.
	PhoneticG = "G"

	// PhoneticF is the phonetic alphabet letter for the F sound.
	PhoneticF = "F"

	// DigraphLength is the length of a digraph (a two-character sequence).
	DigraphLength = 2

	// TrigraphLength is the length of a three-character sequence.
	TrigraphLength = 3
)

// isSpanishVowel returns true if the character is a Spanish vowel.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if character is a Spanish vowel (A, E, I, O, U).
func isSpanishVowel(character byte) bool {
	switch character {
	case 'A', 'E', 'I', 'O', 'U':
		return true
	default:
		return false
	}
}

// isSpanishConsonant returns true if the character is a Spanish consonant.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if the character is a Spanish consonant.
func isSpanishConsonant(character byte) bool {
	if character < 'A' || character > 'Z' {
		return false
	}
	return !isSpanishVowel(character)
}

// isSoftVowel reports whether the vowel causes softening of C and G.
//
// Takes character (byte) which is the vowel character to check.
//
// Returns bool which is true if the character is E or I.
func isSoftVowel(character byte) bool {
	return character == 'E' || character == 'I'
}

// hasPrefix checks if word starting at position has the given prefix.
//
// Takes word (string) which is the string to check within.
// Takes position (int) which is the starting position in word.
// Takes prefix (string) which is the prefix to look for.
//
// Returns bool which is true if word contains prefix at the given position.
func hasPrefix(word string, position int, prefix string) bool {
	if position+len(prefix) > len(word) {
		return false
	}
	for i := range len(prefix) {
		if word[position+i] != prefix[i] {
			return false
		}
	}
	return true
}

// isWordEnd returns true if pos is at or near the end of the word.
//
// Takes word (string) which is the word to check against.
// Takes position (int) which is the position to test.
//
// Returns bool which is true if pos is at or past the last character.
func isWordEnd(word string, position int) bool {
	return position >= len(word)-1
}
