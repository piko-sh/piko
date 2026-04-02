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

package linguistics_phonetic_dutch

const (
	// DigraphLength is the length of a two-letter pattern (e.g., "CH", "IJ").
	DigraphLength = 2

	// TrigraphLength is the length of a three-letter pattern such as "SCH".
	TrigraphLength = 3

	// PhoneticX represents the guttural G or CH sound.
	PhoneticX = "X"

	// PhoneticK represents the hard K sound.
	PhoneticK = "K"

	// PhoneticF is the phonetic letter for the F and V sounds.
	PhoneticF = "F"

	// PhoneticS is the phonetic alphabet letter for the S sound.
	PhoneticS = "S"

	// PhoneticEI represents the EI/IJ diphthong sound.
	PhoneticEI = "EI"

	// PhoneticU represents the OE sound (like "oo").
	PhoneticU = "U"

	// PhoneticOI represents the OI diphthong sound.
	PhoneticOI = "OI"

	// PhoneticE represents the EU sound.
	PhoneticE = "E"

	// PhoneticAU represents the OU/AU vowel sound combination.
	PhoneticAU = "AU"
)

// hasPrefix checks if word starting at position has the given prefix.
//
// Takes word (string) which is the string to check within.
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
