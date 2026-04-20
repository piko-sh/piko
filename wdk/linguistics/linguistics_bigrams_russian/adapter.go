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

package linguistics_bigrams_russian

import (
	"strings"
	"unicode"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this bigram analyser.
const Language = "russian"

// minFieldLength is the minimum letter count for bigram analysis.
const minFieldLength = 4

// maxAnalyseLength is the maximum text byte length processed during analysis.
const maxAnalyseLength = 4096

var _ linguistics_domain.BigramAnalyserPort = (*BigramAnalyser)(nil)

// BigramAnalyser provides Russian character bigram frequency analysis
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
		if _, found := russianBigrams[bigram]; !found {
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

// Factory creates a new Russian bigram analyser instance.
//
// Returns linguistics_domain.BigramAnalyserPort which is the analyser.
// Returns error when creation fails.
func Factory() (linguistics_domain.BigramAnalyserPort, error) {
	return &BigramAnalyser{}, nil
}

func init() {
	linguistics_domain.RegisterBigramAnalyserFactory(Language, Factory)
}

// russianBigrams contains the most frequent Russian letter bigrams.
var russianBigrams = map[string]struct{}{
	"ст": {}, "но": {}, "на": {}, "ен": {}, "то": {},
	"ко": {}, "ов": {}, "ни": {}, "пр": {}, "ер": {},
	"ро": {}, "по": {}, "ре": {}, "во": {}, "не": {},
	"ан": {}, "ос": {}, "ра": {}, "ли": {}, "ом": {},
	"ор": {}, "ве": {}, "ть": {}, "ес": {}, "ва": {},
	"ол": {}, "ет": {}, "ел": {}, "ал": {}, "ат": {},
	"ка": {}, "та": {}, "ри": {}, "ла": {}, "ит": {},
	"те": {}, "де": {}, "ле": {}, "ог": {}, "ас": {},
	"го": {}, "ил": {}, "ны": {}, "он": {}, "се": {},
	"од": {}, "ск": {}, "ак": {}, "ти": {}, "от": {},
	"ий": {}, "ль": {}, "ин": {}, "ая": {}, "ой": {},
	"же": {}, "за": {}, "им": {}, "ид": {}, "из": {},
	"ме": {}, "да": {}, "мо": {}, "че": {}, "ми": {},
	"ых": {}, "ие": {}, "ые": {}, "ок": {}, "ам": {},
	"их": {}, "дн": {}, "тв": {}, "мы": {}, "ну": {},
	"бы": {}, "ду": {}, "жи": {}, "ло": {}, "тр": {},
	"ки": {}, "до": {}, "ма": {}, "сь": {}, "ав": {},
	"ев": {}, "об": {}, "нь": {}, "ши": {}, "ту": {},
	"бо": {}, "ис": {}, "ку": {}, "жн": {}, "ви": {},
	"ег": {}, "пе": {}, "ди": {}, "га": {}, "ча": {},
	"ац": {}, "ия": {}, "бе": {}, "ич": {}, "сл": {},
	"ив": {}, "пи": {}, "ца": {}, "ую": {}, "юю": {},
	"аж": {}, "ым": {}, "ый": {}, "ем": {}, "яз": {},
	"гд": {}, "зв": {}, "со": {}, "вы": {}, "вс": {},
	"тс": {}, "ся": {}, "ут": {}, "ют": {}, "ук": {},
	"ое": {}, "ар": {}, "ах": {}, "ач": {}, "ащ": {},
	"бл": {}, "бр": {}, "бу": {}, "ба": {}, "вр": {},
	"гл": {}, "гр": {}, "гу": {}, "др": {}, "жд": {},
	"зн": {}, "зо": {}, "ик": {}, "йн": {}, "йт": {},
	"кв": {}, "кл": {}, "кр": {}, "кт": {}, "лу": {},
	"лы": {}, "лю": {}, "мн": {}, "нг": {}, "нс": {},
	"нц": {}, "оз": {}, "ож": {}, "ох": {}, "оч": {},
	"пл": {}, "пу": {}, "рг": {}, "рд": {}, "рн": {},
	"рс": {}, "ру": {}, "рь": {}, "ря": {}, "ры": {},
	"са": {}, "св": {}, "сд": {}, "сн": {}, "сп": {},
	"су": {}, "сч": {}, "сы": {}, "тк": {}, "тн": {},
	"уд": {}, "уз": {}, "уй": {}, "ул": {}, "ум": {},
	"ун": {}, "ур": {}, "ус": {}, "уч": {}, "уш": {},
	"ущ": {}, "фе": {}, "фи": {}, "фо": {}, "ха": {},
	"хи": {}, "хо": {}, "хр": {}, "ху": {}, "хв": {},
	"цв": {}, "це": {}, "ци": {}, "чи": {}, "чт": {},
	"ша": {}, "ше": {}, "шк": {}, "шу": {}, "ще": {},
	"щи": {}, "ыв": {}, "ыл": {}, "ын": {}, "эт": {},
	"юб": {}, "юн": {}, "яв": {}, "ял": {}, "ян": {},
	"яр": {}, "яс": {}, "ят": {},
}
