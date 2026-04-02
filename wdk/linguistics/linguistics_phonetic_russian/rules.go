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

package linguistics_phonetic_russian

const (
	// CyrA is the Cyrillic capital letter A.
	CyrA = 'А'

	// CyrE is the Cyrillic capital letter Ie (Е).
	CyrE = 'Е'

	// CyrYO is the Cyrillic capital letter IO (Ё).
	CyrYO = 'Ё'

	// CyrI is the Cyrillic capital letter I (И).
	CyrI = 'И'

	// CyrY is the Cyrillic capital letter short I (Й).
	CyrY = 'Й'

	// CyrO is the Cyrillic capital letter O.
	CyrO = 'О'

	// CyrU is the Cyrillic capital letter U.
	CyrU = 'У'

	// CyrYI is the Cyrillic capital letter Yeru (Ы).
	CyrYI = 'Ы'

	// CyrEE is the Cyrillic capital letter E with descender (Э).
	CyrEE = 'Э'

	// CyrYU is the Cyrillic capital letter Yu (Ю).
	CyrYU = 'Ю'

	// CyrYA is the Cyrillic capital letter Ya (Я).
	CyrYA = 'Я'

	// CyrB is the Cyrillic capital letter Be, a consonant.
	CyrB = 'Б'

	// CyrV is the Cyrillic capital letter Ve.
	CyrV = 'В'

	// CyrG is the Cyrillic capital letter Ge.
	CyrG = 'Г'

	// CyrD is the Cyrillic capital letter De.
	CyrD = 'Д'

	// CyrZH is the Cyrillic capital letter Zhe.
	CyrZH = 'Ж'

	// CyrZ is the Cyrillic capital letter Ze (З).
	CyrZ = 'З'

	// CyrK is the Cyrillic capital letter Ka (U+041A).
	CyrK = 'К'

	// CyrL is the Cyrillic capital letter El (Л).
	CyrL = 'Л'

	// CyrM is the Cyrillic capital letter Em (U+041C).
	CyrM = 'М'

	// CyrN is the Cyrillic capital letter En (Н).
	CyrN = 'Н'

	// CyrP is the Cyrillic capital letter Pe.
	CyrP = 'П'

	// CyrR is the Cyrillic capital letter Er (Р).
	CyrR = 'Р'

	// CyrS is the Cyrillic capital letter Es (U+0421).
	CyrS = 'С'

	// CyrT is the Cyrillic capital letter Te.
	CyrT = 'Т'

	// CyrF is the Cyrillic capital letter Ef (Ф).
	CyrF = 'Ф'

	// CyrKH is the Cyrillic capital letter Kha (Х).
	CyrKH = 'Х'

	// CyrTS is the Cyrillic capital letter Tse.
	CyrTS = 'Ц'

	// CyrCH is the Cyrillic capital letter Che (Ч).
	CyrCH = 'Ч'

	// CyrSH is the Cyrillic capital letter sha (Ш).
	CyrSH = 'Ш'

	// CyrSCH is the Cyrillic capital letter Shcha.
	CyrSCH = 'Щ'

	// CyrHard is the Cyrillic hard sign character.
	CyrHard = 'Ъ'

	// CyrSoft is the Cyrillic soft sign character.
	CyrSoft = 'Ь'

	// PhoneticA is the letter A in the phonetic alphabet.
	PhoneticA = "A"

	// PhoneticB is the NATO phonetic alphabet letter B.
	PhoneticB = "B"

	// PhoneticV is the phonetic alphabet code for the letter V.
	PhoneticV = "V"

	// PhoneticG is the phonetic alphabet letter for G.
	PhoneticG = "G"

	// PhoneticD is the phonetic alphabet letter for D.
	PhoneticD = "D"

	// PhoneticE is the NATO phonetic alphabet letter for E (Echo).
	PhoneticE = "E"

	// PhoneticI is the NATO phonetic alphabet representation for the letter I.
	PhoneticI = "I"

	// PhoneticO is the NATO phonetic alphabet representation for the letter O.
	PhoneticO = "O"

	// PhoneticU is the NATO phonetic alphabet code for the letter U.
	PhoneticU = "U"

	// PhoneticK is the NATO phonetic alphabet representation of the letter K.
	PhoneticK = "K"

	// PhoneticL is the NATO phonetic alphabet representation for the letter L.
	PhoneticL = "L"

	// PhoneticM is the phonetic alphabet code for the letter M.
	PhoneticM = "M"

	// PhoneticN is the NATO phonetic alphabet letter for November.
	PhoneticN = "N"

	// PhoneticP is the letter P in the phonetic alphabet.
	PhoneticP = "P"

	// PhoneticR is the phonetic alphabet letter for R.
	PhoneticR = "R"

	// PhoneticS is the phonetic alphabet code for the letter S.
	PhoneticS = "S"

	// PhoneticT is the phonetic alphabet representation for the letter T.
	PhoneticT = "T"

	// PhoneticF is the NATO phonetic alphabet letter for F.
	PhoneticF = "F"

	// PhoneticX is the phonetic alphabet code for the letter X.
	PhoneticX = "X"

	// PhoneticTS is the phonetic alphabet code for the letter combination TS.
	PhoneticTS = "TS"

	// PhoneticCH is the phonetic alphabet representation for the letter C.
	PhoneticCH = "C"

	// PhoneticSH is the phonetic representation for the SH sound.
	PhoneticSH = "S"

	// PhoneticSC is the phonetic alphabet code for the letter sequence SC.
	PhoneticSC = "SC"

	// PhoneticZ is the NATO phonetic alphabet letter for Z.
	PhoneticZ = "Z"

	// PhoneticZH is the NATO phonetic alphabet code for the letter Z.
	PhoneticZH = "Z"

	// PhoneticJ is the NATO phonetic alphabet representation for the letter J.
	PhoneticJ = "J"

	// PhoneticYA is the phonetic alphabet representation for the letter Y.
	PhoneticYA = "JA"

	// PhoneticYU is the phonetic representation of the Cyrillic letter Ю.
	PhoneticYU = "JU"

	// PhoneticYO is the phonetic representation of the Cyrillic letter Ё.
	PhoneticYO = "JO"

	// PhoneticYE is the phonetic alphabet code for the Cyrillic letter Ye.
	PhoneticYE = "JE"

	// DigraphLength is the length of a two-character sequence.
	DigraphLength = 2

	// TrigraphLength is the length of a trigraph, which is three characters.
	TrigraphLength = 3
)

// isRussianVowel reports whether the rune is a Russian vowel.
//
// Takes character (rune) which is the character to check.
//
// Returns bool which is true if character is a Russian vowel, false otherwise.
func isRussianVowel(character rune) bool {
	switch character {
	case CyrA, CyrE, CyrYO, CyrI, CyrO, CyrU, CyrYI, CyrEE, CyrYU, CyrYA:
		return true
	default:
		return false
	}
}

// isVoicedConsonant reports whether the rune is a voiced consonant.
//
// Takes character (rune) which is the character to check.
//
// Returns bool which is true if character is a voiced consonant, false otherwise.
func isVoicedConsonant(character rune) bool {
	switch character {
	case CyrB, CyrV, CyrG, CyrD, CyrZH, CyrZ:
		return true
	default:
		return false
	}
}

// devoice returns the voiceless counterpart of a voiced consonant.
//
// Takes character (rune) which is the character to convert.
//
// Returns rune which is the voiceless counterpart, or the original character
// if no mapping exists.
func devoice(character rune) rune {
	switch character {
	case CyrB:
		return CyrP
	case CyrV:
		return CyrF
	case CyrG:
		return CyrK
	case CyrD:
		return CyrT
	case CyrZH:
		return CyrSH
	case CyrZ:
		return CyrS
	default:
		return character
	}
}

// isWordEnd reports whether position is at or near the end of the rune slice.
//
// Takes runes ([]rune) which is the slice to check position within.
// Takes position (int) which is the position to evaluate.
//
// Returns bool which is true if position is at or beyond the last index.
func isWordEnd(runes []rune, position int) bool {
	return position >= len(runes)-1
}
