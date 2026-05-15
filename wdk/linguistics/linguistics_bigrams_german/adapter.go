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

package linguistics_bigrams_german

import (
	"strings"
	"unicode"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this bigram analyser.
const Language = "german"

// minFieldLength is the minimum letter count for bigram analysis.
const minFieldLength = 4

// maxAnalyseLength is the maximum text byte length processed during analysis.
const maxAnalyseLength = 4096

var _ linguistics_domain.BigramAnalyserPort = (*BigramAnalyser)(nil)

// BigramAnalyser provides German character bigram frequency analysis
// for detecting gibberish or random text.
type BigramAnalyser struct{}

// BigramFrequencyRatio returns the ratio of uncommon character bigrams
// to total bigrams.
//
// Takes text (string) which is the text to analyse.
//
// Returns float64 which is the uncommon bigram ratio.
// Returns bool which is true when analysis was performed.
func (*BigramAnalyser) BigramFrequencyRatio(text string) (float64, bool) {
	if len(text) > maxAnalyseLength {
		text = text[:maxAnalyseLength]
	}
	lower := strings.ToLower(text)
	letters := make([]rune, 0, len(lower))
	for _, r := range lower {
		if unicode.IsLetter(r) {
			letters = append(letters, r)
		}
	}

	if len(letters) < minFieldLength {
		return 0, false
	}

	totalBigrams := 0
	uncommonBigrams := 0

	for index := 0; index < len(letters)-1; index++ {
		bigram := string(letters[index]) + string(letters[index+1])
		totalBigrams++
		if _, found := germanBigrams[bigram]; !found {
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

// Factory creates a new German bigram analyser instance.
//
// Returns linguistics_domain.BigramAnalyserPort which is the analyser.
// Returns error when creation fails.
func Factory() (linguistics_domain.BigramAnalyserPort, error) {
	return &BigramAnalyser{}, nil
}

func init() {
	linguistics_domain.RegisterBigramAnalyserFactory(Language, Factory)
}

// germanBigrams contains the most frequent German letter bigrams.
var germanBigrams = map[string]struct{}{
	"en": {}, "er": {}, "ch": {}, "de": {}, "ei": {},
	"te": {}, "nd": {}, "in": {}, "ie": {}, "ge": {},
	"st": {}, "un": {}, "ic": {}, "re": {}, "an": {},
	"be": {}, "he": {}, "le": {}, "ne": {}, "se": {},
	"it": {}, "me": {}, "es": {}, "ve": {}, "nt": {},
	"sc": {}, "au": {}, "ng": {}, "rt": {}, "ru": {},
	"tz": {}, "ur": {}, "ut": {}, "la": {}, "li": {},
	"ha": {}, "da": {}, "ti": {}, "al": {}, "on": {},
	"ab": {}, "hr": {}, "as": {}, "ra": {}, "or": {},
	"si": {}, "um": {}, "sp": {}, "wi": {}, "el": {},
	"no": {}, "gr": {}, "ze": {}, "eb": {}, "lt": {},
	"so": {}, "ar": {}, "ah": {}, "rs": {}, "rd": {},
	"rn": {}, "tu": {}, "to": {}, "sa": {}, "ol": {},
	"om": {}, "ni": {}, "bi": {}, "em": {}, "ns": {},
	"ke": {}, "ed": {}, "ig": {}, "il": {}, "is": {},
	"os": {}, "ss": {}, "ts": {}, "eh": {}, "ew": {},
	"fe": {}, "fo": {}, "fr": {}, "hi": {}, "ho": {},
	"ir": {}, "ka": {}, "ko": {}, "ki": {}, "lo": {},
	"ma": {}, "mi": {}, "mo": {}, "mu": {}, "na": {},
	"sw": {}, "ta": {}, "vo": {}, "we": {}, "wo": {},
	"ue": {}, "üb": {}, "ag": {}, "ac": {}, "ad": {},
	"af": {}, "ak": {}, "am": {}, "ap": {}, "at": {},
	"aß": {}, "ba": {}, "bl": {}, "br": {}, "bu": {},
	"ca": {}, "ce": {}, "co": {}, "cr": {}, "cu": {},
	"di": {}, "do": {}, "du": {}, "ea": {}, "ec": {},
	"eg": {}, "ek": {}, "ep": {}, "eu": {}, "ev": {},
	"ex": {}, "ey": {}, "fa": {}, "fi": {}, "fl": {},
	"fu": {}, "ga": {}, "gl": {}, "go": {}, "gu": {},
	"ib": {}, "id": {}, "if": {}, "ih": {}, "im": {},
	"io": {}, "ip": {}, "ja": {}, "je": {}, "jo": {},
	"ju": {}, "kl": {}, "kn": {}, "kr": {}, "ku": {},
	"ld": {}, "ll": {}, "lm": {}, "lp": {}, "ls": {},
	"ly": {}, "mb": {}, "mm": {}, "mp": {}, "ms": {},
	"nc": {}, "nk": {}, "nz": {}, "oa": {}, "ob": {},
	"od": {}, "of": {}, "og": {}, "oh": {}, "oi": {},
	"ok": {}, "op": {}, "ot": {}, "ov": {}, "ow": {},
	"pa": {}, "pe": {}, "ph": {}, "pi": {},
}
