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

package linguistics_phonetic_hungarian

const (
	// PhoneticC represents the "cs" digraph sound, like "ch" in English.
	PhoneticC = "C"

	// PhoneticDJ represents the GY/DZS sound (palatalized D).
	PhoneticDJ = "DJ"

	// PhoneticJ represents the LY sound, like the English letter "y".
	PhoneticJ = "J"

	// PhoneticNJ represents the NY sound (palatalised N, like Spanish n-tilde).
	PhoneticNJ = "NJ"

	// PhoneticS represents the SZ sound, like the English letter "s".
	PhoneticS = "S"

	// PhoneticZ represents the ZS sound (like French "j" or English "zh").
	PhoneticZ = "Z"

	// PhoneticTJ represents the TY sound (palatalized T).
	PhoneticTJ = "TJ"

	// PhoneticK is the letter used for the hard K sound.
	PhoneticK = "K"

	// PhoneticF is the phonetic alphabet code for the letter F.
	PhoneticF = "F"

	// DigraphLength is the length of a digraph (a two-character sequence).
	DigraphLength = 2

	// TrigraphLength is the length of a three-character sequence.
	TrigraphLength = 3
)

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
