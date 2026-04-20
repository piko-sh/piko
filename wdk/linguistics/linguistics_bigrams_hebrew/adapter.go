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

package linguistics_bigrams_hebrew

import (
	"unicode"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this bigram analyser.
const Language = "hebrew"

// minFieldLength is the minimum letter count for bigram analysis.
const minFieldLength = 4

// maxAnalyseLength is the maximum text byte length processed during analysis.
const maxAnalyseLength = 4096

var _ linguistics_domain.BigramAnalyserPort = (*BigramAnalyser)(nil)

// BigramAnalyser provides Hebrew character bigram frequency analysis
// for detecting gibberish or random text.
type BigramAnalyser struct{}

// BigramFrequencyRatio returns the ratio of uncommon character bigrams
// to total bigrams.
//
// Nikkud and other non-letter runes are discarded and the five Hebrew
// final-form letters are folded to their regular-form equivalents
// before the bigrams are built so that pointed, unpointed, and
// mid-word occurrences of the same consonant compare equal.
//
// Takes text (string) which is the text to analyse.
//
// Returns float64 which is the uncommon bigram ratio.
// Returns bool which is true when analysis was performed.
func (*BigramAnalyser) BigramFrequencyRatio(text string) (float64, bool) {
	if len(text) > maxAnalyseLength {
		text = text[:maxAnalyseLength]
	}
	letters := make([]rune, 0, len(text))
	for _, character := range text {
		if !unicode.IsLetter(character) {
			continue
		}
		letters = append(letters, foldFinalForm(character))
	}

	if len(letters) < minFieldLength {
		return 0, false
	}

	totalBigrams := 0
	uncommonBigrams := 0

	for index := 0; index < len(letters)-1; index++ {
		bigram := string(letters[index]) + string(letters[index+1])
		totalBigrams++
		if _, found := hebrewBigrams[bigram]; !found {
			uncommonBigrams++
		}
	}

	if totalBigrams == 0 {
		return 0, false
	}

	return float64(uncommonBigrams) / float64(totalBigrams), true
}

// GetLanguage returns the language this analyser is configured for.
//
// Returns string which is the language code.
func (*BigramAnalyser) GetLanguage() string {
	return Language
}

// Factory creates a new Hebrew bigram analyser instance.
//
// Returns linguistics_domain.BigramAnalyserPort which is the analyser.
// Returns error when creation fails.
func Factory() (linguistics_domain.BigramAnalyserPort, error) {
	return &BigramAnalyser{}, nil
}

// foldFinalForm returns the regular form of a Hebrew final-form
// letter, or the input unchanged when the rune is not a final form.
// The five affected letters are kaf, mem, nun, pe, and tsadi.
//
// Takes character (rune) which is the code point to fold.
//
// Returns rune which is the folded value.
func foldFinalForm(character rune) rune {
	switch character {
	case '\u05DA':
		return '\u05DB'
	case '\u05DD':
		return '\u05DE'
	case '\u05DF':
		return '\u05E0'
	case '\u05E3':
		return '\u05E4'
	case '\u05E5':
		return '\u05E6'
	default:
		return character
	}
}

func init() {
	linguistics_domain.RegisterBigramAnalyserFactory(Language, Factory)
}

// hebrewBigrams contains the most frequent Modern Hebrew letter bigrams.
//
// Entries use the regular (non-final) forms of kaf, mem, nun, pe, and
// tsadi because the analyser folds final forms to regular forms
// before lookup.
var hebrewBigrams = map[string]struct{}{
	"של": {}, "את": {}, "לא": {}, "על": {}, "כל": {},
	"או": {}, "ים": {}, "ות": {}, "הי": {}, "הא": {},
	"הב": {}, "הג": {}, "הד": {}, "הה": {}, "הו": {},
	"הח": {}, "הט": {}, "הכ": {}, "הל": {}, "המ": {},
	"הנ": {}, "הס": {}, "הע": {}, "הפ": {}, "הצ": {},
	"הק": {}, "הר": {}, "הש": {}, "הת": {}, "וא": {},
	"וב": {}, "וה": {}, "וי": {}, "ול": {}, "ומ": {},
	"ונ": {}, "וע": {}, "ור": {}, "יה": {}, "יו": {},
	"יכ": {}, "יי": {}, "יל": {}, "ינ": {}, "ית": {},
	"לה": {}, "לו": {}, "לי": {}, "לע": {}, "לפ": {},
	"מה": {}, "מו": {}, "מי": {}, "מנ": {}, "מע": {},
	"מש": {}, "נו": {}, "ני": {}, "נה": {}, "נת": {},
	"אה": {}, "אי": {}, "אל": {}, "אמ": {}, "אנ": {},
	"אס": {}, "אפ": {}, "אר": {}, "אש": {}, "אב": {},
	"אד": {}, "אח": {}, "בא": {}, "בד": {}, "בה": {},
	"בו": {}, "בח": {}, "בי": {}, "בכ": {}, "בל": {},
	"במ": {}, "בנ": {}, "בס": {}, "בע": {}, "בפ": {},
	"בק": {}, "בר": {}, "בש": {}, "בת": {}, "גד": {},
	"גר": {}, "דב": {}, "דה": {}, "דו": {}, "די": {},
	"דל": {}, "דע": {}, "דר": {}, "חד": {}, "חז": {},
	"חי": {}, "חכ": {}, "חל": {}, "חמ": {}, "חק": {},
	"חר": {}, "חש": {}, "חת": {}, "טו": {}, "כא": {},
	"כב": {}, "כד": {}, "כה": {}, "כו": {}, "כח": {},
	"כי": {}, "כמ": {}, "כנ": {}, "כפ": {}, "כר": {},
	"כש": {}, "כת": {}, "לב": {}, "לג": {}, "לד": {},
	"לח": {}, "לט": {}, "לכ": {}, "לל": {}, "למ": {},
	"לנ": {}, "לק": {}, "לר": {}, "לש": {}, "לת": {},
	"מא": {}, "מב": {}, "מג": {}, "מד": {}, "מח": {},
	"מט": {}, "מכ": {}, "מל": {}, "מפ": {}, "מצ": {},
	"מק": {}, "מר": {}, "מת": {}, "סי": {}, "סק": {},
	"סר": {}, "עב": {}, "עד": {}, "עה": {}, "עו": {},
	"עז": {}, "עי": {}, "עמ": {}, "עצ": {}, "עק": {},
	"ער": {}, "עש": {}, "עת": {}, "פה": {}, "פי": {},
	"פל": {}, "פנ": {}, "פר": {}, "פש": {}, "פת": {},
	"צי": {}, "צל": {}, "צר": {}, "קו": {}, "קי": {},
	"קר": {}, "קש": {}, "רא": {}, "רב": {}, "רה": {},
	"רו": {}, "רי": {}, "רכ": {}, "רע": {}, "רצ": {},
	"רק": {}, "שא": {}, "שב": {}, "שה": {}, "שי": {},
	"שמ": {}, "שנ": {}, "שפ": {}, "שר": {}, "שת": {},
	"תא": {}, "תב": {}, "תה": {}, "תו": {}, "תח": {},
	"תי": {}, "תק": {}, "תר": {},
}
