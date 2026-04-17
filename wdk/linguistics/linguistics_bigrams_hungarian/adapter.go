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

package linguistics_bigrams_hungarian

import (
	"strings"
	"unicode"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this bigram analyser.
const Language = "hungarian"

// minFieldLength is the minimum letter count for bigram analysis.
const minFieldLength = 4

// maxAnalyseLength is the maximum text byte length processed during analysis.
const maxAnalyseLength = 4096

var _ linguistics_domain.BigramAnalyserPort = (*BigramAnalyser)(nil)

// BigramAnalyser provides Hungarian character bigram frequency analysis
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
		if _, found := hungarianBigrams[bigram]; !found {
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

// Factory creates a new Hungarian bigram analyser instance.
//
// Returns linguistics_domain.BigramAnalyserPort which is the analyser.
// Returns error when creation fails.
func Factory() (linguistics_domain.BigramAnalyserPort, error) {
	return &BigramAnalyser{}, nil
}

func init() {
	linguistics_domain.RegisterBigramAnalyserFactory(Language, Factory)
}

// hungarianBigrams contains the most frequent Hungarian letter bigrams.
var hungarianBigrams = map[string]struct{}{
	"el": {}, "et": {}, "sz": {}, "er": {}, "en": {},
	"te": {}, "me": {}, "ek": {}, "ak": {}, "eg": {},
	"gy": {}, "le": {}, "es": {}, "be": {}, "ne": {},
	"an": {}, "re": {}, "ra": {}, "on": {}, "al": {},
	"ar": {}, "as": {}, "eb": {}, "at": {}, "nd": {},
	"nt": {}, "or": {}, "nk": {}, "ol": {}, "ta": {},
	"to": {}, "ke": {}, "ve": {}, "ze": {}, "de": {},
	"ha": {}, "ba": {}, "ly": {}, "ge": {}, "gi": {},
	"he": {}, "il": {}, "is": {}, "it": {}, "ja": {},
	"je": {}, "ka": {}, "ko": {}, "la": {}, "li": {},
	"lo": {}, "lu": {}, "ma": {}, "mi": {}, "mo": {},
	"mu": {}, "na": {}, "ni": {}, "no": {}, "od": {},
	"ok": {}, "om": {}, "os": {}, "ot": {}, "pe": {},
	"po": {}, "ro": {}, "ru": {}, "se": {}, "si": {},
	"so": {}, "su": {}, "sa": {}, "ti": {}, "tu": {},
	"tt": {}, "ud": {}, "ug": {}, "ul": {}, "un": {},
	"ur": {}, "us": {}, "ut": {}, "va": {}, "vi": {},
	"vo": {}, "za": {}, "zi": {}, "zo": {}, "ce": {},
	"ci": {}, "cs": {}, "ny": {}, "ty": {}, "zs": {},
	"rt": {}, "rs": {}, "rn": {}, "rm": {}, "rl": {},
	"rk": {}, "rd": {}, "rc": {}, "rb": {}, "nn": {},
	"mm": {}, "ll": {}, "kk": {}, "ss": {}, "pp": {},
	"bb": {}, "gg": {}, "dd": {}, "cc": {}, "ff": {},
	"nc": {}, "nb": {}, "nf": {}, "ng": {}, "nh": {},
	"nj": {}, "nl": {}, "nm": {}, "np": {}, "nr": {},
	"ns": {}, "nv": {}, "nz": {}, "mb": {}, "mp": {},
	"mf": {}, "mv": {}, "mk": {}, "lm": {}, "ln": {},
	"lp": {}, "lr": {}, "ls": {}, "lt": {}, "lv": {},
	"lz": {}, "ld": {}, "lf": {}, "lg": {}, "lh": {},
	"lk": {}, "tr": {}, "tk": {}, "tn": {}, "tm": {},
	"ts": {}, "tv": {}, "tz": {}, "tl": {}, "kr": {},
	"kl": {}, "kn": {}, "km": {}, "ks": {}, "kt": {},
	"kv": {}, "kz": {}, "br": {}, "bl": {}, "bn": {},
	"dr": {}, "dl": {}, "dn": {}, "dm": {}, "ds": {},
	"dt": {}, "fr": {}, "fl": {}, "gr": {}, "gl": {},
	"gn": {}, "gm": {}, "hr": {}, "hl": {}, "pr": {},
	"pl": {}, "sp": {}, "st": {}, "sk": {}, "sm": {},
	"sn": {}, "sl": {}, "zt": {}, "zd": {}, "zm": {},
	"zn": {}, "zp": {}, "zv": {}, "öl": {}, "ől": {},
	"ör": {}, "őr": {}, "ős": {}, "és": {}, "ét": {},
	"ém": {}, "én": {}, "ér": {}, "ék": {}, "él": {},
	"ép": {}, "án": {}, "ág": {}, "áb": {}, "ád": {},
	"ít": {}, "ós": {}, "ól": {}, "ód": {}, "ór": {},
	"ón": {}, "út": {}, "úl": {}, "úr": {}, "ús": {},
}
