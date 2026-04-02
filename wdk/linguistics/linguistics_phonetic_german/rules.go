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

package linguistics_phonetic_german

const (
	// CologneVowel represents the digit for vowels (A, E, I, J, O, U, Y).
	CologneVowel = '0'

	// CologneBP represents the digit for B and P (not before H).
	CologneBP = '1'

	// CologneDT is the digit for D and T when not before C, S, or Z.
	CologneDT = '2'

	// CologneFVW represents the digit for F, V, W, and P before H.
	CologneFVW = '3'

	// CologneGKQ represents the digit for G, K, Q, and C in certain contexts.
	CologneGKQ = '4'

	// CologneL is the Cologne phonetic code digit for the letter L.
	CologneL = '5'

	// CologneMN is the digit that represents the M and N sounds.
	CologneMN = '6'

	// CologneR is the Cologne phonetic code digit for the letter R.
	CologneR = '7'

	// CologneSZ represents the digit for S, Z, and C/D/T in certain contexts.
	CologneSZ = '8'

	// CologneSkip indicates the character should be skipped (H).
	CologneSkip = byte(0)

	// latinAlphabetSize is the number of letters in the Latin alphabet.
	latinAlphabetSize = 26
)

// isFollowingChar checks if the character at position+1 matches any in the chars
// string.
//
// Takes word (string) which is the string to check within.
// Takes position (int) which is the position before the character to check.
// Takes chars (string) which contains the characters to match against.
//
// Returns bool which is true if the next character matches any in chars.
func isFollowingChar(word string, position int, chars string) bool {
	if position+1 >= len(word) {
		return false
	}
	next := word[position+1]
	for i := range len(chars) {
		if next == chars[i] {
			return true
		}
	}
	return false
}

// isPrecedingChar checks if the character at position-1 matches any character in
// the chars string.
//
// Takes word (string) which is the string to check within.
// Takes position (int) which is the position whose preceding character to check.
// Takes chars (string) which contains the characters to match against.
//
// Returns bool which is true if the preceding character matches any in chars,
// or false if position is 0 or no match is found.
func isPrecedingChar(word string, position int, chars string) bool {
	if position == 0 {
		return false
	}
	previous := word[position-1]
	for i := range len(chars) {
		if previous == chars[i] {
			return true
		}
	}
	return false
}

// isCHardContext reports whether C should encode as 4 (hard sound) at the
// given position. C encodes as 4 before A, H, K, O, Q, U, X, or before L and R
// at word onset only.
//
// Takes word (string) which is the word being analysed.
// Takes position (int) which is the position of C in the word.
//
// Returns bool which is true if C has a hard sound context.
func isCHardContext(word string, position int) bool {
	if position+1 >= len(word) {
		return false
	}
	next := word[position+1]
	switch next {
	case 'A', 'H', 'K', 'O', 'Q', 'U', 'X':
		return true
	case 'L', 'R':
		return position == 0
	default:
		return false
	}
}

// removeDuplicates collapses consecutive duplicate digits in the code.
//
// Takes code ([]byte) which contains the digit sequence to process.
//
// Returns []byte which contains the code with consecutive duplicates removed.
func removeDuplicates(code []byte) []byte {
	if len(code) <= 1 {
		return code
	}
	result := make([]byte, 0, len(code))
	result = append(result, code[0])
	for i := 1; i < len(code); i++ {
		if code[i] != code[i-1] {
			result = append(result, code[i])
		}
	}
	return result
}

// removeInternalZeros removes all '0' digits except at the beginning of the code.
//
// Takes code ([]byte) which is the Cologne phonetic code to process.
//
// Returns []byte which is the code with internal zeros removed.
func removeInternalZeros(code []byte) []byte {
	if len(code) == 0 {
		return code
	}
	result := make([]byte, 0, len(code))
	start := 0
	if code[0] == CologneVowel {
		result = append(result, CologneVowel)
		start = 1
	}
	for i := start; i < len(code); i++ {
		if code[i] != CologneVowel {
			result = append(result, code[i])
		}
	}
	return result
}
