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

package linguistics_phonetic_swedish

const (
	// PhoneticX represents the SJ/TJ/KJ sound (voiceless velar fricative).
	PhoneticX = "X"

	// PhoneticK represents the hard K sound.
	PhoneticK = "K"

	// PhoneticS is the phonetic alphabet letter for the S sound.
	PhoneticS = "S"

	// PhoneticJ is the letter J in the phonetic alphabet.
	PhoneticJ = "J"

	// PhoneticF is the NATO phonetic alphabet code for the letter F.
	PhoneticF = "F"

	// PhoneticNG is the phonetic symbol for the NG nasal sound.
	PhoneticNG = "NG"

	// PhoneticGN is the GN consonant cluster in phonetic notation.
	PhoneticGN = "GN"

	// DigraphLength is the length of a digraph (a two-character sequence).
	DigraphLength = 2

	// TrigraphLength is the length of a three-character sequence.
	TrigraphLength = 3
)

// isSoftVowel reports whether the given character is a soft vowel that causes
// softening of K and G in Swedish. The soft vowels are E, I, Y, Ae, and Oe, which
// are represented as E, I, and Y after normalisation.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if character is a soft vowel.
func isSoftVowel(character byte) bool {
	return character == 'E' || character == 'I' || character == 'Y'
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
