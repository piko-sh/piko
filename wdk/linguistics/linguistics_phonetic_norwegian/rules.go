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

package linguistics_phonetic_norwegian

const (
	// PhoneticX represents the KJ/SJ/SKJ sound (voiceless palatal fricative).
	PhoneticX = "X"

	// PhoneticK represents the hard K sound.
	PhoneticK = "K"

	// PhoneticS represents the S sound.
	PhoneticS = "S"

	// PhoneticJ is the letter J in the phonetic alphabet.
	PhoneticJ = "J"

	// PhoneticF is the phonetic alphabet code for the letter F.
	PhoneticF = "F"

	// PhoneticNG is the phonetic symbol for the NG nasal sound.
	PhoneticNG = "NG"

	// PhoneticAI represents the EI diphthong sound as in "day" or "pain".
	PhoneticAI = "AI"

	// PhoneticAU is the AU diphthong sound, as in "house" or "now".
	PhoneticAU = "AU"

	// PhoneticOI represents the OI diphthong sound, as in "coin" or "boy".
	PhoneticOI = "OI"

	// DigraphLength is the length of a digraph, a two-character sequence.
	DigraphLength = 2

	// TrigraphLength is the length of a three-character sequence.
	TrigraphLength = 3
)

// hasPrefix checks if word starting at position has the given prefix.
//
// Takes word (string) which is the string to search within.
// Takes position (int) which is the starting position in word.
// Takes prefix (string) which is the prefix to look for.
//
// Returns bool which is true if the prefix matches at the given position.
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

// isSoftVowel reports whether the vowel causes softening of a preceding
// consonant.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if character is a soft vowel (I, Y, or E).
func isSoftVowel(character byte) bool {
	return character == 'I' || character == 'Y' || character == 'E'
}
