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

import "strings"

const (
	// digraphLength is the number of runes consumed by a letter plus
	// a single combining mark or by a collapsed double consonant.
	digraphLength = 2

	// endOfRunes is the sentinel returned by nextRune when the caller
	// asks past the last rune of the input slice. It is chosen to be
	// impossible as a valid Unicode code point.
	endOfRunes rune = -1
)

const (
	// hebAlef is the Hebrew letter alef (U+05D0). Alef is silent or
	// marks a glottal stop.
	hebAlef = '\u05D0'

	// hebBet is the Hebrew letter bet (U+05D1). Realised as B with
	// dagesh or word-initially, V elsewhere.
	hebBet = '\u05D1'

	// hebGimel is the Hebrew letter gimel (U+05D2). A geresh can turn
	// it into the affricate DJ in loanwords.
	hebGimel = '\u05D2'

	// hebDalet is the Hebrew letter dalet (U+05D3). A geresh can turn
	// it into DH in Arabic-origin loanwords.
	hebDalet = '\u05D3'

	// hebHe is the Hebrew letter he (U+05D4). Audible at word start,
	// silent medially.
	hebHe = '\u05D4'

	// hebVav is the Hebrew letter vav (U+05D5). Treated as a consonant
	// V in modern Israeli Hebrew.
	hebVav = '\u05D5'

	// hebZayin is the Hebrew letter zayin (U+05D6). A geresh turns it
	// into ZH in loanwords.
	hebZayin = '\u05D6'

	// hebChet is the Hebrew letter chet (U+05D7). Merged with khaf as
	// the velar fricative X in modern speech.
	hebChet = '\u05D7'

	// hebTet is the Hebrew letter tet (U+05D8). Merged with tav as T
	// in modern speech.
	hebTet = '\u05D8'

	// hebYod is the Hebrew letter yod (U+05D9). Consonantal J.
	hebYod = '\u05D9'

	// hebFinalKaf is the final form of kaf (U+05DA). Always the
	// fricative X.
	hebFinalKaf = '\u05DA'

	// hebKaf is the Hebrew letter kaf (U+05DB). Realised as K with
	// dagesh or word-initially, X elsewhere.
	hebKaf = '\u05DB'

	// hebLamed is the Hebrew letter lamed (U+05DC).
	hebLamed = '\u05DC'

	// hebFinalMem is the final form of mem (U+05DD).
	hebFinalMem = '\u05DD'

	// hebMem is the Hebrew letter mem (U+05DE).
	hebMem = '\u05DE'

	// hebFinalNun is the final form of nun (U+05DF).
	hebFinalNun = '\u05DF'

	// hebNun is the Hebrew letter nun (U+05E0).
	hebNun = '\u05E0'

	// hebSamekh is the Hebrew letter samekh (U+05E1). Merged with sin
	// as S.
	hebSamekh = '\u05E1'

	// hebAyin is the Hebrew letter ayin (U+05E2). Silent or a
	// pharyngeal sound usually dropped in modern speech.
	hebAyin = '\u05E2'

	// hebFinalPe is the final form of pe (U+05E3). Always the
	// fricative F.
	hebFinalPe = '\u05E3'

	// hebPe is the Hebrew letter pe (U+05E4). With dagesh P, without
	// F.
	hebPe = '\u05E4'

	// hebFinalTsadi is the final form of tsadi (U+05E5).
	hebFinalTsadi = '\u05E5'

	// hebTsadi is the Hebrew letter tsadi (U+05E6). A geresh turns it
	// into CH in loanwords.
	hebTsadi = '\u05E6'

	// hebQof is the Hebrew letter qof (U+05E7). Merged with kaf as K.
	hebQof = '\u05E7'

	// hebResh is the Hebrew letter resh (U+05E8).
	hebResh = '\u05E8'

	// hebShin is the Hebrew letter shin (U+05E9). With a shin dot it
	// is SH; with a sin dot it is S; unpointed it defaults to S.
	hebShin = '\u05E9'

	// hebTav is the Hebrew letter tav (U+05EA). Merged with tet as T.
	hebTav = '\u05EA'
)

const (
	// dagesh is the combining mark (U+05BC) that hardens begadkephat
	// letters.
	dagesh = '\u05BC'

	// shinDot is the combining mark (U+05C1) that identifies the shin
	// sound on the letter shin.
	shinDot = '\u05C1'

	// sinDot is the combining mark (U+05C2) that identifies the sin
	// sound on the letter shin.
	sinDot = '\u05C2'

	// geresh is the apostrophe-like mark (U+05F3) used to indicate
	// loanword affricates following gimel, zayin, tsadi, and dalet.
	geresh = '\u05F3'
)

const (
	// phoneticB is the hard bilabial stop code for bet with dagesh.
	phoneticB = "B"

	// phoneticV is the fricative code for vav and soft bet.
	phoneticV = "V"

	// phoneticG is the voiced velar stop code for gimel.
	phoneticG = "G"

	// phoneticDJ is the affricate code for gimel followed by geresh
	// in loanwords.
	phoneticDJ = "DJ"

	// phoneticD is the voiced alveolar stop code for dalet.
	phoneticD = "D"

	// phoneticDH is the voiced alveolar affricate code for dalet
	// followed by geresh in Arabic-origin loanwords.
	phoneticDH = "DH"

	// phoneticH is the voiceless glottal fricative code for word
	// initial he.
	phoneticH = "H"

	// phoneticZ is the voiced alveolar sibilant code for zayin.
	phoneticZ = "Z"

	// phoneticZH is the voiced post-alveolar fricative code for
	// zayin with geresh in loanwords.
	phoneticZH = "ZH"

	// phoneticX is the voiceless velar fricative code shared by chet
	// and soft kaf.
	phoneticX = "X"

	// phoneticT is the voiceless alveolar stop code shared by tet
	// and tav.
	phoneticT = "T"

	// phoneticJ is the palatal approximant code for yod.
	phoneticJ = "J"

	// phoneticK is the voiceless velar stop code for hard kaf and
	// qof.
	phoneticK = "K"

	// phoneticL is the alveolar lateral code for lamed.
	phoneticL = "L"

	// phoneticM is the bilabial nasal code for mem.
	phoneticM = "M"

	// phoneticN is the alveolar nasal code for nun.
	phoneticN = "N"

	// phoneticS is the voiceless alveolar sibilant code shared by
	// samekh and sin.
	phoneticS = "S"

	// phoneticSH is the voiceless post-alveolar sibilant code for
	// shin with a shin dot.
	phoneticSH = "SH"

	// phoneticF is the voiceless labiodental fricative code for soft
	// pe and final pe.
	phoneticF = "F"

	// phoneticP is the voiceless bilabial stop code for pe with
	// dagesh.
	phoneticP = "P"

	// phoneticTS is the voiceless alveolar affricate code for tsadi.
	phoneticTS = "TS"

	// phoneticCH is the voiceless post-alveolar affricate code for
	// tsadi with geresh in loanwords.
	phoneticCH = "CH"

	// phoneticR is the rhotic code for resh.
	phoneticR = "R"
)

// nextRune returns the rune immediately following position, or the
// endOfRunes sentinel when the lookahead would run past the end of
// the slice.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
//
// Returns rune which is the lookahead rune.
func nextRune(runes []rune, position int) rune {
	if position+1 >= len(runes) {
		return endOfRunes
	}
	return runes[position+1]
}

// hasDagesh reports whether the rune following position is a dagesh
// combining mark.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
//
// Returns bool which is true when a dagesh follows.
func hasDagesh(runes []rune, position int) bool {
	return nextRune(runes, position) == dagesh
}

// hasGeresh reports whether the rune following position is a geresh
// mark.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
//
// Returns bool which is true when a geresh follows.
func hasGeresh(runes []rune, position int) bool {
	return nextRune(runes, position) == geresh
}

// writeAndSkipMark appends the code to result and advances past the
// current rune and one consumed combining mark.
//
// Takes position (int) which is the index of the current rune.
// Takes code (string) which is the phonetic output.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func writeAndSkipMark(position int, code string, result *strings.Builder) int {
	_, _ = result.WriteString(code)
	return position + digraphLength
}

// handleDoubleConsonant appends the code and collapses an immediately
// repeated identical consonant.
//
// Takes runes ([]rune) which is the word under analysis.
// Takes position (int) which is the index of the current rune.
// Takes letter (rune) which is the consonant to check for doubling.
// Takes code (string) which is the phonetic output.
// Takes result (*strings.Builder) which accumulates the phonetic
// output.
//
// Returns int which is the next position to process.
func handleDoubleConsonant(
	runes []rune,
	position int,
	letter rune,
	code string,
	result *strings.Builder,
) int {
	_, _ = result.WriteString(code)
	if nextRune(runes, position) == letter {
		return position + digraphLength
	}
	return position + 1
}
