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

package linguistics_phonetic_hebrew

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

const (
	// Language is the language code for this encoder.
	Language = "hebrew"

	// DefaultMaxLength is the default maximum length for Hebrew
	// phonetic codes. The 8-character default accommodates trisyllabic
	// roots that contain two-character digraphs such as SH or DJ.
	DefaultMaxLength = 8

	// hebrewAlphabetSize is the count of consonant code points in
	// the main Hebrew block from alef to tav.
	hebrewAlphabetSize = 27

	// hebrewBaseRune is the first letter of the Hebrew consonant
	// block.
	hebrewBaseRune = hebAlef

	// markBlockStart is the first code point in the Hebrew combining
	// mark and punctuation range.
	markBlockStart = '\u0591'

	// markBlockEnd is the last code point in the Hebrew combining
	// mark and punctuation range.
	markBlockEnd = '\u05F4'

	// yiddishDoubleVav is the precomposed Yiddish ligature (U+05F0)
	// for two consecutive vav letters.
	yiddishDoubleVav = '\u05F0'

	// yiddishVavYod is the precomposed Yiddish ligature (U+05F1) for
	// vav followed by yod.
	yiddishVavYod = '\u05F1'

	// yiddishDoubleYod is the precomposed Yiddish ligature (U+05F2)
	// for two consecutive yod letters.
	yiddishDoubleYod = '\u05F2'
)

// runeHandler processes a rune at the given position and returns the
// next position to process.
type runeHandler func(runes []rune, position int, result *strings.Builder) int

// hebrewHandlers is a dispatch table for Hebrew character processing,
// indexed by (character - hebrewBaseRune) so that alef is 0 and tav
// is 26.
var hebrewHandlers = [hebrewAlphabetSize]runeHandler{
	0:  handleAlef,
	1:  handleBet,
	2:  handleGimel,
	3:  handleDalet,
	4:  handleHe,
	5:  handleVav,
	6:  handleZayin,
	7:  handleChet,
	8:  handleTet,
	9:  handleYod,
	10: handleFinalKaf,
	11: handleKaf,
	12: handleLamed,
	13: handleFinalMem,
	14: handleMem,
	15: handleFinalNun,
	16: handleNun,
	17: handleSamekh,
	18: handleAyin,
	19: handleFinalPe,
	20: handlePe,
	21: handleFinalTsadi,
	22: handleTsadi,
	23: handleQof,
	24: handleResh,
	25: handleShin,
	26: handleTav,
}

// Encoder provides Hebrew phonetic encoding using a dispatch table.
// It implements linguistics_domain.PhoneticEncoderPort.
type Encoder struct {
	// maxLength is the maximum number of characters in the output
	// code.
	maxLength int
}

// NewWithMaxLength creates a new Hebrew phonetic encoder with a custom
// maximum code length. Values below or equal to zero fall back to
// DefaultMaxLength.
//
// Takes maxLength (int) which bounds the output code.
//
// Returns *Encoder which is ready for use.
// Returns error which is always nil for this encoder.
func NewWithMaxLength(maxLength int) (*Encoder, error) {
	if maxLength <= 0 {
		maxLength = DefaultMaxLength
	}
	return &Encoder{maxLength: maxLength}, nil
}

// New creates a new Hebrew phonetic encoder using the default maximum
// length.
//
// Returns *Encoder which is ready for use.
// Returns error which is always nil for this encoder.
func New() (*Encoder, error) {
	return NewWithMaxLength(DefaultMaxLength)
}

// Encode returns the phonetic encoding of a Hebrew word. Empty input
// yields an empty code.
//
// Takes word (string) which is the word to encode, optionally with
// nikkud.
//
// Returns string which is the phonetic code truncated to the encoder's
// maximum length.
func (e *Encoder) Encode(word string) string {
	if word == "" {
		return ""
	}

	runes := []rune(preserveSemanticMarks(word))

	var result strings.Builder
	position := 0
	for position < len(runes) && result.Len() < e.maxLength {
		position = processCharacter(runes, position, &result)
	}

	code := result.String()
	if len(code) <= e.maxLength {
		return code
	}
	return truncateToRuneBoundary(code, e.maxLength)
}

// truncateToRuneBoundary returns a prefix of code that is at most
// maxBytes long and never splits a UTF-8 rune. A naive byte slice at
// maxBytes could leave a partial rune at the tail, producing invalid
// UTF-8 output that would corrupt downstream consumers such as search
// indexes or JSON encoders.
//
// Takes code (string) which is the candidate output that has already
// exceeded the byte limit.
// Takes maxBytes (int) which is the hard ceiling on output length.
//
// Returns string which is the longest rune-aligned prefix within the
// byte limit.
func truncateToRuneBoundary(code string, maxBytes int) string {
	for limit := maxBytes; limit > 0; limit-- {
		if utf8.ValidString(code[:limit]) {
			return code[:limit]
		}
	}
	return ""
}

// GetLanguage returns the language this encoder is configured for.
//
// Returns string which is the language code.
func (*Encoder) GetLanguage() string {
	return Language
}

var _ linguistics_domain.PhoneticEncoderPort = (*Encoder)(nil)

// Factory creates a new Hebrew phonetic encoder instance.
//
// Use this with linguistics_domain.RegisterPhoneticEncoderFactory for
// explicit registration.
//
// Returns linguistics_domain.PhoneticEncoderPort which is the encoder
// instance.
// Returns error when the encoder cannot be created.
func Factory() (linguistics_domain.PhoneticEncoderPort, error) {
	return New()
}

func init() {
	linguistics_domain.RegisterPhoneticEncoderFactory(Language, Factory)
}

// preserveSemanticMarks removes cantillation marks and vowel points
// while keeping the combining marks that carry phonemic information
// (dagesh, shin dot, sin dot) and the geresh used for loanwords.
//
// Takes word (string) which may include Hebrew diacritical marks.
//
// Returns string with only the semantically relevant marks retained.
func preserveSemanticMarks(word string) string {
	var builder strings.Builder
	builder.Grow(len(word))
	for _, character := range word {
		if isHebrewMark(character) && !isSemanticMark(character) {
			continue
		}
		_, _ = builder.WriteRune(character)
	}
	return builder.String()
}

// isHebrewMark reports whether a rune is a Hebrew combining mark or
// punctuation sign. Hebrew consonants and the Yiddish ligature block
// (U+05EF through U+05F2) are excluded so that letters are never
// treated as marks.
//
// Takes character (rune) which is the rune to classify.
//
// Returns bool which is true when the rune is within U+0591 through
// U+05F4 but is not a letter.
func isHebrewMark(character rune) bool {
	if character < markBlockStart || character > markBlockEnd {
		return false
	}
	if unicode.IsLetter(character) {
		return false
	}
	return true
}

// isSemanticMark reports whether a rune is a combining mark that the
// encoder needs in order to make phonemic decisions.
//
// Takes character (rune) which is the rune to classify.
//
// Returns bool which is true for dagesh, shin dot, sin dot, and
// geresh.
func isSemanticMark(character rune) bool {
	switch character {
	case dagesh, shinDot, sinDot, geresh:
		return true
	default:
		return false
	}
}

// processCharacter dispatches handling for a single rune using the
// array dispatch table, expanding Yiddish ligatures into their
// underlying letter sequences and falling back to writing
// mixed-script letters directly.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func processCharacter(runes []rune, position int, result *strings.Builder) int {
	character := runes[position]

	if character >= hebrewBaseRune && character < hebrewBaseRune+hebrewAlphabetSize {
		index := int(character - hebrewBaseRune)
		if handler := hebrewHandlers[index]; handler != nil {
			return handler(runes, position, result)
		}
	}

	if next, ok := expandYiddishLigature(character); ok {
		_, _ = result.WriteString(next)
		return position + 1
	}

	if isHebrewMark(character) {
		return position + 1
	}

	if unicode.IsLetter(character) {
		_, _ = result.WriteRune(unicode.ToUpper(character))
	}
	return position + 1
}

// expandYiddishLigature returns the phonetic code for a precomposed
// Yiddish ligature in the U+05F0 through U+05F2 block.
//
// Takes character (rune) which is the candidate ligature rune.
//
// Returns string which is the phonetic output for the ligature.
// Returns bool which is true when the rune was a recognised
// ligature.
func expandYiddishLigature(character rune) (string, bool) {
	switch character {
	case yiddishDoubleVav:
		return phoneticV, true
	case yiddishVavYod:
		return phoneticV + phoneticJ, true
	case yiddishDoubleYod:
		return phoneticJ, true
	default:
		return "", false
	}
}

// handleAlef emits no output for alef; the letter is silent in modern
// Israeli Hebrew.
//
// Takes position (int) which is the index of the current rune.
//
// Returns int which is the next position to process.
func handleAlef(_ []rune, position int, _ *strings.Builder) int {
	return position + 1
}

// handleBet emits the hard stop B when a dagesh follows or when the
// letter is word-initial, and the fricative V otherwise.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleBet(runes []rune, position int, result *strings.Builder) int {
	if hasDagesh(runes, position) {
		return writeAndSkipMark(position, phoneticB, result)
	}
	if position == 0 {
		return handleDoubleConsonant(runes, position, hebBet, phoneticB, result)
	}
	return handleDoubleConsonant(runes, position, hebBet, phoneticV, result)
}

// handleGimel emits DJ when followed by a geresh (loanword notation)
// and G otherwise.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleGimel(runes []rune, position int, result *strings.Builder) int {
	if hasGeresh(runes, position) {
		return writeAndSkipMark(position, phoneticDJ, result)
	}
	return handleDoubleConsonant(runes, position, hebGimel, phoneticG, result)
}

// handleDalet emits DH when followed by a geresh (Arabic-origin
// loanwords) and D otherwise.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleDalet(runes []rune, position int, result *strings.Builder) int {
	if hasGeresh(runes, position) {
		return writeAndSkipMark(position, phoneticDH, result)
	}
	return handleDoubleConsonant(runes, position, hebDalet, phoneticD, result)
}

// handleHe emits H when the letter is word-initial and nothing when
// it appears medially or finally.
//
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleHe(_ []rune, position int, result *strings.Builder) int {
	if position == 0 {
		_, _ = result.WriteString(phoneticH)
	}
	return position + 1
}

// handleVav emits V.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleVav(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebVav, phoneticV, result)
}

// handleZayin emits ZH when followed by a geresh and Z otherwise.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleZayin(runes []rune, position int, result *strings.Builder) int {
	if hasGeresh(runes, position) {
		return writeAndSkipMark(position, phoneticZH, result)
	}
	return handleDoubleConsonant(runes, position, hebZayin, phoneticZ, result)
}

// handleChet emits the velar fricative X.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleChet(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebChet, phoneticX, result)
}

// handleTet emits T.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleTet(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebTet, phoneticT, result)
}

// handleYod emits J.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleYod(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebYod, phoneticJ, result)
}

// handleFinalKaf emits the velar fricative X; the final form is
// always soft.
//
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleFinalKaf(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(phoneticX)
	return position + 1
}

// handleKaf emits K when hardened by dagesh or initial position and X
// otherwise.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleKaf(runes []rune, position int, result *strings.Builder) int {
	if hasDagesh(runes, position) {
		return writeAndSkipMark(position, phoneticK, result)
	}
	if position == 0 {
		return handleDoubleConsonant(runes, position, hebKaf, phoneticK, result)
	}
	return handleDoubleConsonant(runes, position, hebKaf, phoneticX, result)
}

// handleLamed emits L.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleLamed(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebLamed, phoneticL, result)
}

// handleFinalMem emits M.
//
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleFinalMem(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(phoneticM)
	return position + 1
}

// handleMem emits M.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleMem(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebMem, phoneticM, result)
}

// handleFinalNun emits N.
//
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleFinalNun(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(phoneticN)
	return position + 1
}

// handleNun emits N.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleNun(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebNun, phoneticN, result)
}

// handleSamekh emits S.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleSamekh(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebSamekh, phoneticS, result)
}

// handleAyin emits no output; ayin is silent in modern Israeli
// Hebrew.
//
// Takes position (int) which is the index of the current rune.
//
// Returns int which is the next position to process.
func handleAyin(_ []rune, position int, _ *strings.Builder) int {
	return position + 1
}

// handleFinalPe emits the fricative F; the final form is always soft.
//
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleFinalPe(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(phoneticF)
	return position + 1
}

// handlePe emits P with dagesh or word-initial position and F
// otherwise.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handlePe(runes []rune, position int, result *strings.Builder) int {
	if hasDagesh(runes, position) {
		return writeAndSkipMark(position, phoneticP, result)
	}
	if position == 0 {
		return handleDoubleConsonant(runes, position, hebPe, phoneticP, result)
	}
	return handleDoubleConsonant(runes, position, hebPe, phoneticF, result)
}

// handleFinalTsadi emits TS.
//
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleFinalTsadi(_ []rune, position int, result *strings.Builder) int {
	_, _ = result.WriteString(phoneticTS)
	return position + 1
}

// handleTsadi emits CH when followed by a geresh and TS otherwise.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleTsadi(runes []rune, position int, result *strings.Builder) int {
	if hasGeresh(runes, position) {
		return writeAndSkipMark(position, phoneticCH, result)
	}
	return handleDoubleConsonant(runes, position, hebTsadi, phoneticTS, result)
}

// handleQof emits K; qof has merged with kaf in modern Israeli
// pronunciation.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleQof(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebQof, phoneticK, result)
}

// handleResh emits R.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleResh(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebResh, phoneticR, result)
}

// handleShin emits SH with a shin dot, S with a sin dot, and S as the
// default when unpointed.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleShin(runes []rune, position int, result *strings.Builder) int {
	switch nextRune(runes, position) {
	case shinDot:
		return writeAndSkipMark(position, phoneticSH, result)
	case sinDot:
		return writeAndSkipMark(position, phoneticS, result)
	}
	return handleDoubleConsonant(runes, position, hebShin, phoneticS, result)
}

// handleTav emits T; tav has merged with tet in modern Israeli
// pronunciation.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleTav(runes []rune, position int, result *strings.Builder) int {
	return handleDoubleConsonant(runes, position, hebTav, phoneticT, result)
}
