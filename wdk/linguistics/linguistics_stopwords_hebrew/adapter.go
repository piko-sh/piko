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

package linguistics_stopwords_hebrew

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "hebrew"

// stopWords contains common Hebrew stop words curated from the
// stopwords-iso, NNLP-IL, and NLTK reference lists and organised by
// grammatical category. Entries are base forms only because the
// stemmer strips prefixed particles before filtering.
var stopWords = map[string]bool{
	"אני": true, "אתה": true, "את": true, "הוא": true, "היא": true,
	"אנחנו": true, "אתם": true, "אתן": true, "הם": true, "הן": true,

	"זה": true, "זאת": true, "זו": true, "אלה": true, "אלו": true,

	"עצמי": true, "עצמו": true, "עצמה": true, "עצמם": true, "עצמן": true, "עצמנו": true,

	"מי": true, "מה": true, "איפה": true, "היכן": true, "מתי": true,
	"למה": true, "מדוע": true, "איך": true, "כמה": true,

	"אשר": true, "ש": true,

	"של": true, "על": true, "אל": true, "עם": true, "בין": true,
	"לפני": true, "אחרי": true, "תחת": true, "מול": true, "בלי": true,
	"מן": true, "עד": true, "אצל": true, "נגד": true, "דרך": true,
	"מעל": true, "מתחת": true, "ליד": true,

	"ו": true, "או": true, "אבל": true, "אם": true, "כי": true,
	"אלא": true, "גם": true, "רק": true, "עוד": true, "אך": true, "אולם": true,

	"היה": true, "היתה": true, "יהיה": true, "תהיה": true, "להיות": true,
	"יש": true, "אין": true, "יכול": true, "יכולה": true, "יכולים": true,

	"כל": true, "כולם": true, "כולן": true, "כלל": true, "הרבה": true,
	"מעט": true, "איזה": true,

	"מאוד": true, "כבר": true, "פה": true, "שם": true, "כאן": true,
	"עכשיו": true, "תמיד": true, "אף": true, "כך": true, "ככה": true,
	"כן": true, "אז": true, "שוב": true, "יותר": true, "פחות": true,
	"הנה": true, "הרי": true,

	"לא": true,

	"כמו": true, "כפי": true, "כאשר": true, "לפיכך": true, "לכן": true,
	"למרות": true, "בגלל": true, "באמצעות": true, "בשביל": true, "בתוך": true,
	"אולי": true, "כדי": true,

	"שלי": true, "שלך": true, "שלו": true, "שלה": true, "שלנו": true,
	"שלכם": true, "שלכן": true, "שלהם": true, "שלהן": true,
}

// Provider provides Hebrew stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort
// interface.
type Provider struct{}

// GetStopWords returns the Hebrew stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "hebrew".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new Hebrew stop words provider instance.
//
// Use this with linguistics_domain.RegisterStopWordsProviderFactory
// for explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which provides
// Hebrew stop word filtering.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a new Hebrew stop words provider.
//
// Returns *Provider which is ready for use.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() {
	linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory)
}
