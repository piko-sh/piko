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

package linguistics_stemmer_hebrew

// irregularForms maps Hebrew surface forms that resist productive
// prefix or suffix stripping directly to their base form. Entries
// represent linguistic facts about Modern Hebrew.
//
// The map is consulted early in the stemming pipeline so that a hit
// short-circuits the rule-based strippers that would otherwise mangle
// broken-plural, suppletive, or segholate forms.
//
// Entries that map a surface form to itself are deliberate guards:
// they block downstream affix stripping for already-canonical bases
// that the rule-based pipeline would otherwise over-trim.
//
// All keys and values are stored in the folded (no final-form)
// orthography used by the rest of the pipeline.
var irregularForms = map[string]string{
	"אנשימ":   "איש",
	"נשימ":    "אישה",
	"בנימ":    "בנ",
	"בנות":    "בת",
	"אחימ":    "אח",
	"אחיות":   "אחות",
	"אבות":    "אב",
	"אימהות":  "אמ",
	"סבים":    "סב",
	"סבתות":   "סבתא",
	"שכנימ":   "שכנ",
	"שכנות":   "שכנה",
	"דודימ":   "דוד",
	"דודות":   "דודה",
	"תלמידימ": "תלמיד",
	"בתימ":    "בית",
	"ראשימ":   "ראש",
	"ערימ":    "עיר",
	"שמות":    "שמ",
	"אנפימ":   "אפ",
	"עינימ":   "עינ",
	"אוזנימ":  "אוזנ",
	"ידימ":    "יד",
	"רגלימ":   "רגל",
	"שינימ":   "שנ",
	"שולחנות": "שולחנ",
	"כיסאות":  "כיסא",
	"חלונות":  "חלונ",
	"דלתות":   "דלת",
	"שנימ":    "שנה",
	"ימימ":    "יומ",
	"לילות":   "לילה",
	"חודשימ":  "חודש",
	"שבועות":  "שבוע",
	"עשורימ":  "עשור",
	"אוזניימ": "אוזנ",
	"עיניימ":  "עינ",
	"ידיימ":   "יד",
	"רגליימ":  "רגל",
	"שפתיימ":  "שפה",
	"ברכיימ":  "ברכ",
	"הלכ":     "הלכ",
	"הלכה":    "הלכ",
	"הלכו":    "הלכ",
	"הולכ":    "הלכ",
	"הולכת":   "הלכ",
	"הולכימ":  "הלכ",
	"הולכות":  "הלכ",
	"ילכ":     "הלכ",
	"תלכ":     "הלכ",
	"אלכ":     "הלכ",
	"נלכ":     "הלכ",
	"לכת":     "הלכ",
	"ללכת":    "הלכ",
	"אומר":    "אמר",
	"אומרת":   "אמר",
	"אומרימ":  "אמר",
	"אומרות":  "אמר",
	"אוכל":    "אכל",
	"אוכלת":   "אכל",
	"אוכלימ":  "אכל",
	"אוכלות":  "אכל",
	"לומד":    "למד",
	"לומדת":   "למד",
	"לומדימ":  "למד",
	"לומדות":  "למד",
	"כתבתי":   "כתב",
	"כתבת":    "כתב",
	"כתבתמ":   "כתב",
	"כתבתנ":   "כתב",
	"כתבנו":   "כתב",
	"למדתי":   "למד",
	"למדת":    "למד",
	"למדתמ":   "למד",
	"למדנו":   "למד",
	"למדו":    "למד",
	"שמעו":    "שמע",
	"שמענו":   "שמע",
	"לשמוע":   "שמע",
	"כתבו":    "כתב",
	"שרימ":    "שר",
	"שהבית":   "בית",
	"והבית":   "בית",
	"כשהבית":  "בית",
	"ראה":     "ראה",
	"ראתה":    "ראה",
	"ראו":     "ראה",
	"ראיתי":   "ראה",
	"רואה":    "ראה",
	"רואימ":   "ראה",
	"רואות":   "ראה",
	"עשה":     "עשה",
	"עשתה":    "עשה",
	"עשו":     "עשה",
	"עשיתי":   "עשה",
	"עושה":    "עשה",
	"עושימ":   "עשה",
	"עושות":   "עשה",
	"לעשות":   "עשה",
	"לראות":   "ראה",
	"קורא":    "קרא",
	"קוראת":   "קרא",
	"קוראימ":  "קרא",
	"קוראות":  "קרא",
	"קראו":    "קרא",
	"קראתי":   "קרא",
	"אוהב":    "אהב",
	"אוהבת":   "אהב",
	"אוהבימ":  "אהב",
	"אוהבות":  "אהב",
	"אהבתי":   "אהב",
	"אהבת":    "אהב",
	"יפימ":    "יפה",
	"יפות":    "יפה",
	"קשימ":    "קשה",
	"קשות":    "קשה",
	"רעימ":    "רע",
	"רעות":    "רע",
}

// lookupIrregular returns the base form registered for a surface form
// and a boolean indicating whether the lookup succeeded.
//
// Takes word (string) which is the nikkud-stripped, final-form-folded
// surface form produced by earlier pipeline stages.
//
// Returns string which is the mapped base form (empty on miss).
// Returns bool which is true when the map contained the word.
func lookupIrregular(word string) (string, bool) {
	base, ok := irregularForms[word]
	return base, ok
}
