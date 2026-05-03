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

import (
	"strings"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

const (
	// Language is the language code for this encoder.
	Language = "french"

	// DefaultMaxLength is the default maximum length for French phonetic codes.
	DefaultMaxLength = 6

	// latinAlphabetSize is the number of letters in the Latin alphabet after
	// Unicode normalisation.
	latinAlphabetSize = 26
)

// charHandler processes a character at the given position and returns the next
// position.
type charHandler func(word string, position int, result *strings.Builder) int

// charHandlers is an array dispatch table for character processing.
// Index is calculated as character - 'A' for uppercase letters
// (A=0, B=1, ..., Z=25).
var charHandlers = [latinAlphabetSize]charHandler{
	0:  handleA,
	1:  handleB,
	2:  handleC,
	3:  handleD,
	4:  handleE,
	5:  handleF,
	6:  handleG,
	7:  handleH,
	8:  handleI,
	9:  handleJ,
	10: handleK,
	11: handleL,
	12: handleM,
	13: handleN,
	14: handleO,
	15: handleP,
	16: handleQ,
	17: handleR,
	18: handleS,
	19: handleT,
	20: handleU,
	21: handleV,
	22: handleW,
	23: handleX,
	24: handleY,
	25: handleZ,
}

// Encoder provides phonetic encoding using French phonetic rules.
// It implements the linguistics_domain.PhoneticEncoderPort interface.
type Encoder struct {
	// maxLength is the maximum length in runes of the output code.
	maxLength int
}

// NewWithMaxLength creates a new French phonetic encoder with a custom maximum
// code length.
//
// Takes maxLength (int) which controls the maximum length of phonetic codes.
//
// Returns (*Encoder, error) where the error is always nil for this encoder.
func NewWithMaxLength(maxLength int) (*Encoder, error) {
	if maxLength <= 0 {
		maxLength = DefaultMaxLength
	}

	return &Encoder{
		maxLength: maxLength,
	}, nil
}

// Encode returns the phonetic encoding of a French word.
//
// Takes word (string) which is the word to encode phonetically. The word should
// be normalised (no diacritics) for best results.
//
// Returns string which is the phonetic code.
func (e *Encoder) Encode(word string) string {
	if len(word) == 0 {
		return ""
	}

	word = strings.ToUpper(word)

	word = removeSilentEndings(word)

	var result strings.Builder
	position := 0

	for position < len(word) && result.Len() < e.maxLength {
		position = processCharacter(word, position, &result)
	}

	return linguistics_domain.TruncateRunes(result.String(), e.maxLength)
}

// GetLanguage returns the language this encoder is configured for.
//
// Returns string which is the language code.
func (*Encoder) GetLanguage() string {
	return Language
}

var _ linguistics_domain.PhoneticEncoderPort = (*Encoder)(nil)

// Factory creates a new French phonetic encoder instance.
//
// Use this with linguistics_domain.RegisterPhoneticEncoderFactory for
// explicit registration.
//
// Returns linguistics_domain.PhoneticEncoderPort which is the encoder ready
// for use.
// Returns error when the encoder cannot be created.
func Factory() (linguistics_domain.PhoneticEncoderPort, error) {
	return New()
}

// New creates a new French phonetic encoder with the default maximum length.
//
// Returns *Encoder which is ready to encode text using French phonetic rules.
// Returns error which is always nil for this encoder.
func New() (*Encoder, error) {
	return NewWithMaxLength(DefaultMaxLength)
}

// processCharacter processes a single character and returns the next position.
// Uses array dispatch table for O(1) handler lookup.
//
// Takes word (string) which is the input string being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the processed output.
//
// Returns int which is the next position to process in the word.
func processCharacter(word string, position int, result *strings.Builder) int {
	character := word[position]

	if character >= 'A' && character <= 'Z' {
		if handler := charHandlers[character-'A']; handler != nil {
			return handler(word, position, result)
		}
	}

	return position + 1
}

// handleA processes the letter A at the given position in a word.
// It handles French phonetic patterns including AU/EAU to O, AI to E,
// and nasal AN/AM sounds.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the new position after processing.
func handleA(word string, position int, result *strings.Builder) int {
	if hasPrefix(word, position, "AU") {
		_, _ = result.WriteString(PhoneticO)
		return position + DigraphLength
	}

	if hasPrefix(word, position, "AI") {
		_, _ = result.WriteString("E")
		return position + DigraphLength
	}

	if position+1 < len(word) {
		next := word[position+1]
		if (next == 'N' || next == 'M') && isNasalFollower(word, position+1) {
			_, _ = result.WriteString(PhoneticNasalAN)
			return position + DigraphLength
		}
	}

	_, _ = result.WriteString("A")
	return position + 1
}

// handleB processes the letter B at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleB(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'B', "P", result)
}

// handleC processes the letter C at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process.
func handleC(word string, position int, result *strings.Builder) int {
	if hasPrefix(word, position, "CH") {
		_, _ = result.WriteString(PhoneticX)
		return position + DigraphLength
	}

	if position+1 < len(word) {
		next := word[position+1]
		if next == 'E' || next == 'I' || next == 'Y' {
			_, _ = result.WriteString(PhoneticS)
			return position + 1
		}
	}

	if position+1 < len(word) && word[position+1] == 'C' {
		_, _ = result.WriteString(PhoneticK)
		return position + DigraphLength
	}

	_, _ = result.WriteString(PhoneticK)
	return position + 1
}

// handleD processes the letter D at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleD(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'D', "T", result)
}

// handleE processes the letter E at the given position in the word.
//
// Takes word (string) which is the input word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the new position after consuming the E pattern.
func handleE(word string, position int, result *strings.Builder) int {
	if hasPrefix(word, position, "EAU") {
		_, _ = result.WriteString(PhoneticO)
		return position + TrigraphLength
	}

	if hasPrefix(word, position, "EU") {
		_, _ = result.WriteString(PhoneticEU)
		return position + DigraphLength
	}

	if hasPrefix(word, position, "EI") {
		_, _ = result.WriteString("E")
		return position + DigraphLength
	}

	if position+1 < len(word) {
		next := word[position+1]
		if (next == 'N' || next == 'M') && isNasalFollower(word, position+1) {
			_, _ = result.WriteString(PhoneticNasalAN)
			return position + DigraphLength
		}
	}

	_, _ = result.WriteString("E")
	return position + 1
}

// handleF processes the letter F at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleF(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'F', "F", result)
}

// handleG processes the letter G at the given position and writes its phonetic
// representation to result.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which receives the phonetic output.
//
// Returns int which is the new position after processing the G.
func handleG(word string, position int, result *strings.Builder) int {
	if hasPrefix(word, position, "GN") {
		_, _ = result.WriteString(PhoneticN)
		return position + DigraphLength
	}

	if position+1 < len(word) {
		next := word[position+1]
		if next == 'E' || next == 'I' || next == 'Y' {
			_, _ = result.WriteString(PhoneticJ)
			return position + 1
		}
	}

	if hasPrefix(word, position, "GU") {
		if position+DigraphLength < len(word) {
			following := word[position+DigraphLength]
			if following == 'E' || following == 'I' {
				_, _ = result.WriteString("G")
				return position + DigraphLength
			}
		}
	}

	return handleDoubleConsonant(word, position, 'G', "G", result)
}

// handleH dispatches the phonetic encoding for the letter H.
//
// Takes position (int) which is the current position in the word.
//
// Returns int which is the updated position after processing.
func handleH(_ string, position int, _ *strings.Builder) int {
	return position + 1
}

// handleI processes the letter I at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process in the word.
func handleI(word string, position int, result *strings.Builder) int {
	if position+1 < len(word) {
		next := word[position+1]
		if (next == 'N' || next == 'M') && isNasalFollower(word, position+1) {
			_, _ = result.WriteString(PhoneticNasalIN)
			return position + DigraphLength
		}
	}

	_, _ = result.WriteString("I")
	return position + 1
}

// handleJ dispatches the phonetic encoding for the letter J.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleJ(_ string, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticJ)
	return position + 1
}

// handleK processes the letter K at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process.
func handleK(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'K', PhoneticK, result)
}

// handleL processes the letter L in phonetic encoding.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process after handling L.
func handleL(word string, position int, result *strings.Builder) int {
	if position >= 1 && position+1 < len(word) && word[position+1] == 'L' {
		previous := word[position-1]
		if previous == 'I' {
			_, _ = result.WriteString(PhoneticJ)
			return position + DigraphLength
		}
	}

	return handleDoubleConsonant(word, position, 'L', "L", result)
}

// handleM processes the letter M at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleM(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'M', "M", result)
}

// handleN processes the letter N at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleN(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'N', "N", result)
}

// handleO processes the letter O at the given position in word.
//
// Takes word (string) which is the input word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the new position after processing.
func handleO(word string, position int, result *strings.Builder) int {
	if hasPrefix(word, position, "OI") {
		_, _ = result.WriteString(PhoneticWA)
		return position + DigraphLength
	}

	if hasPrefix(word, position, "OU") {
		_, _ = result.WriteString(PhoneticU)
		return position + DigraphLength
	}

	if position+1 < len(word) {
		next := word[position+1]
		if (next == 'N' || next == 'M') && isNasalFollower(word, position+1) {
			_, _ = result.WriteString(PhoneticNasalON)
			return position + DigraphLength
		}
	}

	_, _ = result.WriteString("O")
	return position + 1
}

// handleP processes the letter P at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleP(word string, position int, result *strings.Builder) int {
	if hasPrefix(word, position, "PH") {
		_, _ = result.WriteString("F")
		return position + DigraphLength
	}

	return handleDoubleConsonant(word, position, 'P', "P", result)
}

// handleQ processes the letter Q in a word for phonetic encoding.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process in the word.
func handleQ(word string, position int, result *strings.Builder) int {
	if hasPrefix(word, position, "QU") {
		_, _ = result.WriteString(PhoneticK)
		return position + DigraphLength
	}

	_, _ = result.WriteString(PhoneticK)
	return position + 1
}

// handleR processes the letter R at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleR(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'R', "R", result)
}

// handleS processes the letter S at the given position in a word.
//
// When S appears between two vowels, it is converted to the Z phonetic sound.
// Otherwise, it is handled as a double consonant.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process.
func handleS(word string, position int, result *strings.Builder) int {
	if position > 0 && position+1 < len(word) {
		previous := word[position-1]
		next := word[position+1]
		if isFrenchVowel(previous) && isFrenchVowel(next) {
			_, _ = result.WriteString(PhoneticZ)
			return position + 1
		}
	}

	return handleDoubleConsonant(word, position, 'S', PhoneticS, result)
}

// handleT processes the letter T at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the new position after processing.
func handleT(word string, position int, result *strings.Builder) int {
	if hasPrefix(word, position, "TION") {
		_, _ = result.WriteString("SJO")
		return position + QuadgraphLength
	}

	return handleDoubleConsonant(word, position, 'T', "T", result)
}

// handleU processes the letter U at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process.
func handleU(word string, position int, result *strings.Builder) int {
	if position+1 < len(word) {
		next := word[position+1]
		if (next == 'N' || next == 'M') && isNasalFollower(word, position+1) {
			_, _ = result.WriteString(PhoneticNasalUN)
			return position + DigraphLength
		}
	}

	_, _ = result.WriteString("U")
	return position + 1
}

// handleV processes the letter V at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleV(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'V', "V", result)
}

// handleW dispatches the phonetic encoding for the letter W.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleW(_ string, position int, result *strings.Builder) int {
	_, _ = result.WriteString("V")
	return position + 1
}

// handleX dispatches the phonetic encoding for the letter X.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleX(_ string, position int, result *strings.Builder) int {
	_, _ = result.WriteString("KS")
	return position + 1
}

// handleY dispatches the phonetic encoding for the letter Y.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleY(_ string, position int, result *strings.Builder) int {
	_, _ = result.WriteString("I")
	return position + 1
}

// handleZ processes the letter Z at the given position in a word.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process.
func handleZ(word string, position int, result *strings.Builder) int {
	return handleDoubleConsonant(word, position, 'Z', PhoneticZ, result)
}

// handleDoubleConsonant handles consonants that may double.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
// Takes letter (byte) which is the consonant to check for doubling.
// Takes code (string) which is the phonetic code to write.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleDoubleConsonant(word string, position int, letter byte, code string, result *strings.Builder) int {
	_, _ = result.WriteString(code)
	if position+1 < len(word) && word[position+1] == letter {
		return position + DigraphLength
	}
	return position + 1
}

func init() {
	linguistics_domain.RegisterPhoneticEncoderFactory(Language, Factory)
}
