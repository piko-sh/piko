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

package linguistics_domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// PhoneticK is the phonetic code for the K sound.
	PhoneticK = "K"

	// PhoneticS is the phonetic code for the S sound.
	PhoneticS = "S"

	// PhoneticT is the phonetic code for the T sound.
	PhoneticT = "T"

	// PhoneticX is the phonetic code for the "sh" sound in patterns like CH, SH,
	// SIO, SIA, TIO, TIA, and TCH.
	PhoneticX = "X"

	// sequenceLength3 is the number of characters to skip for a three-letter sequence.
	sequenceLength3 = 3

	// sequenceLength4 is the length of a four-character sequence.
	sequenceLength4 = 4

	// defaultPhoneticLength is the default length for phonetic codes.
	defaultPhoneticLength = 4

	// soundexCodeLength is the fixed length of a Soundex code (four characters).
	soundexCodeLength = 4
)

// PhoneticEncoder implements the PhoneticEncoderPort interface using the
// Double Metaphone algorithm. It matches words that sound alike but are spelt
// differently, such as "Stephen" and "Steven", or "night" and "knight".
type PhoneticEncoder struct {
	// maxLength is the maximum length in runes of the output code.
	maxLength int
}

// NewPhoneticEncoder creates a new Double Metaphone encoder.
//
// Takes maxLength (int) which controls the maximum length of the returned
// phonetic code (typically 4-6).
//
// Returns *PhoneticEncoder which is the configured encoder ready for use.
func NewPhoneticEncoder(maxLength int) *PhoneticEncoder {
	if maxLength <= 0 {
		maxLength = defaultPhoneticLength
	}
	return &PhoneticEncoder{
		maxLength: maxLength,
	}
}

// Encode returns the primary phonetic encoding of a word.
//
// The input should be normalised (lowercase, no diacritics).
//
// Takes word (string) which is the word to encode phonetically.
//
// Returns string which is the phonetic code for the word.
func (p *PhoneticEncoder) Encode(word string) string {
	if len(word) == 0 {
		return ""
	}

	word = strings.ToUpper(word)

	primary := &strings.Builder{}
	position := p.preprocessWord(word, primary)

	for position < len(word) && primary.Len() < p.maxLength {
		position = p.processCharacter(word, position, primary)
	}

	return p.finalisePhoneticCode(primary.String())
}

// GetLanguage returns the language this encoder supports.
// The Double Metaphone algorithm is designed for English words.
//
// Returns string which is always "english".
func (*PhoneticEncoder) GetLanguage() string {
	return LanguageEnglish
}

// preprocessWord handles leading silent letters and special starting
// characters.
//
// Takes word (string) which is the uppercase word to process.
// Takes primary (*strings.Builder) which stores the phonetic encoding.
//
// Returns int which is the starting position for the main processing loop.
func (*PhoneticEncoder) preprocessWord(word string, primary *strings.Builder) int {
	if strings.HasPrefix(word, "GN") || strings.HasPrefix(word, "KN") ||
		strings.HasPrefix(word, "PN") || strings.HasPrefix(word, "WR") ||
		strings.HasPrefix(word, "PS") {
		return 1
	}

	if word[0] == 'X' {
		primary.WriteString(PhoneticS)
		return 1
	}

	return 0
}

// characterHandler processes a character and returns the next position.
type characterHandler func(*PhoneticEncoder, string, int, *strings.Builder) int

// phoneticDispatchTable maps each letter to its handler function.
var phoneticDispatchTable = map[rune]characterHandler{
	'A': handleVowelDispatch, 'E': handleVowelDispatch, 'I': handleVowelDispatch,
	'O': handleVowelDispatch, 'U': handleVowelDispatch, 'Y': handleVowelDispatch,

	'B': handleBDispatch, 'F': handleFDispatch, 'J': handleJDispatch,
	'K': handleKDispatch, 'L': handleLDispatch, 'M': handleMDispatch,
	'N': handleNDispatch, 'Q': handleQDispatch, 'R': handleRDispatch,
	'V': handleVDispatch, 'Z': handleZDispatch,

	'C': handleCDispatch, 'D': handleDDispatch, 'G': handleGDispatch,
	'H': handleHDispatch, 'P': handlePDispatch, 'S': handleSDispatch,
	'T': handleTDispatch, 'W': handleWDispatch,

	'X': handleXDispatch,
}

// processCharacter handles a single character and returns the next position.
//
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of the character to handle.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the next position to handle in the word.
func (p *PhoneticEncoder) processCharacter(word string, current int, primary *strings.Builder) int {
	character := rune(word[current])

	if handler, exists := phoneticDispatchTable[character]; exists {
		return handler(p, word, current, primary)
	}

	return current + 1
}

// finalisePhoneticCode trims the phonetic code to the maximum length.
//
// Takes code (string) which is the phonetic code to trim.
//
// Returns string which is the code shortened to the encoder's maximum length.
func (p *PhoneticEncoder) finalisePhoneticCode(code string) string {
	return TruncateRunes(code, p.maxLength)
}

// TruncateRunes shortens s to at most maxRunes runes. The function is
// rune-aware so it never cuts through a multi-byte UTF-8 sequence, making it
// safe for Cyrillic, accented Latin, CJK, and other non-ASCII text.
//
// Takes s (string) which is the input to truncate.
// Takes maxRunes (int) which is the maximum number of runes the result may
// contain. Values of zero or below produce an empty string.
//
// Returns string which is at most maxRunes runes long. If s is already short
// enough, s is returned unchanged.
func TruncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes])
}

// handleC processes the letter 'C' based on the letters that follow it.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the position of 'C' in the word.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the next position to process.
func (*PhoneticEncoder) handleC(word string, current int, primary *strings.Builder) int {
	if current+1 < len(word) && word[current+1] == 'H' {
		primary.WriteString(PhoneticX)
		return current + 2
	}

	if current+1 < len(word) {
		next := word[current+1]
		if next == 'E' || next == 'I' || next == 'Y' {
			primary.WriteString(PhoneticS)
			return current + 2
		}
	}

	if current+1 < len(word) && word[current+1] == 'C' {
		primary.WriteString(PhoneticK)
		return current + 2
	}

	primary.WriteString("K")
	return current + 1
}

// handleD processes the letter 'D' based on the letters around it.
//
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of the 'D' in the word.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the next position to process after this letter.
func (*PhoneticEncoder) handleD(word string, current int, primary *strings.Builder) int {
	if current+2 < len(word) && word[current+1] == 'G' {
		next := word[current+2]
		if next == 'E' || next == 'I' || next == 'Y' {
			primary.WriteString("J")
			return current + sequenceLength3
		}
	}

	if current+1 < len(word) && word[current+1] == 'D' {
		primary.WriteString(PhoneticT)
		return current + 2
	}

	primary.WriteString(PhoneticT)
	return current + 1
}

// handleG processes the letter 'G' based on the letters around it.
//
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of 'G' in the word.
// Takes primary (*strings.Builder) which receives the phonetic output.
//
// Returns int which is the new position after processing.
func (*PhoneticEncoder) handleG(word string, current int, primary *strings.Builder) int {
	if isSilentGN(word, current) {
		return current + 2
	}

	if current+1 < len(word) && word[current+1] == 'H' {
		return current + 2
	}

	if isSoftG(word, current) {
		primary.WriteString("J")
		return current + 2
	}

	if current+1 < len(word) && word[current+1] == 'G' {
		primary.WriteString(PhoneticK)
		return current + 2
	}

	primary.WriteString("K")
	return current + 1
}

// handleH processes the letter 'H' based on where it appears in the word.
//
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of 'H' in the word.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the next position to process.
func (*PhoneticEncoder) handleH(word string, current int, primary *strings.Builder) int {
	if current == 0 || (current > 0 && isPhoneticVowel(word[current-1])) {
		if current+1 < len(word) && isPhoneticVowel(word[current+1]) {
			primary.WriteString("H")
			return current + 1
		}
	}

	return current + 1
}

// handleP processes the letter 'P' based on the letters that follow it.
//
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of 'P' in the word.
// Takes primary (*strings.Builder) which collects the encoded output.
//
// Returns int which is the next position to process.
func (*PhoneticEncoder) handleP(word string, current int, primary *strings.Builder) int {
	if current+1 < len(word) && word[current+1] == 'H' {
		primary.WriteString("F")
		return current + 2
	}

	if current+1 < len(word) && word[current+1] == 'P' {
		primary.WriteString("P")
		return current + 2
	}

	primary.WriteString("P")
	return current + 1
}

// handleS processes the letter 'S' based on the letters that follow it.
//
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of 'S' in the word.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the next position to process in the word.
func (*PhoneticEncoder) handleS(word string, current int, primary *strings.Builder) int {
	if current+1 < len(word) && word[current+1] == 'H' {
		primary.WriteString(PhoneticX)
		return current + 2
	}

	if current+2 < len(word) {
		if word[current+1] == 'I' && (word[current+2] == 'O' || word[current+2] == 'A') {
			primary.WriteString(PhoneticX)
			return current + sequenceLength3
		}
	}

	if current+1 < len(word) && word[current+1] == 'S' {
		primary.WriteString(PhoneticS)
		return current + 2
	}

	primary.WriteString("S")
	return current + 1
}

// handleT processes the letter 'T' based on the letters around it.
//
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of 'T' in the word.
// Takes primary (*strings.Builder) which receives the phonetic output.
//
// Returns int which is the next position to process.
func (*PhoneticEncoder) handleT(word string, current int, primary *strings.Builder) int {
	if current+2 < len(word) {
		if word[current+1] == 'I' && (word[current+2] == 'O' || word[current+2] == 'A') {
			primary.WriteString(PhoneticX)
			return current + sequenceLength3
		}
	}

	if current+1 < len(word) && word[current+1] == 'H' {
		primary.WriteString("0")
		return current + 2
	}

	if current+2 < len(word) && word[current+1] == 'C' && word[current+2] == 'H' {
		primary.WriteString(PhoneticX)
		return current + sequenceLength3
	}

	if current+1 < len(word) && word[current+1] == 'T' {
		primary.WriteString(PhoneticT)
		return current + 2
	}

	primary.WriteString(PhoneticT)
	return current + 1
}

// handleW processes the letter 'W' based on its position and nearby letters.
//
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of 'W' in the word.
// Takes primary (*strings.Builder) which collects the encoded output.
//
// Returns int which is the next position to process.
func (*PhoneticEncoder) handleW(word string, current int, primary *strings.Builder) int {
	if current == 0 && current+1 < len(word) && word[current+1] == 'H' {
		primary.WriteString("W")
		return current + 2
	}

	if current+1 < len(word) && isPhoneticVowel(word[current+1]) {
		primary.WriteString("W")
		return current + 1
	}

	return current + 1
}

// SoundexEncode encodes a word using the Soundex phonetic algorithm.
// Soundex is faster but less precise than Double Metaphone, and works well
// for encoding names.
//
// Takes word (string) which is the text to encode.
//
// Returns string which is the four-character Soundex code, or an empty
// string if the input is empty.
func SoundexEncode(word string) string {
	if len(word) == 0 {
		return ""
	}

	word = strings.Map(func(r rune) rune {
		return unicode.ToUpper(r)
	}, word)

	result := string(word[0])

	soundexMap := map[rune]rune{
		'B': '1', 'F': '1', 'P': '1', 'V': '1',
		'C': '2', 'G': '2', 'J': '2', 'K': '2', 'Q': '2', 'S': '2', 'X': '2', 'Z': '2',
		'D': '3', 'T': '3',
		'L': '4',
		'M': '5', 'N': '5',
		'R': '6',
	}

	var lastDigit rune
	for _, r := range word[1:] {
		digit, exists := soundexMap[r]
		if exists && digit != lastDigit {
			result += string(digit)
			lastDigit = digit
			if len(result) == soundexCodeLength {
				break
			}
		} else if !exists {
			lastDigit = 0
		}
	}

	for len(result) < soundexCodeLength {
		result += "0"
	}

	return result[:soundexCodeLength]
}

// Dispatch wrapper functions (adapt handlers to common signature)

// handleVowelDispatch dispatches the phonetic encoding for vowels.
//
// Takes _ (*PhoneticEncoder) which is unused by this handler.
// Takes _ (string) which is unused by this handler.
// Takes current (int) which is the current position in the text.
// Takes primary (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the updated position after processing.
func handleVowelDispatch(_ *PhoneticEncoder, _ string, current int, primary *strings.Builder) int {
	return handleVowel(current, primary)
}

// handleBDispatch dispatches the phonetic encoding for the letter B.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleBDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'B', "P", primary)
}

// handleFDispatch dispatches the phonetic encoding for the letter F.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleFDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'F', "F", primary)
}

// handleJDispatch dispatches the phonetic encoding for the letter J.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleJDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'J', "J", primary)
}

// handleKDispatch dispatches the phonetic encoding for the letter K.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleKDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'K', PhoneticK, primary)
}

// handleLDispatch dispatches the phonetic encoding for the letter L.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleLDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'L', "L", primary)
}

// handleMDispatch dispatches the phonetic encoding for the letter M.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleMDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'M', "M", primary)
}

// handleNDispatch dispatches the phonetic encoding for the letter N.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleNDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'N', "N", primary)
}

// handleQDispatch dispatches the phonetic encoding for the letter Q.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleQDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'Q', PhoneticK, primary)
}

// handleRDispatch dispatches the phonetic encoding for the letter R.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleRDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'R', "R", primary)
}

// handleVDispatch dispatches the phonetic encoding for the letter V.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleVDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'V', "F", primary)
}

// handleZDispatch dispatches the phonetic encoding for the letter Z.
//
// Takes word (string) which is the input word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleZDispatch(_ *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return handleSimpleDoublingLetter(word, current, 'Z', PhoneticS, primary)
}

// handleXDispatch dispatches the phonetic encoding for the letter X.
//
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the updated position after processing.
func handleXDispatch(_ *PhoneticEncoder, _ string, current int, primary *strings.Builder) int {
	primary.WriteString("KS")
	return current + 1
}

// handleCDispatch passes C character processing to the encoder's handleC
// method.
//
// Takes p (*PhoneticEncoder) which provides the encoding context.
// Takes word (string) which contains the text being encoded.
// Takes current (int) which specifies the position of the C character.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the new position after processing the C character.
func handleCDispatch(p *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return p.handleC(word, current, primary)
}

// handleDDispatch passes the letter D to the encoder for processing.
//
// Takes p (*PhoneticEncoder) which provides the encoding methods.
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of the D in the word.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the new position after processing.
func handleDDispatch(p *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return p.handleD(word, current, primary)
}

// handleGDispatch passes G character handling to the encoder's handleG method.
//
// Takes p (*PhoneticEncoder) which provides the encoding logic.
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of the G in the word.
// Takes primary (*strings.Builder) which collects the encoded output.
//
// Returns int which is the new position after processing the G.
func handleGDispatch(p *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return p.handleG(word, current, primary)
}

// handleHDispatch delegates H character handling to the encoder's handleH method.
//
// Takes p (*PhoneticEncoder) which provides the encoding logic.
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of the H character.
// Takes primary (*strings.Builder) which accumulates the encoded output.
//
// Returns int which is the new position after processing.
func handleHDispatch(p *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return p.handleH(word, current, primary)
}

// handlePDispatch dispatches P character handling to the encoder.
//
// Takes p (*PhoneticEncoder) which performs the phonetic encoding.
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the current position in the word.
// Takes primary (*strings.Builder) which accumulates the encoded output.
//
// Returns int which is the new position after processing.
func handlePDispatch(p *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return p.handleP(word, current, primary)
}

// handleSDispatch passes S sound handling to the encoder's handleS method.
//
// Takes p (*PhoneticEncoder) which provides the encoding logic.
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position in the word.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the new position after processing.
func handleSDispatch(p *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return p.handleS(word, current, primary)
}

// handleTDispatch delegates T character processing to the encoder's handleT
// method.
//
// Takes p (*PhoneticEncoder) which provides the encoding methods.
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of the T character.
// Takes primary (*strings.Builder) which accumulates the phonetic output.
//
// Returns int which is the new position after processing.
func handleTDispatch(p *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return p.handleT(word, current, primary)
}

// handleWDispatch handles the letter W by passing control to handleW.
//
// Takes p (*PhoneticEncoder) which provides the encoding context.
// Takes word (string) which is the word being encoded.
// Takes current (int) which is the position of the W character.
// Takes primary (*strings.Builder) which collects the encoded output.
//
// Returns int which is the next position to process.
func handleWDispatch(p *PhoneticEncoder, word string, current int, primary *strings.Builder) int {
	return p.handleW(word, current, primary)
}

// handleVowel processes vowel characters (A, E, I, O, U, Y).
// Vowels are only encoded if they appear at the start of the word.
//
// Takes current (int) which is the position of the vowel in the word.
// Takes primary (*strings.Builder) which collects the encoded output.
//
// Returns int which is the next position to process.
func handleVowel(current int, primary *strings.Builder) int {
	if current == 0 {
		primary.WriteString("A")
	}
	return current + 1
}

// handleSimpleDoublingLetter handles letters that produce a phonetic code and
// may appear twice in a row.
//
// This pattern applies to: B, F, J, K, L, M, N, Q, R, V, Z. When the same
// letter appears twice (such as "BB"), both are treated as one sound.
//
// Takes word (string) which is the word being processed.
// Takes current (int) which is the current position in the word.
// Takes letter (byte) which is the letter to check for doubling.
// Takes phoneticCode (string) which is the code to write for this letter.
// Takes primary (*strings.Builder) which collects the phonetic output.
//
// Returns int which is the next position to process. Returns current + 2 if
// the letter is doubled, or current + 1 otherwise.
func handleSimpleDoublingLetter(word string, current int, letter byte, phoneticCode string, primary *strings.Builder) int {
	primary.WriteString(phoneticCode)

	if current+1 < len(word) && word[current+1] == letter {
		return current + 2
	}

	return current + 1
}

// isSilentGN reports whether G is silent in -GN- or -GNED patterns.
//
// Takes word (string) which is the word to check.
// Takes current (int) which is the position of the G character.
//
// Returns bool which is true if the G at this position is silent.
func isSilentGN(word string, current int) bool {
	if current+1 >= len(word) || word[current+1] != 'N' {
		return false
	}
	return current+2 >= len(word) ||
		(current+sequenceLength3 < len(word) && word[current:current+sequenceLength4] == "GNED")
}

// isSoftG reports whether G should be pronounced as J (before E, I, or Y).
//
// Takes word (string) which is the word being checked.
// Takes current (int) which is the position of the G in the word.
//
// Returns bool which is true if the next character is E, I, or Y.
func isSoftG(word string, current int) bool {
	if current+1 >= len(word) {
		return false
	}
	next := word[current+1]
	return next == 'E' || next == 'I' || next == 'Y'
}

// isPhoneticVowel reports whether the given character is a vowel for phonetic
// purposes.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if character is A, E, I, O, U, or Y.
func isPhoneticVowel(character byte) bool {
	return character == 'A' || character == 'E' || character == 'I' || character == 'O' || character == 'U' || character == 'Y'
}
