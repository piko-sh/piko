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

import (
	"strings"
	"unicode"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

const (
	// Language is the language code for this encoder.
	Language = "russian"

	// DefaultMaxLength is the default maximum length for Russian phonetic codes.
	DefaultMaxLength = 6

	// cyrillicAlphabetSize is the number of letters in the main Cyrillic block
	// (А-Я).
	cyrillicAlphabetSize = 32

	// cyrillicBaseRune is the first letter of the Cyrillic uppercase block.
	cyrillicBaseRune = 'А'
)

// runeHandler processes a rune at the given position and returns the next
// position.
type runeHandler func(runes []rune, position int, result *strings.Builder) int

// cyrillicHandlers is a dispatch table for Cyrillic character processing.
// Index is calculated as character - 'А' for uppercase Cyrillic letters
// (А=0, Б=1, ..., Я=31).
var cyrillicHandlers = [cyrillicAlphabetSize]runeHandler{
	0:  handleA,
	1:  handleB,
	2:  handleV,
	3:  handleG,
	4:  handleD,
	5:  handleE,
	6:  handleZH,
	7:  handleZ,
	8:  handleI,
	9:  handleY,
	10: handleK,
	11: handleL,
	12: handleM,
	13: handleN,
	14: handleO,
	15: handleP,
	16: handleR,
	17: handleS,
	18: handleT,
	19: handleU,
	20: handleF,
	21: handleKH,
	22: handleTS,
	23: handleCH,
	24: handleSH,
	25: handleSCH,
	26: handleSign,
	27: handleYI,
	28: handleSign,
	29: handleEE,
	30: handleYU,
	31: handleYA,
}

// Encoder provides phonetic encoding using Russian phonetic rules.
// It implements the linguistics_domain.PhoneticEncoderPort interface.
type Encoder struct {
	// maxLength is the maximum number of characters in the output code.
	maxLength int
}

// NewWithMaxLength creates a new Russian phonetic encoder with a custom maximum
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

// Encode returns the phonetic encoding of a Russian word.
//
// Takes word (string) which is the word to encode phonetically. The word should
// be in Cyrillic script.
//
// Returns string which is the phonetic code.
func (e *Encoder) Encode(word string) string {
	if len(word) == 0 {
		return ""
	}

	runes := []rune(strings.ToUpper(word))

	var result strings.Builder
	position := 0

	for position < len(runes) && result.Len() < e.maxLength {
		position = processCharacter(runes, position, &result)
	}

	code := result.String()
	if len(code) > e.maxLength {
		return code[:e.maxLength]
	}
	return code
}

// GetLanguage returns the language this encoder is configured for.
//
// Returns string which is the language code.
func (*Encoder) GetLanguage() string {
	return Language
}

var _ linguistics_domain.PhoneticEncoderPort = (*Encoder)(nil)

// Factory creates a new Russian phonetic encoder instance. Use this with
// linguistics_domain.RegisterPhoneticEncoderFactory for explicit registration.
//
// Returns linguistics_domain.PhoneticEncoderPort which is the encoder instance.
// Returns error when the encoder cannot be created.
func Factory() (linguistics_domain.PhoneticEncoderPort, error) {
	return New()
}

// New creates a new Russian phonetic encoder.
//
// Returns *Encoder which is ready for use.
// Returns error which is always nil for this encoder.
func New() (*Encoder, error) {
	return NewWithMaxLength(DefaultMaxLength)
}

// processCharacter processes a single character and returns the next position.
// Uses array dispatch table for O(1) handler lookup.
//
// Takes runes ([]rune) which contains the input text as a slice of runes.
// Takes position (int) which specifies the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the output characters.
//
// Returns int which is the next position to process in the rune slice.
func processCharacter(runes []rune, position int, result *strings.Builder) int {
	character := runes[position]

	if isWordEnd(runes, position) && isVoicedConsonant(character) {
		character = devoice(character)
	}

	if character == CyrYO {
		return handleYO(runes, position, result)
	}

	if character >= cyrillicBaseRune && character < cyrillicBaseRune+cyrillicAlphabetSize {
		index := character - cyrillicBaseRune
		if handler := cyrillicHandlers[index]; handler != nil {
			return handler(runes, position, result)
		}
	}

	if unicode.IsLetter(character) {
		_, _ = result.WriteRune(unicode.ToUpper(character))
	}
	return position + 1
}

// handleA dispatches the phonetic encoding for the letter A.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleA(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticA)
	return position + 1
}

// handleB processes a Cyrillic 'Б' character and returns the new position.
//
// Takes runes ([]rune) which is the input slice of characters.
// Takes position (int) which is the current position in the slice.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleB(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrB, PhoneticB, result)
}

// handleV processes the letter V and writes its phonetic equivalent.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the new position after processing.
func handleV(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrV, PhoneticV, result)
}

// handleG processes the letter G at the given position in the input.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the runes slice.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the new position after processing.
func handleG(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrG, PhoneticG, result)
}

// handleD processes the Cyrillic letter D and writes its phonetic form.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the number of runes consumed.
func handleD(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrD, PhoneticD, result)
}

// handleE processes the Cyrillic letter Е and writes its phonetic output.
//
// When the letter appears at word start or after a vowel, writes PhoneticYE.
// Otherwise, writes PhoneticE.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the next position to process.
func handleE(runes []rune, position int, result *strings.Builder) int {
	if position == 0 || (position > 0 && isRussianVowel(runes[position-1])) {
		_, _ = result.WriteString(PhoneticYE)
		return position + 1
	}
	_, _ = result.WriteString(PhoneticE)
	return position + 1
}

// handleZH dispatches the phonetic encoding for the letter ZH.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleZH(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticZH)
	return position + 1
}

// handleZ processes the letter Z for transliteration.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the transliterated output.
//
// Returns int which is the new position after processing.
func handleZ(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrZ, PhoneticZ, result)
}

// handleI dispatches the phonetic encoding for the letter I.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleI(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticI)
	return position + 1
}

// handleY dispatches the phonetic encoding for the letter Y.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleY(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticJ)
	return position + 1
}

// handleK processes the letter K for phonetic conversion.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the new position after processing.
func handleK(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrK, PhoneticK, result)
}

// handleL processes the letter L for transliteration.
//
// Takes runes ([]rune) which contains the input text being processed.
// Takes position (int) which specifies the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the transliterated output.
//
// Returns int which is the next position to process after handling the letter.
func handleL(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrL, PhoneticL, result)
}

// handleM processes the letter M at the given position.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the runes slice.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the new position after processing.
func handleM(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrM, PhoneticM, result)
}

// handleN handles the Cyrillic letter N for transliteration.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the transliterated output.
//
// Returns int which is the new position after processing the character.
func handleN(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrN, PhoneticN, result)
}

// handleO dispatches the phonetic encoding for the letter O.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleO(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticO)
	return position + 1
}

// handleP processes the letter P at the given position in the runes slice.
//
// Takes runes ([]rune) which contains the text being transliterated.
// Takes position (int) which specifies the current position in the runes slice.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the number of runes consumed.
func handleP(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrP, PhoneticP, result)
}

// handleR processes the letter R for transliteration.
//
// Takes runes ([]rune) which contains the input text as runes.
// Takes position (int) which specifies the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the transliterated output.
//
// Returns int which is the number of runes consumed.
func handleR(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrR, PhoneticR, result)
}

// handleS processes the Cyrillic letter С in transliteration.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the transliterated output.
//
// Returns int which is the new position after processing.
func handleS(runes []rune, position int, result *strings.Builder) int {
	if position+DigraphLength < len(runes) && runes[position+1] == CyrT && runes[position+DigraphLength] == CyrN {
		_, _ = result.WriteString(PhoneticS)
		_, _ = result.WriteString(PhoneticN)
		return position + TrigraphLength
	}

	return handleDoubleConsonant(runes, position, CyrS, PhoneticS, result)
}

// handleT processes a Cyrillic T character for transliteration.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the runes slice.
// Takes result (*strings.Builder) which accumulates the transliterated output.
//
// Returns int which is the next position to process after handling the
// character.
func handleT(runes []rune, position int, result *strings.Builder) int {
	if position+1 < len(runes) && runes[position+1] == CyrS {
		_, _ = result.WriteString(PhoneticTS)
		return position + DigraphLength
	}

	return handleDoubleConsonant(runes, position, CyrT, PhoneticT, result)
}

// handleU dispatches the phonetic encoding for the letter U.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleU(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticU)
	return position + 1
}

// handleF processes the Cyrillic letter F and writes its phonetic form.
//
// Takes runes ([]rune) which is the input text as a slice of runes.
// Takes position (int) which is the current position in the rune slice.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the new position after processing.
func handleF(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, CyrF, PhoneticF, result)
}

// handleKH dispatches the phonetic encoding for the letter KH.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleKH(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticX)
	return position + 1
}

// handleTS dispatches the phonetic encoding for the letter TS.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleTS(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticTS)
	return position + 1
}

// handleCH dispatches the phonetic encoding for the letter CH.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleCH(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticCH)
	return position + 1
}

// handleSH dispatches the phonetic encoding for the letter SH.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleSH(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticSH)
	return position + 1
}

// handleSCH dispatches the phonetic encoding for the letter SCH.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleSCH(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticSC)
	return position + 1
}

// handleSign dispatches the phonetic encoding for a soft/hard sign.
//
// Takes position (int) which is the current position in the word.
//
// Returns int which is the updated position after processing.
func handleSign(_ []rune, position int, _ *strings.Builder) int {
	return position + 1
}

// handleYI dispatches the phonetic encoding for the letter YI.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleYI(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticI)
	return position + 1
}

// handleEE dispatches the phonetic encoding for the letter EE.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleEE(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(PhoneticE)
	return position + 1
}

// handleYO dispatches the phonetic encoding for the letter YO.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleYO(_ []rune, position int, result *strings.Builder) int {
	if position == 0 {
		_, _ = result.WriteString(PhoneticYO)
	} else {
		_, _ = result.WriteString(PhoneticO)
	}
	return position + 1
}

// handleYU dispatches the phonetic encoding for the letter YU.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleYU(_ []rune, position int, result *strings.Builder) int {
	if position == 0 {
		_, _ = result.WriteString(PhoneticYU)
	} else {
		_, _ = result.WriteString(PhoneticU)
	}
	return position + 1
}

// handleYA dispatches the phonetic encoding for the letter YA.
//
// Takes position (int) which is the current position in the word.
// Takes result (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleYA(_ []rune, position int, result *strings.Builder) int {
	if position == 0 {
		_, _ = result.WriteString(PhoneticYA)
	} else {
		_, _ = result.WriteString(PhoneticA)
	}
	return position + 1
}

// handleDoubleConsonant handles consonants that may double.
//
// Takes runes ([]rune) which is the input word as a slice of runes.
// Takes position (int) which is the current position in the runes slice.
// Takes letter (rune) which is the consonant to check for doubling.
// Takes code (string) which is the phonetic code to write.
// Takes result (*strings.Builder) which accumulates the output.
//
// Returns int which is the next position to process.
func handleDoubleConsonant(runes []rune, position int, letter rune, code string, result *strings.Builder) int {
	_, _ = result.WriteString(code)
	if position+1 < len(runes) && runes[position+1] == letter {
		return position + DigraphLength
	}
	return position + 1
}

func init() { linguistics_domain.RegisterPhoneticEncoderFactory(Language, Factory) }
