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

import (
	"strings"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

const (
	// Language is the language code for this encoder.
	Language = "german"

	// DefaultMaxLength is the default maximum length for German phonetic codes.
	// Cologne phonetic encoding has no built-in length limit, but this value caps
	// the output for practical use.
	DefaultMaxLength = 10
)

// charHandler processes a character at the given position and returns the digit
// code.
// Returns CologneSkip (0) if the character should be skipped.
type charHandler func(word string, position int) byte

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
	23: nil,
	24: handleY,
	25: handleZ,
}

// Encoder provides phonetic encoding using the Cologne Phonetic (Koelner
// Phonetik) algorithm. It implements the linguistics_domain.PhoneticEncoderPort
// interface.
//
// The Cologne phonetics algorithm was published in 1969 by Hans Joachim Postel
// and is optimised for matching German names and words.
type Encoder struct {
	// maxLength is the maximum number of characters in the output code.
	maxLength int
}

// NewWithMaxLength creates a new German phonetic encoder with a custom maximum
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

// Encode returns the Cologne phonetic code for a German word.
//
// Takes word (string) which is the word to encode phonetically. The word should
// be normalised (umlauts expanded: ae, oe, ue) for best results.
//
// Returns string which is the phonetic code consisting of digits 0-8.
func (e *Encoder) Encode(word string) string {
	if len(word) == 0 {
		return ""
	}

	word = strings.ToUpper(word)
	code := encodeCharacters(word)
	code = removeDuplicates(code)
	code = removeInternalZeros(code)

	if len(code) > e.maxLength {
		code = code[:e.maxLength]
	}

	return string(code)
}

// GetLanguage returns the language this encoder is set up for.
//
// Returns string which is the language code.
func (*Encoder) GetLanguage() string {
	return Language
}

var _ linguistics_domain.PhoneticEncoderPort = (*Encoder)(nil)

// Factory creates a new German phonetic encoder instance. Use this with
// linguistics_domain.RegisterPhoneticEncoderFactory for explicit registration.
//
// Returns linguistics_domain.PhoneticEncoderPort which is the encoder instance.
// Returns error when the encoder cannot be created.
func Factory() (linguistics_domain.PhoneticEncoderPort, error) {
	return New()
}

// New creates a new German phonetic encoder with the default maximum length.
//
// Returns *Encoder which is the configured encoder ready for use.
// Returns error which is always nil for this encoder.
func New() (*Encoder, error) {
	return NewWithMaxLength(DefaultMaxLength)
}

// encodeCharacters applies letter-to-digit mapping for each character.
//
// Takes word (string) which is the input to encode.
//
// Returns []byte which contains the encoded digit sequence.
func encodeCharacters(word string) []byte {
	code := make([]byte, 0, len(word))
	for position := range len(word) {
		code = appendCharDigits(code, word, position)
	}
	return code
}

// appendCharDigits appends the digit(s) for the character at the given
// position.
//
// Takes code ([]byte) which holds the accumulated Cologne phonetic digits.
// Takes word (string) which contains the word being encoded.
// Takes position (int) which specifies the character position to process.
//
// Returns []byte which contains the code with any new digits appended.
func appendCharDigits(code []byte, word string, position int) []byte {
	character := word[position]
	if character < 'A' || character > 'Z' {
		return code
	}

	if character == 'X' {
		return appendXDigits(code, word, position)
	}

	handler := charHandlers[character-'A']
	if handler == nil {
		return code
	}

	if digit := handler(word, position); digit != CologneSkip {
		code = append(code, digit)
	}
	return code
}

// appendXDigits handles the special case of X which produces two digits.
//
// Takes code ([]byte) which is the current Cologne phonetic code buffer.
// Takes word (string) which is the input word being encoded.
// Takes position (int) which is the current position in the word.
//
// Returns []byte which is the updated code with one or two digits appended.
func appendXDigits(code []byte, word string, position int) []byte {
	if isPrecedingChar(word, position, "CKQ") {
		return append(code, CologneSZ)
	}
	return append(code, CologneGKQ, CologneSZ)
}

// handleA returns the Cologne vowel code for any input.
//
// Returns byte which is always CologneVowel regardless of the input values.
func handleA(_ string, _ int) byte { return CologneVowel }

// handleE returns a Cologne vowel code for any input.
//
// Returns byte which is always CologneVowel.
func handleE(_ string, _ int) byte { return CologneVowel }

// handleI returns the Cologne phonetic vowel code.
//
// Returns byte which is the CologneVowel constant.
func handleI(_ string, _ int) byte { return CologneVowel }

// handleJ returns the Cologne vowel code for J characters.
//
// Returns byte which is the CologneVowel constant.
func handleJ(_ string, _ int) byte { return CologneVowel }

// handleO returns the Cologne phonetic code for the letter O.
//
// Returns byte which is CologneVowel representing the vowel code.
func handleO(_ string, _ int) byte { return CologneVowel }

// handleU returns the Cologne phonetic code for the letter U.
//
// Returns byte which is CologneVowel.
func handleU(_ string, _ int) byte { return CologneVowel }

// handleY returns the Cologne vowel code for any input.
//
// Returns byte which is always CologneVowel.
func handleY(_ string, _ int) byte { return CologneVowel }

// handleH returns the skip value for the letter H.
//
// Returns byte which is the Cologne phonetic skip constant.
func handleH(_ string, _ int) byte { return CologneSkip }

// handleB returns the Cologne phonetic code for the letter B.
//
// Returns byte which is the Cologne phonetic value CologneBP.
func handleB(_ string, _ int) byte { return CologneBP }

// handleP returns the Cologne phonetic code for the letter P.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the current position in the word.
//
// Returns byte which is CologneFVW if P is followed by H, otherwise CologneBP.
func handleP(word string, position int) byte {
	if isFollowingChar(word, position, "H") {
		return CologneFVW
	}
	return CologneBP
}

// handleD returns the Cologne phonetic code for the letter D.
//
// Takes word (string) which is the word being encoded.
// Takes position (int) which is the position of the D in the word.
//
// Returns byte which is CologneSZ if D precedes C, S, or Z, otherwise
// CologneDT.
func handleD(word string, position int) byte {
	if isFollowingChar(word, position, "CSZ") {
		return CologneSZ
	}
	return CologneDT
}

// handleT returns the Cologne phonetic code for the letter T.
//
// Takes word (string) which is the word being processed.
// Takes position (int) which is the position of T in the word.
//
// Returns byte which is CologneSZ if T precedes C, S, or Z, otherwise
// CologneDT.
func handleT(word string, position int) byte {
	if isFollowingChar(word, position, "CSZ") {
		return CologneSZ
	}
	return CologneDT
}

// handleF returns the Cologne phonetic code for F-type consonants.
//
// Returns byte which is the Cologne phonetic code CologneFVW.
func handleF(_ string, _ int) byte { return CologneFVW }

// handleV returns the Cologne phonetic code for the letter V.
//
// Returns byte which is the CologneFVW code value.
func handleV(_ string, _ int) byte { return CologneFVW }

// handleW returns the Cologne phonetic code for the letter W.
//
// Returns byte which is the CologneFVW constant.
func handleW(_ string, _ int) byte { return CologneFVW }

// handleG returns the Cologne phonetic code for the letter G.
//
// Returns byte which is the CologneGKQ code value.
func handleG(_ string, _ int) byte { return CologneGKQ }

// handleK returns the Cologne phonetic code for the letter K.
//
// Returns byte which is the CologneGKQ code value.
func handleK(_ string, _ int) byte { return CologneGKQ }

// handleQ returns the Cologne phonetic code for Q sounds.
//
// Returns byte which is the CologneGKQ code value.
func handleQ(_ string, _ int) byte { return CologneGKQ }

// handleC returns the Cologne phonetic code for the letter C.
//
// Takes word (string) which is the word being encoded.
// Takes position (int) which is the position of C in the word.
//
// Returns byte which is the phonetic code based on surrounding context.
func handleC(word string, position int) byte {
	if position == 0 && isCHardContext(word, position) {
		return CologneGKQ
	}
	if isPrecedingChar(word, position, "SZ") {
		return CologneSZ
	}
	if isCHardContext(word, position) {
		return CologneGKQ
	}
	return CologneSZ
}

// handleL returns the Cologne phonetic code for the letter L.
//
// Returns byte which is the CologneL constant.
func handleL(_ string, _ int) byte { return CologneL }

// handleM returns the Cologne phonetic code for the letter M.
//
// Returns byte which is always CologneMN.
func handleM(_ string, _ int) byte { return CologneMN }

// handleN returns the Cologne phonetic code for the letter N.
//
// Returns byte which is always CologneMN.
func handleN(_ string, _ int) byte { return CologneMN }

// handleR returns the Cologne phonetic code for the letter R.
//
// Returns byte which is the CologneR constant.
func handleR(_ string, _ int) byte { return CologneR }

// handleS returns the Cologne phonetic code for the letter S.
//
// Returns byte which is the CologneSZ constant.
func handleS(_ string, _ int) byte { return CologneSZ }

// handleZ returns the Cologne phonetic code for the letter Z.
//
// Returns byte which is the CologneSZ constant.
func handleZ(_ string, _ int) byte { return CologneSZ }

func init() { linguistics_domain.RegisterPhoneticEncoderFactory(Language, Factory) }
