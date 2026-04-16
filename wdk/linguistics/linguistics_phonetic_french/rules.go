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

package linguistics_phonetic_french

const (
	// PhoneticNasalAN represents the nasal vowel sound in AN/AM/EN/EM.
	PhoneticNasalAN = "A"

	// PhoneticNasalIN represents the nasal vowel sound in IN/IM/AIN/EIN.
	PhoneticNasalIN = "E"

	// PhoneticNasalON represents the nasal vowel sound in ON/OM.
	PhoneticNasalON = "O"

	// PhoneticNasalUN represents the nasal vowel sound in UN/UM.
	PhoneticNasalUN = "U"

	// PhoneticX represents the CH sound, as in "shop" or "chef".
	PhoneticX = "X"

	// PhoneticK represents the hard C, K, or QU sound.
	PhoneticK = "K"

	// PhoneticS represents the S, SS, or C (before E or I) sound.
	PhoneticS = "S"

	// PhoneticZ represents the Z or S sound that occurs between vowels.
	PhoneticZ = "Z"

	// PhoneticJ is the J or G sound used before the letters E or I.
	PhoneticJ = "J"

	// PhoneticN represents the GN sound, as in the Spanish letter n-tilde.
	PhoneticN = "N"

	// PhoneticWA represents the OI vowel combination.
	PhoneticWA = "WA"

	// PhoneticU is the phonetic letter for the OU vowel sound.
	PhoneticU = "U"

	// PhoneticO represents the AU/EAU vowel sound combination.
	PhoneticO = "O"

	// PhoneticEU represents the EU/OEU (OE ligature + U) vowel sound.
	PhoneticEU = "E"

	// DigraphLength is the length of a digraph, a two-character sequence.
	DigraphLength = 2

	// TrigraphLength is the length of a three-character sequence.
	TrigraphLength = 3

	// QuadgraphLength is the length of a quadgraph in characters.
	QuadgraphLength = 4

	// minWordLengthForEntRemoval is the minimum word length required before
	// removing -ENT suffix. Words must be longer than this to avoid removing the
	// suffix from roots like "VENT", "DENT".
	minWordLengthForEntRemoval = 4

	// entSuffixLength is the length of the "-ent" verb ending suffix.
	entSuffixLength = 3

	// minWordLengthForSilentConsonant is the minimum word length required for
	// silent consonant removal.
	minWordLengthForSilentConsonant = 2
)

// isFrenchVowel returns true if the character is a French vowel.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if character is one of A, E, I, O, U, or Y.
func isFrenchVowel(character byte) bool {
	switch character {
	case 'A', 'E', 'I', 'O', 'U', 'Y':
		return true
	default:
		return false
	}
}

// isFrenchConsonant reports whether the character is a French consonant.
//
// Takes character (byte) which is the uppercase character to check.
//
// Returns bool which is true if character is an uppercase French consonant.
func isFrenchConsonant(character byte) bool {
	if character < 'A' || character > 'Z' {
		return false
	}
	return !isFrenchVowel(character)
}

// isNasalFollower reports whether the character at the given position can
// follow a nasal vowel. In French, M and N before a consonant or at word end
// create nasal vowels.
//
// Takes word (string) which is the word to check.
// Takes position (int) which is the position of the character in the word.
//
// Returns bool which is true if the character can follow a nasal vowel.
func isNasalFollower(word string, position int) bool {
	if position >= len(word)-1 {
		return true
	}

	next := word[position+1]
	if isFrenchConsonant(next) && next != 'M' && next != 'N' {
		return true
	}

	return false
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

// hasSuffix checks if word ends with the given suffix.
//
// Takes word (string) which is the text to examine.
// Takes suffix (string) which is the ending to look for.
//
// Returns bool which is true if word ends with suffix.
func hasSuffix(word, suffix string) bool {
	if len(suffix) > len(word) {
		return false
	}
	return word[len(word)-len(suffix):] == suffix
}

// removeSilentEndings strips common silent endings from French words.
//
// Takes word (string) which is the French word to process.
//
// Returns string which is the word with silent endings removed.
func removeSilentEndings(word string) string {
	if len(word) > minWordLengthForEntRemoval && hasSuffix(word, "ENT") {
		return word[:len(word)-entSuffixLength]
	}

	if len(word) > minWordLengthForSilentConsonant {
		lastChar := word[len(word)-1]
		switch lastChar {
		case 'S', 'T', 'D', 'X', 'Z':
			return word[:len(word)-1]
		}
	}

	return word
}
