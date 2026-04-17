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

package linguistics_bigrams_dutch

import (
	"strings"
	"unicode"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this bigram analyser.
const Language = "dutch"

// minFieldLength is the minimum letter count for bigram analysis.
const minFieldLength = 4

// maxAnalyseLength is the maximum text byte length processed during analysis.
const maxAnalyseLength = 4096

var _ linguistics_domain.BigramAnalyserPort = (*BigramAnalyser)(nil)

// BigramAnalyser provides Dutch character bigram frequency analysis
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
		if _, found := dutchBigrams[bigram]; !found {
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

// Factory creates a new Dutch bigram analyser instance.
//
// Returns linguistics_domain.BigramAnalyserPort which is the analyser.
// Returns error when creation fails.
func Factory() (linguistics_domain.BigramAnalyserPort, error) {
	return &BigramAnalyser{}, nil
}

func init() {
	linguistics_domain.RegisterBigramAnalyserFactory(Language, Factory)
}

// dutchBigrams contains the most frequent Dutch letter bigrams.
var dutchBigrams = map[string]struct{}{
	"en": {}, "de": {}, "er": {}, "ee": {}, "te": {},
	"an": {}, "ge": {}, "et": {}, "aa": {}, "nd": {},
	"in": {}, "oo": {}, "ij": {}, "ve": {}, "el": {},
	"or": {}, "re": {}, "on": {}, "st": {}, "ie": {},
	"ch": {}, "al": {}, "rd": {}, "nt": {}, "he": {},
	"le": {}, "at": {}, "es": {}, "va": {}, "oe": {},
	"se": {}, "ka": {}, "ni": {}, "ht": {}, "nn": {},
	"ti": {}, "ri": {}, "is": {}, "ke": {}, "it": {},
	"me": {}, "ta": {}, "li": {}, "la": {}, "ik": {},
	"na": {}, "ar": {}, "ng": {}, "be": {}, "to": {},
	"da": {}, "ma": {}, "wa": {}, "we": {}, "ne": {},
	"ra": {}, "ol": {}, "ep": {}, "om": {}, "po": {},
	"ce": {}, "ze": {}, "ec": {}, "ns": {}, "tw": {},
	"ov": {}, "ki": {}, "ev": {}, "hi": {}, "je": {},
	"wi": {}, "pa": {}, "si": {}, "hu": {}, "pr": {},
	"ro": {}, "il": {}, "ts": {}, "ur": {}, "pe": {},
	"do": {}, "ig": {}, "ui": {}, "un": {}, "us": {},
	"tr": {}, "mi": {}, "sa": {}, "em": {}, "wo": {},
	"vo": {}, "pp": {}, "tt": {}, "au": {}, "eu": {},
	"ou": {}, "sc": {}, "zi": {}, "ed": {}, "dr": {},
	"ef": {}, "eg": {}, "ek": {}, "ez": {}, "ba": {},
	"bi": {}, "bl": {}, "br": {}, "bu": {}, "ca": {},
	"ci": {}, "cl": {}, "co": {}, "cr": {}, "fa": {},
	"fe": {}, "fi": {}, "fl": {}, "fo": {}, "fr": {},
	"fu": {}, "ga": {}, "gi": {}, "gl": {}, "gr": {},
	"gu": {}, "ha": {}, "ho": {}, "ja": {}, "jo": {},
	"ju": {}, "kl": {}, "kn": {}, "kr": {}, "ku": {},
	"lo": {}, "lu": {}, "ly": {}, "mo": {}, "mu": {},
	"ny": {}, "pi": {}, "pl": {}, "pu": {}, "ru": {},
	"sk": {}, "sl": {}, "sm": {}, "sn": {}, "sp": {},
	"su": {}, "sw": {}, "sy": {}, "ty": {}, "vi": {},
	"vr": {}, "vu": {}, "wr": {}, "ya": {}, "yo": {},
	"za": {}, "zo": {}, "zu": {}, "zw": {}, "oi": {},
	"ua": {}, "yl": {}, "yp": {}, "xe": {}, "sf": {},
	"sr": {}, "sh": {}, "sq": {}, "qu": {}, "pj": {},
	"pn": {}, "of": {}, "oc": {}, "ob": {}, "od": {},
	"ok": {}, "oy": {}, "iz": {}, "ix": {}, "if": {},
	"io": {}, "ip": {}, "iv": {}, "kh": {}, "ky": {},
	"ld": {}, "lf": {}, "lg": {}, "ls": {},
}
